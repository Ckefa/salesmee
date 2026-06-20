package business

import (
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/onboarding"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
)

func (h *BusinessHandler) CheckOnboardingProgress(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusOK, gin.H{"advanced": false, "message": ""})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"advanced": false, "message": ""})
		return
	}

	prevStep := business.OnboardingStep
	data, err := onboarding.DetectStep(h.db, &business)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"advanced": false, "message": ""})
		return
	}

	if data.Completed {
		ws.BroadcastOnboardingUpdate(h.hub, bizIDStr(c), data.Step, data.TotalSteps, true)
		c.JSON(http.StatusOK, gin.H{"completed": true})
		return
	}

	if data.Step > prevStep {
		ws.BroadcastOnboardingUpdate(h.hub, bizIDStr(c), data.Step, data.TotalSteps, false)
		c.JSON(http.StatusOK, gin.H{"advanced": true, "step": data.Step})
		return
	}

	c.JSON(http.StatusOK, gin.H{"advanced": false, "message": "Not quite yet — complete the current step first!"})
}

func (h *BusinessHandler) GetOnboardingStatus(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusOK, gin.H{"completed": true})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"completed": true})
		return
	}

	data, err := onboarding.DetectStep(h.db, &business)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"completed": true})
		return
	}

	if data.Completed {
		c.JSON(http.StatusOK, gin.H{"completed": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"step":        data.Step,
		"total_steps": data.TotalSteps,
		"completed":   false,
	})
}

func (h *BusinessHandler) AdvanceOnboarding(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not authenticated"})
		return
	}

	data, err := onboarding.AdvanceStep(h.db, businessID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to advance"})
		return
	}

	ws.BroadcastOnboardingUpdate(h.hub, bizIDStr(c), data.Step, data.TotalSteps, data.Completed)

	c.JSON(http.StatusOK, gin.H{
		"step":        data.Step,
		"total_steps": data.TotalSteps,
		"completed":   data.Completed,
	})
}

func (h *BusinessHandler) SkipOnboarding(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not authenticated"})
		return
	}

	if err := onboarding.SkipOnboarding(h.db, businessID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to skip onboarding"})
		return
	}

	ws.BroadcastOnboardingUpdate(h.hub, bizIDStr(c), 0, 0, true)

	c.JSON(http.StatusOK, gin.H{"completed": true})
}
