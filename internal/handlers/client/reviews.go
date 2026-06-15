package client

import (
	"fmt"
	"net/http"
	"strconv"
	"salesmee/internal/db"
	"salesmee/internal/models"
	"salesmee/internal/ws"
	"time"

	"github.com/gin-gonic/gin"
)

func SubmitReview(c *gin.Context) {
	clientID := c.GetUint("client_id")

	businessIDStr := c.PostForm("business_id")
	orderIDStr := c.PostForm("order_id")
	bookingIDStr := c.PostForm("booking_id")
	ratingStr := c.PostForm("rating")
	title := c.PostForm("title")
	content := c.PostForm("content")

	if businessIDStr == "" || ratingStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Business ID and rating are required"})
		return
	}

	var businessID, orderID, bookingID uint
	if _, err := parseUint(businessIDStr, &businessID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	rating := 0
	if _, err := parseInt(ratingStr, &rating); err != nil || rating < 1 || rating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 1 and 5"})
		return
	}

	if orderIDStr != "" {
		if _, err := parseUint(orderIDStr, &orderID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
			return
		}
	} else if bookingIDStr != "" {
		if _, err := parseUint(bookingIDStr, &bookingID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID or Booking ID is required"})
		return
	}

	// Verify client owns this business relationship
	var conv models.Conversation
	if err := db.DB.Where("client_id = ? AND business_id = ?", clientID, businessID).First(&conv).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Client not associated with this business"})
		return
	}

	// Validate order/booking is completed and belongs to client
	if orderID > 0 {
		var order models.Order
		if err := db.DB.Where("id = ? AND client_id = ? AND business_id = ?", orderID, clientID, businessID).First(&order).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		if order.Status != "fulfilled" && order.Status != "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Can only review completed orders"})
			return
		}
		// Check existing review
		var count int64
		db.DB.Model(&models.Review{}).Where("order_id = ?", orderID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order already reviewed"})
			return
		}
	} else if bookingID > 0 {
		var booking models.Booking
		if err := db.DB.Where("id = ? AND client_id = ? AND business_id = ?", bookingID, clientID, businessID).First(&booking).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
			return
		}
		if booking.Status != "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Can only review completed bookings"})
			return
		}
		var count int64
		db.DB.Model(&models.Review{}).Where("booking_id = ?", bookingID).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Booking already reviewed"})
			return
		}
	}

	now := time.Now()
	review := models.Review{
		BusinessID: businessID,
		ClientID:   clientID,
		OrderID:    uintPtrOrNil(orderID),
		BookingID:  uintPtrOrNil(bookingID),
		Rating:     rating,
		Title:      title,
		Content:    content,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := db.DB.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit review"})
		return
	}

	updateBusinessRating(businessID)

	if wsHub != nil {
		if orderID > 0 {
			var order models.Order
			if db.DB.First(&order, orderID).Error == nil {
				bizCardHTML := renderBizOrderCard(db.DB, order)
				ws.BroadcastOrderUpdateFull(
					wsHub,
					strconv.Itoa(int(order.ID)),
					order.Status,
					order.PaidAmount,
					order.TotalAmount,
					0,
					true,
					int32(rating),
					bizCardHTML,
					"",
					strconv.Itoa(int(order.BusinessID)),
					strconv.Itoa(int(order.ClientID)),
				)
			}
		} else if bookingID > 0 {
			var booking models.Booking
			if db.DB.First(&booking, bookingID).Error == nil {
				bizCardHTML := renderBizBookingCard(db.DB, booking)
				ws.BroadcastBookingUpdateFull(
					wsHub,
					strconv.Itoa(int(booking.ID)),
					booking.Status,
					booking.PaidAmount,
					booking.TotalAmount,
					0,
					true,
					int32(rating),
					bizCardHTML,
					"",
					strconv.Itoa(int(booking.BusinessID)),
					strconv.Itoa(int(booking.ClientID)),
				)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"review":  review,
		"message": "Review submitted successfully!",
	})
}

func updateBusinessRating(businessID uint) {
	var avg float64
	var count int64
	db.DB.Model(&models.Review{}).Where("business_id = ?", businessID).Select("COALESCE(AVG(rating), 0)").Scan(&avg)
	db.DB.Model(&models.Review{}).Where("business_id = ?", businessID).Count(&count)
	db.DB.Model(&models.Business{}).Where("id = ?", businessID).Updates(map[string]interface{}{
		"average_rating": avg,
		"review_count":   int(count),
	})
}

func parseUint(s string, val *uint) (uint, error) {
	var v uint
	_, err := uintScan(s, &v)
	if err == nil {
		*val = v
	}
	return v, err
}

func parseInt(s string, val *int) (int, error) {
	var v int
	_, err := intScan(s, &v)
	if err == nil {
		*val = v
	}
	return v, err
}

func uintScan(s string, v *uint) (int64, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	*v = uint(n)
	return int64(n), nil
}

func intScan(s string, v *int) (int64, error) {
	n := 0
	sign := 1
	for i, c := range s {
		if i == 0 && c == '-' {
			sign = -1
			continue
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	*v = n * sign
	return int64(n), nil
}

func uintPtrOrNil(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}
