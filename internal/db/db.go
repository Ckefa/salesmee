package db

import (
	"fmt"
	"log"

	"salesmee/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	var err error
	env := config.C.AppEnv
	dbPath := config.C.DBPath
	dbHost := config.C.DBHost

	// Use SQLite for development if PostgreSQL is not available
	if env == "dev" && dbPath != "" {
		log.Println("Using development sqlite Database")
		DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	} else if dbHost != "" {
		log.Println("Using PostgreSQL database")

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
			dbHost,
			config.C.DBUser,
			config.C.DBPassword,
			config.C.DBName,
			config.C.DBPort,
		)
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	} else {
		log.Fatal("Missing Database configuration: DB_PATH | DB_HOST")
	}

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")
}
