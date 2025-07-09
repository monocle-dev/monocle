package db

import (
	"github.com/monocle-dev/monocle/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(dsn string) error {
	var err error

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return err
	}

	return nil
}

func MigrateDatabase() error {
	models := []interface{}{
		&models.User{},
		&models.Project{},
		&models.ProjectMembership{},
		&models.Monitor{},
		&models.MonitorCheck{},
		&models.Incident{},
		&models.Notification{},
		&models.NotificationRule{},
	}

	migrator := DB.Migrator()

	for _, model := range models {
		if !migrator.HasTable(model) {
			if err := DB.AutoMigrate(model); err != nil {
				return err
			}
		}
	}

	return nil
}
