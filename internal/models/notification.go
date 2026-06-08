package models

import "time"

type BusinessNotifPrefs struct {
	BusinessID          uint  `gorm:"primaryKey"`
	BookingReminder1h   bool  `gorm:"default:true"`
	BookingReminder24h  bool  `gorm:"default:true"`
	OrderStatusChange   bool  `gorm:"default:true"`
	BookingStatusChange bool  `gorm:"default:true"`
	PaymentDueReminder  bool  `gorm:"default:true"`
	AbandonedCart       bool  `gorm:"default:true"`
	ReEngagement        bool  `gorm:"default:true"`
	AbandonedCartHours  int   `gorm:"default:24"`
	InactiveDays        int   `gorm:"default:30"`
	SoundEnabled        bool  `gorm:"default:true"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type NotificationLog struct {
	ID            uint      `gorm:"primaryKey"`
	BusinessID    uint      `gorm:"index"`
	ClientID      uint
	Type          string    `gorm:"type:varchar(50);index"`
	ReferenceType string    `gorm:"type:varchar(20)"`
	ReferenceID   *uint
	Email         string
	Status        string    `gorm:"type:varchar(20);default:sent"`
	SentAt        time.Time `gorm:"autoCreateTime"`
	CreatedAt     time.Time
}

type InAppNotification struct {
	ID         uint      `gorm:"primaryKey"`
	BusinessID uint      `gorm:"index"`
	ClientID   *uint
	Title      string
	Body       string
	Icon       string `gorm:"type:varchar(50)"`
	Link       string
	IsRead     bool   `gorm:"default:false"`
	CreatedAt  time.Time
}
