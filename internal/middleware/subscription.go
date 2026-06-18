package middleware

import (
	"net/http"
	"salesmee/internal/db"
	"salesmee/internal/services/notifier"
	"salesmee/internal/services/subscription"

	"github.com/gin-gonic/gin"
)

func RequireFeature(feature string, featureName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		businessID := c.GetUint("business_id")
		if businessID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if subscription.HasFeature(businessID, feature) {
			c.Next()
			return
		}

		planName := subscription.PlanDisplayName(businessID)
		msg := featureName + " is not available on your " + planName + " plan. Upgrade to unlock it."

		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "components/upgrade_prompt", gin.H{
				"Feature":    feature,
				"FeatureMsg": msg,
				"PlanName":   planName,
			})
			c.Abort()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":          msg,
			"requires_upgrade": true,
			"upgrade_url":    "/business/subscription#plans",
		})
		c.Abort()
	}
}

func CheckResourceLimit(resource string, label string) gin.HandlerFunc {
	return func(c *gin.Context) {
		businessID := c.GetUint("business_id")
		if businessID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		grace := c.Query("grace") == "1"

		if grace {
			subscription.UseGrace(businessID, resource)
			c.Next()
			return
		}

		check := subscription.CheckResourceLimit(businessID, resource)
		if check.Allowed {
			c.Next()
			return
		}

		if !check.GraceAllowed {
			notifier.NotifyLimitReached(db.DB, businessID, resource, label, check.Current, check.Max)
			c.JSON(http.StatusForbidden, gin.H{
				"error":            check.Message,
				"current":          check.Current,
				"max":              check.Max,
				"requires_upgrade": true,
"upgrade_url":    "/business/subscription#plans",
		})
		c.Abort()
		return
	}

	planName := subscription.PlanDisplayName(businessID)
	used := check.GraceUsed
	msg := check.Message
	if !used {
		msg = "You've reached the " + label + " limit for your " + planName + " plan (" + itoa(check.Current) + " of " + itoa(check.Max) + "). You have one grace action remaining."
	} else {
		msg = "You've reached the " + label + " limit for your " + planName + " plan (" + itoa(check.Current) + " of " + itoa(check.Max) + "). Please upgrade to add more."
	}

	c.JSON(http.StatusConflict, gin.H{
		"error":            msg,
		"current":          check.Current,
		"max":              check.Max,
		"limit_reached":    true,
		"grace_allowed":    !used,
		"requires_upgrade": true,
		"upgrade_url":      "/business/subscription#plans",
		})
		c.Abort()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	r := ""
	for n > 0 {
		r = string(rune('0'+n%10)) + r
		n /= 10
	}
	return r
}
