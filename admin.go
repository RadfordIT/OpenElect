package main

import (
	"context"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
			"positions":      configEditor.GetStringSlice("positions"),
			"maxvotes":       configEditor.GetInt("maxvotes"),
			"candidategroup": configEditor.GetString("candidategroup"),
		})
	})
	r.POST("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		colors := c.PostFormMap("colors")
		colorsEditor.Set("colors", colors)
		colorsEditor.WriteConfig()

		positions := c.PostFormArray("position[]")
		configEditor.Set("positions", positions)
		maxVotes, err := strconv.Atoi(c.PostForm("maxvotes"))
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid max votes: %v", err)
			return
		}
		configEditor.Set("maxvotes", maxVotes)
		candidateGroup := c.PostForm("candidategroup")
		configEditor.Set("candidategroup", candidateGroup)
		configEditor.WriteConfig()
		c.Redirect(http.StatusSeeOther, "/admin")
	})
	r.GET("/admin/candidates", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		rows, err := dbpool.Query(context.Background(), "SELECT * FROM candidates WHERE published IS FALSE")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to query candidates: %v", err)
			return
		}
		var candidates []Candidate
		for rows.Next() {
			var candidate Candidate
			err = rows.Scan(&candidate.ID, &candidate.Name, &candidate.Description, &candidate.HookStatement, nil, &candidate.Keywords, &candidate.Positions, nil)
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
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1 AND published IS FALSE", name).Scan(&userId, &name, &description, &hookstatement, &video, &keywords, &positions, nil)
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
		_, err := dbpool.Exec(context.Background(), "DELETE FROM candidates WHERE name = $1", name)
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
		_, err := dbpool.Exec(context.Background(), "UPDATE candidates SET published = TRUE WHERE name = $1", name)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to publish candidate: %v", err)
			return
		}
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		err = dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1 AND published IS TRUE", name).Scan(&userId, &name, &description, &hookstatement, nil, &keywords, &positions, nil)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		index(userId, name, description, hookstatement, keywords, positions)
		session.AddFlash("Candidate " + name + " successfully accepted")
		session.Save()
		c.Redirect(http.StatusSeeOther, "/admin/candidates")
	})
}
