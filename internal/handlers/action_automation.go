package handlers

import (
	"strconv"
	"time"

	"oneflow/internal/db"
	"oneflow/internal/models"

	"github.com/gin-gonic/gin"
)

// AutoAdvanceConversationProgress automatically updates conversation stage based on action type
func AutoAdvanceConversationProgress(actionType string, currentStage models.ConversationStage) models.ConversationStage {
	switch actionType {
	case "booking":
		// Booking confirmation moves to completed stage
		if currentStage == models.StageConfirmation {
			return models.StageCompleted
		}
		return models.StageConfirmation
	case "task":
		// Task creation suggests moving from initial to qualification
		if currentStage == models.StageInitial {
			return models.StageQualification
		}
		return currentStage
	case "reminder":
		// Reminders don't change stage but may indicate follow-up needed
		if currentStage == models.StageCompleted {
			return models.StageFollowUp
		}
		return currentStage
	default:
		// Other actions don't change stage
		return currentStage
	}
}

// CreateActionWithProgress creates an action and updates conversation progress
func CreateActionWithProgress(c *gin.Context) {
	userID := c.GetUint("business_id")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(400, "Invalid message ID")
		return
	}

	// Verify message belongs to user
	var message models.Message
	if err := db.DB.Where("id = ? AND conversation_id IN (SELECT id FROM conversations WHERE client_id IN (SELECT id FROM clients WHERE business_id = ?))",
		messageID, userID).First(&message); err != nil {
		c.String(404, "Message not found")
		return
	}

	// Parse action data
	actionType := c.PostForm("type")
	title := c.PostForm("title")
	description := c.PostForm("description")
	priority := c.PostForm("priority")
	dueDateStr := c.PostForm("due_date")

	// Create action
	action := models.Action{
		MessageID:   uint(messageID),
		Type:        models.ActionType(actionType),
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if dueDateStr != "" {
		if dueDate, err := time.Parse("2006-01-02", dueDateStr); err == nil {
			action.DueDate = &dueDate
		}
	}

	if err := db.DB.Create(&action).Error; err != nil {
		c.String(500, "Failed to create action")
		return
	}

	// Get conversation and current progress
	var conversation models.Conversation
	if err := db.DB.Where("id = ?", message.ConversationID).First(&conversation); err != nil {
		c.String(404, "Conversation not found")
		return
	}

	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", conversation.ID).First(&progress); err != nil {
		// Create default progress if not exists
		progress = models.ConversationProgress{
			ConversationID: conversation.ID,
			CurrentStage:   models.StageInitial,
			ProgressScore:  10,
		}
		db.DB.Create(&progress)
	}

	// Auto-advance conversation stage
	newStage := AutoAdvanceConversationProgress(actionType, progress.CurrentStage)
	if newStage != progress.CurrentStage {
		// Update progress score based on stage
		newScore := getProgressScore(newStage)

		// Add stage transition to history
		transition := models.StageTransition{
			Stage:     newStage,
			ChangedAt: time.Now(),
			Reason:    "Auto-advanced from action: " + actionType,
		}

		progress.StageHistory = append(progress.StageHistory, transition)
		progress.CurrentStage = newStage
		progress.ProgressScore = newScore
		progress.UpdatedAt = time.Now()

		db.DB.Save(&progress)
	}

	// Return updated actions panel
	c.HTML(200, "actions_panel.html", gin.H{
		"Actions": []models.Action{action},
		"Message": message,
	})
}

// getProgressScore returns progress score for each stage
func getProgressScore(stage models.ConversationStage) int {
	switch stage {
	case models.StageInitial:
		return 10
	case models.StageQualification:
		return 25
	case models.StageNegotiation:
		return 50
	case models.StageConfirmation:
		return 75
	case models.StageInProgress:
		return 90
	case models.StageCompleted:
		return 100
	case models.StageFollowUp:
		return 85
	default:
		return 10
	}
}
