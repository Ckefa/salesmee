package config

import (
	"log"
	"os"
	"strconv"
)

var C Config

type Config struct {
	// App
	AppPort  string
	AppURL   string
	AppEnv   string
	GinMode  string
	FBLogin  bool

	// DB
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBPath          string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBMaxLifetime   int // minutes
	DBMaxIdleTime   int // minutes

	// JWT
	JWTSecret   string
	CSRFSecret  string

	// Auth (Admin)
	AdminEmail    string
	AdminPassword string

	// Auth (Social)
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	GoogleClientRedirect string
	FBAppID              string
	FBSecret             string
	FBRedirectURL        string
	FBClientRedirectURL  string

	// Resend (Email)
	ResendEnabled  bool
	ResendAPIKey   string
	ResendFromEmail string
	AppDomain      string

	// AI
	GroqAPIKey string

	// WebSocket
	AllowedOrigins string

	// Subscription
	StripeEnabled       bool
	StripeSecretKey     string
	StripePublishableKey string
	StripeWebhookSecret  string
	PaddleEnabled       bool
	PaddleClientToken   string
	PaddleAPIKey        string
	PaddleEnvironment   string
	PaddleWebhookSecret  string
	PolarEnabled        bool
	PolarAccessToken    string
	PolarEnvironment    string
	PolarWebhookSecret  string

	// Support
	SupportEmail string

	// Notifications
	NotifScheduler bool

	// Table
	TablePageSize int

	// WebSocket Auth
	BizID string
}

func IsDev() bool {
	return C.AppEnv == "dev"
}

func Load() {
	C = Config{
		// App
		AppPort:  getEnv("APP_PORT", "8080"),
		AppURL:   getEnv("APP_URL", ""),
		AppEnv:   getEnv("ENV", "production"),
		GinMode:  getEnv("GIN_MODE", "release"),
		FBLogin:  os.Getenv("FB_LOGIN") == "TRUE",

		// DB
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", ""),
		DBPath:          getEnv("DB_PATH", ""),
		DBMaxOpenConns:  getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:  getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLifetime:   getEnvInt("DB_CONN_MAX_LIFETIME", 30),
		DBMaxIdleTime:   getEnvInt("DB_CONN_MAX_IDLE_TIME", 5),

		// JWT / CSRF
		JWTSecret:  requireEnv("JWT_SECRET"),
		CSRFSecret: getEnv("CSRF_SECRET", ""),

		// Admin
		AdminEmail:    requireEnv("ADMIN_EMAIL"),
		AdminPassword: requireEnv("ADMIN_PASSWORD"),

		// Social Auth
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", ""),
		GoogleClientRedirect: getEnv("GOOGLE_CLIENT_REDIRECT_URL", ""),
		FBAppID:              getEnv("FB_APP_ID", ""),
		FBSecret:             getEnv("FB_SECRET", ""),
		FBRedirectURL:        getEnv("FB_REDIRECT_URL", ""),
		FBClientRedirectURL:  getEnv("FB_CLIENT_REDIRECT_URL", ""),

		// Resend
		ResendEnabled:   os.Getenv("RESEND") == "true",
		ResendAPIKey:    getEnv("RESEND_API_KEY", ""),
		ResendFromEmail: getEnv("RESEND_FROM_EMAIL", ""),
		AppDomain:       getEnv("APP_DOMAIN", ""),

		// AI
		GroqAPIKey: getEnv("GROQ_API_KEY", ""),

		// WebSocket
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),

		// Subscription
		StripeEnabled:        os.Getenv("STRIPE_ENABLED") == "true",
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		PaddleEnabled:        os.Getenv("PADDLE_ENABLED") == "true",
		PaddleClientToken:    getEnv("PADDLE_CLIENT_TOKEN", ""),
		PaddleAPIKey:         getEnv("PADDLE_API_KEY", ""),
		PaddleEnvironment:    getEnv("PADDLE_ENVIRONMENT", "sandbox"),
		PaddleWebhookSecret:  getEnv("PADDLE_WEBHOOK_SECRET", ""),
		PolarEnabled:         os.Getenv("POLAR_ENABLED") == "true",
		PolarAccessToken:     getEnv("POLAR_ACCESS_TOKEN", ""),
		PolarEnvironment:     getEnv("POLAR_ENVIRONMENT", "sandbox"),
		PolarWebhookSecret:   getEnv("POLAR_WEBHOOK_SECRET", ""),

		// Support
		SupportEmail: getEnv("SUPPORT_EMAIL", "support@salesmee.com"),

		// Notifications
		NotifScheduler: os.Getenv("NOTIF_SCHEDULER") == "true",

		// Table
		TablePageSize: getEnvInt("TABLE_PAGE_SIZE", 10),

		// WS Auth
		BizID: getEnv("BIZ_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("FATAL: required environment variable %s is not set", key)
	}
	return v
}
