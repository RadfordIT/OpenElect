package main

import (
	"context"
	"fmt"
	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"log"
	"os"
)

var client *typesense.Client

func searchSetup() {
	client = typesense.NewClient(
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
}

func search(query string) []Candidate {
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
		return nil
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
	return candidates
}

func index(name string, description string, hookstatement string, keywords []string) {
	client.Collection("candidates").Documents().Create(context.Background(), &Candidate{
		Name:          name,
		Keywords:      keywords,
		HookStatement: hookstatement,
		Description:   description,
	})
}
