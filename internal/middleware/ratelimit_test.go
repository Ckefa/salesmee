package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"/business/login", true},
		{"/business/forgot-password", true},
		{"/business/dashboard", false},
		{"/client/send-otp", true},
		{"/client/verify-otp/123", true},
		{"/api/connect/some-slug", true},
		{"/business/register/new", true},
		{"/business/team/login", true},
		{"/business/orders", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expect, isRateLimited(tt.path))
		})
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    10,
		burst:   5,
	}

	for i := 0; i < 5; i++ {
		assert.True(t, rl.allow("127.0.0.1"), "request %d should be allowed", i+1)
	}

	assert.False(t, rl.allow("127.0.0.1"), "burst exceeded")
}

func TestRateLimiterAllowDifferentIPs(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    10,
		burst:   3,
	}

	for i := 0; i < 3; i++ {
		assert.True(t, rl.allow("10.0.0.1"))
	}
	assert.False(t, rl.allow("10.0.0.1"))

	assert.True(t, rl.allow("10.0.0.2"))
	assert.True(t, rl.allow("10.0.0.3"))
}

func TestRateLimiterRefill(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    60,
		burst:   60,
	}

	for i := 0; i < 60; i++ {
		rl.allow("10.0.0.1")
	}

	assert.False(t, rl.allow("10.0.0.1"))

	time.Sleep(50 * time.Millisecond)

	assert.True(t, rl.allow("10.0.0.1"), "should have refilled at least 1 token")
}

func TestRateLimitAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RateLimitAuth()
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/", nil)

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitGlobal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RateLimitGlobal()
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/business/login", nil)

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RateLimitAPI()
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitGlobalNonRatelimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := RateLimitGlobal()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/business/dashboard", nil)

	handler(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiterCleanup(t *testing.T) {
	old := time.Second
	rl := &rateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     10,
		burst:    10,
		cleanup:  old,
	}

	rl.allow("10.0.0.1")
	rl.allow("10.0.0.2")

	assert.Len(t, rl.buckets, 2)

	rl.buckets["10.0.0.1"].lastCheck = time.Now().Add(-2 * time.Hour)

	rl.mu.Lock()
	now := time.Now()
	for ip, bucket := range rl.buckets {
		if now.Sub(bucket.lastCheck) > rl.cleanup {
			delete(rl.buckets, ip)
		}
	}
	rl.mu.Unlock()

	assert.Len(t, rl.buckets, 1)
	_, exists := rl.buckets["10.0.0.2"]
	assert.True(t, exists)
}

func TestRateLimiterConcurrency(t *testing.T) {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    1000,
		burst:   100,
	}

	done := make(chan bool)
	for i := 0; i < 50; i++ {
		go func(n int) {
			rl.allow(fmt.Sprintf("10.0.0.%d", n%10))
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}
