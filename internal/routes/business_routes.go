package routes

import (
	"salesmee/internal/db"
	"salesmee/internal/handlers"
	"salesmee/internal/handlers/business"
	"salesmee/internal/handlers/client"
	"salesmee/internal/middleware"

	"github.com/gin-gonic/gin"
)

var businessHandler *business.BusinessHandler

func SetupBusinessRoutes(r *gin.Engine) {

	// Initialize business handler
	businessHandler = business.NewBusinessHandler(db.DB, wsHub)

	// PUBLIC - Business Auth Routes
	r.GET("/business/login", handlers.ShowLogin)
	r.GET("/business/register", handlers.ShowRegisterStep1)
	r.POST("/business/register", handlers.RegisterStep1)
	r.GET("/business/register/step2", handlers.ShowRegisterStep2)
	r.POST("/business/register/step2", handlers.RegisterStep2)
	r.GET("/business/register/step3", handlers.ShowRegisterStep3)
	r.POST("/business/register/step3", handlers.RegisterStep3)
	r.POST("/business/login", handlers.Login)
	r.GET("/business/logout", handlers.Logout)

	// Public - Password Reset
	r.GET("/business/forgot-password", handlers.ShowForgotPassword)
	r.POST("/business/forgot-password", handlers.SendForgotPassword)
	r.GET("/business/reset-password", handlers.ShowResetPassword)
	r.POST("/business/reset-password", handlers.SubmitResetPassword)

	// Public - Email Verification
	r.GET("/business/verify", handlers.VerifyBusinessEmail)

	// PUBLIC - Business Google Auth
	r.GET("/business/auth/google", handlers.InitiateBusinessGoogleAuth)
	r.GET("/business/auth/google/callback", handlers.HandleBusinessGoogleCallback)
	r.GET("/business/register/google", handlers.ShowRegisterGoogle)
	r.POST("/business/register/google/complete", handlers.CompleteRegisterGoogle)

	// PUBLIC - Business Facebook Auth
	r.GET("/business/auth/facebook", handlers.InitiateBusinessFacebookAuth)
	r.GET("/business/auth/facebook/callback", handlers.HandleBusinessFacebookCallback)

	// PROTECTED BUSINESS ROUTES
	protected := r.Group("/business")
	protected.Use(middleware.BizzMiddleware())
	{
		// Business Dashboard routes
		// Email verification (protected)
		protected.POST("/verify/send", handlers.SendBusinessVerification)
		protected.POST("/verify/status", handlers.CheckVerificationStatus)

		protected.GET("/", businessHandler.GetBizHome)
		protected.GET("/dashboard", businessHandler.GetDashboard)
		protected.GET("/dashboard/stats", businessHandler.GetDashboardStats)

		// Products — staff has no products permission, so require products:rw for all
		protected.GET("/products", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.GetProducts)
		protected.GET("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.GetProduct)
		protected.POST("/products", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.CreateProduct)
		protected.PUT("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.UpdateProduct)
		protected.DELETE("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.DeleteProduct)
		protected.POST("/products/:id/image", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.UploadProductImage)
		protected.GET("/products/:id/images", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.GetProductImages)
		protected.DELETE("/products/:id/images/:image_id", middleware.RequirePermission(middleware.PermProductsWrite), businessHandler.DeleteProductImage)

		// Services — staff has no services permission, so require services:rw for all
		protected.GET("/services", middleware.RequirePermission(middleware.PermServicesWrite), businessHandler.GetServices)
		protected.GET("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), businessHandler.GetService)
		protected.POST("/services", middleware.RequirePermission(middleware.PermServicesWrite), businessHandler.CreateService)
		protected.PUT("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), businessHandler.UpdateService)
		protected.DELETE("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), businessHandler.DeleteService)

		// Orders — staff can read (:r), only managers/owners can write (:rw)
		protected.GET("/orders", businessHandler.GetOrders)
		protected.GET("/orders/stats", businessHandler.GetOrdersStats)
		protected.GET("/orders/stats-grid", businessHandler.GetOrdersStatsGrid)
		protected.POST("/orders", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.CreateOrder)
		protected.PUT("/orders/:id", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.UpdateOrder)
		protected.PUT("/orders/:id/status", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.UpdateOrderStatus)

		// Bookings — staff can read (:r), only managers/owners can write (:rw)
		protected.GET("/bookings", businessHandler.GetBookings)
		protected.GET("/bookings/stats", businessHandler.GetBookingsStats)
		protected.GET("/bookings/stats-grid", businessHandler.GetBookingsStatsGrid)
		protected.GET("/bookings/:id", businessHandler.GetBooking)
		protected.POST("/bookings", middleware.RequirePermission(middleware.PermBookingsWrite), businessHandler.CreateBooking)
		protected.PUT("/bookings/:id", middleware.RequirePermission(middleware.PermBookingsWrite), businessHandler.UpdateBooking)
		protected.PUT("/bookings/:id/status", middleware.RequirePermission(middleware.PermBookingsWrite), businessHandler.UpdateBookingStatus)
		protected.PUT("/bookings/:id/paid", middleware.RequirePermission(middleware.PermBookingsWrite), businessHandler.MarkBookingAsPaid)

		// Client routes — staff can read (:r), only managers/owners can write (:rw)
		protected.POST("/clients", middleware.RequirePermission(middleware.PermClientsWrite), client.CreateClient)
		protected.DELETE("/clients/:id", middleware.RequirePermission(middleware.PermClientsWrite), client.DeleteClient)
		protected.PUT("/clients/:id/status", middleware.RequirePermission(middleware.PermClientsWrite), client.UpdateClientStatus)
		protected.GET("/clients/:id/conversation-id", client.GetClientConversationID)

		// Business Message routes — accessible to all authenticated users
		protected.GET("/clients/:id/messages", handlers.GetMessages)
		protected.POST("/clients/:id/messages", handlers.CreateMessage)
		protected.PUT("/messages/:message_id", handlers.UpdateMessage)
		protected.PUT("/messages/:message_id/read", handlers.MarkMessageAsRead)
		protected.DELETE("/messages/:message_id", handlers.DeleteMessage)
		protected.PUT("/clients/:id/read", handlers.MarkConversationAsRead)
		protected.DELETE("/clients/:id/messages", handlers.ClearChat)

		// Action routes — accessible to all authenticated users
		protected.POST("/messages/:message_id/actions", handlers.CreateAction)
		protected.POST("/messages/:message_id/actions/enhanced", handlers.CreateActionWithProgress)
		protected.GET("/actions", handlers.GetActions)
		protected.PUT("/actions/:id/status", handlers.UpdateActionStatus)

		// Conversation status route
		protected.PUT("/conversations/:conversation_id/status", handlers.UpdateConversationStatus)

		// Enhanced action routes
		protected.GET("/actions/modal/:message_id", handlers.ShowActionModal)
		protected.POST("/actions/enhanced", handlers.CreateEnhancedAction)
		protected.GET("/actions/enhanced", handlers.GetEnhancedActions)
		protected.PUT("/actions/:id/enhanced-status", handlers.UpdateEnhancedActionStatus)

		// Business widget routes — require clients:rw (writing to client context)
		protected.POST("/clients/:id/quick-booking", middleware.RequirePermission(middleware.PermClientsWrite), business.QuickBooking)
		protected.POST("/clients/:id/quick-order", middleware.RequirePermission(middleware.PermClientsWrite), business.QuickOrder)
		protected.POST("/clients/:id/request-payment", middleware.RequirePermission(middleware.PermClientsWrite), business.RequestPayment)
		protected.POST("/clients/:id/set-goal", middleware.RequirePermission(middleware.PermClientsWrite), business.SetGoal)

		// Product picker & order lifecycle routes
		protected.GET("/conversations/:conversation_id/products", businessHandler.GetConversationProducts)
		protected.GET("/conversations/:conversation_id/services", businessHandler.GetConversationServices)
		protected.POST("/conversations/:conversation_id/order-draft", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.CreateOrderDraft)
		protected.POST("/orders/:id/send", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.SendOrderToClient)
		protected.POST("/orders/:id/confirm", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.ConfirmOrderBusiness)
		protected.POST("/orders/:id/reject", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.RejectOrder)
		protected.POST("/orders/:id/fulfill", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.FulfillOrder)
		protected.PUT("/orders/:id/paid", middleware.RequirePermission(middleware.PermOrdersWrite), businessHandler.MarkOrderAsPaid)
		protected.GET("/orders/:id/receipt", businessHandler.GetOrderReceipt)
		protected.GET("/bookings/:id/receipt", businessHandler.GetBookingReceipt)

		// Share page — owner only
		protected.GET("/share", middleware.RequireOwner(), businessHandler.GetSharePage)
		protected.POST("/regenerate-slug", middleware.RequireOwner(), businessHandler.RegenerateSlug)

		// Profile & Logo upload — owner only
		protected.POST("/logo", middleware.RequireOwner(), businessHandler.UploadBusinessLogo)
		protected.PUT("/profile", middleware.RequireOwner(), businessHandler.UpdateBusinessProfile)
		protected.POST("/profile/initiate", middleware.RequireOwner(), businessHandler.InitiateProfileChange)
		protected.POST("/profile/confirm", middleware.RequireOwner(), businessHandler.ConfirmProfileChange)
		protected.POST("/profile/resend-otp", middleware.RequireOwner(), businessHandler.ResendProfileOTP)

		// Onboarding — owner only
		protected.GET("/onboarding/status", middleware.RequireOwner(), businessHandler.GetOnboardingStatus)
		protected.POST("/onboarding/progress", middleware.RequireOwner(), businessHandler.CheckOnboardingProgress)
		protected.POST("/onboarding/advance", middleware.RequireOwner(), businessHandler.AdvanceOnboarding)
		protected.POST("/onboarding/skip", middleware.RequireOwner(), businessHandler.SkipOnboarding)

		// Analytics — require analytics:view (managers have it, staff don't)
		protected.GET("/analytics", middleware.RequirePermission(middleware.PermAnalyticsView), businessHandler.GetAnalytics)
		protected.GET("/analytics/stats", middleware.RequirePermission(middleware.PermAnalyticsView), businessHandler.GetAnalyticsStats)
		protected.GET("/analytics/stats-grid", middleware.RequirePermission(middleware.PermAnalyticsView), businessHandler.GetAnalyticsStatsGrid)

		// Payments — staff has no payments permission, require payments:rw
		protected.GET("/payments", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.GetPayments)
		protected.GET("/payments/stats", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.GetPaymentsStats)
		protected.GET("/payments/stats-grid", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.GetPaymentsStatsGrid)
		protected.POST("/payment-instructions", middleware.RequireOwner(), businessHandler.UpdatePaymentInstructions)
		protected.POST("/orders/:id/payments/:payment_id/confirm", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.ConfirmOrderPayment)
		protected.POST("/orders/:id/payments/confirm-all", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.ConfirmAllOrderPayments)
		protected.POST("/orders/:id/payments/:payment_id/reject", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.RejectOrderPayment)
		protected.GET("/orders/:id/payments", businessHandler.GetOrderPayments)
		protected.POST("/bookings/:id/payments/confirm-all", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.ConfirmAllBookingPayments)
		protected.POST("/bookings/:id/payments/:payment_id/confirm", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.ConfirmBookingPayment)
		protected.POST("/bookings/:id/payments/:payment_id/reject", middleware.RequirePermission(middleware.PermPaymentsWrite), businessHandler.RejectBookingPayment)
		protected.GET("/bookings/:id/payments", businessHandler.GetBookingPayments)

		// Payment Methods — owner only (business config)
		protected.GET("/payment-methods", middleware.RequireOwner(), businessHandler.GetPaymentMethods)
		protected.POST("/payment-methods", middleware.RequireOwner(), businessHandler.CreatePaymentMethod)
		protected.PUT("/payment-methods/:id", middleware.RequireOwner(), businessHandler.UpdatePaymentMethod)
		protected.DELETE("/payment-methods/:id", middleware.RequireOwner(), businessHandler.DeletePaymentMethod)

		// Conversation progress routes — all authenticated users
		protected.GET("/conversations/:conversation_id/progress", handlers.GetConversationProgress)
		protected.PUT("/conversations/:conversation_id/stage", handlers.UpdateConversationStage)

		// Customer insight routes — all authenticated users
		protected.GET("/conversations/:conversation_id/insights-badge", handlers.GetConversationInsightsBadge)
		protected.GET("/conversations/:conversation_id/insights-panel", handlers.GetConversationInsightsPanel)
		protected.POST("/conversations/:conversation_id/insights/refresh", handlers.RefreshConversationInsights)

		// Subscription & Billing routes — owner only
		protected.GET("/subscription", middleware.RequireOwner(), businessHandler.GetSubscriptionPage)
		protected.GET("/subscription/plans", middleware.RequireOwner(), businessHandler.GetPlansPage)
		protected.GET("/subscription/checkout", middleware.RequireOwner(), businessHandler.GetCheckoutPage)
		protected.POST("/subscription/checkout", middleware.RequireOwner(), businessHandler.CreateCheckout)
		protected.POST("/subscription/change", middleware.RequireOwner(), businessHandler.ChangePlan)
		protected.POST("/subscription/cancel", middleware.RequireOwner(), businessHandler.CancelSubscription)
		protected.GET("/subscription/portal", middleware.RequireOwner(), businessHandler.BillingPortal)
		protected.GET("/subscription/badge", middleware.RequireOwner(), businessHandler.GetPlanBadge)
		protected.GET("/subscription/badge-sidebar", middleware.RequireOwner(), businessHandler.GetPlanBadgeSidebar)

		// Reports — require reports:view (managers have it, staff don't)
		protected.GET("/reports", middleware.RequirePermission(middleware.PermReportsView), businessHandler.GetReportsPage)
		protected.GET("/reports/revenue", middleware.RequirePermission(middleware.PermReportsView), businessHandler.GetRevenueReport)
		protected.GET("/reports/sales", middleware.RequirePermission(middleware.PermReportsView), businessHandler.GetSalesReport)
		protected.GET("/reports/clients", middleware.RequirePermission(middleware.PermReportsView), businessHandler.GetClientReport)
		protected.GET("/reports/tax", middleware.RequirePermission(middleware.PermReportsView), businessHandler.GetTaxReport)
		protected.GET("/reports/export/orders.csv", middleware.RequirePermission(middleware.PermReportsView), businessHandler.ExportOrdersCSV)
		protected.GET("/reports/export/bookings.csv", middleware.RequirePermission(middleware.PermReportsView), businessHandler.ExportBookingsCSV)
		protected.GET("/reports/export/payments.csv", middleware.RequirePermission(middleware.PermReportsView), businessHandler.ExportPaymentsCSV)
		protected.GET("/reports/export/revenue.csv", middleware.RequirePermission(middleware.PermReportsView), businessHandler.ExportRevenueCSV)
		protected.GET("/reports/export/clients.csv", middleware.RequirePermission(middleware.PermReportsView), businessHandler.ExportClientsCSV)

		// Reviews — require clients:rw (client-facing feature)
		protected.GET("/reviews", middleware.RequirePermission(middleware.PermClientsWrite), businessHandler.GetReviews)
		protected.POST("/reviews/:id/reply", middleware.RequirePermission(middleware.PermClientsWrite), businessHandler.ReplyToReview)

		// Business Hours routes — owner only
		protected.GET("/hours", middleware.RequireOwner(), businessHandler.GetBusinessHours)
		protected.PUT("/hours", middleware.RequireOwner(), businessHandler.UpdateBusinessHours)
		protected.PUT("/hours/special", middleware.RequireOwner(), businessHandler.UpdateSpecialHours)
		protected.POST("/hours/toggle", middleware.RequireOwner(), businessHandler.ToggleAcceptingBookings)

		// Location routes — owner only
		protected.GET("/locations", middleware.RequireOwner(), businessHandler.GetLocations)
		protected.POST("/locations", middleware.RequireOwner(), businessHandler.CreateLocation)
		protected.PUT("/locations/:id", middleware.RequireOwner(), businessHandler.UpdateLocation)
		protected.DELETE("/locations/:id", middleware.RequireOwner(), businessHandler.DeleteLocation)

		// Team routes — owner only
		protected.GET("/team", middleware.RequireOwner(), businessHandler.GetTeam)
		protected.POST("/team", middleware.RequireOwner(), businessHandler.InviteTeamMember)
		protected.PUT("/team/:id", middleware.RequireOwner(), businessHandler.UpdateTeamMember)
		protected.DELETE("/team/:id", middleware.RequireOwner(), businessHandler.DeleteTeamMember)

		// Assist AI routes — all authenticated users
		protected.POST("/assist/chat", businessHandler.AssistChat)
		protected.GET("/assist/suggestions", businessHandler.GetAssistSuggestions)

		// Notification routes — all authenticated users
		protected.GET("/notifications", businessHandler.GetNotifications)
		protected.GET("/notifications/count", businessHandler.GetNotificationCount)
		protected.POST("/notifications/:id/read", businessHandler.MarkNotificationRead)
		protected.DELETE("/notifications/:id", businessHandler.DeleteNotification)
		protected.POST("/notifications/read-all", businessHandler.MarkAllNotificationsRead)
		protected.GET("/notification-settings", businessHandler.GetNotificationSettings)
		protected.PUT("/notification-settings", businessHandler.UpdateNotificationSettings)
	}

	// Public team routes (outside auth)
	r.GET("/business/team/login", businessHandler.ShowTeamLogin)
	r.POST("/business/team/login", businessHandler.TeamLogin)
	r.GET("/business/team/logout", businessHandler.TeamLogout)
	r.GET("/business/team/accept", businessHandler.ShowAcceptInvite)
	r.POST("/business/team/accept", businessHandler.AcceptInvite)

	// Webhooks (public)
	r.POST("/stripe/webhook", business.StripeWebhook(businessHandler))
	r.POST("/paddle/webhook", business.PaddleWebhook(businessHandler))

}
