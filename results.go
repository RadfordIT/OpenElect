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
		endElectionTime, err := time.Parse("2006-02-01", endElectionTimeString)
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

type Result struct {
	Candidate   string
	CandidateID string
	Votes       int
	Winner      bool
}

func resultsRoutes() {
	r.GET("/results", authMiddleware(), checkElectionEndedMiddleware(), func(c *gin.Context) {
		winners := make(map[string]string)
		// TODO: implement
		c.HTML(http.StatusOK, "results.tmpl", gin.H{
			"candidates": winners,
		})
	})
	r.GET("/admin/results", adminAuthMiddleware(), func(c *gin.Context) {
		positionsMap := configEditor.GetStringMapString("positions")
		// TODO: concurrency
		highest := make(map[string][]Result)
		for position, _ := range positionsMap {
			func() {
				rows, err := dbpool.Query(context.Background(), `
					SELECT candidate_id, candidate, COUNT(*) AS vote_count
					FROM votes
					WHERE position = $1
					GROUP BY candidate_id, candidate
					ORDER BY COUNT(*) DESC
				`, position)
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to get winners: %v", err)
					return
				}
				defer rows.Close()
				for rows.Next() {
					var result Result
					err = rows.Scan(&result.CandidateID, &result.Candidate, &result.Votes)
					if err != nil {
						c.String(http.StatusInternalServerError, "Failed to scan winner: %v", err)
						return
					}
					//TODO: fix
					fmt.Printf("Executing query: SELECT 1 FROM winners WHERE position_name = %s AND candidate_id = %s\n", position, result.CandidateID)
					err = dbpool.QueryRow(context.Background(), "SELECT TRUE FROM winners WHERE position_name = $1 AND candidate_id = $2", position, result.CandidateID).Scan(&result.Winner)
					if err != nil && !errors.Is(err, pgx.ErrNoRows) {
						c.String(http.StatusInternalServerError, "Failed to check winner: %v", err)
						return
					}
					fmt.Println(result)
					highest[position] = append(highest[position], result)
				}
			}()
		}

		c.HTML(http.StatusOK, "adminresults.tmpl", gin.H{
			"candidates": highest,
		})
	})
	r.POST("/admin/results/add", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		position := c.Query("position")
		candidate := c.Query("candidate")
		candidateID := c.Query("candidate_id")
		_, err := dbpool.Exec(context.Background(), "INSERT INTO winners (position_name, candidate_id, candidate) VALUES ($1, $2, $3) ON CONFLICT (position_name, candidate_id) DO NOTHING", position, candidateID, candidate)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to add winner: %v", err)
			return
		}
		session.AddFlash(fmt.Sprintf("Added winner for %s: %s", position, candidate))
		session.Save()
		c.Redirect(http.StatusSeeOther, "/admin/results")
	})
	r.POST("/admin/results/remove", adminAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		position := c.Query("position")
		candidate := c.Query("candidate")
		candidateID := c.Query("candidate_id")
		_, err := dbpool.Exec(context.Background(), "DELETE FROM winners WHERE position_name = $1 AND candidate_id = $2 AND candidate = $3", position, candidateID, candidate)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to remove winner: %v", err)
			return
		}
		session.AddFlash(fmt.Sprintf("Removed winner for %s", position))
		session.Save()
		c.Redirect(http.StatusSeeOther, "/admin/results")
	})
}
