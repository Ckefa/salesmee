package handlers

import (
	"log"
	"strconv"
	"time"

	"salesmee/internal/db"
	businessh "salesmee/internal/handlers/business"
	clienth "salesmee/internal/handlers/client"
	"salesmee/internal/models"
	"salesmee/internal/services/media"
	prog "salesmee/internal/services/progress"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
)

var wsHub *ws.Hub

func SetWSHub(hub *ws.Hub) {
	wsHub = hub
}

type MessageObj struct {
	ID          uint        `json:"id"`
	MsgType     string      `json:"msgtype"` // "message", "order", "booking"
	Value       string      `json:"value"`   // string content for normal messages, empty for orders/bookings
	Data        interface{} `json:"data"`    // order object or booking object as JSON, null for normal messages
	Sender      string      `json:"sender"`
	MediaURL    string      `json:"media_url"`
	MediaType   string      `json:"media_type"`
	CreatedAt   time.Time   `json:"created_at"`
	IsDelivered bool        `json:"is_delivered"`
	IsRead      bool        `json:"is_read"`
}



func DeleteMessage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	messageIDStr := c.Param("message_id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid message ID"})
		return
	}

	switch {
	case messageID >= 20000:
		// Booking card — set HiddenFromChat
		bookingID := messageID - 20000
		var booking models.Booking
		if err := db.DB.Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
			c.JSON(404, gin.H{"error": "Booking not found"})
			return
		}
		db.DB.Model(&booking).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastBookingUpdate(wsHub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
		}
		c.JSON(200, gin.H{"success": true, "type": "booking", "id": bookingID})

	case messageID >= 10000:
		// Order card — set HiddenFromChat
		orderID := messageID - 10000
		var order models.Order
		if err := db.DB.Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
			c.JSON(404, gin.H{"error": "Order not found"})
			return
		}
		db.DB.Model(&order).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastOrderUpdate(wsHub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
		}
		c.JSON(200, gin.H{"success": true, "type": "order", "id": orderID})

	default:
		// Regular message — hard delete, verify ownership via conversation
		var msg models.Message
		if err := db.DB.Preload("Conversation").First(&msg, messageID).Error; err != nil {
			c.JSON(404, gin.H{"error": "Message not found"})
			return
		}
		if msg.Conversation.BusinessID != businessID {
			c.JSON(403, gin.H{"error": "Unauthorized"})
			return
		}
		db.DB.Delete(&msg)
		c.JSON(200, gin.H{"success": true, "type": "message", "id": messageID})
	}
}

