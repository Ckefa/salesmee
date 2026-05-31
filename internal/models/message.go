package models

import "time"

type Message struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ConversationID uint       `gorm:"not null;index" json:"conversation_id"`
	Content        string     `gorm:"not null" json:"content"`
	Type           string     `gorm:"not null;default:'message'" json:"type"` // "message", "order", "booking"
	Sender         string     `gorm:"not null" json:"sender"`                 // "user" or "client"
	MediaURL       string     `json:"media_url"`                              // "uploads/media/...", empty if text-only
	MediaType      string     `json:"media_type"`                             // "image", "document", "audio"
	ReadByBusiness bool       `gorm:"default:false" json:"read_by_business"`  // true when business has seen it
	ReadAt         *time.Time `json:"read_at"`                                // when business read it
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Actions      []Action     `gorm:"foreignKey:MessageID" json:"actions,omitempty"`
}
