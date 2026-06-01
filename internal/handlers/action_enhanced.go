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

	// Auto-advance stage based on message count
	var messageCount int64
	db.DB.Model(&models.Message{}).Where("conversation_id = ?", message.ConversationID).Count(&messageCount)

	var progress models.ConversationProgress
	if err := db.DB.Where("conversation_id = ?", message.ConversationID).First(&progress).Error; err != nil {
		return
	}

	// Simple auto-progression based on message count
	newStage := progress.CurrentStage
	if messageCount >= 8 && progress.CurrentStage == models.StageInitial {
		newStage = models.StageQualification
	} else if messageCount >= 15 && progress.CurrentStage == models.StageQualification {
		newStage = models.StageNegotiation
	}

	if newStage != progress.CurrentStage {
		progress.CurrentStage = newStage
		progress.ProgressScore = calculateProgressScore(newStage)
		db.DB.Save(&progress)
	}
}
