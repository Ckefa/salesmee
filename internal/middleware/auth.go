package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func BizzMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		token := extractToken(c)
		if token != "" {
			claims, err := services.ValidateToken(token)
			if err == nil {
				var exists bool
				db.DB.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM businesses WHERE id = ?)", claims.UserID).Scan(&exists)
				if exists {
					c.Set("business_id", claims.UserID)
					c.Set("email", claims.Email)
					c.Set("auth_type", "owner")
					c.Next()
					return
				}
				c.SetCookie("token", "", -1, "/", "", false, true)
			}
		}

		// Fallback: check team_token for staff/manager
		teamToken, _ := c.Cookie("team_token")
		if teamToken != "" {
			claims, err := services.ValidateToken(teamToken)
			if err == nil {
				var member struct{ ID uint; BusinessID uint; Role string; IsActive bool; Permissions string }
				db.DB.WithContext(ctx).Raw("SELECT id, business_id, role, is_active, permissions FROM team_members WHERE id = ?", claims.UserID).Scan(&member)
				if member.IsActive && member.BusinessID > 0 {
					c.Set("business_id", member.BusinessID)
					c.Set("team_member_id", member.ID)
					c.Set("role", member.Role)
					c.Set("auth_type", "team")
					c.Set("team_permissions", parsePermissions(member.Permissions))
					c.Next()
					return
				}
				c.SetCookie("team_token", "", -1, "/", "", false, true)
			}
		}

		c.Redirect(http.StatusFound, "/business/login")
		c.Abort()
	}
}

// DBWithTimeout runs a function with a database session that has a context timeout.
// This is a helper for handlers that want consistent timeout behavior without
// adding the import to every file.
func DBWithTimeout(c *gin.Context, timeout time.Duration) *gorm.DB {
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	return db.DB.WithContext(ctx)
}

func extractToken(c *gin.Context) string {
	// Try to get token from Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	token, err := c.Cookie("token")
	if err == nil {
		return token
	}

	return ""
}
