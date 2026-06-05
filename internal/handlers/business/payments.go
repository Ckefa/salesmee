package business

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"salesmee/internal/data"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
)

// UpdatePaymentInstructions saves the business's payment instructions
func (h *BusinessHandler) UpdatePaymentInstructions(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	instructions := c.PostForm("payment_instructions")
	if err := h.db.Model(&business).Update("payment_instructions", instructions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment instructions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ClientSubmitOrderPayment lets a client claim they've paid for an order
func (h *BusinessHandler) ClientSubmitOrderPayment(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated as client"})
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Where("id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "client_confirmed" && order.Status != "confirmed" && order.Status != "fulfilled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be confirmed before payment"})
		return
	}

	var request struct {
		Amount    float64 `json:"amount" binding:"required"`
		Method    string  `json:"method" binding:"required"` // cash, card, bank_transfer, mobile_money
		Reference string  `json:"reference"`
		Notes     string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remaining := order.TotalAmount - order.PaidAmount
	if request.Amount <= 0 || request.Amount > remaining {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Amount must be between 0 and %.2f (remaining balance)", remaining),
		})
		return
	}

	payment := models.Payment{
		OrderID:   &order.ID,
		ClientID:  clientID,
		Amount:    request.Amount,
		Method:    request.Method,
		Status:    "pending",
		Reference: request.Reference,
		Notes:     request.Notes,
	}

	if err := h.db.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
		"message": "Payment recorded. Awaiting business confirmation.",
	})
}

// ClientSubmitBookingPayment lets a client claim they've paid for a booking
func (h *BusinessHandler) ClientSubmitBookingPayment(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated as client"})
		return
	}

	bookingIDStr := c.Param("id")
	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.db.Where("id = ? AND client_id = ?", bookingID, clientID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != "client_confirmed" && booking.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking must be confirmed before payment"})
		return
	}

	var request struct {
		Amount    float64 `json:"amount" binding:"required"`
		Method    string  `json:"method" binding:"required"`
		Reference string  `json:"reference"`
		Notes     string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remaining := booking.TotalAmount - booking.PaidAmount
	if request.Amount <= 0 || request.Amount > remaining {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Amount must be between 0 and %.2f (remaining balance)", remaining),
		})
		return
	}

	payment := models.Payment{
		BookingID: &booking.ID,
		ClientID:  clientID,
		Amount:    request.Amount,
		Method:    request.Method,
		Status:    "pending",
		Reference: request.Reference,
		Notes:     request.Notes,
	}

	if err := h.db.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
		"message": "Payment recorded. Awaiting business confirmation.",
	})
}

// ConfirmOrderPayment confirms a payment for an order (business side)
func (h *BusinessHandler) ConfirmOrderPayment(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	orderIDStr := c.Param("id")
	paymentIDStr := c.Param("payment_id")

	orderID, _ := strconv.ParseUint(orderIDStr, 10, 32)
	paymentID, _ := strconv.ParseUint(paymentIDStr, 10, 32)

	var order models.Order
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var payment models.Payment
	if err := h.db.Where("id = ? AND order_id = ?", paymentID, orderID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if payment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment is not in pending status"})
		return
	}

	now := time.Now()
	payment.Status = "completed"
	payment.UpdatedAt = now

	if err := h.db.Save(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm payment"})
		return
	}

	order.PaidAmount += payment.Amount
	order.UpdatedAt = now
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payment":        payment,
		"paid_amount":    order.PaidAmount,
		"total_amount":   order.TotalAmount,
		"remaining":      order.TotalAmount - order.PaidAmount,
		"message":        "Payment confirmed successfully",
	})
}

// RejectOrderPayment rejects a payment claim for an order (business side)
func (h *BusinessHandler) RejectOrderPayment(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	orderIDStr := c.Param("id")
	paymentIDStr := c.Param("payment_id")

	orderID, _ := strconv.ParseUint(orderIDStr, 10, 32)
	paymentID, _ := strconv.ParseUint(paymentIDStr, 10, 32)

	var order models.Order
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var payment models.Payment
	if err := h.db.Where("id = ? AND order_id = ?", paymentID, orderID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if payment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment is not in pending status"})
		return
	}

	payment.Status = "failed"
	payment.UpdatedAt = time.Now()
	h.db.Save(&payment)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
		"message": "Payment claim rejected",
	})
}

