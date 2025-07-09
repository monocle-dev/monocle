package models

import "gorm.io/gorm"

type Project struct {
	gorm.Model

	Name        string `gorm:"not null"`
	Description string
	OwnerID     uint `gorm:"not null;index"`

	// Relationships
	Owner              User                `gorm:"foreignKey:OwnerID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	ProjectMemberships []ProjectMembership `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Monitors           []Monitor           `gorm:"foreignKey:ProjectID"`
	NotificationRules  []NotificationRule  `gorm:"foreignKey:ProjectID"`
}
