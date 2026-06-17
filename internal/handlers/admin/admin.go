package admin

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"salesmee/internal/config"
	"salesmee/internal/db"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func pageSize() int {
	if n := config.C.TablePageSize; n > 0 {
		return n
	}
	return 10
}

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

	token, err := services.GenerateAdminToken(admin.ID, admin.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "admin_login.html", middleware.TemplateData(c, gin.H{
			"Title": "Admin Login - SalesMee",
			"Error": "Failed to generate session",
		}))
		return
	}

	services.SetSecureCookie(c, "admin_token", token, 86400, "/admin")
	c.Redirect(http.StatusFound, "/admin")
}

func AdminLogout(c *gin.Context) {
	services.ClearCookie(c, "admin_token", "/admin")
	c.Redirect(http.StatusFound, "/admin/login")
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("admin_token")
		if err == nil && cookie != "" {
			claims, err := services.ValidateToken(cookie)
			if err == nil && claims.Subject == "admin" {
				var admin models.Admin
				if db.DB.First(&admin, claims.UserID).Error == nil {
					c.Set("admin_id", admin.ID)
					c.Set("admin_email", admin.Email)
					c.Set("admin_name", admin.Name)
					c.Set("admin_role", admin.Role)
					// Set CSRF token so TemplateData() works
					middleware.GetCSRFToken(c)
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
	var totalBiz, activeBiz, suspendedBiz int64
	var totalClients, totalOrders, totalBookings int64
	var totalRevenue float64

	db.DB.Model(&models.Business{}).Count(&totalBiz)
	db.DB.Model(&models.Business{}).Where("is_public = ?", true).Count(&activeBiz)
	db.DB.Model(&models.Business{}).Where("is_public = ?", false).Count(&suspendedBiz)
	db.DB.Model(&models.Client{}).Count(&totalClients)
	db.DB.Model(&models.Order{}).Count(&totalOrders)
	db.DB.Model(&models.Booking{}).Count(&totalBookings)
	db.DB.Model(&models.Payment{}).Where("status = ?", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	var recentBiz []models.Business
	db.DB.Order("created_at desc").Limit(5).Find(&recentBiz)

	var recentLogs []models.AuditLog
	db.DB.Preload("Admin").Order("created_at desc").Limit(5).Find(&recentLogs)

	var planDist []struct {
		Name  string
		Code  string
		Count int64
	}
	db.DB.Table("businesses").
		Select("COALESCE(sp.name, 'No Plan') as name, COALESCE(sp.code, 'none') as code, COUNT(*) as count").
		Joins("LEFT JOIN business_subscriptions bs ON bs.business_id = businesses.id").
		Joins("LEFT JOIN subscription_plans sp ON sp.id = bs.plan_id").
		Group("sp.name, sp.code").
		Order("count DESC").
		Scan(&planDist)

	baseData := gin.H{
		"TotalBusinesses":    totalBiz,
		"ActiveBusinesses":   activeBiz,
		"SuspendedBiz":       suspendedBiz,
		"TotalClients":       totalClients,
		"TotalOrders":        totalOrders,
		"TotalBookings":      totalBookings,
		"TotalRevenue":       totalRevenue,
		"RecentBusinesses":   recentBiz,
		"RecentLogs":         recentLogs,
		"PlanDistribution":   planDist,
		"AdminName":          c.GetString("admin_name"),
		"AdminEmail":         c.GetString("admin_email"),
		"ActiveTab":          "dashboard",
		"Title":              "Dashboard - SalesMee Admin",
	}

	isHTMX := c.GetHeader("HX-Request") == "true"
	if isHTMX {
		c.HTML(http.StatusOK, "admin_dashboard_content.html", middleware.TemplateData(c, baseData))
		return
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", middleware.TemplateData(c, baseData))
}

func ListBusinesses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize()
	search := strings.TrimSpace(c.Query("search"))
	statusF := c.Query("status")
	planF := c.Query("plan")

	var total int64
	countQ := db.DB.Model(&models.Business{})
	if search != "" {
		like := "%" + search + "%"
		countQ = countQ.Where("name ILIKE ? OR email ILIKE ? OR slug ILIKE ?", like, like, like)
	}
	if statusF == "active" {
		countQ = countQ.Where("is_public = ?", true)
	} else if statusF == "suspended" {
		countQ = countQ.Where("is_public = ?", false)
	}
	if planF != "" {
		countQ = countQ.Where("subscription_plan_id = (SELECT id FROM subscription_plans WHERE code = ?)", planF)
	}
	countQ.Count(&total)

	var businesses []models.Business
	findQ := db.DB.Model(&models.Business{}).Preload("Subscription.Plan")
	if search != "" {
		like := "%" + search + "%"
		findQ = findQ.Where("name ILIKE ? OR email ILIKE ? OR slug ILIKE ?", like, like, like)
	}
	if statusF == "active" {
		findQ = findQ.Where("is_public = ?", true)
	} else if statusF == "suspended" {
		findQ = findQ.Where("is_public = ?", false)
	}
	if planF != "" {
		findQ = findQ.Where("subscription_plan_id = (SELECT id FROM subscription_plans WHERE code = ?)", planF)
	}
	findQ.Order("created_at desc").Limit(pageSize()).Offset(offset).Find(&businesses)

	clientCounts := make(map[uint]int64)
	orderCounts := make(map[uint]int64)
	bookingCounts := make(map[uint]int64)
	if len(businesses) > 0 {
		var ids []uint
		for _, b := range businesses {
			ids = append(ids, b.ID)
		}
		type cc struct {
			BusinessID uint
			Count      int64
		}
		var ccs []cc
		db.DB.Model(&models.Client{}).
			Select("business_id, COUNT(*) as count").
			Where("business_id IN ?", ids).
			Group("business_id").Scan(&ccs)
		for _, v := range ccs {
			clientCounts[v.BusinessID] = v.Count
		}
		var ocs []cc
		db.DB.Table("orders").
			Select("clients.business_id, COUNT(*) as count").
			Joins("JOIN clients ON clients.id = orders.client_id").
			Where("clients.business_id IN ?", ids).
			Group("clients.business_id").Scan(&ocs)
		for _, v := range ocs {
			orderCounts[v.BusinessID] = v.Count
		}
		var bcs []cc
		db.DB.Table("bookings").
			Select("clients.business_id, COUNT(*) as count").
			Joins("JOIN clients ON clients.id = bookings.client_id").
			Where("clients.business_id IN ?", ids).
			Group("clients.business_id").Scan(&bcs)
		for _, v := range bcs {
			bookingCounts[v.BusinessID] = v.Count
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize())))

	c.HTML(http.StatusOK, "admin_businesses.html", middleware.TemplateData(c, gin.H{
		"Title":         "Businesses - SalesMee Admin",
		"Businesses":    businesses,
		"ClientCounts":  clientCounts,
		"OrderCounts":   orderCounts,
		"BookingCounts": bookingCounts,
		"Page":         page,
		"TotalPages":   totalPages,
		"Total":        total,
		"Search":       search,
		"StatusFilter": statusF,
		"PlanFilter":   planF,
		"ActiveTab":    "businesses",
	}))
}

func GetBusinessDetail(c *gin.Context) {
	id := c.Param("id")
	var biz models.Business
	if err := db.DB.Preload("Subscription.Plan").First(&biz, id).Error; err != nil {
		c.String(http.StatusNotFound, "Business not found")
		return
	}
	var clientCount, orderCount, bookingCount, productCount, serviceCount, teamCount, locationCount int64
	db.DB.Model(&models.Client{}).Where("business_id = ?", biz.ID).Count(&clientCount)
	db.DB.Table("orders").Joins("JOIN clients ON clients.id = orders.client_id").
		Where("clients.business_id = ?", biz.ID).Count(&orderCount)
	db.DB.Table("bookings").Joins("JOIN clients ON clients.id = bookings.client_id").
		Where("clients.business_id = ?", biz.ID).Count(&bookingCount)
	db.DB.Model(&models.Product{}).Where("business_id = ?", biz.ID).Count(&productCount)
	db.DB.Model(&models.Service{}).Where("business_id = ?", biz.ID).Count(&serviceCount)
	db.DB.Model(&models.TeamMember{}).Where("business_id = ?", biz.ID).Count(&teamCount)
	db.DB.Model(&models.Location{}).Where("business_id = ?", biz.ID).Count(&locationCount)

	c.HTML(http.StatusOK, "admin_business_detail.html", middleware.TemplateData(c, gin.H{
		"Business":      biz,
		"ClientCount":   clientCount,
		"OrderCount":    orderCount,
		"BookingCount":  bookingCount,
		"ProductCount":  productCount,
		"ServiceCount":  serviceCount,
		"TeamCount":     teamCount,
		"LocationCount": locationCount,
	}))
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

func DeleteBusiness(c *gin.Context) {
	id := c.Param("id")
	db.DB.Model(&models.Business{}).Where("id = ?", id).Update("is_public", false)
	adminID, ip := adminCtx(c)
	db.DB.Create(&models.AuditLog{
		AdminID:    adminID,
		Action:     "delete",
		Resource:   "business",
		ResourceID: id,
		Details:    "Soft-deleted (suspended)",
		IP:         ip,
	})
	c.Redirect(http.StatusFound, "/admin/businesses")
}

func ListClients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize()
	search := strings.TrimSpace(c.Query("search"))
	statusF := c.Query("status")

	var total int64
	countQ := db.DB.Model(&models.Client{}).
		Joins("LEFT JOIN businesses ON businesses.id = clients.business_id")
	if search != "" {
		like := "%" + search + "%"
		countQ = countQ.Where("clients.name ILIKE ? OR clients.email ILIKE ? OR clients.phone ILIKE ? OR businesses.name ILIKE ?", like, like, like, like)
	}
	if statusF == "inactive" {
		countQ = countQ.Where("clients.status != ?", "active")
	} else if statusF != "" {
		countQ = countQ.Where("clients.status = ?", statusF)
	}
	countQ.Count(&total)

	type ClientRow struct {
		models.Client
		BusinessName string `gorm:"column:business_name"`
	}
	var clients []ClientRow
	findQ := db.DB.Table("clients").
		Select("clients.*, businesses.name as business_name").
		Joins("LEFT JOIN businesses ON businesses.id = clients.business_id")
	if search != "" {
		like := "%" + search + "%"
		findQ = findQ.Where("clients.name ILIKE ? OR clients.email ILIKE ? OR clients.phone ILIKE ? OR businesses.name ILIKE ?", like, like, like, like)
	}
	if statusF == "inactive" {
		findQ = findQ.Where("clients.status != ?", "active")
	} else if statusF != "" {
		findQ = findQ.Where("clients.status = ?", statusF)
	}
	findQ.Order("clients.created_at desc").Limit(pageSize()).Offset(offset).Scan(&clients)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize())))

	c.HTML(http.StatusOK, "admin_clients.html", middleware.TemplateData(c, gin.H{
		"Title":      "Clients - SalesMee Admin",
		"Clients":    clients,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
		"Search":     search,
		"StatusFilter": statusF,
		"ActiveTab":  "clients",
	}))
}

