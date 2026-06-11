package models

import (
	"time"

	"gorm.io/gorm"
)

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
	PaymentInstructions  string   `gorm:"type:text" json:"payment_instructions"`
	OnboardingStep       int       `gorm:"default:0" json:"onboarding_step"`
	TimeZone             string    `gorm:"default:'UTC'" json:"timezone"`
	BufferTime           int       `gorm:"default:0" json:"buffer_time"`
	MaxBookingsPerSlot   int       `gorm:"default:1" json:"max_bookings_per_slot"`
	IsAcceptingBookings  bool      `gorm:"default:true" json:"is_accepting_bookings"`
	BusinessHours        string    `gorm:"type:jsonb;default:'{}'" json:"business_hours"`
	SpecialHours         string    `gorm:"type:jsonb;default:'[]'" json:"special_hours"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	AverageRating float64   `gorm:"default:0"`
	ReviewCount   int       `gorm:"default:0"`

	Clients      []Client              `gorm:"foreignKey:BusinessID" json:"clients,omitempty"`
	Subscription *BusinessSubscription `gorm:"foreignKey:BusinessID" json:"subscription,omitempty"`
	Reviews      []Review              `gorm:"foreignKey:BusinessID" json:"reviews,omitempty"`
	Locations    []Location            `gorm:"foreignKey:BusinessID" json:"locations,omitempty"`
	TeamMembers  []TeamMember          `gorm:"foreignKey:BusinessID" json:"team_members,omitempty"`
}

func (b *Business) BeforeCreate(tx *gorm.DB) error {
	if b.BusinessHours == "" || b.BusinessHours == "{}" {
		b.BusinessHours = `{"monday":[{"open":"08:00","close":"19:00"}],"tuesday":[{"open":"08:00","close":"19:00"}],"wednesday":[{"open":"08:00","close":"19:00"}],"thursday":[{"open":"08:00","close":"19:00"}],"friday":[{"open":"08:00","close":"19:00"}]}`
	}
	return nil
}
