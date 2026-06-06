package models

import "time"

type Review struct {
	ID         uint      `gorm:"primaryKey"`
	BusinessID uint      `gorm:"not null;index"`
	ClientID   uint      `gorm:"not null;index"`
	OrderID    *uint     `gorm:"index"`
	BookingID  *uint     `gorm:"index"`
	Rating     int       `gorm:"not null"`
	Title      string    `gorm:"type:varchar(200)"`
	Content    string    `gorm:"type:text"`
	Reply      string    `gorm:"type:text"`
	ReplyAt    *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Business Business `gorm:"foreignKey:BusinessID"`
	Client   Client   `gorm:"foreignKey:ClientID"`
}
