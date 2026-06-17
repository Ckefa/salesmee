package models

import "time"

type ProfileChangeRequest struct {
	Token         string    `gorm:"primaryKey" json:"token"`
	BusinessID    uint      `gorm:"not null;index" json:"business_id"`
	Name          string    `json:"name"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Password      string    `json:"-"`
	Logo          string    `json:"logo"`
	Country       string    `json:"country"`
	Currency      string    `json:"currency"`
	OTPCode       string    `json:"-"`
	OTPExpiresAt  time.Time `json:"otp_expires_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}
