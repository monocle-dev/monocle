package models

import (
	"time"
)

type MonitorCheck struct {
	BaseModel

	MonitorID    uint   `gorm:"not null;index"`
	Status       string `gorm:"not null"`
	ResponseTime int    `gorm:"not null"`
	Message      string
	CheckedAt    time.Time `gorm:"not null"`

	// Relationships
	Monitor Monitor `gorm:"foreignKey:MonitorID;constraint:OnUpdate:Cascade,OnDelete:CASCADE" json:"-"`
}
