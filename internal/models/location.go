package models

import "time"

type Location struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	BusinessID uint      `gorm:"not null;index" json:"business_id"`
	Name       string    `gorm:"not null" json:"name"`
	Address    string    `json:"address"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	TimeZone   string    `gorm:"default:'UTC'" json:"timezone"`
	Lat        float64   `gorm:"default:0" json:"lat"`
	Lng        float64   `gorm:"default:0" json:"lng"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	SortOrder  int       `gorm:"default:0" json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Business Business `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
}
