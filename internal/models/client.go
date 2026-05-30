package models

import "time"

type ClientStatus string

const (
	StatusNew             ClientStatus = "new"
	StatusActive          ClientStatus = "active"
	StatusAwaitingPayment ClientStatus = "awaiting_payment"
	StatusCompleted       ClientStatus = "completed"
)

type Client struct {
	ID             uint         `gorm:"primaryKey" json:"id"`
	BusinessID     *uint        `gorm:"column:business_id;index" json:"business_id"`
	Name           string       `gorm:"not null" json:"name"`
	Email          string       `json:"email"`
	Phone          string       `json:"phone"`
	Status         ClientStatus `gorm:"default:'new'" json:"status"`
	IsOnline       bool         `gorm:"default:false" json:"is_online"`
	LastSeenAt     *time.Time   `json:"last_seen_at"`
	ConversationID uint         `json:"conversation_id"`
	GoogleID       string       `gorm:"uniqueIndex;default:null" json:"google_id"`
	FacebookID     string       `gorm:"uniqueIndex;default:null" json:"facebook_id"`
	AvatarURL      string       `json:"avatar_url"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`

	Business     Business     `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Conversation Conversation `gorm:"foreignKey:ClientID" json:"conversation,omitempty"`
}
