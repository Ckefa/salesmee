package business

import (
	"fmt"
	"net/http"
	"salesmee/internal/data"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/assist"
	"salesmee/internal/services/notifier"
	"salesmee/internal/services/progress"
	"salesmee/internal/services/subscription"
	"salesmee/internal/ws"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ClientCreateBooking allows customers to create bookings
func (h *BookingHandler) ClientCreateBooking(c *gin.Context) {
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
	if err := h.dbc(c).First(&service, request.ServiceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Get client by primary key
	var client models.Client
	if err := h.dbc(c).First(&client, clientID).Error; err != nil {
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
		Status: models.BookingPending,
		Sender:        "client",
		ScheduledDate: bookingDate,
		Duration:      service.Duration,
		TotalAmount:   service.MaxPrice,
		Notes:         request.Notes,
	}

	if err := h.dbc(c).Create(&booking).Error; err != nil {
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

	if err := h.dbc(c).Create(&bookingItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking item"})
		return
	}

	if conv, err := getOrCreateConversation(h.db, client.ID, request.BusinessID); err == nil {
		progress.AutoCalculateProgress(conv.ID)
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
			var bizUnread int64
			h.dbc(c).Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conv.ID, false).
				Count(&bizUnread)
			ws.BroadcastUnreadCount(h.hub, strconv.Itoa(int(conv.ID)), int32(bizUnread), strconv.Itoa(int(request.BusinessID)), "biz")
			bizCardHTML := renderBizBookingCard(h.db, booking)
			clientCardHTML := renderClientBookingCard(h.db, booking)
			ws.BroadcastBookingUpdateFull(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(request.BusinessID)), strconv.Itoa(int(client.ID)))

			var biz models.Business
			h.dbc(c).First(&biz, request.BusinessID)
			bizCardSidebar := RenderBizSidebarCard(client, conv.ID, "", booking.CreatedAt, int(bizUnread))
			clientCardSidebar := RenderClientSidebarCard(biz, conv.ID, "", booking.CreatedAt, 0)
			ws.BroadcastConversationUpdate(h.hub, strconv.Itoa(int(conv.ID)), bizCardSidebar, clientCardSidebar, strconv.Itoa(int(request.BusinessID)), strconv.Itoa(int(client.ID)))
		}
	}

	broadcastBizPendingCounts(h.db, h.hub, request.BusinessID)

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking, "service_name": service.Name})
}

func (h *BookingHandler) GetBookings(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
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
	h.dbc(c).Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case models.BookingPending:
			pendingCount = sc.Count
		case models.BookingClientConfirmed:
			confirmedCount = sc.Count
		case models.BookingCompleted:
			completedCount = sc.Count
		case models.BookingCancelled:
			cancelledCount = sc.Count
		}
	}

	// Total revenue for full date range
	var totalRevenue float64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	// Total count for pagination
	var totalCount int64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	// Paginated bookings
	var bookings []models.Booking
	h.dbc(c).Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").
		Where(baseWhere, baseArgs...).
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'completed' THEN 2 WHEN status = 'cancelled' THEN 3 ELSE 4 END, created_at DESC`).
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&bookings)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

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
			"Onboarding":     onboardingData(h.db, businessID),
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
		"AssistEnabled":  assist.IsEnabled(),
		"Onboarding":     onboardingData(h.db, businessID),
		"Locations":      locations,
		"AuthType":       c.GetString("auth_type"),
		"Role":           c.GetString("role"),
		"ActivePage":     "bookings",
	})
}

func (h *BookingHandler) GetBookingsStats(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
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
	h.dbc(c).Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case models.BookingPending:
			pendingCount = sc.Count
		case models.BookingClientConfirmed:
			confirmedCount = sc.Count
		case models.BookingCompleted:
			completedCount = sc.Count
		case models.BookingCancelled:
			cancelledCount = sc.Count
		}
	}

	var totalRevenue float64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	var totalCount int64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	var bookings []models.Booking
	h.dbc(c).Preload("Client").Preload("BookingItems").Preload("BookingItems.Service").
		Where(baseWhere, baseArgs...).
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'completed' THEN 2 WHEN status = 'cancelled' THEN 3 ELSE 4 END, created_at DESC`).
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&bookings)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

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

