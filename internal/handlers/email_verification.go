package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

var (
	resendCooldownsMu sync.Mutex
	resendCooldowns   = make(map[uint]time.Time)
)

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			resendCooldownsMu.Lock()
			now := time.Now()
			for id, t := range resendCooldowns {
				if now.Sub(t) > 5*time.Minute {
					delete(resendCooldowns, id)
				}
			}
			resendCooldownsMu.Unlock()
		}
	}()
}

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

	resendCooldownsMu.Lock()
	lastSent, exists := resendCooldowns[businessID]
	remaining := time.Duration(0)
	if exists {
		remaining = 60*time.Second - time.Since(lastSent)
	}
	if remaining > 0 {
		resendCooldownsMu.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":     fmt.Sprintf("Please wait %d seconds before resending", int(remaining.Seconds())+1),
			"cooldown":  int(remaining.Seconds()) + 1,
		})
		return
	}
	resendCooldowns[businessID] = time.Now()
	resendCooldownsMu.Unlock()

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

	c.JSON(http.StatusOK, gin.H{
		"message":  "Verification email sent",
		"cooldown": 60,
	})
}

func CheckVerificationStatus(c *gin.Context) {
	businessID := c.GetUint("business_id")
	var verified bool
	if err := db.DB.Model(&models.Business{}).Select("email_verified").Where("id = ?", businessID).Scan(&verified).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": verified})
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