func GetMessages(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)

	var business models.Business
	db.DB.First(&business, businessID)
	if err != nil {
		log.Println("GetMessages: =>> Invalid customer ID")
		c.String(400, "Invalid customer ID")
		return
	}

	// Get conversation (implicitly verifies client+business relationship)
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		log.Println("GetMessages: =>> Conversation not found", clientID, businessID)
		c.String(404, "Conversation not found")
		return
	}

	// Load client
	var client models.Client
	if err := db.DB.First(&client, clientID).Error; err != nil {
		c.String(404, "Customer not found")
		return
	}

	// Add conversation ID to client struct for template use
	client.ConversationID = conversation.ID

	// Load conversation progress
	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", conversation.ID).First(&progress).Error; err != nil {
		// Create default progress if not exists
		progress = models.ConversationProgress{
			ConversationID: conversation.ID,
			CurrentStage:   models.StageInitial,
			ProgressScore:  10,
		}
		if err := db.DB.Create(&progress).Error; err != nil {
			log.Println("GetMessages: =>> Failed to Crete conversation progress", clientID, businessID)
			c.String(500, "Failed to create conversation progress")
			return
		}
	}

	// Convert messages to MessageObj
	var messageObjs []MessageObj
	var messages []models.Message
	db.DB.Where("conversation_id = ?", conversation.ID).Order("created_at ASC").Find(&messages)

	for _, msg := range messages {
		isSelf := msg.Sender == "business"
		var isDelivered, isRead bool
		if isSelf {
			isDelivered = msg.DeliveredAt != nil
			isRead = msg.ReadByClient
		}
		messageObj := MessageObj{
			ID:          msg.ID,
			MsgType:     "message",
			Value:       msg.Content,
			Data:        msg,
			Sender:      msg.Sender,
			MediaURL:    msg.MediaURL,
			MediaType:   msg.MediaType,
			CreatedAt:   msg.CreatedAt,
			IsDelivered: isDelivered,
			IsRead:      isRead,
		}
		messageObjs = append(messageObjs, messageObj)
	}

	// Fetch orders
	var orders []models.Order
	db.DB.Where("client_id = ? AND business_id = ? AND hidden_from_chat = ?", client.ID, businessID, false).Order("created_at ASC").Find(&orders)
	for _, order := range orders {
		var orderItems []models.OrderItem
		db.DB.Where("order_id = ?", order.ID).Preload("Product").Find(&orderItems)

		var productNames []string
		var firstProductName string
		for _, item := range orderItems {
			if firstProductName == "" {
				firstProductName = item.Product.Name
			}
			productNames = append(productNames, item.Product.Name)
		}

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
			} else if order.Sender == "client" && !order.ConfirmedByBusiness {
				actionRequired = "business"
			} else {
				actionRequired = "none"
			}
		case "client_confirmed":
			actionRequired = "business"
		case "confirmed":
			actionRequired = "none"
		case "fulfilled":
			actionRequired = "none"
		case "cancelled":
			actionRequired = "none"
		default:
			actionRequired = "none"
		}

		remaining := order.TotalAmount - order.PaidAmount
		if remaining < 0 {
			remaining = 0
		}

		// Get payment methods from business
		var orderPaymentMethods []models.PaymentMethod
		db.DB.Where("business_id = ? AND is_active = ?", order.BusinessID, true).Order("sort_order ASC, id ASC").Find(&orderPaymentMethods)

		var orderPendingAmt float64
		db.DB.Model(&models.Payment{}).Where("order_id = ? AND status = ?", order.ID, "pending").
			Select("COALESCE(SUM(amount), 0)").Scan(&orderPendingAmt)

		var orderReview struct {
			Rating int
			Title  string
		}
		var orderHasReview bool
		db.DB.Model(&models.Review{}).Where("order_id = ?", order.ID).Select("rating, title").Scan(&orderReview)
		if orderReview.Rating > 0 {
			orderHasReview = true
		}

		orderData := map[string]interface{}{
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
			"pending_amount":       orderPendingAmt,
			"remaining":            remaining,
			"is_fully_paid":        order.PaidAmount >= order.TotalAmount,
			"has_review":           orderHasReview,
			"review_rating":        orderReview.Rating,
			"quantity":             order.Quantity,
			"notes":                order.Notes,
			"currency":             business.Currency,
			"product_names":        productNames,
			"first_product_name":   firstProductName,
			"created_at":           order.CreatedAt,
			"payment_methods":      orderPaymentMethods,
		}

		isSelf := order.Sender == "business"
		isDelivered := isSelf && conversation.LastReadByClientAt != nil && order.CreatedAt.Before(*conversation.LastReadByClientAt)

		messageObjs = append(messageObjs, MessageObj{
			ID:          order.ID + 10000,
			MsgType:     "order",
			Value:       "",
			Data:        orderData,
			Sender:      order.Sender,
			CreatedAt:   order.CreatedAt,
			IsDelivered: isDelivered,
			IsRead:      false,
		})
	}

	// Fetch bookings
	var bookings []models.Booking
	db.DB.Where("client_id = ? AND business_id = ? AND hidden_from_chat = ?", client.ID, businessID, false).Order("created_at ASC").Find(&bookings)
	for _, booking := range bookings {
		var serviceName string
		var serviceNames []string
		var bookingItems []models.BookingItem
		db.DB.Where("booking_id = ?", booking.ID).Find(&bookingItems)

		var firstServiceID uint
		for _, item := range bookingItems {
			var service models.Service
			if err := db.DB.First(&service, item.ServiceID).Error; err == nil {
				serviceName = service.Name
				serviceNames = append(serviceNames, service.Name)
				if firstServiceID == 0 {
					firstServiceID = item.ServiceID
				}
			}
		}

		bookingRemaining := booking.TotalAmount - booking.PaidAmount
		if bookingRemaining < 0 {
			bookingRemaining = 0
		}

		// Get payment methods from business
		var bookingPaymentMethods []models.PaymentMethod
		db.DB.Where("business_id = ? AND is_active = ?", booking.BusinessID, true).Order("sort_order ASC, id ASC").Find(&bookingPaymentMethods)

		var bookingPendingAmt float64
		db.DB.Model(&models.Payment{}).Where("booking_id = ? AND status = ?", booking.ID, "pending").
			Select("COALESCE(SUM(amount), 0)").Scan(&bookingPendingAmt)

		var bookingActionRequired string
		if booking.Status == "pending" && booking.Sender == "client" {
			bookingActionRequired = "business"
		} else if booking.Status == "client_confirmed" && !(booking.PaidAmount >= booking.TotalAmount) {
			bookingActionRequired = "business"
		} else {
			bookingActionRequired = "none"
		}

		var bookingReview struct {
			Rating int
			Title  string
		}
		var bookingHasReview bool
		db.DB.Model(&models.Review{}).Where("booking_id = ?", booking.ID).Select("rating, title").Scan(&bookingReview)
		if bookingReview.Rating > 0 {
			bookingHasReview = true
		}

		bookingData := map[string]interface{}{
			"id":                   booking.ID,
			"booking_number":       booking.BookingNumber,
			"service_id":           firstServiceID,
			"service_name":         serviceName,
			"service_names":        serviceNames,
			"scheduled_date":       booking.ScheduledDate,
			"duration":             booking.Duration,
			"total_amount":         booking.TotalAmount,
			"paid_amount":          booking.PaidAmount,
			"pending_amount":       bookingPendingAmt,
			"remaining":            bookingRemaining,
			"is_fully_paid":        booking.PaidAmount >= booking.TotalAmount,
			"has_review":           bookingHasReview,
			"review_rating":        bookingReview.Rating,
			"notes":                booking.Notes,
			"status":               booking.Status,
			"sender":               booking.Sender,
			"action_required":      bookingActionRequired,
			"currency":             business.Currency,
			"created_at":           booking.CreatedAt,
			"payment_methods":      bookingPaymentMethods,
		}

		isSelf := booking.Sender == "business"
		isDelivered := isSelf && conversation.LastReadByClientAt != nil && booking.CreatedAt.Before(*conversation.LastReadByClientAt)

		messageObjs = append(messageObjs, MessageObj{
			ID:          booking.ID + 20000,
			MsgType:     "booking",
			Value:       "",
			Data:        bookingData,
			Sender:      booking.Sender,
			CreatedAt:   booking.CreatedAt,
			IsDelivered: isDelivered,
			IsRead:      false,
		})
	}

	// Sort messageObjs by CreatedAt
	for i := 0; i < len(messageObjs); i++ {
		for j := i + 1; j < len(messageObjs); j++ {
			if messageObjs[i].CreatedAt.After(messageObjs[j].CreatedAt) {
				messageObjs[i], messageObjs[j] = messageObjs[j], messageObjs[i]
			}
		}
	}

	var insight models.CustomerInsight
	if err := db.DB.Where("conversation_id = ?", conversation.ID).First(&insight).Error; err != nil {
		insight = models.CustomerInsight{
			ConversationID: conversation.ID,
			Tier:           "bronze",
			TierScore:      0,
			ActivityScore:  0,
			EngagementScore: 0,
			BehaviorTrend:  "inactive",
			TotalSpent:     0,
		}
	}

	c.HTML(200, "business_chat.html", gin.H{
		"Customer": client,
		"Messages": messageObjs,
		"Progress": progress,
		"Business": business,
		"Insight":  insight,
	})
}

