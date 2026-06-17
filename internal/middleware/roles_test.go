package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPermissionConstants(t *testing.T) {
	assert.Equal(t, Permission("orders:r"), PermOrdersRead)
	assert.Equal(t, Permission("orders:rw"), PermOrdersWrite)
	assert.Equal(t, Permission("bookings:r"), PermBookingsRead)
	assert.Equal(t, Permission("bookings:rw"), PermBookingsWrite)
	assert.Equal(t, Permission("clients:r"), PermClientsRead)
	assert.Equal(t, Permission("clients:rw"), PermClientsWrite)
	assert.Equal(t, Permission("products:rw"), PermProductsWrite)
	assert.Equal(t, Permission("services:rw"), PermServicesWrite)
	assert.Equal(t, Permission("payments:rw"), PermPaymentsWrite)
	assert.Equal(t, Permission("analytics:view"), PermAnalyticsView)
	assert.Equal(t, Permission("reports:view"), PermReportsView)
	assert.Equal(t, Permission("locations:view"), PermLocationsView)
}

func TestHasPermissionOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "owner")

	assert.True(t, HasPermission(c, PermOrdersRead))
	assert.True(t, HasPermission(c, PermOrdersWrite))
	assert.True(t, HasPermission(c, PermPaymentsWrite))
}

func TestHasPermissionTeamWithAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "team")
	c.Set("team_permissions", map[string]bool{
		"orders:rw": true,
		"clients:r": true,
	})

	assert.True(t, HasPermission(c, PermOrdersRead))
	assert.True(t, HasPermission(c, PermOrdersWrite))
	assert.True(t, HasPermission(c, PermClientsRead))
	assert.False(t, HasPermission(c, PermBookingsRead))
}

func TestHasPermissionTeamWithoutAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "team")
	c.Set("team_permissions", map[string]bool{
		"orders:rw": false,
	})

	assert.False(t, HasPermission(c, PermOrdersRead))
}

func TestHasPermissionNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	assert.False(t, HasPermission(c, PermOrdersRead))
}

func TestHasPermissionWriteImpliesRead(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "team")
	c.Set("team_permissions", map[string]bool{
		"orders:rw": true,
	})

	assert.True(t, HasPermission(c, Permission("orders:r")))
}

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequirePermission(PermOrdersRead)
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "owner")

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequirePermission(PermPaymentsWrite)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "team")
	c.Set("team_permissions", map[string]bool{})

	handler(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireOwner()
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "owner")

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireOwnerDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RequireOwner()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "team")

	handler(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestParsePermissions(t *testing.T) {
	perms := parsePermissions(`{"orders:rw":true,"clients:r":false}`)
	assert.NotNil(t, perms)
	assert.True(t, perms["orders:rw"])
	assert.False(t, perms["clients:r"])
}

func TestParsePermissionsInvalid(t *testing.T) {
	perms := parsePermissions(`invalid json`)
	assert.Nil(t, perms)
}

func TestParsePermissionsEmpty(t *testing.T) {
	perms := parsePermissions(`{}`)
	assert.NotNil(t, perms)
	assert.Empty(t, perms)
}
