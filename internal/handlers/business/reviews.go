package business

import (
	"math"
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewStats struct {
	AverageRating float64
	ReviewCount   int
	Rating5Count  int
	Rating4Count  int
	Rating3Count  int
	Rating2Count  int
	Rating1Count  int
}

func (h *ReviewHandler) GetReviews(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	var reviews []models.Review
	h.dbc(c).Where("business_id = ?", businessID).
		Preload("Client").
		Order("created_at DESC").
		Find(&reviews)

	stats := computeReviewStats(h.db, businessID)

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/reviews_content", gin.H{
			"Business":       currentBusiness,
			"ActivePage":     "reviews",
			"Reviews":        reviews,
			"AverageRating":  math.Round(stats.AverageRating*10) / 10,
			"ReviewCount":    stats.ReviewCount,
			"Rating5Count":   stats.Rating5Count,
			"Rating4Count":   stats.Rating4Count,
			"Rating3Count":   stats.Rating3Count,
			"Rating2Count":   stats.Rating2Count,
			"Rating1Count":   stats.Rating1Count,
			"Onboarding":     onboardingData(h.db, businessID),
			"AuthType":       c.GetString("auth_type"),
			"Role":           c.GetString("role"),
		})
		return
	}

	c.HTML(http.StatusOK, "reviews.html", gin.H{
		"Business":       currentBusiness,
		"ActivePage":     "reviews",
		"Reviews":        reviews,
		"AverageRating":  math.Round(stats.AverageRating*10) / 10,
		"ReviewCount":    stats.ReviewCount,
		"Rating5Count":   stats.Rating5Count,
		"Rating4Count":   stats.Rating4Count,
		"Rating3Count":   stats.Rating3Count,
		"Rating2Count":   stats.Rating2Count,
		"Rating1Count":   stats.Rating1Count,
		"Onboarding":     onboardingData(h.db, businessID),
		"AuthType":       c.GetString("auth_type"),
		"Role":           c.GetString("role"),
		"AssistEnabled":  assist.IsEnabled(),
	})
}

func (h *ReviewHandler) ReplyToReview(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	reviewID := c.Param("id")
	var review models.Review
	if err := h.dbc(c).Where("id = ? AND business_id = ?", reviewID, businessID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	reply := c.PostForm("reply")
	if reply == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reply is required"})
		return
	}

	now := time.Now()
	review.Reply = reply
	review.ReplyAt = &now
	review.UpdatedAt = now

	if err := h.dbc(c).Save(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reply"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"reply":    reply,
		"reply_at": now,
		"message":  "Reply posted successfully!",
	})
}

func computeReviewStats(d *gorm.DB, businessID uint) ReviewStats {
	var s ReviewStats

	d.Model(&models.Review{}).Where("business_id = ?", businessID).
		Select("COALESCE(AVG(rating), 0)").Scan(&s.AverageRating)

	var total, c5, c4, c3, c2, c1 int64
	d.Model(&models.Review{}).Where("business_id = ?", businessID).Count(&total)
	d.Model(&models.Review{}).Where("business_id = ? AND rating = 5", businessID).Count(&c5)
	d.Model(&models.Review{}).Where("business_id = ? AND rating = 4", businessID).Count(&c4)
	d.Model(&models.Review{}).Where("business_id = ? AND rating = 3", businessID).Count(&c3)
	d.Model(&models.Review{}).Where("business_id = ? AND rating = 2", businessID).Count(&c2)
	d.Model(&models.Review{}).Where("business_id = ? AND rating = 1", businessID).Count(&c1)
	s.ReviewCount = int(total)
	s.Rating5Count = int(c5)
	s.Rating4Count = int(c4)
	s.Rating3Count = int(c3)
	s.Rating2Count = int(c2)
	s.Rating1Count = int(c1)

	return s
}
