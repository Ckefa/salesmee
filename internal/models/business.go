package models

import "time"

type Business struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Email              string    `gorm:"unique;not null" json:"email"`
	Password           *string   `gorm:"default:null" json:"-"`
	Name               string    `json:"name"`
	Username           string    `json:"username"`
	BusinessType       string    `json:"business_type"`
	Slug               string    `gorm:"unique;index" json:"slug"`
	IsPublic           bool      `gorm:"default:true" json:"is_public"`
	Logo               string    `json:"logo"`
	GoogleID           string    `gorm:"uniqueIndex;default:null" json:"google_id"`
	AvatarURL          string    `json:"avatar_url"`
	SubscriptionPlanID *uint     `gorm:"default:null" json:"subscription_plan_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	Clients      []Client             `gorm:"foreignKey:BusinessID" json:"clients,omitempty"`
	Subscription *BusinessSubscription `gorm:"foreignKey:BusinessID" json:"subscription,omitempty"`
}
