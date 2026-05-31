package models

import "time"

type PasswordResetToken struct {
	ID         uint      `gorm:"primaryKey"`
	BusinessID uint      `gorm:"not null;index"`
	Email      string    `gorm:"not null"`
	Token      string    `gorm:"uniqueIndex;not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	Used       bool      `gorm:"default:false"`
	CreatedAt  time.Time
	Business   Business `gorm:"foreignKey:BusinessID"`
}
