package business

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"salesmee/internal/models"
	"salesmee/internal/services"
	"salesmee/internal/services/payment"
	"salesmee/internal/services/subscription"

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

	// Sidebar badge counts
	ProductCount        int
	ServiceCount        int
	PendingOrderCount   int
	PendingBookingCount int

	// Enhanced fields
	DaysRemaining      int
	TotalDays          int
	IsYearly           bool
	YearlySavings      float64
	YearlyPrice        float64
	MonthlyPrice       float64
	HasPaymentMethod   bool
	IsCanceled         bool
	IsTrialing         bool
	TrialDaysRemaining int
	PlanIcon           string
	PlanColor          string

	// Provider
	PaymentProvider string
	AuthType        string
	Role            string
}

type UpcomingInvoiceInfo struct {
	Amount   float64
	Date     string
	Interval string
}

type CheckoutPageData struct {
	ActivePage string
	Business   models.Business
	Plan       *models.SubscriptionPlan
	Interval   string
	UnitAmount float64
	Providers  []string
	CSRFToken  string

	PaddleClientToken  string
	PaddleEnvironment  string

	ProductCount        int
	ServiceCount        int
	PendingOrderCount   int
	PendingBookingCount int
	AuthType            string
	Role                string
}

type PlansPageData struct {
	ActivePage string
	Business   models.Business
	Plans      []models.SubscriptionPlan
	Current    *models.SubscriptionPlan

	// Sidebar badge counts
	ProductCount        int
	ServiceCount        int
	PendingOrderCount   int
	PendingBookingCount int
	AuthType            string
	Role                string
}

func (h *BusinessHandler) GetSubscriptionPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found", "AuthType": c.GetString("auth_type"), "Role": c.GetString("role")})
		return
	}

	// Handle checkout success callback (webhooks can't reach localhost)
	if c.Query("checkout") == "success" {
		checkoutPlanCode := c.Query("plan_code")
		checkoutInterval := c.DefaultQuery("interval", "month")
		subscriptionID := c.Query("subscription_id")
		if subscriptionID == "" {
			subscriptionID = c.Query("session_id")
		}
		if checkoutPlanCode != "" {
			var plan models.SubscriptionPlan
			if err := h.db.Where("code = ? AND is_active = ?", checkoutPlanCode, true).First(&plan).Error; err == nil {
				now := time.Now()
				periodEnd := now.AddDate(0, 1, 0)
				if checkoutInterval == "year" {
					periodEnd = now.AddDate(1, 0, 0)
				}

				sub := models.BusinessSubscription{
					BusinessID:           businessID,
					PlanID:               plan.ID,
					Status:               "active",
					StripeSubscriptionID: subscriptionID,
					BillingInterval:      checkoutInterval,
					CurrentPeriodStart:   now,
					CurrentPeriodEnd:     periodEnd,
				}

				var existing models.BusinessSubscription
				if err := h.db.Where("business_id = ?", businessID).First(&existing).Error; err != nil {
					h.db.Create(&sub)
				} else {
					h.db.Model(&existing).Updates(map[string]interface{}{
						"plan_id":                plan.ID,
						"status":                 "active",
						"stripe_subscription_id": subscriptionID,
						"billing_interval":       checkoutInterval,
						"current_period_start":   now,
						"current_period_end":     periodEnd,
					})
				}

				h.db.Model(&models.Business{}).Where("id = ?", businessID).Update("subscription_plan_id", plan.ID)

				c.Redirect(http.StatusFound, "/business/subscription")
				return
			}
		}
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
		sub := business.Subscription
		data.Plan = &sub.Plan

		if sub.Status == "active" || sub.Status == "trialing" {
			amount := sub.Plan.PriceMonthly
			if sub.BillingInterval == "year" {
				amount = sub.Plan.PriceYearly
			}
			data.UpcomingInvoice = &UpcomingInvoiceInfo{
				Amount:   amount,
				Date:     sub.CurrentPeriodEnd.Format("Jan 2, 2006"),
				Interval: sub.BillingInterval,
			}
		}

		now := time.Now()
		if sub.CurrentPeriodEnd.After(now) {
			data.TotalDays = int(sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart).Hours() / 24)
			data.DaysRemaining = int(sub.CurrentPeriodEnd.Sub(now).Hours() / 24)
		}

		data.IsYearly = sub.BillingInterval == "year"
		data.MonthlyPrice = sub.Plan.PriceMonthly
		data.YearlyPrice = sub.Plan.PriceYearly
		data.YearlySavings = (sub.Plan.PriceMonthly - sub.Plan.PriceYearly) * 12
		data.HasPaymentMethod = sub.StripeCustomerID != "" || sub.PaddleCustomerID != ""
		data.PaymentProvider = "stripe"
		if sub.PaddleCustomerID != "" {
			data.PaymentProvider = "paddle"
		}
		data.IsCanceled = sub.Status == "canceled"
		data.IsTrialing = sub.Status == "trialing"

		if sub.TrialEndsAt != nil && sub.TrialEndsAt.After(now) {
			data.TrialDaysRemaining = int(sub.TrialEndsAt.Sub(now).Hours() / 24)
		}

		switch sub.Plan.Code {
		case "diamond":
			data.PlanIcon = "crown"
			data.PlanColor = "amber"
		case "gold":
			data.PlanIcon = "rocket"
			data.PlanColor = "teal"
		default:
			data.PlanIcon = "gem"
			data.PlanColor = "slate"
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
		c.HTML(http.StatusNotFound, "dashboard.html", gin.H{"error": "Business not found", "AuthType": c.GetString("auth_type"), "Role": c.GetString("role")})
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
		AuthType:   c.GetString("auth_type"),
		Role:       c.GetString("role"),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "subscription_plans_content", data)
	} else {
		c.HTML(http.StatusOK, "subscription_plans.html", data)
	}
}

