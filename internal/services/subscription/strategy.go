package subscription

import (
	"fmt"
	"oneflow/internal/models"

	"gorm.io/gorm"
)

type ProrationResult struct {
	CreditAmount    float64
	ChargeAmount    float64
	ImmediateCharge bool
}

type TransitionResult struct {
	Success       bool
	Message       string
	Proration     *ProrationResult
	ImmediateAction string
}

type PlanTransitionStrategy interface {
	Type() string
	Validate(current, target *models.SubscriptionPlan) error
	Calculate(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan, interval string) (*ProrationResult, error)
	Execute(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan) error
}

type UpgradeStrategy struct{}

func (s *UpgradeStrategy) Type() string { return "upgrade" }

func (s *UpgradeStrategy) Validate(current, target *models.SubscriptionPlan) error {
	if current.SortOrder >= target.SortOrder {
		return fmt.Errorf("target plan must be higher tier than current plan")
	}
	return nil
}

func (s *UpgradeStrategy) Calculate(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan, interval string) (*ProrationResult, error) {
	var currentPrice, targetPrice float64
	if interval == "year" {
		currentPrice = current.PriceYearly
		targetPrice = target.PriceYearly
	} else {
		currentPrice = current.PriceMonthly
		targetPrice = target.PriceMonthly
	}
	if targetPrice <= currentPrice {
		return &ProrationResult{ImmediateCharge: false}, nil
	}
	diff := targetPrice - currentPrice
	return &ProrationResult{
		CreditAmount:    0,
		ChargeAmount:    diff,
		ImmediateCharge: true,
	}, nil
}

func (s *UpgradeStrategy) Execute(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan) error {
	return db.Model(&models.Business{}).Where("id = ?", businessID).Update("subscription_plan_id", target.ID).Error
}

type DowngradeStrategy struct{}

func (s *DowngradeStrategy) Type() string { return "downgrade" }

func (s *DowngradeStrategy) Validate(current, target *models.SubscriptionPlan) error {
	if current.SortOrder <= target.SortOrder {
		return fmt.Errorf("target plan must be lower tier than current plan")
	}
	return nil
}

func (s *DowngradeStrategy) Calculate(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan, interval string) (*ProrationResult, error) {
	return &ProrationResult{
		CreditAmount:    0,
		ChargeAmount:    0,
		ImmediateCharge: false,
	}, nil
}

func (s *DowngradeStrategy) Execute(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan) error {
	var sub models.BusinessSubscription
	if err := db.Where("business_id = ?", businessID).First(&sub).Error; err != nil {
		return err
	}
	return db.Model(&sub).Update("plan_id", target.ID).Error
}

type CancelStrategy struct{}

func (s *CancelStrategy) Type() string { return "cancel" }

func (s *CancelStrategy) Validate(current, target *models.SubscriptionPlan) error {
	return nil
}

func (s *CancelStrategy) Calculate(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan, interval string) (*ProrationResult, error) {
	return &ProrationResult{
		CreditAmount:    0,
		ChargeAmount:    0,
		ImmediateCharge: false,
	}, nil
}

func (s *CancelStrategy) Execute(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan) error {
	return db.Model(&models.BusinessSubscription{}).
		Where("business_id = ?", businessID).
		Update("status", "canceled").Error
}

type ReactivateStrategy struct{}

func (s *ReactivateStrategy) Type() string { return "reactivate" }

func (s *ReactivateStrategy) Validate(current, target *models.SubscriptionPlan) error {
	return nil
}

func (s *ReactivateStrategy) Calculate(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan, interval string) (*ProrationResult, error) {
	var price float64
	if interval == "year" {
		price = target.PriceYearly
	} else {
		price = target.PriceMonthly
	}
	return &ProrationResult{
		CreditAmount:    0,
		ChargeAmount:    price,
		ImmediateCharge: true,
	}, nil
}

func (s *ReactivateStrategy) Execute(db *gorm.DB, businessID uint, current, target *models.SubscriptionPlan) error {
	return db.Model(&models.BusinessSubscription{}).
		Where("business_id = ?", businessID).
		Updates(map[string]interface{}{
			"status": "active",
			"plan_id": target.ID,
		}).Error
}

func GetTransitionStrategy(current, target *models.SubscriptionPlan) (PlanTransitionStrategy, error) {
	if current == nil || current.ID == 0 {
		return &UpgradeStrategy{}, nil
	}
	if current.ID == target.ID {
		return nil, fmt.Errorf("cannot transition to the same plan")
	}
	if target.SortOrder > current.SortOrder {
		return &UpgradeStrategy{}, nil
	}
	if target.SortOrder < current.SortOrder {
		return &DowngradeStrategy{}, nil
	}
	return nil, fmt.Errorf("no valid transition strategy found")
}
