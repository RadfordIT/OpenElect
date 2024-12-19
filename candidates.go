package main

import (
	"context"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
)

func createTables() {
	dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS candidates,votes")
	dbpool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS candidates (
    	id TEXT NOT NULL PRIMARY KEY, 
    	name TEXT NOT NULL, 
    	description TEXT NOT NULL CHECK (char_length(description) <= 5000), 
    	hookstatement TEXT NOT NULL CHECK (char_length(hookstatement) <= 150), 
    	keywords TEXT[] CHECK (array_length(keywords, 1) <= 6), 
    	positions TEXT[],
    	published BOOLEAN DEFAULT NULL
    )`)
	dbpool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS votes (
    	vote_id SERIAL PRIMARY KEY,
    	candidate_id TEXT NOT NULL,
    	voter_id TEXT NOT NULL,
    	position TEXT NOT NULL,
    	UNIQUE(candidate_id, voter_id, position)
    )`)
}

func voteRoutes() {
	r.GET("/:candidate", authMiddleware(), func(c *gin.Context) {
		name := c.Param("candidate")
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1 AND published IS TRUE", name).Scan(&userId, &name, &description, &hookstatement, &keywords, &positions, nil)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"userId":        userId,
			"name":          name,
			"description":   description,
			"hookstatement": hookstatement,
			"keywords":      keywords,
			"published":     true,
			"admin":         false,
			"positions":     positions,
		})
	})

	r.POST("/vote", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		candidate_id := c.Query("candidate")
		position := c.Query("position")
		user := session.Get("user_id").(string)
		var voted bool
		err := dbpool.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM votes WHERE candidate_id = $1 AND voter_id = $2 AND position = $3)", candidate_id, user, position,
		).Scan(&voted)
		if err != nil {
			fmt.Println(err)
			c.String(http.StatusInternalServerError, "Failed to check vote: %v", err)
			return
		}
		vote := c.DefaultQuery("vote", "true")
		if vote == "false" {
			if voted {
				c.HTML(http.StatusOK, "votebutton.tmpl", gin.H{
					"voted":     true,
					"candidate": candidate_id,
					"position":  position,
				})
			} else {
				c.HTML(http.StatusOK, "votebutton.tmpl", gin.H{
					"voted":     false,
					"candidate": candidate_id,
					"position":  position,
				})
			}
			return
		}
		if voted {
			_, err = dbpool.Exec(context.Background(), "DELETE FROM votes WHERE candidate_id = $1 AND voter_id = $2 AND position = $3", candidate_id, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to delete vote: %v", err)
				return
			}
			c.HTML(http.StatusOK, "votebutton.tmpl", gin.H{
				"voted":     false,
				"candidate": candidate_id,
				"position":  position,
			})
		} else {
			_, err = dbpool.Exec(context.Background(), "INSERT INTO votes (candidate_id, voter_id, position) VALUES ($1, $2, $3)", candidate_id, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to insert vote: %v", err)
				return
			}
			c.HTML(http.StatusOK, "votebutton.tmpl", gin.H{
				"voted":     true,
				"candidate": candidate_id,
				"position":  position,
			})
		}
	})
}
