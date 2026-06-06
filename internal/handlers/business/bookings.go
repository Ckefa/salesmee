package business

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"salesmee/internal/data"
	"salesmee/internal/handlers"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/notifier"
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

	r := c.DefaultQuery("range", "this_month")
	startTime, endTime, _ := timeRangeBounds(r)

	var bookings []models.Booking
	bookingQuery := h.db.Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime)
	if locID := c.Query("location_id"); locID != "" {
		bookingQuery = bookingQuery.Where("location_id = ?", locID)
	}
	bookingQuery.Find(&bookings)

	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	var totalRevenue float64

	for _, booking := range bookings {
		switch booking.Status {
		case "pending":
			pendingCount++
		case "client_confirmed":
			confirmedCount++
		case "completed":
			completedCount++
			totalRevenue += booking.TotalAmount
		case "cancelled":
			cancelledCount++
		}
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

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
		"Countries":       data.Countries,
		"Currencies":      data.Currencies,
		"Onboarding":      h.onboardingData(businessID),
		"Locations":       locations,
	})
}

func (h *BusinessHandler) GetBookingsStats(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	startTime, endTime, _ := timeRangeBounds(r)

	var bookings []models.Booking
	h.db.Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).Find(&bookings)

	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	var totalRevenue float64

	for _, booking := range bookings {
		switch booking.Status {
		case "pending":
			pendingCount++
		case "client_confirmed":
			confirmedCount++
		case "completed":
			completedCount++
			totalRevenue += booking.TotalAmount
		case "cancelled":
			cancelledCount++
		}
	}

	c.HTML(http.StatusOK, "bookings_content", gin.H{
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

func (h *BusinessHandler) GetBookingsStatsGrid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.db.First(&currentBusiness, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	startTime, endTime, _ := timeRangeBounds(r)

	var bookings []models.Booking
	h.db.Where("business_id = ? AND created_at BETWEEN ? AND ?", businessID, startTime, endTime).Find(&bookings)

	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	var totalRevenue float64

	for _, booking := range bookings {
		switch booking.Status {
		case "pending":
			pendingCount++
		case "client_confirmed":
			confirmedCount++
		case "completed":
			completedCount++
			totalRevenue += booking.TotalAmount
		case "cancelled":
			cancelledCount++
		}
	}

	c.HTML(http.StatusOK, "bookings_stats_grid", gin.H{
		"Business":       currentBusiness,
		"PendingCount":    int(pendingCount),
		"ConfirmedCount":  int(confirmedCount),
		"CompletedCount":  int(completedCount),
		"CancelledCount":  int(cancelledCount),
		"TotalBookings":   len(bookings),
		"TotalRevenue":    totalRevenue,
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

func (h *BusinessHandler) sendBookingNotif(booking models.Booking, status string) {
	prefs, err := notifier.GetOrCreatePrefs(h.db, booking.BusinessID)
	if err != nil || !prefs.BookingStatusChange {
		return
	}
	var client models.Client
	if err := h.db.First(&client, booking.ClientID).Error; err != nil || client.Email == "" {
		return
	}
	var biz models.Business
	if err := h.db.First(&biz, booking.BusinessID).Error; err != nil {
		return
	}

	notifType := "booking_status"
	rid := booking.ID
	if notifier.HasBeenSent(h.db, booking.BusinessID, client.ID, notifType, &rid) {
		return
	}

	statusLabel := status
	if err := services.SendBookingStatusEmail(client.Email, client.Name, biz.Name, booking.BookingNumber, statusLabel); err != nil {
		notifier.MarkNotificationSent(h.db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "failed")
		return
	}
	notifier.MarkNotificationSent(h.db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "sent")
	notifier.CreateInAppNotif(h.db, booking.BusinessID, &client.ID,
		fmt.Sprintf("Booking %s", statusLabel),
		fmt.Sprintf("Booking %s is now %s", booking.BookingNumber, statusLabel),
		"fa-calendar-check",
		fmt.Sprintf("/business/bookings/%d", booking.ID))
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
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", id, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	newStatus := request.Status
	validTransitions := map[string][]string{
		"pending":          {"cancelled"},
		"client_confirmed": {"completed", "cancelled"},
		"completed":        {},
		"cancelled":        {},
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

	h.sendBookingNotif(booking, newStatus)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", booking.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

// MarkBookingAsPaid sets the booking's paid amount to the total (quick mark as fully paid)
func (h *BusinessHandler) MarkBookingAsPaid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingIDStr := c.Param("id")
	bookingID, _ := strconv.ParseUint(bookingIDStr, 10, 32)

	var booking models.Booking
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != "client_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking must be confirmed before marking as paid"})
		return
	}

	booking.PaidAmount = booking.TotalAmount
	booking.UpdatedAt = time.Now()
	h.db.Save(&booking)

	h.db.Create(&models.Payment{
		BookingID: &booking.ID,
		ClientID:  booking.ClientID,
		Amount:    booking.TotalAmount,
		Method:    "cash",
		Status:    "completed",
		Reference: "quick-paid",
		Notes:     "Marked as paid from dashboard",
	})

	h.sendBookingNotif(booking, "paid")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", booking.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Booking marked as paid",
	})
}

// GetBookingReceipt renders a print-friendly receipt for a completed booking
func (h *BusinessHandler) GetBookingReceipt(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	bookingID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.db.Preload("Client").Preload("BookingItems.Service").Preload("Payments").Preload("Business").Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != "completed" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Booking is not completed"})
		return
	}

	c.HTML(http.StatusOK, "receipt_booking.html", gin.H{
		"Booking":  booking,
		"Business": booking.Business,
	})
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
		ClientID      uint   `json:"client_id"`
		ServiceID     uint   `json:"service_id" binding:"required"`
		CustomerName  string `json:"customer_name"`
		CustomerEmail string `json:"customer_email"`
		CustomerPhone string `json:"customer_phone"`
		BookingDate   string `json:"booking_date" binding:"required"`
		Notes         string `json:"notes"`
		MarkCompleted bool   `json:"mark_completed"`
		LocationID    *uint  `json:"location_id"`
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

	// Get or create client
	var client models.Client
	if request.ClientID > 0 {
		if err := h.db.First(&client, request.ClientID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find client"})
			return
		}
	} else {
		client = models.Client{
			BusinessID: &businessID,
			Name:       request.CustomerName,
			Email:      request.CustomerEmail,
			Phone:      request.CustomerPhone,
		}
		if err := h.db.Create(&client).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client"})
			return
		}
	}

	// Parse booking date
	bookingDate, err := time.Parse(time.RFC3339, request.BookingDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Validate against business hours & availability
	if err := h.validateBookingSlot(businessID, bookingDate, service.Duration); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := "pending"
	if request.MarkCompleted {
		status = "completed"
	}

	// Create booking
	booking := models.Booking{
		BusinessID:    businessID,
		ClientID:      client.ID,
		BookingNumber: generateBookingNumber(),
		Status:        status,
		Sender:        "business",
		ScheduledDate: bookingDate,
		Duration:      service.Duration,
		TotalAmount:   service.MaxPrice,
		Notes:         request.Notes,
		LocationID:    request.LocationID,
	}

	if request.MarkCompleted {
		booking.PaidAmount = booking.TotalAmount
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

	// If mark_completed, create payment record
	if request.MarkCompleted {
		payment := models.Payment{
			BookingID: &booking.ID,
			Amount:    booking.TotalAmount,
			Method:    "cash",
			Status:    "completed",
			Reference: "Walk-in counter payment",
		}
		h.db.Create(&payment)
	}

	// Send notification for new booking
	h.sendBookingNotif(booking, "created")

	// Auto-advance conversation progress
	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", client.ID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
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

func (h *BusinessHandler) validateBookingSlot(businessID uint, scheduledAt time.Time, duration int) error {
	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		return fmt.Errorf("business not found")
	}

	if !business.IsAcceptingBookings {
		return fmt.Errorf("business is not accepting bookings at this time")
	}

	// Parse business hours
	var hoursJSON map[string][]map[string]string
	if business.BusinessHours != "" && business.BusinessHours != "{}" {
		if err := json.Unmarshal([]byte(business.BusinessHours), &hoursJSON); err != nil {
			return fmt.Errorf("invalid business hours configuration")
		}
	}

	dayName := scheduledAt.Weekday().String()
	dayKey := weekDayKey(dayName)

	slots, hasHours := hoursJSON[dayKey]
	if !hasHours || len(slots) == 0 {
		return fmt.Errorf("business is closed on %s", dayName)
	}

	bookingTime := scheduledAt.Format("15:04")
	var withinHours bool
	for _, slot := range slots {
		if bookingTime >= slot["open"] && bookingTime <= slot["close"] {
			withinHours = true
			break
		}
	}
	if !withinHours {
		return fmt.Errorf("booking time %s is outside business hours on %s", bookingTime, dayName)
	}

	// Check special hours / closures
	if business.SpecialHours != "" && business.SpecialHours != "[]" {
		var special []map[string]interface{}
		if err := json.Unmarshal([]byte(business.SpecialHours), &special); err == nil {
			dateStr := scheduledAt.Format("2006-01-02")
			for _, s := range special {
				if s["date"] == dateStr {
					if closed, ok := s["is_closed"].(bool); ok && closed {
						return fmt.Errorf("business is closed on this date")
					}
					if open, ok := s["open"].(string); ok && open != "" {
						if close, ok := s["close"].(string); ok && close != "" {
							if bookingTime < open || bookingTime > close {
								return fmt.Errorf("booking time is outside special hours on this date")
							}
						}
					}
				}
			}
		}
	}

	// Check max bookings per slot
	if business.MaxBookingsPerSlot > 0 {
		slotStart := scheduledAt.Truncate(time.Hour)
		slotEnd := slotStart.Add(time.Hour)
		var existingCount int64
		h.db.Model(&models.Booking{}).
			Where("business_id = ? AND status NOT IN ('cancelled') AND scheduled_date >= ? AND scheduled_date < ?",
				businessID, slotStart, slotEnd).
			Count(&existingCount)
		if existingCount >= int64(business.MaxBookingsPerSlot) {
			return fmt.Errorf("this time slot is fully booked")
		}
	}

	// Check buffer time
	if business.BufferTime > 0 {
		bufferStart := scheduledAt.Add(-time.Duration(business.BufferTime) * time.Minute)
		bufferEnd := scheduledAt.Add(time.Duration(duration+business.BufferTime) * time.Minute)
		var conflicting int64
		h.db.Model(&models.Booking{}).
			Where("business_id = ? AND status NOT IN ('cancelled','completed') AND ((scheduled_date >= ? AND scheduled_date < ?) OR (scheduled_date + (duration || ' minutes')::interval >= ? AND scheduled_date + (duration || ' minutes')::interval < ?))",
				businessID, bufferStart, bufferEnd, bufferStart, bufferEnd).
			Count(&conflicting)
		if conflicting > 0 {
			return fmt.Errorf("there is not enough buffer time between bookings")
		}
	}

	return nil
}

func weekDayKey(dayName string) string {
	switch dayName {
	case "Monday":
		return "monday"
	case "Tuesday":
		return "tuesday"
	case "Wednesday":
		return "wednesday"
	case "Thursday":
		return "thursday"
	case "Friday":
		return "friday"
	case "Saturday":
		return "saturday"
	case "Sunday":
		return "sunday"
	}
	return strings.ToLower(dayName)
}
