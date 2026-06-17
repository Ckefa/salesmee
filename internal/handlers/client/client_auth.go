package client

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/services/media"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/assist"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
)

var wsHub *ws.Hub

func SetWSHub(hub *ws.Hub) {
	wsHub = hub
}

func ShowClientLogin(c *gin.Context) {
	if token, err := c.Cookie("client_token"); err == nil && token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		if claims, err := services.ValidateToken(token); err == nil && claims.Subject == "client" {
			// Already authed — redirect to saved destination or /client
			redirect, _ := c.Cookie("client_redirect")
			c.SetCookie("client_redirect", "", -1, "/client", "", false, true)
			c.Redirect(http.StatusFound, safeClientOAuthRedirect(redirect))
			return
		}
	}
	c.HTML(http.StatusOK, "client_login.html", middleware.TemplateData(c, gin.H{
		"Title": "Client Login - SalesMee",
	}))
}

func SendClientOTP(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Email is required",
		}))
		return
	}

	// Try to find existing client by email (any business)
	var client models.Client
	err := db.DB.Where("email = ?", email).First(&client).Error
	if err != nil {
		// Client not found — create standalone (no business association)
		client = models.Client{
			Email:  email,
			Name:   email,
			Status: models.StatusNew,
		}
		if err := db.DB.Create(&client).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
				"Title": "Client Login - SalesMee",
				"Error": "Failed to create account",
			}))
			return
		}
	}

	_, err = services.SendClientOTP(email)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to send OTP",
		}))
		return
	}

	c.HTML(http.StatusOK, "client_otp.html", middleware.TemplateData(c, gin.H{
		"Title": "Enter OTP - SalesMee",
		"Email": email,
	}))
}

func VerifyClientOTP(c *gin.Context) {
	email := c.PostForm("email")
	otpCode := c.PostForm("otp")

	if email == "" || otpCode == "" {
		c.HTML(http.StatusBadRequest, "client_otp.html", middleware.TemplateData(c, gin.H{
			"Title": "Enter OTP - SalesMee",
			"Email": email,
			"Error": "Email and OTP are required",
		}))
		return
	}

	clientAuth, err := services.VerifyClientOTP(email, otpCode)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client_otp.html", middleware.TemplateData(c, gin.H{
			"Title": "Enter OTP - SalesMee",
			"Email": email,
			"Error": "Invalid or expired OTP",
		}))
		return
	}

	// Mark as verified
	clientAuth.IsVerified = true
	clientAuth.OTPCode = "" // Clear OTP after verification
	db.DB.Save(&clientAuth)

	// Update client online status
	now := time.Now()
	db.DB.Model(&models.Client{}).Where("id = ?", clientAuth.ClientID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	})

	// Generate JWT token
	token, err := services.GenerateClientToken(clientAuth)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_otp.html", middleware.TemplateData(c, gin.H{
			"Title": "Enter OTP - SalesMee",
			"Email": email,
			"Error": "Failed to generate token",
		}))
		return
	}

	// Set cookie and redirect to saved destination
	c.SetCookie("client_token", token, 86400, "/", "", false, false)
	redirect, _ := c.Cookie("client_redirect")
	c.SetCookie("client_redirect", "", -1, "/client", "", false, true)
	c.Redirect(http.StatusFound, safeClientOAuthRedirect(redirect))
}

