package business

import (
	"net/http"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
)

func (h *BusinessHandler) GetLocations(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	var locations []models.Location
	h.db.Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	data := gin.H{
		"Business":   business,
		"Locations":  locations,
		"ActivePage": "locations",
		"AuthType":   c.GetString("auth_type"),
		"Role":       c.GetString("role"),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard/locations_content", data)
		return
	}

	c.HTML(http.StatusOK, "locations.html", data)
}

func (h *BusinessHandler) CreateLocation(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		Name      string  `json:"name"`
		Address   string  `json:"address"`
		Phone     string  `json:"phone"`
		Email     string  `json:"email"`
		TimeZone  string  `json:"timezone"`
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
		SortOrder int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}
	if req.TimeZone == "" {
		req.TimeZone = "UTC"
	}

	loc := models.Location{
		BusinessID: businessID,
		Name:       req.Name,
		Address:    req.Address,
		Phone:      req.Phone,
		Email:      req.Email,
		TimeZone:   req.TimeZone,
		Lat:        req.Lat,
		Lng:        req.Lng,
		SortOrder:  req.SortOrder,
		IsActive:   true,
	}

	if err := h.db.Create(&loc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "location": loc})
}

func (h *BusinessHandler) UpdateLocation(c *gin.Context) {
	businessID := c.GetUint("business_id")
	locationID := c.Param("id")

	var loc models.Location
	if err := h.db.Where("id = ? AND business_id = ?", locationID, businessID).First(&loc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
		return
	}

	var req struct {
		Name      string  `json:"name"`
		Address   string  `json:"address"`
		Phone     string  `json:"phone"`
		Email     string  `json:"email"`
		TimeZone  string  `json:"timezone"`
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
		IsActive  *bool   `json:"is_active"`
		SortOrder int     `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["address"] = req.Address
	updates["phone"] = req.Phone
	updates["email"] = req.Email
	if req.TimeZone != "" {
		updates["time_zone"] = req.TimeZone
	}
	updates["lat"] = req.Lat
	updates["lng"] = req.Lng
	updates["sort_order"] = req.SortOrder
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := h.db.Model(&loc).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BusinessHandler) DeleteLocation(c *gin.Context) {
	businessID := c.GetUint("business_id")
	locationID := c.Param("id")

	var loc models.Location
	if err := h.db.Where("id = ? AND business_id = ?", locationID, businessID).First(&loc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
		return
	}

	if err := h.db.Delete(&loc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
