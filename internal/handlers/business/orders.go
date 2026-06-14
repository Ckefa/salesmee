package business

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"salesmee/internal/data"
	"salesmee/internal/handlers"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/notifier"
	"salesmee/internal/ws"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateOrder  Creation
func (h *BusinessHandler) CreateOrder(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var request struct {
		ClientID        uint   `json:"client_id"`
		ProductID       uint   `json:"product_id" binding:"required"`
		Quantity        int    `json:"quantity" binding:"required"`
		CustomerName    string `json:"customer_name"`
		CustomerEmail   string `json:"customer_email"`
		CustomerPhone   string `json:"customer_phone"`
		DeliveryAddress string `json:"delivery_address"`
		Notes           string `json:"notes"`
		MarkCompleted   bool   `json:"mark_completed"`
		LocationID      *uint  `json:"location_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get product details
	var product models.Product
	if err := h.db.Where("id = ? AND business_id = ?", request.ProductID, businessID).First(&product).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Check stock availability
	if product.Stock < request.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
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

	status := "pending"
	if request.MarkCompleted {
		status = "fulfilled"
	}

	// Create order
	order := models.Order{
		BusinessID:   businessID,
		ClientID:     client.ID,
		Quantity:     request.Quantity,
		OrderNumber:  generateOrderNumber(),
		Status:       status,
		Sender:       "business",
		TotalAmount:  float64(request.Quantity) * product.Price,
		Notes:        fmt.Sprintf("Delivery: %s. %s", request.DeliveryAddress, request.Notes),
		DeliveryDate: &[]time.Time{time.Now().AddDate(0, 0, 7)}[0],
		LocationID:   request.LocationID,
	}

	if request.MarkCompleted {
		order.PaidAmount = order.TotalAmount
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Create order item
	orderItem := models.OrderItem{
		OrderID:    order.ID,
		ProductID:  product.ID,
		Quantity:   request.Quantity,
		UnitPrice:  product.Price,
		TotalPrice: float64(request.Quantity) * product.Price,
	}

	if err := h.db.Create(&orderItem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order item"})
		return
	}

	// Update product stock
	product.Stock -= request.Quantity
	if err := h.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
		return
	}

	// Create inventory log
	inventoryLog := models.InventoryLog{
		ProductID: product.ID,
		Type:      "out",
		Quantity:  request.Quantity,
		Reason:    fmt.Sprintf("Order #%s", order.OrderNumber),
	}
	h.db.Create(&inventoryLog)

	// If mark_completed, create payment record
	if request.MarkCompleted {
		payment := models.Payment{
			OrderID:   &order.ID,
			Amount:    order.TotalAmount,
			Method:    "cash",
			Status:    "completed",
			Reference: "Walk-in counter payment",
		}
		h.db.Create(&payment)
	}

	// Auto-advance conversation progress
	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", client.ID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": fmt.Sprintf("Order %s created successfully", order.OrderNumber),
	})
}

func (h *BusinessHandler) GetOrders(c *gin.Context) {
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
	h.db.Model(&models.Order{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var draftCount, pendingCount, clientConfirmedCount, confirmedCount, fulfilledCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "draft":
			draftCount = sc.Count
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case "confirmed":
			confirmedCount = sc.Count
		case "fulfilled":
			fulfilledCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	// Total revenue for full date range
	var totalRevenue float64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	// Total count for pagination
	var totalCount int64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	// Paginated orders
	var orders []models.Order
	h.db.Preload("Client").Preload("OrderItems").Preload("OrderItems.Product").
		Where(baseWhere, baseArgs...).
		Order("created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&orders)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/orders_content", gin.H{
			"Business":             currentBusiness,
			"Orders":               orders,
			"DraftCount":           draftCount,
			"PendingCount":         pendingCount,
			"ClientConfirmedCount": clientConfirmedCount,
			"ConfirmedCount":       confirmedCount,
			"FulfilledCount":       fulfilledCount,
			"CancelledCount":       cancelledCount,
			"TotalOrders":          totalCount,
			"TotalRevenue":         totalRevenue,
			"Page":                 float64(page),
			"TotalPages":           float64(totalPages),
			"PageSize":             pageSize,
			"Range":                r,
			"Countries":            data.Countries,
			"Currencies":           data.Currencies,
			"Onboarding":           h.onboardingData(businessID),
			"Locations":            locations,
			"AuthType":             c.GetString("auth_type"),
			"Role":                 c.GetString("role"),
			"ActivePage":           "orders",
		})
		return
	}

	c.HTML(http.StatusOK, "orders.html", gin.H{
		"Business":             currentBusiness,
		"Orders":               orders,
		"DraftCount":           draftCount,
		"PendingCount":         pendingCount,
		"ClientConfirmedCount": clientConfirmedCount,
		"ConfirmedCount":       confirmedCount,
		"FulfilledCount":       fulfilledCount,
		"CancelledCount":       cancelledCount,
		"TotalOrders":          totalCount,
		"TotalRevenue":         totalRevenue,
		"Page":                 float64(page),
		"TotalPages":           float64(totalPages),
		"PageSize":             pageSize,
		"Range":                r,
		"Countries":            data.Countries,
		"Currencies":           data.Currencies,
		"Onboarding":           h.onboardingData(businessID),
		"Locations":            locations,
		"AuthType":             c.GetString("auth_type"),
		"Role":                 c.GetString("role"),
		"ActivePage":           "orders",
	})
}

func (h *BusinessHandler) GetOrdersStats(c *gin.Context) {
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

	// Status counts for full date range
	var statusCounts []struct {
		Status string
		Count  int64
	}
	h.db.Model(&models.Order{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var draftCount, pendingCount, clientConfirmedCount, confirmedCount, fulfilledCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "draft":
			draftCount = sc.Count
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case "confirmed":
			confirmedCount = sc.Count
		case "fulfilled":
			fulfilledCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	// Total revenue for full date range
	var totalRevenue float64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	// Total count for pagination
	var totalCount int64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	// Paginated orders
	var orders []models.Order
	h.db.Preload("Client").Preload("OrderItems").Preload("OrderItems.Product").
		Where(baseWhere, baseArgs...).
		Order("created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&orders)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	c.HTML(http.StatusOK, "dashboard/orders_content", gin.H{
		"Business":             currentBusiness,
		"Orders":               orders,
		"DraftCount":           draftCount,
		"PendingCount":         pendingCount,
		"ClientConfirmedCount": clientConfirmedCount,
		"ConfirmedCount":       confirmedCount,
		"FulfilledCount":       fulfilledCount,
		"CancelledCount":       cancelledCount,
		"TotalOrders":          totalCount,
		"TotalRevenue":         totalRevenue,
		"Page":                 float64(page),
		"TotalPages":           float64(totalPages),
		"PageSize":             pageSize,
		"Range":                r,
		"ActivePage":           "orders",
		"Locations":            locations,
		"AuthType":             c.GetString("auth_type"),
		"Role":                 c.GetString("role"),
	})
}

func (h *BusinessHandler) GetOrdersStatsGrid(c *gin.Context) {
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
	h.db.Model(&models.Order{}).Select("status, COUNT(*) as count").Where(baseWhere, baseArgs...).Group("status").Scan(&statusCounts)
	var draftCount, pendingCount, clientConfirmedCount, confirmedCount, fulfilledCount, cancelledCount int64
	for _, sc := range statusCounts {
		switch sc.Status {
		case "draft":
			draftCount = sc.Count
		case "pending":
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case "confirmed":
			confirmedCount = sc.Count
		case "fulfilled":
			fulfilledCount = sc.Count
		case "cancelled":
			cancelledCount = sc.Count
		}
	}

	var totalOrders int64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Count(&totalOrders)

	var totalRevenue float64
	h.db.Model(&models.Order{}).Where(baseWhere, baseArgs...).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	c.HTML(http.StatusOK, "orders_stats_grid", gin.H{
		"Business":             currentBusiness,
		"DraftCount":           int(draftCount),
		"PendingCount":         int(pendingCount),
		"ClientConfirmedCount": int(clientConfirmedCount),
		"ConfirmedCount":       int(confirmedCount),
		"FulfilledCount":       int(fulfilledCount),
		"CancelledCount":       int(cancelledCount),
		"TotalOrders":          int(totalOrders),
		"TotalRevenue":         totalRevenue,
	})
}

func (h *BusinessHandler) UpdateOrderStatus(c *gin.Context) {
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

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", id, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	order.Status = request.Status
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}

	h.sendOrderNotif(order, request.Status)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

// ClientCreateOrder allows customers to create orders
func (h *BusinessHandler) ClientCreateOrder(c *gin.Context) {
	// Get client ID from authenticated context (set by ClientMiddleware)
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated as client"})
		return
	}

	var request struct {
		BusinessID uint `json:"business_id" binding:"required"`
		ProductID  uint `json:"product_id"`
		Quantity   int  `json:"quantity"`
		Items      []struct {
			ProductID uint `json:"product_id"`
			Quantity  int  `json:"quantity"`
		} `json:"items"`
		DeliveryAddress string `json:"delivery_address"`
		Notes           string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Support both single-item and multi-item formats
	var itemList []struct {
		ProductID uint
		Quantity  int
	}

	if len(request.Items) > 0 {
		itemList = make([]struct {
			ProductID uint
			Quantity  int
		}, len(request.Items))
		for i, item := range request.Items {
			itemList[i] = struct {
				ProductID uint
				Quantity  int
			}{ProductID: item.ProductID, Quantity: item.Quantity}
		}
	} else {
		if request.ProductID == 0 || request.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id and quantity are required"})
			return
		}
		itemList = []struct {
			ProductID uint
			Quantity  int
		}{{ProductID: request.ProductID, Quantity: request.Quantity}}
	}

	// Get client by primary key
	var client models.Client
	if err := h.db.First(&client, clientID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find client"})
		return
	}

	// Build order
	var totalAmount float64
	var firstProductName string
	var orderItems []models.OrderItem

	for _, item := range itemList {
		var product models.Product
		if err := h.db.First(&product, item.ProductID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %d not found", item.ProductID)})
			return
		}
		if firstProductName == "" {
			firstProductName = product.Name
		}
		itemTotal := float64(item.Quantity) * product.Price
		totalAmount += itemTotal
		orderItems = append(orderItems, models.OrderItem{
			ProductID:  product.ID,
			Quantity:   item.Quantity,
			UnitPrice:  product.Price,
			TotalPrice: itemTotal,
		})
	}

	now := time.Now()
	order := models.Order{
		BusinessID:  request.BusinessID,
		ClientID:    client.ID,
		OrderNumber: generateOrderNumber(),
		Status:      "pending",
		Sender:      "client",
		Quantity:    len(itemList),
		TotalAmount: totalAmount,
		Notes:       request.Notes,
		Draft:       false,
		CreatedAt:   now,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	if err := h.db.Create(&orderItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order items"})
		return
	}

	// Deduct stock
	for _, item := range itemList {
		var product models.Product
		h.db.First(&product, item.ProductID)
		product.Stock -= item.Quantity
		h.db.Save(&product)
		h.db.Create(&models.InventoryLog{
			ProductID: product.ID,
			Type:      "out",
			Quantity:  item.Quantity,
			Reason:    fmt.Sprintf("Order #%s", order.OrderNumber),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order":        order,
		"product_name": firstProductName,
		"quantity":     len(itemList),
	})
}

// UpdateOrder updates an existing order's items, notes, and delivery address
func (h *BusinessHandler) UpdateOrder(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("OrderItems").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var request struct {
		Items []struct {
			ProductID uint `json:"product_id"`
			Quantity  int  `json:"quantity"`
		} `json:"items"`
		Notes           string `json:"notes"`
		DeliveryAddress string `json:"delivery_address"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one item is required"})
		return
	}

	// Restore stock from old items
	for _, oldItem := range order.OrderItems {
		var product models.Product
		h.db.First(&product, oldItem.ProductID)
		product.Stock += oldItem.Quantity
		h.db.Save(&product)
	}

	// Delete old order items
	h.db.Where("order_id = ?", order.ID).Delete(&models.OrderItem{})

	// Build new items and calculate total
	var totalAmount float64
	var orderItems []models.OrderItem

	for _, item := range request.Items {
		var product models.Product
		if err := h.db.Where("id = ? AND business_id = ?", item.ProductID, businessID).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %d not found", item.ProductID)})
			return
		}
		if product.Stock < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for %s", product.Name)})
			return
		}
		itemTotal := float64(item.Quantity) * product.Price
		totalAmount += itemTotal
		orderItems = append(orderItems, models.OrderItem{
			OrderID:    order.ID,
			ProductID:  product.ID,
			Quantity:   item.Quantity,
			UnitPrice:  product.Price,
			TotalPrice: itemTotal,
		})

		product.Stock -= item.Quantity
		h.db.Save(&product)
	}

	// Update order
	order.TotalAmount = totalAmount
	order.Quantity = len(request.Items)

	fullNotes := request.Notes
	if request.DeliveryAddress != "" {
		fullNotes = "📍 Delivery Address: " + request.DeliveryAddress + "\n" + fullNotes
	}
	order.Notes = fullNotes
	order.UpdatedAt = time.Now()

	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	// Create new order items
	for i := range orderItems {
		if err := h.db.Create(&orderItems[i]).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order items"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%d", time.Now().Unix())
}