func (h *BusinessHandler) GetCheckoutPage(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	var business models.Business
	if err := h.db.First(&business, businessID).Error; err != nil {
		c.Redirect(http.StatusFound, "/business/login")
		return
	}

	planCode := c.Query("plan")
	billingInterval := c.DefaultQuery("interval", "month")

	var plan models.SubscriptionPlan
	if err := h.db.Where("code = ? AND is_active = ?", planCode, true).First(&plan).Error; err != nil {
		c.Redirect(http.StatusFound, "/business/subscription/plans")
		return
	}

	var unitAmount float64
	if billingInterval == "year" {
		unitAmount = plan.PriceYearly
	} else {
		unitAmount = plan.PriceMonthly
	}

	paddleEnv := os.Getenv("PADDLE_ENVIRONMENT")
	if paddleEnv == "" {
		paddleEnv = "sandbox"
	}

	data := CheckoutPageData{
		ActivePage: "subscription",
		Business:   business,
		Plan:       &plan,
		Interval:   billingInterval,
		UnitAmount: unitAmount,
		Providers:  []string{"stripe", "paddle"},
		CSRFToken:  c.GetString("csrf_token"),

		PaddleClientToken: os.Getenv("PADDLE_CLIENT_TOKEN"),
		PaddleEnvironment: paddleEnv,
		AuthType:          c.GetString("auth_type"),
		Role:              c.GetString("role"),
	}

	c.HTML(http.StatusOK, "checkout.html", data)
}

