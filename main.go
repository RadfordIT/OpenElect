package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
	"math/rand"
	"net/http"
	"os"
)

var r *gin.Engine
var dbpool *pgxpool.Pool
var configEditor, colorsEditor *viper.Viper

func main() {
	var err error

	configEditor = viper.New()
	configEditor.SetConfigName("config")
	configEditor.SetConfigType("yaml")
	configEditor.AddConfigPath("./config")
	configEditor.ReadInConfig()

	colorsEditor = viper.New()
	colorsEditor.SetConfigName("colors")
	colorsEditor.SetConfigType("json")
	colorsEditor.AddConfigPath("./config")
	colorsEditor.ReadInConfig()

	positions := configEditor.GetStringSlice("positions")
	fmt.Println(positions)
	positions = append(positions, "Candidate")
	fmt.Println(positions)
	configEditor.Set("positions", positions)
	configEditor.WriteConfig()

	colors := colorsEditor.GetStringMapString("colors")
	fmt.Println(colors)
	colors["primary"] = "#FF0000"
	fmt.Println(colors)
	colorsEditor.Set("colors", colors)
	colorsEditor.WriteConfig()

	authSetup()
	dbpool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()
	//dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS candidates")
	dbpool.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS candidates (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL CHECK (char_length(description) <= 5000), hookstatement TEXT NOT NULL CHECK (char_length(hookstatement) <= 150), keywords TEXT[] CHECK (array_length(keywords, 1) <= 6))")
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
		query := c.DefaultQuery("q", "")
		candidates := search(query)
		rand.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"text": candidates,
		})
	})
	r.GET("/:candidate", authMiddleware(), func(c *gin.Context) {
		name := c.Param("candidate")
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		err := dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE name = $1", name).Scan(&userId, &name, &description, &hookstatement, &keywords)
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
		})
	})
	r.GET("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE id = $1", session.Get("user_id")).Scan(&userId, nil, &description, &hookstatement, &keywords)
		c.HTML(http.StatusOK, "profile.tmpl", gin.H{
			"userId":        userId,
			"description":   description,
			"hookstatement": hookstatement,
			"keywords":      keywords,
		})
	})
	r.POST("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		name := session.Get("name").(string)
		userID := session.Get("user_id").(string)
		description := c.PostForm("description")
		hookstatement := c.PostForm("hookstatement")
		tags := c.PostFormArray("tag[]")
		_, err := dbpool.Exec(context.Background(), "INSERT INTO candidates (id, name, description, hookstatement, keywords) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(id) DO UPDATE SET id = $1, name = $2, description = $3, hookstatement = $4, keywords = $5", userID, name, description, hookstatement, tags)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to upsert candidate: %v", err)
			return
		} else {
			index(userID, name, description, hookstatement, tags)
		}
		c.Redirect(http.StatusSeeOther, "/"+name)
	})
	r.Run()
}
