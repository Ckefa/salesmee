package notifier

import (
	"fmt"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"time"

	"gorm.io/gorm"
)

func HasBeenSent(db *gorm.DB, businessID, clientID uint, notifType string, referenceID *uint) bool {
	query := db.Where("business_id = ? AND client_id = ? AND type = ?", businessID, clientID, notifType)
	if referenceID != nil {
		query = query.Where("reference_id = ?", *referenceID)
	}
	var count int64
	query.Model(&models.NotificationLog{}).Count(&count)
	return count > 0
}

func MarkNotificationSent(db *gorm.DB, businessID, clientID uint, notifType, refType string, referenceID *uint, email, status string) error {
	return db.Create(&models.NotificationLog{
		BusinessID:    businessID,
		ClientID:      clientID,
		Type:          notifType,
		ReferenceType: refType,
		ReferenceID:   referenceID,
		Email:         email,
		Status:        status,
		SentAt:        time.Now(),
	}).Error
}

func CreateInAppNotif(db *gorm.DB, businessID uint, clientID *uint, title, body, icon, link string) error {
	return db.Create(&models.InAppNotification{
		BusinessID: businessID,
		ClientID:   clientID,
		Title:      title,
		Body:       body,
		Icon:       icon,
		Link:       link,
	}).Error
}

func GetOrCreatePrefs(db *gorm.DB, businessID uint) (*models.BusinessNotifPrefs, error) {
	var prefs models.BusinessNotifPrefs
	err := db.Where("business_id = ?", businessID).First(&prefs).Error
	if err == nil {
		return &prefs, nil
	}
	if err == gorm.ErrRecordNotFound {
		prefs = models.BusinessNotifPrefs{BusinessID: businessID}
		if e := db.Create(&prefs).Error; e != nil {
			return nil, e
		}
		return &prefs, nil
	}
	return nil, err
}

func NotifyLimitReached(db *gorm.DB, businessID uint, resourceKey, resourceLabel string, current, max int) {
	var business models.Business
	if err := db.First(&business, businessID).Error; err != nil {
		return
	}

	prefs, err := GetOrCreatePrefs(db, businessID)
	if err != nil {
		return
	}
	if !prefs.LimitReached {
		return
	}

	planName := "Silver"
	if business.SubscriptionPlanID != nil {
		var plan models.SubscriptionPlan
		if err := db.First(&plan, *business.SubscriptionPlanID).Error; err == nil {
			planName = plan.Name
		}
	}

	// Deduplicate: only send notification once per resource
	var count int64
	db.Model(&models.NotificationLog{}).Where("business_id = ? AND type = ? AND reference_type = ?", businessID, "limit_reached", resourceKey).Count(&count)
	alreadySent := count > 0

	if !alreadySent {
		CreateInAppNotif(db, businessID, nil,
			"Plan Limit Reached — "+resourceLabel,
			fmt.Sprintf("You've reached the %s limit on your %s plan (%d of %d). Upgrade to add more.", resourceLabel, planName, current, max),
			"fa-exclamation-triangle",
			"/business/subscription#plans",
		)

		if business.Email != "" {
			if err := services.SendLimitReachedEmail(business.Email, business.Name, resourceLabel, current, max, planName); err == nil {
				MarkNotificationSent(db, businessID, 0, "limit_reached", resourceKey, nil, business.Email, "sent")
			}
		}
	}
}