// ConfirmBookingPayment confirms a payment for a booking (business side)
func (h *BusinessHandler) ConfirmBookingPayment(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingIDStr := c.Param("id")
	paymentIDStr := c.Param("payment_id")

	bookingID, _ := strconv.ParseUint(bookingIDStr, 10, 32)
	paymentID, _ := strconv.ParseUint(paymentIDStr, 10, 32)

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var payment models.Payment
	if err := h.db.Where("id = ? AND booking_id = ?", paymentID, bookingID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if payment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment is not in pending status"})
		return
	}

	now := time.Now()
	payment.Status = "completed"
	payment.UpdatedAt = now
	h.db.Save(&payment)

	booking.PaidAmount += payment.Amount
	booking.UpdatedAt = now
	h.db.Save(&booking)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payment":        payment,
		"paid_amount":    booking.PaidAmount,
		"total_amount":   booking.TotalAmount,
		"remaining":      booking.TotalAmount - booking.PaidAmount,
		"message":        "Payment confirmed successfully",
	})
}

// RejectBookingPayment rejects a payment claim for a booking (business side)
func (h *BusinessHandler) RejectBookingPayment(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingIDStr := c.Param("id")
	paymentIDStr := c.Param("payment_id")

	bookingID, _ := strconv.ParseUint(bookingIDStr, 10, 32)
	paymentID, _ := strconv.ParseUint(paymentIDStr, 10, 32)

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var payment models.Payment
	if err := h.db.Where("id = ? AND booking_id = ?", paymentID, bookingID).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
		return
	}

	if payment.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment is not in pending status"})
		return
	}

	payment.Status = "failed"
	payment.UpdatedAt = time.Now()
	h.db.Save(&payment)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"payment": payment,
		"message": "Payment claim rejected",
	})
}

// GetPayments renders the payments dashboard with real data
func (h *BusinessHandler) GetPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}

	// Get all orders and bookings for this business to find their payments
	var orderIDs []uint
	h.db.Model(&models.Order{}).Where("business_id = ?", businessID).Pluck("id", &orderIDs)

	var bookingIDs []uint
	h.db.Model(&models.Booking{}).Where("business_id = ?", businessID).Pluck("id", &bookingIDs)

	// Fetch all related payments
	var payments []models.Payment
	if len(orderIDs) > 0 || len(bookingIDs) > 0 {
		query := h.db.Preload("Client")
		if len(orderIDs) > 0 && len(bookingIDs) > 0 {
			query = query.Where("order_id IN ? OR booking_id IN ?", orderIDs, bookingIDs)
		} else if len(orderIDs) > 0 {
			query = query.Where("order_id IN ?", orderIDs)
		} else {
			query = query.Where("booking_id IN ?", bookingIDs)
		}
		query.Order("created_at DESC").Find(&payments)
	}

	// Enrich payments with order/booking info
	type PaymentRow struct {
		models.Payment
		SourceType   string  `json:"source_type"`   // "order" or "booking"
		SourceNumber string  `json:"source_number"` // OrderNumber or BookingNumber
		ClientName   string  `json:"client_name"`
	}

	var rows []PaymentRow
	var totalCompleted, totalPending, totalFailed float64

	orderMap := make(map[uint]models.Order)
	if len(orderIDs) > 0 {
		var orders []models.Order
		h.db.Where("id IN ?", orderIDs).Find(&orders)
		for _, o := range orders {
			orderMap[o.ID] = o
		}
	}

	bookingMap := make(map[uint]models.Booking)
	if len(bookingIDs) > 0 {
		var bookings []models.Booking
		h.db.Where("id IN ?", bookingIDs).Find(&bookings)
		for _, b := range bookings {
			bookingMap[b.ID] = b
		}
	}

	for _, p := range payments {
		row := PaymentRow{
			Payment:    p,
			ClientName: p.Client.Name,
		}
		if p.OrderID != nil {
			if o, ok := orderMap[*p.OrderID]; ok {
				row.SourceType = "order"
				row.SourceNumber = o.OrderNumber
			}
		} else if p.BookingID != nil {
			if b, ok := bookingMap[*p.BookingID]; ok {
				row.SourceType = "booking"
				row.SourceNumber = b.BookingNumber
			}
		}
		rows = append(rows, row)

		switch p.Status {
		case "completed":
			totalCompleted += p.Amount
		case "pending":
			totalPending += p.Amount
		case "failed":
			totalFailed += p.Amount
		}
	}

	// Revenue from paid amounts on orders/bookings
	var paidOrdersRevenue, paidBookingsRevenue float64
	h.db.Model(&models.Order{}).Select("COALESCE(SUM(paid_amount), 0)").Where("business_id = ?", businessID).Scan(&paidOrdersRevenue)
	h.db.Model(&models.Booking{}).Select("COALESCE(SUM(paid_amount), 0)").Where("business_id = ?", businessID).Scan(&paidBookingsRevenue)
	totalPaidRevenue := paidOrdersRevenue + paidBookingsRevenue

	// Load payment methods
	var paymentMethods []models.PaymentMethod
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	c.HTML(http.StatusOK, "payments.html", gin.H{
		"Business":          business,
		"ActivePage":        "payments",
		"Countries":         data.Countries,
		"Currencies":        data.Currencies,
		"Payments":          rows,
		"PaymentMethods":    paymentMethods,
		"TotalCompleted":    totalCompleted,
		"TotalPending":      totalPending,
		"TotalFailed":       totalFailed,
		"TotalPaidRevenue":  totalPaidRevenue,
		"PaymentCount":      len(rows),
		"Onboarding":        h.onboardingData(businessID),
	})
}

