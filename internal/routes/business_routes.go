package routes

import (
	"time"

	"salesmee/internal/db"
	"salesmee/internal/handlers"
	"salesmee/internal/handlers/business"
	"salesmee/internal/handlers/client"
	"salesmee/internal/middleware"
	"salesmee/internal/services/cache"

	"github.com/gin-gonic/gin"
)

var bizHandler *business.BusinessHandler
var productHandler *business.ProductHandler
var serviceHandler *business.ServiceHandler
var orderHandler *business.OrderHandler
var bookingHandler *business.BookingHandler
var paymentHandler *business.PaymentHandler
var analyticsHandler *business.AnalyticsHandler
var reportHandler *business.ReportHandler
var hoursHandler *business.HoursHandler
var locationHandler *business.LocationHandler
var teamHandler *business.TeamHandler
var reviewHandler *business.ReviewHandler
var subscriptionHandler *business.SubscriptionHandler
var assistHandler *business.AssistHandler
var notificationHandler *business.NotificationHandler

func SetupBusinessRoutes(r *gin.Engine) {
	fcache := cache.NewFragmentCache(15 * time.Second)
	deps := &business.HandlerDeps{DB: db.DB, Hub: wsHub, FCache: fcache}

	bizHandler = business.NewBusinessHandler(deps)
	productHandler = business.NewProductHandler(deps)
	serviceHandler = business.NewServiceHandler(deps)
	orderHandler = business.NewOrderHandler(deps)
	bookingHandler = business.NewBookingHandler(deps)
	paymentHandler = business.NewPaymentHandler(deps)
	analyticsHandler = business.NewAnalyticsHandler(deps)
	reportHandler = business.NewReportHandler(deps)
	hoursHandler = business.NewHoursHandler(deps)
	locationHandler = business.NewLocationHandler(deps)
	teamHandler = business.NewTeamHandler(deps)
	reviewHandler = business.NewReviewHandler(deps)
	subscriptionHandler = business.NewSubscriptionHandler(deps)
	assistHandler = business.NewAssistHandler(deps)
	notificationHandler = business.NewNotificationHandler(deps)

	r.GET("/business/login", handlers.ShowLogin)
	r.GET("/business/register", handlers.ShowRegisterStep1)
	r.POST("/business/register", handlers.RegisterStep1)
	r.GET("/business/register/step2", handlers.ShowRegisterStep2)
	r.POST("/business/register/step2", handlers.RegisterStep2)
	r.GET("/business/register/step3", handlers.ShowRegisterStep3)
	r.POST("/business/register/step3", handlers.RegisterStep3)
	r.POST("/business/login", handlers.Login)
	r.GET("/business/logout", handlers.Logout)

	r.GET("/business/forgot-password", handlers.ShowForgotPassword)
	r.POST("/business/forgot-password", handlers.SendForgotPassword)
	r.GET("/business/reset-password", handlers.ShowResetPassword)
	r.POST("/business/reset-password", handlers.SubmitResetPassword)

	r.GET("/business/verify", handlers.VerifyBusinessEmail)

	r.GET("/business/auth/google", handlers.InitiateBusinessGoogleAuth)
	r.GET("/business/auth/google/callback", handlers.HandleBusinessGoogleCallback)
	r.GET("/business/register/google", handlers.ShowRegisterGoogle)
	r.POST("/business/register/google/complete", handlers.CompleteRegisterGoogle)

	r.GET("/business/auth/facebook", handlers.InitiateBusinessFacebookAuth)
	r.GET("/business/auth/facebook/callback", handlers.HandleBusinessFacebookCallback)

	protected := r.Group("/business")
	protected.Use(middleware.BizzMiddleware())
	{
		protected.POST("/verify/send", handlers.SendBusinessVerification)
		protected.POST("/verify/status", handlers.CheckVerificationStatus)

		protected.GET("/", bizHandler.GetBizHome)
		protected.GET("/dashboard", bizHandler.GetDashboard)
		protected.GET("/dashboard/stats", bizHandler.GetDashboardStats)

		protected.GET("/products", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.GetProducts)
		protected.GET("/products/quick-list", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.GetProductsQuickList)
		protected.GET("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.GetProduct)
		protected.POST("/products", middleware.RequirePermission(middleware.PermProductsWrite), middleware.CheckResourceLimit("product", "products"), productHandler.CreateProduct)
		protected.PUT("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.UpdateProduct)
		protected.DELETE("/products/:id", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.DeleteProduct)
		protected.POST("/products/:id/image", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.UploadProductImage)
		protected.GET("/products/:id/images", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.GetProductImages)
		protected.DELETE("/products/:id/images/:image_id", middleware.RequirePermission(middleware.PermProductsWrite), productHandler.DeleteProductImage)

		protected.GET("/services", middleware.RequirePermission(middleware.PermServicesWrite), serviceHandler.GetServices)
		protected.GET("/services/quick-list", middleware.RequirePermission(middleware.PermServicesWrite), serviceHandler.GetServicesQuickList)
		protected.GET("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), serviceHandler.GetService)
		protected.POST("/services", middleware.RequirePermission(middleware.PermServicesWrite), middleware.CheckResourceLimit("service", "services"), serviceHandler.CreateService)
		protected.PUT("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), serviceHandler.UpdateService)
		protected.DELETE("/services/:id", middleware.RequirePermission(middleware.PermServicesWrite), serviceHandler.DeleteService)

		protected.GET("/orders", orderHandler.GetOrders)
		protected.GET("/orders/stats", orderHandler.GetOrdersStats)
		protected.GET("/orders/stats-grid", orderHandler.GetOrdersStatsGrid)
		protected.POST("/orders", middleware.RequirePermission(middleware.PermOrdersWrite), middleware.RequireFeature("orders_and_bookings", "Orders & Bookings"), orderHandler.CreateOrder)
		protected.PUT("/orders/:id", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.UpdateOrder)
		protected.PUT("/orders/:id/status", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.UpdateOrderStatus)

		protected.GET("/bookings", bookingHandler.GetBookings)
		protected.GET("/bookings/stats", bookingHandler.GetBookingsStats)
		protected.GET("/bookings/stats-grid", bookingHandler.GetBookingsStatsGrid)
		protected.GET("/bookings/:id", bookingHandler.GetBooking)
		protected.POST("/bookings", middleware.RequirePermission(middleware.PermBookingsWrite), middleware.RequireFeature("orders_and_bookings", "Orders & Bookings"), bookingHandler.CreateBooking)
		protected.PUT("/bookings/:id", middleware.RequirePermission(middleware.PermBookingsWrite), bookingHandler.UpdateBooking)
		protected.PUT("/bookings/:id/status", middleware.RequirePermission(middleware.PermBookingsWrite), bookingHandler.UpdateBookingStatus)
		protected.PUT("/bookings/:id/paid", middleware.RequirePermission(middleware.PermBookingsWrite), bookingHandler.MarkBookingAsPaid)

		protected.POST("/clients", middleware.RequirePermission(middleware.PermClientsWrite), middleware.CheckResourceLimit("client", "clients"), client.CreateClient)
		protected.DELETE("/clients/:id", middleware.RequirePermission(middleware.PermClientsWrite), client.DeleteClient)
		protected.PUT("/clients/:id/status", middleware.RequirePermission(middleware.PermClientsWrite), client.UpdateClientStatus)
		protected.GET("/clients/:id/conversation-id", client.GetClientConversationID)

		protected.GET("/clients/:id/messages", handlers.GetMessages)
		protected.POST("/clients/:id/messages", handlers.CreateMessage)
		protected.PUT("/messages/:message_id", handlers.UpdateMessage)
		protected.PUT("/messages/:message_id/read", handlers.MarkMessageAsRead)
		protected.DELETE("/messages/:message_id", handlers.DeleteMessage)
		protected.PUT("/clients/:id/read", handlers.MarkConversationAsRead)
		protected.DELETE("/clients/:id/messages", handlers.ClearChat)

		protected.POST("/messages/:message_id/actions", handlers.CreateAction)
		protected.POST("/messages/:message_id/actions/enhanced", handlers.CreateActionWithProgress)
		protected.GET("/actions", handlers.GetActions)
		protected.PUT("/actions/:id/status", handlers.UpdateActionStatus)

		protected.PUT("/conversations/:conversation_id/status", handlers.UpdateConversationStatus)

		protected.GET("/actions/modal/:message_id", handlers.ShowActionModal)
		protected.POST("/actions/enhanced", handlers.CreateEnhancedAction)
		protected.GET("/actions/enhanced", handlers.GetEnhancedActions)
		protected.PUT("/actions/:id/enhanced-status", handlers.UpdateEnhancedActionStatus)

		protected.POST("/clients/:id/quick-booking", middleware.RequirePermission(middleware.PermClientsWrite), middleware.RequireFeature("orders_and_bookings", "Orders & Bookings"), business.QuickBooking)
		protected.POST("/clients/:id/quick-order", middleware.RequirePermission(middleware.PermClientsWrite), middleware.RequireFeature("orders_and_bookings", "Orders & Bookings"), business.QuickOrder)
		protected.POST("/clients/:id/request-payment", middleware.RequirePermission(middleware.PermClientsWrite), business.RequestPayment)
		protected.POST("/clients/:id/set-goal", middleware.RequirePermission(middleware.PermClientsWrite), business.SetGoal)

		protected.GET("/conversations/:conversation_id/products", orderHandler.GetConversationProducts)
		protected.GET("/conversations/:conversation_id/services", orderHandler.GetConversationServices)
		protected.POST("/conversations/:conversation_id/order-draft", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.CreateOrderDraft)
		protected.POST("/orders/:id/send", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.SendOrderToClient)
		protected.POST("/orders/:id/confirm", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.ConfirmOrderBusiness)
		protected.POST("/orders/:id/reject", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.RejectOrder)
		protected.POST("/orders/:id/fulfill", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.FulfillOrder)
		protected.PUT("/orders/:id/paid", middleware.RequirePermission(middleware.PermOrdersWrite), orderHandler.MarkOrderAsPaid)
		protected.GET("/orders/:id/receipt", orderHandler.GetOrderReceipt)
		protected.GET("/bookings/:id/receipt", bookingHandler.GetBookingReceipt)

		protected.GET("/share", middleware.RequirePermission(middleware.PermShareView), bizHandler.GetSharePage)
		protected.POST("/regenerate-slug", middleware.RequireOwner(), bizHandler.RegenerateSlug)

		protected.POST("/logo", middleware.RequireOwner(), bizHandler.UploadBusinessLogo)
		protected.PUT("/profile", middleware.RequireOwner(), bizHandler.UpdateBusinessProfile)
		protected.POST("/profile/initiate", middleware.RequireOwner(), bizHandler.InitiateProfileChange)
		protected.POST("/profile/confirm", middleware.RequireOwner(), bizHandler.ConfirmProfileChange)
		protected.POST("/profile/resend-otp", middleware.RequireOwner(), bizHandler.ResendProfileOTP)

		protected.GET("/onboarding/status", middleware.RequireOwner(), bizHandler.GetOnboardingStatus)
		protected.POST("/onboarding/progress", middleware.RequireOwner(), bizHandler.CheckOnboardingProgress)
		protected.POST("/onboarding/advance", middleware.RequireOwner(), bizHandler.AdvanceOnboarding)
		protected.POST("/onboarding/skip", middleware.RequireOwner(), bizHandler.SkipOnboarding)
		protected.GET("/analytics", middleware.RequirePermission(middleware.PermAnalyticsView), analyticsHandler.GetAnalytics)

		protected.GET("/analytics/stats", middleware.RequirePermission(middleware.PermAnalyticsView), analyticsHandler.GetAnalyticsStats)

		protected.GET("/analytics/stats-grid", middleware.RequirePermission(middleware.PermAnalyticsView), analyticsHandler.GetAnalyticsStatsGrid)
		protected.GET("/payments", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.GetPayments)
		protected.GET("/payments/stats", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.GetPaymentsStats)
		protected.GET("/payments/stats-grid", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.GetPaymentsStatsGrid)
		protected.POST("/payment-instructions", middleware.RequireOwner(), paymentHandler.UpdatePaymentInstructions)
		protected.POST("/orders/:id/payments/:payment_id/confirm", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.ConfirmOrderPayment)
		protected.POST("/orders/:id/payments/confirm-all", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.ConfirmAllOrderPayments)
		protected.POST("/orders/:id/payments/:payment_id/reject", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.RejectOrderPayment)
		protected.GET("/orders/:id/payments", paymentHandler.GetOrderPayments)
		protected.POST("/bookings/:id/payments/confirm-all", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.ConfirmAllBookingPayments)
		protected.POST("/bookings/:id/payments/:payment_id/confirm", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.ConfirmBookingPayment)
		protected.POST("/bookings/:id/payments/:payment_id/reject", middleware.RequirePermission(middleware.PermPaymentsWrite), paymentHandler.RejectBookingPayment)
		protected.GET("/bookings/:id/payments", paymentHandler.GetBookingPayments)

		protected.GET("/payment-methods", middleware.RequireOwner(), paymentHandler.GetPaymentMethods)
		protected.POST("/payment-methods", middleware.RequireOwner(), paymentHandler.CreatePaymentMethod)
		protected.PUT("/payment-methods/:id", middleware.RequireOwner(), paymentHandler.UpdatePaymentMethod)
		protected.DELETE("/payment-methods/:id", middleware.RequireOwner(), paymentHandler.DeletePaymentMethod)

		protected.GET("/conversations/:conversation_id/progress", handlers.GetConversationProgress)
		protected.PUT("/conversations/:conversation_id/stage", handlers.UpdateConversationStage)

		protected.GET("/conversations/:conversation_id/insights-badge", handlers.GetConversationInsightsBadge)
		protected.GET("/conversations/:conversation_id/insights-panel", handlers.GetConversationInsightsPanel)
		protected.POST("/conversations/:conversation_id/insights/refresh", handlers.RefreshConversationInsights)

		protected.GET("/subscription", middleware.RequirePermission(middleware.PermSubscriptionView), subscriptionHandler.GetSubscriptionPage)
		protected.GET("/subscription/plans", middleware.RequirePermission(middleware.PermSubscriptionView), subscriptionHandler.GetPlansPage)
		protected.GET("/subscription/checkout", middleware.RequirePermission(middleware.PermSubscriptionView), subscriptionHandler.GetCheckoutPage)
		protected.POST("/subscription/checkout", middleware.RequireOwner(), subscriptionHandler.CreateCheckout)
		protected.POST("/subscription/change", middleware.RequireOwner(), subscriptionHandler.ChangePlan)
		protected.POST("/subscription/cancel", middleware.RequireOwner(), subscriptionHandler.CancelSubscription)
		protected.GET("/subscription/portal", middleware.RequireOwner(), subscriptionHandler.BillingPortal)
		protected.GET("/subscription/badge", middleware.RequirePermission(middleware.PermSubscriptionView), subscriptionHandler.GetPlanBadge)
		protected.GET("/subscription/badge-sidebar", middleware.RequirePermission(middleware.PermSubscriptionView), subscriptionHandler.GetPlanBadgeSidebar)

		protected.GET("/reports", middleware.RequirePermission(middleware.PermReportsView), reportHandler.GetReportsPage)
		protected.GET("/reports/revenue", middleware.RequirePermission(middleware.PermReportsView), reportHandler.GetRevenueReport)
		protected.GET("/reports/sales", middleware.RequirePermission(middleware.PermReportsView), reportHandler.GetSalesReport)
		protected.GET("/reports/clients", middleware.RequirePermission(middleware.PermReportsView), reportHandler.GetClientReport)
		protected.GET("/reports/tax", middleware.RequirePermission(middleware.PermReportsView), reportHandler.GetTaxReport)
		protected.GET("/reports/export/orders.csv", middleware.RequirePermission(middleware.PermReportsView), reportHandler.ExportOrdersCSV)
		protected.GET("/reports/export/bookings.csv", middleware.RequirePermission(middleware.PermReportsView), reportHandler.ExportBookingsCSV)
		protected.GET("/reports/export/payments.csv", middleware.RequirePermission(middleware.PermReportsView), reportHandler.ExportPaymentsCSV)
		protected.GET("/reports/export/revenue.csv", middleware.RequirePermission(middleware.PermReportsView), reportHandler.ExportRevenueCSV)
		protected.GET("/reports/export/clients.csv", middleware.RequirePermission(middleware.PermReportsView), reportHandler.ExportClientsCSV)

		protected.GET("/reviews", middleware.RequirePermission(middleware.PermClientsWrite), reviewHandler.GetReviews)
		protected.POST("/reviews/:id/reply", middleware.RequirePermission(middleware.PermClientsWrite), reviewHandler.ReplyToReview)

		protected.GET("/hours", middleware.RequireOwner(), hoursHandler.GetBusinessHours)
		protected.PUT("/hours", middleware.RequireOwner(), hoursHandler.UpdateBusinessHours)
		protected.PUT("/hours/special", middleware.RequireOwner(), hoursHandler.UpdateSpecialHours)
		protected.POST("/hours/toggle", middleware.RequireOwner(), hoursHandler.ToggleAcceptingBookings)

		protected.GET("/locations", middleware.RequireOwner(), locationHandler.GetLocations)
		protected.POST("/locations", middleware.RequireOwner(), locationHandler.CreateLocation)
		protected.PUT("/locations/:id", middleware.RequireOwner(), locationHandler.UpdateLocation)
		protected.DELETE("/locations/:id", middleware.RequireOwner(), locationHandler.DeleteLocation)

		protected.GET("/team", middleware.RequireOwner(), teamHandler.GetTeam)
		protected.POST("/team", middleware.RequireOwner(), teamHandler.InviteTeamMember)
		protected.PUT("/team/:id", middleware.RequireOwner(), teamHandler.UpdateTeamMember)
		protected.DELETE("/team/:id", middleware.RequireOwner(), teamHandler.DeleteTeamMember)

		protected.POST("/assist/chat", assistHandler.AssistChat)
		protected.GET("/assist/suggestions", assistHandler.GetAssistSuggestions)

		protected.GET("/notifications", bizHandler.GetNotifications)
		protected.GET("/notifications/count", bizHandler.GetNotificationCount)
		protected.POST("/notifications/:id/read", bizHandler.MarkNotificationRead)
		protected.DELETE("/notifications/:id", bizHandler.DeleteNotification)
		protected.POST("/notifications/read-all", bizHandler.MarkAllNotificationsRead)
		protected.GET("/notification-settings", notificationHandler.GetNotificationSettings)
		protected.PUT("/notification-settings", notificationHandler.UpdateNotificationSettings)
	}

	r.GET("/business/team/login", teamHandler.ShowTeamLogin)
	r.POST("/business/team/login", teamHandler.TeamLogin)
	r.GET("/business/team/logout", teamHandler.TeamLogout)
	r.GET("/business/team/accept", teamHandler.ShowAcceptInvite)
	r.POST("/business/team/accept", teamHandler.AcceptInvite)

	r.POST("/stripe/webhook", business.StripeWebhook(bizHandler))
	r.POST("/paddle/webhook", business.PaddleWebhook(bizHandler))
	r.POST("/polar/webhook", business.PolarWebhook(bizHandler))
}