func (h *BusinessHandler) CreateCheckout(c *gin.Context) {
	businessID := c.GetUint("business_id")
	planCode := c.PostForm("plan_code")
	billingInterval := c.PostForm("interval")
	providerName := c.PostForm("provider")
	if billingInterval == "" {
		billingInterval = "month"
	}
	if providerName == "" {
		providerName = "stripe"
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

	provider := payment.GetProvider(providerName)

	paddleEnv := os.Getenv("PADDLE_ENVIRONMENT")
	if paddleEnv == "" {
		paddleEnv = "sandbox"
	}
	paddleEnvUpper := strings.ToUpper(paddleEnv)
	envKey := fmt.Sprintf("PADDLE_%s_%s_%s_PRICE_ID", paddleEnvUpper, strings.ToUpper(planCode), strings.ToUpper(billingInterval))
	paddlePriceID := os.Getenv(envKey)
	if paddlePriceID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Paddle %s price ID not configured for plan %s %s. Set env var %s", paddleEnv, planCode, billingInterval, envKey)})
		return
	}

	checkoutCtx := &payment.CheckoutContext{
		CustomerEmail:   business.Email,
		BusinessID:      business.ID,
		BusinessName:    business.Name,
		PlanCode:        planCode,
		PlanName:        plan.Name,
		BillingInterval: billingInterval,
		UnitAmount:      unitAmount,
		Currency:        "usd",
		SuccessURL:      fmt.Sprintf("%s://%s/business/subscription?checkout=success&plan_code=%s&interval=%s", scheme, host, planCode, billingInterval),
		CancelURL:       fmt.Sprintf("%s://%s/business/subscription?checkout=canceled", scheme, host),
		Plan: &payment.PlanMeta{
			Name:          plan.Name,
			Description:   plan.Description,
			PaddlePriceID: paddlePriceID,
			Original:      &plan,
		},
	}

	session, err := provider.CreateCheckoutSession(checkoutCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session: " + err.Error()})
		return
	}

	if providerName == "paddle" {
		c.JSON(http.StatusOK, gin.H{
			"checkout_id": session.ID,
			"plan_code":   planCode,
			"interval":    billingInterval,
		})
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

	sub := business.Subscription

	// Cancel at the provider level first
	if sub.StripeSubscriptionID != "" {
		provider := payment.NewStripeAdapter()
		if err := provider.CancelSubscription(sub.StripeSubscriptionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel with Stripe: " + err.Error()})
			return
		}
	} else if sub.PaddleSubscriptionID != "" {
		provider := payment.NewPaddleAdapter()
		if err := provider.CancelSubscription(sub.PaddleSubscriptionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel with Paddle: " + err.Error()})
			return
		}
	}

	strategy := &subscription.CancelStrategy{}
	if err := strategy.Execute(h.db, businessID, &sub.Plan, nil); err != nil {
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

	if business.Subscription == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No subscription found"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	returnURL := fmt.Sprintf("%s://%s/business/subscription", scheme, c.Request.Host)

	var portalURL string
	var err error

	if business.Subscription.PaddleCustomerID != "" {
		provider := payment.NewPaddleAdapter()
		portalURL, err = provider.CreateBillingPortalSession(business.Subscription.PaddleCustomerID, returnURL)
	} else if business.Subscription.StripeCustomerID != "" {
		provider := payment.NewStripeAdapter()
		portalURL, err = provider.CreateBillingPortalSession(business.Subscription.StripeCustomerID, returnURL)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No billing customer found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create portal session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": portalURL})
}

func (h *BusinessHandler) GetPlanBadge(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusOK, "")
		return
	}

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.String(http.StatusOK, "")
		return
	}

	planName := "Silver"
	status := ""
	if business.Subscription != nil && business.Subscription.Plan.Name != "" {
		planName = business.Subscription.Plan.Name
		status = business.Subscription.Status
	}

	if status == "canceled" || status == "past_due" {
		c.String(http.StatusOK, fmt.Sprintf(
			`<a href="/business/subscription/plans" class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300 hover:bg-rose-200 dark:hover:bg-rose-900/50 transition-colors shadow-sm">
				<i class="fas fa-exclamation-triangle text-[10px]"></i> %s - %s
			</a>`, planName, status))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf(
		`<a href="/business/subscription" class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium bg-gradient-to-r from-teal-500 to-cyan-600 text-white hover:opacity-90 transition-opacity shadow-sm">
			<i class="fas fa-crown text-[10px]"></i> %s
		</a>`, planName))
}

