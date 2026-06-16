package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"salesmee/internal/models"
)

type PaddleAdapter struct {
	apiKey        string
	clientToken   string
	webhookSecret string
	environment   string
	apiBase       string
	httpClient    *http.Client
}

func NewPaddleAdapter() *PaddleAdapter {
	apiKey := os.Getenv("PADDLE_API_KEY")
	clientToken := os.Getenv("PADDLE_CLIENT_TOKEN")
	webhookSecret := os.Getenv("PADDLE_WEBHOOK_SECRET")
	environment := os.Getenv("PADDLE_ENVIRONMENT")
	if environment == "" {
		environment = "sandbox"
	}

	apiBase := "https://api.paddle.com"
	if environment == "sandbox" {
		apiBase = "https://sandbox-api.paddle.com"
	}

	if apiKey == "" || clientToken == "" {
		log.Println("WARNING: Paddle keys not fully configured. Payment features will fail at runtime.")
	}

	return &PaddleAdapter{
		apiKey:        apiKey,
		clientToken:   clientToken,
		webhookSecret: webhookSecret,
		environment:   environment,
		apiBase:       apiBase,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *PaddleAdapter) Name() string {
	return "paddle"
}

func (p *PaddleAdapter) CreateCheckoutSession(ctx *CheckoutContext) (*CheckoutSession, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Paddle API key not configured")
	}

	priceID := ctx.Plan.PaddlePriceID
	if priceID == "" {
		return nil, fmt.Errorf("Paddle price ID not configured for plan %s", ctx.PlanCode)
	}

	txnID, err := p.createTransaction(ctx, priceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create Paddle transaction: %w", err)
	}

	return &CheckoutSession{
		ID:  txnID,
		URL: "",
	}, nil
}

type paddleTransactionRequest struct {
	Items        []paddleTransactionItem `json:"items"`
	Customer     *paddleCustomer         `json:"customer,omitempty"`
	CustomData   map[string]string       `json:"custom_data,omitempty"`
	CurrencyCode string                  `json:"currency_code,omitempty"`
	Status       string                  `json:"status,omitempty"`
}

type paddleTransactionItem struct {
	PriceID  string `json:"price_id"`
	Quantity int    `json:"quantity"`
}

type paddleCustomer struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type paddleTransactionResponse struct {
	Data paddleTransactionData `json:"data"`
}

type paddleTransactionData struct {
	ID string `json:"id"`
}

