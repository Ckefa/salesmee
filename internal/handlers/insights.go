package handlers

import (
	"strconv"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
)

var tierThresholds = []struct {
	Min   int
	Label string
}{
	{0, "bronze"},
	{25, "silver"},
	{50, "gold"},
	{75, "platinum"},
	{100, "diamond"},
}

func tierLabel(score int) string {
	label := "bronze"
	for _, t := range tierThresholds {
		if score >= t.Min {
			label = t.Label
		}
	}
	return label
}

func nextTierScore(score int) int {
	for _, t := range tierThresholds {
		if score < t.Min {
			return t.Min
		}
	}
	return 100
}

const maxTierScore = 100

func CalculateConversationInsights(conversationID uint) {
	var conv models.Conversation
	if err := db.DB.First(&conv, conversationID).Error; err != nil {
		return
	}
	customerID := conv.ClientID

	tierScore := 0
	var totalOrders, completedOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ?", customerID).Count(&totalOrders)
	db.DB.Model(&models.Order{}).Where("client_id = ? AND status IN ('fulfilled','completed')", customerID).Count(&completedOrders)
	tierScore += int(completedOrders) * 15

	var pendingOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ? AND status = 'pending'", customerID).Count(&pendingOrders)
	tierScore += int(pendingOrders) * 5

	var cancelledOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ? AND status = 'cancelled'", customerID).Count(&cancelledOrders)
	tierScore -= int(cancelledOrders) * 10

	var totalBookings, completedBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ?", customerID).Count(&totalBookings)
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'completed'", customerID).Count(&completedBookings)
	tierScore += int(completedBookings) * 10

	var pendingBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'pending'", customerID).Count(&pendingBookings)
	tierScore += int(pendingBookings) * 5

	var cancelledBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'cancelled'", customerID).Count(&cancelledBookings)
	tierScore -= int(cancelledBookings) * 10

	var paidTransactions int64
	db.DB.Model(&models.Payment{}).Where("(order_id IN (SELECT id FROM orders WHERE client_id = ?) OR booking_id IN (SELECT id FROM bookings WHERE client_id = ?)) AND status = 'completed'", customerID, customerID).Count(&paidTransactions)
	tierScore += int(paidTransactions) * 5

	var msgCount int64
	db.DB.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&msgCount)
	tierScore += int(msgCount) * 2
	if tierScore > maxTierScore {
		tierScore = maxTierScore
	}
	if tierScore < 0 {
		tierScore = 0
	}

	activityScore := 0
	var recentOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ? AND created_at > ?", customerID, time.Now().AddDate(0, 0, -30)).Count(&recentOrders)
	activityScore += int(recentOrders)

	var recentBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND created_at > ?", customerID, time.Now().AddDate(0, 0, -30)).Count(&recentBookings)
	activityScore += int(recentBookings)

	var recentMsgs int64
	db.DB.Model(&models.Message{}).Where("conversation_id = ? AND created_at > ?", conversationID, time.Now().AddDate(0, 0, -30)).Count(&recentMsgs)
	if recentMsgs >= 10 {
		activityScore += 2
	} else if recentMsgs >= 5 {
		activityScore += 1
	}

	if activityScore > 5 {
		activityScore = 5
	}

	behaviorTrend := "inactive"
	if tierScore >= 50 && activityScore >= 3 {
		behaviorTrend = "high_value"
	} else if activityScore >= 2 {
		behaviorTrend = "active"
	}

	engagementScore := 0
	var monthlyMessages int64
	db.DB.Model(&models.Message{}).Where("conversation_id = ? AND created_at > ?", conversationID, time.Now().AddDate(0, 0, -30)).Count(&monthlyMessages)
	if monthlyMessages >= 20 {
		engagementScore = 100
	} else if monthlyMessages >= 10 {
		engagementScore = 75
	} else if monthlyMessages >= 5 {
		engagementScore = 50
	} else if monthlyMessages >= 2 {
		engagementScore = 25
	}

	var orderSpent, bookingSpent float64
	db.DB.Model(&models.Order{}).Select("COALESCE(SUM(total_amount), 0)").Where("client_id = ? AND status IN ('fulfilled','completed')", customerID).Scan(&orderSpent)
	db.DB.Model(&models.Booking{}).Select("COALESCE(SUM(total_amount), 0)").Where("client_id = ? AND status = 'completed'", customerID).Scan(&bookingSpent)
	totalSpent := orderSpent + bookingSpent

	lastActive := time.Now()
	var lastMsg models.Message
	if err := db.DB.Where("conversation_id = ?", conversationID).Order("created_at DESC").First(&lastMsg).Error; err == nil {
		lastActive = lastMsg.CreatedAt
	} else {
		var lastOrder models.Order
		if err := db.DB.Where("client_id = ?", customerID).Order("created_at DESC").First(&lastOrder).Error; err == nil {
			lastActive = lastOrder.CreatedAt
		}
	}

	var insight models.CustomerInsight
	if err := db.DB.Where("conversation_id = ?", conversationID).First(&insight).Error; err != nil {
		insight = models.CustomerInsight{
			ConversationID: conversationID,
			CustomerID:     customerID,
		}
	}

	insight.Tier = tierLabel(tierScore)
	insight.TierScore = tierScore
	insight.ActivityScore = activityScore
	insight.BehaviorTrend = behaviorTrend
	insight.EngagementScore = engagementScore
	insight.TotalOrders = int(totalOrders)
	insight.PendingOrders = int(pendingOrders)
	insight.CompletedOrders = int(completedOrders)
	insight.TotalBookings = int(totalBookings)
	insight.PendingBookings = int(pendingBookings)
	insight.CompletedBookings = int(completedBookings)
	insight.TotalSpent = totalSpent
	insight.TotalMessages = int(msgCount)
	insight.LastActiveAt = &lastActive
	insight.UpdatedAt = time.Now()

	db.DB.Save(&insight)
}

func GetConversationInsightsBadge(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.Status(400)
		return
	}

	CalculateConversationInsights(uint(conversationID))
	var insight models.CustomerInsight
	db.DB.Where("conversation_id = ?", conversationID).First(&insight)

	c.HTML(200, "insights_badge", gin.H{
		"Insight": insight,
	})
}

func GetConversationInsightsPanel(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.Status(400)
		return
	}

	CalculateConversationInsights(uint(conversationID))
	var insight models.CustomerInsight
	db.DB.Where("conversation_id = ?", conversationID).First(&insight)

	businessID := c.GetUint("business_id")
	var business models.Business
	db.DB.First(&business, businessID)

	c.HTML(200, "insights_panel", gin.H{
		"Insight":  insight,
		"Business": business,
	})
}

func RefreshConversationInsights(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.Status(400)
		return
	}

	CalculateConversationInsights(uint(conversationID))

	var insight models.CustomerInsight
	db.DB.Where("conversation_id = ?", conversationID).First(&insight)

	c.HTML(200, "insights_badge", gin.H{
		"Insight": insight,
	})
}
