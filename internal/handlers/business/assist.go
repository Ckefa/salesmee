package business

import (
	"log"
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

type chatRequest struct {
	Message  string           `json:"message"`
	History  []assist.Message `json:"history,omitempty"`
	Page     string           `json:"page,omitempty"`
}

type suggestionItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Prompt  string `json:"prompt"`
}

func (h *AssistHandler) AssistChat(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
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

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	var productCount, serviceCount int64
	h.db.Model(&models.Product{}).Where("business_id = ?", businessID).Count(&productCount)
	h.db.Model(&models.Service{}).Where("business_id = ?", businessID).Count(&serviceCount)

	var conversationCount int64
	h.db.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&conversationCount)

	systemPrompt := assist.BuildSystemPrompt(
		business.Name,
		int(productCount),
		int(serviceCount),
		int(conversationCount),
	)

	if req.Page != "" {
		ctx := assist.BuildDataContext(h.db, businessID, req.Page, req.Message)
		if ctx != "" {
			systemPrompt += "\n\n" + ctx
		}
	}

	messages := req.History
	if messages == nil {
		messages = []assist.Message{}
	}
	messages = append(messages, assist.Message{Role: "user", Content: req.Message})

	reply, err := assist.ChatCompletion(systemPrompt, messages)
	if err != nil {
		log.Printf("assist chat error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI assistant temporarily unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func (h *AssistHandler) GetAssistSuggestions(c *gin.Context) {
	suggestions := []suggestionItem{
		{ID: "draft-reply", Label: "Draft a reply", Prompt: "Draft a professional reply to a customer conversation. Keep it friendly and helpful."},
		{ID: "suggest-product", Label: "Suggest a product", Prompt: "Suggest a product recommendation for a customer. Briefly explain why it's a good choice."},
		{ID: "help-platform", Label: "Help with SalesMee", Prompt: "I need help with using SalesMee. What can you tell me about managing orders, bookings, and customers?"},
		{ID: "customer-tip", Label: "Customer service tip", Prompt: "Give me a practical customer service tip for running my small business."},
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}
