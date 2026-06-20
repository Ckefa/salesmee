package handlers

import (
	"net/http"
	"strconv"

	"salesmee/internal/models"
	"salesmee/internal/services/progress"

	"github.com/gin-gonic/gin"
)

func GetConversationInsightsBadge(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.Status(400)
		return
	}

	progress.CalculateConversationInsights(uint(conversationID))
	var insight models.CustomerInsight
	dbc(c).Where("conversation_id = ?", conversationID).First(&insight)

	c.HTML(http.StatusOK, "insights_badge", gin.H{
		"Insight": insight,
	})
}

func GetConversationInsightsPanel(c *gin.Context) {
	conversationID, err := strconv.ParseUint(c.Param("conversation_id"), 10, 32)
	if err != nil {
		c.Status(400)
		return
	}

	progress.CalculateConversationInsights(uint(conversationID))
	var insight models.CustomerInsight
	dbc(c).Where("conversation_id = ?", conversationID).First(&insight)

	businessID := c.GetUint("business_id")
	var business models.Business
	dbc(c).First(&business, businessID)

	c.HTML(http.StatusOK, "insights_panel", gin.H{
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

	progress.CalculateConversationInsights(uint(conversationID))

	var insight models.CustomerInsight
	dbc(c).Where("conversation_id = ?", conversationID).First(&insight)

	c.HTML(http.StatusOK, "insights_badge", gin.H{
		"Insight": insight,
	})
}
