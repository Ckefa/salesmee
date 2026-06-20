package client

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"salesmee/internal/config"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	clientGoogleAdapter       *services.GoogleAdapter
	clientGoogleAdapterOnce   sync.Once
	clientFacebookAdapter     *services.FacebookAdapter
	clientFacebookAdapterOnce sync.Once
)

func getClientGoogleAdapter() *services.GoogleAdapter {
	clientGoogleAdapterOnce.Do(func() {
		clientGoogleAdapter = services.NewGoogleAdapter(config.C.GoogleClientRedirect)
	})
	return clientGoogleAdapter
}

func getClientFacebookAdapter() *services.FacebookAdapter {
	clientFacebookAdapterOnce.Do(func() {
		clientFacebookAdapter = services.NewFacebookAdapter(config.C.FBClientRedirectURL)
	})
	return clientFacebookAdapter
}

func clientGenerateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func safeClientOAuthRedirect(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/client"
	}
	return raw
}

func InitiateClientGoogleAuth(c *gin.Context) {
	state := clientGenerateState()
	redirect := c.Query("redirect")
	if redirect == "" {
		redirect, _ = c.Cookie("client_redirect")
	}
	services.SetSecureCookie(c, "client_google_oauth_state", state, 600, "/client/auth/google")
	services.SetSecureCookie(c, "client_google_oauth_redirect", safeClientOAuthRedirect(redirect), 600, "/client/auth/google")
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
	return token.SignedString([]byte(config.C.JWTSecret))
}

func HandleClientGoogleCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("client_google_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	services.ClearCookie(c, "client_google_oauth_state", "/client/auth/google")
	redirectTo, _ := c.Cookie("client_google_oauth_redirect")
	services.ClearCookie(c, "client_google_oauth_redirect", "/client/auth/google")

	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getClientGoogleAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to authenticate with Google.",
		}))
		return
	}

	var client models.Client
	if err := dbc(c).Where("google_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&client).Error; err != nil {
		client = models.Client{
			Name:      user.Name,
			Email:     user.Email,
			GoogleID:  user.ProviderID,
			AvatarURL: user.AvatarURL,
			Status:    models.StatusNew,
		}
		if err := dbc(c).Create(&client).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
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
			if err := dbc(c).Save(&client).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
					"Title": "Client Login - SalesMee",
					"Error": "Failed to link Google account.",
				}))
				return
			}
		}
	}

	now := time.Now()
	if err := dbc(c).Model(&models.Client{}).Where("id = ?", client.ID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	}).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to update status.",
		}))
		return
	}

	token, err := generateClientTokenFromID(client.ID, client.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	secure := config.C.AppEnv != "dev"
	c.SetCookie("client_token", token, 86400, "/", "", secure, false)
	c.Redirect(http.StatusFound, safeClientOAuthRedirect(redirectTo))
}

func InitiateClientFacebookAuth(c *gin.Context) {
	state := clientGenerateState()
	redirect := c.Query("redirect")
	if redirect == "" {
		redirect, _ = c.Cookie("client_redirect")
	}
	services.SetSecureCookie(c, "client_facebook_oauth_state", state, 600, "/client/auth/facebook")
	services.SetSecureCookie(c, "client_facebook_oauth_redirect", safeClientOAuthRedirect(redirect), 600, "/client/auth/facebook")
	url := getClientFacebookAdapter().GetAuthURL(state)
	c.Redirect(http.StatusFound, url)
}

func HandleClientFacebookCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("client_facebook_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	services.ClearCookie(c, "client_facebook_oauth_state", "/client/auth/facebook")
	redirectTo, _ := c.Cookie("client_facebook_oauth_redirect")
	services.ClearCookie(c, "client_facebook_oauth_redirect", "/client/auth/facebook")

	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getClientFacebookAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to authenticate with Facebook.",
		}))
		return
	}

	var client models.Client
	if err := dbc(c).Where("facebook_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&client).Error; err != nil {
		client = models.Client{
			Name:       user.Name,
			Email:      user.Email,
			FacebookID: user.ProviderID,
			AvatarURL:  user.AvatarURL,
			Status:     models.StatusNew,
		}
		if err := dbc(c).Create(&client).Error; err != nil {
			c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
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
			if err := dbc(c).Save(&client).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
					"Title": "Client Login - SalesMee",
					"Error": "Failed to link Facebook account.",
				}))
				return
			}
		}
	}

	now := time.Now()
	if err := dbc(c).Model(&models.Client{}).Where("id = ?", client.ID).Updates(map[string]interface{}{
		"is_online":    true,
		"last_seen_at": &now,
	}).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to update status.",
		}))
		return
	}

	token, err := generateClientTokenFromID(client.ID, client.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "client_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Client Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	secure := config.C.AppEnv != "dev"
	c.SetCookie("client_token", token, 86400, "/", "", secure, false)
	c.Redirect(http.StatusFound, safeClientOAuthRedirect(redirectTo))
}
