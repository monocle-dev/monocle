package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/monocle-dev/monocle/db"
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

	db.ConnectDatabase(dsn)
	db.MigrateDatabase()

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
