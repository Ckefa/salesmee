package business

import (
	"fmt"
	"net/http"
	"salesmee/internal/data"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/notifier"
	"salesmee/internal/services/progress"
	"salesmee/internal/ws"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateOrder  Creation
func (h *OrderHandler) CreateOrder(c *gin.Context) {
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

	// Create order in a transaction with row-level locking to prevent overselling
	var order models.Order
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var product models.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND business_id = ?", request.ProductID, businessID).
			First(&product).Error; err != nil {
			return err
		}

		if product.Stock < request.Quantity {
			return fmt.Errorf("insufficient stock")
		}

		status := models.OrderPending
		if request.MarkCompleted {
			status = models.OrderFulfilled
		}

		order = models.Order{
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

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		orderItem := models.OrderItem{
			OrderID:    order.ID,
			ProductID:  product.ID,
			Quantity:   request.Quantity,
			UnitPrice:  product.Price,
			TotalPrice: float64(request.Quantity) * product.Price,
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			return err
		}

		product.Stock -= request.Quantity
		if err := tx.Save(&product).Error; err != nil {
			return err
		}

		if err := tx.Create(&models.InventoryLog{
			ProductID: product.ID,
			Type:      "out",
			Quantity:  request.Quantity,
			Reason:    fmt.Sprintf("Order #%s", order.OrderNumber),
		}).Error; err != nil {
			return err
		}

		if request.MarkCompleted {
			if err := tx.Create(&models.Payment{
				OrderID:   &order.ID,
				Amount:    order.TotalAmount,
				Method:    "cash",
				Status:    models.OrderCompleted,
				Reference: "Walk-in counter payment",
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if err.Error() == "insufficient stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Product not found"})
		}
		return
	}

	// Auto-advance conversation progress and notify any open chat panes.
	if conv, err := getOrCreateConversation(h.db, client.ID, businessID); err == nil {
		progress.AutoCalculateProgress(conv.ID)
		if h.hub != nil {
			ws.BroadcastNewMessage(
				h.hub,
				strconv.Itoa(int(conv.ID)),
				strconv.Itoa(int(businessID)),
				"business",
				strconv.Itoa(int(order.ID+10000)),
				"",
				"",
				"",
				"order",
				nil,
				order.CreatedAt,
				strconv.Itoa(int(businessID)),
				strconv.Itoa(int(client.ID)),
			)
			var clientUnread int64
			h.db.Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conv.ID, false).
				Count(&clientUnread)
			var bizUnread int64
			h.db.Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conv.ID, false).
				Count(&bizUnread)
			ws.BroadcastUnreadCount(h.hub, strconv.Itoa(int(conv.ID)), int32(clientUnread), strconv.Itoa(int(client.ID)), "client")

			var biz models.Business
			h.db.First(&biz, businessID)
			bizCardSidebar := RenderBizSidebarCard(client, conv.ID, order.Notes, order.CreatedAt, int(bizUnread))
			clientCardSidebar := RenderClientSidebarCard(biz, conv.ID, order.Notes, order.CreatedAt, int(clientUnread))
			ws.BroadcastConversationUpdate(h.hub, strconv.Itoa(int(conv.ID)), bizCardSidebar, clientCardSidebar, strconv.Itoa(int(businessID)), strconv.Itoa(int(client.ID)))

			bizCardHTML := renderBizOrderCard(h.db, order)
			clientCardHTML := renderClientOrderCard(h.db, order)
			ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(client.ID)))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": fmt.Sprintf("Order %s created successfully", order.OrderNumber),
	})
}