func DeleteClient(c *gin.Context) {
	id := c.Param("id")
	db.DB.Delete(&models.Client{}, id)
	adminID := c.GetUint("admin_id")
	ip := c.ClientIP()
	db.DB.Create(&models.AuditLog{
		AdminID:    adminID,
		Action:     "delete",
		Resource:   "client",
		ResourceID: id,
		IP:         ip,
	})
	c.Redirect(http.StatusFound, "/admin/clients")
}

func ListSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize()
	planF := c.Query("plan")
	statusF := c.Query("status")

	var total int64
	countSubQ := db.DB.Model(&models.BusinessSubscription{})
	if planF != "" {
		countSubQ = countSubQ.Where("plan_id = (SELECT id FROM subscription_plans WHERE code = ?)", planF)
	}
	if statusF != "" {
		countSubQ = countSubQ.Where("status = ?", statusF)
	}
	countSubQ.Count(&total)

	findSubQ := db.DB.Model(&models.BusinessSubscription{}).Preload("Business").Preload("Plan")
	if planF != "" {
		findSubQ = findSubQ.Where("plan_id = (SELECT id FROM subscription_plans WHERE code = ?)", planF)
	}
	if statusF != "" {
		findSubQ = findSubQ.Where("status = ?", statusF)
	}
	var subs []models.BusinessSubscription
	findSubQ.Order("created_at desc").Limit(pageSize()).Offset(offset).Find(&subs)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize())))

	c.HTML(http.StatusOK, "admin_subscriptions.html", middleware.TemplateData(c, gin.H{
		"Title":       "Subscriptions - SalesMee Admin",
		"Subscriptions": subs,
		"Page":        page,
		"TotalPages":  totalPages,
		"Total":       total,
		"PlanFilter":  planF,
		"StatusFilter": statusF,
		"ActiveTab":   "subscriptions",
	}))
}

func ShowAuditLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize()
	actionF := c.Query("action")
	if actionF == "delete_business" || actionF == "delete_client" {
		actionF = "delete"
	}
	resourceF := c.Query("resource")

	var total int64
	countAuditQ := db.DB.Model(&models.AuditLog{})
	if actionF != "" {
		countAuditQ = countAuditQ.Where("action = ?", actionF)
	}
	if resourceF != "" {
		countAuditQ = countAuditQ.Where("resource = ?", resourceF)
	}
	countAuditQ.Count(&total)

	findAuditQ := db.DB.Model(&models.AuditLog{}).Preload("Admin")
	if actionF != "" {
		findAuditQ = findAuditQ.Where("action = ?", actionF)
	}
	if resourceF != "" {
		findAuditQ = findAuditQ.Where("resource = ?", resourceF)
	}
	var logs []models.AuditLog
	findAuditQ.Order("created_at desc").Limit(pageSize()).Offset(offset).Find(&logs)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize())))

	c.HTML(http.StatusOK, "admin_audit.html", middleware.TemplateData(c, gin.H{
		"Title":      "Audit Log - SalesMee Admin",
		"Logs":       logs,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
		"ActionFilter":   actionF,
		"ResourceFilter": resourceF,
		"ActiveTab":  "audit",
	}))
}

func SeedAdmin() {
	var count int64
	db.DB.Model(&models.Admin{}).Count(&count)
	if count > 0 {
		return
	}

	email := config.C.AdminEmail
	password := config.C.AdminPassword
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

func adminCtx(c *gin.Context) (uint, string) {
	return c.GetUint("admin_id"), c.ClientIP()
}
