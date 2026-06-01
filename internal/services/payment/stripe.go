package payment

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"salesmee/internal/models"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/billingportal/session"
	stripecheckout "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	stripewebhook "github.com/stripe/stripe-go/v76/webhook"
)

type StripeAdapter struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
}

func NewStripeAdapter() *StripeAdapter {
	sk := os.Getenv("STRIPE_SECRET_KEY")
	pk := os.Getenv("STRIPE_PUBLISHABLE_KEY")
	ws := os.Getenv("STRIPE_WEBHOOK_SECRET")

	if sk == "" || pk == "" {
		log.Println("WARNING: Stripe keys not fully configured. Payment features will fail at runtime.")
	}

	stripe.Key = sk

	return &StripeAdapter{
		secretKey:      sk,
		publishableKey: pk,
		webhookSecret:  ws,
	}
}

func (s *StripeAdapter) Name() string {
	return "stripe"
}

func (s *StripeAdapter) CreateCheckoutSession(ctx *CheckoutContext) (*CheckoutSession, error) {
	amount := int64(ctx.UnitAmount * 100)
	currency := ctx.Currency
	if currency == "" {
		currency = "usd"
	}

	mode := "subscription"

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(mode),
		Currency:   stripe.String(currency),
		SuccessURL: stripe.String(ctx.SuccessURL),
		CancelURL:  stripe.String(ctx.CancelURL),
		CustomerEmail: stripe.String(ctx.CustomerEmail),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(ctx.PlanName),
					},
					UnitAmount: stripe.Int64(amount),
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String(ctx.BillingInterval),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"business_id":       fmt.Sprintf("%d", ctx.BusinessID),
			"plan_code":         ctx.PlanCode,
			"billing_interval":  ctx.BillingInterval,
		},
		AllowPromotionCodes: stripe.Bool(true),
	}

	ss, err := stripecheckout.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe checkout session creation failed: %w", err)
	}

	return &CheckoutSession{
		ID:  ss.ID,
		URL: ss.URL,
	}, nil
}

func (s *StripeAdapter) CreateBillingPortalSession(customerID, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	ps, err := session.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe billing portal session creation failed: %w", err)
	}

	return ps.URL, nil
}

func (s *StripeAdapter) HandleWebhook(payload []byte, sigHeader string) (*WebhookEvent, error) {
	event, err := stripewebhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("stripe webhook signature verification failed: %w", err)
	}

	var webhookEvent WebhookEvent
	webhookEvent.Type = string(event.Type)

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal checkout session: %w", err)
		}
		webhookEvent.CustomerID = cs.Customer.ID

		if cs.Subscription != nil {
			webhookEvent.SubscriptionID = cs.Subscription.ID
		}
		if cs.Metadata != nil {
			if bid, ok := cs.Metadata["business_id"]; ok {
				fmt.Sscanf(bid, "%d", &webhookEvent.BusinessID)
			}
			webhookEvent.SubscriptionPlanCode = cs.Metadata["plan_code"]
			webhookEvent.BillingInterval = cs.Metadata["billing_interval"]
		}

	case stripe.EventTypeCustomerSubscriptionUpdated,
		stripe.EventTypeCustomerSubscriptionDeleted:
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
		webhookEvent.CustomerID = sub.Customer.ID
		webhookEvent.SubscriptionID = sub.ID
		webhookEvent.SubscriptionStatus = string(sub.Status)
		if sub.CurrentPeriodStart > 0 {
			webhookEvent.CurrentPeriodStart = sub.CurrentPeriodStart
		}
		if sub.CurrentPeriodEnd > 0 {
			webhookEvent.CurrentPeriodEnd = sub.CurrentPeriodEnd
		}
		if sub.TrialEnd > 0 {
			webhookEvent.TrialEnd = sub.TrialEnd
		}

	case stripe.EventTypeInvoicePaid:
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return nil, fmt.Errorf("failed to unmarshal invoice: %w", err)
		}
		webhookEvent.CustomerID = inv.Customer.ID
		if inv.Subscription != nil {
			webhookEvent.SubscriptionID = inv.Subscription.ID
		}
		webhookEvent.SubscriptionStatus = "active"

	case stripe.EventTypeInvoicePaymentFailed:
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return nil, fmt.Errorf("failed to unmarshal invoice: %w", err)
		}
		webhookEvent.CustomerID = inv.Customer.ID
		if inv.Subscription != nil {
			webhookEvent.SubscriptionID = inv.Subscription.ID
			webhookEvent.SubscriptionStatus = "past_due"
		}
	}

	return &webhookEvent, nil
}

func (s *StripeAdapter) GetOrCreateCustomer(business *models.Business) (string, error) {
	if business.Subscription != nil && business.Subscription.StripeCustomerID != "" {
		return business.Subscription.StripeCustomerID, nil
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(business.Email),
		Name:  stripe.String(business.Name),
		Metadata: map[string]string{
			"business_id": fmt.Sprintf("%d", business.ID),
		},
	}

	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe customer creation failed: %w", err)
	}

	return c.ID, nil
}

func init() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}
