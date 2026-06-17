package business

import (
	"crypto/rand"
	"encoding/hex"
	"salesmee/internal/db"
	"salesmee/internal/models"
	"time"
)

type PendingProfileChange struct {
	BusinessID    uint
	Token         string
	Name          string
	Username      string
	Email         string
	Password      string
	Logo          string
	Country       string
	Currency      string
	OTPCode       string
	OTPExpiresAt  time.Time
	ExpiresAt     time.Time
}

func saveProfileChange(data *PendingProfileChange) string {
	token := data.Token
	if token == "" {
		b := make([]byte, 16)
		rand.Read(b)
		token = hex.EncodeToString(b)
	}

	rec := models.ProfileChangeRequest{
		Token:        token,
		BusinessID:   data.BusinessID,
		Name:         data.Name,
		Username:     data.Username,
		Email:        data.Email,
		Password:     data.Password,
		Logo:         data.Logo,
		Country:      data.Country,
		Currency:     data.Currency,
		OTPCode:      data.OTPCode,
		OTPExpiresAt: data.OTPExpiresAt,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		CreatedAt:    time.Now(),
	}

	db.DB.Where("token = ?", token).Delete(&models.ProfileChangeRequest{})
	db.DB.Create(&rec)
	return token
}

func getProfileChange(token string) (*PendingProfileChange, bool) {
	var rec models.ProfileChangeRequest
	if err := db.DB.Where("token = ? AND expires_at > ?", token, time.Now()).First(&rec).Error; err != nil {
		return nil, false
	}

	return &PendingProfileChange{
		BusinessID:    rec.BusinessID,
		Token:         rec.Token,
		Name:          rec.Name,
		Username:      rec.Username,
		Email:         rec.Email,
		Password:      rec.Password,
		Logo:          rec.Logo,
		Country:       rec.Country,
		Currency:      rec.Currency,
		OTPCode:       rec.OTPCode,
		OTPExpiresAt:  rec.OTPExpiresAt,
		ExpiresAt:     rec.ExpiresAt,
	}, true
}

func deleteProfileChange(token string) {
	db.DB.Where("token = ?", token).Delete(&models.ProfileChangeRequest{})
}

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			db.DB.Where("expires_at <= ?", time.Now()).Delete(&models.ProfileChangeRequest{})
		}
	}()
}
