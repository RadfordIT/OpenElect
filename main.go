package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"net/http"
)

var r *gin.Engine

type Candidate struct {
	Name          string   `json:"name"`
	Keywords      []string `json:"keywords"`
	HookStatement string   `json:"hookstatement"`
	Description   string   `json:"description"`
}

func toStringSlice(input []interface{}) []string {
	output := make([]string, len(input))
	for i, v := range input {
		output[i] = v.(string)
	}
	return output
}

func main() {
	authSetup()
	dbSetup()
	searchSetup()
	gob.Register(map[string]interface{}{})

	r = gin.Default()
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/style.css", "./css/output.css")
	r.StaticFile("/icon.png", "./static/icon.png")
	r.LoadHTMLGlob("templates/*")
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("session", store))

	loginRoutes()

	r.GET("/", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		fmt.Println(session.Get("user_id"), session.Get("groups"))
		query := c.DefaultQuery("q", "")
		candidates := search(query)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"text": candidates,
		})
	})
	r.GET("/:candidate", authMiddleware(), func(c *gin.Context) {
		name := c.Param("candidate")
		var description string
		var hookstatement string
		var keywords []string
		err := dbpool.QueryRow(context.Background(), "SELECT name FROM candidates WHERE name = $1", name).Scan(&description, &hookstatement, &keywords)
		if err != nil {
			c.String(http.StatusNotFound, "Candidate not found")
			return
		}
		fmt.Println(description, hookstatement)
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"name": name,
		})
	})
	r.GET("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "profile.tmpl", gin.H{})
	})
	r.POST("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		description := c.PostForm("description")
	})
	r.Run()
}
