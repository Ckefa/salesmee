package handlers

import (
	"encoding/json"
	"net/http"
	"log"
	"sort"
	"strconv"
	"time"

	"salesmee/internal/db"
	businessh "salesmee/internal/handlers/business"
	clienth "salesmee/internal/handlers/client"
	"salesmee/internal/models"
	"salesmee/internal/services/media"
	"salesmee/internal/services/subscription"
	prog "salesmee/internal/services/progress"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var wsHub *ws.Hub

func SetWSHub(hub *ws.Hub) {
	wsHub = hub
}

func dbc(c *gin.Context) *gorm.DB {
	return db.DB.WithContext(c.Request.Context())
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	switch {
	case messageID >= 20000:
		// Booking card — set HiddenFromChat
		bookingID := messageID - 20000
		var booking models.Booking
		if err := dbc(c).Where("id = ? AND business_id = ?", bookingID, businessID).First(&booking).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		dbc(c).Model(&booking).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastBookingUpdate(wsHub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(booking.ClientID)))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "booking", "id": bookingID})

	case messageID >= 10000:
		// Order card — set HiddenFromChat
		orderID := messageID - 10000
		var order models.Order
		if err := dbc(c).Where("id = ? AND business_id = ?", orderID, businessID).First(&order).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		dbc(c).Model(&order).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastOrderUpdate(wsHub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(businessID)), strconv.Itoa(int(order.ClientID)))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "order", "id": orderID})

	default:
		// Regular message — hard delete, verify ownership via conversation
		var msg models.Message
		if err := dbc(c).Preload("Conversation").First(&msg, messageID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		if msg.Conversation.BusinessID != businessID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		dbc(c).Delete(&msg)
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "message", "id": messageID})
	}
}

