package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"

	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
)

var (
	businessGoogleAdapter     *services.GoogleAdapter
	businessGoogleAdapterOnce sync.Once
	businessFacebookAdapter     *services.FacebookAdapter
	businessFacebookAdapterOnce sync.Once
)

func getBusinessGoogleAdapter() *services.GoogleAdapter {
	businessGoogleAdapterOnce.Do(func() {
		businessGoogleAdapter = services.NewGoogleAdapter(os.Getenv("GOOGLE_REDIRECT_URL"))
	})
	return businessGoogleAdapter
}

func getBusinessFacebookAdapter() *services.FacebookAdapter {
	businessFacebookAdapterOnce.Do(func() {
		businessFacebookAdapter = services.NewFacebookAdapter(os.Getenv("FB_REDIRECT_URL"))
	})
	return businessFacebookAdapter
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func InitiateBusinessGoogleAuth(c *gin.Context) {
	state := generateState()
	c.SetCookie("google_oauth_state", state, 600, "/business/auth/google", "", false, true)
	url := getBusinessGoogleAdapter().GetAuthURL(state)
	c.Redirect(http.StatusFound, url)
}

func HandleBusinessGoogleCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("google_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(400, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	c.SetCookie("google_oauth_state", "", -1, "/business/auth/google", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.HTML(400, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getBusinessGoogleAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(500, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Failed to authenticate with Google.",
		}))
		return
	}

	var business models.Business
	if err := db.DB.Where("google_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&business).Error; err != nil {
		tok := RegStore.Save(&RegistrationData{
			Name:      user.Name,
			Username:  generateSlug(user.Name),
			Email:     user.Email,
			GoogleID:  user.ProviderID,
			AvatarURL: user.AvatarURL,
		})
		c.Redirect(http.StatusFound, "/business/register/google?token="+tok)
		return
	}

	if business.GoogleID == "" {
		business.GoogleID = user.ProviderID
		business.EmailVerified = true
		if user.AvatarURL != "" {
			business.AvatarURL = user.AvatarURL
		}
		db.DB.Save(&business)
	}

	token, err := services.GenerateToken(business.ID, business.Email)
	if err != nil {
		c.HTML(500, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	c.SetCookie("token", token, 86400, "/business", "", false, true)
	c.Redirect(http.StatusFound, "/business")
}

func InitiateBusinessFacebookAuth(c *gin.Context) {
	state := generateState()
	c.SetCookie("facebook_oauth_state", state, 600, "/business/auth/facebook", "", false, true)
	url := getBusinessFacebookAdapter().GetAuthURL(state)
	c.Redirect(http.StatusFound, url)
}

func HandleBusinessFacebookCallback(c *gin.Context) {
	state := c.Query("state")
	cookieState, err := c.Cookie("facebook_oauth_state")
	if err != nil || state == "" || state != cookieState {
		c.HTML(400, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Invalid state parameter. Please try again.",
		}))
		return
	}
	c.SetCookie("facebook_oauth_state", "", -1, "/business/auth/facebook", "", false, true)

	code := c.Query("code")
	if code == "" {
		c.HTML(400, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "No authorization code provided.",
		}))
		return
	}

	user, err := getBusinessFacebookAdapter().ExchangeCode(code)
	if err != nil {
		c.HTML(500, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Failed to authenticate with Facebook.",
		}))
		return
	}

	var business models.Business
	if err := db.DB.Where("facebook_id = ?", user.ProviderID).Or("email = ?", user.Email).First(&business).Error; err != nil {
		tok := RegStore.Save(&RegistrationData{
			Name:       user.Name,
			Username:   generateSlug(user.Name),
			Email:      user.Email,
			FacebookID: user.ProviderID,
			AvatarURL:  user.AvatarURL,
		})
		c.Redirect(http.StatusFound, "/business/register/google?token="+tok)
		return
	}

	if business.FacebookID == "" {
		business.FacebookID = user.ProviderID
		business.EmailVerified = true
		if user.AvatarURL != "" {
			business.AvatarURL = user.AvatarURL
		}
		db.DB.Save(&business)
	}

	token, err := services.GenerateToken(business.ID, business.Email)
	if err != nil {
		c.HTML(500, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Failed to generate token.",
		}))
		return
	}

	c.SetCookie("token", token, 86400, "/business", "", false, true)
	c.Redirect(http.StatusFound, "/business")
}

func ShowRegisterGoogle(c *gin.Context) {
	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	c.HTML(200, "register_google.html", middleware.TemplateData(c, gin.H{
		"Title":     "Complete Registration - SalesMee",
		"Token":     tok,
		"Name":      data.Name,
		"Email":     data.Email,
		"AvatarURL": data.AvatarURL,
	}))
}

func CompleteRegisterGoogle(c *gin.Context) {
	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	businessType := c.PostForm("business_type")
	if businessType == "" || !validBusinessTypes[businessType] {
		c.HTML(200, "register_google.html", middleware.TemplateData(c, gin.H{
			"Title":     "Complete Registration - SalesMee",
			"Token":     tok,
			"Name":      data.Name,
			"Email":     data.Email,
			"AvatarURL": data.AvatarURL,
			"Error":     "Please select a valid business type",
		}))
		return
	}

	slug := uniqueSlug(generateSlug(data.Name))
	if slug == "" {
		slug = uniqueSlug(generateSlug(data.Email))
	}

	user := models.Business{
		Email:          data.Email,
		Password:       nil,
		Name:           data.Name,
		Username:       data.Username,
		BusinessType:   businessType,
		Slug:           slug,
		GoogleID:       data.GoogleID,
		FacebookID:     data.FacebookID,
		AvatarURL:      data.AvatarURL,
		EmailVerified:  true,
		IsPublic:       true,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		RegStore.Delete(tok)
		c.HTML(200, "register_google.html", middleware.TemplateData(c, gin.H{
			"Title":     "Complete Registration - SalesMee",
			"Token":     tok,
			"Name":      data.Name,
			"Email":     data.Email,
			"AvatarURL": data.AvatarURL,
			"Error":     "An account with this email already exists",
		}))
		return
	}

	RegStore.Delete(tok)

	token, err := services.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	c.SetCookie("token", token, 86400, "/business", "", false, true)
	c.Redirect(http.StatusFound, "/business")
}
