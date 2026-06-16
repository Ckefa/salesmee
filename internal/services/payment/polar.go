package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	polargo "github.com/polarsource/polar-go"
	"github.com/polarsource/polar-go/models/components"
	"github.com/polarsource/polar-go/models/operations"

	"salesmee/internal/models"
)

type PolarAdapter struct {
	accessToken   string
	webhookSecret string
	environment   string
	sdk           *polargo.Polar
}

func NewPolarAdapter() *PolarAdapter {
	accessToken := os.Getenv("POLAR_ACCESS_TOKEN")
	webhookSecret := os.Getenv("POLAR_WEBHOOK_SECRET")
	environment := os.Getenv("POLAR_ENVIRONMENT")
	if environment == "" {
		environment = "sandbox"
	}

	var sdk *polargo.Polar
	if accessToken != "" {
		opts := []polargo.SDKOption{
			polargo.WithSecurity(accessToken),
		}
		if environment == "sandbox" {
			opts = append(opts, polargo.WithServer("sandbox"))
		}
		sdk = polargo.New(opts...)
	}

	return &PolarAdapter{
		accessToken:   accessToken,
		webhookSecret: webhookSecret,
		environment:   environment,
		sdk:           sdk,
	}
}

func (p *PolarAdapter) Name() string {
	return "polar"
}

func (p *PolarAdapter) CreateCheckoutSession(ctx *CheckoutContext) (*CheckoutSession, error) {
	if p.sdk == nil {
		return nil, fmt.Errorf("Polar SDK not initialized: POLAR_ACCESS_TOKEN not set")
	}

	metadata := map[string]components.CheckoutCreateMetadata{
		"business_id":      components.CreateCheckoutCreateMetadataStr(fmt.Sprintf("%d", ctx.BusinessID)),
		"plan_code":        components.CreateCheckoutCreateMetadataStr(ctx.PlanCode),
		"billing_interval": components.CreateCheckoutCreateMetadataStr(ctx.BillingInterval),
	}

	req := components.CheckoutCreate{
		Products:      []string{ctx.Plan.PolarPriceID},
		SuccessURL:    &ctx.SuccessURL,
		ReturnURL:     &ctx.CancelURL,
		Metadata:      metadata,
		CustomerEmail: &ctx.CustomerEmail,
	}

	res, err := p.sdk.Checkouts.Create(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("polar create checkout: %w", err)
	}
	if res.Checkout == nil {
		return nil, fmt.Errorf("polar: empty checkout response")
	}

	return &CheckoutSession{
		ID:  res.Checkout.ID,
		URL: res.Checkout.URL,
	}, nil
}

func (p *PolarAdapter) CreateBillingPortalSession(customerID, returnURL string) (string, error) {
	if p.sdk == nil {
		return "", fmt.Errorf("Polar SDK not initialized: POLAR_ACCESS_TOKEN not set")
	}

	req := operations.CreateCustomerSessionsCreateCustomerSessionCreateCustomerSessionCustomerExternalIDCreate(
		components.CustomerSessionCustomerExternalIDCreate{
			ExternalCustomerID: customerID,
			ReturnURL:          &returnURL,
		},
	)

	res, err := p.sdk.CustomerSessions.Create(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("polar create customer session: %w", err)
	}
	if res.CustomerSession == nil {
		return "", fmt.Errorf("polar: empty customer session response")
	}

	return res.CustomerSession.CustomerPortalURL, nil
}

func (p *PolarAdapter) CancelSubscription(subscriptionID string) error {
	if p.sdk == nil {
		return fmt.Errorf("Polar SDK not initialized: POLAR_ACCESS_TOKEN not set")
	}

	_, err := p.sdk.Subscriptions.Revoke(context.Background(), subscriptionID)
	if err != nil {
		return fmt.Errorf("polar revoke subscription: %w", err)
	}
	return nil
}

type polarWebhookPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type polarSubscriptionData struct {
	ID                 string                 `json:"id"`
	Status             string                 `json:"status"`
	CustomerID         string                 `json:"customer_id"`
	PriceID            string                 `json:"price_id"`
	CurrentPeriodStart string                 `json:"current_period_start"`
	CurrentPeriodEnd   string                 `json:"current_period_end"`
	CancelAtPeriodEnd  bool                   `json:"cancel_at_period_end"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	ProductID          string                 `json:"product_id"`
}

func (p *PolarAdapter) HandleWebhook(payload []byte, sigHeader string, extraHeaders ...map[string]string) (*WebhookEvent, error) {
	if p.webhookSecret == "" {
		return nil, fmt.Errorf("POLAR_WEBHOOK_SECRET not configured")
	}

	var wh polarWebhookPayload
	if err := json.Unmarshal(payload, &wh); err != nil {
		return nil, fmt.Errorf("polar webhook parse: %w", err)
	}

	sig := sigHeader
	webhookID := ""
	timestamp := ""

	if len(extraHeaders) > 0 {
		webhookID = extraHeaders[0]["webhook-id"]
		timestamp = extraHeaders[0]["webhook-timestamp"]
	} else {
		parts := strings.SplitN(sigHeader, "|", 3)
		if len(parts) == 3 {
			webhookID = parts[0]
			timestamp = parts[1]
			sig = parts[2]
		}
	}

	if webhookID != "" && timestamp != "" {
		if err := p.verifyWebhookSignature(payload, sig, webhookID, timestamp); err != nil {
			return nil, fmt.Errorf("polar webhook verification: %w", err)
		}
	}

	var subData polarSubscriptionData
	if err := json.Unmarshal(wh.Data, &subData); err != nil {
		return nil, fmt.Errorf("polar webhook data parse: %w", err)
	}

	var businessID uint
	planCode := ""
	billingInterval := "month"

	if subData.Metadata != nil {
		if v, ok := subData.Metadata["business_id"]; ok {
			if s, ok := v.(string); ok {
				fmt.Sscanf(s, "%d", &businessID)
			}
		}
		if v, ok := subData.Metadata["plan_code"]; ok {
			if s, ok := v.(string); ok {
				planCode = s
			}
		}
		if v, ok := subData.Metadata["billing_interval"]; ok {
			if s, ok := v.(string); ok {
				billingInterval = s
			}
		}
	}

	event := &WebhookEvent{
		Type:               wh.Type,
		CustomerID:         subData.CustomerID,
		SubscriptionID:     subData.ID,
		SubscriptionStatus: subData.Status,
		BusinessID:         businessID,
		SubscriptionPlanCode: planCode,
		BillingInterval:    billingInterval,
	}

	if subData.CurrentPeriodStart != "" {
		if t, err := time.Parse(time.RFC3339, subData.CurrentPeriodStart); err == nil {
			event.CurrentPeriodStart = t.Unix()
		}
	}
	if subData.CurrentPeriodEnd != "" {
		if t, err := time.Parse(time.RFC3339, subData.CurrentPeriodEnd); err == nil {
			event.CurrentPeriodEnd = t.Unix()
		}
	}

	return event, nil
}

func (p *PolarAdapter) verifyWebhookSignature(payload []byte, sigHeader, webhookID, timestamp string) error {
	signingContent := fmt.Sprintf("%s.%s.%s", webhookID, timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write([]byte(signingContent))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	parts := strings.SplitN(sigHeader, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid signature header format")
	}
	providedSig := strings.TrimSpace(parts[1])

	if !hmac.Equal([]byte(expectedSig), []byte(providedSig)) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

func (p *PolarAdapter) GetOrCreateCustomer(business *models.Business) (string, error) {
	if p.sdk == nil {
		return "", fmt.Errorf("Polar SDK not initialized: POLAR_ACCESS_TOKEN not set")
	}

	externalID := fmt.Sprintf("biz_%d", business.ID)
	customerReq := components.CreateCustomerCreateCustomerIndividualCreate(
		components.CustomerIndividualCreate{
			Email:      business.Email,
			Name:       &business.Name,
			ExternalID: &externalID,
		},
	)

	res, err := p.sdk.Customers.Create(context.Background(), customerReq)
	if err != nil {
		return "", fmt.Errorf("polar create customer: %w", err)
	}

	var customerID string
	if res.Customer != nil && res.Customer.CustomerIndividual != nil {
		customerID = res.Customer.CustomerIndividual.ID
	} else if res.Customer != nil && res.Customer.CustomerTeam != nil {
		customerID = res.Customer.CustomerTeam.ID
	} else {
		return "", fmt.Errorf("polar: empty customer response")
	}

	return customerID, nil
}
