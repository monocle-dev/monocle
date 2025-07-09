package models

import (
	"time"

	"gorm.io/gorm"
)

type Incident struct {
	gorm.Model

	MonitorID   uint   `gorm:"not null;index"`
	Status      string `gorm:"not null"`
	Severity    string `gorm:"not null"`
	Title       string `gorm:"not null"`
	Description string
	StartedAt   *time.Time
	ResolvedAt  *time.Time

	// Relationships
	Monitor       Monitor        `gorm:"foreignKey:MonitorID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Notifications []Notification `gorm:"foreignKey:IncidentID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
