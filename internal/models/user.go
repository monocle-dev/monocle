package models

import "gorm.io/gorm"

type User struct {
	gorm.Model

	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Name         string `gorm:"not null"`

	// Relationships
	OwnedProjects      []Project           `gorm:"foreignKey:OwnerID"`
	ProjectMemberships []ProjectMembership `gorm:"foreignKey:UserID"`
	Notifications      []Notification      `gorm:"foreignKey:UserID"`
	NotificationRules  []NotificationRule  `gorm:"foreignKey:UserID"`
}
