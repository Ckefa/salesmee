package routes

import (
	"oneflow/internal/handlers"
	"oneflow/internal/handlers/business"

	"github.com/gin-gonic/gin"
)

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
	r.GET("/user-deletion", handlers.ShowUserDeletion)
	r.POST("/user-deletion", handlers.SubmitUserDeletion)

	// Public business profile
	r.GET("/b/:slug", business.GetPublicProfile)

	// Public connect flow (self-registration via slug)
	r.GET("/api/connect/:slug", business.ShowConnect)
	r.POST("/api/connect/:slug/send-otp", business.SendConnectOTP)
	r.POST("/api/connect/:slug/verify-otp", business.VerifyConnectOTP)

}
