package client

import (
	"log"
	"net/http"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

type chatRequest struct {
	Message  string           `json:"message"`
	History  []assist.Message `json:"history,omitempty"`
}

type suggestionItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Prompt  string `json:"prompt"`
}

func ClientAssistChat(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	var businessCount int64
	db.DB.Model(&models.Conversation{}).Where("client_id = ?", clientID).Count(&businessCount)

	systemPrompt := assist.BuildClientSystemPrompt(int(businessCount))

	messages := req.History
	if messages == nil {
		messages = []assist.Message{}
	}
	messages = append(messages, assist.Message{Role: "user", Content: req.Message})

	reply, err := assist.ChatCompletion(systemPrompt, messages)
	if err != nil {
		log.Printf("client assist chat error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI assistant temporarily unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func ClientGetAssistSuggestions(c *gin.Context) {
	suggestions := []suggestionItem{
		{ID: "draft-message", Label: "Draft a message", Prompt: "Draft a friendly message to a business I'm connected with on SalesMee."},
		{ID: "place-order", Label: "How to place an order", Prompt: "Explain how I can place an order with a business on SalesMee."},
		{ID: "help-platform", Label: "Help with SalesMee", Prompt: "I need help with using SalesMee as a customer. How do I navigate the Customer Portal?"},
		{ID: "booking-tip", Label: "Booking tips", Prompt: "Give me tips for booking services with businesses on SalesMee."},
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}
