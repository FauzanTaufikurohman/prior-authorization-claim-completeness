package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/database"
	"github.com/FauzanTaufikurohman/prior-authorization-assistant/backend/internal/routes"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Failed to load .env: %v", err)
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	log.Println("PostgreSQL connection established")
	log.Printf("Prior Authorization Assistant API starting on :%s", port)

	router := routes.SetupRouter(pool)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}