func (h *OrderHandler) GetOrders(c *gin.Context) {
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
		case models.OrderDraft:
			draftCount = sc.Count
		case models.OrderPending:
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case models.OrderConfirmed:
			confirmedCount = sc.Count
		case models.OrderFulfilled:
			fulfilledCount = sc.Count
		case models.OrderCancelled:
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
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'confirmed' THEN 2 WHEN status = 'draft' THEN 3 WHEN status IN ('fulfilled','completed') THEN 4 WHEN status = 'cancelled' THEN 5 ELSE 6 END, created_at DESC`).
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
			"Onboarding":           onboardingData(h.db, businessID),
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
		"Onboarding":           onboardingData(h.db, businessID),
		"Locations":            locations,
		"AuthType":             c.GetString("auth_type"),
		"Role":                 c.GetString("role"),
		"ActivePage":           "orders",
	})
}

func (h *OrderHandler) GetOrdersStats(c *gin.Context) {
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
		case models.OrderDraft:
			draftCount = sc.Count
		case models.OrderPending:
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case models.OrderConfirmed:
			confirmedCount = sc.Count
		case models.OrderFulfilled:
			fulfilledCount = sc.Count
		case models.OrderCancelled:
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
		Order(`CASE WHEN status = 'pending' THEN 0 WHEN status = 'client_confirmed' THEN 1 WHEN status = 'confirmed' THEN 2 WHEN status = 'draft' THEN 3 WHEN status IN ('fulfilled','completed') THEN 4 WHEN status = 'cancelled' THEN 5 ELSE 6 END, created_at DESC`).
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

func (h *OrderHandler) GetOrdersStatsGrid(c *gin.Context) {
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
		case models.OrderDraft:
			draftCount = sc.Count
		case models.OrderPending:
			pendingCount = sc.Count
		case "client_confirmed":
			clientConfirmedCount = sc.Count
		case models.OrderConfirmed:
			confirmedCount = sc.Count
		case models.OrderFulfilled:
			fulfilledCount = sc.Count
		case models.OrderCancelled:
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

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
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

	sendOrderNotif(h.db, order, request.Status)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

// ClientCreateOrder allows customers to create orders
func (h *OrderHandler) ClientCreateOrder(c *gin.Context) {
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
		Status:      models.OrderPending,
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

	// Deduct stock in a transaction with row-level locking
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range itemList {
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&product, item.ProductID).Error; err != nil {
				return fmt.Errorf("product_not_found:%d", item.ProductID)
			}
			product.Stock -= item.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.InventoryLog{
				ProductID: product.ID,
				Type:      "out",
				Quantity:  item.Quantity,
				Reason:    fmt.Sprintf("Order #%s", order.OrderNumber),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process order"})
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
				strconv.Itoa(int(order.ID+10000)),
				"",
				"",
				"",
				"order",
				nil,
				order.CreatedAt,
				strconv.Itoa(int(request.BusinessID)),
				strconv.Itoa(int(client.ID)),
			)
			var bizUnread int64
			h.db.Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conv.ID, false).
				Count(&bizUnread)
			ws.BroadcastUnreadCount(h.hub, strconv.Itoa(int(conv.ID)), int32(bizUnread), strconv.Itoa(int(request.BusinessID)), "biz")

			var clientUnread int64
			h.db.Model(&models.Message{}).
				Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conv.ID, false).
				Count(&clientUnread)
			var biz models.Business
			h.db.First(&biz, request.BusinessID)
			bizCardSidebar := RenderBizSidebarCard(client, conv.ID, "", order.CreatedAt, int(bizUnread))
			clientCardSidebar := RenderClientSidebarCard(biz, conv.ID, "", order.CreatedAt, int(clientUnread))
			ws.BroadcastConversationUpdate(h.hub, strconv.Itoa(int(conv.ID)), bizCardSidebar, clientCardSidebar, strconv.Itoa(int(request.BusinessID)), strconv.Itoa(int(client.ID)))

			bizCardHTML := renderBizOrderCard(h.db, order)
			clientCardHTML := renderClientOrderCard(h.db, order)
			ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(request.BusinessID)), strconv.Itoa(int(client.ID)))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order":        order,
		"product_name": firstProductName,
		"quantity":     len(itemList),
	})
}

// UpdateOrder updates an existing order's items, notes, and delivery address
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
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

	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Restore stock from old items
		for _, oldItem := range order.OrderItems {
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&product, oldItem.ProductID).Error; err != nil {
				return err
			}
			product.Stock += oldItem.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
		}

		// Delete old order items
		if err := tx.Where("order_id = ?", order.ID).Delete(&models.OrderItem{}).Error; err != nil {
			return err
		}

		// Build new items and calculate total
		var totalAmount float64
		var newItems []models.OrderItem

		for _, item := range request.Items {
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND business_id = ?", item.ProductID, businessID).
				First(&product).Error; err != nil {
				return fmt.Errorf("product_not_found:%d", item.ProductID)
			}
			if product.Stock < item.Quantity {
				return fmt.Errorf("insufficient_stock:%s", product.Name)
			}
			itemTotal := float64(item.Quantity) * product.Price
			totalAmount += itemTotal
			newItems = append(newItems, models.OrderItem{
				OrderID:    order.ID,
				ProductID:  product.ID,
				Quantity:   item.Quantity,
				UnitPrice:  product.Price,
				TotalPrice: itemTotal,
			})

			product.Stock -= item.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
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

		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		for i := range newItems {
			if err := tx.Create(&newItems[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		errStr := err.Error()
		if len(errStr) > 18 && errStr[:18] == "insufficient_stock:" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for %s", errStr[18:])})
		} else if len(errStr) > 17 && errStr[:17] == "product_not_found:" {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %s not found", errStr[17:])})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

func generateOrderNumber() string {
	return fmt.Sprintf("ORD-%d", time.Now().Unix())
}

// GetConversationProducts returns all active products for the business in a conversation
func (h *OrderHandler) GetConversationProducts(c *gin.Context) {
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

func (h *OrderHandler) GetConversationServices(c *gin.Context) {
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
func (h *OrderHandler) CreateOrderDraft(c *gin.Context) {
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

	// Build and create order in a transaction with row-level locking
	var orderItems []models.OrderItem
	var order models.Order
	var firstProductName string
	var productNames []string

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var totalAmount float64
		productNames = nil
		firstProductName = ""
		orderItems = nil

		for _, item := range request.Items {
			var product models.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND business_id = ?", item.ProductID, businessID).
				First(&product).Error; err != nil {
				return fmt.Errorf("product_not_found:%d", item.ProductID)
			}
			if product.Stock < item.Quantity {
				return fmt.Errorf("insufficient_stock:%s", product.Name)
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

			product.Stock -= item.Quantity
			if err := tx.Save(&product).Error; err != nil {
				return err
			}
		}

		now := time.Now()
		fullNotes := request.Notes
		if request.DeliveryAddress != "" {
			fullNotes = "📍 Delivery Address: " + request.DeliveryAddress + "\n" + fullNotes
		}
		order = models.Order{
			BusinessID:  businessID,
			ClientID:    conversation.ClientID,
			OrderNumber: generateOrderNumber(),
			Status:      models.OrderDraft,
			Sender:      "business",
			Quantity:    len(request.Items),
			TotalAmount: totalAmount,
			Notes:       fullNotes,
			Draft:       true,
			CreatedAt:   now,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		for i := range orderItems {
			orderItems[i].OrderID = order.ID
		}
		if err := tx.Create(&orderItems).Error; err != nil {
			return err
		}

		for _, item := range request.Items {
			if err := tx.Create(&models.InventoryLog{
				ProductID: item.ProductID,
				Type:      "out",
				Quantity:  item.Quantity,
				Reason:    fmt.Sprintf("Draft order #%s", order.OrderNumber),
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		errStr := err.Error()
		if len(errStr) > 18 && errStr[:18] == "insufficient_stock:" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Insufficient stock for %s", errStr[18:])})
		} else if len(errStr) > 17 && errStr[:17] == "product_not_found:" {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Product %s not found", errStr[17:])})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		}
		return
	}

	progress.AutoCalculateProgress(conversation.ID)
	if h.hub != nil {
		ws.BroadcastNewMessage(
			h.hub,
			strconv.Itoa(int(conversation.ID)),
			strconv.Itoa(int(businessID)),
			"business",
			strconv.Itoa(int(order.ID+10000)),
			"",
			"",
			"",
			"order",
			nil,
			order.CreatedAt,
			strconv.Itoa(int(businessID)),
			strconv.Itoa(int(conversation.ClientID)),
		)
		var clientUnread int64
		h.db.Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conversation.ID, false).
			Count(&clientUnread)
		ws.BroadcastUnreadCount(h.hub, strconv.Itoa(int(conversation.ID)), int32(clientUnread), strconv.Itoa(int(conversation.ClientID)), "client")
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(conversation.ClientID)))
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
		models.OrderDraft:         true,
	})
}

// SendOrderToClient publishes a draft order to the client
func (h *OrderHandler) SendOrderToClient(c *gin.Context) {
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

	if order.Status != models.OrderDraft {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is not in draft status"})
		return
	}

	order.Status = models.OrderPending
	order.Draft = false
	now := time.Now()
	order.UpdatedAt = now
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send order"})
		return
	}

	sendOrderNotif(h.db, order, models.OrderPending)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
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
func (h *OrderHandler) ConfirmOrderBusiness(c *gin.Context) {
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

	if order.Status != models.OrderPending && order.Status != "client_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be confirmed in current status"})
		return
	}

	now := time.Now()
	order.ConfirmedByBusiness = true
	order.ConfirmedByBusinessAt = &now
	order.Status = models.OrderConfirmed
	order.UpdatedAt = now
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm order"})
		return
	}

	sendOrderNotif(h.db, order, models.OrderConfirmed)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order confirmed successfully",
	})
}

// RejectOrder cancels/rejects an order
func (h *OrderHandler) RejectOrder(c *gin.Context) {
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

	if order.Status == models.OrderConfirmed || order.Status == models.OrderFulfilled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot reject a confirmed/fulfilled order"})
		return
	}

	order.Status = models.OrderCancelled
	order.UpdatedAt = time.Now()
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject order"})
		return
	}

	sendOrderNotif(h.db, order, models.OrderCancelled)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order rejected/cancelled",
	})
}

// FulfillOrder transitions confirmed → fulfilled
func (h *OrderHandler) FulfillOrder(c *gin.Context) {
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

	if order.Status != models.OrderConfirmed && order.Status != "client_confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be confirmed before fulfillment"})
		return
	}

	now := time.Now()
	order.Status = models.OrderFulfilled
	order.UpdatedAt = now
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fulfill order"})
		return
	}

	sendOrderNotif(h.db, order, models.OrderCompleted)

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order fulfilled successfully",
	})
}

// GetOrderReceipt renders a print-friendly receipt for a completed order
func (h *OrderHandler) GetOrderReceipt(c *gin.Context) {
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

	if order.Status != models.OrderFulfilled && order.Status != models.OrderCompleted {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Order is not completed"})
		return
	}

	c.HTML(http.StatusOK, "receipt_order.html", gin.H{
		"Order":    order,
		"Business": order.Business,
	})
}

// MarkOrderAsPaid sets the order's paid amount to the total (quick mark as fully paid)
func (h *OrderHandler) MarkOrderAsPaid(c *gin.Context) {
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

	if order.Status != models.OrderConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must be confirmed before marking as paid"})
		return
	}

	order.PaidAmount = order.TotalAmount
	order.UpdatedAt = time.Now()
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark order as paid"})
		return
	}

	// Create a completed payment record
	if err := h.db.Create(&models.Payment{
		OrderID:   &order.ID,
		ClientID:  order.ClientID,
		Amount:    order.TotalAmount,
		Method:    "cash",
		Status:    models.OrderCompleted,
		Reference: "quick-paid",
		Notes:     "Marked as paid from dashboard",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment record"})
		return
	}

	sendOrderNotif(h.db, order, "paid")

	var conv models.Conversation
	if err := h.db.Where("client_id = ? AND business_id = ?", order.ClientID, businessID).First(&conv).Error; err == nil {
		progress.AutoCalculateProgress(conv.ID)
	}

	if h.hub != nil {
		bizCardHTML := renderBizOrderCard(h.db, order)
		clientCardHTML := renderClientOrderCard(h.db, order)
		ws.BroadcastOrderUpdateFull(h.hub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Order marked as paid",
	})
}

func sendOrderNotif(db *gorm.DB, order models.Order, status string) {
	prefs, err := notifier.GetOrCreatePrefs(db, order.BusinessID)
	if err != nil || !prefs.OrderStatusChange {
		return
	}
	var client models.Client
	if err := db.First(&client, order.ClientID).Error; err != nil || client.Email == "" {
		return
	}
	var biz models.Business
	if err := db.First(&biz, order.BusinessID).Error; err != nil {
		return
	}

	statusLabel := status
	notifType := "order_status"
	rid := order.ID
	if notifier.HasBeenSent(db, order.BusinessID, client.ID, notifType, &rid) {
		return
	}

	chatLink := services.AppURL(fmt.Sprintf("/client?business_id=%d", biz.ID))

	if err := services.SendOrderStatusEmail(client.Email, client.Name, biz.Name, order.OrderNumber, statusLabel, chatLink); err != nil {
		notifier.MarkNotificationSent(db, order.BusinessID, client.ID, notifType, "order", &rid, client.Email, "failed")
		return
	}
	notifier.MarkNotificationSent(db, order.BusinessID, client.ID, notifType, "order", &rid, client.Email, "sent")
	notifier.CreateInAppNotif(db, order.BusinessID, &client.ID,
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
	case models.OrderDraft:
		actionRequired = "none"
		editable = true
	case models.OrderPending:
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
	case models.OrderConfirmed:
		actionRequired = "none"
		editable = false
	case models.OrderFulfilled:
		actionRequired = "none"
		editable = false
	case models.OrderCancelled:
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
		"id":                 order.ID,
		"order_number":       order.OrderNumber,
		"status":             order.Status,
		"client_confirmed":   order.ConfirmedByClient,
		"business_confirmed": order.ConfirmedByBusiness,
		"action_required":    actionRequired,
		"editable":           editable,
		"sender":             order.Sender,
		models.OrderDraft:              order.Draft,
		"items":              items,
		"total_amount":       order.TotalAmount,
		"paid_amount":        order.PaidAmount,
		"remaining":          remaining,
		"is_fully_paid":      order.PaidAmount >= order.TotalAmount,
		"quantity":           order.Quantity,
		"notes":              order.Notes,
		"product_names":      productNames,
		"first_product_name": firstProductName,
		"created_at":         order.CreatedAt,
		"payment_methods":    paymentMethods,
	}
}
