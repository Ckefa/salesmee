package testutil

import (
	"fmt"
	"salesmee/internal/models"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var migrateModels = []interface{}{
	&models.Business{},
	&models.Client{},
	&models.Conversation{},
	&models.Message{},
	&models.Action{},
	&models.ClientAuth{},
	&models.ConversationProgress{},
	&models.Product{},
	&models.Service{},
	&models.Order{},
	&models.OrderItem{},
	&models.Booking{},
	&models.BookingItem{},
	&models.Payment{},
	&models.PaymentMethod{},
	&models.InventoryLog{},
	&models.ProductImage{},
	&models.CustomerInsight{},
	&models.SubscriptionPlan{},
	&models.BusinessSubscription{},
	&models.PasswordResetToken{},
	&models.Admin{},
	&models.AuditLog{},
	&models.BusinessNotifPrefs{},
	&models.NotificationLog{},
	&models.InAppNotification{},
	&models.Review{},
	&models.Location{},
	&models.TeamMember{},
	&models.ProfileChangeRequest{},
}

func SetupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}
	for _, m := range migrateModels {
		if err := db.AutoMigrate(m); err != nil {
			panic("failed to migrate " + err.Error())
		}
	}
	return db
}

func CreateBusiness(db *gorm.DB, overrides map[string]interface{}) models.Business {
	b := models.Business{
		Email:            "test@example.com",
		Name:             "Test Business",
		Username:         "testbiz",
		Slug:             "test-biz-" + randomSuffix(),
		Country:          "US",
		Currency:         "USD",
		OnboardingStep:   6,
		IsAcceptingBookings: true,
		BusinessHours:    "{}",
		SpecialHours:     "[]",
	}
	if v, ok := overrides["Email"].(string); ok {
		b.Email = v
	}
	if v, ok := overrides["Name"].(string); ok {
		b.Name = v
	}
	if v, ok := overrides["Slug"].(string); ok {
		b.Slug = v
	}
	if v, ok := overrides["Password"].(*string); ok {
		b.Password = v
	}
	if v, ok := overrides["BusinessID"].(uint); ok {
		b.ID = v
	}
	db.Create(&b)
	return b
}

func CreateClient(db *gorm.DB, businessID uint, overrides map[string]interface{}) models.Client {
	c := models.Client{
		BusinessID: &businessID,
		Name:       "Test Client",
		Email:      "client@example.com",
		Status:     models.StatusActive,
	}
	if v, ok := overrides["Name"].(string); ok {
		c.Name = v
	}
	if v, ok := overrides["Email"].(string); ok {
		c.Email = v
	}
	db.Create(&c)

	conv := models.Conversation{
		ClientID:   c.ID,
		BusinessID: businessID,
	}
	db.Create(&conv)

	c.ConversationID = conv.ID
	db.Save(&c)
	return c
}

func CreateProduct(db *gorm.DB, businessID uint, overrides map[string]interface{}) models.Product {
	p := models.Product{
		BusinessID: businessID,
		Name:       "Test Product",
		Price:      29.99,
		Stock:      100,
		MinStock:   5,
		IsActive:   true,
	}
	if v, ok := overrides["Name"].(string); ok {
		p.Name = v
	}
	if v, ok := overrides["Price"].(float64); ok {
		p.Price = v
	}
	if v, ok := overrides["Stock"].(int); ok {
		p.Stock = v
	}
	db.Create(&p)
	return p
}

func CreateService(db *gorm.DB, businessID uint, overrides map[string]interface{}) models.Service {
	s := models.Service{
		BusinessID: businessID,
		Name:       "Test Service",
		MaxPrice:   49.99,
		MinPrice:   49.99,
		Duration:   60,
		IsActive:   true,
	}
	if v, ok := overrides["Name"].(string); ok {
		s.Name = v
	}
	if v, ok := overrides["MaxPrice"].(float64); ok {
		s.MaxPrice = v
		s.MinPrice = v
	}
	db.Create(&s)
	return s
}

func CreateOrder(db *gorm.DB, businessID, clientID uint, overrides map[string]interface{}) models.Order {
	o := models.Order{
		BusinessID:  businessID,
		ClientID:    clientID,
		OrderNumber: "ORD-" + randomSuffix(),
		Status:      models.OrderDraft,
		TotalAmount: 29.99,
		PaidAmount:  0,
		Quantity:    1,
	}
	if v, ok := overrides["Status"].(string); ok {
		o.Status = v
	}
	if v, ok := overrides["TotalAmount"].(float64); ok {
		o.TotalAmount = v
	}
	db.Create(&o)
	return o
}

func CreateBooking(db *gorm.DB, businessID, clientID uint, overrides map[string]interface{}) models.Booking {
	now := time.Now()
	b := models.Booking{
		BusinessID:    businessID,
		ClientID:      clientID,
		BookingNumber: "BKG-" + randomSuffix(),
		Status:        models.BookingPending,
		TotalAmount:   49.99,
		PaidAmount:    0,
		ScheduledDate: now.Add(24 * time.Hour),
		Duration:      60,
	}
	if v, ok := overrides["Status"].(string); ok {
		b.Status = v
	}
	if v, ok := overrides["TotalAmount"].(float64); ok {
		b.TotalAmount = v
	}
	db.Create(&b)
	return b
}

var suffixCounter int

func randomSuffix() string {
	suffixCounter++
	return fmt.Sprintf("%d%d", time.Now().UnixNano()%100000, suffixCounter)
}
