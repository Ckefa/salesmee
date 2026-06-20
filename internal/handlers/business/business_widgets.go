package business

import (
	"net/http"
	"strconv"
	"time"

	"salesmee/internal/models"
	"salesmee/internal/services/progress"

	"github.com/gin-gonic/gin"
)

// QuickBooking creates a booking for current client
func QuickBooking(c *gin.Context) {
	userID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid client ID")
		return
	}

	// Verify client belongs to user
	var client models.Client
	if err := dbc(c).Where("id = ? AND business_id = ?", clientID, userID).First(&client).Error; err != nil {
		c.String(http.StatusNotFound, "Client not found")
		return
	}

	// Get conversation
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ?", clientID).First(&conversation); err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Create booking action
	title := "Quick Booking: " + client.Name
	description := "Created from business dashboard"

	action := models.Action{
		MessageID:   0, // Will be updated after message creation
		Type:        "booking",
		Title:       title,
		Description: description,
		Priority:    "high",
		Status:      models.OrderPending,
		DueDate:     &time.Time{}, // Will be set by user
		CreatedAt:   time.Now(),
	}

	if err := dbc(c).Create(&action).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create booking")
		return
	}

	// Create initial message for booking
	message := models.Message{
		ConversationID: conversation.ID,
		Content:        "📅 **Booking Created**\n\nClient: " + client.Name + "\nTitle: " + title + "\n\nClick to set date and details.",
		Sender:         "business",
	}

	if err := dbc(c).Create(&message).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create booking message")
		return
	}

	// Update action with message ID
	action.MessageID = message.ID
	dbc(c).Save(&action)

	progress.AutoCalculateProgress(conversation.ID)

	c.HTML(http.StatusOK, "confirmations/booking_confirmation.html", gin.H{
		"Client":  client,
		"Action":  action,
		"Message": message,
	})
}

// QuickOrder creates an order for current client
func QuickOrder(c *gin.Context) {
	userID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid client ID")
		return
	}

	// Verify client belongs to user
	var client models.Client
	if err := dbc(c).Where("id = ? AND business_id = ?", clientID, userID).First(&client).Error; err != nil {
		c.String(http.StatusNotFound, "Client not found")
		return
	}

	// Get conversation
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ?", clientID).First(&conversation); err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Create order action
	title := "Quick Order: " + client.Name
	description := "Created from business dashboard"

	action := models.Action{
		MessageID:   0,
		Type:        "order",
		Title:       title,
		Description: description,
		Priority:    "medium",
		Status:      models.OrderPending,
		CreatedAt:   time.Now(),
	}

	if err := dbc(c).Create(&action).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create order")
		return
	}

	// Create initial message for order
	message := models.Message{
		ConversationID: conversation.ID,
		Content:        "🛒 **Order Created**\n\nClient: " + client.Name + "\nTitle: " + title + "\n\nClick to set order details.",
		Sender:         "business",
	}

	if err := dbc(c).Create(&message).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create order message")
		return
	}

	// Update action with message ID
	action.MessageID = message.ID
	dbc(c).Save(&action)

	progress.AutoCalculateProgress(conversation.ID)

	c.HTML(http.StatusOK, "confirmations/order_confirmation.html", gin.H{
		"Client":  client,
		"Action":  action,
		"Message": message,
	})
}

// RequestPayment creates a payment request for current client
func RequestPayment(c *gin.Context) {
	userID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid client ID")
		return
	}

	// Verify client belongs to user
	var client models.Client
	if err := dbc(c).Where("id = ? AND business_id = ?", clientID, userID).First(&client).Error; err != nil {
		c.String(http.StatusNotFound, "Client not found")
		return
	}

	// Get conversation
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ?", clientID).First(&conversation); err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Create payment request action
	title := "Payment Request: " + client.Name
	description := "Created from business dashboard"

	action := models.Action{
		MessageID:   0,
		Type:        "payment",
		Title:       title,
		Description: description,
		Priority:    "high",
		Status:      models.OrderPending,
		CreatedAt:   time.Now(),
	}

	if err := dbc(c).Create(&action).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create payment request")
		return
	}

	// Create initial message for payment request
	message := models.Message{
		ConversationID: conversation.ID,
		Content:        "💳 **Payment Requested**\n\nClient: " + client.Name + "\nTitle: " + title + "\n\nClick to set payment details.",
		Sender:         "business",
	}

	if err := dbc(c).Create(&message).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create payment message")
		return
	}

	// Update action with message ID
	action.MessageID = message.ID
	dbc(c).Save(&action)

	progress.AutoCalculateProgress(conversation.ID)

	c.HTML(http.StatusOK, "confirmations/payment_confirmation.html", gin.H{
		"Client":  client,
		"Action":  action,
		"Message": message,
	})
}

// SetGoal creates a goal for current client
func SetGoal(c *gin.Context) {
	userID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid client ID")
		return
	}

	// Verify client belongs to user
	var client models.Client
	if err := dbc(c).Where("id = ? AND business_id = ?", clientID, userID).First(&client).Error; err != nil {
		c.String(http.StatusNotFound, "Client not found")
		return
	}

	// Get conversation
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ?", clientID).First(&conversation); err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Parse goal data
	goalText := c.PostForm("goal")
	targetDateStr := c.PostForm("target_date")

	// Create goal action
	title := "Goal Set: " + client.Name
	description := "Goal: " + goalText

	action := models.Action{
		MessageID:   0,
		Type:        "goal",
		Title:       title,
		Description: description,
		Priority:    "medium",
		Status:      models.OrderPending,
		CreatedAt:   time.Now(),
	}

	if targetDateStr != "" {
		if targetDate, err := time.Parse("2006-01-02", targetDateStr); err == nil {
			action.DueDate = &targetDate
		}
	}

	if err := dbc(c).Create(&action).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create goal")
		return
	}

	// Create initial message for goal
	message := models.Message{
		ConversationID: conversation.ID,
		Content:        "🎯 **Goal Set**\n\nClient: " + client.Name + "\nGoal: " + goalText + "\n\nClick to track progress.",
		Sender:         "business",
	}

	if err := dbc(c).Create(&message).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create goal message")
		return
	}

	// Update action with message ID
	action.MessageID = message.ID
	dbc(c).Save(&action)

	progress.AutoCalculateProgress(conversation.ID)

	c.HTML(http.StatusOK, "confirmations/goal_confirmation.html", gin.H{
		"Client":  client,
		"Action":  action,
		"Message": message,
	})
}
