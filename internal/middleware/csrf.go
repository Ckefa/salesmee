package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"salesmee/internal/config"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

func TemplateData(c *gin.Context, data gin.H) gin.H {
	if data == nil {
		data = gin.H{}
	}
	if _, exists := data["CSRFToken"]; !exists {
		data["CSRFToken"] = c.GetString("csrf_token")
	}
	if _, exists := data["AuthType"]; !exists {
		data["AuthType"] = c.GetString("auth_type")
	}
	if _, exists := data["Role"]; !exists {
		data["Role"] = c.GetString("role")
	}
	if _, exists := data["SupportEmail"]; !exists {
		data["SupportEmail"] = config.C.SupportEmail
	}
	if _, exists := data["IsDev"]; !exists {
		data["IsDev"] = config.C.AppEnv == "dev"
	}
	if _, exists := data["AppDomain"]; !exists {
		data["AppDomain"] = config.C.AppDomain
	}
	if _, exists := data["AssistEnabled"]; !exists {
		data["AssistEnabled"] = assist.IsEnabled()
	}
	return data
}

var csrfSecret string

func init() {
	csrfSecret = config.C.CSRFSecret
	if csrfSecret == "" {
		csrfSecret = config.C.JWTSecret
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
	if existing, err := c.Cookie("csrf_token"); err == nil && existing != "" {
		c.Set("csrf_token", existing)
		c.SetCookie("csrf_token", existing, 3600, "/", "", config.C.AppEnv != "dev", false)
		return existing
	}
	token := generateCSRFToken()
	c.SetCookie("csrf_token", token, 3600, "/", "", config.C.AppEnv != "dev", false)
	// Also set in context for template rendering
	c.Set("csrf_token", token)
	return token
}

func CSRFMiddleware() gin.HandlerFunc {
	skipPaths := map[string]bool{
		"/stripe/webhook": true,
		"/paddle/webhook": true,
		"/polar/webhook":  true,
	}

	return func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

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

		c.Set("csrf_token", cookieToken)
		c.Next()
	}
}
