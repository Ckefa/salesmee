package admin

import (
	"net/http"
	"os"
	"time"

	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login.html", middleware.TemplateData(c, gin.H{
		"Title": "Admin Login - SalesMee",
	}))
}

func Login(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if email == "" || password == "" {
		c.HTML(http.StatusOK, "admin_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Admin Login - SalesMee",
			"Error": "Email and password are required",
		}))
		return
	}

	var admin models.Admin
	if err := db.DB.Where("email = ?", email).First(&admin).Error; err != nil {
		c.HTML(http.StatusOK, "admin_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Admin Login - SalesMee",
			"Error": "Invalid credentials",
		}))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		c.HTML(http.StatusOK, "admin_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Admin Login - SalesMee",
			"Error": "Invalid credentials",
		}))
		return
	}

	c.SetCookie("admin_token", admin.Email+":"+admin.Password[:20], 86400, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin")
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("admin_token")
		if err == nil && cookie != "" {
			var admin models.Admin
			if db.DB.Where("email = ?", cookie[:len(cookie)-21]).First(&admin).Error == nil {
				if cookie == admin.Email+":"+admin.Password[:20] {
					c.Set("admin_id", admin.ID)
					c.Set("admin_email", admin.Email)
					c.Set("admin_name", admin.Name)
					c.Set("admin_role", admin.Role)
					c.Next()
					return
				}
			}
		}
		c.Redirect(http.StatusFound, "/admin/login")
		c.Abort()
	}
}

func ShowDashboard(c *gin.Context) {
	var totalBusinesses int64
	var totalClients int64
	var totalOrders int64
	var totalSubscriptions int64

	db.DB.Model(&models.Business{}).Count(&totalBusinesses)
	db.DB.Model(&models.Client{}).Count(&totalClients)
	db.DB.Model(&models.Order{}).Count(&totalOrders)
	db.DB.Model(&models.BusinessSubscription{}).Count(&totalSubscriptions)

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"Title":            "Admin Dashboard - SalesMee",
		"TotalBusinesses":  totalBusinesses,
		"TotalClients":     totalClients,
		"TotalOrders":      totalOrders,
		"TotalSubscriptions": totalSubscriptions,
	})
}

func ListBusinesses(c *gin.Context) {
	var businesses []models.Business
	db.DB.Preload("Subscription.Plan").Order("created_at desc").Limit(50).Find(&businesses)

	c.HTML(http.StatusOK, "admin_businesses.html", gin.H{
		"Title":      "Manage Businesses - SalesMee",
		"Businesses": businesses,
	})
}

func SuspendBusiness(c *gin.Context) {
	id := c.Param("id")
	db.DB.Model(&models.Business{}).Where("id = ?", id).Update("is_public", false)

	adminID := c.GetUint("admin_id")
	ip := c.ClientIP()
	db.DB.Create(&models.AuditLog{
		AdminID:    adminID,
		Action:     "suspend",
		Resource:   "business",
		ResourceID: id,
		IP:         ip,
	})

	c.Redirect(http.StatusFound, "/admin/businesses")
}

func ActivateBusiness(c *gin.Context) {
	id := c.Param("id")
	db.DB.Model(&models.Business{}).Where("id = ?", id).Update("is_public", true)

	adminID := c.GetUint("admin_id")
	ip := c.ClientIP()
	db.DB.Create(&models.AuditLog{
		AdminID:    adminID,
		Action:     "activate",
		Resource:   "business",
		ResourceID: id,
		IP:         ip,
	})

	c.Redirect(http.StatusFound, "/admin/businesses")
}

func ListSubscriptions(c *gin.Context) {
	var subs []models.BusinessSubscription
	db.DB.Preload("Business").Preload("Plan").Order("created_at desc").Limit(50).Find(&subs)

	c.HTML(http.StatusOK, "admin_subscriptions.html", gin.H{
		"Title":         "Subscriptions - SalesMee",
		"Subscriptions": subs,
	})
}

func ShowAuditLog(c *gin.Context) {
	var logs []models.AuditLog
	db.DB.Preload("Admin").Order("created_at desc").Limit(100).Find(&logs)

	c.HTML(http.StatusOK, "admin_audit.html", gin.H{
		"Title": "Audit Log - SalesMee",
		"Logs":  logs,
	})
}

func SeedAdmin() {
	var count int64
	db.DB.Model(&models.Admin{}).Count(&count)
	if count > 0 {
		return
	}

	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	db.DB.Create(&models.Admin{
		Email:    email,
		Password: string(hashed),
		Name:     "Super Admin",
		Role:     "super_admin",
	})

	// Log seed
	var admin models.Admin
	db.DB.Where("email = ?", email).First(&admin)
	db.DB.Create(&models.AuditLog{
		AdminID:    admin.ID,
		Action:     "seed",
		Resource:   "admin",
		ResourceID: "1",
		Details:    "Initial admin account seeded",
		CreatedAt:  time.Now(),
	})
}
