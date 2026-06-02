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
	businessHandler = business.NewBusinessHandler(db.DB)

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

		protected.GET("/", businessHandler.GetBizHome)
		protected.GET("/dashboard", businessHandler.GetDashboard)
		protected.GET("/products", businessHandler.GetProducts)
		protected.GET("/products/:id", businessHandler.GetProduct)
		protected.POST("/products", businessHandler.CreateProduct)
		protected.PUT("/products/:id", businessHandler.UpdateProduct)
		protected.DELETE("/products/:id", businessHandler.DeleteProduct)
		protected.POST("/products/:id/image", businessHandler.UploadProductImage)
		protected.GET("/products/:id/images", businessHandler.GetProductImages)
		protected.DELETE("/products/:id/images/:image_id", businessHandler.DeleteProductImage)
		protected.GET("/services", businessHandler.GetServices)
		protected.GET("/services/:id", businessHandler.GetService)
		protected.POST("/services", businessHandler.CreateService)
		protected.PUT("/services/:id", businessHandler.UpdateService)
		protected.DELETE("/services/:id", businessHandler.DeleteService)
		protected.GET("/orders", businessHandler.GetOrders)
		protected.POST("/orders", businessHandler.CreateOrder)
		protected.PUT("/orders/:id", businessHandler.UpdateOrder)
		protected.PUT("/orders/:id/status", businessHandler.UpdateOrderStatus)
		protected.GET("/bookings", businessHandler.GetBookings)
		protected.GET("/bookings/:id", businessHandler.GetBooking)
		protected.POST("/bookings", businessHandler.CreateBooking)
		protected.PUT("/bookings/:id", businessHandler.UpdateBooking)
		protected.PUT("/bookings/:id/status", businessHandler.UpdateBookingStatus)

		// client routes
		protected.POST("/clients", client.CreateClient)
		protected.DELETE("/clients/:id", client.DeleteClient)
		protected.PUT("/clients/:id/status", client.UpdateClientStatus)
		protected.GET("/clients/:id/conversation-id", client.GetClientConversationID)

		// Business Message routes
		protected.GET("/clients/:id/messages", handlers.GetMessages)
		protected.POST("/clients/:id/messages", handlers.CreateMessage)
		protected.PUT("/messages/:message_id", handlers.UpdateMessage)
		protected.PUT("/clients/:id/read", handlers.MarkConversationAsRead)

		// Action routes
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

		// Business widget routes
		protected.POST("/clients/:id/quick-booking", business.QuickBooking)
		protected.POST("/clients/:id/quick-order", business.QuickOrder)
		protected.POST("/clients/:id/request-payment", business.RequestPayment)
		protected.POST("/clients/:id/set-goal", business.SetGoal)

		// Product picker & order lifecycle routes
		protected.GET("/conversations/:conversation_id/products", businessHandler.GetConversationProducts)
		protected.GET("/conversations/:conversation_id/services", businessHandler.GetConversationServices)
		protected.POST("/conversations/:conversation_id/order-draft", businessHandler.CreateOrderDraft)
		protected.POST("/orders/:id/send", businessHandler.SendOrderToClient)
		protected.POST("/orders/:id/confirm", businessHandler.ConfirmOrderBusiness)
		protected.POST("/orders/:id/reject", businessHandler.RejectOrder)
		protected.POST("/orders/:id/fulfill", businessHandler.FulfillOrder)

		// Share page
		protected.GET("/share", businessHandler.GetSharePage)
		protected.POST("/regenerate-slug", businessHandler.RegenerateSlug)

		// Profile & Logo upload
		protected.POST("/logo", businessHandler.UploadBusinessLogo)
		protected.PUT("/profile", businessHandler.UpdateBusinessProfile)
		protected.POST("/profile/initiate", businessHandler.InitiateProfileChange)
		protected.POST("/profile/confirm", businessHandler.ConfirmProfileChange)
		protected.POST("/profile/resend-otp", businessHandler.ResendProfileOTP)

		// Analytics
		protected.GET("/analytics", businessHandler.GetAnalytics)

		// Payments
		protected.GET("/payments", businessHandler.GetPayments)
		protected.POST("/payment-instructions", businessHandler.UpdatePaymentInstructions)
		protected.POST("/orders/:id/payments/:payment_id/confirm", businessHandler.ConfirmOrderPayment)
		protected.POST("/orders/:id/payments/confirm-all", businessHandler.ConfirmAllOrderPayments)
		protected.POST("/orders/:id/payments/:payment_id/reject", businessHandler.RejectOrderPayment)
		protected.GET("/orders/:id/payments", businessHandler.GetOrderPayments)
		protected.POST("/bookings/:id/payments/:payment_id/confirm", businessHandler.ConfirmBookingPayment)
		protected.POST("/bookings/:id/payments/:payment_id/reject", businessHandler.RejectBookingPayment)
		protected.GET("/bookings/:id/payments", businessHandler.GetBookingPayments)

		// Payment Methods (structured, multi-type)
		protected.GET("/payment-methods", businessHandler.GetPaymentMethods)
		protected.POST("/payment-methods", businessHandler.CreatePaymentMethod)
		protected.PUT("/payment-methods/:id", businessHandler.UpdatePaymentMethod)
		protected.DELETE("/payment-methods/:id", businessHandler.DeletePaymentMethod)

		// Conversation progress routes
		protected.GET("/conversations/:conversation_id/progress", handlers.GetConversationProgress)
		protected.PUT("/conversations/:conversation_id/stage", handlers.UpdateConversationStage)

		// Subscription & Billing routes
		protected.GET("/subscription", businessHandler.GetSubscriptionPage)
		protected.GET("/subscription/plans", businessHandler.GetPlansPage)
		protected.GET("/subscription/checkout", businessHandler.GetCheckoutPage)
		protected.POST("/subscription/checkout", businessHandler.CreateCheckout)
		protected.POST("/subscription/change", businessHandler.ChangePlan)
		protected.POST("/subscription/cancel", businessHandler.CancelSubscription)
		protected.GET("/subscription/portal", businessHandler.BillingPortal)
		protected.GET("/subscription/badge", businessHandler.GetPlanBadge)
		protected.GET("/subscription/badge-sidebar", businessHandler.GetPlanBadgeSidebar)
	}

	// Webhooks (public)
	r.POST("/stripe/webhook", business.StripeWebhook(businessHandler))
	r.POST("/paypal/webhook", business.PayPalWebhook(businessHandler))

}
