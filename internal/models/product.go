package models

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	BusinessID  uint      `gorm:"not null;index" json:"business_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       float64   `gorm:"not null" json:"price"`
	SKU         string    `gorm:"unique" json:"sku"`
	Stock       int       `gorm:"default:0" json:"stock"`
	MinStock    int       `gorm:"default:0" json:"min_stock"`
	ImageURL    string    `json:"image_url"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Business   Business       `gorm:"foreignKey:BusinessID" json:"business,omitempty"`
	OrderItems []OrderItem    `gorm:"foreignKey:ProductID" json:"order_items,omitempty"`
	Images     []ProductImage `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Inventory  []InventoryLog `gorm:"foreignKey:ProductID" json:"inventory_logs,omitempty"`
}

type ProductImage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	ImageURL  string    `json:"image_url"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

type InventoryLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"not null;index" json:"product_id"`
	Type      string    `gorm:"not null" json:"type"` // "in", "out", "adjustment"
	Quantity  int       `gorm:"not null" json:"quantity"`
	Reason    string    `gorm:"type:text" json:"reason"`
	CreatedAt time.Time `json:"created_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}
