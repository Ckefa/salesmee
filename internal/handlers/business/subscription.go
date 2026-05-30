package business

import (
	"fmt"
	"net/http"
	"time"
	"oneflow/internal/models"
	"oneflow/internal/services/payment"
	"oneflow/internal/services/subscription"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubscriptionPageData struct {
	ActivePage   string
	Business     models.Business
	Subscription *models.BusinessSubscription
	Plan         *models.SubscriptionPlan
	Plans        []models.SubscriptionPlan
	Usage        map[string]*subscription.LimitCheck
	UpcomingInvoice *UpcomingInvoiceInfo
}

type UpcomingInvoiceInfo struct {
	Amount   float64
	Date     string
	Interval string
}

type PlansPageData struct {
	ActivePage string
	Business   models.Business
	Plans      []models.SubscriptionPlan
	Current    *models.SubscriptionPlan
}

func (h *BusinessHandler) GetSubscriptionPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}

	var plans []models.SubscriptionPlan
	h.db.Where("is_active = ?", true).Order("sort_order asc").Find(&plans)

	usage := map[string]*subscription.LimitCheck{
		"clients":       subscription.CheckResourceLimit(businessID, "client"),
		"products":      subscription.CheckResourceLimit(businessID, "product"),
		"services":      subscription.CheckResourceLimit(businessID, "service"),
		"conversations": subscription.CheckResourceLimit(businessID, "conversation"),
	}

	data := SubscriptionPageData{
		ActivePage:   "subscription",
		Business:     business,
		Subscription: business.Subscription,
		Plans:        plans,
		Usage:        usage,
	}

	if business.Subscription != nil {
		data.Plan = &business.Subscription.Plan
		if business.Subscription.Status == "active" || business.Subscription.Status == "trialing" {
			data.UpcomingInvoice = &UpcomingInvoiceInfo{
				Amount:   business.Subscription.Plan.PriceMonthly,
				Date:     business.Subscription.CurrentPeriodEnd.Format("Jan 2, 2006"),
				Interval: business.Subscription.BillingInterval,
			}
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "subscription_content", data)
	} else {
		c.HTML(http.StatusOK, "subscription.html", data)
	}
}

func (h *BusinessHandler) GetPlansPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found"})
		return
	}

	var plans []models.SubscriptionPlan
	h.db.Where("is_active = ?", true).Order("sort_order asc").Find(&plans)

	var current *models.SubscriptionPlan
	if business.SubscriptionPlanID != nil {
		var plan models.SubscriptionPlan
		if err := h.db.First(&plan, *business.SubscriptionPlanID).Error; err == nil {
			current = &plan
		}
	}

	data := PlansPageData{
		ActivePage: "subscription",
		Business:   business,
		Plans:      plans,
		Current:    current,
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "subscription_plans_content", data)
	} else {
		c.HTML(http.StatusOK, "subscription_plans.html", data)
	}
}

