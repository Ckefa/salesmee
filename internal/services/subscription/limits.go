package subscription

import (
	"salesmee/internal/db"
	"salesmee/internal/models"
)

type LimitCheck struct {
	Allowed bool
	Message string
	Current int
	Max     int
}

func CheckResourceLimit(businessID uint, resource string) *LimitCheck {
	var business models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		return &LimitCheck{Allowed: false, Message: "Business not found"}
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

	var current int64
	switch resource {
	case "client":
		db.DB.Model(&models.Client{}).Where("business_id = ?", businessID).Count(&current)
	case "product":
		db.DB.Model(&models.Product{}).Where("business_id = ?", businessID).Count(&current)
	case "service":
		db.DB.Model(&models.Service{}).Where("business_id = ?", businessID).Count(&current)
	case "conversation":
		db.DB.Model(&models.Conversation{}).Where("business_id = ?", businessID).Count(&current)
	default:
		return &LimitCheck{Allowed: false, Message: "Unknown resource type"}
	}

	limit := 0
	switch resource {
	case "client":
		limit = maxClients
	case "product":
		limit = maxProducts
	case "service":
		limit = maxServices
	case "conversation":
		limit = maxConversations
	}

	if limit == -1 {
		return &LimitCheck{Allowed: true, Current: int(current), Max: -1}
	}

	if int(current) >= limit {
		return &LimitCheck{
			Allowed: false,
			Message: "You've reached the " + resource + " limit for your plan. Upgrade to add more.",
			Current: int(current),
			Max:     limit,
		}
	}

	return &LimitCheck{
		Allowed: true,
		Current: int(current),
		Max:     limit,
	}
}

func HasFeature(businessID uint, feature string) bool {
	var business models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		return false
	}

	if business.Subscription == nil || business.Subscription.Plan.ID == 0 {
		return false
	}

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
	default:
		return false
	}
}
