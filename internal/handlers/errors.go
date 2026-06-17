package handlers

import (
	"net/http"

	"salesmee/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Show404(c *gin.Context) {
	c.HTML(http.StatusNotFound, "error_404.html", middleware.TemplateData(c, gin.H{
		"Title": "Page Not Found - SalesMee",
	}))
}

func Show500(c *gin.Context) {
	c.HTML(http.StatusInternalServerError, "error_500.html", middleware.TemplateData(c, gin.H{
		"Title": "Server Error - SalesMee",
	}))
}
