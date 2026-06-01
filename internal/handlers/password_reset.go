package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

func ShowForgotPassword(c *gin.Context) {
	c.HTML(http.StatusOK, "forgot_password.html", gin.H{
		"Title": "Forgot Password - SalesMee",
	})
}

func SendForgotPassword(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		c.HTML(http.StatusOK, "forgot_password.html", gin.H{
			"Title": "Forgot Password - SalesMee",
			"Error": "Email is required",
		})
		return
	}

	var business models.Business
	if err := db.DB.Where("email = ?", email).First(&business).Error; err != nil {
		// Don't reveal if email exists - always show success
		c.HTML(http.StatusOK, "forgot_password.html", gin.H{
			"Title":   "Forgot Password - SalesMee",
			"Success": "If that email is registered, you'll receive a password reset link shortly.",
		})
		return
	}

	// Generate reset token
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	reset := models.PasswordResetToken{
		BusinessID: business.ID,
		Email:      email,
		Token:      token,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	if err := db.DB.Create(&reset).Error; err != nil {
		c.HTML(http.StatusOK, "forgot_password.html", gin.H{
			"Title": "Forgot Password - SalesMee",
			"Error": "Something went wrong. Please try again.",
		})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	resetLink := scheme + "://" + c.Request.Host + "/business/reset-password?token=" + token

	if err := services.SendPasswordResetEmail(email, resetLink); err != nil {
		c.HTML(http.StatusOK, "forgot_password.html", gin.H{
			"Title": "Forgot Password - SalesMee",
			"Error": "Failed to send email. Please try again later.",
		})
		return
	}

	c.HTML(http.StatusOK, "forgot_password.html", gin.H{
		"Title":   "Forgot Password - SalesMee",
		"Success": "If that email is registered, you'll receive a password reset link shortly.",
	})
}

func ShowResetPassword(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var reset models.PasswordResetToken
	if err := db.DB.Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&reset).Error; err != nil {
		c.HTML(http.StatusOK, "reset_password.html", gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Invalid or expired reset link.",
		})
		return
	}

	c.HTML(http.StatusOK, "reset_password.html", gin.H{
		"Title": "Reset Password - SalesMee",
		"Token": token,
	})
}

func SubmitResetPassword(c *gin.Context) {
	token := c.PostForm("token")
	password := c.PostForm("password")

	if token == "" || password == "" {
		c.HTML(http.StatusOK, "reset_password.html", gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "All fields are required",
			"Token": token,
		})
		return
	}

	if len(password) < 6 {
		c.HTML(http.StatusOK, "reset_password.html", gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Password must be at least 6 characters",
			"Token": token,
		})
		return
	}

	var reset models.PasswordResetToken
	if err := db.DB.Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&reset).Error; err != nil {
		c.HTML(http.StatusOK, "reset_password.html", gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Invalid or expired reset link.",
		})
		return
	}

	hashed := services.Hash(password)

	tx := db.DB.Begin()
	tx.Model(&models.Business{}).Where("id = ?", reset.BusinessID).Update("password", hashed)
	tx.Model(&reset).Update("used", true)
	tx.Commit()

	c.Redirect(http.StatusFound, "/business/login")
}
