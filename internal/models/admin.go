package models

import "time"

type Admin struct {
	ID        uint      `gorm:"primaryKey"`
	Email     string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null"`
	Name      string    `gorm:"not null"`
	Role      string    `gorm:"default:'admin'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey"`
	AdminID    uint      `gorm:"index"`
	Action     string    `gorm:"not null"`
	Resource   string    `gorm:"not null"`
	ResourceID string
	Details    string    `gorm:"type:text"`
	IP         string
	CreatedAt  time.Time
	Admin      Admin     `gorm:"foreignKey:AdminID"`
}
