package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"salesmee/internal/data"
	"salesmee/internal/db"
	"salesmee/internal/handlers/admin"
	"salesmee/internal/handlers/business"
	"salesmee/internal/handlers/client"
	"salesmee/internal/middleware"
	"salesmee/internal/models"
	"salesmee/internal/routes"
	"salesmee/internal/services/notifier"
	"salesmee/internal/ws"
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
	migrateModels := []interface{}{
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
		&models.Review{},
		&models.Location{},
		&models.TeamMember{},
	}
	for _, m := range migrateModels {
		if err := db.DB.AutoMigrate(m); err != nil {
			log.Printf("⚠️ Migration warning for %T: %v", m, err)
		}
	}
	log.Println("✅ Database auto-migration completed")

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
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS time_zone VARCHAR(50) DEFAULT 'UTC'")
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS buffer_time INTEGER DEFAULT 0")
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS max_bookings_per_slot INTEGER DEFAULT 1")
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS is_accepting_bookings BOOLEAN DEFAULT true")
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS business_hours JSONB DEFAULT '{}'")
	db.DB.Exec("ALTER TABLE businesses ADD COLUMN IF NOT EXISTS special_hours JSONB DEFAULT '[]'")
	db.DB.Exec("UPDATE businesses SET business_hours = ?::jsonb WHERE business_hours IS NULL OR business_hours = '{}'::jsonb", `{"monday":[{"open":"08:00","close":"19:00"}],"tuesday":[{"open":"08:00","close":"19:00"}],"wednesday":[{"open":"08:00","close":"19:00"}],"thursday":[{"open":"08:00","close":"19:00"}],"friday":[{"open":"08:00","close":"19:00"}]}`)
	db.DB.Exec("ALTER TABLE products ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL")
	db.DB.Exec(`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uni_products_sku') AND NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'uni_products_sku') THEN ALTER TABLE products ADD CONSTRAINT uni_products_sku UNIQUE (sku); END IF; END $$;`)
	db.DB.Exec("ALTER TABLE services ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL")
	db.DB.Exec("ALTER TABLE orders ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL")
	db.DB.Exec("ALTER TABLE bookings ADD COLUMN IF NOT EXISTS location_id INTEGER REFERENCES locations(id) ON DELETE SET NULL")
	db.DB.Exec("ALTER TABLE order_items ADD COLUMN IF NOT EXISTS staff_id INTEGER REFERENCES team_members(id) ON DELETE SET NULL")
	db.DB.Exec("ALTER TABLE order_items ADD COLUMN IF NOT EXISTS commission_type VARCHAR(20) DEFAULT 'percentage'")
	db.DB.Exec("ALTER TABLE order_items ADD COLUMN IF NOT EXISTS commission_value DOUBLE PRECISION DEFAULT 0")
	db.DB.Exec("ALTER TABLE order_items ADD COLUMN IF NOT EXISTS commission_earned DOUBLE PRECISION DEFAULT 0")
	db.DB.Exec("ALTER TABLE booking_items ADD COLUMN IF NOT EXISTS staff_id INTEGER REFERENCES team_members(id) ON DELETE SET NULL")
	db.DB.Exec("ALTER TABLE booking_items ADD COLUMN IF NOT EXISTS commission_type VARCHAR(20) DEFAULT 'percentage'")
	db.DB.Exec("ALTER TABLE booking_items ADD COLUMN IF NOT EXISTS commission_value DOUBLE PRECISION DEFAULT 0")
	db.DB.Exec("ALTER TABLE booking_items ADD COLUMN IF NOT EXISTS commission_earned DOUBLE PRECISION DEFAULT 0")
	db.DB.Exec(`CREATE TABLE IF NOT EXISTS team_member_locations (
		team_member_id INTEGER NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
		location_id INTEGER NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
		PRIMARY KEY (team_member_id, location_id)
	)`)
	// Add indexes for frequently queried fields
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_orders_biz_status_date ON orders (business_id, status, created_at)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_biz_status_date ON bookings (business_id, status, created_at)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_orders_client_status ON orders (client_id, status)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_bookings_client_status ON bookings (client_id, status)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments (created_at)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_conv_created ON messages (conversation_id, created_at)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_products_biz_active ON products (business_id, is_active)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_services_biz_active ON services (business_id, is_active)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_clients_biz_name ON clients (business_id, name)")
	db.DB.Exec("CREATE INDEX IF NOT EXISTS idx_clients_conversation ON clients (conversation_id)")
	log.Println("✅ Indexes created")

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
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
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
		"printf": func(format string, args ...interface{}) string {
			return fmt.Sprintf(format, args...)
		},
		"substr": func(s string, start, length int) string {
			runes := []rune(s)
			if start >= len(runes) {
				return ""
			}
			end := start + length
			if end > len(runes) {
				end = len(runes)
			}
			return string(runes[start:end])
		},
	}).ParseFiles(files...))
	r.SetHTMLTemplate(tmpl)
	business.SetTemplate(tmpl)
	client.SetTemplate(tmpl)

	// Get Static Path
	r.Static("/static", "./web/static")
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./web/static/images/salesmee.ico")
	})
	r.GET("/service-worker.js", func(c *gin.Context) {
		c.File("./web/static/service-worker.js")
	})

	hub := ws.NewHub()
	routes.SetWSHub(hub)
	go hub.Run()

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
