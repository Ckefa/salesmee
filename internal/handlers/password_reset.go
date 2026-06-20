package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

var (
	passwordResetCooldownsMu sync.Mutex
	passwordResetCooldowns   = make(map[string]time.Time)
)

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			passwordResetCooldownsMu.Lock()
			now := time.Now()
			for email, t := range passwordResetCooldowns {
				if now.Sub(t) > 5*time.Minute {
					delete(passwordResetCooldowns, email)
				}
			}
			passwordResetCooldownsMu.Unlock()
		}
	}()
}

func ShowForgotPassword(c *gin.Context) {
	c.HTML(http.StatusOK, "forgot_password.html", middleware.TemplateData(c, gin.H{
		"Title": "Forgot Password - SalesMee",
	}))
}

func SendForgotPassword(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		var req struct {
			Email string `json:"email"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			email = req.Email
		}
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	// Check per-email cooldown (60s)
	passwordResetCooldownsMu.Lock()
	lastSent, exists := passwordResetCooldowns[email]
	remaining := time.Duration(0)
	if exists {
		remaining = 60*time.Second - time.Since(lastSent)
	}
	if remaining > 0 {
		passwordResetCooldownsMu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":    fmt.Sprintf("Please wait %d seconds before requesting another reset", int(remaining.Seconds())+1),
			"cooldown": int(remaining.Seconds()) + 1,
		})
		return
	}
	passwordResetCooldowns[email] = time.Now()
	passwordResetCooldownsMu.Unlock()

	var business models.Business
	if err := dbc(c).Where("email = ?", email).First(&business).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "If that email is registered, you'll receive a password reset link shortly.",
			"cooldown": 60,
		})
		return
	}

	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	reset := models.PasswordResetToken{
		BusinessID: business.ID,
		Email:      email,
		Token:      token,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	if err := dbc(c).Create(&reset).Error; err != nil {
		passwordResetCooldownsMu.Lock()
		delete(passwordResetCooldowns, email)
		passwordResetCooldownsMu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong. Please try again."})
		return
	}

	resetLink := services.GetBaseURL(c) + "/business/reset-password?token=" + token

	if err := services.SendPasswordResetEmail(email, resetLink); err != nil {
		passwordResetCooldownsMu.Lock()
		delete(passwordResetCooldowns, email)
		passwordResetCooldownsMu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email. Please try again later."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "If that email is registered, you'll receive a password reset link shortly.",
		"cooldown": 60,
	})
}

func ShowResetPassword(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var reset models.PasswordResetToken
	if err := dbc(c).Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&reset).Error; err != nil {
		c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Invalid or expired reset link.",
		}))
		return
	}

	c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
		"Title": "Reset Password - SalesMee",
		"Token": token,
	}))
}

func SubmitResetPassword(c *gin.Context) {
	token := c.PostForm("token")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("confirm_password")

	if token == "" || password == "" || confirmPassword == "" {
		c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "All fields are required",
			"Token": token,
		}))
		return
	}

	if len(password) < 6 {
		c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Password must be at least 6 characters",
			"Token": token,
		}))
		return
	}

	if password != confirmPassword {
		c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Passwords do not match",
			"Token": token,
		}))
		return
	}

	var reset models.PasswordResetToken
	if err := dbc(c).Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&reset).Error; err != nil {
		c.HTML(http.StatusOK, "reset_password.html", middleware.TemplateData(c, gin.H{
			"Title": "Reset Password - SalesMee",
			"Error": "Invalid or expired reset link.",
		}))
		return
	}

	hashed := services.Hash(password)

	tx := dbc(c).Begin()
	tx.Model(&models.Business{}).Where("id = ?", reset.BusinessID).Update("password", hashed)
	tx.Model(&reset).Update("used", true)
	tx.Commit()

	c.Redirect(http.StatusFound, "/business/login")
}
