package business

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"salesmee/internal/db"
	"salesmee/internal/services/onboarding"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

func GetPublicProfile(c *gin.Context) {
	slug := c.Param("slug")

	var business models.Business
	if err := db.DB.Where("slug = ? AND is_public = ?", slug, true).Preload("Clients").First(&business).Error; err != nil {
		c.HTML(http.StatusNotFound, "public_profile.html", middleware.TemplateData(c, gin.H{
			"Title": "Business Not Found - SalesMee",
			"Error": "Business not found or not available",
		}))
		return
	}

	var products []models.Product
	db.DB.Where("business_id = ? AND is_active = ?", business.ID, true).Find(&products)

	var svcList []models.Service
	db.DB.Where("business_id = ? AND is_active = ?", business.ID, true).Find(&svcList)

	// Determine if business is accepting clients (has any recent activity)
	var totalClients int64
	db.DB.Model(&models.Client{}).Where("business_id = ?", business.ID).Count(&totalClients)

	// Load reviews with client name
	var reviews []struct {
		models.Review
		ClientName string
	}
	db.DB.Raw(`
		SELECT r.*, c.name as client_name
		FROM reviews r
		JOIN clients c ON c.id = r.client_id
		WHERE r.business_id = ?
		ORDER BY r.created_at DESC
		LIMIT 10
	`, business.ID).Scan(&reviews)

	var avgRating float64
	var reviewCount int64
	db.DB.Model(&models.Review{}).Where("business_id = ?", business.ID).Select("COALESCE(AVG(rating), 0)").Scan(&avgRating)
	db.DB.Model(&models.Review{}).Where("business_id = ?", business.ID).Count(&reviewCount)

	var hoursObj interface{}
	if business.BusinessHours != "" && business.BusinessHours != "{}" {
		json.Unmarshal([]byte(business.BusinessHours), &hoursObj)
	}
	if hoursObj == nil {
		hoursObj = map[string]interface{}{}
	}

	var specialObj interface{}
	if business.SpecialHours != "" && business.SpecialHours != "[]" {
		json.Unmarshal([]byte(business.SpecialHours), &specialObj)
	}
	if specialObj == nil {
		specialObj = []interface{}{}
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	profileURL := fmt.Sprintf("%s://%s/b/%s", scheme, c.Request.Host, business.Slug)
	encodedProfileURL := url.QueryEscape(profileURL)

	c.HTML(http.StatusOK, "public_profile.html", middleware.TemplateData(c, gin.H{
		"Title":                business.Name + " - SalesMee",
		"Business":             business,
		"Products":             products,
		"Services":             svcList,
		"TotalClients":         int(totalClients),
		"IsAcceptingClients":   totalClients >= 0,
		"IsAcceptingBookings":  business.IsAcceptingBookings,
		"BusinessHours":        hoursObj,
		"SpecialHours":         specialObj,
		"Reviews":              reviews,
		"AverageRating":        avgRating,
		"ReviewCount":          int(reviewCount),
		"ProfileURL":           profileURL,
		"EncodedProfileURL":    encodedProfileURL,
	}))
}

type ConnectRequest struct {
	Email string `form:"email" binding:"required"`
}

type ConnectVerifyRequest struct {
	Email string `form:"email" binding:"required"`
	OTP   string `form:"otp" binding:"required"`
}

func ShowConnect(c *gin.Context) {
	slug := c.Param("slug")

	var business models.Business
	if err := db.DB.Where("slug = ?", slug).First(&business).Error; err != nil {
		c.HTML(http.StatusNotFound, "public_profile.html", middleware.TemplateData(c, gin.H{
			"Title": "Business Not Found - SalesMee",
			"Error": "Business not found",
		}))
		return
	}

	token, _ := c.Cookie("client_token")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		claims, err := services.ValidateToken(token)
		if err == nil && claims.Subject == "client" {
			clientID := claims.UserID

			now := time.Now()
			db.DB.Model(&models.Client{}).Where("id = ?", clientID).Updates(map[string]interface{}{
				"is_online":    true,
				"last_seen_at": &now,
			})

			var conversation models.Conversation
			err = db.DB.Where("client_id = ? AND business_id = ?", clientID, business.ID).
				First(&conversation).Error
			if err != nil {
				conversation = models.Conversation{
					ClientID:   clientID,
					BusinessID: business.ID,
				}
				db.DB.Create(&conversation)
			}

			onboarding.DetectStep(db.DB, &business)

			c.Redirect(http.StatusFound, fmt.Sprintf("/client?business_id=%d", business.ID))
			return
		}
	}

	c.HTML(http.StatusOK, "client_connect.html", middleware.TemplateData(c, gin.H{
		"Title":    "Connect - " + business.Name,
		"Business": business,
	}))
}

