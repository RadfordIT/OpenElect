package main

import (
	"context"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"slices"
	"time"
)

func checkElectionEndedMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("user_id")
		if token == nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		groups := session.Get("groups").([]string)
		if slices.Contains(groups, configEditor.GetString("admingroup")) {
			c.Next()
			return
		}
		endElectionTimeString := configEditor.GetString("endelectiontime")
		endElectionTime, err := time.Parse("2006-01-02", endElectionTimeString)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error parsing election end time")
			c.Abort()
			return
		}
		if time.Now().Before(endElectionTime) {

			c.String(http.StatusForbidden, "The election has not ended yet, results are not available.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func resultsRoutes() {
	r.GET("/results", authMiddleware(), checkElectionEndedMiddleware(), func(c *gin.Context) {
		positionsMap := configEditor.GetStringMapString("positions")
		var positions []string
		for k := range positionsMap {
			positions = append(positions, k)
		}
		var winners map[string]string
		for _, position := range positions {
			var candidate string
			err := dbpool.QueryRow(context.Background(), `
				SELECT candidate_id, COUNT(*) AS vote_count
				FROM votes
				WHERE position = $1
				GROUP BY candidate_id
				ORDER BY vote_count DESC
				LIMIT 1;
			`, position).Scan(&candidate)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to get winner: %v", err)
				return
			}
			winners[position] = candidate
		}
		c.HTML(http.StatusOK, "results.tmpl", gin.H{
			"candidates": "",
		})
	})
}
