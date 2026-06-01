package payment

import (
	"salesmee/internal/models"
)

type PlanMeta struct {
	Name                 string
	Description          string
	PayPalProductID      string
	PayPalMonthlyPlanID  string
	PayPalYearlyPlanID   string
	Original             *models.SubscriptionPlan
}

type CheckoutContext struct {
	CustomerEmail   string
	BusinessID      uint
	BusinessName    string
	PlanCode        string
	PlanName        string
	BillingInterval string
	UnitAmount      float64
	Currency        string
	SuccessURL      string
	CancelURL       string
	Plan            *PlanMeta
	SavePlan        func(*PlanMeta) error
}

type CheckoutSession struct {
	ID  string
	URL string
}

type WebhookEvent struct {
	Type                      string
	CustomerID                string
	SubscriptionID            string
	SubscriptionStatus        string
	BusinessID                uint
	SubscriptionPlanCode      string
	BillingInterval           string
	CurrentPeriodStart        int64
	CurrentPeriodEnd          int64
	TrialEnd                  int64
}

type PaymentProvider interface {
	Name() string
	CreateCheckoutSession(ctx *CheckoutContext) (*CheckoutSession, error)
	CreateBillingPortalSession(customerID, returnURL string) (string, error)
	HandleWebhook(payload []byte, sigHeader string) (*WebhookEvent, error)
	GetOrCreateCustomer(business *models.Business) (string, error)
}
