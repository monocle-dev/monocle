package models

import (
	"gorm.io/datatypes"
)

type Monitor struct {
	BaseModel

	ProjectID uint           `gorm:"not null;index"` // Foreign key to the Project
	Name      string         `gorm:"not null"`
	Type      string         `gorm:"not null"` // "http", "ping", "database", etc.
	Status    string         `gorm:"not null"` // "active", "inactive", "error", etc.
	Interval  int            `gorm:"not null"` // Interval in seconds for the monitor to run
	Config    datatypes.JSON `gorm:"type:jsonb"`

	// Relationships
	Project       Project        `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	MonitorChecks []MonitorCheck `gorm:"foreignKey:MonitorID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Incidents     []Incident     `gorm:"foreignKey:MonitorID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
