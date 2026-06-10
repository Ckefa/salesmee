package business

import (
	"net/http"
	"salesmee/internal/models"
	"salesmee/internal/services/notifier"

	"github.com/gin-gonic/gin"
)

func (h *BusinessHandler) GetNotificationSettings(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "notification_settings.html", gin.H{"error": "Business not found", "AuthType": c.GetString("auth_type"), "Role": c.GetString("role")})
		return
	}

	prefs, err := notifier.GetOrCreatePrefs(h.db, businessID)
	if err != nil {
		prefs = &models.BusinessNotifPrefs{BusinessID: businessID}
	}

	data := gin.H{
		"Business":      business,
		"ActivePage":    "notifications",
		"Title":         "Notification Settings — SalesMee",
		"Prefs":         prefs,
		"NotificationSettings": prefs,
		"AuthType":      c.GetString("auth_type"),
		"Role":          c.GetString("role"),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "dashboard/notification_settings_content", data)
		return
	}

	c.HTML(http.StatusOK, "notification_settings.html", data)
}

func (h *BusinessHandler) UpdateNotificationSettings(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Business not authenticated"})
		return
	}

	var request struct {
		BookingReminder1h   bool `json:"booking_reminder_1h"`
		BookingReminder24h  bool `json:"booking_reminder_24h"`
		OrderStatusChange   bool `json:"order_status_change"`
		BookingStatusChange bool `json:"booking_status_change"`
		PaymentDueReminder  bool `json:"payment_due_reminder"`
		AbandonedCart       bool `json:"abandoned_cart"`
		ReEngagement        bool `json:"re_engagement"`
		AbandonedCartHours  int  `json:"abandoned_cart_hours"`
		InactiveDays        int  `json:"inactive_days"`
		SoundEnabled        bool `json:"sound_enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := models.BusinessNotifPrefs{
		BusinessID:          businessID,
		BookingReminder1h:   request.BookingReminder1h,
		BookingReminder24h:  request.BookingReminder24h,
		OrderStatusChange:   request.OrderStatusChange,
		BookingStatusChange: request.BookingStatusChange,
		PaymentDueReminder:  request.PaymentDueReminder,
		AbandonedCart:       request.AbandonedCart,
		ReEngagement:        request.ReEngagement,
		AbandonedCartHours:  request.AbandonedCartHours,
		InactiveDays:        request.InactiveDays,
		SoundEnabled:        request.SoundEnabled,
	}

	if prefs.AbandonedCartHours <= 0 {
		prefs.AbandonedCartHours = 24
	}
	if prefs.InactiveDays <= 0 {
		prefs.InactiveDays = 30
	}

	if err := h.db.Where("business_id = ?", businessID).Assign(prefs).FirstOrCreate(&prefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Notification settings updated"})
}
