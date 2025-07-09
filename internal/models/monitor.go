package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Monitor struct {
	gorm.Model

	ProjectID uint           `gorm:"not null"` // Foreign key to the Project
	Project   Project        `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Name      string         `gorm:"not null"`
	Type      string         `gorm:"not null"` // "http", "ping", "database", etc.
	Status    string         `gorm:"not null"` // "active", "inactive", "error", etc.
	Interval  int            `gorm:"not null"` // Interval in seconds for the monitor to run'
	Config    datatypes.JSON `gorm:"type:jsonb"`

	// Relationships
	MonitorChecks []MonitorCheck `gorm:"foreignKey:MonitorID"`
	Incidents     []Incident     `gorm:"foreignKey:MonitorID"`
}
