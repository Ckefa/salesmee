package handlers

import (
	"net/http"

	"salesmee/internal/config"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

func HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
"Title": "SalesMee — Chat CRM: Manage Clients, Orders & Bookings",
		"AssistEnabled": assist.IsEnabled(),
		"SupportEmail":  config.C.SupportEmail,
		"IsDev":         config.C.AppEnv == "dev",
	})
}
