package handlers

import (
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

func HomePage(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{
		"AssistEnabled": assist.IsEnabled(),
	})
}
