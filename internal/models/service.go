package models

import (
	"time"
)

type Service struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	BusinessID   uint      `gorm:"not null;index" json:"business_id"`
	Name         string    `gorm:"not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	MinPrice     float64   `gorm:"not null" json:"min_price"`
	MaxPrice     float64   `gorm:"not null" json:"max_price"`
	IsNegotiable bool      `gorm:"default:false" json:"is_negotiable"`
	Duration     int       `gorm:"comment:duration in minutes" json:"duration"`
	ImageURL     string    `json:"image_url"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	LocationID   *uint     `gorm:"index;default:null" json:"location_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	Business     Business      `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	BookingItems []BookingItem `gorm:"foreignKey:ServiceID" json:"booking_items,omitempty"`
}
