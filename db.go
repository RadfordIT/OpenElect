package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

func dbSetup() {
	var err error
	dbpool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	//dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS candidates")
	//dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS votes")
	dbpool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS candidates (
    	id TEXT NOT NULL PRIMARY KEY, 
    	name TEXT NOT NULL, 
    	description TEXT NOT NULL CHECK (char_length(description) <= 5000), 
    	hookstatement TEXT NOT NULL CHECK (char_length(hookstatement) <= 150), 
    	video TEXT DEFAULT NULL,
    	keywords TEXT[] CHECK (array_length(keywords, 1) <= 6), 
    	positions TEXT[] CHECK (array_length(positions, 1) >= 1),
    	published BOOLEAN DEFAULT NULL
    )`)
	dbpool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS votes (
    	vote_id SERIAL PRIMARY KEY,
    	candidate TEXT NOT NULL CHECK (char_length(candidate) > 0),
    	candidate_id TEXT NOT NULL CHECK (char_length(candidate_id) > 0),
    	voter_id TEXT NOT NULL CHECK (char_length(voter_id) > 0),
    	position TEXT NOT NULL CHECK (char_length(position) > 0),
    	UNIQUE(candidate_id, voter_id, position)
    )`)
}
