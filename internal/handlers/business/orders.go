package business

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"oneflow/internal/data"
	"oneflow/internal/models"
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
		ClientID        uint   `json:"client_id" binding:"required"`
		ProductID       uint   `json:"product_id" binding:"required"`
		Quantity        int    `json:"quantity" binding:"required"`
		DeliveryAddress string `json:"delivery_address"`
		Notes           string `json:"notes"`
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

	// Create or get client
	var client models.Client
	if err := h.db.Where("id = ?", request.ClientID).First(&client).Error; err != nil {
		log.Println("Failed to create client", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client"})
		return
	}

	// Create order
	order := models.Order{
		BusinessID:   businessID,
		ClientID:     client.ID,
		Quantity:     request.Quantity,
		OrderNumber:  generateOrderNumber(),
		Status:       "pending",
		Sender:       "business",
		TotalAmount:  float64(request.Quantity) * product.Price,
		Notes:        fmt.Sprintf("Delivery: %s. %s", request.DeliveryAddress, request.Notes),
		DeliveryDate: &[]time.Time{time.Now().AddDate(0, 0, 7)}[0],
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

	var orders []models.Order
	h.db.Where("business_id = ?", businessID).Find(&orders)

	var pendingCount, confirmedCount, completedCount, canceledCount int64
	var totalRevenue float64

	for _, order := range orders {
		switch order.Status {
		case "pending":
			pendingCount++
		case "confirmed":
			confirmedCount++
		case "completed":
			completedCount++
		case "canceled":
			canceledCount++
		}
		totalRevenue += order.TotalAmount
	}

	c.HTML(http.StatusOK, "orders.html", gin.H{
		"Business":       currentBusiness,
		"Orders":         orders,
		"PendingCount":   pendingCount,
		"ConfirmedCount": confirmedCount,
		"CompletedCount": completedCount,
		"CanceledCount":  canceledCount,
		"TotalOrders":    len(orders),
		"TotalRevenue":   totalRevenue,
		"ActivePage":     "orders",
		"Countries":      data.Countries,
		"Currencies":     data.Currencies,
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
	if err := h.db.Where("id = ? AND business_id = ?", id, businessID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	order.Status = request.Status
	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
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
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
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
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
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
	if err := h.db.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order rejected/cancelled",
	})
}

// buildOrderData creates the rich order data map for templates
func buildOrderData(order models.Order, orderItems []models.OrderItem, productNames []string, firstProductName string) map[string]interface{} {
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

	return map[string]interface{}{
		"id":                 order.ID,
		"order_number":       order.OrderNumber,
		"status":             order.Status,
		"client_confirmed":   order.ConfirmedByClient,
		"business_confirmed": order.ConfirmedByBusiness,
		"action_required":    actionRequired,
		"editable":           editable,
		"sender":             order.Sender,
		"draft":              order.Draft,
		"items":              items,
		"total_amount":       order.TotalAmount,
		"quantity":           order.Quantity,
		"notes":              order.Notes,
		"product_names":      productNames,
		"first_product_name": firstProductName,
		"created_at":         order.CreatedAt,
	}
}