func (h *BusinessHandler) GetPaymentsStats(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	startTime, endTime, _ := timeRangeBounds(r)
	timeClause := "business_id = ? AND created_at BETWEEN ? AND ?"

	var orderIDs []uint
	h.db.Model(&models.Order{}).Where(timeClause, businessID, startTime, endTime).Pluck("id", &orderIDs)

	var bookingIDs []uint
	h.db.Model(&models.Booking{}).Where(timeClause, businessID, startTime, endTime).Pluck("id", &bookingIDs)

	var payments []models.Payment
	if len(orderIDs) > 0 || len(bookingIDs) > 0 {
		query := h.db.Preload("Client")
		if len(orderIDs) > 0 && len(bookingIDs) > 0 {
			query = query.Where("order_id IN ? OR booking_id IN ?", orderIDs, bookingIDs)
		} else if len(orderIDs) > 0 {
			query = query.Where("order_id IN ?", orderIDs)
		} else {
			query = query.Where("booking_id IN ?", bookingIDs)
		}
		query.Order("created_at DESC").Find(&payments)
	}

	type PaymentRow struct {
		models.Payment
		SourceType   string
		SourceNumber string
		ClientName   string
	}

	var rows []PaymentRow
	var totalCompleted, totalPending, totalFailed float64

	orderMap := make(map[uint]models.Order)
	if len(orderIDs) > 0 {
		var orders []models.Order
		h.db.Where("id IN ?", orderIDs).Find(&orders)
		for _, o := range orders {
			orderMap[o.ID] = o
		}
	}

	bookingMap := make(map[uint]models.Booking)
	if len(bookingIDs) > 0 {
		var bookings []models.Booking
		h.db.Where("id IN ?", bookingIDs).Find(&bookings)
		for _, b := range bookings {
			bookingMap[b.ID] = b
		}
	}

	for _, p := range payments {
		row := PaymentRow{
			Payment:    p,
			ClientName: p.Client.Name,
		}
		if p.OrderID != nil {
			if o, ok := orderMap[*p.OrderID]; ok {
				row.SourceType = "order"
				row.SourceNumber = o.OrderNumber
			}
		} else if p.BookingID != nil {
			if b, ok := bookingMap[*p.BookingID]; ok {
				row.SourceType = "booking"
				row.SourceNumber = b.BookingNumber
			}
		}
		rows = append(rows, row)

		switch p.Status {
		case "completed":
			totalCompleted += p.Amount
		case "pending":
			totalPending += p.Amount
		case "failed":
			totalFailed += p.Amount
		}
	}

	var paidOrdersRevenue, paidBookingsRevenue float64
	h.db.Model(&models.Order{}).Where(timeClause, businessID, startTime, endTime).Select("COALESCE(SUM(paid_amount), 0)").Scan(&paidOrdersRevenue)
	h.db.Model(&models.Booking{}).Where(timeClause, businessID, startTime, endTime).Select("COALESCE(SUM(paid_amount), 0)").Scan(&paidBookingsRevenue)

	var paymentMethods []models.PaymentMethod
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	c.HTML(http.StatusOK, "payments_content", gin.H{
		"Business":         business,
		"ActivePage":       "payments",
		"Payments":         rows,
		"PaymentMethods":   paymentMethods,
		"TotalCompleted":   totalCompleted,
		"TotalPending":     totalPending,
		"TotalFailed":      totalFailed,
		"TotalPaidRevenue": paidOrdersRevenue + paidBookingsRevenue,
		"PaymentCount":     len(rows),
	})
}

