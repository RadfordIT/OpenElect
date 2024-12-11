package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

var dbpool *pgxpool.Pool

func dbSetup() {
	var err error
	dbpool, err = pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()
	dbpool.Exec(context.Background(), "DROP TABLE IF EXISTS candidates")
	dbpool.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS candidates (id TEXT NOT NULL PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL CHECK (char_length(description) <= 1500), hookstatement TEXT NOT NULL CHECK (char_length(hookstatement) <= 150), keywords TEXT[] CHECK (array_length(keywords, 1) <= 6))")
}
