package models

import "time"

type CustomerInsight struct {
	ID                uint       `gorm:"primaryKey"`
	ConversationID    uint       `gorm:"not null;uniqueIndex"`
	CustomerID        uint       `gorm:"not null;index"`
	Tier              string     `gorm:"default:'bronze'"`
	TierScore         int        `gorm:"default:0"`
	ActivityScore     int        `gorm:"default:0"`
	BehaviorTrend     string     `gorm:"default:'inactive'"`
	EngagementScore   int        `gorm:"default:0"`
	TotalOrders       int        `gorm:"default:0"`
	CompletedOrders   int        `gorm:"default:0"`
	TotalBookings     int        `gorm:"default:0"`
	CompletedBookings int        `gorm:"default:0"`
	TotalMessages     int        `gorm:"default:0"`
	TotalSpent        float64    `gorm:"default:0"`
	LastActiveAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
