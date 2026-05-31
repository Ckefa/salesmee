package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Show404(c *gin.Context) {
	c.HTML(http.StatusNotFound, "error_404.html", gin.H{
		"Title": "Page Not Found - OneFlow",
	})
}

func Show500(c *gin.Context) {
	c.HTML(http.StatusInternalServerError, "error_500.html", gin.H{
		"Title": "Server Error - OneFlow",
	})
}