func GetMessages(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)

	var business models.Business
	dbc(c).First(&business, businessID)
	if err != nil {
		log.Println("GetMessages: =>> Invalid customer ID")
		c.String(http.StatusBadRequest, "Invalid customer ID")
		return
	}

	// Get conversation (implicitly verifies client+business relationship)
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		log.Println("GetMessages: =>> Conversation not found", clientID, businessID)
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Load client
	var client models.Client
	if err := dbc(c).First(&client, clientID).Error; err != nil {
		c.String(http.StatusNotFound, "Customer not found")
		return
	}

	// Add conversation ID to client struct for template use
	client.ConversationID = conversation.ID

	// Load conversation progress
	var progress models.ConversationProgress
	if err := dbc(c).Where("conversation_id = ?", conversation.ID).First(&progress).Error; err != nil {
		// Create default progress if not exists
		progress = models.ConversationProgress{
			ConversationID: conversation.ID,
			CurrentStage:   models.StageInitial,
			ProgressScore:  10,
		}
		if err := dbc(c).Create(&progress).Error; err != nil {
			log.Println("GetMessages: =>> Failed to Crete conversation progress", clientID, businessID)
			c.String(http.StatusInternalServerError, "Failed to create conversation progress")
			return
		}
	}

	// Pagination params
	before := c.Query("before")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Convert messages to MessageObj
	var messageObjs []MessageObj
	var messages []models.Message
	msgQuery := dbc(c).Where("conversation_id = ?", conversation.ID).Order("created_at DESC")
	if before != "" {
		if beforeTime, err := time.Parse(time.RFC3339, before); err == nil {
			msgQuery = msgQuery.Where("created_at < ?", beforeTime)
		}
	}
	msgQuery.Limit(limit + 1).Find(&messages)

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	// Reverse to ASC for rendering
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	var olderCursor string
	if hasMore && len(messages) > 0 {
		olderCursor = messages[0].CreatedAt.Format(time.RFC3339Nano)
	}

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
	dbc(c).Where("client_id = ? AND business_id = ? AND hidden_from_chat = ?", client.ID, businessID, false).Order("created_at ASC").Find(&orders)
	for _, order := range orders {
		var orderItems []models.OrderItem
		dbc(c).Where("order_id = ?", order.ID).Preload("Product").Find(&orderItems)

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
		case models.OrderDraft:
			actionRequired = "none"
			editable = true
		case models.OrderPending:
			if order.Sender == "business" && !order.ConfirmedByClient {
				actionRequired = "client"
			} else if order.Sender == "client" && !order.ConfirmedByBusiness {
				actionRequired = "business"
			} else {
				actionRequired = "none"
			}
		case models.OrderClientConfirmed:
			actionRequired = "business"
		case models.OrderConfirmed:
			actionRequired = "none"
		case models.OrderFulfilled:
			actionRequired = "none"
		case models.OrderCancelled:
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
		dbc(c).Where("business_id = ? AND is_active = ?", order.BusinessID, true).Order("sort_order ASC, id ASC").Find(&orderPaymentMethods)

		var orderPendingAmt float64
		dbc(c).Model(&models.Payment{}).Where("order_id = ? AND status = ?", order.ID, models.OrderPending).
			Select("COALESCE(SUM(amount), 0)").Scan(&orderPendingAmt)

		var orderPayments []models.Payment
		dbc(c).Where("order_id = ?", order.ID).Order("created_at desc").Find(&orderPayments)

		var orderReview struct {
			Rating int
			Title  string
		}
		var orderHasReview bool
		dbc(c).Model(&models.Review{}).Where("order_id = ?", order.ID).Select("rating, title").Scan(&orderReview)
		if orderReview.Rating > 0 {
			orderHasReview = true
		}

		orderData := map[string]interface{}{
			"id":                   order.ID,
			"order_number":         order.OrderNumber,
			"status":               order.Status,
			models.OrderClientConfirmed:     order.ConfirmedByClient,
			"business_confirmed":   order.ConfirmedByBusiness,
			"action_required":      actionRequired,
			"editable":             editable,
			"sender":               order.Sender,
			models.OrderDraft:                order.Draft,
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
			"payments":             orderPayments,
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
	dbc(c).Where("client_id = ? AND business_id = ? AND hidden_from_chat = ?", client.ID, businessID, false).Order("created_at ASC").Find(&bookings)
	for _, booking := range bookings {
		var serviceName string
		var serviceNames []string
		var bookingItems []models.BookingItem
		dbc(c).Where("booking_id = ?", booking.ID).Find(&bookingItems)

		var firstServiceID uint
		for _, item := range bookingItems {
			var service models.Service
			if err := dbc(c).First(&service, item.ServiceID).Error; err == nil {
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
		dbc(c).Where("business_id = ? AND is_active = ?", booking.BusinessID, true).Order("sort_order ASC, id ASC").Find(&bookingPaymentMethods)

		var bookingPendingAmt float64
		dbc(c).Model(&models.Payment{}).Where("booking_id = ? AND status = ?", booking.ID, models.OrderPending).
			Select("COALESCE(SUM(amount), 0)").Scan(&bookingPendingAmt)

		var bookingPayments []models.Payment
		dbc(c).Where("booking_id = ?", booking.ID).Order("created_at desc").Find(&bookingPayments)

		var bookingActionRequired string
		if booking.Status == models.OrderPending && booking.Sender == "client" {
			bookingActionRequired = "business"
		} else if booking.Status == models.OrderClientConfirmed && !(booking.PaidAmount >= booking.TotalAmount) {
			bookingActionRequired = "business"
		} else {
			bookingActionRequired = "none"
		}

		var bookingReview struct {
			Rating int
			Title  string
		}
		var bookingHasReview bool
		dbc(c).Model(&models.Review{}).Where("booking_id = ?", booking.ID).Select("rating, title").Scan(&bookingReview)
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
			"payments":             bookingPayments,
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
	sort.Slice(messageObjs, func(i, j int) bool {
		return messageObjs[i].CreatedAt.Before(messageObjs[j].CreatedAt)
	})

	var insight models.CustomerInsight
	if err := dbc(c).Where("conversation_id = ?", conversation.ID).First(&insight).Error; err != nil {
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

	c.HTML(http.StatusOK, "business_chat.html", gin.H{
		"Customer":   client,
		"Messages":   messageObjs,
		"Progress":   progress,
		"Business":   business,
		"Insight":    insight,
		"HasMore":    hasMore,
		"OlderCursor": olderCursor,
	})
}

func CreateMessage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid customer ID")
		return
	}

	// Get conversation (implicitly verifies client+business relationship)
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.String(http.StatusNotFound, "Conversation not found")
		return
	}

	// Load client
	var client models.Client
	if err := dbc(c).First(&client, clientID).Error; err != nil {
		c.String(http.StatusNotFound, "Customer not found")
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
	hasMediaFeature := subscription.HasFeature(businessID, "media_sharing")
	hasUploadedFile := false
	if hasMediaFeature {
		for _, field := range []string{"media_image", "media_document", "media_audio"} {
			mediaURL, mediaType, err := media.SaveMediaFile(c, field)
			if err == nil {
				message.MediaURL = mediaURL
				message.MediaType = mediaType
				hasUploadedFile = true
				break
			}
		}
	} else {
		for _, field := range []string{"media_image", "media_document", "media_audio"} {
			_, err := c.FormFile(field)
			if err == nil {
				hasUploadedFile = true
				break
			}
		}
	}

	if !hasMediaFeature && hasUploadedFile {
		msg := "Media sharing is not available on your " + subscription.PlanDisplayName(businessID) + " plan. Upgrade to Diamond to send images, files and audio."
		if c.GetHeader("HX-Request") == "true" {
			triggerData, _ := json.Marshal(map[string]interface{}{
				"show-upgrade-modal": map[string]interface{}{
					"message":    msg,
					"upgradeUrl": "/business/subscription#plans",
				},
			})
			c.Header("HX-Trigger", string(triggerData))
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error":            msg,
			"requires_upgrade": true,
			"upgrade_url":      "/business/subscription#plans",
		})
		return
	}

	if content == "" && message.MediaURL == "" {
		c.String(http.StatusBadRequest, "Message cannot be empty")
		return
	}

	if err := dbc(c).Create(&message).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create message")
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
		dbc(c).Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'business' AND read_by_client = ?", conversation.ID, false).
			Count(&clientUnread)

		var bizUnread int64
		dbc(c).Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conversation.ID, false).
			Count(&bizUnread)

		var business models.Business
		dbc(c).First(&business, businessID)

		bizCard := businessh.RenderBizSidebarCard(client, conversation.ID, message.Content, message.CreatedAt, int(bizUnread))
		clientCard := clienth.RenderClientSidebarCard(business, conversation.ID, message.Content, message.CreatedAt, int(clientUnread))
		ws.BroadcastConversationUpdate(wsHub, strconv.Itoa(int(conversation.ID)), bizCard, clientCard, strconv.Itoa(int(businessID)), strconv.FormatUint(clientID, 10))

		ws.BroadcastUnreadCount(wsHub, strconv.Itoa(int(conversation.ID)), int32(clientUnread), strconv.FormatUint(clientID, 10), "client")
	}

	// Return message partial
	c.HTML(http.StatusOK, "message_partial.html", gin.H{
		"Message": message,
	})
}

