package business

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *TeamHandler) GetTeam(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	var members []models.TeamMember
	h.dbc(c).Where("business_id = ?", businessID).Preload("Locations").Order("created_at DESC").Find(&members)

	var locations []models.Location
	h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Order("name ASC").Find(&locations)

	data := gin.H{
		"Business":      business,
		"Members":       members,
		"Locations":     locations,
		"ActivePage":    "team",
		"AuthType":      c.GetString("auth_type"),
		"Role":          c.GetString("role"),
		"AssistEnabled": assist.IsEnabled(),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard/team_content", data)
		return
	}

	c.HTML(http.StatusOK, "team.html", data)
}

func (h *TeamHandler) InviteTeamMember(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var req struct {
		Name      string   `json:"name"`
		Email     string   `json:"email"`
		Role      string   `json:"role"`
		Phone     string   `json:"phone"`
		Locations []uint   `json:"location_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and email are required"})
		return
	}
	if req.Role == "" {
		req.Role = "staff"
	}

	var existing models.TeamMember
	if h.dbc(c).Where("business_id = ? AND email = ?", businessID, req.Email).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A team member with this email already exists"})
		return
	}

	token, err := generateInviteToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite"})
		return
	}

	var permJSON string
	if req.Role == "manager" {
		permJSON = `{"orders:rw":true,"bookings:rw":true,"clients:rw":true,"analytics:view":true,"products:rw":true,"services:rw":true,"payments:rw":true,"reports:view":true,"locations:view":true,"share:view":true,"subscription:view":true}`
	} else {
		permJSON = `{"orders:r":true,"bookings:r":true,"clients:r":true,"share:view":true,"subscription:view":true}`
	}

	member := models.TeamMember{
		BusinessID:  businessID,
		Name:        req.Name,
		Email:       req.Email,
		Role:        req.Role,
		Phone:       req.Phone,
		Permissions: permJSON,
		InviteToken: token,
		IsActive:    true,
	}

	if err := h.dbc(c).Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team member"})
		return
	}

	if len(req.Locations) > 0 {
		var locs []models.Location
		h.dbc(c).Where("id IN ? AND business_id = ?", req.Locations, businessID).Find(&locs)
		h.dbc(c).Model(&member).Association("Locations").Append(locs)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "invite_token": token})
}

func (h *TeamHandler) UpdateTeamMember(c *gin.Context) {
	businessID := c.GetUint("business_id")
	memberID := c.Param("id")

	var member models.TeamMember
	if err := h.dbc(c).Where("id = ? AND business_id = ?", memberID, businessID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team member not found"})
		return
	}

	var req struct {
		Name      string   `json:"name"`
		Role      string   `json:"role"`
		Phone     string   `json:"phone"`
		IsActive  *bool    `json:"is_active"`
		Locations []uint   `json:"location_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Role != "" {
		updates["role"] = req.Role
		if req.Role == "manager" {
			updates["permissions"] = `{"orders:rw":true,"bookings:rw":true,"clients:rw":true,"analytics:view":true,"products:rw":true,"services:rw":true,"payments:rw":true,"reports:view":true,"locations:view":true,"share:view":true,"subscription:view":true}`
		} else {
			updates["permissions"] = `{"orders:r":true,"bookings:r":true,"clients:r":true,"share:view":true,"subscription:view":true}`
		}
	}
	updates["phone"] = req.Phone
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := h.dbc(c).Model(&member).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team member"})
			return
		}
	}

	if req.Locations != nil {
		var locs []models.Location
		h.dbc(c).Where("id IN ? AND business_id = ?", req.Locations, businessID).Find(&locs)
		h.dbc(c).Model(&member).Association("Locations").Replace(locs)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *TeamHandler) DeleteTeamMember(c *gin.Context) {
	businessID := c.GetUint("business_id")
	memberID := c.Param("id")

	var member models.TeamMember
	if err := h.dbc(c).Where("id = ? AND business_id = ?", memberID, businessID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team member not found"})
		return
	}

	if err := h.dbc(c).Delete(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete team member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *TeamHandler) ShowAcceptInvite(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "team_accept.html", gin.H{"Error": "Invalid invite token"})
		return
	}

	var member models.TeamMember
	if err := h.dbc(c).Where("invite_token = ?", token).First(&member).Error; err != nil {
		c.HTML(http.StatusBadRequest, "team_accept.html", gin.H{"Error": "Invalid or expired invite token"})
		return
	}

	var business models.Business
	h.dbc(c).First(&business, member.BusinessID)

	c.HTML(http.StatusOK, "team_accept.html", gin.H{
		"Token":    token,
		"Email":    member.Email,
		"Name":     member.Name,
		"Business": business.Name,
	})
}

func (h *TeamHandler) AcceptInvite(c *gin.Context) {
	var req struct {
		Token           string `json:"token"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Token == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token and password are required"})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters"})
		return
	}
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
		return
	}

	var member models.TeamMember
	if err := h.dbc(c).Where("invite_token = ?", req.Token).First(&member).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired invite token"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	if err := h.dbc(c).Model(&member).Updates(map[string]interface{}{
		"password":     string(hashed),
		"invite_token": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
