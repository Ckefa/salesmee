package business

import (
	"fmt"
	"net/http"
	"salesmee/internal/data"
	"salesmee/internal/handlers"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/notifier"
	"salesmee/internal/ws"
	"strconv"
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

	if conv, err := h.getOrCreateConversation(client.ID, request.BusinessID); err == nil {
		handlers.AutoCalculateProgress(conv.ID)
		if h.hub != nil {
			ws.BroadcastNewMessage(
				h.hub,
				strconv.Itoa(int(conv.ID)),
				strconv.Itoa(int(client.ID)),
				"client",
				strconv.Itoa(int(booking.ID+20000)),
				"",
				"",
				"",
				"booking",
				nil,
				booking.CreatedAt,
				strconv.Itoa(int(request.BusinessID)),
				strconv.Itoa(int(client.ID)),
			)
		}
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	baseWhere := "business_id = ? AND created_at BETWEEN ? AND ?"
	baseArgs := []interface{}{businessID, startTime, endTime}
	if locID := c.Query("location_id"); locID != "" {
		baseWhere += " AND location_id = ?"
		baseArgs = append(baseArgs, locID)
	}

	// Status counts for full date range
	var statusCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			confirmedCount = sc.Count
		case "completed":
			completedCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	// Total revenue for full date range
	var totalRevenue float64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	// Total count for pagination
	var totalCount int64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	// Paginated bookings
	var bookings []models.Booking
	h.db.Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").
		Where(baseWhere, baseArgs...).
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'completed' THEN 2 WHEN status = 'cancelled' THEN 3 ELSE 4 END, created_at DESC`).
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&bookings)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/bookings_content", gin.H{
			"Business":       currentBusiness,
			"Bookings":       bookings,
			"PendingCount":   pendingCount,
			"ConfirmedCount": confirmedCount,
			"CompletedCount": completedCount,
			"CancelledCount": cancelledCount,
			"TotalBookings":  totalCount,
			"TotalRevenue":   totalRevenue,
			"Page":           float64(page),
			"TotalPages":     float64(totalPages),
			"PageSize":       pageSize,
			"Range":          r,
			"Countries":      data.Countries,
			"Currencies":     data.Currencies,
			"Onboarding":     h.onboardingData(businessID),
			"Locations":      locations,
			"AuthType":       c.GetString("auth_type"),
			"Role":           c.GetString("role"),
			"ActivePage":     "bookings",
		})
		return
	}

	c.HTML(http.StatusOK, "bookings.html", gin.H{
		"Business":       currentBusiness,
		"Bookings":       bookings,
		"PendingCount":   pendingCount,
		"ConfirmedCount": confirmedCount,
		"CompletedCount": completedCount,
		"CancelledCount": cancelledCount,
		"TotalBookings":  totalCount,
		"TotalRevenue":   totalRevenue,
		"Page":           float64(page),
		"TotalPages":     float64(totalPages),
		"PageSize":       pageSize,
		"Range":          r,
		"Countries":      data.Countries,
		"Currencies":     data.Currencies,
		"Onboarding":     h.onboardingData(businessID),
		"Locations":      locations,
		"AuthType":       c.GetString("auth_type"),
		"Role":           c.GetString("role"),
		"ActivePage":     "bookings",
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	baseWhere := "business_id = ? AND created_at BETWEEN ? AND ?"
	baseArgs := []interface{}{businessID, startTime, endTime}
	if locID := c.Query("location_id"); locID != "" {
		baseWhere += " AND location_id = ?"
		baseArgs = append(baseArgs, locID)
	}

	var statusCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			confirmedCount = sc.Count
		case "completed":
			completedCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	var totalRevenue float64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	var totalCount int64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	var bookings []models.Booking
	h.db.Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").
		Where(baseWhere, baseArgs...).
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'completed' THEN 2 WHEN status = 'cancelled' THEN 3 ELSE 4 END, created_at DESC`).
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&bookings)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	c.HTML(http.StatusOK, "dashboard/bookings_content", gin.H{
		"Business":       currentBusiness,
		"Bookings":       bookings,
		"PendingCount":   pendingCount,
		"ConfirmedCount": confirmedCount,
		"CompletedCount": completedCount,
		"CancelledCount": cancelledCount,
		"TotalBookings":  totalCount,
		"TotalRevenue":   totalRevenue,
		"Page":           float64(page),
		"TotalPages":     float64(totalPages),
		"PageSize":       pageSize,
		"Range":          r,
		"ActivePage":     "bookings",
		"Locations":      locations,
		"AuthType":       c.GetString("auth_type"),
		"Role":           c.GetString("role"),
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

	baseWhere := "business_id = ? AND created_at BETWEEN ? AND ?"
	baseArgs := []interface{}{businessID, startTime, endTime}

	var statusCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			confirmedCount = sc.Count
		case "completed":
			completedCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	var totalBookings int64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalBookings)

	var totalRevenue float64
	h.db.Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	c.HTML(http.StatusOK, "bookings_stats_grid", gin.H{
		"Business":       currentBusiness,
		"PendingCount":   int(pendingCount),
		"ConfirmedCount": int(confirmedCount),
		"CompletedCount": int(completedCount),
		"CancelledCount": int(cancelledCount),
		"TotalBookings":  int(totalBookings),
		"TotalRevenue":   totalRevenue,
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
	chatLink := services.AppURL(fmt.Sprintf("/client?business_id=%d", biz.ID))

	if err := services.SendBookingStatusEmail(client.Email, client.Name, biz.Name, booking.BookingNumber, statusLabel, chatLink); err != nil {
		notifier.MarkNotificationSent(h.db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "failed")
		return
	}
	notifier.MarkNotificationSent(h.db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "sent")
	notifier.CreateInAppNotif(h.db, booking.BusinessID, &client.ID,
		fmt.Sprintf("Booking %s", statusLabel),
		fmt.Sprintf("Booking %s is now %s", booking.BookingNumber, statusLabel),
		"fa-calendar-check",
		"/business/bookings")
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
		"pending":          {"client_confirmed", "cancelled"},
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

	if h.hub != nil {
		ws.BroadcastBookingUpdate(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
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

	if h.hub != nil {
		ws.BroadcastBookingUpdate(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
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

	// Auto-advance conversation progress and notify any open chat panes.
	if conv, err := h.getOrCreateConversation(client.ID, businessID); err == nil {
		handlers.AutoCalculateProgress(conv.ID)
		if h.hub != nil {
			ws.BroadcastNewMessage(
				h.hub,
				strconv.Itoa(int(conv.ID)),
				strconv.Itoa(int(businessID)),
				"business",
				strconv.Itoa(int(booking.ID+20000)),
				"",
				"",
				"",
				"booking",
				nil,
				booking.CreatedAt,
				strconv.Itoa(int(businessID)),
				strconv.Itoa(int(client.ID)),
			)
		}
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
