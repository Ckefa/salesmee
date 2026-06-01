package middleware

import (
	"net/http"
	"salesmee/internal/services/subscription"

	"github.com/gin-gonic/gin"
)

func RequireFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		businessID := c.GetUint("business_id")
		if businessID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if !subscription.HasFeature(businessID, feature) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your current plan does not include this feature. Please upgrade to access it.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func CheckResourceLimit(resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		businessID := c.GetUint("business_id")
		if businessID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		check := subscription.CheckResourceLimit(businessID, resource)
		if !check.Allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   check.Message,
				"current": check.Current,
				"max":     check.Max,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
