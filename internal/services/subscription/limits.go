package subscription

import (
	"encoding/json"
	"fmt"
	"salesmee/internal/db"
	"salesmee/internal/models"
	"strings"
)

type LimitCheck struct {
	Allowed      bool
	Message      string
	Current      int
	Max          int
	GraceAllowed bool
	GraceUsed    bool
}

func CheckResourceLimit(businessID uint, resource string) *LimitCheck {
	limit, current := resolveLimits(businessID, resource)

	if limit == -1 {
		return &LimitCheck{Allowed: true, Current: current, Max: -1}
	}

	if current >= limit {
		graceUsed := hasGraceUsed(businessID, resource)

		msg := fmt.Sprintf("You've reached the %s limit for your plan (%d of %d).", resource, current, limit)
		if !graceUsed {
			msg += " You have one grace action remaining."
		}

		return &LimitCheck{
			Allowed:      false,
			Message:      msg,
			Current:      current,
			Max:          limit,
			GraceAllowed: true,
			GraceUsed:    graceUsed,
		}
	}

	return &LimitCheck{
		Allowed: true,
		Current: current,
		Max:     limit,
	}
}

func UseGrace(businessID uint, resource string) {
	var sub models.BusinessSubscription
	if err := db.DB.Where("business_id = ?", businessID).First(&sub).Error; err != nil {
		return
	}

	var used []string
	if sub.GraceUsed != "" {
		json.Unmarshal([]byte(sub.GraceUsed), &used)
	}

	for _, r := range used {
		if r == resource {
			return
		}
	}

	used = append(used, resource)
	data, _ := json.Marshal(used)
	db.DB.Model(&sub).Update("grace_used", string(data))
}

func hasGraceUsed(businessID uint, resource string) bool {
	var sub models.BusinessSubscription
	if err := db.DB.Where("business_id = ?", businessID).First(&sub).Error; err != nil {
		return false
	}

	if sub.GraceUsed == "" || sub.GraceUsed == "[]" {
		return false
	}

	var used []string
	if err := json.Unmarshal([]byte(sub.GraceUsed), &used); err != nil {
		return false
	}

	for _, r := range used {
		if r == resource {
			return true
		}
	}
	return false
}

func resolveLimits(businessID uint, resource string) (limit int, current int) {
	var business models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		return 10, 0
	}

	maxClients := 10
	maxProducts := 10
	maxServices := 10
	maxConversations := 10

	if business.Subscription != nil && business.Subscription.Plan.ID > 0 {
		plan := business.Subscription.Plan
		if plan.MaxClients > 0 {
			maxClients = plan.MaxClients
		} else if plan.Code == "diamond" {
			maxClients = -1
		}
		if plan.MaxProducts > 0 {
			maxProducts = plan.MaxProducts
		} else if plan.Code == "diamond" {
			maxProducts = -1
		}
		if plan.MaxServices > 0 {
			maxServices = plan.MaxServices
		} else if plan.Code == "diamond" {
			maxServices = -1
		}
		if plan.MaxConversations > 0 {
			maxConversations = plan.MaxConversations
		} else if plan.Code == "diamond" {
			maxConversations = -1
		}
	}

	var count int64
	switch resource {
	case "client":
		db.DB.Model(&models.Client{}).Where("business_id = ?", businessID).Count(&count)
	case "product":
		db.DB.Model(&models.Product{}).Where("business_id = ?", businessID).Count(&count)
	case "service":
		db.DB.Model(&models.Service{}).Where("business_id = ?", businessID).Count(&count)
	case "conversation":
		db.DB.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&count)
	}

	l := 0
	switch resource {
	case "client":
		l = maxClients
	case "product":
		l = maxProducts
	case "service":
		l = maxServices
	case "conversation":
		l = maxConversations
	}

	return l, int(count)
}

func HasFeature(businessID uint, feature string) bool {
	var business models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		return false
	}

	if business.Subscription != nil && business.Subscription.Plan.ID > 0 {
		plan := business.Subscription.Plan
		switch feature {
		case "analytics":
			return plan.HasAnalytics
		case "media_sharing":
			return plan.HasMediaSharing
		case "priority_support":
			return plan.HasPrioritySupport
		case "orders_and_bookings":
			return plan.HasOrdersAndBookings
		}
		return false
	}

	planCode := ""
	if business.SubscriptionPlanID != nil {
		var plan models.SubscriptionPlan
		if err := db.DB.First(&plan, *business.SubscriptionPlanID).Error; err == nil {
			planCode = plan.Code
		}
	}

	if planCode == "" {
		planCode = "silver"
	}

	switch feature {
	case "analytics":
		return planCode == "gold" || planCode == "diamond"
	case "media_sharing":
		return planCode == "diamond"
	case "priority_support":
		return planCode == "diamond"
	case "orders_and_bookings":
		return planCode == "gold" || planCode == "diamond"
	}
	return false
}

func PlanDisplayName(businessID uint) string {
	var business models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		return "Free"
	}

	if business.Subscription != nil && business.Subscription.Plan.ID > 0 {
		return business.Subscription.Plan.Name
	}
	if business.SubscriptionPlanID != nil {
		var plan models.SubscriptionPlan
		if err := db.DB.First(&plan, *business.SubscriptionPlanID).Error; err == nil {
			return plan.Name
		}
	}
	return "Free"
}

func IsSilverPlan(businessID uint) bool {
	return strings.EqualFold(PlanDisplayName(businessID), "silver")
}