func (p *PaddleAdapter) createTransaction(ctx *CheckoutContext, priceID string) (string, error) {
	body := paddleTransactionRequest{
		Items: []paddleTransactionItem{
			{
				PriceID:  priceID,
				Quantity: 1,
			},
		},
		CustomData: map[string]string{
			"business_id":      fmt.Sprintf("%d", ctx.BusinessID),
			"plan_code":        ctx.PlanCode,
			"billing_interval": ctx.BillingInterval,
		},
		CurrencyCode: strings.ToUpper(ctx.Currency),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.apiBase+"/transactions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Printf("Paddle createTransaction error %d: %s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("Paddle API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var txnResp paddleTransactionResponse
	if err := json.Unmarshal(respBody, &txnResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if txnResp.Data.ID == "" {
		return "", fmt.Errorf("Paddle returned empty transaction ID")
	}

	log.Printf("Paddle transaction created: %s", txnResp.Data.ID)
	return txnResp.Data.ID, nil
}

func (p *PaddleAdapter) CreateBillingPortalSession(customerID, returnURL string) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Paddle API key not configured")
	}

	body := map[string]interface{}{
		"customer_id": customerID,
		"return_url":  returnURL,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.apiBase+"/customer-portal/sessions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Paddle API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			URLs struct {
				General struct {
					Overview string `json:"overview"`
				} `json:"general"`
			} `json:"urls"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse portal session response: %w", err)
	}

	portalURL := result.Data.URLs.General.Overview
	if portalURL == "" {
		return "", fmt.Errorf("Paddle returned empty portal URL")
	}

	return portalURL, nil
}

func (p *PaddleAdapter) CancelSubscription(subscriptionID string) error {
	if p.apiKey == "" {
		return fmt.Errorf("Paddle API key not configured")
	}

	body := map[string]interface{}{
		"reason": "User requested cancellation",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal cancel request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.apiBase+"/subscriptions/"+subscriptionID+"/cancel", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Paddle cancel API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read cancel response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Paddle cancelSubscription error %d: %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("Paddle API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

type paddleWebhookPayload struct {
	EventID        string            `json:"event_id"`
	EventType      string            `json:"event_type"`
	OccurredAt     string            `json:"occurred_at"`
	Data           json.RawMessage   `json:"data"`
}

type paddleSubscriptionData struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	CustomerID string `json:"customer_id"`
	CustomData map[string]string `json:"custom_data"`
	StartedAt  string `json:"started_at"`
	CurrentBillingPeriod *struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"current_billing_period"`
	TrialDates *struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"trial_dates"`
}

type paddleTransactionDataEvent struct {
	ID         string            `json:"id"`
	Status     string            `json:"status"`
	CustomerID string            `json:"customer_id"`
	CustomData map[string]string `json:"custom_data"`
	SubscriptionID *string       `json:"subscription_id"`
	BilledAt   string            `json:"billed_at"`
}

func (p *PaddleAdapter) HandleWebhook(payload []byte, sigHeader string, extraHeaders ...map[string]string) (*WebhookEvent, error) {
	if p.webhookSecret == "" {
		// In dev mode, allow processing without signature verification
		log.Println("WARNING: Paddle webhook secret not configured, skipping signature verification")
	}

	if p.webhookSecret != "" && sigHeader != "" {
		if err := p.verifyWebhookSignature(payload, sigHeader); err != nil {
			return nil, fmt.Errorf("webhook signature verification failed: %w", err)
		}
	}

	var event paddleWebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	webhookEvent := &WebhookEvent{
		Type: event.EventType,
	}

	switch event.EventType {
	case "subscription.created", "subscription.updated", "subscription.cancelled", "subscription.past_due":
		var sub paddleSubscriptionData
		if err := json.Unmarshal(event.Data, &sub); err != nil {
			return nil, fmt.Errorf("failed to parse subscription data: %w", err)
		}

		webhookEvent.SubscriptionID = sub.ID
		webhookEvent.CustomerID = sub.CustomerID

		switch event.EventType {
		case "subscription.created":
			webhookEvent.SubscriptionStatus = "active"
		case "subscription.updated":
			webhookEvent.SubscriptionStatus = sub.Status
		case "subscription.cancelled":
			webhookEvent.SubscriptionStatus = "canceled"
		case "subscription.past_due":
			webhookEvent.SubscriptionStatus = "past_due"
		}

		if sub.CustomData != nil {
			if bid, ok := sub.CustomData["business_id"]; ok {
				bidUint, err := strconv.ParseUint(bid, 10, 64)
				if err == nil {
					webhookEvent.BusinessID = uint(bidUint)
				}
			}
			webhookEvent.SubscriptionPlanCode = sub.CustomData["plan_code"]
			webhookEvent.BillingInterval = sub.CustomData["billing_interval"]
		}

		if sub.CurrentBillingPeriod != nil {
			if start, err := time.Parse(time.RFC3339, sub.CurrentBillingPeriod.StartsAt); err == nil {
				webhookEvent.CurrentPeriodStart = start.Unix()
			}
			if end, err := time.Parse(time.RFC3339, sub.CurrentBillingPeriod.EndsAt); err == nil {
				webhookEvent.CurrentPeriodEnd = end.Unix()
			}
		}

		if sub.TrialDates != nil {
			if end, err := time.Parse(time.RFC3339, sub.TrialDates.EndsAt); err == nil {
				webhookEvent.TrialEnd = end.Unix()
			}
		}

	case "transaction.completed", "transaction.payment_failed":
		var txn paddleTransactionDataEvent
		if err := json.Unmarshal(event.Data, &txn); err != nil {
			return nil, fmt.Errorf("failed to parse transaction data: %w", err)
		}

		webhookEvent.CustomerID = txn.CustomerID

		if txn.SubscriptionID != nil {
			webhookEvent.SubscriptionID = *txn.SubscriptionID
		}

		if txn.CustomData != nil {
			if bid, ok := txn.CustomData["business_id"]; ok {
				bidUint, err := strconv.ParseUint(bid, 10, 64)
				if err == nil {
					webhookEvent.BusinessID = uint(bidUint)
				}
			}
			webhookEvent.SubscriptionPlanCode = txn.CustomData["plan_code"]
			webhookEvent.BillingInterval = txn.CustomData["billing_interval"]
		}

		if event.EventType == "transaction.completed" {
			webhookEvent.SubscriptionStatus = "active"
		} else {
			webhookEvent.SubscriptionStatus = "past_due"
		}
	}

	return webhookEvent, nil
}

func (p *PaddleAdapter) verifyWebhookSignature(payload []byte, sigHeader string) error {
	var ts, h1 string
	for _, part := range strings.Split(sigHeader, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "ts":
			ts = kv[1]
		case "h1":
			h1 = kv[1]
		}
	}
	if ts == "" || h1 == "" {
		return fmt.Errorf("invalid Paddle-Signature header format: missing ts or h1")
	}

	signedContent := fmt.Sprintf("%s.%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write([]byte(signedContent))
	expectedSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(h1)) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

func (p *PaddleAdapter) GetOrCreateCustomer(business *models.Business) (string, error) {
	if business.Subscription != nil && business.Subscription.PaddleCustomerID != "" {
		return business.Subscription.PaddleCustomerID, nil
	}

	if p.apiKey == "" {
		return "", fmt.Errorf("Paddle API key not configured")
	}

	body := map[string]interface{}{
		"email": business.Email,
		"name":  business.Name,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal customer request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.apiBase+"/customers", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Paddle API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse customer response: %w", err)
	}

	if result.Data.ID == "" {
		return "", fmt.Errorf("Paddle returned empty customer ID")
	}

	return result.Data.ID, nil
}
