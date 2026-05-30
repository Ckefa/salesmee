package handlers

import (
	"strconv"
	"time"

	"oneflow/internal/db"
	"oneflow/internal/models"

	"github.com/gin-gonic/gin"
)

func CreateAction(c *gin.Context) {
	userID := c.GetUint("business_id")
	messageID, err := strconv.ParseUint(c.Param("message_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid message ID"})
		return
	}

	// Verify message belongs to user's conversation
	var message models.Message
	if err := db.DB.Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("messages.id = ? AND conversations.business_id = ?", messageID, userID).
		First(&message).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	actionType := c.PostForm("type")
	title := c.PostForm("title")
	description := c.PostForm("description")

	// Auto-generate title for quick actions if not provided
	if title == "" {
		switch actionType {
		case "task":
			title = "Task from customer message"
		case "reminder":
			title = "Reminder from customer message"
		case "booking":
			title = "Booking from customer message"
		default:
			title = "Action from customer message"
		}
	}

	// Auto-generate description if not provided
	if description == "" {
		description = "Created from message: " + message.Content
	}

	var dueDate *time.Time
	if dueDateStr := c.PostForm("due_date"); dueDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", dueDateStr); err == nil {
			dueDate = &parsed
		}
	}

	action := models.Action{
		MessageID:   uint(messageID),
		Type:        models.ActionType(actionType),
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		Priority:    "medium",
		Status:      "pending",
		IsCompleted: false,
	}

	if err := db.DB.Create(&action).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create action"})
		return
	}

	c.JSON(200, gin.H{"action": action})
}

func UpdateConversationStatus(c *gin.Context) {
	userID := c.GetUint("business_id")
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid conversation ID"})
		return
	}

	// Verify conversation belongs to user
	var conversation models.Conversation
	if err := db.DB.Where("id = ? AND business_id = ?", conversationID, userID).
		First(&conversation).Error; err != nil {
		c.JSON(404, gin.H{"error": "Conversation not found"})
		return
	}

	newStage := c.PostForm("stage")
	reason := c.PostForm("reason")

	// Get or create conversation progress
	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", conversationID).First(&progress).Error; err != nil {
		// Create new progress record
		progress = models.ConversationProgress{
			ConversationID: uint(conversationID),
			CurrentStage:   models.ConversationStage(newStage),
			ProgressScore:  calculateProgressScore(models.ConversationStage(newStage)),
			StageHistory: []models.StageTransition{
				{
					Stage:     models.ConversationStage(newStage),
					ChangedAt: time.Now(),
					Reason:    reason,
				},
			},
		}
		db.DB.Create(&progress)
	} else {
		// Update existing progress
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
	}

	c.JSON(200, gin.H{"progress": progress})
}

func GetActions(c *gin.Context) {
	userID := c.GetUint("business_id")

	var actions []models.Action
	if err := db.DB.Joins("JOIN messages ON actions.message_id = messages.id").
		Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("conversations.business_id = ?", userID).
		Order("actions.created_at DESC").
		Find(&actions).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to load actions"})
		return
	}

	c.JSON(200, gin.H{"actions": actions})
}

func UpdateActionStatus(c *gin.Context) {
	userID := c.GetUint("business_id")
	actionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid action ID"})
		return
	}

	// Verify action belongs to user
	var action models.Action
	if err := db.DB.Joins("JOIN messages ON actions.message_id = messages.id").
		Joins("JOIN conversations ON messages.conversation_id = conversations.id").
		Where("actions.id = ? AND conversations.business_id = ?", actionID, userID).
		First(&action).Error; err != nil {
		c.JSON(404, gin.H{"error": "Action not found"})
		return
	}

	isCompleted := c.PostForm("is_completed") == "true"
	action.IsCompleted = isCompleted

	if err := db.DB.Save(&action).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to update action"})
		return
	}

	c.JSON(200, gin.H{"action": action})
}
