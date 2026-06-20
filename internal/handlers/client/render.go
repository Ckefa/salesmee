package client

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"time"
	"salesmee/internal/models"
	"gorm.io/gorm"
)

var CardTmpl *template.Template

func SetTemplate(tmpl *template.Template) {
	CardTmpl = tmpl
}

func renderCard(tmplName string, data map[string]interface{}, sender string, msgID uint, createdAt time.Time) (string, error) {
	var buf bytes.Buffer
	err := CardTmpl.ExecuteTemplate(&buf, tmplName, map[string]interface{}{
		"Data":        data,
		"Sender":      sender,
		"ID":          msgID,
		"CreatedAt":   createdAt,
		"IsDelivered": false,
		"IsRead":      false,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func clientOrderActionRequired(order models.Order) (string, bool) {
	switch order.Status {
	case "draft":
		return "none", true
	case "pending":
		if order.Sender == "business" && !order.ConfirmedByClient {
			return "client", true
		} else if order.Sender == "client" && !order.ConfirmedByBusiness {
			return "business", false
		}
		return "none", false
	case "client_confirmed":
		return "business", false
	case "confirmed":
		return "none", false
	case "fulfilled":
		return "none", false
	case "cancelled":
		return "none", false
	default:
		return "none", false
	}
}

func clientBookingActionRequired(booking models.Booking) string {
	if booking.Status == "pending" && booking.Sender == "business" {
		return "client"
	}
	if booking.Status == "client_confirmed" && booking.PaidAmount < booking.TotalAmount {
		return "client"
	}
	return "none"
}

func BuildClientOrderCardData(d *gorm.DB, order models.Order) map[string]interface{} {
	var orderItems []models.OrderItem
	d.Where("order_id = ?", order.ID).Preload("Product").Find(&orderItems)

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

	actionRequired, editable := clientOrderActionRequired(order)

	remaining := order.TotalAmount - order.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	var paymentMethods []models.PaymentMethod
	d.Where("business_id = ? AND is_active = ?", order.BusinessID, true).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	var pendingAmt float64
	d.Model(&models.Payment{}).Where("order_id = ? AND status = ?", order.ID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	var payments []models.Payment
	d.Where("order_id = ?", order.ID).Order("created_at desc").Find(&payments)

	var reviewRating int
	var hasReview bool
	d.Model(&models.Review{}).Where("order_id = ?", order.ID).Select("rating").Scan(&reviewRating)
	if reviewRating > 0 {
		hasReview = true
	}

	return map[string]interface{}{
		"id":                 order.ID,
		"business_id":        order.BusinessID,
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
		"paid_amount":        order.PaidAmount,
		"pending_amount":     pendingAmt,
		"remaining":          remaining,
		"is_fully_paid":      order.PaidAmount >= order.TotalAmount,
		"has_review":         hasReview,
		"review_rating":      reviewRating,
		"quantity":           order.Quantity,
		"notes":              order.Notes,
		"currency": func() string {
			var biz models.Business
			d.Select("currency").First(&biz, order.BusinessID)
			return biz.Currency
		}(),
		"product_names":      productNames,
		"first_product_name": firstProductName,
		"created_at":         order.CreatedAt,
		"payment_methods":    paymentMethods,
		"payments":           payments,
	}
}

func BuildClientBookingCardData(d *gorm.DB, booking models.Booking) map[string]interface{} {
	var bookingItems []models.BookingItem
	d.Where("booking_id = ?", booking.ID).Find(&bookingItems)

	var serviceNames []string
	var firstServiceID uint
	for _, item := range bookingItems {
		var service models.Service
		d.First(&service, item.ServiceID)
		if firstServiceID == 0 {
			firstServiceID = item.ServiceID
		}
		serviceNames = append(serviceNames, service.Name)
	}

	remaining := booking.TotalAmount - booking.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	var paymentMethods []models.PaymentMethod
	d.Where("business_id = ? AND is_active = ?", booking.BusinessID, true).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	var pendingAmt float64
	d.Model(&models.Payment{}).Where("booking_id = ? AND status = ?", booking.ID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	var payments []models.Payment
	d.Where("booking_id = ?", booking.ID).Order("created_at desc").Find(&payments)

	actionRequired := clientBookingActionRequired(booking)

	var reviewRating int
	var hasReview bool
	d.Model(&models.Review{}).Where("booking_id = ?", booking.ID).Select("rating").Scan(&reviewRating)
	if reviewRating > 0 {
		hasReview = true
	}

	return map[string]interface{}{
		"id":               booking.ID,
		"business_id":      booking.BusinessID,
		"booking_number":   booking.BookingNumber,
		"service_id":       firstServiceID,
		"status":           booking.Status,
		"scheduled_date":   booking.ScheduledDate.Format("Jan 2, 2006 3:04 PM"),
		"scheduled_date_iso": booking.ScheduledDate.Format("2006-01-02"),
		"scheduled_time_iso": booking.ScheduledDate.Format("15:04"),
		"duration":         booking.Duration,
		"total_amount":     booking.TotalAmount,
		"paid_amount":      booking.PaidAmount,
		"pending_amount":   pendingAmt,
		"remaining":        remaining,
		"is_fully_paid":    booking.PaidAmount >= booking.TotalAmount,
		"has_review":       hasReview,
		"review_rating":    reviewRating,
		"notes":            booking.Notes,
		"sender":           booking.Sender,
		"action_required":  actionRequired,
		"currency": func() string {
			var biz models.Business
			d.Select("currency").First(&biz, booking.BusinessID)
			return biz.Currency
		}(),
		"created_at":       booking.CreatedAt,
		"service_names":    serviceNames,
		"payment_methods":  paymentMethods,
		"payments":         payments,
	}
}

func renderClientOrderCard(d *gorm.DB, order models.Order) string {
	data := BuildClientOrderCardData(d, order)
	html, err := renderCard("client_order_card", data, order.Sender, order.ID, order.CreatedAt)
	if err != nil {
		return ""
	}
	return html
}

func renderClientBookingCard(d *gorm.DB, booking models.Booking) string {
	data := BuildClientBookingCardData(d, booking)
	html, err := renderCard("client_booking_card", data, booking.Sender, booking.ID, booking.CreatedAt)
	if err != nil {
		return ""
	}
	return html
}

func bizOrderActionRequired(order models.Order) (string, bool) {
	switch order.Status {
	case "draft":
		return "none", true
	case "pending":
		if order.Sender == "business" && !order.ConfirmedByClient {
			return "client", true
		} else if order.Sender == "client" && !order.ConfirmedByBusiness {
			return "business", false
		}
		return "none", false
	case "client_confirmed":
		return "business", false
	case "confirmed":
		return "none", false
	case "fulfilled":
		return "none", false
	case "cancelled":
		return "none", false
	default:
		return "none", false
	}
}

func bizBookingActionRequired(booking models.Booking) string {
	if booking.Status == "pending" && booking.Sender == "client" {
		return "business"
	}
	if booking.Status == "client_confirmed" && !(booking.PaidAmount >= booking.TotalAmount) {
		return "business"
	}
	return "none"
}

func BuildBizOrderCardData(d *gorm.DB, order models.Order) map[string]interface{} {
	var orderItems []models.OrderItem
	d.Where("order_id = ?", order.ID).Preload("Product").Find(&orderItems)

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

	actionRequired, editable := bizOrderActionRequired(order)

	remaining := order.TotalAmount - order.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	var paymentMethods []models.PaymentMethod
	d.Where("business_id = ? AND is_active = ?", order.BusinessID, true).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	var pendingAmt float64
	d.Model(&models.Payment{}).Where("order_id = ? AND status = ?", order.ID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	var payments []models.Payment
	d.Where("order_id = ?", order.ID).Order("created_at desc").Find(&payments)

	var reviewRating int
	var hasReview bool
	d.Model(&models.Review{}).Where("order_id = ?", order.ID).Select("rating").Scan(&reviewRating)
	if reviewRating > 0 {
		hasReview = true
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
		"paid_amount":        order.PaidAmount,
		"pending_amount":     pendingAmt,
		"remaining":          remaining,
		"is_fully_paid":      order.PaidAmount >= order.TotalAmount,
		"has_review":         hasReview,
		"review_rating":      reviewRating,
		"quantity":           order.Quantity,
		"notes":              order.Notes,
		"currency": func() string {
			var biz models.Business
			d.Select("currency").First(&biz, order.BusinessID)
			return biz.Currency
		}(),
		"product_names":      productNames,
		"first_product_name": firstProductName,
		"created_at":         order.CreatedAt,
		"payment_methods":    paymentMethods,
		"payments":           payments,
	}
}

func BuildBizBookingCardData(d *gorm.DB, booking models.Booking) map[string]interface{} {
	var bookingItems []models.BookingItem
	d.Where("booking_id = ?", booking.ID).Find(&bookingItems)

	var serviceName string
	var serviceNames []string
	var firstServiceID uint
	for _, item := range bookingItems {
		var service models.Service
		if err := d.First(&service, item.ServiceID).Error; err == nil {
			serviceName = service.Name
			serviceNames = append(serviceNames, service.Name)
			if firstServiceID == 0 {
				firstServiceID = item.ServiceID
			}
		}
	}

	remaining := booking.TotalAmount - booking.PaidAmount
	if remaining < 0 {
		remaining = 0
	}

	var paymentMethods []models.PaymentMethod
	d.Where("business_id = ? AND is_active = ?", booking.BusinessID, true).Order("sort_order ASC, id ASC").Find(&paymentMethods)

	var pendingAmt float64
	d.Model(&models.Payment{}).Where("booking_id = ? AND status = ?", booking.ID, "pending").
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingAmt)

	var payments []models.Payment
	d.Where("booking_id = ?", booking.ID).Order("created_at desc").Find(&payments)

	actionRequired := bizBookingActionRequired(booking)

	var reviewRating int
	var hasReview bool
	d.Model(&models.Review{}).Where("booking_id = ?", booking.ID).Select("rating").Scan(&reviewRating)
	if reviewRating > 0 {
		hasReview = true
	}

	return map[string]interface{}{
		"id":              booking.ID,
		"booking_number":  booking.BookingNumber,
		"service_id":      firstServiceID,
		"service_name":    serviceName,
		"service_names":   serviceNames,
		"scheduled_date":  booking.ScheduledDate,
		"duration":        booking.Duration,
		"total_amount":    booking.TotalAmount,
		"paid_amount":     booking.PaidAmount,
		"pending_amount":  pendingAmt,
		"remaining":       remaining,
		"is_fully_paid":   booking.PaidAmount >= booking.TotalAmount,
		"has_review":      hasReview,
		"review_rating":   reviewRating,
		"notes":           booking.Notes,
		"status":          booking.Status,
		"sender":          booking.Sender,
		"action_required": actionRequired,
		"currency": func() string {
			var biz models.Business
			d.Select("currency").First(&biz, booking.BusinessID)
			return biz.Currency
		}(),
		"created_at":      booking.CreatedAt,
		"payment_methods": paymentMethods,
		"payments":        payments,
	}
}

func renderBizOrderCard(d *gorm.DB, order models.Order) string {
	data := BuildBizOrderCardData(d, order)
	html, err := renderCard("order_card", data, order.Sender, order.ID, order.CreatedAt)
	if err != nil {
		return ""
	}
	return html
}

func renderBizBookingCard(d *gorm.DB, booking models.Booking) string {
	data := BuildBizBookingCardData(d, booking)
	html, err := renderCard("booking_card", data, booking.Sender, booking.ID, booking.CreatedAt)
	if err != nil {
		return ""
	}
	return html
}

func RenderClientSidebarCard(business models.Business, conversationID uint, lastMessage string, lastMessageAt time.Time, unreadCount int) string {
	timeStr := ""
	if !lastMessageAt.IsZero() {
		timeStr = lastMessageAt.Format(time.RFC3339)
	}
	preview := "No messages yet"
	if lastMessage != "" {
		if len(lastMessage) > 60 {
			preview = html.EscapeString(lastMessage[:57]) + "..."
		} else {
			preview = html.EscapeString(lastMessage)
		}
	} else {
		preview = "<span class=\"italic opacity-60\">No messages yet</span>"
	}
	badge := ""
	if unreadCount > 0 {
		countStr := fmt.Sprintf("%d", unreadCount)
		if unreadCount > 99 {
			countStr = "99+"
		}
		badge = fmt.Sprintf("<span class=\"wa-unread-badge\">%s</span>", countStr)
	}
	avatar := fmt.Sprintf(`<div class="wa-chat-avatar wa-sidebar-avatar-placeholder"><i class="fas fa-store"></i></div>`)
	if business.Logo != "" {
		avatar = fmt.Sprintf(`<img src="/static/%s" alt="%s" class="wa-chat-avatar" loading="lazy">`, html.EscapeString(business.Logo), html.EscapeString(business.Name))
	}
	return fmt.Sprintf(
		`<div class="wa-chat-item business-item" data-business-id="%d" data-conversation-id="%d" data-business-name="%s" data-business-type="%s" data-last-message-at="%s" data-unread="%d" data-online="false">
			<div class="wa-chat-avatar-wrapper">
				%s
				<div class="wa-online-dot"></div>
			</div>
			<div class="wa-chat-info">
				<div class="wa-chat-top">
					<span class="wa-chat-name">%s</span>
					<span class="wa-chat-time time-ago" data-time="%s"></span>
				</div>
				<div class="wa-chat-bottom">
					<span class="wa-chat-preview">%s</span>
					%s
				</div>
			</div>
			<div class="flex items-center gap-0.5 ml-auto shrink-0">
				<button onclick="event.stopPropagation(); togglePinBusiness(%d)" class="pin-btn wa-chat-icon-btn text-[var(--color-text-muted)] hover:text-amber-500" title="Pin to top">
					<i class="fas fa-star text-[10px]"></i>
				</button>
				<button onclick="event.stopPropagation(); disconnectBusiness(%d)" class="wa-chat-icon-btn text-[var(--color-text-muted)] hover:text-[var(--color-error)]" title="Remove business">
					<i class="fas fa-trash-alt text-[10px]"></i>
				</button>
			</div>
		</div>`,
		business.ID, conversationID, html.EscapeString(business.Name), html.EscapeString(business.BusinessType), timeStr, unreadCount,
		avatar, html.EscapeString(business.Name), timeStr, preview, badge, business.ID, business.ID,
	)
}

func RenderBizSidebarCard(client models.Client, conversationID uint, lastMessage string, lastMessageAt time.Time, unreadCount int) string {
	timeStr := ""
	if !lastMessageAt.IsZero() {
		timeStr = lastMessageAt.Format(time.RFC3339)
	}
	preview := "No messages yet"
	if lastMessage != "" {
		if len(lastMessage) > 60 {
			preview = html.EscapeString(lastMessage[:57]) + "..."
		} else {
			preview = html.EscapeString(lastMessage)
		}
	} else {
		preview = "<span class=\"italic opacity-60\">No messages yet</span>"
	}
	badge := ""
	if unreadCount > 0 {
		countStr := fmt.Sprintf("%d", unreadCount)
		if unreadCount > 99 {
			countStr = "99+"
		}
		badge = fmt.Sprintf("<span class=\"wa-unread-badge\">%s</span>", countStr)
	}
	onlineStr := fmt.Sprintf("%t", client.IsOnline)
	onlineDot := "offline"
	if client.IsOnline {
		onlineDot = "online"
	}
	return fmt.Sprintf(
		`<div class="wa-chat-item group" data-client-id="%d" data-conversation-id="%d" data-client-name="%s" data-last-message-at="%s" data-unread="%d" data-online="%s" role="row">
			<div class="wa-chat-avatar-wrapper">
				<div class="wa-chat-avatar avatar-placeholder"><i class="fas fa-user text-white"></i></div>
				<span class="wa-online-dot %s"></span>
			</div>
			<div class="wa-chat-info">
				<div class="wa-chat-top">
					<span class="wa-chat-name">%s</span>
					<div class="wa-chat-top-right">
						<span class="wa-chat-time time-ago" data-time="%s"></span>
						%s
					</div>
				</div>
				<div class="wa-chat-bottom">
					<span class="wa-chat-preview">%s</span>
				</div>
			</div>
			<div class="flex items-center gap-0.5 ml-auto shrink-0">
				<button onclick="event.stopPropagation(); togglePinClient('%d')" class="pin-btn wa-chat-icon-btn text-[var(--color-text-muted)] hover:text-amber-500" title="Pin to top">
					<i class="fas fa-star text-[10px]"></i>
				</button>
				<button onclick="event.stopPropagation(); deleteClient('%d')" title="Delete" class="wa-chat-icon-btn text-[var(--color-text-muted)] hover:text-[var(--color-error)]">
					<i class="fas fa-trash-alt text-[10px]"></i>
				</button>
			</div>
		</div>`,
		client.ID, conversationID, html.EscapeString(client.Name), timeStr, unreadCount, onlineStr,
		onlineDot, html.EscapeString(client.Name), timeStr, badge, preview, client.ID, client.ID,
	)
}
