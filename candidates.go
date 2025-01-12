package main

import (
	"context"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"slices"
)

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
		votedForRows, err := dbpool.Query(context.Background(), "SELECT position FROM votes WHERE candidate = $1 AND voter_id = $2", name, user)
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
		votesRemaining := configEditor.GetInt("maxvotes") - numVotes
		allPositions := configEditor.GetStringMapString("positions")
		groups := session.Get("groups").([]string)
		var eligiblePositions []string
		for position, group := range allPositions {
			if group == " " || slices.Contains(groups, group) {
				eligiblePositions = append(eligiblePositions, position)
			}
		}
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"userId":         userId,
			"name":           name,
			"description":    description,
			"hookstatement":  hookstatement,
			"video":          video,
			"keywords":       keywords,
			"published":      true,
			"admin":          false,
			"positions":      eligiblePositions,
			"votedFor":       votedFor,
			"votesRemaining": votesRemaining,
		})
	})

	r.POST("/vote", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		candidate := c.Query("candidate")
		candidateID := c.Query("candidate_id")
		position := c.Query("position")
		user := session.Get("user_id").(string)
		var voted bool
		err := dbpool.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM votes WHERE candidate = $1 AND candidate_id = $2 AND voter_id = $3 AND position = $4)", candidate, candidateID, user, position,
		).Scan(&voted)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to check vote: %v", err)
			return
		}
		if voted {
			_, err = dbpool.Exec(context.Background(), "DELETE FROM votes WHERE candidate = $1 AND candidate_id = $2 AND voter_id = $3 AND position = $4)", candidate, candidateID, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to delete vote: %v", err)
				return
			}
			c.Redirect(http.StatusFound, "/"+candidate)
		} else {
			_, err = dbpool.Exec(context.Background(), "INSERT INTO votes (candidate, candidate_id, voter_id, position) VALUES ($1, $2, $3, $4)", candidate, candidateID, user, position)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to insert vote: %v", err)
				return
			}
			c.Redirect(http.StatusFound, "/"+candidate)
		}
	})
}
