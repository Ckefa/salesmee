package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Permission string

const (
	PermOrdersRead    Permission = "orders:r"
	PermOrdersWrite   Permission = "orders:rw"
	PermBookingsRead  Permission = "bookings:r"
	PermBookingsWrite Permission = "bookings:rw"
	PermClientsRead   Permission = "clients:r"
	PermClientsWrite  Permission = "clients:rw"
	PermProductsWrite Permission = "products:rw"
	PermServicesWrite Permission = "services:rw"
	PermPaymentsWrite Permission = "payments:rw"
	PermAnalyticsView Permission = "analytics:view"
	PermReportsView   Permission = "reports:view"
	PermLocationsView Permission = "locations:view"
)

func HasPermission(c *gin.Context, perm Permission) bool {
	authType := c.GetString("auth_type")
	if authType == "owner" {
		return true
	}
	if authType != "team" {
		return false
	}

	perms, exists := c.Get("team_permissions")
	if !exists {
		return false
	}
	permMap, ok := perms.(map[string]bool)
	if !ok || permMap == nil {
		return false
	}

	if permMap[string(perm)] {
		return true
	}

	permStr := string(perm)
	if strings.HasSuffix(permStr, ":r") {
		writePerm := strings.TrimSuffix(permStr, ":r") + ":rw"
		if permMap[writePerm] {
			return true
		}
	}

	return false
}

func RequirePermission(perm Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasPermission(c, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}
		c.Next()
	}
}

func RequireOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("auth_type") != "owner" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Owner access required"})
			return
		}
		c.Next()
	}
}

func parsePermissions(permJSON string) map[string]bool {
	var perms map[string]bool
	if err := json.Unmarshal([]byte(permJSON), &perms); err != nil {
		return nil
	}
	return perms
}
