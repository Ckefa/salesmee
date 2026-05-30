package handlers

import (
	"oneflow/internal/db"
	"oneflow/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

func DevPage(c *gin.Context) {
	// Get user info from context
	businessID, exists := c.Get("business_id")
	var isLoggedIn bool
	if exists {
		isLoggedIn = businessID != nil
	}

	// Get clients for display
	var clients []models.Client
	if exists {
		db.DB.Where("business_id = ?", businessID).Find(&clients)
	}

	c.HTML(200, "test.html", gin.H{
		"Title":    "Dev Panel",
		"LoggedIn": isLoggedIn,
		"Clients":  clients,
	})
}

func Ping(c *gin.Context) {
	c.HTML(200, "ping.html", gin.H{
		"Status": "OK",
		"Time":   time.Now().Format("15:04:05"),
	})
}

func ListItems(c *gin.Context) {
	userID := c.GetUint("business_id")
	var clients []models.Client
	db.DB.Where("business_id = ?", userID).Find(&clients)
	c.HTML(200, "items.html", gin.H{
		"Items": clients,
		"Count": len(clients),
	})
}

func CreateItem(c *gin.Context) {
	clientID := c.GetUint("client_id")
	name := c.PostForm("name")
	client := models.Client{
		ID:     clientID,
		Name:   name,
		Status: models.StatusNew,
	}
	db.DB.Create(&client)
	ListItems(c)
}

func DeleteItem(c *gin.Context) {
	userID := c.GetUint("business_id")
	id := c.Param("id")
	db.DB.Where("id = ? AND business_id = ?", id, userID).Delete(&models.Client{})
	ListItems(c)
}