// ConfirmAllOrderPayments confirms all pending payments for an order at once
func (h *BusinessHandler) ConfirmAllOrderPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.ParseUint(orderIDStr, 10, 32)

	var order models.Order
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var pendingPayments []models.Payment
	h.db.Where("order_id = ? AND status = ?", orderID, "pending").Find(&pendingPayments)

	if len(pendingPayments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No pending payments to confirm"})
		return
	}

	now := time.Now()
	var totalConfirmed float64
	for _, payment := range pendingPayments {
		payment.Status = "completed"
		payment.UpdatedAt = now
		h.db.Save(&payment)
		totalConfirmed += payment.Amount
	}

	order.PaidAmount += totalConfirmed
	order.UpdatedAt = now
	h.db.Save(&order)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"confirmed":      len(pendingPayments),
		"total_amount":   totalConfirmed,
		"paid_amount":    order.PaidAmount,
		"message":        fmt.Sprintf("Confirmed %d payment(s) totaling $%.2f", len(pendingPayments), totalConfirmed),
	})
}

// ConfirmAllBookingPayments confirms all pending payments for a booking at once
func (h *BusinessHandler) ConfirmAllBookingPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	bookingIDStr := c.Param("id")
	bookingID, _ := strconv.ParseUint(bookingIDStr, 10, 32)

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var pendingPayments []models.Payment
	h.db.Where("booking_id = ? AND status = ?", bookingID, "pending").Find(&pendingPayments)

	if len(pendingPayments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No pending payments to confirm"})
		return
	}

	now := time.Now()
	var totalConfirmed float64
	for _, payment := range pendingPayments {
		payment.Status = "completed"
		payment.UpdatedAt = now
		h.db.Save(&payment)
		totalConfirmed += payment.Amount
	}

	booking.PaidAmount += totalConfirmed
	booking.UpdatedAt = now
	h.db.Save(&booking)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"confirmed":      len(pendingPayments),
		"total_amount":   totalConfirmed,
		"paid_amount":    booking.PaidAmount,
		"message":        fmt.Sprintf("Confirmed %d payment(s) totaling $%.2f", len(pendingPayments), totalConfirmed),
	})
}

// GetOrderPayments returns payments for a specific order
func (h *BusinessHandler) GetOrderPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var payments []models.Payment
	h.db.Where("order_id = ?", orderID).Order("created_at DESC").Find(&payments)

	var pendingAmt float64
	h.db.Model(&models.Payment{}).Where("order_id = ? AND status = ?", orderID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payments":       payments,
		"paid_amount":    order.PaidAmount,
		"pending_amount": pendingAmt,
		"total":          order.TotalAmount,
		"remaining":      order.TotalAmount - order.PaidAmount,
	})
}

// GetBookingPayments returns payments for a specific booking
func (h *BusinessHandler) GetBookingPayments(c *gin.Context) {
	businessID := c.GetUint("business_id")
	bookingIDStr := c.Param("id")

	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := h.db.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var payments []models.Payment
	h.db.Where("booking_id = ?", bookingID).Order("created_at DESC").Find(&payments)

	var pendingAmt float64
	h.db.Model(&models.Payment{}).Where("booking_id = ? AND status = ?", bookingID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payments":       payments,
		"paid_amount":    booking.PaidAmount,
		"pending_amount": pendingAmt,
		"total":          booking.TotalAmount,
		"remaining":      booking.TotalAmount - booking.PaidAmount,
	})
}

// buildOrderPaymentsData enriches order data with payment info for templates
func buildOrderPaymentsData(order models.Order, dbPaymentInfo []models.Payment) map[string]interface{} {
	var pendingPayments []map[string]interface{}
	var completedPayments []map[string]interface{}

	for _, p := range dbPaymentInfo {
		pm := map[string]interface{}{
			"id":         p.ID,
			"amount":     p.Amount,
			"method":     p.Method,
			"status":     p.Status,
			"reference":  p.Reference,
			"notes":      p.Notes,
			"created_at": p.CreatedAt,
		}
		if p.Status == "pending" {
			pendingPayments = append(pendingPayments, pm)
		} else {
			completedPayments = append(completedPayments, pm)
		}
	}

	return map[string]interface{}{
		"paid_amount":        order.PaidAmount,
		"total_amount":       order.TotalAmount,
		"remaining":          order.TotalAmount - order.PaidAmount,
		"is_fully_paid":      order.PaidAmount >= order.TotalAmount,
		"pending_payments":   pendingPayments,
		"completed_payments": completedPayments,
	}
}

