package client

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"oneflow/internal/db"
	"oneflow/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowDiscover(c *gin.Context) {
	clientID := c.GetUint("client_id")
	log.Printf("[ShowDiscover] clientID=%d", clientID)

	var client models.Client
	db.DB.First(&client, clientID)

	businesses := getDiscoverableBusinesses(clientID)

	// If HTMX request, render partial for in-chat-area loading
	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "client_discover_content.html", gin.H{
			"Businesses": businesses,
		})
		return
	}

	c.HTML(http.StatusOK, "client_discover.html", gin.H{
		"Title":      "Discover Businesses - OneFlow",
		"Businesses": businesses,
		"Email":      c.GetString("client_email"),
		"Client":     client,
	})
}

func getDiscoverableBusinesses(clientID uint) []models.Business {
	var existingIDs []uint
	db.DB.Model(&models.Conversation{}).Where("client_id = ?", clientID).Pluck("business_id", &existingIDs)

	var businesses []models.Business
	query := db.DB.Where("is_public = ?", true)
	if len(existingIDs) > 0 {
		query = query.Where("id NOT IN ?", existingIDs)
	}
	query.Order("name ASC").Limit(20).Find(&businesses)
	return businesses
}

func SearchBusinesses(c *gin.Context) {
	clientID := c.GetUint("client_id")
	q := strings.TrimSpace(c.Query("q"))
	log.Printf("[SearchBusinesses] clientID=%d, query=%q", clientID, q)

	// Get client's existing business IDs
	var existingIDs []uint
	db.DB.Model(&models.Conversation{}).Where("client_id = ?", clientID).Pluck("business_id", &existingIDs)

	var businesses []models.Business
	query := db.DB.Where("is_public = ?", true)
	if len(existingIDs) > 0 {
		query = query.Where("id NOT IN ?", existingIDs)
	}
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("name ILIKE ? OR business_type ILIKE ? OR slug ILIKE ?", like, like, like)
	}
	query.Order("name ASC").Limit(50).Find(&businesses)
	log.Printf("[SearchBusinesses] clientID=%d, foundBusinesses=%d", clientID, len(businesses))

	c.JSON(http.StatusOK, businesses)
}

func ConnectToBusiness(c *gin.Context) {
	clientID := c.GetUint("client_id")
	businessIDStr := c.Param("business_id")
	log.Printf("[ConnectToBusiness] clientID=%d, businessIDStr=%s", clientID, businessIDStr)

	businessID, err := strconv.ParseUint(businessIDStr, 10, 32)
	if err != nil {
		log.Printf("[ConnectToBusiness] ERROR: invalid business ID string=%s", businessIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	// Verify business exists
	var business models.Business
	if err := db.DB.First(&business, businessID).Error; err != nil {
		log.Printf("[ConnectToBusiness] ERROR: business not found for businessID=%d: %v", businessID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}
	log.Printf("[ConnectToBusiness] found business: ID=%d, Name=%s", business.ID, business.Name)

	// Check if conversation already exists
	var conversation models.Conversation
	err = db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error
	if err == nil {
		log.Printf("[ConnectToBusiness] existing conversation found: ID=%d, clientID=%d, businessID=%d", conversation.ID, conversation.ClientID, conversation.BusinessID)
		c.JSON(http.StatusOK, gin.H{"success": true, "conversation_id": conversation.ID, "already_connected": true})
		return
	}
	log.Printf("[ConnectToBusiness] no existing conversation found, creating new one")

	// Create new conversation
	conversation = models.Conversation{
		ClientID:   clientID,
		BusinessID: uint(businessID),
	}
	if err := db.DB.Create(&conversation).Error; err != nil {
		log.Printf("[ConnectToBusiness] ERROR: failed to create conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}
	log.Printf("[ConnectToBusiness] created conversation ID=%d for clientID=%d, businessID=%d", conversation.ID, clientID, businessID)

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"conversation_id":   conversation.ID,
		"already_connected": false,
	})
}

func CreateClient(c *gin.Context) {
	businessID := c.GetUint("business_id")

	name := c.PostForm("name")
	email := c.PostForm("email")
	phone := c.PostForm("phone")

	client := models.Client{
		BusinessID: &businessID,
		Name:       name,
		Email:      email,
		Phone:      phone,
		Status:     models.StatusNew,
	}

	if err := db.DB.Create(&client).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create client"})
		return
	}

	// Create conversation for the new client
	conversation := models.Conversation{
		ClientID:   client.ID,
		BusinessID: businessID,
	}
	db.DB.Create(&conversation)

	c.JSON(200, gin.H{
		"success": true,
		"message": "Customer created successfully",
		"client":  client,
	})
}

func deleteConversationWithDeps(clientID, businessID uint) error {
	var conv models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conv).Error; err != nil {
		return nil
	}
	var msgIDs []uint
	db.DB.Model(&models.Message{}).Where("conversation_id = ?", conv.ID).Pluck("id", &msgIDs)
	tx := db.DB.Begin()
	if len(msgIDs) > 0 {
		tx.Where("message_id IN ?", msgIDs).Delete(&models.Action{})
	}
	tx.Where("conversation_id = ?", conv.ID).Delete(&models.Message{})
	tx.Where("conversation_id = ?", conv.ID).Delete(&models.ConversationProgress{})
	tx.Delete(&conv)
	return tx.Commit().Error
}

func DeleteClient(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := deleteConversationWithDeps(uint(clientID), businessID); err != nil {
		c.JSON(500, gin.H{"error": "Failed to disconnect customer"})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "Customer disconnected successfully",
	})
}

func UpdateClientStatus(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid customer ID"})
		return
	}

	// Verify client has a conversation with this business
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.JSON(404, gin.H{"error": "Customer not found"})
		return
	}

	var client models.Client
	if err := db.DB.First(&client, clientID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Customer not found"})
		return
	}

	newStatus := c.PostForm("status")
	client.Status = models.ClientStatus(newStatus)

	if err := db.DB.Save(&client).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to update customer status"})
		return
	}

	c.JSON(200, gin.H{"client": client})
}

func GetClientConversationID(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid customer ID"})
		return
	}

	// Verify client has a conversation with this business
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.JSON(404, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(200, gin.H{"conversation_id": conversation.ID})
}

func DisconnectFromBusiness(c *gin.Context) {
	clientID := c.GetUint("client_id")
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid business ID"})
		return
	}

	if err := deleteConversationWithDeps(clientID, uint(businessID)); err != nil {
		c.JSON(500, gin.H{"error": "Failed to disconnect"})
		return
	}

	c.JSON(200, gin.H{"success": true})
}
