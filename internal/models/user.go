package models

type User struct {
	BaseModel

	Name         string `gorm:"not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`

	// Relationships
	OwnedProjects      []Project           `gorm:"foreignKey:OwnerID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	ProjectMemberships []ProjectMembership `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Notifications      []Notification      `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	NotificationRules  []NotificationRule  `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
