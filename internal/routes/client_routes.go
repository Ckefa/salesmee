package routes

import (
	"salesmee/internal/handlers"
	"salesmee/internal/handlers/client"

	"github.com/gin-gonic/gin"
)

func SetupClientRoutes(r *gin.Engine) {

	// PUBLIC - client Route
	r.GET("/client/login", client.ShowClientLogin)
	r.POST("/client/send-otp", client.SendClientOTP)
	r.POST("/client/verify-otp", client.VerifyClientOTP)
	r.GET("/client/logout", client.ClientLogout)

	// PUBLIC - Client Google Auth
	r.GET("/client/auth/google", client.InitiateClientGoogleAuth)
	r.GET("/client/auth/google/callback", client.HandleClientGoogleCallback)

	// PUBLIC - Client Facebook Auth
	r.GET("/client/auth/facebook", client.InitiateClientFacebookAuth)
	r.GET("/client/auth/facebook/callback", client.HandleClientFacebookCallback)

	// PROTECTED client ROUTES
	clientProtected := r.Group("/client")
	clientProtected.Use(client.ClientMiddleware())
	{
		clientProtected.GET("/", client.ClientDashboard)
		clientProtected.GET("/discover", client.ShowDiscover)
		clientProtected.GET("/discover-page", client.ShowDiscover)
		clientProtected.GET("/discover/search", client.SearchBusinesses)
		clientProtected.POST("/connect/:business_id", client.ConnectToBusiness)
		clientProtected.GET("/businesses/:business_id/messages", client.GetClientMessages)
		clientProtected.POST("/businesses/:business_id/messages", client.CreateClientMessage)
		clientProtected.GET("/businesses/:business_id/products", productHandler.GetBusinessProducts)
		clientProtected.GET("/businesses/:business_id/products-page", productHandler.ShowClientProductsPage)
		clientProtected.GET("/businesses/:business_id/services-page", serviceHandler.ShowClientServicesPage)
		clientProtected.GET("/products/:id/images", productHandler.GetClientProductImages)
		clientProtected.GET("/businesses/:business_id/services", serviceHandler.GetBusinessServices)
		clientProtected.POST("/businesses/:business_id/bookings", bookingHandler.ClientCreateBooking)
		clientProtected.POST("/orders", orderHandler.ClientCreateOrder)
		clientProtected.POST("/orders/:id/confirm", client.ClientConfirmOrder)
		clientProtected.POST("/orders/:id/cancel", client.ClientCancelOrder)
		clientProtected.POST("/orders/:id/update", client.ClientUpdateOrder)
		clientProtected.POST("/bookings", bookingHandler.ClientCreateBooking)
		clientProtected.POST("/bookings/:id/update", client.ClientUpdateBooking)
		clientProtected.POST("/bookings/:id/cancel", client.ClientCancelBooking)
		clientProtected.POST("/bookings/:id/confirm", client.ClientConfirmBooking)
		clientProtected.PUT("/businesses/:business_id/read", handlers.MarkClientConversationAsRead)
		clientProtected.POST("/disconnect/:business_id", client.DisconnectFromBusiness)
		clientProtected.POST("/reviews", client.SubmitReview)
		clientProtected.POST("/orders/:id/payment", paymentHandler.ClientSubmitOrderPayment)
		clientProtected.POST("/bookings/:id/payment", paymentHandler.ClientSubmitBookingPayment)
		clientProtected.DELETE("/messages/:message_id", client.DeleteClientMessage)
		clientProtected.POST("/assist/chat", client.ClientAssistChat)
		clientProtected.GET("/assist/suggestions", client.ClientGetAssistSuggestions)
	}

}
