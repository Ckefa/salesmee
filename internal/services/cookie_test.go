package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetSecureCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	SetSecureCookie(c, "test_cookie", "test_value", 86400, "/")

	cookies := w.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "test_cookie" {
			found = true
			assert.Equal(t, "test_value", cookie.Value)
			assert.True(t, cookie.HttpOnly)
			break
		}
	}
	assert.True(t, found, "cookie should be set")
}

func TestClearCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	ClearCookie(c, "test_cookie", "/")

	cookies := w.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == "test_cookie" {
			found = true
			assert.Equal(t, "", cookie.Value)
			assert.True(t, cookie.HttpOnly)
			break
		}
	}
	assert.True(t, found, "cookie should be cleared")
}
