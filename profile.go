package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"slices"
)

func profileRoutes() {
	r.GET("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		var userId string
		var description string
		var hookstatement string
		var video string
		var keywords []string
		var positions []string
		dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE id = $1", session.Get("user_id")).Scan(&userId, nil, &description, &hookstatement, &video, &keywords, &positions, nil)
		allPositions := configEditor.GetStringMapString("positions")
		groups := session.Get("groups").([]string)
		var eligiblePositions []string
		for position, group := range allPositions {
			if group == "" || slices.Contains(groups, group) {
				eligiblePositions = append(eligiblePositions, position)
			}
		}
		c.HTML(http.StatusOK, "profile.tmpl", gin.H{
			"userId":        userId,
			"description":   description,
			"hookstatement": hookstatement,
			"video":         video,
			"keywords":      keywords,
			"positions":     positions,
			"allpositions":  eligiblePositions,
		})
	})
	r.POST("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := session.Get("name").(string)
		userID := session.Get("user_id").(string)
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
			if errors.Is(err, http.ErrMissingFile) {
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
			`INSERT INTO candidates 
    			(id, name, description, hookstatement, video, keywords, positions, published) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL)
				ON CONFLICT(id) DO UPDATE SET id = $1, name = $2, description = $3, hookstatement = $4, video = $5, keywords = $6, positions = $7, published = NULL`,
			userID, name, description, hookstatement, videoFilename, tags, positions,
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
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1 AND published IS NULL", name).Scan(&userId, &name, &description, &hookstatement, &video, &keywords, &positions, nil)
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
		name := session.Get("name").(string)
		_, err := dbpool.Exec(context.Background(), "UPDATE candidates SET published = FALSE WHERE name = $1", name)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		deindex(session.Get("user_id").(string))
		session.AddFlash("Your profile has been submitted for review.")
		err = session.Save()
		if err != nil {
			fmt.Println(err)
		}
		c.Redirect(http.StatusSeeOther, "/")
	})
}