// GetConversationProducts returns all active products for the business in a conversation
func (h *BusinessHandler) GetConversationProducts(c *gin.Context) {
	convIDStr := c.Param("conversation_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var conv models.Conversation
	if err := h.db.First(&conv, convID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	var products []models.Product
	if err := h.db.Where("business_id = ? AND is_active = ?", conv.BusinessID, true).
		Order("name ASC").Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
	})
}

func (h *BusinessHandler) GetConversationServices(c *gin.Context) {
	convIDStr := c.Param("conversation_id")
	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var conv models.Conversation
	if err := h.db.First(&conv, convID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	var services []models.Service
	if err := h.db.Where("business_id = ? AND is_active = ?", conv.BusinessID, true).
		Order("name ASC").Find(&services).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"services": services,
	})
}

// CreateOrderDraft creates a draft order inline in the chat (HTMX partial)
func (h *BusinessHandler) CreateOrderDraft(c *gin.Context) {
	businessID := c.GetUint("business_id")
	conversationIDStr := c.Param("conversation_id")

	convID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	var request struct {
		Items []struct {
			ProductID uint `json:"product_id"`
			Quantity  int  `json:"quantity"`
		} `json:"items"`
		Notes           string `json:"notes"`
		DeliveryAddress string `json:"delivery_address"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one item is required"})
		return
	}

	// Get conversation to find client
	var conversation models.Conversation
	if err := h.db.First(&conversation, convID).Error; err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	if conversation.BusinessID != businessID {
		c.String(http.StatusForbidden, "Not your conversation")
		return
	}

	// Build order items and calculate total
	var orderItems []models.OrderItem
	var totalAmount float64
	var productNames []string
	var firstProductName string

	for _, item := range request.Items {
		var product models.Product
		if err := h.db.Where("id = ? AND business_id = ?", item.ProductID, businessID).First(&product).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %d not found", item.ProductID)})
			return
		}
		if product.Stock < item.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for %s", product.Name)})
			return
		}
		if firstProductName == "" {
			firstProductName = product.Name
		}
		productNames = append(productNames, product.Name)
		itemTotal := float64(item.Quantity) * product.Price
		totalAmount += itemTotal
		orderItems = append(orderItems, models.OrderItem{
			ProductID:  product.ID,
			Quantity:   item.Quantity,
			UnitPrice:  product.Price,
			TotalPrice: itemTotal,
		})
	}

	now := time.Now()
	fullNotes := request.Notes
	if request.DeliveryAddress != "" {
		fullNotes = "📍 Delivery Address: " + request.DeliveryAddress + "\n" + fullNotes
	}
	order := models.Order{
		BusinessID:  businessID,
		ClientID:    conversation.ClientID,
		OrderNumber: generateOrderNumber(),
		Status:      "draft",
		Sender:      "business",
		Quantity:    len(request.Items),
		TotalAmount: totalAmount,
		Notes:       fullNotes,
		Draft:       true,
		CreatedAt:   now,
	}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Save order items
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}
	if err := h.db.Create(&orderItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order items"})
		return
	}

	// Deduct stock and create inventory log for each item
	for _, item := range request.Items {
		var product models.Product
		h.db.First(&product, item.ProductID)
		product.Stock -= item.Quantity
		h.db.Save(&product)
		h.db.Create(&models.InventoryLog{
			ProductID: product.ID,
			Type:      "out",
			Quantity:  item.Quantity,
			Reason:    fmt.Sprintf("Draft order #%s", order.OrderNumber),
		})
	}

	// Create Message for this order so it appears in chat
	msg := models.Message{
		ConversationID: conversation.ID,
		Content:        "",
		Type:           "order",
		Sender:         "user",
		CreatedAt:      now,
	}
	if err := h.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"order":         order,
		"order_id":      order.ID,
		"order_number":  order.OrderNumber,
		"status":        order.Status,
		"total_amount":  order.TotalAmount,
		"product_names": productNames,
		"items":         request.Items,
		"draft":         true,
	})
}

// SendOrderToClient publishes a draft order to the client
func (h *BusinessHandler) SendOrderToClient(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "draft" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not in draft status"})
		return
	}

	order.Status = "pending"
	order.Draft = false
	now := time.Now()
	order.UpdatedAt = now
	h.db.Save(&order)

	h.sendOrderNotif(order, "pending")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order":        order,
		"order_id":     order.ID,
		"order_number": order.OrderNumber,
		"status":       order.Status,
		"total_amount": order.TotalAmount,
	})
}

// ConfirmOrderBusiness confirms the order from the business side
func (h *BusinessHandler) ConfirmOrderBusiness(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "pending" && order.Status != "client_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be confirmed in current status"})
		return
	}

	now := time.Now()
	order.ConfirmedByBusiness = true
	order.ConfirmedByBusinessAt = &now
	order.Status = "confirmed"
	order.UpdatedAt = now
	h.db.Save(&order)

	h.sendOrderNotif(order, "confirmed")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order confirmed successfully",
	})
}

// RejectOrder cancels/rejects an order
func (h *BusinessHandler) RejectOrder(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status == "confirmed" || order.Status == "fulfilled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reject a confirmed/fulfilled order"})
		return
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now()
	h.db.Save(&order)

	h.sendOrderNotif(order, "cancelled")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order rejected/cancelled",
	})
}

// FulfillOrder transitions confirmed → fulfilled
func (h *BusinessHandler) FulfillOrder(c *gin.Context) {
	businessID := c.GetUint("business_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "confirmed" && order.Status != "client_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be confirmed before fulfillment"})
		return
	}

	now := time.Now()
	order.Status = "fulfilled"
	order.UpdatedAt = now
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fulfill order"})
		return
	}

	h.sendOrderNotif(order, "completed")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order fulfilled successfully",
	})
}

// GetOrderReceipt renders a print-friendly receipt for a completed order
func (h *BusinessHandler) GetOrderReceipt(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := h.db.Preload("Client").Preload("OrderItems.Product").Preload("Payments").Preload("Business").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "fulfilled" && order.Status != "completed" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Order is not completed"})
		return
	}

	c.HTML(http.StatusOK, "receipt_order.html", gin.H{
		"Order":    order,
		"Business": order.Business,
	})
}

// MarkOrderAsPaid sets the order's paid amount to the total (quick mark as fully paid)
func (h *BusinessHandler) MarkOrderAsPaid(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	orderIDStr := c.Param("id")
	orderID, _ := strconv.ParseUint(orderIDStr, 10, 32)

	var order models.Order
	if err := h.db.Preload("Client").Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be confirmed before marking as paid"})
		return
	}

	order.PaidAmount = order.TotalAmount
	order.UpdatedAt = time.Now()
	h.db.Save(&order)

	// Create a completed payment record
	h.db.Create(&models.Payment{
		OrderID:   &order.ID,
		ClientID:  order.ClientID,
		Amount:    order.TotalAmount,
		Method:    "cash",
		Status:    "completed",
		Reference: "quick-paid",
		Notes:     "Marked as paid from dashboard",
	})

	h.sendOrderNotif(order, "paid")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		handlers.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		ws.BroadcastOrderUpdate(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order marked as paid",
	})
}

func (h *BusinessHandler) sendOrderNotif(order models.Order, status string) {
	prefs, err := notifier.GetOrCreatePrefs(h.db, order.BusinessID)
	if err != nil || !prefs.OrderStatusChange {
		return
	}
	var client models.Client
	if err := h.db.First(&client, order.ClientID).Error; err != nil || client.Email == "" {
		return
	}
	var biz models.Business
	if err := h.db.First(&biz, order.BusinessID).Error; err != nil {
		return
	}

	statusLabel := status
	notifType := "order_status"
	rid := order.ID
	if notifier.HasBeenSent(h.db, order.BusinessID, client.ID, notifType, &rid) {
		return
	}

	chatLink := fmt.Sprintf("https://%s/client/businesses/%d/messages", os.Getenv("APP_DOMAIN"), biz.ID)
	if os.Getenv("APP_DOMAIN") == "" {
		chatLink = fmt.Sprintf("/client/businesses/%d/messages", biz.ID)
	}

	if err := services.SendOrderStatusEmail(client.Email, client.Name, biz.Name, order.OrderNumber, statusLabel, chatLink); err != nil {
		notifier.MarkNotificationSent(h.db, order.BusinessID, client.ID, notifType, "order", &rid, client.Email, "failed")
		return
	}
	notifier.MarkNotificationSent(h.db, order.BusinessID, client.ID, notifType, "order", &rid, client.Email, "sent")
	notifier.CreateInAppNotif(h.db, order.BusinessID, &client.ID,
		fmt.Sprintf("Order %s", statusLabel),
		fmt.Sprintf("Order %s is now %s", order.OrderNumber, statusLabel),
		"fa-shopping-cart",
		"/business/orders")
}

// buildOrderData creates the rich order data map for templates
func buildOrderData(order models.Order, orderItems []models.OrderItem, productNames []string, firstProductName string, paymentMethods []models.PaymentMethod) map[string]interface{} {
	var items []map[string]interface{}
	for _, item := range orderItems {
		itemMap := map[string]interface{}{
			"product_id":  item.ProductID,
			"name":        item.Product.Name,
			"quantity":    item.Quantity,
			"unit_price":  item.UnitPrice,
			"total_price": item.TotalPrice,
			"image_url":   item.Product.ImageURL,
		}
		if item.Product.ID == 0 {
			itemMap["name"] = "Product"
		}
		items = append(items, itemMap)
	}

	var actionRequired string
	editable := false

	switch order.Status {
	case "draft":
		actionRequired = "none"
		editable = true
	case "pending":
		if order.Sender == "business" && !order.ConfirmedByClient {
			actionRequired = "client"
			editable = true // client can edit quantities before confirming
		} else if order.Sender == "client" && !order.ConfirmedByBusiness {
			actionRequired = "business"
			editable = false
		} else {
			actionRequired = "none"
			editable = false
		}
	case "client_confirmed":
		actionRequired = "business"
		editable = false
	case "confirmed":
		actionRequired = "none"
		editable = false
	case "fulfilled":
		actionRequired = "none"
		editable = false
	case "cancelled":
		actionRequired = "none"
		editable = false
	default:
		actionRequired = "none"
		editable = false
	}

	if firstProductName == "" && len(productNames) > 0 {
		firstProductName = productNames[0]
	}

	remaining := order.TotalAmount - order.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	return map[string]interface{}{
		"id":                   order.ID,
		"order_number":         order.OrderNumber,
		"status":               order.Status,
		"client_confirmed":     order.ConfirmedByClient,
		"business_confirmed":   order.ConfirmedByBusiness,
		"action_required":      actionRequired,
		"editable":             editable,
		"sender":               order.Sender,
		"draft":                order.Draft,
		"items":                items,
		"total_amount":         order.TotalAmount,
		"paid_amount":          order.PaidAmount,
		"remaining":            remaining,
		"is_fully_paid":        order.PaidAmount >= order.TotalAmount,
		"quantity":             order.Quantity,
		"notes":                order.Notes,
		"product_names":        productNames,
		"first_product_name":   firstProductName,
		"created_at":           order.CreatedAt,
		"payment_methods":      paymentMethods,
	}
}
