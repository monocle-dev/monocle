package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/auth"
	"github.com/monocle-dev/monocle/internal/handlers"
	"github.com/monocle-dev/monocle/internal/router"
	"github.com/monocle-dev/monocle/internal/scheduler"
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

	if err = scheduler.Initialize(); err != nil {
		log.Fatalf("Failed to initialize scheduler: %v", err)
	}

	r := router.NewRouter()

	scheduler.SetBroadcastCallback(func(projectID string) {
		handlers.BroadCastRefresh(projectID)
	})

	var port string

	if port = os.Getenv("PORT"); port == "" {
		port = "3000"
		log.Println("PORT not set, defaulting to 3000")
	}

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		if err = r.Run(":" + port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
	scheduler.Shutdown()
}
