package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

func SendBusinessVerification(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := db.DB.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	if business.EmailVerified {
		c.JSON(http.StatusOK, gin.H{"message": "Email already verified"})
		return
	}

	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	db.DB.Model(&business).Update("verification_token", token)

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	verifyLink := scheme + "://" + c.Request.Host + "/business/verify?token=" + token

	if err := services.SendVerificationEmail(business.Email, verifyLink); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

func VerifyBusinessEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusOK, "verify_email.html", gin.H{
			"Title": "Verify Email - SalesMee",
			"Error": "Missing verification token.",
		})
		return
	}

	var business models.Business
	if err := db.DB.Where("verification_token = ?", token).First(&business).Error; err != nil {
		c.HTML(http.StatusOK, "verify_email.html", gin.H{
			"Title": "Verify Email - SalesMee",
			"Error": "Invalid or expired verification link.",
		})
		return
	}

	db.DB.Model(&business).Updates(map[string]interface{}{
		"email_verified":    true,
		"verification_token": "",
	})

	c.HTML(http.StatusOK, "verify_email.html", gin.H{
		"Title":   "Verify Email - SalesMee",
		"Success": "Your email has been verified! You can now log in to your account.",
	})
}
