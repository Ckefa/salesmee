package services

import (
	"strings"

	"salesmee/internal/config"

	"github.com/gin-gonic/gin"
)

func isDev() bool {
	return config.C.AppEnv == "dev"
}

func SetSecureCookie(c *gin.Context, name, value string, maxAge int, path string) {
	secure := !isDev()
	c.SetCookie(name, value, maxAge, path, "", secure, true)
}

func ClearCookie(c *gin.Context, name, path string) {
	c.SetCookie(name, "", -1, path, "", !isDev(), true)
}

func GetBaseURL(c *gin.Context) string {
	if config.C.AppURL != "" {
		return strings.TrimRight(config.C.AppURL, "/")
	}
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

func GetScheme(c *gin.Context) string {
	if config.C.AppURL != "" {
		parts := strings.Split(config.C.AppURL, "://")
		if len(parts) == 2 {
			return parts[0]
		}
	}
	if c.Request.TLS == nil {
		return "http"
	}
	return "https"
}
