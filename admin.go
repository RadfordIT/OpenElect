package main

import (
	"context"
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
)

func adminRoutes() {
	r.GET("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.tmpl", gin.H{
			"colors": colorsEditor.GetStringMapString("colors"),
			"colorNames": [...]string{
				"accent",
				"accentContent",
				"base100",
				"base200",
				"base300",
				"baseContent",
				"error",
				"errorContent",
				"info",
				"infoContent",
				"neutral",
				"neutralContent",
				"primary",
				"primaryContent",
				"secondary",
				"secondaryContent",
				"success",
				"successContent",
				"warning",
				"warningContent",
			},
			"positions":       configEditor.GetStringMapString("positions"),
			"maxvotes":        configEditor.GetInt("maxvotes"),
			"indeximage":      configEditor.GetString("indeximage"),
			"candidategroup":  configEditor.GetString("candidategroup"),
			"endelectiontime": configEditor.GetString("endelectiontime"),
		})
	})
	r.POST("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		colors := c.PostFormMap("colors")
		colorsEditor.Set("colors", colors)
		colorsEditor.WriteConfig()

		positionNames := c.PostFormArray("position[]")
		requiredGroups := c.PostFormArray("requiredgroup[]")
		positions := make(map[string]string)
		for i, position := range positionNames {
			positions[position] = requiredGroups[i]
			dbpool.Exec(context.Background(), "INSERT INTO positions (name) VALUES ($1) ON CONFLICT DO NOTHING", position)
		}
		configEditor.Set("positions", positions)
		maxVotes, err := strconv.Atoi(c.PostForm("maxvotes"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid max votes: %v", err)
			return
		}
		configEditor.Set("maxvotes", maxVotes)
		candidateGroup := c.PostForm("candidategroup")
		configEditor.Set("candidategroup", candidateGroup)
		indexImage := c.PostForm("indeximage")
		configEditor.Set("indeximage", indexImage)
		endElectionTime := c.PostForm("endelectiontime")
		configEditor.Set("endelectiontime", endElectionTime)
		configEditor.WriteConfig()
		c.Redirect(http.StatusSeeOther, "/admin")
	})
	r.GET("/admin/candidates", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		rows, err := dbpool.Query(context.Background(), "SELECT * FROM pending")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to query candidates: %v", err)
			return
		}
		var candidates []Candidate
		for rows.Next() {
			var candidate Candidate
			err = rows.Scan(&candidate.ID, &candidate.Name, nil, &candidate.Description, &candidate.HookStatement, nil, &candidate.Keywords, &candidate.Positions)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to scan candidate: %v", err)
				return
			}
			candidates = append(candidates, candidate)
		}
		c.HTML(http.StatusOK, "admincandidates.tmpl", gin.H{
			"candidates": candidates,
			"flashes":    session.Flashes(),
		})
		session.Save()
	})
	r.GET("/admin/candidates/:name", adminAuthMiddleware(), func(c *gin.Context) {
		name := c.Param("name")
		var userId string
		var description string
		var hookstatement string
		var video string
		var keywords []string
		var positions []string
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM pending WHERE name = $1", name).Scan(&userId, &name, nil, &description, &hookstatement, &video, &keywords, &positions)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"userId":        userId,
			"name":          name,
			"description":   description,
			"hookstatement": hookstatement,
			"video":         video,
			"keywords":      keywords,
			"published":     false,
			"admin":         true,
			"positions":     positions,
		})
	})
	r.POST("/admin/candidates/:name/reject", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := c.Param("name")
		reason := c.PostForm("reason")
		var email string
		err := dbpool.QueryRow(context.Background(), "SELECT email FROM candidates WHERE name = $1", name).Scan(&email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err = dbpool.QueryRow(context.Background(), "SELECT email FROM pending WHERE name = $1", name).Scan(&email)
				if err != nil {
					c.String(http.StatusNotFound, "Candidate not found: %v", err)
					return
				}
			} else {
				c.String(http.StatusInternalServerError, "Failed to get email: %v", err)
				return
			}
		}
		body := "Your candidate profile has been rejected. Please log in to edit your profile."
		if reason != "" {
			body += "\n\nReason: \n" + reason
		}
		err = sendEmail(session.Get("email").(string), email, "Candidate Rejected", body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to send email: %v", err)
			return
		}
		_, err = dbpool.Exec(context.Background(), "DELETE FROM pending WHERE name = $1", name)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to reject candidate: %v", err)
			return
		}
		session.AddFlash("Candidate " + name + " successfully rejected")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/admin/candidates")
	})
	r.POST("/admin/candidates/:name/accept", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := c.Param("name")
		_, err := dbpool.Exec(context.Background(),
			`WITH candidate AS (DELETE FROM pending WHERE name = $1 RETURNING *)
			INSERT INTO candidates SELECT * FROM candidate
			ON CONFLICT(id) DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name, email = EXCLUDED.email, description = EXCLUDED.description, hookstatement = EXCLUDED.hookstatement, video = EXCLUDED.video, keywords = EXCLUDED.keywords, positions = EXCLUDED.positions`,
			name)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to publish candidate: %v", err)
			return
		}
		var userId string
		var email string
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		err = dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1", name).Scan(&userId, &name, &email, &description, &hookstatement, nil, &keywords, &positions)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		err = index(userId, name, description, hookstatement, keywords, positions)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to index candidate: %v", err)
			return
		}
		err = sendEmail(session.Get("email").(string), email, "Candidate Accepted", "Your candidate profile has been accepted. Please log in to view your profile.")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to send email: %v", err)
			return
		}
		session.AddFlash("Candidate " + name + " successfully accepted")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/admin/candidates")
	})
}