func ClientDashboard(c *gin.Context) {
	clientID := c.GetUint("client_id")
	email := c.GetString("client_email")
	log.Printf("[ClientDashboard] clientID=%d, email=%s", clientID, email)

	// If business_id is provided, ensure a conversation exists
	if businessIDStr := c.Query("business_id"); businessIDStr != "" {
		log.Printf("[ClientDashboard] business_id query param: %s", businessIDStr)
		if businessID, err := strconv.ParseUint(businessIDStr, 10, 32); err == nil {
			log.Printf("[ClientDashboard] parsed businessID=%d, calling getOrCreateConversation", businessID)
			conv, client, err := getOrCreateConversation(clientID, uint(businessID))
			if err != nil {
				log.Printf("[ClientDashboard] getOrCreateConversation error: %v", err)
			} else {
				log.Printf("[ClientDashboard] getOrCreateConversation result: conversationID=%d, clientID=%d", conv.ID, client.ID)
			}
		} else {
			log.Printf("[ClientDashboard] ERROR parsing business_id=%s: %v", businessIDStr, err)
		}
	}

	type BusinessWithUnread struct {
		models.Business
		ConversationID uint       `json:"conversation_id"`
		UnreadCount    int        `json:"unread_count"`
		LastMessageAt  *time.Time `json:"last_message_at"`
		LastMessage    *string    `json:"last_message"`
	}

	var businesses []BusinessWithUnread
	query := `
		SELECT
			businesses.*,
			conversations.id as conversation_id,
			COUNT(CASE WHEN messages.sender = 'business' AND messages.created_at > COALESCE(conversations.last_read_by_client_at, '1970-01-01') THEN 1 END) as unread_count,
			MAX(messages.created_at) as last_message_at,
			(SELECT content FROM messages WHERE conversation_id = conversations.id ORDER BY created_at DESC LIMIT 1) as last_message
		FROM businesses
		JOIN conversations ON conversations.business_id = businesses.id AND conversations.client_id = ?
		LEFT JOIN messages ON messages.conversation_id = conversations.id
		GROUP BY businesses.id, conversations.id
		ORDER BY unread_count DESC, last_message_at DESC
	`
	if err := db.DB.Raw(query, clientID).Scan(&businesses).Error; err != nil {
		log.Printf("[ClientDashboard] ERROR running businesses query: %v", err)
		c.HTML(http.StatusInternalServerError, "client.html", gin.H{
			"Title":         "Client Dashboard - SalesMee",
			"Error":         "Failed to load businesses",
			"AssistEnabled": assist.IsEnabled(),
		})
		return
	}
	log.Printf("[ClientDashboard] query returned %d businesses for clientID=%d", len(businesses), clientID)
	for i, b := range businesses {
		log.Printf("[ClientDashboard] business[%d]: ID=%d, Name=%s, ConversationID=%d", i, b.ID, b.Name, b.ConversationID)
	}

	// Debug: check conversations table directly
	var convCount int64
	db.DB.Model(&models.Conversation{}).Where("client_id = ?", clientID).Count(&convCount)
	log.Printf("[ClientDashboard] total conversations for clientID=%d: %d", clientID, convCount)
	var allConvs []models.Conversation
	db.DB.Where("client_id = ?", clientID).Find(&allConvs)
	for _, c := range allConvs {
		log.Printf("[ClientDashboard] conversation DB row: ID=%d, ClientID=%d, BusinessID=%d", c.ID, c.ClientID, c.BusinessID)
	}

	c.HTML(http.StatusOK, "client.html", gin.H{
		"Title":         "Client Dashboard - SalesMee",
		"Email":         email,
		"Businesses":    businesses,
		"AssistEnabled": assist.IsEnabled(),
	})
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

// Helper function to get or create conversation by client email and business ID
func getOrCreateConversation(clientID uint, businessID uint) (*models.Conversation, *models.Client, error) {

	if clientID == 0 || businessID == 0 {
		return nil, nil, fmt.Errorf("missing client_id or business_id field")
	}
	// Get client
	var client models.Client
	if err := db.DB.Where("id = ?", clientID).First(&client).Error; err != nil {
		log.Print("Client not found by id ", clientID)
	}

	// Get or create conversation by client_id AND business_id
	var conversation models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conversation).Error; err != nil {
		conversation = models.Conversation{
			ClientID:   clientID,
			BusinessID: businessID,
		}
		if err := db.DB.Create(&conversation).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to create conversation: %v", err)
		}
		log.Printf("Created new conversation ID=%d for client_id=%d, business_id=%d",
			conversation.ID, clientID, businessID)
	}

	return &conversation, &client, nil
}

