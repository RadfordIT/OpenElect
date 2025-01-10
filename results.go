package main

import (
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

func resultsRoutes() {
	r.GET("/results", authMiddleware(), checkElectionEndedMiddleware(), func(c *gin.Context) {
		winners := make(map[string]string)
		c.HTML(http.StatusOK, "results.tmpl", gin.H{
			"candidates": winners,
		})
	})
}
