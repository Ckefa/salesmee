package routes

import (
	"salesmee/internal/handlers"
	"salesmee/internal/handlers/business"
	"salesmee/internal/handlers/client"
	"salesmee/internal/ws"

	"github.com/gin-gonic/gin"
)

var wsHub *ws.Hub

func SetWSHub(hub *ws.Hub) {
	wsHub = hub
}

func Setup(r *gin.Engine) {
	// Main routes
	r.GET("/", handlers.HomePage)

	// SEO
	r.GET("/sitemap.xml", handlers.SitemapXML)
	r.GET("/robots.txt", handlers.RobotsTXT)

	// 404 handler
	r.NoRoute(handlers.Show404)

	// API
	api := r.Group("/api/v1")
	{
		api.GET("/ping", handlers.Ping)
	}

	// Test template rendering
	r.GET("/test-template", func(c *gin.Context) {
		c.HTML(200, "minimal.html", gin.H{
			"Title": "Test Template",
		})
	})

	// Simple test route
	r.GET("/simple", func(c *gin.Context) {
		c.HTML(200, "simple.html", gin.H{
			"Title": "Simple Test",
		})
	})

	// DEV ONLY
	if gin.Mode() == gin.DebugMode {
		dev := r.Group("/test")
		{
			dev.GET("", handlers.DevPage)
			dev.GET("/items", handlers.ListItems)
			dev.POST("/items", handlers.CreateItem)
			dev.DELETE("/items/:id", handlers.DeleteItem)
		}
	}

	// Guide
	r.GET("/guide", handlers.ShowGuide)

	// Legal pages
	r.GET("/privacy", handlers.ShowPrivacy)
	r.GET("/terms", handlers.ShowTerms)
	r.GET("/cookies", handlers.ShowCookies)
	r.GET("/refund-policy", handlers.ShowRefund)
	r.GET("/user-deletion", handlers.ShowUserDeletion)
	r.POST("/user-deletion", handlers.SubmitUserDeletion)

	// Landing AI assistant (public)
	r.GET("/assist/suggestions", handlers.LandingGetAssistSuggestions)
	r.POST("/assist/chat", handlers.LandingAssistChat)

	// Public business profile
	r.GET("/b/:slug", business.GetPublicProfile)

	// Public connect flow (self-registration via slug)
	r.GET("/api/connect/:slug", business.ShowConnect)
	r.POST("/api/connect/:slug/send-otp", business.SendConnectOTP)
	r.POST("/api/connect/:slug/verify-otp", business.VerifyConnectOTP)

	// Wire WebSocket hub into handler packages
	if wsHub != nil {
		handlers.SetWSHub(wsHub)
		client.SetWSHub(wsHub)
	}

	// WebSocket routes (auth via query param token)
	if wsHub != nil {
		r.GET("/ws/business", ws.ServeBusinessWS(wsHub))
		r.GET("/ws/client", ws.ServeClientWS(wsHub))
	}

}