func GetClientMessages(c *gin.Context) {
	clientID := c.GetUint("client_id")

	businessIDStr := c.Param("business_id")
	var businessID uint
	if _, err := fmt.Sscanf(businessIDStr, "%d", &businessID); err != nil {
		c.String(http.StatusBadRequest, "Invalid business ID")
		return
	}

	// Get or create conversation using helper
	conversation, client, err := getOrCreateConversation(clientID, businessID)
	if err != nil {
		log.Printf("Error getting conversation: %v", err)
		c.String(http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	// Get messages for this conversation
	var messages []models.Message
	if err := db.DB.Where("conversation_id = ?", conversation.ID).Order("created_at ASC").Find(&messages).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to load messages")
		return
	}

	// Get business info (need currency for order/booking data)
	var business struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		BusinessType string `json:"business_type"`
		Logo         string `json:"logo"`
		Currency     string `json:"currency"`
	}
	db.DB.Raw("SELECT id, name, business_type, logo, currency FROM businesses WHERE id = ?", businessID).First(&business)

	// Convert messages to MessageObj
	var messageObjs []MessageObj
	for _, msg := range messages {
		isSelf := msg.Sender == "client"
		var isDelivered, isRead bool
		if isSelf {
			isDelivered = msg.DeliveredAt != nil
			isRead = msg.ReadByBusiness
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
		case models.OrderDraft:
			actionRequired = "none"
			editable = true
		case "pending":
			if order.Sender == "business" && !order.ConfirmedByClient {
				actionRequired = "client"
				editable = true
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

		var orderPaymentMethods []models.PaymentMethod
		db.DB.Where("business_id = ? AND is_active = ?", order.BusinessID, true).Order("sort_order ASC, id ASC").Find(&orderPaymentMethods)

		var orderPendingAmt float64
		db.DB.Model(&models.Payment{}).Where("order_id = ? AND status = ?", order.ID, models.PaymentPending).
			Select("COALESCE(SUM(amount), 0)").Scan(&orderPendingAmt)

		var orderPayments []models.Payment
		db.DB.Where("order_id = ?", order.ID).Order("created_at desc").Find(&orderPayments)

		var orderHasReview int64
		db.DB.Model(&models.Review{}).Where("order_id = ?", order.ID).Count(&orderHasReview)

		orderData := map[string]interface{}{
			"id":                   order.ID,
			"business_id":          order.BusinessID,
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
			"has_review":           orderHasReview > 0,
			"quantity":             order.Quantity,
			"notes":                order.Notes,
			"currency":             business.Currency,
			"product_names":        productNames,
			"first_product_name":   firstProductName,
			"created_at":           order.CreatedAt,
			"payment_methods":      orderPaymentMethods,
			"payments":             orderPayments,
		}

		isSelf := order.Sender == "client"
		isDelivered := isSelf && conversation.LastReadByBusinessAt != nil && order.CreatedAt.Before(*conversation.LastReadByBusinessAt)

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
		var bookingItems []models.BookingItem
		db.DB.Where("booking_id = ?", booking.ID).Find(&bookingItems)

		var serviceNames []string
		var firstServiceID uint
		for _, item := range bookingItems {
			var service models.Service
			db.DB.First(&service, item.ServiceID)
			serviceNames = append(serviceNames, service.Name)
			if firstServiceID == 0 {
				firstServiceID = item.ServiceID
			}
		}

		bookingRemaining := booking.TotalAmount - booking.PaidAmount
		if bookingRemaining < 0 {
			bookingRemaining = 0
		}

		var bookingPaymentMethods []models.PaymentMethod
		db.DB.Where("business_id = ? AND is_active = ?", booking.BusinessID, true).Order("sort_order ASC, id ASC").Find(&bookingPaymentMethods)

		var bookingPendingAmt float64
		db.DB.Model(&models.Payment{}).Where("booking_id = ? AND status = ?", booking.ID, models.PaymentPending).
			Select("COALESCE(SUM(amount), 0)").Scan(&bookingPendingAmt)

		var bookingPayments []models.Payment
		db.DB.Where("booking_id = ?", booking.ID).Order("created_at desc").Find(&bookingPayments)

		var bookingActionRequired string
		if booking.Status == models.BookingPending && booking.Sender == "business" {
			bookingActionRequired = "client"
		} else if booking.Status == models.BookingClientConfirmed && booking.PaidAmount < booking.TotalAmount {
			bookingActionRequired = "client"
		} else {
			bookingActionRequired = "none"
		}

		var bookingHasReview int64
		db.DB.Model(&models.Review{}).Where("booking_id = ?", booking.ID).Count(&bookingHasReview)

		bookingData := map[string]interface{}{
			"id":                   booking.ID,
			"business_id":          booking.BusinessID,
			"booking_number":       booking.BookingNumber,
			"service_id":           firstServiceID,
			"status":               booking.Status,
			"scheduled_date":       booking.ScheduledDate.Format("Jan 2, 2006 3:04 PM"),
			"scheduled_date_iso":   booking.ScheduledDate.Format("2006-01-02"),
			"scheduled_time_iso":   booking.ScheduledDate.Format("15:04"),
			"duration":             booking.Duration,
			"total_amount":         booking.TotalAmount,
			"paid_amount":          booking.PaidAmount,
			"pending_amount":       bookingPendingAmt,
			"remaining":            bookingRemaining,
			"is_fully_paid":        booking.PaidAmount >= booking.TotalAmount,
			"has_review":           bookingHasReview > 0,
			"notes":                booking.Notes,
			"sender":               booking.Sender,
			"action_required":      bookingActionRequired,
			"currency":             business.Currency,
			"created_at":           booking.CreatedAt,
			"service_names":        serviceNames,
			"payment_methods":      bookingPaymentMethods,
			"payments":             bookingPayments,
		}

		isSelf := booking.Sender == "client"
		isDelivered := isSelf && conversation.LastReadByBusinessAt != nil && booking.CreatedAt.Before(*conversation.LastReadByBusinessAt)

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

	// Sort all messageObjs by CreatedAt
	for i := 0; i < len(messageObjs); i++ {
		for j := i + 1; j < len(messageObjs); j++ {
			if messageObjs[i].CreatedAt.After(messageObjs[j].CreatedAt) {
				messageObjs[i], messageObjs[j] = messageObjs[j], messageObjs[i]
			}
		}
	}

	c.HTML(http.StatusOK, "client_chat.html", gin.H{
		"Business":       business,
		"Client":         client,
		"ConversationID": conversation.ID,
		"Messages":       messages,
		"MessageObjs":    messageObjs,
	})
}

func CreateClientMessage(c *gin.Context) {
	clientID := c.GetUint("client_id")

	businessIDStr := c.Param("business_id")
	var businessID uint
	if _, err := fmt.Sscanf(businessIDStr, "%d", &businessID); err != nil {
		c.String(http.StatusBadRequest, "Invalid business ID")
		return
	}

	log.Printf("CreateClientMessage: clientID=%d, businessID=%d", clientID, businessID)

	// Get or create conversation using helper (same as GetClientMessages)
	conversation, _, err := getOrCreateConversation(clientID, businessID)
	if err != nil {
		log.Printf("Error getting conversation: %v", err)
		c.String(http.StatusInternalServerError, "Failed to get conversation")
		return
	}

	content := c.PostForm("content")
	sender := c.PostForm("sender")
	if sender == "" {
		sender = "client"
	}

	// Create message with the correct conversation ID
	message := models.Message{
		ConversationID: conversation.ID,
		Content:        content,
		Type:           "message",
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

	if content == "" && message.MediaURL == "" {
		c.String(http.StatusBadRequest, "Message cannot be empty")
		return
	}

	if err := db.DB.Create(&message).Error; err != nil {
		log.Printf("Error creating message: %v", err)
		c.String(http.StatusInternalServerError, "Failed to create message")
		return
	}

	log.Printf("Message created: ID=%d, ConvoID=%d, Content='%s', Sender='%s'",
		message.ID, message.ConversationID, message.Content, message.Sender)

	if wsHub != nil {
		var dataJSON []byte
		ws.BroadcastNewMessage(
			wsHub,
			strconv.Itoa(int(conversation.ID)),
			strconv.Itoa(int(clientID)),
			"client",
			strconv.Itoa(int(message.ID)),
			message.Content,
			message.MediaURL,
			message.MediaType,
			message.Type,
			dataJSON,
			message.CreatedAt,
			strconv.FormatUint(uint64(businessID), 10),
			strconv.Itoa(int(clientID)),
		)

		var bizUnread int64
		db.DB.Model(&models.Message{}).
			Where("conversation_id = ? AND sender = 'client' AND read_by_business = ?", conversation.ID, false).
			Count(&bizUnread)
		ws.BroadcastUnreadCount(wsHub, strconv.Itoa(int(conversation.ID)), int32(bizUnread), strconv.FormatUint(uint64(businessID), 10), "biz")
	}

	// Return the newly created message as MessageObj for HTMX
	messageObj := MessageObj{
		ID:        message.ID,
		MsgType:   "message",
		Value:     message.Content,
		Data:      nil,
		Sender:    message.Sender,
		MediaURL:  message.MediaURL,
		MediaType: message.MediaType,
		CreatedAt: message.CreatedAt,
	}

	c.HTML(http.StatusOK, "client_message_partial.html", gin.H{
		"MessageObj": messageObj,
	})
}

func ClientMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token, _ = c.Cookie("client_token")
		}

		if token == "" {
			c.SetCookie("client_redirect", c.Request.URL.String(), 300, "/client", "", false, true)
			c.Redirect(http.StatusFound, "/client/login")
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := services.ValidateToken(token)
		if err != nil || claims.Subject != "client" {
			c.SetCookie("client_redirect", c.Request.URL.String(), 300, "/client", "", false, true)
			c.Redirect(http.StatusFound, "/client/login")
			c.Abort()
			return
		}

		c.Set("client_id", claims.UserID)
		c.Set("client_email", claims.Email)
		c.Next()
	}
}

func ClientLogout(c *gin.Context) {
	// Get client info from token
	token, _ := c.Cookie("client_token")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		claims, err := services.ValidateToken(token)
		if err == nil && claims.Subject == "client" {
			// Update client offline status
			db.DB.Model(&models.Client{}).Where("id = ?", claims.UserID).Update("is_online", false)

			// Broadcast offline presence via WebSocket
			if wsHub != nil {
				var conversations []models.Conversation
				db.DB.Where("client_id = ?", claims.UserID).Find(&conversations)
				clientID := strconv.Itoa(int(claims.UserID))
				now := time.Now().UnixMilli()
				for _, conv := range conversations {
					ws.BroadcastPresenceUpdate(wsHub, clientID, false, now, strconv.Itoa(int(conv.BusinessID)))
				}
			}
		}
	}

	// Clear cookie and redirect
	c.SetCookie("client_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/client/login")
}

// ClientUpdateOrder allows clients to update their order items, notes, and address
// Supports both JSON (new multi-item format) and form-encoded (legacy) payloads
func ClientUpdateOrder(c *gin.Context) {
	clientID := c.GetUint("client_id")
	orderID := c.Param("id")

	var order models.Order
	if err := db.DB.Where("id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Try JSON first (new multi-item format)
	var jsonRequest struct {
		Items           []struct {
			ProductID uint `json:"product_id"`
			Quantity  int  `json:"quantity"`
		} `json:"items"`
		Notes           string `json:"notes"`
		DeliveryAddress string `json:"delivery_address"`
	}

	if c.GetHeader("Content-Type") == "application/json" {
		if err := c.ShouldBindJSON(&jsonRequest); err == nil && len(jsonRequest.Items) > 0 {
			// Multi-item JSON update: replace all items
			var oldItems []models.OrderItem
			db.DB.Where("order_id = ?", order.ID).Find(&oldItems)

			// Restore stock for old items
			for _, oldItem := range oldItems {
				var product models.Product
				db.DB.First(&product, oldItem.ProductID)
				product.Stock += oldItem.Quantity
				db.DB.Save(&product)
			}

			// Delete old items
			db.DB.Where("order_id = ?", order.ID).Delete(&models.OrderItem{})

			// Create new items
			var totalAmount float64
			for _, item := range jsonRequest.Items {
				var product models.Product
				if err := db.DB.First(&product, item.ProductID).Error; err != nil {
					continue
				}
				if product.Stock < item.Quantity {
					continue
				}
				itemTotal := float64(item.Quantity) * product.Price
				totalAmount += itemTotal
				db.DB.Create(&models.OrderItem{
					OrderID:    order.ID,
					ProductID:  product.ID,
					Quantity:   item.Quantity,
					UnitPrice:  product.Price,
					TotalPrice: itemTotal,
				})
				product.Stock -= item.Quantity
				db.DB.Save(&product)
			}

			fullNotes := jsonRequest.Notes
			if jsonRequest.DeliveryAddress != "" {
				fullNotes = "📍 Delivery Address: " + jsonRequest.DeliveryAddress + "\n" + fullNotes
			}
			order.TotalAmount = totalAmount
			order.Quantity = len(jsonRequest.Items)
			order.Notes = fullNotes
			db.DB.Save(&order)
			c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
			return
		}
	}

	// Fallback: form-encoded legacy update (notes + quantity)
	notes := c.PostForm("notes")
	if notes != "" {
		order.Notes = notes
	}

	quantityStr := c.PostForm("quantity")
	if quantityStr != "" {
		if quantity, err := strconv.Atoi(quantityStr); err == nil && quantity > 0 {
			var orderItem models.OrderItem
			if err := db.DB.Where("order_id = ?", order.ID).First(&orderItem).Error; err == nil {
				order.TotalAmount = float64(quantity) * orderItem.UnitPrice
				orderItem.Quantity = quantity
				orderItem.TotalPrice = order.TotalAmount
				db.DB.Save(&orderItem)
			}
			order.Quantity = quantity
		}
	}

	db.DB.Save(&order)
	c.JSON(http.StatusOK, gin.H{"success": true, "order": order})
}

// ClientConfirmOrder allows the client to confirm an order
func ClientConfirmOrder(c *gin.Context) {
	clientID := c.GetUint("client_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := db.DB.Where("id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != models.OrderPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be confirmed in current status"})
		return
	}

	if order.ConfirmedByClient {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already confirmed by client"})
		return
	}

	if order.Sender == "client" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is pending business approval"})
		return
	}

	// Update client-side items if quantities changed
	var request struct {
		Items []struct {
			ProductID uint `json:"product_id"`
			Quantity  int  `json:"quantity"`
		} `json:"items,omitempty"`
	}
	c.ShouldBindJSON(&request)

	if len(request.Items) > 0 {
		var totalAmount float64
		for _, reqItem := range request.Items {
			var orderItem models.OrderItem
			if err := db.DB.Where("order_id = ? AND product_id = ?", order.ID, reqItem.ProductID).First(&orderItem).Error; err != nil {
				continue
			}
			oldQty := orderItem.Quantity
			orderItem.Quantity = reqItem.Quantity
			orderItem.TotalPrice = float64(reqItem.Quantity) * orderItem.UnitPrice
			totalAmount += orderItem.TotalPrice
			db.DB.Save(&orderItem)

			// Adjust stock for quantity change
			diff := oldQty - reqItem.Quantity
			if diff != 0 {
				var product models.Product
				db.DB.First(&product, reqItem.ProductID)
				product.Stock += diff
				db.DB.Save(&product)
				db.DB.Create(&models.InventoryLog{
					ProductID: product.ID,
					Type: func() string {
						if diff > 0 { return "in" } else { return "out" }
					}(),
					Quantity: func() int {
						if diff < 0 { return -diff } else { return diff }
					}(),
					Reason: fmt.Sprintf("Client qty change on confirm #%s", order.OrderNumber),
				})
			}
		}
		if totalAmount > 0 {
			order.TotalAmount = totalAmount
		}
	}

	now := time.Now()
	order.ConfirmedByClient = true
	order.ConfirmedByClientAt = &now
	order.Status = models.OrderClientConfirmed
	order.UpdatedAt = now

	if err := db.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm order"})
		return
	}

	if wsHub != nil {
		bizCardHTML := renderBizOrderCard(db.DB, order)
		clientCardHTML := renderClientOrderCard(db.DB, order)
		ws.BroadcastOrderUpdateFull(wsHub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(order.BusinessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order confirmed! Waiting for business approval.",
	})
}

// ClientCancelOrder allows a client to cancel their own order
func ClientCancelOrder(c *gin.Context) {
	clientID := c.GetUint("client_id")
	orderIDStr := c.Param("id")

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := db.DB.Where("id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status == models.OrderConfirmed || order.Status == models.OrderFulfilled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel a confirmed/fulfilled order"})
		return
	}

	if order.Status == models.OrderCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order is already cancelled"})
		return
	}

	order.Status = models.OrderCancelled
	order.UpdatedAt = time.Now()
	db.DB.Save(&order)

	if wsHub != nil {
		bizCardHTML := renderBizOrderCard(db.DB, order)
		clientCardHTML := renderClientOrderCard(db.DB, order)
		ws.BroadcastOrderUpdateFull(wsHub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(order.BusinessID)), strconv.Itoa(int(order.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"order":   order,
		"message": "Order cancelled",
	})
}

