package main

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"salesmee/internal/data"
	"salesmee/internal/db"
	"salesmee/internal/handlers/admin"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/routes"
	"salesmee/internal/services/notifier"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()

	gin.SetMode(os.Getenv("GIN_MODE"))

	db.Connect()

	log.Println("🔄 Starting database auto-migration...")
	db.DB.AutoMigrate(
		&models.Business{},
		&models.Client{},
		&models.Conversation{},
		&models.Message{},
		&models.Action{},
		&models.ClientAuth{},
		&models.ConversationProgress{},
		&models.Product{},
		&models.Service{},
		&models.Order{},
		&models.OrderItem{},
		&models.Booking{},
		&models.BookingItem{},
		&models.Payment{},
		&models.InventoryLog{},
		&models.ProductImage{},
		&models.CustomerInsight{},
		&models.SubscriptionPlan{},
		&models.BusinessSubscription{},
		&models.PasswordResetToken{},
		&models.Admin{},
		&models.AuditLog{},
		&models.BusinessNotifPrefs{},
		&models.NotificationLog{},
		&models.InAppNotification{},
	)
	log.Println("✅ Database auto-migration completed successfully")

	// Schema sync: add missing columns for models modified after initial migration
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS customer_id INTEGER NOT NULL DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS total_orders INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS pending_orders INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS completed_orders INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS total_bookings INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS pending_bookings INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS completed_bookings INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS total_messages INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS total_spent DOUBLE PRECISION DEFAULT 0")
	db.DB.Exec("ALTER TABLE customer_insights ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ")
	log.Println("✅ Schema sync completed")

	// Data migration: copy old first_name/last_name to name/username for existing records
	log.Println("🔄 Running data migration for business fields...")
	db.DB.Exec("UPDATE businesses SET name = first_name WHERE (name IS NULL OR name = '') AND (first_name IS NOT NULL AND first_name != '')")
	db.DB.Exec("UPDATE businesses SET username = last_name WHERE (username IS NULL OR username = '') AND (last_name IS NOT NULL AND last_name != '')")
	log.Println("✅ Data migration completed")

	// Seed default subscription plans
	seedSubscriptionPlans(db.DB)

	// Seed admin account
	admin.SeedAdmin()

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	// Rate limiting (before CSRF so blocked requests don't need tokens)
	r.Use(middleware.RateLimitGlobal())
	// Apply CSRF protection globally
	r.Use(middleware.CSRFMiddleware())
	// Collect template files
	var files []string

	err := filepath.Walk("web/templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"hasPrefix": strings.HasPrefix,
		"fbLogin": func() bool {
			return os.Getenv("FB_LOGIN") == "TRUE"
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				if i+1 < len(values) {
					key := values[i].(string)
					dict[key] = values[i+1]
				}
			}
			return dict, nil
		},
		"title": strings.Title,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"formatTime": func(t time.Time) string {
			return t.Format("3:04 PM")
		},
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b float64) float64 { return a * b },
		"div": func(a, b float64) float64 { return a / b },
		"float": func(i int) float64 { return float64(i) },
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"currencySymbol": func(code string) string {
			for _, c := range data.Currencies {
				if c.Code == code {
					return c.Symbol
				}
			}
			return code
		},
		"seq": func(start, end int) []int {
			s := []int{}
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"percent": func(current, total int) int {
			if total <= 0 {
				return 0
			}
			return int(float64(current) / float64(total) * 100)
		},
	}).ParseFiles(files...))
	r.SetHTMLTemplate(tmpl)

	// Get Static Path
	r.Static("/static", "./web/static")
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./web/static/images/salesmee.ico")
	})

	routes.Setup(r)
	routes.SetupBusinessRoutes(r)
	routes.SetupClientRoutes(r)
	routes.SetupAdminRoutes(r)

	// Start background notification scheduler
	notifier.StartNotificationScheduler(db.DB)

	log.Println("🚀 Running on :" + os.Getenv("APP_PORT"))
	r.Run(":" + os.Getenv("APP_PORT"))
}

func seedSubscriptionPlans(d *gorm.DB) {
	var count int64
	d.Model(&models.SubscriptionPlan{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("🌱 Seeding subscription plans...")

	plans := []models.SubscriptionPlan{
		{
			Code:                 "silver",
			Name:                 "Silver",
			Description:          "For small businesses getting started",
			PriceMonthly:         0,
			PriceYearly:          0,
			Currency:             "usd",
			MaxClients:           10,
			MaxProducts:          10,
			MaxServices:          10,
			MaxConversations:     10,
			HasAnalytics:         false,
			HasMediaSharing:      false,
			HasPrioritySupport:   false,
			HasOrdersAndBookings: false,
			SortOrder:            0,
			IsActive:             true,
		},
		{
			Code:                 "gold",
			Name:                 "Gold",
			Description:          "For growing businesses",
			PriceMonthly:         8,
			PriceYearly:          6.40,
			Currency:             "usd",
			MaxClients:           50,
			MaxProducts:          200,
			MaxServices:          200,
			MaxConversations:     50,
			HasAnalytics:         true,
			HasMediaSharing:      false,
			HasPrioritySupport:   false,
			HasOrdersAndBookings: true,
			SortOrder:            1,
			IsActive:             true,
		},
		{
			Code:                 "diamond",
			Name:                 "Diamond",
			Description:          "For businesses that need it all",
			PriceMonthly:         15,
			PriceYearly:          12,
			Currency:             "usd",
			MaxClients:           0,
			MaxProducts:          0,
			MaxServices:          0,
			MaxConversations:     0,
			HasAnalytics:         true,
			HasMediaSharing:      true,
			HasPrioritySupport:   true,
			HasOrdersAndBookings: true,
			SortOrder:            2,
			IsActive:             true,
		},
	}

	for _, plan := range plans {
		d.FirstOrCreate(&plan, models.SubscriptionPlan{Code: plan.Code})
	}

	log.Println("✅ Subscription plans seeded successfully")
}