// GetPaymentMethods returns all payment methods for the business
func (h *BusinessHandler) GetPaymentMethods(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var methods []models.PaymentMethod
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, id ASC").Find(&methods)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"methods": methods,
	})
}

// CreatePaymentMethod adds a new payment method
func (h *BusinessHandler) CreatePaymentMethod(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var req struct {
		MethodType string         `json:"method_type" binding:"required"`
		Label      string         `json:"label" binding:"required"`
		Details    map[string]any `json:"details"`
		SortOrder  int            `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Details == nil {
		req.Details = make(map[string]any)
	}

	method := models.PaymentMethod{
		BusinessID: businessID,
		MethodType: req.MethodType,
		Label:      req.Label,
		Details:    req.Details,
		IsActive:   true,
		SortOrder:  req.SortOrder,
	}

	if err := h.db.Create(&method).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"method":  method,
	})
}

// UpdatePaymentMethod updates an existing payment method
func (h *BusinessHandler) UpdatePaymentMethod(c *gin.Context) {
	businessID := c.GetUint("business_id")
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	var method models.PaymentMethod
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&method).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	var req struct {
		MethodType *string        `json:"method_type"`
		Label      *string        `json:"label"`
		Details    *map[string]any `json:"details"`
		IsActive   *bool          `json:"is_active"`
		SortOrder  *int           `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MethodType != nil {
		method.MethodType = *req.MethodType
	}
	if req.Label != nil {
		method.Label = *req.Label
	}
	if req.Details != nil {
		method.Details = *req.Details
	}
	if req.IsActive != nil {
		method.IsActive = *req.IsActive
	}
	if req.SortOrder != nil {
		method.SortOrder = *req.SortOrder
	}

	if err := h.db.Save(&method).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"method":  method,
	})
}

// DeletePaymentMethod removes a payment method
func (h *BusinessHandler) DeletePaymentMethod(c *gin.Context) {
	businessID := c.GetUint("business_id")
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	var method models.PaymentMethod
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&method).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	if err := h.db.Delete(&method).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) GetPaymentsStatsGrid(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	r := c.DefaultQuery("range", "this_month")
	startTime, endTime, _ := timeRangeBounds(r)
	timeClause := "business_id = ? AND created_at BETWEEN ? AND ?"

	var orderIDs []uint
	h.db.Model(&models.Order{}).Where(timeClause, businessID, startTime, endTime).Pluck("id", &orderIDs)

	var bookingIDs []uint
	h.db.Model(&models.Booking{}).Where(timeClause, businessID, startTime, endTime).Pluck("id", &bookingIDs)

	var payments []models.Payment
	if len(orderIDs) > 0 || len(bookingIDs) > 0 {
		query := h.db.Preload("Client")
		if len(orderIDs) > 0 && len(bookingIDs) > 0 {
			query = query.Where("order_id IN ? OR booking_id IN ?", orderIDs, bookingIDs)
		} else if len(orderIDs) > 0 {
			query = query.Where("order_id IN ?", orderIDs)
		} else {
			query = query.Where("booking_id IN ?", bookingIDs)
		}
		query.Order("created_at DESC").Find(&payments)
	}

	var totalCompleted, totalPending, totalFailed float64
	for _, p := range payments {
		switch p.Status {
		case "completed":
			totalCompleted += p.Amount
		case "pending":
			totalPending += p.Amount
		case "failed":
			totalFailed += p.Amount
		}
	}

	var paidOrdersRevenue, paidBookingsRevenue float64
	h.db.Model(&models.Order{}).Where(timeClause, businessID, startTime, endTime).Select("COALESCE(SUM(paid_amount), 0)").Scan(&paidOrdersRevenue)
	h.db.Model(&models.Booking{}).Where(timeClause, businessID, startTime, endTime).Select("COALESCE(SUM(paid_amount), 0)").Scan(&paidBookingsRevenue)

	c.HTML(http.StatusOK, "payments_stats_grid", gin.H{
		"Business":         business,
		"TotalCompleted":   totalCompleted,
		"TotalPending":     totalPending,
		"TotalFailed":      totalFailed,
		"TotalPaidRevenue": paidOrdersRevenue + paidBookingsRevenue,
	})
}

func logPaymentError(msg string, err error) {
	log.Printf("[Payment] %s: %v", msg, err)
}