func UpdateMessage(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	var request struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var message models.Message
	if err := dbc(c).First(&message, messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	message.Content = request.Content
	if err := dbc(c).Save(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func MarkConversationAsRead(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	// Get conversation for broadcast
	var conversation models.Conversation
	if err := dbc(c).Where("business_id = ? AND client_id = ?", businessID, clientID).First(&conversation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	// Update conversation's last read time
	now := time.Now()
	if err := dbc(c).Model(&conversation).Update("last_read_by_business_at", &now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark conversation as read"})
		return
	}

	// Also mark all unread messages as read by business
	if err := dbc(c).Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE business_id = ? AND client_id = ?) AND sender = 'client' AND read_by_business = ?", businessID, clientID, false).
		Updates(map[string]interface{}{
			"read_by_business": true,
			"read_at":          &now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark messages as read"})
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

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func MarkMessageAsRead(c *gin.Context) {
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}
	businessID := c.GetUint("business_id")

	// Only regular messages (not synthetic order/booking cards)
	if messageID >= 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot mark order/booking cards as read"})
		return
	}

	var msg models.Message
	if err := dbc(c).Preload("Conversation").First(&msg, messageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	if msg.Conversation.BusinessID != businessID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
		return
	}

	now := time.Now()
	dbc(c).Model(&msg).Updates(map[string]interface{}{
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

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ClearChat(c *gin.Context) {
	businessID := c.GetUint("business_id")
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	now := time.Now()
	if err := dbc(c).Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE business_id = ? AND client_id = ?)", businessID, clientID).
		Delete(&models.Message{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear chat"})
		return
	}

	dbc(c).Model(&models.Conversation{}).
		Where("business_id = ? AND client_id = ?", businessID, clientID).
		Updates(map[string]interface{}{
			"last_read_by_business_at": &now,
		})

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func MarkClientConversationAsRead(c *gin.Context) {
	clientID := c.GetUint("client_id")
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	// Get conversation for broadcast
	var conversation models.Conversation
	if err := dbc(c).Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	now := time.Now()
	if err := dbc(c).Model(&conversation).Update("last_read_by_client_at", &now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark conversation as read"})
		return
	}

	// Also mark all unread business-sent messages as read by client
	if err := dbc(c).Model(&models.Message{}).
		Where("conversation_id IN (SELECT id FROM conversations WHERE client_id = ? AND business_id = ?) AND sender = 'business' AND read_by_client = ?", clientID, businessID, false).
		Updates(map[string]interface{}{
			"read_by_client":    true,
			"read_by_client_at": &now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark messages as read"})
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

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
