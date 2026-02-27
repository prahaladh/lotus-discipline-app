package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// db is a global connection pool used by handlers.
var db *pgxpool.Pool

// initDB initializes the PostgreSQL connection pool.
func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("WARNING: DATABASE_URL is not set. API will start but DB-backed endpoints will fail.")
		return
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to create db pool: %v", err)
	}

	db = pool
}

