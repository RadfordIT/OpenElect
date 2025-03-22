package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"net/http"
	"slices"
)

func profileRoutes() {
	type Keyword struct {
		Name  string
		Count int
	}

	r.GET("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		var userId string
		var description string
		var hookstatement string
		var video string
		var keywords []string
		var positions []string
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM pending WHERE id = $1", session.Get("user_id")).Scan(&userId, nil, nil, &description, &hookstatement, &video, &keywords, &positions)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE id = $1", session.Get("user_id")).Scan(&userId, nil, nil, &description, &hookstatement, &video, &keywords, &positions)
				if err != nil {
					if !errors.Is(err, pgx.ErrNoRows) {
						c.String(http.StatusInternalServerError, "Failed to get profile: %v", err)
						return
					}
				}
			} else {
				c.String(http.StatusInternalServerError, "Failed to get profile: %v", err)
				return
			}
		}
		allPositions := configEditor.GetStringMapString("positions")
		groups := session.Get("groups").([]string)
		var eligiblePositions []string
		for position, group := range allPositions {
			if group == "" || slices.Contains(groups, group) {
				eligiblePositions = append(eligiblePositions, position)
			}
		}
		rows, err := dbpool.Query(context.Background(), `
			SELECT keyword, COUNT(*) AS occurrences
			FROM (
				SELECT unnest(keywords) AS keyword FROM candidates
				UNION ALL
				SELECT unnest(keywords) FROM pending
			) AS all_keywords
			GROUP BY keyword
			ORDER BY occurrences DESC`,
		)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get keywords: %v", err)
			return
		}
		defer rows.Close()
		var allKeywords []Keyword
		for rows.Next() {
			var keyword Keyword
			err = rows.Scan(&keyword.Name, &keyword.Count)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to scan keyword: %v", err)
				return
			}
			allKeywords = append(allKeywords, keyword)
		}
		c.HTML(http.StatusOK, "profile.tmpl", gin.H{
			"userId":        userId,
			"description":   description,
			"hookstatement": hookstatement,
			"video":         video,
			"keywords":      keywords,
			"positions":     positions,
			"allpositions":  eligiblePositions,
			"allkeywords":   allKeywords,
			"maxtags":       configEditor.GetInt("maxtags"),
		})
	})
	r.POST("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := session.Get("name").(string)
		userID := session.Get("user_id").(string)
		email := session.Get("email").(string)
		description := c.PostForm("description")
		hookstatement := c.PostForm("hookstatement")
		tags := c.PostFormArray("tag[]")
		positions := c.PostFormArray("position[]")
		deleteVideoFlag := c.PostForm("deletevideo")
		videoFilename := c.PostForm("oldvideo")
		if deleteVideoFlag == "true" || videoFilename == "" {
			video, header, err := c.Request.FormFile("video")
			if err != nil && !errors.Is(err, http.ErrMissingFile) {
				c.String(http.StatusInternalServerError, "Failed to upload video: %v", err)
				return
			}
			if errors.Is(err, http.ErrMissingFile) && videoFilename == "" {
				// user didn't upload a video and didn't have one before, so we don't need to do anything
			} else if errors.Is(err, http.ErrMissingFile) {
				err = deleteVideo(videoFilename)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to delete video: %v", err)
					return
				}
				videoFilename = ""
			} else {
				if header.Header.Get("Content-Type") != "video/mp4" {
					c.String(http.StatusBadRequest, "Invalid video format: only mp4 is supported")
					return
				}
				defer video.Close()
				videoFilename = fmt.Sprintf("%s.mp4", userID)
				err = uploadVideo(videoFilename, video)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to upload video: %v", err)
					return
				}
				fmt.Println("Uploaded video to", videoFilename)
			}
		}
		_, err := dbpool.Exec(context.Background(),
			`INSERT INTO pending 
    			(id, name, email, description, hookstatement, video, keywords, positions) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				ON CONFLICT(id) DO UPDATE SET id = $1, name = $2, email = $3, description = $4, hookstatement = $5, video = $6, keywords = $7, positions = $8`,
			userID, name, email, description, hookstatement, videoFilename, tags, positions,
		)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to upsert candidate: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/preview")
	})
	r.GET("/preview", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := session.Get("name").(string)
		userId := session.Get("user_id").(string)
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		video := ""
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
			"admin":         false,
			"positions":     positions,
		})
	})
	r.POST("/preview", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		session.AddFlash("Your profile has been submitted for review.")
		err := session.Save()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
			return
		}
		c.Redirect(http.StatusSeeOther, "/")
	})
}
