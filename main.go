package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"log"
	"net/http"
	"os"
)

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
	// err := godotenv.Load()
	// if err != nil {
	// log.Fatal("Error loading .env file")
	// }

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	client := typesense.NewClient(
		typesense.WithServer(os.Getenv("TYPESENSE_URL")),
		typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	schema := &api.CollectionSchema{
		Name: "candidates",
		Fields: []api.Field{
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "keywords",
				Type: "string[]",
			},
			{
				Name: "hookstatement",
				Type: "string",
			},
			{
				Name: "description",
				Type: "string",
			},
		},
	}
	client.Collections().Create(context.Background(), schema)

	r := gin.Default()
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.StaticFile("/style.css", "./css/output.css")
	r.StaticFile("/icon.png", "./static/icon.png")
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		query := c.DefaultQuery("q", "")
		searchParameters := &api.SearchCollectionParams{
			Q:       pointer.String(query),
			QueryBy: pointer.String("name,keywords,hookstatement,description"),
		}
		results, err := client.Collection("candidates").Documents().Search(context.Background(), searchParameters)
		if err != nil {
			log.Fatal(err)
		}
		if results.Hits == nil {
			fmt.Println("No results found")
			return
		}
		var candidates []Candidate
		for _, hit := range *results.Hits {
			document := *hit.Document
			candidates = append(candidates, Candidate{
				Name:          document["name"].(string),
				Keywords:      toStringSlice(document["keywords"].([]interface{})),
				HookStatement: document["hookstatement"].(string),
				Description:   document["description"].(string),
			})
		}
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"text": candidates,
		})
		return
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/:candidate", func(c *gin.Context) {
		name := c.Param("candidate")
		var description string
		var hookstatement string
		err = dbpool.QueryRow(context.Background(), "SELECT name FROM candidates WHERE name = $1", name).Scan(&description, &hookstatement)
		fmt.Println(description, hookstatement)
		c.HTML(http.StatusOK, "candidate.tmpl", gin.H{
			"name": name,
		})
	})
	r.Run()
}
