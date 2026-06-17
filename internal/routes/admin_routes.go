package routes

import (
	"salesmee/internal/handlers/admin"
	"salesmee/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAdminRoutes(r *gin.Engine) {
	r.GET("/admin/login", admin.ShowLogin)
	r.POST("/admin/login", admin.Login)

	adminGroup := r.Group("/admin")
	adminGroup.Use(admin.AdminMiddleware(), middleware.CSRFMiddleware())
	{
		adminGroup.GET("", admin.ShowDashboard)
		adminGroup.GET("/logout", admin.AdminLogout)
		adminGroup.GET("/businesses", admin.ListBusinesses)
		adminGroup.GET("/businesses/:id", admin.GetBusinessDetail)
		adminGroup.POST("/businesses/:id/suspend", admin.SuspendBusiness)
		adminGroup.POST("/businesses/:id/activate", admin.ActivateBusiness)
		adminGroup.POST("/businesses/:id/delete", admin.DeleteBusiness)
		adminGroup.GET("/clients", admin.ListClients)
		adminGroup.POST("/clients/:id/delete", admin.DeleteClient)
		adminGroup.GET("/subscriptions", admin.ListSubscriptions)
		adminGroup.GET("/audit", admin.ShowAuditLog)
	}
}
