package handlers

import (
	"strconv"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowActionModal(c *gin.Context) {
	messageID := c.Param("message_id")

	// Get message details
	var message models.Message
	if err := db.DB.First(&message, messageID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	c.HTML(200, "action_modal.html", gin.H{
		"Message": message,
	})
}

func CreateEnhancedAction(c *gin.Context) {
	messageID, _ := strconv.ParseUint(c.PostForm("message_id"), 10, 32)
	actionType := c.PostForm("type")
	title := c.PostForm("title")
	description := c.PostForm("description")
	priority := c.PostForm("priority")
	assignedTo := c.PostForm("assigned_to")

	// Parse dates
	var dueDate *time.Time
	if dueDateStr := c.PostForm("due_date"); dueDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dueDateStr); err == nil {
			dueDate = &parsed
		}
	}

	// Parse costs
	var estimatedCost float64
	if costStr := c.PostForm("estimated_cost"); costStr != "" {
		if parsed, err := strconv.ParseFloat(costStr, 64); err == nil {
			estimatedCost = parsed
		}
	}

	action := models.Action{
		MessageID:     uint(messageID),
		Type:          models.ActionType(actionType),
		Title:         title,
		Description:   description,
		DueDate:       dueDate,
		Priority:      priority,
		Status:        "pending",
		AssignedTo:    assignedTo,
		EstimatedCost: estimatedCost,
		Notes:         c.PostForm("notes"),
	}

	if err := db.DB.Create(&action).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create action"})
		return
	}

	// Update conversation progress
	updateConversationProgress(uint(messageID))

	c.JSON(200, gin.H{"success": true, "action": action})
}

func GetEnhancedActions(c *gin.Context) {
	userID := c.GetUint("business_id")

	var actions []models.Action
	err := db.DB.Joins("JOIN messages ON actions.message_id = messages.id").
		Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("conversations.business_id = ?", userID).
		Preload("Message").
		Order("actions.created_at DESC").
		Find(&actions).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to load actions"})
		return
	}

	c.JSON(200, gin.H{"actions": actions})
}

func UpdateEnhancedActionStatus(c *gin.Context) {
	actionID := c.Param("id")
	status := c.PostForm("status")

	if err := db.DB.Model(&models.Action{}).Where("id = ?", actionID).Update("status", status).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to update action"})
		return
	}

	// Mark as completed if status is completed
	if status == "completed" {
		db.DB.Model(&models.Action{}).Where("id = ?", actionID).Update("is_completed", true)
	}

	c.JSON(200, gin.H{"success": true})
}

func GetConversationProgress(c *gin.Context) {
	conversationID := c.Param("conversation_id")

	var progress models.ConversationProgress
	err := db.DB.Where("conversation_id = ?", conversationID).First(&progress).Error

	if err != nil {
		// Create initial progress if not exists
		convID, _ := strconv.ParseUint(conversationID, 10, 32)
		progress = models.ConversationProgress{
			ConversationID: uint(convID),
			CurrentStage:   models.StageInitial,
			ProgressScore:  10,
			StageHistory: []models.StageTransition{
				{
					Stage:     models.StageInitial,
					ChangedAt: time.Now(),
					Reason:    "Conversation started",
				},
			},
		}
		db.DB.Create(&progress)
	}

	c.JSON(200, gin.H{"progress": progress})
}

func UpdateConversationStage(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	newStage := c.PostForm("stage")
	reason := c.PostForm("reason")

	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", conversationID).First(&progress).Error; err != nil {
		c.JSON(404, gin.H{"error": "Progress not found"})
		return
	}

	// Add stage transition
	transition := models.StageTransition{
		Stage:     models.ConversationStage(newStage),
		ChangedAt: time.Now(),
		Reason:    reason,
	}

	// Calculate duration in previous stage
	if len(progress.StageHistory) > 0 {
		lastTransition := progress.StageHistory[len(progress.StageHistory)-1]
		duration := int(time.Since(lastTransition.ChangedAt).Hours())
		transition.Duration = duration
	}

	progress.StageHistory = append(progress.StageHistory, transition)
	progress.CurrentStage = models.ConversationStage(newStage)

	// Update progress score based on stage
	progress.ProgressScore = calculateProgressScore(models.ConversationStage(newStage))

	// Set expected close date if in confirmation stage
	if newStage == string(models.StageConfirmation) {
		expectedClose := time.Now().AddDate(0, 0, 7) // 7 days from confirmation
		progress.ExpectedClose = &expectedClose
	}

	// Mark as completed
	if newStage == string(models.StageCompleted) {
		now := time.Now()
		progress.ActualClose = &now
	}

	db.DB.Save(&progress)
	c.JSON(200, gin.H{"success": true, "progress": progress})
}

func calculateProgressScore(stage models.ConversationStage) int {
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

func updateConversationProgress(messageID uint) {
	// Get conversation from message
	var message models.Message
	db.DB.Preload("Conversation").First(&message, messageID)
	AutoCalculateProgress(message.ConversationID)
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

	// 1. Check for fulfilled/completed orders → StageCompleted
	var completedOrders int64
	db.DB.Model(&models.Order{}).Where("client_id = ? AND status IN ('fulfilled','completed')", clientID).Count(&completedOrders)

	// 2. Check for completed bookings → StageCompleted
	var completedBookings int64
	db.DB.Model(&models.Booking{}).Where("client_id = ? AND status = 'completed'", clientID).Count(&completedBookings)

	if completedOrders > 0 || completedBookings > 0 {
		if isBefore(newStage, models.StageCompleted) {
			newStage = models.StageCompleted
			reason = "auto: completed order(s) or booking(s) exist"
		}
	} else {
		// 3. Check for confirmed orders / client_confirmed bookings → StageConfirmation
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
			// 4. Check for pending orders/bookings → StageNegotiation
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
				// 5. Check message count → StageQualification
				var msgCount int64
				db.DB.Model(&models.Message{}).Where("conversation_id = ?", conversationID).Count(&msgCount)

				if msgCount >= 5 && isBefore(newStage, models.StageQualification) {
					newStage = models.StageQualification
					reason = "auto: 5+ messages in conversation"
				}
			}
		}
	}

	// 6. Payments received → StageInProgress (overrides earlier stages but not completed)
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

	// 7. 14+ days idle after completion → StageFollowUp
	if newStage == models.StageCompleted || progress.CurrentStage == models.StageCompleted {
		var lastMsg models.Message
		if err := db.DB.Where("conversation_id = ?", conversationID).Order("created_at DESC").First(&lastMsg).Error; err == nil {
			if time.Since(lastMsg.CreatedAt).Hours() > 336 && progress.CurrentStage == models.StageCompleted { // 14 days
				if isBefore(newStage, models.StageFollowUp) {
					newStage = models.StageFollowUp
					reason = "auto: 14+ days since last message after completion"
				}
			}
		}
	}

	// Apply change if stage advanced
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
		progress.ProgressScore = calculateProgressScore(newStage)

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
