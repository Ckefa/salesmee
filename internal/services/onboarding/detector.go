package onboarding

import (
	"salesmee/internal/models"
	"time"

	"gorm.io/gorm"
)

type OnboardingData struct {
	Step       int
	TotalSteps int
	Completed  bool
}

func DetectStep(db *gorm.DB, business *models.Business) (*OnboardingData, error) {
	var productCount, serviceCount, clientCount, orderCount, bookingCount int64

	db.Model(&models.Product{}).Where("business_id = ?", business.ID).Count(&productCount)
	db.Model(&models.Service{}).Where("business_id = ?", business.ID).Count(&serviceCount)
	db.Model(&models.Client{}).Where("business_id = ?", business.ID).Count(&clientCount)
	db.Model(&models.Order{}).Where("business_id = ?", business.ID).Count(&orderCount)
	db.Model(&models.Booking{}).Where("business_id = ?", business.ID).Count(&bookingCount)

	hasProducts := productCount > 0
	hasServices := serviceCount > 0
	hasLogo := business.Logo != ""
	hasClients := clientCount > 0
	hasOrdersOrBookings := orderCount > 0 || bookingCount > 0

	currentStep := business.OnboardingStep

	detectedStep := currentStep
	if currentStep == 0 {
		detectedStep = 1
	}
	if hasProducts || hasServices {
		if detectedStep < 2 {
			detectedStep = 2
		} else if detectedStep == 2 {
			detectedStep = 3
		}
	}
	if hasLogo {
		if detectedStep < 3 {
			detectedStep = 3
		} else if detectedStep == 3 {
			detectedStep = 4
		}
	}
	if hasClients {
		if detectedStep < 4 {
			detectedStep = 4
		} else if detectedStep == 4 {
			detectedStep = 5
		}
	}
	if hasOrdersOrBookings {
		if detectedStep < 5 {
			detectedStep = 5
		} else if detectedStep == 5 {
			detectedStep = 6
		}
	}

	if detectedStep != currentStep {
		now := time.Now()
		db.Model(business).Updates(map[string]interface{}{
			"onboarding_step": detectedStep,
			"updated_at":      now,
		})
		business.OnboardingStep = detectedStep
	}

	return &OnboardingData{
		Step:       detectedStep,
		TotalSteps: 5,
		Completed:  detectedStep > 5,
	}, nil
}

func AdvanceStep(db *gorm.DB, businessID uint) (*OnboardingData, error) {
	var business models.Business
	if err := db.First(&business, businessID).Error; err != nil {
		return nil, err
	}
	next := business.OnboardingStep + 1
	if next > 6 {
		next = 6
	}
	now := time.Now()
	db.Model(&business).Updates(map[string]interface{}{
		"onboarding_step": next,
		"updated_at":      now,
	})
	business.OnboardingStep = next
	return &OnboardingData{
		Step:       next,
		TotalSteps: 5,
		Completed:  next > 5,
	}, nil
}

func SkipOnboarding(db *gorm.DB, businessID uint) error {
	now := time.Now()
	return db.Model(&models.Business{}).Where("id = ?", businessID).Updates(map[string]interface{}{
		"onboarding_step": 6,
		"updated_at":      now,
	}).Error
}
