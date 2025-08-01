package models

type Project struct {
	BaseModel

	Name           string `gorm:"not null"`
	Description    string
	OwnerID        uint `gorm:"not null;index"`
	DiscordWebhook string
	SlackWebhook   string

	// Relationships
	Owner              User                `gorm:"foreignKey:OwnerID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	ProjectMemberships []ProjectMembership `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Monitors           []Monitor           `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	NotificationRules  []NotificationRule  `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
