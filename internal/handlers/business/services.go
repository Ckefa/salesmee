package business

import (
	"net/http"
	"strconv"
	"salesmee/internal/config"
	"salesmee/internal/data"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

func (h *ServiceHandler) GetBusinessServices(c *gin.Context) {
	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid business ID"})
		return
	}

	var services []models.Service
	if err := h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Find(&services).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	c.JSON(http.StatusOK, services)
}

// GetServices for the business
func (h *ServiceHandler) GetServices(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Business not authenticated"})
		return
	}

	var currentBusiness models.Business
	if err := h.dbc(c).First(&currentBusiness, businessID).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Business not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := pageSize()

	baseWhere := "business_id = ?"
	baseArgs := []interface{}{businessID}
	locID := c.Query("location_id")
	if locID != "" {
		baseWhere += " AND (location_id IS NULL OR location_id = ?)"
		baseArgs = append(baseArgs, locID)
	}

	var totalCount int64
	h.dbc(c).Model(&models.Service{}).Where(baseWhere, baseArgs...).Count(&totalCount)

	var services []models.Service
	h.dbc(c).Where(baseWhere, baseArgs...).
		Order("created_at DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&services)

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var locations []models.Location
	h.dbc(c).Where("business_id = ?", businessID).Order("sort_order ASC, name ASC").Find(&locations)

	// HX-Request: Return only content partial
	if htmxRequest := c.GetHeader("HX-Request"); htmxRequest != "" {
		c.HTML(http.StatusOK, "dashboard/services_content", gin.H{
			"Business":         currentBusiness,
			"Services":         services,
			"Page":             float64(page),
			"TotalPages":       float64(totalPages),
			"TotalCount":       totalCount,
			"Countries":        data.Countries,
			"Currencies":       data.Currencies,
			"Onboarding":       onboardingData(h.db, businessID),
			"Locations":        locations,
			"AuthType":         c.GetString("auth_type"),
			"Role":             c.GetString("role"),
			"QueryLocationID":  locID,
			"ActivePage":       "services",
			"IsDev":            config.IsDev(),
		})
		return
	}

	c.HTML(http.StatusOK, "services.html", gin.H{
		"Business":         currentBusiness,
		"Services":         services,
		"Page":             float64(page),
		"TotalPages":       float64(totalPages),
		"TotalCount":       totalCount,
		"ActivePage":       "services",
		"Countries":        data.Countries,
		"Currencies":       data.Currencies,
		"Onboarding":       onboardingData(h.db, businessID),
		"Locations":        locations,
		"AuthType":         c.GetString("auth_type"),
		"Role":             c.GetString("role"),
		"QueryLocationID":  locID,
		"AssistEnabled":    assist.IsEnabled(),
		"IsDev":            config.IsDev(),
	})
}

func (h *ServiceHandler) CreateService(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Description  string  `json:"description"`
		MinPrice     float64 `json:"min_price"`
		MaxPrice     float64 `json:"max_price"`
		IsNegotiable bool    `json:"is_negotiable"`
		Duration     int     `json:"duration"`
		ImageURL     string  `json:"image_url"`
		LocationID   *uint   `json:"location_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := models.Service{
		BusinessID:   businessID,
		Name:         req.Name,
		Description:  req.Description,
		MinPrice:     req.MinPrice,
		MaxPrice:     req.MaxPrice,
		IsNegotiable: req.IsNegotiable,
		Duration:     req.Duration,
		ImageURL:     req.ImageURL,
		LocationID:   req.LocationID,
		IsActive:     true,
	}
	service.IsActive = true

	if err := h.dbc(c).Create(&service).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "service": service})
}

func (h *ServiceHandler) GetService(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	serviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var service models.Service
	if err := h.dbc(c).Where("id = ? AND business_id = ?", serviceID, businessID).First(&service).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "service": service})
}

func (h *ServiceHandler) UpdateService(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	serviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	var service models.Service
	if err := h.dbc(c).Where("id = ? AND business_id = ?", serviceID, businessID).First(&service).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbc(c).Save(&service).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "service": service})
}

func (h *ServiceHandler) DeleteService(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	serviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service ID"})
		return
	}

	if err := h.dbc(c).Where("id = ? AND business_id = ?", serviceID, businessID).Delete(&models.Service{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceHandler) ShowClientServicesPage(c *gin.Context) {
	clientID := c.GetUint("client_id")
	if clientID == 0 {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Client not authenticated"})
		return
	}

	businessID, err := strconv.ParseUint(c.Param("business_id"), 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client.html", gin.H{"error": "Invalid business ID", "IsDev": config.IsDev()})
		return
	}

	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "client.html", gin.H{"error": "Business not found", "IsDev": config.IsDev()})
		return
	}

	var services []models.Service
	h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Order("created_at DESC").Find(&services)

	var client models.Client
	h.dbc(c).First(&client, clientID)

	c.HTML(http.StatusOK, "client_services.html", gin.H{
		"Business": business,
		"Client":   client,
		"Services": services,
	})
}

type quickService struct {
	ID       uint    `json:"id"`
	Name     string  `json:"name"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	Duration int     `json:"duration"`
	ImageURL string  `json:"image_url"`
}

func (h *ServiceHandler) GetServicesQuickList(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var services []models.Service
	if err := h.dbc(c).Where("business_id = ? AND is_active = ?", businessID, true).Order("name ASC").Find(&services).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	list := make([]quickService, 0, len(services))
	for _, s := range services {
		list = append(list, quickService{
			ID:       s.ID,
			Name:     s.Name,
			MinPrice: s.MinPrice,
			MaxPrice: s.MaxPrice,
			Duration: s.Duration,
			ImageURL: s.ImageURL,
		})
	}

	c.JSON(http.StatusOK, list)
}
