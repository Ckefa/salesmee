package client

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	clientGoogleAdapter     *services.GoogleAdapter
	clientGoogleAdapterOnce sync.Once
	clientFacebookAdapter     *services.FacebookAdapter
	clientFacebookAdapterOnce sync.Once
)

func getClientGoogleAdapter() *services.GoogleAdapter {
	clientGoogleAdapterOnce.Do(func() {
		clientGoogleAdapter = services.NewGoogleAdapter(os.Getenv("GOOGLE_CLIENT_REDIRECT_URL"))
	})
	return clientGoogleAdapter
}

func getClientFacebookAdapter() *services.FacebookAdapter {
	clientFacebookAdapterOnce.Do(func() {
		clientFacebookAdapter = services.NewFacebookAdapter(os.Getenv("FB_CLIENT_REDIRECT_URL"))
	})
	return clientFacebookAdapter
}

func clientGenerateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func InitiateClientGoogleAuth(c *gin.Context) {
	state := clientGenerateState()
	c.SetCookie("client_google_oauth_state", state, 600, "/client/auth/google", "", false, true)
	url := getClientGoogleAdapter().GetAuthURL(state)
	c.Redirect(http.StatusFound, url)
}

func generateClientTokenFromID(clientID uint, email string) (string, error) {
	claims := &services.Claims{
		UserID: clientID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "client",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func HandleClientGoogleCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("client_google_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(400, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	c.SetCookie("client_google_oauth_state", "", -1, "/client/auth/google", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.HTML(400, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getClientGoogleAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to authenticate with Google.",
		}))
		return
	}

	var client models.Client
	if err := db.DB.Where("google_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&client).Error; err != nil {
		client = models.Client{
			Name:      user.Name,
			Email:     user.Email,
			GoogleID:  user.ProviderID,
			AvatarURL: user.AvatarURL,
			Status:    models.StatusNew,
		}
		if err := db.DB.Create(&client).Error; err != nil {
			c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
				"Title": "Client Login - SalesMee",
				"Error": "Failed to create account.",
			}))
			return
		}
	} else {
		if client.GoogleID == "" {
			client.GoogleID = user.ProviderID
			if user.AvatarURL != "" {
				client.AvatarURL = user.AvatarURL
			}
			db.DB.Save(&client)
		}
	}

	now := time.Now()
	db.DB.Model(&models.Client{}).Where("id = ?", client.ID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	})

	token, err := generateClientTokenFromID(client.ID, client.Email)
	if err != nil {
		c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	c.SetCookie("client_token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/client")
}

func InitiateClientFacebookAuth(c *gin.Context) {
	state := clientGenerateState()
	c.SetCookie("client_facebook_oauth_state", state, 600, "/client/auth/facebook", "", false, true)
	url := getClientFacebookAdapter().GetAuthURL(state)
	c.Redirect(http.StatusFound, url)
}

func HandleClientFacebookCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("client_facebook_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(400, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	c.SetCookie("client_facebook_oauth_state", "", -1, "/client/auth/facebook", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.HTML(400, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getClientFacebookAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to authenticate with Facebook.",
		}))
		return
	}

	var client models.Client
	if err := db.DB.Where("facebook_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&client).Error; err != nil {
		client = models.Client{
			Name:       user.Name,
			Email:      user.Email,
			FacebookID: user.ProviderID,
			AvatarURL:  user.AvatarURL,
			Status:     models.StatusNew,
		}
		if err := db.DB.Create(&client).Error; err != nil {
			c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
				"Title": "Client Login - SalesMee",
				"Error": "Failed to create account.",
			}))
			return
		}
	} else {
		if client.FacebookID == "" {
			client.FacebookID = user.ProviderID
			if user.AvatarURL != "" {
				client.AvatarURL = user.AvatarURL
			}
			db.DB.Save(&client)
		}
	}

	now := time.Now()
	db.DB.Model(&models.Client{}).Where("id = ?", client.ID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	})

	token, err := generateClientTokenFromID(client.ID, client.Email)
	if err != nil {
		c.HTML(500, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	c.SetCookie("client_token", token, 86400, "/", "", false, true)
	c.Redirect(http.StatusFound, "/client")
}
