package client

import (
	"bytes"
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
