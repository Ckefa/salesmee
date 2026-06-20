package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	appdata "salesmee/internal/data"

	"github.com/gin-gonic/gin"
)

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.TrimSpace(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "&", "and")
	// Remove non-alphanumeric except hyphens
	var result []rune
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	slug = string(result)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ""
	}
	return slug
}

func uniqueSlug(base string) string {
	slug := base
	counter := 1
	for {
		var existing models.Business
		if db.DB.Where("slug = ?", slug).First(&existing).Error != nil {
			break
		}
		slug = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}
	return slug
}

func ShowLogin(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}
	c.HTML(http.StatusOK, "business_login.html", middleware.TemplateData(c, gin.H{
		"Title": "Login - SalesMee",
	}))
}

func ShowRegisterStep1(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}
	c.HTML(http.StatusOK, "register_step1.html", middleware.TemplateData(c, gin.H{
		"Title": "Register - SalesMee",
	}))
}

var validBusinessTypes = func() map[string]bool {
	m := make(map[string]bool)
	for _, bt := range appdata.BusinessTypes {
		m[bt.Value] = true
	}
	return m
}()

func RegisterStep1(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}

	name := c.PostForm("name")
	username := c.PostForm("username")
	email := c.PostForm("email")

	if name == "" || username == "" || email == "" {
		c.HTML(http.StatusOK, "register_step1.html", middleware.TemplateData(c, gin.H{
			"Title": "Register - SalesMee",
			"Error": "All fields are required",
		}))
		return
	}

	var existing models.Business
	if dbc(c).Where("email = ?", email).First(&existing).Error == nil {
		c.HTML(http.StatusOK, "register_step1.html", middleware.TemplateData(c, gin.H{
			"Title": "Register - SalesMee",
			"Error": "Email already exists",
		}))
		return
	}

	tok := RegStore.Save(&RegistrationData{
		Name:     name,
		Username: username,
		Email:    email,
	})

	c.Redirect(http.StatusFound, "/business/register/step2?token="+tok)
}

func ShowRegisterStep2(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}

	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	c.HTML(http.StatusOK, "register_step2.html", middleware.TemplateData(c, gin.H{
		"Title":         "Register - SalesMee",
		"Token":         tok,
		"Name":          data.Name,
		"Username":      data.Username,
		"Email":         data.Email,
		"BusinessType":  data.BusinessType,
		"Countries":     appdata.Countries,
		"Currencies":    appdata.Currencies,
		"BusinessTypes": appdata.BusinessTypes,
	}))
}

func RegisterStep2(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}

	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	businessType := c.PostForm("business_type")
	if businessType == "" || !validBusinessTypes[businessType] {
		c.HTML(http.StatusOK, "register_step2.html", middleware.TemplateData(c, gin.H{
			"Title":         "Register - SalesMee",
			"Token":         tok,
			"Name":          data.Name,
			"Username":      data.Username,
			"Email":         data.Email,
			"BusinessType":  data.BusinessType,
			"Error":         "Please select a valid business type",
			"Countries":     appdata.Countries,
			"Currencies":    appdata.Currencies,
			"BusinessTypes": appdata.BusinessTypes,
		}))
		return
	}

	country := c.PostForm("country")
	currency := c.PostForm("currency")

	if country == "" {
		country = "US"
	}
	if currency == "" {
		currency = "USD"
	}

	data.BusinessType = businessType
	data.Country = country
	data.Currency = currency
	RegStore.Delete(tok)
	newTok := RegStore.Save(data)

	c.Redirect(http.StatusFound, "/business/register/step3?token="+newTok)
}

func ShowRegisterStep3(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}

	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	c.HTML(http.StatusOK, "register_step3.html", middleware.TemplateData(c, gin.H{
		"Title":        "Register - SalesMee",
		"Token":        tok,
		"Name":         data.Name,
		"Username":     data.Username,
		"Email":        data.Email,
		"BusinessType": data.BusinessType,
		"Country":      data.Country,
		"Currency":     data.Currency,
	}))
}

