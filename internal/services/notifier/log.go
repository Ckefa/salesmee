package notifier

import (
	"salesmee/internal/models"
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
