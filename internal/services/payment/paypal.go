package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"salesmee/internal/models"

	paypal "github.com/plutov/paypal/v4"
)

type PayPalAdapter struct {
	client      *paypal.Client
	webhookID   string
	clientID    string
	clientSecret string
	apiBase     string
}

func NewPayPalAdapter() *PayPalAdapter {
	clientID := os.Getenv("PAYPAL_CLIENT_ID")
	secret := os.Getenv("PAYPAL_CLIENT_SECRET")
	webhookID := os.Getenv("PAYPAL_WEBHOOK_ID")

	apiBase := paypal.APIBaseLive
	if os.Getenv("PAYPAL_SANDBOX") == "true" {
		apiBase = paypal.APIBaseSandBox
	}

	client, err := paypal.NewClient(clientID, secret, apiBase)
	if err != nil {
		log.Printf("WARNING: Failed to create PayPal client: %v", err)
		return &PayPalAdapter{webhookID: webhookID}
	}

	client.SetReturnRepresentation()

	return &PayPalAdapter{
		client:       client,
		webhookID:    webhookID,
		clientID:     clientID,
		clientSecret: secret,
		apiBase:      apiBase,
	}
}

func (p *PayPalAdapter) Name() string {
	return "paypal"
}

func (p *PayPalAdapter) CreateCheckoutSession(ctx *CheckoutContext) (*CheckoutSession, error) {
	if p.client == nil {
		return nil, fmt.Errorf("PayPal client not initialized")
	}

	plan := ctx.Plan
	if plan == nil {
		return nil, fmt.Errorf("PayPal adapter requires PlanMeta in context")
	}

	// 1. Create or reuse PayPal product
	productID := plan.PayPalProductID
	if productID == "" {
		prod, err := p.client.CreateProduct(context.Background(), paypal.Product{
			Name:        plan.Name + " Subscription",
			Description: plan.Description,
			Type:        paypal.ProductTypeService,
			Category:    paypal.ProductCategorySoftwareOnlineServices,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create PayPal product: %w", err)
		}
		productID = prod.ID
		if ctx.SavePlan != nil {
			plan.PayPalProductID = productID
			if err := ctx.SavePlan(plan); err != nil {
				return nil, fmt.Errorf("failed to save PayPal product ID: %w", err)
			}
		}
	}

	// 2. Create or reuse PayPal billing plan
	var planID string
	if ctx.BillingInterval == "year" {
		planID = plan.PayPalYearlyPlanID
	} else {
		planID = plan.PayPalMonthlyPlanID
	}

	if planID == "" {
		intervalUnit := paypal.IntervalUnitMonth
		intervalCount := 1
		if ctx.BillingInterval == "year" {
			intervalCount = 12
		}

		amountStr := strconv.FormatFloat(ctx.UnitAmount, 'f', 2, 64)

		billingPlan := paypal.SubscriptionPlan{
			ProductId:   productID,
			Name:        plan.Name + " (" + ctx.BillingInterval + "ly)",
			Description: plan.Description,
			BillingCycles: []paypal.BillingCycle{
				{
					Frequency: paypal.Frequency{
						IntervalUnit:  intervalUnit,
						IntervalCount: intervalCount,
					},
					TenureType:  paypal.TenureTypeRegular,
					Sequence:    1,
					TotalCycles: 0,
					PricingScheme: paypal.PricingScheme{
						FixedPrice: paypal.Money{
							Currency: "USD",
							Value:    amountStr,
						},
					},
				},
			},
			PaymentPreferences: &paypal.PaymentPreferences{
				AutoBillOutstanding:     true,
				SetupFeeFailureAction:   "CONTINUE",
				PaymentFailureThreshold: 3,
			},
		}

		billingResp, err := p.client.CreateSubscriptionPlan(context.Background(), billingPlan)
		if err != nil {
			return nil, fmt.Errorf("failed to create PayPal billing plan: %w", err)
		}
		planID = billingResp.ID

		if ctx.SavePlan != nil {
			if ctx.BillingInterval == "year" {
				plan.PayPalYearlyPlanID = planID
			} else {
				plan.PayPalMonthlyPlanID = planID
			}
			if err := ctx.SavePlan(plan); err != nil {
				return nil, fmt.Errorf("failed to save PayPal plan ID: %w", err)
			}
		}
	}

	// 3. Create subscription
	sub := paypal.SubscriptionBase{
		PlanID: planID,
		Subscriber: &paypal.Subscriber{
			EmailAddress: ctx.CustomerEmail,
			Name: paypal.CreateOrderPayerName{
				GivenName: ctx.BusinessName,
			},
		},
		ApplicationContext: &paypal.ApplicationContext{
			BrandName:    ctx.BusinessName,
			ReturnURL:    ctx.SuccessURL,
			CancelURL:    ctx.CancelURL,
			UserAction:   "SUBSCRIBE_NOW",
		},
		CustomID: fmt.Sprintf("%d:%s", ctx.BusinessID, ctx.PlanCode),
	}

	subResp, err := p.client.CreateSubscription(context.Background(), sub)
	if err != nil {
		return nil, fmt.Errorf("failed to create PayPal subscription: %w", err)
	}

	// Extract approval URL from links
	approvalURL := ""
	for _, link := range subResp.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
			break
		}
	}

	if approvalURL == "" {
		return nil, fmt.Errorf("no approval URL returned from PayPal")
	}

	return &CheckoutSession{
		ID:  subResp.ID,
		URL: approvalURL,
	}, nil
}