// ClientUpdateBooking allows clients to update their booking date, notes, and service
func ClientUpdateBooking(c *gin.Context) {
	clientID := c.GetUint("client_id")
	bookingID := c.Param("id")

	var booking models.Booking
	if err := db.DB.Where("id = ? AND client_id = ?", bookingID, clientID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if c.GetHeader("Content-Type") == "application/json" {
		var req struct {
			ServiceID       uint   `json:"service_id"`
			ScheduledDate   string `json:"scheduled_date"`
			Notes           string `json:"notes"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.Notes != "" {
				booking.Notes = req.Notes
			}
			if req.ScheduledDate != "" {
				if newDate, err := time.Parse(time.RFC3339, req.ScheduledDate); err == nil {
					booking.ScheduledDate = newDate
				}
			}
			if req.ServiceID > 0 {
				var bookingItems []models.BookingItem
				db.DB.Where("booking_id = ?", booking.ID).Find(&bookingItems)
				if len(bookingItems) > 0 {
					var service models.Service
					if err := db.DB.First(&service, req.ServiceID).Error; err == nil {
						bookingItems[0].ServiceID = req.ServiceID
						bookingItems[0].UnitPrice = service.MaxPrice
						bookingItems[0].TotalPrice = service.MaxPrice
						db.DB.Save(&bookingItems[0])
						booking.TotalAmount = service.MaxPrice
					}
				}
			}
			db.DB.Save(&booking)
			c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
			return
		}
	}

	// Fallback: form-encoded legacy update
	notes := c.PostForm("notes")
	scheduledDate := c.PostForm("scheduled_date")

	if notes != "" {
		booking.Notes = notes
	}
	if scheduledDate != "" {
		if newDate, err := time.Parse(time.RFC3339, scheduledDate); err == nil {
			booking.ScheduledDate = newDate
		}
	}

	db.DB.Save(&booking)
	c.JSON(http.StatusOK, gin.H{"success": true, "booking": booking})
}

// ClientCancelBooking allows a client to cancel their own booking
func ClientCancelBooking(c *gin.Context) {
	clientID := c.GetUint("client_id")
	bookingIDStr := c.Param("id")

	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := db.DB.Where("id = ? AND client_id = ?", bookingID, clientID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status == models.BookingCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel a completed booking"})
		return
	}

	if booking.Status == models.BookingCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking is already cancelled"})
		return
	}

	booking.Status = models.BookingCancelled
	booking.UpdatedAt = time.Now()
	db.DB.Save(&booking)

	if wsHub != nil {
		bizCardHTML := renderBizBookingCard(db.DB, booking)
		clientCardHTML := renderClientBookingCard(db.DB, booking)
		ws.BroadcastBookingUpdateFull(wsHub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(booking.BusinessID)), strconv.Itoa(int(booking.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"booking": booking,
		"message": "Booking cancelled successfully",
	})
}

// ClientConfirmBooking allows a client to confirm/approve a booking
func ClientConfirmBooking(c *gin.Context) {
	clientID := c.GetUint("client_id")
	bookingIDStr := c.Param("id")

	bookingID, err := strconv.ParseUint(bookingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := db.DB.Where("id = ? AND client_id = ?", bookingID, clientID).First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	if booking.Status != models.BookingPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking cannot be confirmed in current status"})
		return
	}

	if booking.Sender == "client" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Booking is pending business approval"})
		return
	}

	booking.Status = models.BookingClientConfirmed
	booking.UpdatedAt = time.Now()
	db.DB.Save(&booking)

	if wsHub != nil {
		bizCardHTML := renderBizBookingCard(db.DB, booking)
		clientCardHTML := renderClientBookingCard(db.DB, booking)
		ws.BroadcastBookingUpdateFull(wsHub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, 0, false, 0, bizCardHTML, clientCardHTML, strconv.Itoa(int(booking.BusinessID)), strconv.Itoa(int(booking.ClientID)))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"booking": booking,
		"message": "Booking confirmed! Waiting for business to complete.",
	})
}

func DeleteClientMessage(c *gin.Context) {
	clientID := c.GetUint("client_id")
	messageIDStr := c.Param("message_id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	switch {
	case messageID >= 20000:
		bookingID := messageID - 20000
		var booking models.Booking
		if err := db.DB.Where("id = ? AND client_id = ?", bookingID, clientID).First(&booking).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		db.DB.Model(&booking).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastBookingUpdate(wsHub, strconv.Itoa(int(booking.ID)), booking.Status, booking.PaidAmount, booking.TotalAmount, strconv.Itoa(int(booking.BusinessID)), strconv.Itoa(int(booking.ClientID)))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "booking", "id": bookingID})

	case messageID >= 10000:
		orderID := messageID - 10000
		var order models.Order
		if err := db.DB.Where("id = ? AND client_id = ?", orderID, clientID).First(&order).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		db.DB.Model(&order).Update("hidden_from_chat", true)
		if wsHub != nil {
			ws.BroadcastOrderUpdate(wsHub, strconv.Itoa(int(order.ID)), order.Status, order.PaidAmount, order.TotalAmount, strconv.Itoa(int(order.BusinessID)), strconv.Itoa(int(order.ClientID)))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "order", "id": orderID})

	default:
		var msg models.Message
		if err := db.DB.Preload("Conversation").First(&msg, messageID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		if msg.Conversation.ClientID != clientID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
			return
		}
		db.DB.Delete(&msg)
		c.JSON(http.StatusOK, gin.H{"success": true, "type": "message", "id": messageID})
	}
}