func CreateMessage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(400, "Invalid customer ID")
		return
	}

	// Get conversation (implicitly verifies client+business relationship)
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.String(404, "Conversation not found")
		return
	}

	// Load client
	var client models.Client
	if err := db.DB.First(&client, clientID).Error; err != nil {
		c.String(404, "Customer not found")
		return
	}

	content := c.PostForm("content")
	sender := c.PostForm("sender") // "user" or "client"

	message := models.Message{
		ConversationID: conversation.ID,
		Content:        content,
		Sender:         sender,
	}

	// Handle media upload
	for _, field := range []string{"media_image", "media_document", "media_audio"} {
		mediaURL, mediaType, err := media.SaveMediaFile(c, field)
		if err == nil {
			message.MediaURL = mediaURL
			message.MediaType = mediaType
			break
		}
	}

	if err := db.DB.Create(&message).Error; err != nil {
		c.String(500, "Failed to create message")
		return
	}

	prog.AutoCalculateProgress(conversation.ID)

	if wsHub != nil {
		ws.BroadcastNewMessage(
			wsHub,
			strconv.Itoa(int(conversation.ID)),
			strconv.Itoa(int(businessID)),
			"business",
			strconv.Itoa(int(message.ID)),
			message.Content,
			message.MediaURL,
			message.MediaType,
			message.Type,
			nil,
			message.CreatedAt,
			strconv.Itoa(int(businessID)),
			strconv.FormatUint(clientID, 10),
		)

		var clientUnread int64
		db.DB.Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conversation.ID, false).
			Count(&clientUnread)

		var bizUnread int64
		db.DB.Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conversation.ID, false).
			Count(&bizUnread)

		var business models.Business
		db.DB.First(&business, businessID)

		bizCard := businessh.RenderBizSidebarCard(client, conversation.ID, message.Content, message.CreatedAt, int(bizUnread))
		clientCard := clienth.RenderClientSidebarCard(business, conversation.ID, message.Content, message.CreatedAt, int(clientUnread))
		ws.BroadcastConversationUpdate(wsHub, strconv.Itoa(int(conversation.ID)), bizCard, clientCard, strconv.Itoa(int(businessID)), strconv.FormatUint(clientID, 10))

		ws.BroadcastUnreadCount(wsHub, strconv.Itoa(int(conversation.ID)), int32(clientUnread), strconv.FormatUint(clientID, 10), "client")
	}

	// Return message partial
	c.HTML(200, "message_partial.html", gin.H{
		"Message": message,
	})
}

func UpdateMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid message ID"})
		return
	}

	var request struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var message models.Message
	if err := db.DB.First(&message, messageID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	message.Content = request.Content
	if err := db.DB.Save(&message).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to update message"})
		return
	}

	c.JSON(200, gin.H{"success": true, "message": message})
}

func MarkConversationAsRead(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid client ID"})
		return
	}

	// Get conversation for broadcast
	var conversation models.Conversation
	if err := db.DB.Where("business_id = ? AND client_id = ?", businessID, clientID).First(&conversation).Error; err != nil {
		c.JSON(404, gin.H{"error": "Conversation not found"})
		return
	}

	// Update conversation's last read time
	now := time.Now()
	if err := db.DB.Model(&conversation).Update("last_read_by_business_at", &now).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark conversation as read"})
		return
	}

	// Also mark all unread messages as read by business
	if err := db.DB.Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE business_id = ? AND client_id = ?) AND sender = 'client' AND read_by_business = ?", businessID, clientID, false).
		Updates(map[string]interface{}{
			"read_by_business": true,
			"read_at":          &now,
		}).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark messages as read"})
		return
	}

	if wsHub != nil {
		ws.BroadcastReadReceipt(
			wsHub,
			strconv.Itoa(int(conversation.ID)),
			strconv.Itoa(int(businessID)),
			"business",
			"",
			strconv.Itoa(int(businessID)),
			strconv.FormatUint(clientID, 10),
		)

		// After business reads, business unread count is 0
		ws.BroadcastUnreadCount(wsHub, strconv.Itoa(int(conversation.ID)), 0, strconv.Itoa(int(businessID)), "biz")
	}

	c.JSON(200, gin.H{"status": "ok"})
}

func MarkMessageAsRead(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid message ID"})
		return
	}
	businessID := c.GetUint("business_id")

	// Only regular messages (not synthetic order/booking cards)
	if messageID >= 10000 {
		c.JSON(400, gin.H{"error": "Cannot mark order/booking cards as read"})
		return
	}

	var msg models.Message
	if err := db.DB.Preload("Conversation").First(&msg, messageID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	if msg.Conversation.BusinessID != businessID {
		c.JSON(403, gin.H{"error": "Unauthorized"})
		return
	}

	now := time.Now()
	db.DB.Model(&msg).Updates(map[string]interface{}{
		"read_by_business": true,
		"read_at":          &now,
	})

	if wsHub != nil {
		ws.BroadcastReadReceipt(
			wsHub,
			strconv.Itoa(int(msg.ConversationID)),
			strconv.Itoa(int(businessID)),
			"business",
			strconv.FormatUint(messageID, 10),
			strconv.Itoa(int(businessID)),
			strconv.Itoa(int(msg.Conversation.ClientID)),
		)
	}

	c.JSON(200, gin.H{"success": true})
}

func ClearChat(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid client ID"})
		return
	}

	now := time.Now()
	if err := db.DB.Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE business_id = ? AND client_id = ?)", businessID, clientID).
		Delete(&models.Message{}).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to clear chat"})
		return
	}

	db.DB.Model(&models.Conversation{}).
		Where("business_id = ? AND client_id = ?", businessID, clientID).
		Updates(map[string]interface{}{
			"last_read_by_business_at": &now,
		})

	c.JSON(200, gin.H{"success": true})
}

func MarkClientConversationAsRead(c *gin.Context) {
	clientID := c.GetUint("client_id")
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid business ID"})
		return
	}

	// Get conversation for broadcast
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.JSON(404, gin.H{"error": "Conversation not found"})
		return
	}

	now := time.Now()
	if err := db.DB.Model(&conversation).Update("last_read_by_client_at", &now).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark conversation as read"})
		return
	}

	// Also mark all unread business-sent messages as read by client
	if err := db.DB.Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE client_id = ? AND business_id = ?) AND sender = 'business' AND read_by_client = ?", clientID, businessID, false).
		Updates(map[string]interface{}{
			"read_by_client":    true,
			"read_by_client_at": &now,
		}).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to mark messages as read"})
		return
	}

	if wsHub != nil {
		ws.BroadcastReadReceipt(
			wsHub,
			strconv.Itoa(int(conversation.ID)),
			strconv.Itoa(int(clientID)),
			"client",
			"",
			strconv.FormatUint(businessID, 10),
			strconv.Itoa(int(clientID)),
		)

		// After client reads, client unread count is 0
		ws.BroadcastUnreadCount(wsHub, strconv.Itoa(int(conversation.ID)), 0, strconv.Itoa(int(clientID)), "client")
	}

	c.JSON(200, gin.H{"status": "ok"})
}
