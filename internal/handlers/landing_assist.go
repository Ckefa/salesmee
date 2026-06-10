package handlers

import (
	"log"
	"net/http"

	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

type landingChatRequest struct {
	Message string           `json:"message"`
	History []assist.Message `json:"history,omitempty"`
}

type landingSuggestionItem struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

func LandingAssistChat(c *gin.Context) {
	var req landingChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	messages := req.History
	if messages == nil {
		messages = []assist.Message{}
	}
	messages = append(messages, assist.Message{Role: "user", Content: req.Message})

	systemPrompt := assist.BuildLandingSystemPrompt()
	reply, err := assist.ChatCompletion(systemPrompt, messages)
	if err != nil {
		log.Printf("landing assist chat error: %v", err)
		c.JSON(http.StatusOK, gin.H{"reply": "I'm having trouble connecting right now. Please try again later or explore the site on your own!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func LandingGetAssistSuggestions(c *gin.Context) {
	suggestions := []landingSuggestionItem{
		{ID: "intro", Label: "What is SalesMee?", Prompt: "Tell me about SalesMee and what it does. Who is it for?"},
		{ID: "business-features", Label: "I own a business", Prompt: "Explain how SalesMee can help me manage my small business. Walk me through the key features for business owners."},
		{ID: "features", Label: "Key features tour", Prompt: "What are the standout features of SalesMee? Give me a tour of what makes it special."},
		{ID: "getting-started", Label: "How do I start?", Prompt: "I'm new to SalesMee. Walk me through the steps to get started from scratch."},
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}
