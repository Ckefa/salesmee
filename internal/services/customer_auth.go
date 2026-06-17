package services

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"salesmee/internal/config"
	"salesmee/internal/db"
	"salesmee/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateOTP() string {
	// Generate 6-digit OTP
	otp := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		otp += n.String()
	}
	return otp
}

func SendClientOTP(email string) (string, error) {
	// Find customer by email
	var client models.Client
	if err := db.DB.Where("email = ?", email).First(&client).Error; err != nil {
		return "", fmt.Errorf("customer not found")
	}

	// Generate OTP
	otpCode := GenerateOTP()

	// Create or update customer auth record
	var customerAuth models.ClientAuth
	err := db.DB.Where("client_id = ?", client.ID).First(&customerAuth).Error
	if err != nil {
		customerAuth = models.ClientAuth{
			ClientID:     client.ID,
			Email:        email,
			OTPCode:      otpCode,
			OTPExpiresAt: time.Now().Add(10 * time.Minute),
		}
		db.DB.Create(&customerAuth)
	} else {
		customerAuth.OTPCode = otpCode
		customerAuth.OTPExpiresAt = time.Now().Add(10 * time.Minute)
		customerAuth.IsVerified = false
		db.DB.Save(&customerAuth)
	}

	// Log OTP for dev fallback (only in dev mode)
	if config.IsDev() {
		log.Printf("[DEV] OTP for %s: %s", email, otpCode)
	}

	// Send OTP via email only if RESEND=true
	if config.C.ResendEnabled {
		if err := SendOTPEmail(email, otpCode); err != nil {
			log.Printf("Warning: failed to send OTP email to %s: %v", email, err)
		}
	}

	return otpCode, nil
}

func VerifyClientOTP(email, otpCode string) (*models.ClientAuth, error) {
	var customerAuth models.ClientAuth
	err := db.DB.Joins("JOIN clients ON client_auths.client_id = clients.id").
		Where("client_auths.email = ? AND client_auths.otp_code = ? AND client_auths.otp_expires_at > ?",
			email, otpCode, time.Now()).
		First(&customerAuth).Error

	if err != nil {
		return nil, fmt.Errorf("invalid or expired OTP")
	}

	// Mark as verified
	customerAuth.IsVerified = true
	customerAuth.OTPCode = "" // Clear OTP after verification
	db.DB.Save(&customerAuth)

	return &customerAuth, nil
}

func GenerateClientToken(clientAuth *models.ClientAuth) (string, error) {
	claims := &Claims{
		UserID: clientAuth.ClientID,
		Email:  clientAuth.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "client",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.C.JWTSecret))
}
