package business

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"oneflow/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

// ClientCreateBooking allows customers to create bookings
func (h *BusinessHandler) ClientCreateBooking(c *gin.Context) {
	// Get client ID from authenticated context (set by ClientMiddleware)
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated as client"})
		return
	}

	var request struct {
		ServiceID     uint   `json:"service_id" binding:"required"`
		ScheduledDate string `json:"scheduled_date" binding:"required"`
		Notes         string `json:"notes"`
		BusinessID    uint   `json:"business_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Get service details
	var service models.Service
	if err := h.db.First(&service, request.ServiceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Get client by primary key
	var client models.Client
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find client"})
		return
	}

	// Parse booking date
	bookingDate, err := time.Parse(time.RFC3339, request.ScheduledDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Create booking
	booking := models.Booking{
		BusinessID:    request.BusinessID,
		ClientID:      client.ID,
		BookingNumber: generateBookingNumber(),
		Status:        "pending",
		Sender:        "client",
		ScheduledDate: bookingDate,
		Duration:      service.Duration,
		TotalAmount:   service.MaxPrice,
		Notes:         request.Notes,
	}

	if err := h.db.Create(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking!"})
		return
	}

	// Create booking item
	bookingItem := models.BookingItem{
		BookingID:  booking.ID,
		ServiceID:  request.ServiceID,
		Quantity:   1,
		UnitPrice:  service.MaxPrice,
		TotalPrice: service.MaxPrice,
	}

	if err := h.db.Create(&bookingItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking, "service_name": service.Name})
}

// Helper functions
func parseBookingDateTime(bookingDateTime string) (time.Time, error) {
	parts := strings.Split(bookingDateTime, "T")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid datetime format, expected dateTtime")
	}

	date, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date: %v", err)
	}

	timeOnly, err := time.Parse("15:04", parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time: %v", err)
	}

	result := time.Date(date.Year(), date.Month(), date.Day(),
		timeOnly.Hour(), timeOnly.Minute(), 0, 0, time.UTC)

	return result, nil
}

func (h *BusinessHandler) GetBookings(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	var bookings []models.Booking
	h.db.Where("business_id = ?", businessID).Find(&bookings)

	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	var totalRevenue float64

	for _, booking := range bookings {
		switch booking.Status {
		case "pending":
			pendingCount++
		case "confirmed":
			confirmedCount++
		case "completed":
			completedCount++
			totalRevenue += booking.TotalAmount
		case "cancelled":
			cancelledCount++
		}
	}

	c.HTML(http.StatusOK, "bookings.html", gin.H{
		"Business":        currentBusiness,
		"Bookings":        bookings,
		"PendingCount":    pendingCount,
		"ConfirmedCount":  confirmedCount,
		"CompletedCount":  completedCount,
		"CancelledCount":  cancelledCount,
		"TotalBookings":   len(bookings),
		"TotalRevenue":    totalRevenue,
		"ActivePage":      "bookings",
	})
}

func (h *BusinessHandler) GetBooking(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

func (h *BusinessHandler) UpdateBookingStatus(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	id := c.Param("id")
	var request struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	newStatus := request.Status
	validTransitions := map[string][]string{
		"pending":   {"confirmed", "cancelled"},
		"confirmed": {"completed", "cancelled"},
		"completed": {},
		"cancelled": {},
	}

	allowed, ok := validTransitions[booking.Status]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid current booking status"})
		return
	}

	transitionAllowed := false
	for _, s := range allowed {
		if s == newStatus {
			transitionAllowed = true
			break
		}
	}

	if !transitionAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot transition booking from '%s' to '%s'", booking.Status, newStatus),
		})
		return
	}

	booking.Status = newStatus
	if err := h.db.Save(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

func (h *BusinessHandler) UpdateBooking(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var request struct {
		ServiceID   uint   `json:"service_id"`
		BookingDate string `json:"booking_date"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	booking.Notes = request.Notes

	if request.BookingDate != "" {
		scheduledDate, err := time.Parse(time.RFC3339, request.BookingDate)
		if err == nil {
			booking.ScheduledDate = scheduledDate
		}
	}

	if request.ServiceID > 0 {
		var service models.Service
		if err := h.db.First(&service, request.ServiceID).Error; err == nil {
			if len(booking.BookingItems) > 0 {
				booking.BookingItems[0].ServiceID = request.ServiceID
				booking.BookingItems[0].UnitPrice = service.MaxPrice
				booking.BookingItems[0].TotalPrice = service.MaxPrice
				h.db.Save(&booking.BookingItems[0])
			}
			booking.TotalAmount = service.MaxPrice
		}
	}

	if err := h.db.Save(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

// CreateBooking for business
func (h *BusinessHandler) CreateBooking(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var request struct {
		ClientID    uint   `json:"client_id" binding:"required"`
		ServiceID   uint   `json:"service_id" binding:"required"`
		clientName  string `json:"client_name" binding:"required"`
		clientEmail string `json:"client_email"`
		clientPhone string `json:"client_phone"`
		BookingDate string `json:"booking_date" binding:"required"`
		Notes       string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		fmt.Printf("Booking binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get service details
	var service models.Service
	if err := h.db.Where("id = ? AND business_id = ?", request.ServiceID, businessID).First(&service).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Create or get client
	var client models.Client
	if err := h.db.Where("id = ? AND business_id = ?", request.ClientID, businessID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find client"})
		return
	}

	// Parse booking date
	bookingDate, err := parseBookingDateTime(request.BookingDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create booking
	booking := models.Booking{
		BusinessID:    businessID,
		ClientID:      client.ID,
		BookingNumber: generateBookingNumber(),
		Status:        "pending",
		Sender:        "business",
		ScheduledDate: bookingDate,
		Duration:      service.Duration,
		TotalAmount:   service.MaxPrice,
		Notes:         request.Notes,
	}

	if err := h.db.Create(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking"})
		return
	}

	// Create booking item
	bookingItem := models.BookingItem{
		BookingID:  booking.ID,
		ServiceID:  service.ID,
		UnitPrice:  service.MaxPrice,
		TotalPrice: service.MaxPrice,
	}

	if err := h.db.Create(&bookingItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"booking": booking,
		"message": fmt.Sprintf("Booking %s created successfully", booking.BookingNumber),
	})
}
func generateBookingNumber() string {
	return fmt.Sprintf("BOOK-%d", time.Now().Unix())
}
