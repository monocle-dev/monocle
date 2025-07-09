package models

import (
	"time"

	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model

	IncidentID uint   `gorm:"not null;index"`
	UserID     uint   `gorm:"not null;index"`
	Channel    string `gorm:"not null"`
	Status     string `gorm:"not null"`
	Message    string
	SentAt     *time.Time

	// Relationships
	Incident Incident `gorm:"foreignKey:IncidentID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	User     User     `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