func SendConnectOTP(c *gin.Context) {
	slug := c.Param("slug")

	var business models.Business
	if err := db.DB.Where("slug = ?", slug).First(&business).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	email := c.PostForm("email")
	if email == "" {
		c.HTML(http.StatusBadRequest, "client_connect.html", middleware.TemplateData(c, gin.H{
			"Title":    "Connect - SalesMee",
			"Business": business,
			"Error":    "Email is required",
		}))
		return
	}

	var client models.Client
	err := db.DB.Where("email = ? AND business_id = ?", email, business.ID).First(&client).Error
	if err != nil {
		bizID := business.ID
		client = models.Client{
			BusinessID: &bizID,
			Email:      email,
			Name:       email,
			Status:     models.StatusNew,
		}
		if err := db.DB.Create(&client).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "client_connect.html", middleware.TemplateData(c, gin.H{
				"Title":    "Connect - SalesMee",
				"Business": business,
				"Error":    "Failed to create client",
			}))
			return
		}
	}

	_, err = services.SendClientOTP(email)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client_connect.html", middleware.TemplateData(c, gin.H{
			"Title":    "Connect - SalesMee",
			"Business": business,
			"Error":    "Failed to send OTP",
		}))
		return
	}

	c.HTML(http.StatusOK, "client_connect_otp.html", middleware.TemplateData(c, gin.H{
		"Title":    "Verify OTP - SalesMee",
		"Business": business,
		"Email":    email,
	}))
}

func VerifyConnectOTP(c *gin.Context) {
	slug := c.Param("slug")

	var business models.Business
	if err := db.DB.Where("slug = ?", slug).First(&business).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	email := c.PostForm("email")
	otpCode := c.PostForm("otp")

	if email == "" || otpCode == "" {
		c.HTML(http.StatusBadRequest, "client_connect_otp.html", middleware.TemplateData(c, gin.H{
			"Title":    "Verify OTP - SalesMee",
			"Business": business,
			"Email":    email,
			"Error":    "Email and OTP are required",
		}))
		return
	}

	clientAuth, err := services.VerifyClientOTP(email, otpCode)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client_connect_otp.html", middleware.TemplateData(c, gin.H{
			"Title":    "Verify OTP - SalesMee",
			"Business": business,
			"Email":    email,
			"Error":    "Invalid or expired OTP",
		}))
		return
	}

	clientAuth.IsVerified = true
	clientAuth.OTPCode = ""
	db.DB.Save(&clientAuth)

	now := time.Now()
	db.DB.Model(&models.Client{}).Where("id = ?", clientAuth.ClientID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	})

	token, err := services.GenerateClientToken(clientAuth)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_connect_otp.html", middleware.TemplateData(c, gin.H{
			"Title": "Verify OTP - SalesMee",
			"Email": email,
			"Error": "Failed to generate token",
		}))
		return
	}

	c.SetCookie("client_token", token, 86400, "/", "", false, false)
	c.Redirect(http.StatusFound, fmt.Sprintf("/client?business_id=%d", business.ID))
}
