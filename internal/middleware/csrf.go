package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var csrfSecret string

func init() {
	csrfSecret = os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		csrfSecret = os.Getenv("JWT_SECRET")
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	h := sha256.Sum256(append(b, []byte(csrfSecret)...))
	return hex.EncodeToString(h[:])
}

func GetCSRFToken(c *gin.Context) string {
	token := generateCSRFToken()
	c.SetCookie("csrf_token", token, 3600, "/", "", false, false)
	// Also set in context for template rendering
	c.Set("csrf_token", token)
	return token
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Set CSRF token for GET requests
			GetCSRFToken(c)
			c.Next()
			return
		}

		// For state-changing requests, validate CSRF
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		formToken := c.PostForm("_csrf")
		headerToken := c.GetHeader("X-CSRF-Token")
		if formToken == "" && headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}

		valid := formToken == cookieToken || headerToken == cookieToken
		if !valid {
			// Double hash check
			h := sha256.Sum256([]byte(cookieToken + csrfSecret))
			expected := hex.EncodeToString(h[:])
			valid = formToken == expected || headerToken == expected
		}

		if !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token invalid"})
			return
		}

		c.Next()
	}
}
