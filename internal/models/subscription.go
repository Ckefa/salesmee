package models

import "time"

type SubscriptionPlan struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Code                  string    `gorm:"unique;not null" json:"code"`
	Name                  string    `gorm:"not null" json:"name"`
	Description           string    `json:"description"`
	PriceMonthly          float64   `gorm:"not null" json:"price_monthly"`
	PriceYearly           float64   `gorm:"not null" json:"price_yearly"`
	Currency              string    `gorm:"default:'usd'" json:"currency"`
	MaxClients            int       `gorm:"default:0" json:"max_clients"`
	MaxProducts           int       `gorm:"default:0" json:"max_products"`
	MaxServices           int       `gorm:"default:0" json:"max_services"`
	MaxConversations      int       `gorm:"default:0" json:"max_conversations"`
	HasAnalytics          bool      `gorm:"default:false" json:"has_analytics"`
	HasMediaSharing       bool      `gorm:"default:false" json:"has_media_sharing"`
	HasPrioritySupport    bool      `gorm:"default:false" json:"has_priority_support"`
	HasOrdersAndBookings  bool      `gorm:"default:false" json:"has_orders_and_bookings"`
	SortOrder             int       `gorm:"default:0" json:"sort_order"`
	StripeMonthlyPriceID  string    `json:"stripe_monthly_price_id"`
	StripeYearlyPriceID   string    `json:"stripe_yearly_price_id"`
	IsActive              bool      `gorm:"default:true" json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	Businesses    []Business           `gorm:"foreignKey:SubscriptionPlanID" json:"businesses,omitempty"`
	Subscriptions []BusinessSubscription `gorm:"foreignKey:PlanID" json:"subscriptions,omitempty"`
}

type BusinessSubscription struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	BusinessID           uint       `gorm:"not null;index" json:"business_id"`
	PlanID               uint       `gorm:"not null" json:"plan_id"`
	Status               string     `gorm:"default:'trialing'" json:"status"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	StripeCustomerID     string     `json:"stripe_customer_id"`
	BillingInterval      string     `gorm:"default:'month'" json:"billing_interval"`
	CurrentPeriodStart   time.Time  `json:"current_period_start"`
	CurrentPeriodEnd     time.Time  `json:"current_period_end"`
	TrialEndsAt          *time.Time `json:"trial_ends_at"`
	CanceledAt           *time.Time `json:"canceled_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	Business Business          `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Plan     SubscriptionPlan  `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}
