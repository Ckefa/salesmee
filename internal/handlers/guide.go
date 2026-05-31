package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ShowGuide(c *gin.Context) {
	c.HTML(http.StatusOK, "guide.html", nil)
}
