package models

type ProjectMembership struct {
	BaseModel

	UserID    uint   `gorm:"not null;uniqueIndex:idx_user_project"`
	ProjectID uint   `gorm:"not null;uniqueIndex:idx_user_project"`
	Role      string `gorm:"not null"`

	// Relationships
	User    User    `gorm:"foreignKey:UserID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
	Project Project `gorm:"foreignKey:ProjectID;constraint:OnUpdate:Cascade,OnDelete:CASCADE"`
}
