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
	FacebookID         string    `gorm:"uniqueIndex;default:null" json:"facebook_id"`
	AvatarURL          string    `json:"avatar_url"`
	Country            string    `gorm:"default:'US'" json:"country"`
	Currency           string    `gorm:"default:'USD'" json:"currency"`
	SubscriptionPlanID *uint     `gorm:"default:null" json:"subscription_plan_id"`
	EmailVerified      bool      `gorm:"default:false" json:"email_verified"`
	VerificationToken  string    `gorm:"default:null" json:"-"`
	PaymentInstructions string   `gorm:"type:text" json:"payment_instructions"`
	OnboardingStep     int       `gorm:"default:0" json:"onboarding_step"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	AverageRating float64   `gorm:"default:0"`
	ReviewCount   int       `gorm:"default:0"`

	Clients      []Client              `gorm:"foreignKey:BusinessID" json:"clients,omitempty"`
	Subscription *BusinessSubscription `gorm:"foreignKey:BusinessID" json:"subscription,omitempty"`
	Reviews      []Review              `gorm:"foreignKey:BusinessID" json:"reviews,omitempty"`
}