func (h *BusinessHandler) GetPlanBadgeSidebar(c *gin.Context) {
	businessID := c.GetUint("business_id")
	if businessID == 0 {
		c.String(http.StatusOK, "")
		return
	}

	var business models.Business
	if err := h.db.Preload("Subscription.Plan").First(&business, businessID).Error; err != nil {
		c.String(http.StatusOK, "")
		return
	}

	planName := "Silver"
	if business.Subscription != nil && business.Subscription.Plan.Name != "" {
		planName = business.Subscription.Plan.Name
	}

	if planName == "Silver" {
		c.String(http.StatusOK, `<span class="px-1.5 py-0.5 rounded-full bg-[var(--color-surface-tertiary)] text-[var(--color-text-muted)]">Free</span>`)
		return
	}

	c.String(http.StatusOK, fmt.Sprintf(
		`<span class="px-1.5 py-0.5 rounded-full bg-gradient-to-r from-teal-500 to-cyan-600 text-white">%s</span>`, planName))
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

func PaddleWebhook(h *BusinessHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sigHeader := c.GetHeader("P-Thook-Signature")

		provider := payment.NewPaddleAdapter()
		event, err := provider.HandleWebhook(payload, sigHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch event.Type {
		case "subscription.created":
			handleCheckoutCompleted(h.db, event)
		case "subscription.updated":
			handleSubscriptionUpdated(h.db, event)
		case "subscription.cancelled":
			handleSubscriptionDeleted(h.db, event)
		case "subscription.past_due":
			handleInvoicePaymentFailed(h.db, event)
		case "transaction.completed":
			handleInvoicePaid(h.db, event)
		case "transaction.payment_failed":
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
		PaddleSubscriptionID: event.SubscriptionID,
		PaddleCustomerID:     event.CustomerID,
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
		updates := map[string]interface{}{
			"plan_id":              plan.ID,
			"status":               sub.Status,
			"billing_interval":     interval,
			"current_period_start": sub.CurrentPeriodStart,
			"current_period_end":   sub.CurrentPeriodEnd,
			"trial_ends_at":        sub.TrialEndsAt,
		}
		if event.SubscriptionID != "" {
			updates["stripe_subscription_id"] = event.SubscriptionID
			updates["paddle_subscription_id"] = event.SubscriptionID
		}
		if event.CustomerID != "" {
			updates["stripe_customer_id"] = event.CustomerID
			updates["paddle_customer_id"] = event.CustomerID
		}
		d.Model(&existing).Updates(updates)
	}

	d.Model(&business).Update("subscription_plan_id", plan.ID)

	if err := services.SendSubscriptionSuccess(business.Email, business.Name, plan.Name); err != nil {
		fmt.Printf("Warning: failed to send subscription success email to %s: %v\n", business.Email, err)
	}
}

func handleSubscriptionUpdated(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_subscription_id = ? OR paddle_subscription_id = ?", event.SubscriptionID, event.SubscriptionID).First(&sub).Error; err != nil {
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
	var sub models.BusinessSubscription
	if err := d.Where("stripe_subscription_id = ? OR paddle_subscription_id = ?", event.SubscriptionID, event.SubscriptionID).Preload("Business").First(&sub).Error; err != nil {
		return
	}

	d.Model(&sub).Update("status", "canceled")

	if err := services.SendSubscriptionExpired(sub.Business.Email, sub.Business.Name); err != nil {
		fmt.Printf("Warning: failed to send subscription expired email to %s: %v\n", sub.Business.Email, err)
	}
}

func handleInvoicePaid(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_customer_id = ? OR paddle_customer_id = ?", event.CustomerID, event.CustomerID).Preload("Business").Preload("Plan").First(&sub).Error; err != nil {
		return
	}

	d.Model(&sub).Update("status", "active")

	if err := services.SendSubscriptionSuccess(sub.Business.Email, sub.Business.Name, sub.Plan.Name); err != nil {
		fmt.Printf("Warning: failed to send renewal success email to %s: %v\n", sub.Business.Email, err)
	}
}

func handleInvoicePaymentFailed(d *gorm.DB, event *payment.WebhookEvent) {
	var sub models.BusinessSubscription
	if err := d.Where("stripe_customer_id = ? OR paddle_customer_id = ?", event.CustomerID, event.CustomerID).Preload("Business").First(&sub).Error; err != nil {
		return
	}

	d.Model(&sub).Update("status", "past_due")

	if err := services.SendSubscriptionFailed(sub.Business.Email, sub.Business.Name); err != nil {
		fmt.Printf("Warning: failed to send payment failed email to %s: %v\n", sub.Business.Email, err)
	}
}