func RegisterStep3(c *gin.Context) {
	if token, err := c.Cookie("token"); err == nil && token != "" {
		if _, err := services.ValidateToken(token); err == nil {
			c.Redirect(http.StatusFound, "/business")
			return
		}
	}

	tok := c.Query("token")
	data, ok := RegStore.Get(tok)
	if !ok {
		c.Redirect(http.StatusFound, "/business/register")
		return
	}

	password := c.PostForm("password")

	if password == "" || len(password) < 6 {
		c.HTML(http.StatusOK, "register_step3.html", middleware.TemplateData(c, gin.H{
			"Title":        "Register - SalesMee",
			"Token":        tok,
			"Name":         data.Name,
			"Username":     data.Username,
			"Email":        data.Email,
			"BusinessType": data.BusinessType,
			"Country":      data.Country,
			"Currency":     data.Currency,
			"Error":        "Password must be at least 6 characters",
		}))
		return
	}

	hashedPassword := services.Hash(password)

	slug := uniqueSlug(generateSlug(data.Name))
	if slug == "" {
		slug = uniqueSlug(generateSlug(data.Username))
	}

	country := data.Country
	if country == "" {
		country = "US"
	}
	currency := data.Currency
	if currency == "" {
		currency = "USD"
	}

	user := models.Business{
		Email:        data.Email,
		Password:     &hashedPassword,
		Name:         data.Name,
		Username:     data.Username,
		BusinessType: data.BusinessType,
		Country:      country,
		Currency:     currency,
		Slug:         slug,
		IsPublic:     true,
	}

	if err := dbc(c).Create(&user).Error; err != nil {
		RegStore.Delete(tok)
		c.HTML(http.StatusOK, "register_step3.html", middleware.TemplateData(c, gin.H{
			"Title":        "Register - SalesMee",
			"Token":        tok,
			"Name":         data.Name,
			"Username":     data.Username,
			"Email":        data.Email,
			"BusinessType": data.BusinessType,
			"Country":      data.Country,
			"Currency":     data.Currency,
			"Error":        "Email already exists",
		}))
		return
	}

	assignSilverPlan(user.ID)

	// Send verification email
	b := make([]byte, 32)
	rand.Read(b)
	verificationToken := hex.EncodeToString(b)
	dbc(c).Model(&user).Update("verification_token", verificationToken)

	verifyLink := services.GetBaseURL(c) + "/business/verify?token=" + verificationToken
	services.SendVerificationEmail(user.Email, verifyLink)

	RegStore.Delete(tok)

	token, err := services.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	services.SetSecureCookie(c, "token", token, 86400, "/business")
	c.Redirect(http.StatusFound, "/business")
}

func assignSilverPlan(businessID uint) {
	var silver models.SubscriptionPlan
	if err := db.DB.Where("code = ?", "silver").First(&silver).Error; err != nil {
		return
	}

	var existing models.BusinessSubscription
	if err := db.DB.Where("business_id = ?", businessID).First(&existing).Error; err == nil {
		return
	}

	sub := models.BusinessSubscription{
		BusinessID: businessID,
		PlanID:     silver.ID,
		Status:     "active",
	}
	db.DB.Create(&sub)
	db.DB.Model(&models.Business{}).Where("id = ?", businessID).Update("subscription_plan_id", silver.ID)
}

func Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	var user models.Business
	if err := dbc(c).Where("email = ?", email).First(&user).Error; err != nil {
		c.HTML(http.StatusUnauthorized, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Invalid email or password",
		}))
		return
	}

	if user.Password == nil || !services.Check(*user.Password, password) {
		c.HTML(http.StatusUnauthorized, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Invalid email or password",
		}))
		return
	}

	token, err := services.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "business_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Login - SalesMee",
			"Error": "Failed to generate token",
		}))
		return
	}

	services.SetSecureCookie(c, "token", token, 86400, "/business")
	c.Redirect(http.StatusFound, "/business")
}

func Logout(c *gin.Context) {
	services.ClearCookie(c, "token", "/business")
	c.Redirect(http.StatusFound, "/business/login")
}
