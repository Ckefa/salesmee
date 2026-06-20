package business

import (
	"encoding/json"
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/assist"

	"github.com/gin-gonic/gin"
)

type hoursForm struct {
	Monday    []timeRange `json:"monday"`
	Tuesday   []timeRange `json:"tuesday"`
	Wednesday []timeRange `json:"wednesday"`
	Thursday  []timeRange `json:"thursday"`
	Friday    []timeRange `json:"friday"`
	Saturday  []timeRange `json:"saturday"`
	Sunday    []timeRange `json:"sunday"`
}

type timeRange struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type specialHourEntry struct {
	Date     string `json:"date"`
	IsClosed bool   `json:"is_closed"`
	Open     string `json:"open,omitempty"`
	Close    string `json:"close,omitempty"`
	Reason   string `json:"reason"`
}

func (h *HoursHandler) GetBusinessHours(c *gin.Context) {
	businessID := c.GetUint("business_id")
	var business models.Business
	if err := h.dbc(c).First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	// Parse JSON strings into proper types so json.Marshal in template works correctly
	var hoursObj interface{}
	if business.BusinessHours != "" && business.BusinessHours != "{}" {
		json.Unmarshal([]byte(business.BusinessHours), &hoursObj)
	}
	if hoursObj == nil {
		hoursObj = map[string]interface{}{}
	}

	var specialObj interface{}
	if business.SpecialHours != "" && business.SpecialHours != "[]" {
		json.Unmarshal([]byte(business.SpecialHours), &specialObj)
	}
	if specialObj == nil {
		specialObj = []interface{}{}
	}

	data := gin.H{
		"Title":               "Business Hours - SalesMee",
		"ActivePage":          "hours",
		"Business":            business,
		"TimeZone":            business.TimeZone,
		"BufferTime":          business.BufferTime,
		"MaxBookingsPerSlot":  business.MaxBookingsPerSlot,
		"IsAcceptingBookings": business.IsAcceptingBookings,
		"BusinessHours":       hoursObj,
		"SpecialHours":        specialObj,
		"AuthType":            c.GetString("auth_type"),
		"Role":                c.GetString("role"),
		"AssistEnabled":       assist.IsEnabled(),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard/hours_content", data)
		return
	}

	c.HTML(http.StatusOK, "hours.html", data)
}

func (h *HoursHandler) UpdateBusinessHours(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var req struct {
		TimeZone    string `json:"timezone"`
		BufferTime  int    `json:"buffer_time"`
		MaxPerSlot  int    `json:"max_bookings_per_slot"`
		HoursJSON   string `json:"hours_json"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"time_zone":            req.TimeZone,
		"buffer_time":          req.BufferTime,
		"max_bookings_per_slot": req.MaxPerSlot,
	}

	if req.HoursJSON != "" {
		var parsed hoursForm
		if err := json.Unmarshal([]byte(req.HoursJSON), &parsed); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hours format"})
			return
		}
		updates["business_hours"] = req.HoursJSON
	}

	if err := h.dbc(c).Model(&models.Business{}).Where("id = ?", businessID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hours"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HoursHandler) UpdateSpecialHours(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var req struct {
		SpecialJSON string `json:"special_json"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SpecialJSON != "" {
		var parsed []specialHourEntry
		if err := json.Unmarshal([]byte(req.SpecialJSON), &parsed); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid special hours format"})
			return
		}
	}

	if err := h.dbc(c).Model(&models.Business{}).Where("id = ?", businessID).Update("special_hours", req.SpecialJSON).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update special hours"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HoursHandler) ToggleAcceptingBookings(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var req struct {
		Accepting bool `json:"accepting"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbc(c).Model(&models.Business{}).Where("id = ?", businessID).Update("is_accepting_bookings", req.Accepting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
