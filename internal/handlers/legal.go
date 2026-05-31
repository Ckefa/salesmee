package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ShowPrivacy(c *gin.Context) {
	c.HTML(http.StatusOK, "privacy.html", nil)
}

func ShowTerms(c *gin.Context) {
	c.HTML(http.StatusOK, "terms.html", nil)
}

func ShowCookies(c *gin.Context) {
	c.HTML(http.StatusOK, "cookies.html", nil)
}

func ShowUserDeletion(c *gin.Context) {
	externalID := c.Query("id")
	c.HTML(http.StatusOK, "user_deletion.html", gin.H{
		"ExternalID":  externalID,
		"Submitted":   false,
		"Email":       "",
		"AccountType": "",
	})
}

func SubmitUserDeletion(c *gin.Context) {
	email := c.PostForm("email")
	accountType := c.PostForm("account_type")
	externalID := c.PostForm("external_id")
	reason := c.PostForm("reason")

	requestID := generateDeletionRequestID()

	fmt.Printf("[DELETION REQUEST] ID=%s Email=%s Type=%s ExternalID=%s Reason=%s\n",
		requestID, email, accountType, externalID, reason)

	c.HTML(http.StatusOK, "user_deletion.html", gin.H{
		"Submitted":   true,
		"RequestID":   requestID,
		"Email":       email,
		"AccountType": accountType,
		"ExternalID":  externalID,
	})
}

func generateDeletionRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("DEL-%d", 0)
	}
	return fmt.Sprintf("DEL-%x", b)
}
