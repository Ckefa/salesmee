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
		clientProtected.GET("/businesses/:business_id/products", businessHandler.GetBusinessProducts)
		clientProtected.GET("/businesses/:business_id/products-page", businessHandler.ShowClientProductsPage)
		clientProtected.GET("/products/:id/images", businessHandler.GetClientProductImages)
		clientProtected.GET("/businesses/:business_id/services", businessHandler.GetBusinessServices)
		clientProtected.POST("/businesses/:business_id/bookings", businessHandler.ClientCreateBooking)
		clientProtected.POST("/orders", businessHandler.ClientCreateOrder)
		clientProtected.POST("/orders/:id/confirm", client.ClientConfirmOrder)
		clientProtected.POST("/orders/:id/cancel", client.ClientCancelOrder)
		clientProtected.POST("/orders/:id/update", client.ClientUpdateOrder)
		clientProtected.POST("/bookings", businessHandler.ClientCreateBooking)
		clientProtected.POST("/bookings/:id/update", client.ClientUpdateBooking)
		clientProtected.POST("/bookings/:id/cancel", client.ClientCancelBooking)
		clientProtected.PUT("/businesses/:business_id/read", handlers.MarkClientConversationAsRead)
		clientProtected.POST("/disconnect/:business_id", client.DisconnectFromBusiness)
		clientProtected.POST("/heartbeat", client.ClientHeartbeat)
	}

}
