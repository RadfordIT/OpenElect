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
    	video TEXT DEFAULT NULL,
    	keywords TEXT[] CHECK (array_length(keywords, 1) <= 6), 
    	positions TEXT[] CHECK (array_length(positions, 1) >= 1),
    	published BOOLEAN DEFAULT NULL
    )`)
	dbpool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS votes (
    	vote_id SERIAL PRIMARY KEY,
    	candidate_id TEXT NOT NULL CHECK (char_length(candidate_id) > 0),
    	voter_id TEXT NOT NULL CHECK (char_length(voter_id) > 0),
    	position TEXT NOT NULL CHECK (char_length(position) > 0),
    	UNIQUE(candidate_id, voter_id, position)
    )`)
}

func voteRoutes() {
	r.GET("/:candidate", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := c.Param("candidate")
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		video := ""
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1 AND published IS TRUE", name).Scan(&userId, &name, &description, &hookstatement, &video, &keywords, &positions, nil)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		var numVotes int
		user := session.Get("user_id").(string)
		err = dbpool.QueryRow(context.Background(), "SELECT COUNT(*) FROM votes WHERE voter_id = $1", user).Scan(&numVotes)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to check vote count: %v", err)
			return
		}
		votedForRows, err := dbpool.Query(context.Background(), "SELECT position FROM votes WHERE candidate_id = $1 AND voter_id = $2", name, user)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to check vote: %v", err)
			return
		}
		var votedFor []string
		for votedForRows.Next() {
			var position string
			err = votedForRows.Scan(&position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to scan vote: %v", err)
				return
			}
			votedFor = append(votedFor, position)
		}
		fmt.Println(votedFor)
		votesRemaining := configEditor.GetInt("maxvotes") - numVotes
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"userId":         userId,
			"name":           name,
			"description":    description,
			"hookstatement":  hookstatement,
			"video":          video,
			"keywords":       keywords,
			"published":      true,
			"admin":          false,
			"positions":      positions,
			"votedFor":       votedFor,
			"votesRemaining": votesRemaining,
		})
	})

	r.POST("/vote", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		candidate := c.Query("candidate")
		position := c.Query("position")
		user := session.Get("user_id").(string)
		var voted bool
		err := dbpool.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM votes WHERE candidate_id = $1 AND voter_id = $2 AND position = $3)", candidate, user, position,
		).Scan(&voted)
		if err != nil {
			fmt.Println(err)
			c.String(http.StatusInternalServerError, "Failed to check vote: %v", err)
			return
		}
		if voted {
			_, err = dbpool.Exec(context.Background(), "DELETE FROM votes WHERE candidate_id = $1 AND voter_id = $2 AND position = $3", candidate, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to delete vote: %v", err)
				return
			}
			c.Redirect(http.StatusFound, "/"+candidate)
		} else {
			_, err = dbpool.Exec(context.Background(), "INSERT INTO votes (candidate_id, voter_id, position) VALUES ($1, $2, $3)", candidate, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to insert vote: %v", err)
				return
			}
			c.Redirect(http.StatusFound, "/"+candidate)
		}
	})
}