func (h *BusinessHandler) CreateCheckout(c *gin.Context) {
	businessID := c.GetUint("business_id")
	planCode := c.PostForm("plan_code")
	billingInterval := c.PostForm("interval")
	if billingInterval == "" {
		billingInterval = "month"
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	var plan models.SubscriptionPlan
	if err := h.db.Where("code = ? AND is_active = ?", planCode, true).First(&plan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	var unitAmount float64
	if billingInterval == "year" {
		unitAmount = plan.PriceYearly
	} else {
		unitAmount = plan.PriceMonthly
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	host := c.Request.Host

	provider := payment.NewStripeAdapter()
	checkoutCtx := &payment.CheckoutContext{
		CustomerEmail:   business.Email,
		BusinessID:      business.ID,
		BusinessName:    business.Name,
		PlanCode:        planCode,
		PlanName:        plan.Name,
		BillingInterval: billingInterval,
		UnitAmount:      unitAmount,
		Currency:        "usd",
		SuccessURL:      fmt.Sprintf("%s://%s/business/subscription?checkout=success", scheme, host),
		CancelURL:       fmt.Sprintf("%s://%s/business/subscription/plans?checkout=canceled", scheme, host),
	}

	session, err := provider.CreateCheckoutSession(checkoutCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session: " + err.Error()})
		return
	}

	c.Redirect(http.StatusSeeOther, session.URL)
}

func (h *BusinessHandler) ChangePlan(c *gin.Context) {
	businessID := c.GetUint("business_id")
	planCode := c.PostForm("plan_code")

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	var target models.SubscriptionPlan
	if err := h.db.Where("code = ? AND is_active = ?", planCode, true).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}

	var current *models.SubscriptionPlan
	if business.Subscription != nil {
		current = &business.Subscription.Plan
	}

	strategy, err := subscription.GetTransitionStrategy(current, &target)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if current != nil {
		if err := strategy.Validate(current, &target); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := strategy.Execute(h.db, businessID, current, &target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to switch plan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Plan changed successfully"})
}

func (h *BusinessHandler) CancelSubscription(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	if business.Subscription == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active subscription"})
		return
	}

	strategy := &subscription.CancelStrategy{}
	if err := strategy.Execute(h.db, businessID, &business.Subscription.Plan, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
		return
	}

	var silver models.SubscriptionPlan
	if err := h.db.Where("code = ?", "silver").First(&silver).Error; err == nil {
		h.db.Model(&business).Update("subscription_plan_id", silver.ID)
	}

	now := time.Now()
	h.db.Model(&models.BusinessSubscription{}).
		Where("business_id = ?", businessID).
		Updates(map[string]interface{}{
			"canceled_at": &now,
			"status":      "canceled",
		})

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Subscription canceled. You still have access until the end of your billing period."})
}

func (h *BusinessHandler) BillingPortal(c *gin.Context) {
	businessID := c.GetUint("business_id")

	var business models.Business
	if err := h.db.Preload("Subscription").First(&business, businessID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Business not found"})
		return
	}

	if business.Subscription == nil || business.Subscription.StripeCustomerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe customer found"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	returnURL := fmt.Sprintf("%s://%s/business/subscription", scheme, c.Request.Host)

	provider := payment.NewStripeAdapter()
	portalURL, err := provider.CreateBillingPortalSession(business.Subscription.StripeCustomerID, returnURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create portal session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": portalURL})
}

func StripeWebhook(h *BusinessHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sigHeader := c.GetHeader("Stripe-Signature")

		provider := payment.NewStripeAdapter()
		event, err := provider.HandleWebhook(payload, sigHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch event.Type {
		case "checkout.session.completed":
			handleCheckoutCompleted(h.db, event)
		case "customer.subscription.updated":
			handleSubscriptionUpdated(h.db, event)
		case "customer.subscription.deleted":
			handleSubscriptionDeleted(h.db, event)
		case "invoice.paid":
			handleInvoicePaid(h.db, event)
		case "invoice.payment_failed":
			handleInvoicePaymentFailed(h.db, event)
		}

		c.JSON(http.StatusOK, gin.H{"received": true})
	}
}

func handleCheckoutCompleted(d *gorm.DB, event *payment.WebhookEvent) {
	var business models.Business
	if err := d.First(&business, event.BusinessID).Error; err != nil {
		return
	}

	var plan models.SubscriptionPlan
	if err := d.Where("code = ?", event.SubscriptionPlanCode).First(&plan).Error; err != nil {
		return
	}

	interval := event.BillingInterval
	if interval == "" {
		interval = "month"
	}

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	if interval == "year" {
		periodEnd = now.AddDate(1, 0, 0)
	}

	sub := models.BusinessSubscription{
		BusinessID:           business.ID,
		PlanID:               plan.ID,
		Status:               "trialing",
		StripeSubscriptionID: event.SubscriptionID,
		StripeCustomerID:     event.CustomerID,
		BillingInterval:      interval,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     periodEnd,
	}

	if event.TrialEnd > 0 {
		trialEnd := time.Unix(event.TrialEnd, 0)
		sub.TrialEndsAt = &trialEnd
		sub.CurrentPeriodEnd = trialEnd
	}

	if event.CurrentPeriodStart > 0 {
		sub.CurrentPeriodStart = time.Unix(event.CurrentPeriodStart, 0)
	}
	if event.CurrentPeriodEnd > 0 {
		sub.CurrentPeriodEnd = time.Unix(event.CurrentPeriodEnd, 0)
	}

	if event.SubscriptionStatus == "active" || event.SubscriptionStatus == "" {
		sub.Status = "active"
	}

	var existing models.BusinessSubscription
	if err := d.Where("business_id = ?", business.ID).First(&existing).Error; err != nil {
		d.Create(&sub)
	} else {
		d.Model(&existing).Updates(map[string]interface{}{
			"plan_id":                plan.ID,
			"status":                 sub.Status,
			"stripe_subscription_id": event.SubscriptionID,
			"stripe_customer_id":     event.CustomerID,
			"billing_interval":       interval,
			"current_period_start":   sub.CurrentPeriodStart,
			"current_period_end":     sub.CurrentPeriodEnd,
			"trial_ends_at":          sub.TrialEndsAt,
		})
	}

	d.Model(&business).Update("subscription_plan_id", plan.ID)
}

func handleSubscriptionUpdated(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_subscription_id = ?", event.SubscriptionID).First(&sub).Error; err != nil {
		return
	}

	updates := map[string]interface{}{}
	if event.SubscriptionStatus != "" {
		updates["status"] = event.SubscriptionStatus
	}
	if event.CurrentPeriodStart > 0 {
		updates["current_period_start"] = time.Unix(event.CurrentPeriodStart, 0)
	}
	if event.CurrentPeriodEnd > 0 {
		updates["current_period_end"] = time.Unix(event.CurrentPeriodEnd, 0)
	}
	if event.TrialEnd > 0 {
		t := time.Unix(event.TrialEnd, 0)
		updates["trial_ends_at"] = &t
	}

	if len(updates) > 0 {
		d.Model(&sub).Updates(updates)
	}
}

func handleSubscriptionDeleted(d *gorm.DB, event *payment.WebhookEvent) {
	d.Model(&models.BusinessSubscription{}).
		Where("stripe_subscription_id = ?", event.SubscriptionID).
		Update("status", "canceled")
}

func handleInvoicePaid(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_customer_id = ?", event.CustomerID).First(&sub).Error; err != nil {
		return
	}

	d.Model(&sub).Update("status", "active")
}

func handleInvoicePaymentFailed(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_customer_id = ?", event.CustomerID).First(&sub).Error; err != nil {
		return
	}

	d.Model(&sub).Update("status", "past_due")
}
