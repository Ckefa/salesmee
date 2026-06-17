package models

// Order status constants
const (
	OrderDraft           = "draft"
	OrderPending         = "pending"
	OrderClientConfirmed = "client_confirmed"
	OrderConfirmed       = "confirmed"
	OrderFulfilled       = "fulfilled"
	OrderCompleted       = "completed"
	OrderCancelled       = "cancelled"
)

// Booking status constants
const (
	BookingPending         = "pending"
	BookingClientConfirmed = "client_confirmed"
	BookingConfirmed       = "confirmed"
	BookingCompleted       = "completed"
	BookingCancelled       = "cancelled"
)

// Payment status constants
const (
	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentFailed    = "failed"
)

// Order sender constants
const (
	SenderBusiness = "business"
	SenderClient   = "client"
)

// Payment method constants
const (
	PayMethodCash        = "cash"
	PayMethodCard        = "card"
	PayMethodBank        = "bank_transfer"
	PayMethodMobileMoney = "mobile_money"
)

// Inventory log type
const (
	InvLogOut = "out"
)
