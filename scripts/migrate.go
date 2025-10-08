package main

import (
	"log"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/database"
	"github.com/valpere/shopogoda/internal/models"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting ShoPogoda database migrations...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database with PreferSimpleProtocol for local PostgreSQL
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying SQL DB to check connection
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection successful")

	// Run migrations
	log.Println("Running AutoMigrate...")
	if err := models.Migrate(db); err != nil {
		log.Printf("AutoMigrate failed: %v", err)
		log.Println("Attempting manual table creation...")

		// Fallback: Create tables manually if AutoMigrate fails
		if err := manualMigrate(db); err != nil {
			log.Fatalf("Manual migration also failed: %v", err)
		}
		log.Println("Manual migration completed successfully!")
	} else {
		log.Println("AutoMigrate completed successfully!")
	}

	log.Println("ShoPogoda migrations completed successfully!")
}

func manualMigrate(db *gorm.DB) error {
	// Drop existing tables first to ensure clean schema
	dropSqls := []string{
		`DROP TABLE IF EXISTS user_sessions CASCADE`,
		`DROP TABLE IF EXISTS environmental_alerts CASCADE`,
		`DROP TABLE IF EXISTS alert_configs CASCADE`,
		`DROP TABLE IF EXISTS subscriptions CASCADE`,
		`DROP TABLE IF EXISTS weather_data CASCADE`,
		`DROP TABLE IF EXISTS users CASCADE`,
	}

	for _, sql := range dropSqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	// Create tables manually using raw SQL
	sqls := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY,
			username VARCHAR(255),
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			language VARCHAR(10) DEFAULT 'en',
			units VARCHAR(20) DEFAULT 'metric',
			timezone VARCHAR(100) DEFAULT 'UTC',
			role INTEGER DEFAULT 1,
			is_active BOOLEAN DEFAULT true,
			location_name VARCHAR(255),
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			country VARCHAR(100),
			city VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,

		`CREATE TABLE IF NOT EXISTS weather_data (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id BIGINT,
			temperature DOUBLE PRECISION,
			humidity INTEGER,
			pressure DOUBLE PRECISION,
			wind_speed DOUBLE PRECISION,
			wind_degree INTEGER,
			visibility DOUBLE PRECISION,
			uv_index DOUBLE PRECISION,
			description VARCHAR(255),
			icon VARCHAR(50),
			aqi INTEGER,
			co DOUBLE PRECISION,
			no2 DOUBLE PRECISION,
			o3 DOUBLE PRECISION,
			pm25 DOUBLE PRECISION,
			pm10 DOUBLE PRECISION,
			timestamp TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_weather_data_user_id ON weather_data(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_weather_data_timestamp ON weather_data(timestamp)`,

		`CREATE TABLE IF NOT EXISTS subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id BIGINT,
			subscription_type VARCHAR(50),
			frequency VARCHAR(50),
			time_of_day VARCHAR(10),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)`,

		`CREATE TABLE IF NOT EXISTS alert_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id BIGINT,
			alert_type VARCHAR(50),
			threshold_value DOUBLE PRECISION,
			comparison_operator VARCHAR(10),
			is_active BOOLEAN DEFAULT true,
			cooldown_minutes INTEGER DEFAULT 60,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_configs_user_id ON alert_configs(user_id)`,

		`CREATE TABLE IF NOT EXISTS environmental_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id BIGINT,
			alert_config_id UUID,
			alert_type VARCHAR(50),
			severity VARCHAR(20),
			triggered_value DOUBLE PRECISION,
			threshold_value DOUBLE PRECISION,
			message TEXT,
			is_resolved BOOLEAN DEFAULT false,
			triggered_at TIMESTAMP,
			resolved_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_environmental_alerts_user_id ON environmental_alerts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_environmental_alerts_triggered_at ON environmental_alerts(triggered_at)`,

		`CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id BIGINT,
			session_data TEXT,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at)`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}
