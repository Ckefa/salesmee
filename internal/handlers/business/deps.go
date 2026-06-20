package business

import (
	"salesmee/internal/services/cache"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var wsHub *ws.Hub

func SetWSHub(hub *ws.Hub) {
	wsHub = hub
}

type dbProvider struct {
	db     *gorm.DB
	hub    *ws.Hub
	fcache *cache.FragmentCache
}

func (p *dbProvider) dbc(c *gin.Context) *gorm.DB {
	return p.db.WithContext(c.Request.Context())
}

type HandlerDeps struct {
	DB     *gorm.DB
	Hub    *ws.Hub
	FCache *cache.FragmentCache
}

type ProductHandler struct {
	dbProvider
}

type ServiceHandler struct {
	dbProvider
}

type OrderHandler struct {
	dbProvider
}

type BookingHandler struct {
	dbProvider
}

type PaymentHandler struct {
	dbProvider
}

type AnalyticsHandler struct {
	dbProvider
}

type ReportHandler struct {
	dbProvider
}

type HoursHandler struct {
	dbProvider
}

type LocationHandler struct {
	dbProvider
}

type TeamHandler struct {
	dbProvider
}

type ReviewHandler struct {
	dbProvider
}

type SubscriptionHandler struct {
	dbProvider
}

type AssistHandler struct {
	dbProvider
}

type NotificationHandler struct {
	dbProvider
}

func NewProductHandler(deps *HandlerDeps) *ProductHandler {
	return &ProductHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewServiceHandler(deps *HandlerDeps) *ServiceHandler {
	return &ServiceHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewOrderHandler(deps *HandlerDeps) *OrderHandler {
	return &OrderHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewBookingHandler(deps *HandlerDeps) *BookingHandler {
	return &BookingHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewPaymentHandler(deps *HandlerDeps) *PaymentHandler {
	return &PaymentHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewAnalyticsHandler(deps *HandlerDeps) *AnalyticsHandler {
	return &AnalyticsHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewReportHandler(deps *HandlerDeps) *ReportHandler {
	return &ReportHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewHoursHandler(deps *HandlerDeps) *HoursHandler {
	return &HoursHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewLocationHandler(deps *HandlerDeps) *LocationHandler {
	return &LocationHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewTeamHandler(deps *HandlerDeps) *TeamHandler {
	return &TeamHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewReviewHandler(deps *HandlerDeps) *ReviewHandler {
	return &ReviewHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewSubscriptionHandler(deps *HandlerDeps) *SubscriptionHandler {
	return &SubscriptionHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewAssistHandler(deps *HandlerDeps) *AssistHandler {
	return &AssistHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}

func NewNotificationHandler(deps *HandlerDeps) *NotificationHandler {
	return &NotificationHandler{dbProvider{db: deps.DB, hub: deps.Hub, fcache: deps.FCache}}
}
