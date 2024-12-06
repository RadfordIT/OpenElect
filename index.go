package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"log"
	"os"
)

func index() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	client := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
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
	client.Collection("candidates").Documents().Create(context.Background(), &Candidate{
		Name:          "John Doe",
		Keywords:      []string{"Software Engineer", "Golang", "Docker"},
		HookStatement: "I am a software engineer with 5 years of experience",
		Description:   "I am a software engineer with 5 years of experience in Golang and Docker",
	})
	client.Collection("candidates").Documents().Create(context.Background(), &Candidate{
		Name:          "Jane Doe",
		Keywords:      []string{"Software Engineer", "Python", "Kubernetes"},
		HookStatement: "I am a software engineer with 3 years of experience",
		Description:   "I am a software engineer with 3 years of experience in Python and Kubernetes",
	})
}
