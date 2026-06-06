package models

import (
	"time"

	"gorm.io/gorm"
)

type TeamMember struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	BusinessID  uint           `gorm:"not null;index" json:"business_id"`
	Name        string         `gorm:"not null" json:"name"`
	Email       string         `gorm:"not null;uniqueIndex:idx_team_email_business,priority:1" json:"email"`
	Password    string         `json:"-"`
	Role        string         `gorm:"not null;default:'staff'" json:"role"` // manager, staff
	Phone       string         `json:"phone"`
	Photo       string         `json:"photo"`
	Permissions string         `gorm:"type:jsonb;default:'{}'" json:"permissions"`
	InviteToken string         `gorm:"default:null;uniqueIndex" json:"-"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	LastLoginAt *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Business  Business   `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Locations []Location `gorm:"many2many:team_member_locations;" json:"locations,omitempty"`
}
