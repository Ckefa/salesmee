package business

import (
	"salesmee/internal/ws"
	"gorm.io/gorm"
)

type HandlerDeps struct {
	DB  *gorm.DB
	Hub *ws.Hub
}

type ProductHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type ServiceHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type OrderHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type BookingHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type PaymentHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type AnalyticsHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type ReportHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type HoursHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type LocationHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type TeamHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type ReviewHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type SubscriptionHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type AssistHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

type NotificationHandler struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewProductHandler(deps *HandlerDeps) *ProductHandler {
	return &ProductHandler{db: deps.DB, hub: deps.Hub}
}

func NewServiceHandler(deps *HandlerDeps) *ServiceHandler {
	return &ServiceHandler{db: deps.DB, hub: deps.Hub}
}

func NewOrderHandler(deps *HandlerDeps) *OrderHandler {
	return &OrderHandler{db: deps.DB, hub: deps.Hub}
}

func NewBookingHandler(deps *HandlerDeps) *BookingHandler {
	return &BookingHandler{db: deps.DB, hub: deps.Hub}
}

func NewPaymentHandler(deps *HandlerDeps) *PaymentHandler {
	return &PaymentHandler{db: deps.DB, hub: deps.Hub}
}

func NewAnalyticsHandler(deps *HandlerDeps) *AnalyticsHandler {
	return &AnalyticsHandler{db: deps.DB, hub: deps.Hub}
}

func NewReportHandler(deps *HandlerDeps) *ReportHandler {
	return &ReportHandler{db: deps.DB, hub: deps.Hub}
}

func NewHoursHandler(deps *HandlerDeps) *HoursHandler {
	return &HoursHandler{db: deps.DB, hub: deps.Hub}
}

func NewLocationHandler(deps *HandlerDeps) *LocationHandler {
	return &LocationHandler{db: deps.DB, hub: deps.Hub}
}

func NewTeamHandler(deps *HandlerDeps) *TeamHandler {
	return &TeamHandler{db: deps.DB, hub: deps.Hub}
}

func NewReviewHandler(deps *HandlerDeps) *ReviewHandler {
	return &ReviewHandler{db: deps.DB, hub: deps.Hub}
}

func NewSubscriptionHandler(deps *HandlerDeps) *SubscriptionHandler {
	return &SubscriptionHandler{db: deps.DB, hub: deps.Hub}
}

func NewAssistHandler(deps *HandlerDeps) *AssistHandler {
	return &AssistHandler{db: deps.DB, hub: deps.Hub}
}

func NewNotificationHandler(deps *HandlerDeps) *NotificationHandler {
	return &NotificationHandler{db: deps.DB, hub: deps.Hub}
}
