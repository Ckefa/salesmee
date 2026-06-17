package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFTokenGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("csrf_token", "test-token")

	data := TemplateData(c, gin.H{})
	assert.Contains(t, data, "CSRFToken")
	assert.Equal(t, "test-token", data["CSRFToken"])
}

func TestCSRFTokenValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("csrf_token", "test-token")

	data := TemplateData(c, gin.H{"test": "value"})
	assert.Equal(t, "value", data["test"])
	assert.Contains(t, data, "CSRFToken")
	assert.Contains(t, data, "AuthType")
	assert.Contains(t, data, "Role")
}

func TestTemplateData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("auth_type", "owner")
	c.Set("role", "admin")
	c.Set("csrf_token", "test-token")

	data := TemplateData(c, gin.H{"custom": "val"})
	assert.Equal(t, "val", data["custom"])
	assert.Equal(t, "owner", data["AuthType"])
	assert.Equal(t, "admin", data["Role"])
	assert.Equal(t, "test-token", data["CSRFToken"])
}

func TestTemplateDataDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	data := TemplateData(c, nil)
	assert.Contains(t, data, "CSRFToken")
	assert.Contains(t, data, "AuthType")
	assert.Contains(t, data, "Role")
	assert.Empty(t, data["AuthType"])
	assert.Empty(t, data["Role"])
}
