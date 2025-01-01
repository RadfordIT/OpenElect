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
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"strings"
)

var r *gin.Engine
var dbpool *pgxpool.Pool
var configEditor, colorsEditor *viper.Viper

func main() {
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

	var err error
	dbpool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	authSetup()
	createTables()
	searchSetup()
	gob.Register(map[string]interface{}{})

	r = gin.Default()
	r.SetFuncMap(template.FuncMap{
		"contains": slices.Contains[[]string, string],
		"replace": func(toEscape string) string {
			return strings.ReplaceAll(toEscape, " ", "%20")
		},
	})
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/style.css", "./css/output.css")
	r.StaticFile("/icon.png", "./static/icon.png")
	r.LoadHTMLGlob("templates/*")
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("session", store))

	loginRoutes()
	adminRoutes()
	voteRoutes()

	r.GET("/", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		query := c.DefaultQuery("q", "")
		candidates := search(query)
		rand.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
		})
		flashes := session.Flashes()
		session.Save()
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"text":    candidates,
			"flashes": flashes,
		})
	})
	r.GET("/profile", candidateAuthMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		var userId string
		var description string
		var hookstatement string
		var keywords []string
		var positions []string
		dbpool.QueryRow(context.Background(), "SELECT * FROM candidates WHERE id = $1", session.Get("user_id")).Scan(&userId, nil, &description, &hookstatement, &keywords, &positions, nil)
		c.HTML(http.StatusOK, "profile.tmpl", gin.H{
			"userId":        userId,
			"description":   description,
			"hookstatement": hookstatement,
			"keywords":      keywords,
			"positions":     positions,
			"allpositions":  configEditor.GetStringSlice("positions"),
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
		_, err := dbpool.Exec(context.Background(),
			`INSERT INTO candidates 
    			(id, name, description, hookstatement, video, keywords, positions, published) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL)
				ON CONFLICT(id) DO UPDATE SET id = $1, name = $2, description = $3, hookstatement = $4, video = $5, keywords = $6, positions = $7, published = NULL`,
			userID, name, description, hookstatement, nil, tags, positions,
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
		_, err = dbpool.Exec(context.Background(), "UPDATE candidates SET published = FALSE WHERE name = $1", name)
		if err != nil {
			fmt.Println(err)
			c.String(http.StatusNotFound, "Candidate not found: %v", err)
			return
		}
		deindex(session.Get("user_id").(string))
		session.AddFlash("Your profile has been submitted for review.")
		err := session.Save()
		if err != nil {
			fmt.Println(err)
		}
		c.Redirect(http.StatusSeeOther, "/")
	})
	r.Run()
}
