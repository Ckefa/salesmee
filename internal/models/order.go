package models

import (
	"time"
)

type Order struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	BusinessID         uint       `gorm:"not null;index" json:"business_id"`
	ClientID           uint       `gorm:"not null;index" json:"client_id"`
	OrderNumber        string     `gorm:"unique;not null" json:"order_number"`
	Status             string `gorm:"default:'draft'" json:"status"`
	Sender             string `gorm:"default:'client'" json:"sender"`
	Quantity           int        `gorm:"default:1" json:"quantity"`
	TotalAmount        float64    `gorm:"not null" json:"total_amount"`
	PaidAmount         float64    `gorm:"default:0" json:"paid_amount"`
	DeliveryDate       *time.Time `json:"delivery_date"`
	Notes              string     `gorm:"type:text" json:"notes"`
	ConfirmedByClient   bool       `gorm:"default:false" json:"confirmed_by_client"`
	ConfirmedByBusiness bool      `gorm:"default:false" json:"confirmed_by_business"`
	HiddenFromChat      bool      `gorm:"default:false" json:"hidden_from_chat"`
	LocationID          *uint     `gorm:"index;default:null" json:"location_id,omitempty"`
	ConfirmedByClientAt *time.Time `json:"confirmed_by_client_at,omitempty"`
	ConfirmedByBusinessAt *time.Time `json:"confirmed_by_business_at,omitempty"`
	Draft              bool       `gorm:"default:true" json:"draft"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Relationships
	Business   Business    `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Client     Client      `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID" json:"order_items,omitempty"`
	Payments   []Payment   `gorm:"foreignKey:OrderID" json:"payments,omitempty"`
}

type OrderItem struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	OrderID         uint    `gorm:"not null;index" json:"order_id"`
	ProductID       uint    `gorm:"not null;index" json:"product_id"`
	Quantity        int     `gorm:"not null" json:"quantity"`
	UnitPrice       float64 `gorm:"not null" json:"unit_price"`
	TotalPrice      float64 `gorm:"not null" json:"total_price"`
	StaffID         *uint   `gorm:"index;default:null" json:"staff_id,omitempty"`
	CommissionType  string  `gorm:"default:'percentage'" json:"commission_type"`
	CommissionValue float64 `gorm:"default:0" json:"commission_value"`
	CommissionEarned float64 `gorm:"default:0" json:"commission_earned"`

	// Relationships
	Order   Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Staff   *TeamMember `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}

type Booking struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	BusinessID    uint      `gorm:"not null;index" json:"business_id"`
	ClientID      uint      `gorm:"not null;index" json:"client_id"`
	BookingNumber string    `gorm:"unique;not null" json:"booking_number"`
	Status        string `gorm:"default:'pending'" json:"status"`
	Sender        string `gorm:"default:'client'" json:"sender"`
	ScheduledDate time.Time `gorm:"not null" json:"scheduled_date"`
	Duration      int       `gorm:"not null" json:"duration"` // in minutes
	TotalAmount   float64   `gorm:"not null" json:"total_amount"`
	PaidAmount    float64   `gorm:"default:0" json:"paid_amount"`
	Notes           string    `gorm:"type:text" json:"notes"`
	HiddenFromChat  bool      `gorm:"default:false" json:"hidden_from_chat"`
	LocationID      *uint     `gorm:"index;default:null" json:"location_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Business     Business      `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	Client       Client        `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	BookingItems []BookingItem `gorm:"foreignKey:BookingID" json:"booking_items,omitempty"`
	Payments     []Payment     `gorm:"foreignKey:BookingID" json:"payments,omitempty"`
}

type BookingItem struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	BookingID       uint    `gorm:"not null;index" json:"booking_id"`
	ServiceID       uint    `gorm:"not null;index" json:"service_id"`
	Quantity        int     `gorm:"default:1" json:"quantity"`
	UnitPrice       float64 `gorm:"not null" json:"unit_price"`
	TotalPrice      float64 `gorm:"not null" json:"total_price"`
	StaffID         *uint   `gorm:"index;default:null" json:"staff_id,omitempty"`
	CommissionType  string  `gorm:"default:'percentage'" json:"commission_type"`
	CommissionValue float64 `gorm:"default:0" json:"commission_value"`
	CommissionEarned float64 `gorm:"default:0" json:"commission_earned"`

	// Relationships
	Booking Booking     `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
	Service Service     `gorm:"foreignKey:ServiceID" json:"service,omitempty"`
	Staff   *TeamMember `gorm:"foreignKey:StaffID" json:"staff,omitempty"`
}

type Payment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   *uint     `gorm:"index" json:"order_id,omitempty"`
	BookingID *uint     `gorm:"index" json:"booking_id,omitempty"`
	ClientID  uint      `gorm:"not null;index" json:"client_id"`
	Amount    float64   `gorm:"not null" json:"amount"`
	Method    string    `gorm:"not null" json:"method"`
	Status    string    `gorm:"default:'pending'" json:"status"`
	Reference string    `json:"reference"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Order   Order   `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Booking Booking `gorm:"foreignKey:BookingID" json:"booking,omitempty"`
	Client  Client  `gorm:"foreignKey:ClientID" json:"client,omitempty"`
}
