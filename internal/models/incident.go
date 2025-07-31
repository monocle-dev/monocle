package models

import (
	"time"
)

type Incident struct {
	BaseModel

	MonitorID   uint   `gorm:"not null;index"`
	Status      string `gorm:"not null"` // e.g., "Active", "Resolved"
	Title       string `gorm:"not null"`
	Description string
	StartedAt   *time.Time
	ResolvedAt  *time.Time

	// Relationships
	Monitor       Monitor        `gorm:"foreignKey:MonitorID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Notifications []Notification `gorm:"foreignKey:IncidentID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
