package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/monocle-dev/monocle/internal/router"
)

func main() {
	var err error

	err = godotenv.Load()

	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
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
