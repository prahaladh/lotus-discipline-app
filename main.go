package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	initDB()
	InitRedis()

	router := setupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := router.Run(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

