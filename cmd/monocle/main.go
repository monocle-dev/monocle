package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/auth"
	"github.com/monocle-dev/monocle/internal/router"
)

func main() {
	var err error

	err = godotenv.Load()

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dsn := os.Getenv("POSTGRES_DSN")

	if dsn == "" {
		log.Fatal("POSTGRES_DSN environment variable is not set")
	}

	if err = db.ConnectDatabase(dsn); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.MigrateDatabase(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize JWT secret
	if err = auth.InitJWTSecret(); err != nil {
		log.Fatalf("Failed to initialize JWT secret: %v", err)
	}

	r := router.NewRouter()

	var port string

	if port = os.Getenv("PORT"); port == "" {
		port = "3000"
		log.Println("PORT not set, defaulting to 3000")
	}

	if err = r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
