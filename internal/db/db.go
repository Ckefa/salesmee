package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"salesmee/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var err error
	env := config.C.AppEnv
	dbPath := config.C.DBPath
	dbHost := config.C.DBHost

	gormConfig := &gorm.Config{
		PrepareStmt:           true,
		SkipDefaultTransaction: true,
		QueryFields:           true,
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: 200 * time.Millisecond,
				LogLevel:      logger.Warn,
				Colorful:      false,
			},
		),
	}

	// Use SQLite for development if PostgreSQL is not available
	if env == "dev" && dbPath != "" {
		log.Println("Using development sqlite Database")
		DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	} else if dbHost != "" {
		log.Println("Using PostgreSQL database")

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
			dbHost,
			config.C.DBUser,
			config.C.DBPassword,
			config.C.DBName,
			config.C.DBPort,
		)
		DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	} else {
		log.Fatal("Missing Database configuration: DB_PATH | DB_HOST")
	}

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(config.C.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(config.C.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.C.DBMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(config.C.DBMaxIdleTime) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("Database connected successfully (pool: %d/%d, lifetime: %dm, idle: %dm)",
		config.C.DBMaxOpenConns, config.C.DBMaxIdleConns,
		config.C.DBMaxLifetime, config.C.DBMaxIdleTime)
}
