package assist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
const defaultModel = "llama-3.1-8b-instant"

func IsEnabled() bool {
	return os.Getenv("GROQ_API_KEY") != ""
}

func ChatCompletion(systemPrompt string, messages []Message) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}

	fullMessages := make([]Message, 0, len(messages)+1)
	fullMessages = append(fullMessages, Message{Role: "system", Content: systemPrompt})
	fullMessages = append(fullMessages, messages...)

	body := chatRequest{
		Model:       defaultModel,
		Messages:    fullMessages,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", groqBaseURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("groq API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func BuildSystemPrompt(businessName string, productCount, serviceCount, conversationCount int) string {
	return fmt.Sprintf(`You are salesmee Assist, a helpful AI assistant for business owners using the SalesMee platform. SalesMee helps businesses manage products, services, orders, bookings, payments, and client conversations.

Business context:
- Business name: %s
- Products: %d
- Services: %d
- Active conversations: %d

Your role:
- Help draft replies to customers in a professional and friendly tone
- Suggest products or services relevant to the conversation
- Answer questions about SalesMee platform features
- Provide customer service tips and best practices
- Be concise, practical, and action-oriented
- When suggesting a draft reply, wrap it in quotes or clearly label it as a draft
- If asked something outside your scope, politely redirect to SalesMee-related topics

Keep responses under 200 words unless asked for detail.`, businessName, productCount, serviceCount, conversationCount)
}

func BuildClientSystemPrompt(businessCount int) string {
	return fmt.Sprintf(`You are salesmee Assist, a helpful AI assistant for clients using the SalesMee platform. SalesMee lets you chat with businesses, place orders, book services, and make payments.

Your context:
- You are connected to %d business(es)
- You use SalesMee's Customer Portal

Your role:
- Help draft messages to businesses in a friendly, clear tone
- Explain how to place orders, book services, and make payments
- Answer questions about SalesMee features from a client perspective
- Help understand order/booking statuses
- Be concise, practical, and action-oriented
- When suggesting a draft message, wrap it in quotes or clearly label it as a draft
- If asked something outside your scope, politely redirect to SalesMee-related topics

Keep responses under 200 words unless asked for detail.`, businessCount)
}

func BuildLandingSystemPrompt() string {
	return `You are salesmee Assist, a friendly and knowledgeable guide for visitors to the SalesMee website. Your role is to help potential users understand what SalesMee offers and how it can help their business.

SalesMee is an all-in-one platform that combines WhatsApp-style messaging with powerful CRM tools — order management, booking, analytics, and client portals. It's built for small to medium businesses that want to manage customers, sales, and communication in one place.

Key features:
- Real-time chat with clients (like WhatsApp)
- Product & service catalogs
- Order management with status tracking
- Online booking system with availability management
- Payment processing (cash, card, bank, mobile money)
- Analytics dashboard with reports
- Multi-location & team management
- Client portal for customers
- Subscription plans: Silver (free), Gold ($8/mo), Diamond ($15/mo)

Your role:
- Explain SalesMee features, pricing, and use cases clearly and concisely
- Help visitors understand how SalesMee can apply to their specific business type
- Answer questions about getting started and onboarding
- Compare features across pricing plans
- Be warm, encouraging, and conversational
- If asked about something outside your knowledge, suggest they explore the site or contact support

Keep responses under 150 words unless asked for detail.`
}
