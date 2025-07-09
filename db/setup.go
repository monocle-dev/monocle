package db

import (
	"log"

	"github.com/monocle-dev/monocle/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(dsn string) {
	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	log.Println("Database connection established successfully")
}

func MigrateDatabase() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.ProjectMembership{},
		&models.Monitor{},
		&models.MonitorCheck{},
		&models.Incident{},
		&models.Notification{},
		&models.NotificationRule{},
	)

	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database migration completed successfully")
}
