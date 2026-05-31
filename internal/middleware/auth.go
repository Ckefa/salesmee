package middleware

import (
	"net/http"
	"strings"

	"oneflow/internal/services"

	"github.com/gin-gonic/gin"
)

func BizzMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.Redirect(http.StatusFound, "/business/login")
			c.Abort()
			return
		}

		claims, err := services.ValidateToken(token)
		if err != nil {
			c.Redirect(http.StatusFound, "/business/login")
			c.Abort()
			return
		}

		c.Set("business_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
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