func (h *BookingHandler) GetBookingsStatsGrid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
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
	h.dbc(c).Model(&models.Booking{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var pendingCount, confirmedCount, completedCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case models.BookingPending:
			pendingCount = sc.Count
		case models.BookingClientConfirmed:
			confirmedCount = sc.Count
		case models.BookingCompleted:
			completedCount = sc.Count
		case models.BookingCancelled:
			cancelledCount = sc.Count
		}
	}

	var totalBookings int64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Count(&totalBookings)

	var totalRevenue float64
	h.dbc(c).Model(&models.Booking{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

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

func (h *BookingHandler) GetBooking(c *gin.Context) {
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
	if err := h.dbc(c).Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

func sendBookingNotif(db *gorm.DB, hub *ws.Hub, booking models.Booking, status string) {
	prefs, err := notifier.GetOrCreatePrefs(db, booking.BusinessID)
	if err != nil || !prefs.BookingStatusChange {
		return
	}
	var client models.Client
	if err := db.First(&client, booking.ClientID).Error; err != nil || client.Email == "" {
		return
	}
	var biz models.Business
	if err := db.First(&biz, booking.BusinessID).Error; err != nil {
		return
	}

	notifType := "booking_status"
	rid := booking.ID
	if notifier.HasBeenSent(db, booking.BusinessID, client.ID, notifType, &rid) {
		return
	}

	statusLabel := status
	chatLink := services.AppURL(fmt.Sprintf("/client?business_id=%d", biz.ID))

	if err := services.SendBookingStatusEmail(client.Email, client.Name, biz.Name, booking.BookingNumber, statusLabel, chatLink); err != nil {
		notifier.MarkNotificationSent(db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "failed")
		return
	}
	notifier.MarkNotificationSent(db, booking.BusinessID, client.ID, notifType, "booking", &rid, client.Email, "sent")
	notifier.CreateInAppNotif(db, booking.BusinessID, &client.ID,
		fmt.Sprintf("Booking %s", statusLabel),
		fmt.Sprintf("Booking %s is now %s", booking.BookingNumber, statusLabel),
		"calendar-days",
		"/business/bookings")
	if hub != nil {
		broadcastBizPendingCounts(db, hub, booking.BusinessID)
	}
}

func (h *BookingHandler) UpdateBookingStatus(c *gin.Context) {
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
	if err := h.dbc(c).Preload("Client").Where("id = ? AND business_id = ?", id, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	newStatus := request.Status
	validTransitions := map[string][]string{
		"pending":          {models.BookingClientConfirmed, models.BookingCancelled},
		models.BookingClientConfirmed: {models.BookingCompleted, models.BookingCancelled},
		models.BookingCompleted:        {},
		models.BookingCancelled:        {},
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
	if err := h.dbc(c).Save(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking status"})
		return
	}

	sendBookingNotif(h.db, h.hub, booking, newStatus)

	var conv models.Conversation
	if err := h.dbc(c).Where("client_id = ? AND business_id = ?", booking.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizBookingCard(h.db, booking)
		clientCardHTML := renderClientBookingCard(h.db, booking)
		ws.BroadcastBookingUpdateFull(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
	}

	broadcastBizPendingCounts(h.db, h.hub, businessID)

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

// MarkBookingAsPaid sets the booking's paid amount to the total (quick mark as fully paid)
func (h *BookingHandler) MarkBookingAsPaid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingIDStr := c.Param("id")
	bookingID, _ := strconv.ParseUint(bookingIDStr, 10, 32)

	var booking models.Booking
	if err := h.dbc(c).Preload("Client").Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != models.BookingClientConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking must be confirmed before marking as paid"})
		return
	}

	booking.PaidAmount = booking.TotalAmount
	booking.UpdatedAt = time.Now()
	h.dbc(c).Save(&booking)

	h.dbc(c).Create(&models.Payment{
		BookingID: &booking.ID,
		ClientID:  booking.ClientID,
		Amount:    booking.TotalAmount,
		Method:    "cash",
		Status:    models.BookingCompleted,
		Reference: "quick-paid",
		Notes:     "Marked as paid from dashboard",
	})

	sendBookingNotif(h.db, h.hub, booking, "paid")

	var conv models.Conversation
	if err := h.dbc(c).Where("client_id = ? AND business_id = ?", booking.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizBookingCard(h.db, booking)
		clientCardHTML := renderClientBookingCard(h.db, booking)
		ws.BroadcastBookingUpdateFull(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Booking marked as paid",
	})
}

// GetBookingReceipt renders a print-friendly receipt for a completed booking
func (h *BookingHandler) GetBookingReceipt(c *gin.Context) {
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
	if err := h.dbc(c).Preload("Client").Preload("BookingItems.Service").Preload("Payments").Preload("Business").Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != models.BookingCompleted {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Booking is not completed"})
		return
	}

	c.HTML(http.StatusOK, "receipt_booking.html", gin.H{
		"Booking":  booking,
		"Business": booking.Business,
	})
}

func (h *BookingHandler) UpdateBooking(c *gin.Context) {
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
	if err := h.dbc(c).Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
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
		if err := h.dbc(c).First(&service, request.ServiceID).Error; err == nil {
			if len(booking.BookingItems) > 0 {
				booking.BookingItems[0].ServiceID = request.ServiceID
				booking.BookingItems[0].UnitPrice = service.MaxPrice
				booking.BookingItems[0].TotalPrice = service.MaxPrice
				h.dbc(c).Save(&booking.BookingItems[0])
			}
			booking.TotalAmount = service.MaxPrice
		}
	}

	if err := h.dbc(c).Save(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

// CreateBooking for business
func (h *BookingHandler) CreateBooking(c *gin.Context) {
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
	if err := h.dbc(c).Where("id = ? AND business_id = ?", request.ServiceID, businessID).First(&service).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Get or create client
	var client models.Client
	if request.ClientID > 0 {
		if err := h.dbc(c).First(&client, request.ClientID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find client"})
			return
		}
	} else {
		check := subscription.CheckResourceLimit(businessID, "client")
		if !check.Allowed && !check.GraceAllowed {
			notifier.NotifyLimitReached(h.db, businessID, "client", "customers", check.Current, check.Max)
			c.JSON(http.StatusConflict, gin.H{
				"error":            "Client limit reached. You need to upgrade your plan to add more customers.",
				"limit_reached":    true,
				"requires_upgrade": true,
				"upgrade_url":      "/business/subscription#plans",
				"grace_allowed":    false,
			})
			return
		}
		if !check.Allowed && check.GraceAllowed {
			subscription.UseGrace(businessID, "client")
		}
		client = models.Client{
			BusinessID: &businessID,
			Name:       request.CustomerName,
			Email:      request.CustomerEmail,
			Phone:      request.CustomerPhone,
		}
		if err := h.dbc(c).Create(&client).Error; err != nil {
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

	status := models.BookingPending
	if request.MarkCompleted {
		status = models.BookingCompleted
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

	if err := h.dbc(c).Create(&booking).Error; err != nil {
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

	if err := h.dbc(c).Create(&bookingItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking item"})
		return
	}

	// If mark_completed, create payment record
	if request.MarkCompleted {
		payment := models.Payment{
			BookingID: &booking.ID,
			Amount:    booking.TotalAmount,
			Method:    "cash",
			Status:    models.BookingCompleted,
			Reference: "Walk-in counter payment",
		}
		h.dbc(c).Create(&payment)
	}

	// Send notification for new booking
	sendBookingNotif(h.db, h.hub, booking, "created")

	// Auto-advance conversation progress and notify any open chat panes.
	if conv, err := getOrCreateConversation(h.db, client.ID, businessID); err == nil {
		progress.AutoCalculateProgress(conv.ID)
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
			var clientUnread int64
			h.dbc(c).Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conv.ID, false).
				Count(&clientUnread)
			ws.BroadcastUnreadCount(h.hub, strconv.Itoa(int(conv.ID)), int32(clientUnread), strconv.Itoa(int(client.ID)), "client")
			bizCardHTML := renderBizBookingCard(h.db, booking)
			clientCardHTML := renderClientBookingCard(h.db, booking)
			ws.BroadcastBookingUpdateFull(h.hub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(client.ID)))

			var biz models.Business
			h.dbc(c).First(&biz, businessID)
			var bizUnread int64
			h.dbc(c).Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conv.ID, false).
				Count(&bizUnread)
			bizCardSidebar := RenderBizSidebarCard(client, conv.ID, "", booking.CreatedAt, int(bizUnread))
			clientCardSidebar := RenderClientSidebarCard(biz, conv.ID, "", booking.CreatedAt, int(clientUnread))
			ws.BroadcastConversationUpdate(h.hub, strconv.Itoa(int(conv.ID)), bizCardSidebar, clientCardSidebar, strconv.Itoa(int(businessID)), strconv.Itoa(int(client.ID)))
		}
	}

	broadcastBizPendingCounts(h.db, h.hub, businessID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"booking": booking,
		"message": fmt.Sprintf("Booking %s created successfully", booking.BookingNumber),
	})
}
func generateBookingNumber() string {
	return fmt.Sprintf("BOOK-%d", time.Now().Unix())
}
