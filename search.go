package main

import (
	"context"
	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"log"
	"os"
)

var client *typesense.Client

type Candidate struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Keywords      []string `json:"keywords"`
	HookStatement string   `json:"hookstatement"`
	Description   string   `json:"description"`
	Positions     []string `json:"positions"`
}

func toStringSlice(input []interface{}) []string {
	output := make([]string, len(input))
	for i, v := range input {
		output[i] = v.(string)
	}
	return output
}

func searchSetup() {
	client = typesense.NewClient(
		typesense.WithServer(os.Getenv("TYPESENSE_URL")),
		typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")))
	schema := &api.CollectionSchema{
		Name: "candidates",
		Fields: []api.Field{
			{
				Name: "id",
				Type: "string",
			},
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
			{
				Name: "positions",
				Type: "string[]",
			},
		},
	}
	//client.Collection("candidates").Delete(context.Background())
	client.Collections().Create(context.Background(), schema)
}

func search(query string) []Candidate {
	searchParameters := &api.SearchCollectionParams{
		Q:       pointer.String(query),
		QueryBy: pointer.String("name,keywords,hookstatement,description,positions"),
	}
	results, err := client.Collection("candidates").Documents().Search(context.Background(), searchParameters)
	if err != nil {
		log.Fatal(err)
	}
	if results.Hits == nil {
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
			Positions:     toStringSlice(document["positions"].([]interface{})),
		})
	}
	return candidates
}

func index(id string, name string, description string, hookstatement string, keywords []string, positions []string) {
	client.Collection("candidates").Documents().Upsert(context.Background(), &Candidate{
		ID:            id,
		Name:          name,
		Keywords:      keywords,
		HookStatement: hookstatement,
		Description:   description,
		Positions:     positions,
	})
}

func deindex(id string) {
	client.Collection("candidates").Document(id).Delete(context.Background())
}
