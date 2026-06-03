package models

import (
	"time"
)

type PaymentMethod struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	BusinessID uint           `gorm:"not null;index" json:"business_id"`
	MethodType string         `gorm:"not null" json:"method_type"`
	Label      string         `gorm:"not null" json:"label"`
	Details    map[string]any `gorm:"type:jsonb;serializer:json" json:"details"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	SortOrder  int            `gorm:"default:0" json:"sort_order"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`

	Business Business `gorm:"foreignKey:BusinessID" json:"-"`
}
