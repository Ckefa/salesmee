package routes

import (
	"salesmee/internal/handlers/admin"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(r *gin.Engine) {
	// Public admin login
	r.GET("/admin/login", admin.ShowLogin)
	r.POST("/admin/login", admin.Login)

	// Protected admin routes
	adminGroup := r.Group("/admin")
	adminGroup.Use(admin.AdminMiddleware())
	{
		adminGroup.GET("", admin.ShowDashboard)
		adminGroup.GET("/businesses", admin.ListBusinesses)
		adminGroup.POST("/businesses/:id/suspend", admin.SuspendBusiness)
		adminGroup.POST("/businesses/:id/activate", admin.ActivateBusiness)
		adminGroup.GET("/subscriptions", admin.ListSubscriptions)
		adminGroup.GET("/audit", admin.ShowAuditLog)
	}
}
