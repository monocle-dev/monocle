package models

import (
	"gorm.io/datatypes"
)

type NotificationRule struct {
	BaseModel

	ProjectID   uint           `gorm:"not null;index"`
	UserID      uint           `gorm:"not null;index"`
	TriggerType string         `gorm:"not null"` // e.g., "incident_created", "incident_resolved"
	Channel     string         `gorm:"not null"` // e.g., "email", "slack", "webhook"
	IsActive    bool           `gorm:"default:true"`
	Config      datatypes.JSON `gorm:"type:jsonb"`

	// Relationships
	Project Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	User    User    `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
