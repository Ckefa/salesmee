package progress

import (
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"
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

// isBefore returns true if stage a is earlier in the pipeline than stage b.
func isBefore(a, b models.ConversationStage) bool {
	order := map[models.ConversationStage]int{
		models.StageInitial:       0,
		models.StageQualification: 1,
		models.StageNegotiation:   2,
		models.StageConfirmation:  3,
		models.StageInProgress:    4,
		models.StageCompleted:     5,
		models.StageFollowUp:      6,
	}
	return order[a] < order[b]
}

func CalculateProgressScore(stage models.ConversationStage) int {
	scores := map[models.ConversationStage]int{
		models.StageInitial:       10,
		models.StageQualification: 25,
		models.StageNegotiation:   50,
		models.StageConfirmation:  75,
		models.StageInProgress:    90,
		models.StageCompleted:     100,
		models.StageFollowUp:      100,
	}

	if score, exists := scores[stage]; exists {
		return score
	}
	return 0
}

// AutoCalculateProgress analyzes conversation activity and forward-advances the stage.
// Never downgrades — only moves forward in the pipeline.
func AutoCalculateProgress(conversationID uint) {
	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", conversationID).First(&progress).Error; err != nil {
		return
	}

	var conversation models.Conversation
	if err := db.DB.First(&conversation, conversationID).Error; err != nil {
		return
	}
	clientID := conversation.ClientID

	newStage := progress.CurrentStage
	reason := ""

	var completedOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ? AND status IN ('fulfilled','completed')", clientID).Count(&completedOrders)

	var completedBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'completed'", clientID).Count(&completedBookings)

	if completedOrders > 0 || completedBookings > 0 {
		if isBefore(newStage, models.StageCompleted) {
			newStage = models.StageCompleted
			reason = "auto: completed order(s) or booking(s) exist"
		}
	} else {
		var confirmedOrders int64
		db.DB.Model(&models.Order{}).Where("client_id = ? AND status IN ('confirmed','client_confirmed')", clientID).Count(&confirmedOrders)

		var confirmedBookings int64
		db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'client_confirmed'", clientID).Count(&confirmedBookings)

		if confirmedOrders > 0 || confirmedBookings > 0 {
			if isBefore(newStage, models.StageConfirmation) {
				newStage = models.StageConfirmation
				reason = "auto: confirmed order(s) or booking(s) exist"
			}
		} else {
			var pendingOrders int64
			db.DB.Model(&models.Order{}).Where("client_id = ? AND status = 'pending'", clientID).Count(&pendingOrders)

			var pendingBookings int64
			db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'pending'", clientID).Count(&pendingBookings)

			if pendingOrders > 0 || pendingBookings > 0 {
				if isBefore(newStage, models.StageNegotiation) {
					newStage = models.StageNegotiation
					reason = "auto: pending order(s) or booking(s) exist"
				}
			} else {
				var msgCount int64
				db.DB.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&msgCount)

				if msgCount >= 5 && isBefore(newStage, models.StageQualification) {
					newStage = models.StageQualification
					reason = "auto: 5+ messages in conversation"
				}
			}
		}
	}

	if isBefore(newStage, models.StageCompleted) {
		var paidOrders int64
		db.DB.Model(&models.Order{}).Where("client_id = ? AND paid_amount > 0", clientID).Count(&paidOrders)

		var bookingPayments int64
		db.DB.Model(&models.Booking{}).Where("client_id = ? AND paid_amount > 0", clientID).Count(&bookingPayments)

		if paidOrders > 0 || bookingPayments > 0 {
			if isBefore(newStage, models.StageInProgress) {
				newStage = models.StageInProgress
				reason = "auto: payment(s) received"
			}
		}
	}

	if newStage == models.StageCompleted || progress.CurrentStage == models.StageCompleted {
		var lastMsg models.Message
		if err := db.DB.Where("conversation_id = ?", conversationID).Order("created_at DESC").First(&lastMsg).Error; err == nil {
			if time.Since(lastMsg.CreatedAt).Hours() > 336 && progress.CurrentStage == models.StageCompleted {
				if isBefore(newStage, models.StageFollowUp) {
					newStage = models.StageFollowUp
					reason = "auto: 14+ days since last message after completion"
				}
			}
		}
	}

	if newStage != progress.CurrentStage {
		transition := models.StageTransition{
			Stage:     newStage,
			ChangedAt: time.Now(),
			Reason:    reason,
		}
		if len(progress.StageHistory) > 0 {
			lastTransition := progress.StageHistory[len(progress.StageHistory)-1]
			transition.Duration = int(time.Since(lastTransition.ChangedAt).Hours())
		}

		progress.StageHistory = append(progress.StageHistory, transition)
		progress.CurrentStage = newStage
		progress.ProgressScore = CalculateProgressScore(newStage)

		if newStage == models.StageConfirmation {
			expectedClose := time.Now().AddDate(0, 0, 7)
			progress.ExpectedClose = &expectedClose
		}
		if newStage == models.StageCompleted {
			now := time.Now()
			progress.ActualClose = &now
		}

		db.DB.Save(&progress)
	}

	CalculateConversationInsights(conversationID)
}