func (p *PayPalAdapter) CreateBillingPortalSession(customerID, returnURL string) (string, error) {
	return "https://www.paypal.com/business/manage", nil
}

func (p *PayPalAdapter) HandleWebhook(payload []byte, sigHeader string) (*WebhookEvent, error) {
	if p.webhookID == "" {
		return nil, fmt.Errorf("PayPal webhook ID not configured")
	}

	// Parse the webhook event
	var anyEvent paypal.AnyEvent
	if err := json.Unmarshal(payload, &anyEvent); err != nil {
		return nil, fmt.Errorf("failed to parse PayPal webhook: %w", err)
	}

	// Note: full signature verification requires the HTTP request
	// The handler calling this should verify via PayPal SDK's VerifyWebhookSignature
	// Here we just parse the event type and extract data

	event := &WebhookEvent{
		Type: anyEvent.EventType,
	}

	switch anyEvent.EventType {
	case "BILLING.SUBSCRIPTION.ACTIVATED",
		"BILLING.SUBSCRIPTION.UPDATED",
		"BILLING.SUBSCRIPTION.CANCELLED",
		"BILLING.SUBSCRIPTION.SUSPENDED":
		var resource struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			CustomID string `json:"custom_id"`
			Subscriber struct {
				EmailAddress string `json:"email_address"`
				PayerID      string `json:"payer_id"`
			} `json:"subscriber"`
		}
		if err := json.Unmarshal(anyEvent.Resource, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse subscription resource: %w", err)
		}
		event.SubscriptionID = resource.ID
		event.CustomerID = resource.Subscriber.PayerID
		if resource.CustomID != "" {
			fmt.Sscanf(resource.CustomID, "%d:%s", &event.BusinessID, &event.SubscriptionPlanCode)
		}
		switch anyEvent.EventType {
		case "BILLING.SUBSCRIPTION.ACTIVATED":
			event.SubscriptionStatus = "active"
		case "BILLING.SUBSCRIPTION.CANCELLED":
			event.SubscriptionStatus = "canceled"
		case "BILLING.SUBSCRIPTION.SUSPENDED":
			event.SubscriptionStatus = "past_due"
		case "BILLING.SUBSCRIPTION.UPDATED":
			event.SubscriptionStatus = "active"
		}

	case "PAYMENT.SALE.COMPLETED":
		var resource struct {
			BillingAgreementID string `json:"billing_agreement_id"`
			State             string `json:"state"`
		}
		if err := json.Unmarshal(anyEvent.Resource, &resource); err != nil {
			return nil, fmt.Errorf("failed to parse sale resource: %w", err)
		}
		event.SubscriptionID = resource.BillingAgreementID
		event.SubscriptionStatus = "active"
	}

	return event, nil
}

func (p *PayPalAdapter) GetOrCreateCustomer(business *models.Business) (string, error) {
	return "", nil
}
