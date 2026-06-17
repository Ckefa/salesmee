package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     int
	burst    int
	cleanup  time.Duration
}

type tokenBucket struct {
	tokens    int
	lastCheck time.Time
}

var (
	authLimiter = &rateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     5,
		burst:    10,
		cleanup:  time.Hour,
	}
	apiLimiter = &rateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     60,
		burst:    100,
		cleanup:  time.Hour,
	}
)

func init() {
	go authLimiter.cleanupLoop()
	go apiLimiter.cleanupLoop()
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[ip]
	now := time.Now()

	if !exists {
		bucket = &tokenBucket{
			tokens:    rl.burst,
			lastCheck: now,
		}
		rl.buckets[ip] = bucket
	}

	elapsed := now.Sub(bucket.lastCheck)
	bucket.lastCheck = now

	// Refill tokens
	refill := int(elapsed.Seconds() * float64(rl.rate))
	if refill > 0 {
		bucket.tokens += refill
		if bucket.tokens > rl.burst {
			bucket.tokens = rl.burst
		}
	}

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			if now.Sub(bucket.lastCheck) > rl.cleanup {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

var rateLimitedPaths = map[string]bool{
	"/business/login":           true,
	"/business/forgot-password": true,
}

var rateLimitedPrefixes = []string{
	"/client/send-otp",
	"/client/verify-otp",
	"/api/connect/",
	"/business/register",
	"/business/team/login",
}

func isRateLimited(path string) bool {
	if rateLimitedPaths[path] {
		return true
	}
	for _, prefix := range rateLimitedPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func RateLimitGlobal() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" && isRateLimited(c.Request.URL.Path) {
			ip := c.ClientIP()
			if !authLimiter.allow(ip) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Too many requests. Please try again later.",
				})
				return
			}
		}
		c.Next()
	}
}

func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authLimiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}

func RateLimitAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !apiLimiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
