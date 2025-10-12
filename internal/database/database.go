package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Disable prepared statement cache for Supabase pooler
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Optimized connection pool settings for Supabase/Railway
	sqlDB.SetMaxOpenConns(25)                    // Maximum concurrent connections
	sqlDB.SetMaxIdleConns(5)                     // Keep only 20% idle (reduce resource usage)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)    // Recycle connections every 5 minutes
	sqlDB.SetConnMaxIdleTime(1 * time.Minute)    // Close idle connections after 1 minute

	return db, nil
}

func ConnectRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	// Enable TLS for Upstash Redis (required for cloud connections)
	if cfg.Host != "localhost" && cfg.Host != "127.0.0.1" {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rdb, nil
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.WeatherData{},
		&models.Subscription{},
		&models.AlertConfig{},
		&models.EnvironmentalAlert{},
		&models.UserSession{},
	)
}
