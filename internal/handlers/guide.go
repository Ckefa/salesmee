package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"salesmee/internal/config"
	"salesmee/internal/services/assist"
)

func ShowGuide(c *gin.Context) {
	c.HTML(http.StatusOK, "guide.html", gin.H{
		"AssistEnabled": assist.IsEnabled(),
		"IsDev":         config.C.AppEnv == "dev",
	})
}
