// Package services provides the business logic layer for the ShoPogoda bot.
// It implements service-oriented architecture with dependency injection and
// separation of concerns. Each service encapsulates specific business domain logic.
package services

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/pkg/metrics"
)

// Services is the central container for all business logic services.
// It provides a single point of access to all service layer functionality.
//
// Architecture:
//   - Uses dependency injection for all services
//   - Services are initialized once during application startup
//   - Thread-safe and designed for concurrent use
//
// Usage:
//
//	svcs := services.New(db, redis, cfg, logger, metrics)
//	defer svcs.Stop()
//
//	user, err := svcs.User.GetUser(ctx, userID)
//	weather, err := svcs.Weather.GetCurrentWeather(ctx, lat, lon)
type Services struct {
	User         *UserService         // User management, locations, timezones, statistics
	Weather      *WeatherService      // Weather data retrieval and geocoding
	Alert        *AlertService        // Custom alert configurations and monitoring
	Subscription *SubscriptionService // Notification subscription management
	Notification *NotificationService // Dual-platform notification delivery (Telegram + Slack)
	Scheduler    *SchedulerService    // Background job scheduling for alerts and notifications
	Export       *ExportService       // Data export for GDPR compliance and backups
	Localization *LocalizationService // Multi-language translation support
	Demo         *DemoService         // Demo data management for testing
	startTime    time.Time            // Application start time for uptime calculation
}

// New creates a new Services container with all dependencies initialized.
// This is the primary constructor for the service layer.
//
// Parameters:
//   - db: GORM database connection (PostgreSQL)
//   - redis: Redis client for caching
//   - cfg: Application configuration
//   - logger: Structured logger (zerolog)
//   - metricsCollector: Prometheus metrics collector
//
// Returns:
//   - *Services: Initialized services container
//
// Example:
//
//	db := database.Connect(cfg.Database)
//	redis := database.ConnectRedis(cfg.Redis)
//	logger := zerolog.New(os.Stdout)
//	metrics := metrics.New()
//
//	svcs := services.New(db, redis, cfg, logger, metrics)
//	defer svcs.Stop()
func New(db *gorm.DB, redis *redis.Client, cfg *config.Config, logger *zerolog.Logger, metricsCollector *metrics.Metrics) *Services {
	startTime := time.Now()

	userService := NewUserService(db, redis, metricsCollector, logger, startTime)
	weatherService := NewWeatherService(&cfg.Weather, redis, logger)
	alertService := NewAlertService(db, redis)
	subscriptionService := NewSubscriptionService(db, redis)
	notificationService := NewNotificationService(&cfg.Integrations, logger)
	schedulerService := NewSchedulerService(db, redis, weatherService, alertService, notificationService, logger)
	localizationService := NewLocalizationService(logger)
	exportService := NewExportService(db, logger, localizationService)
	demoService := NewDemoService(db, logger)

	return &Services{
		User:         userService,
		Weather:      weatherService,
		Alert:        alertService,
		Subscription: subscriptionService,
		Notification: notificationService,
		Scheduler:    schedulerService,
		Export:       exportService,
		Localization: localizationService,
		Demo:         demoService,
		startTime:    startTime,
	}
}

// StartScheduler starts the background scheduler service for processing
// alerts and scheduled notifications.
//
// The scheduler runs two concurrent jobs:
//  1. Alert processing - Every 10 minutes
//  2. Scheduled notifications - Every hour (timezone-aware)
//
// Parameters:
//   - ctx: Context for cancellation and lifecycle management
//
// Example:
//
//	ctx := context.Background()
//	svcs.StartScheduler(ctx)
func (s *Services) StartScheduler(ctx context.Context) {
	s.Scheduler.Start(ctx)
}

// Stop gracefully stops all background services, particularly the scheduler.
// Should be called during application shutdown to ensure clean termination.
//
// Example:
//
//	defer svcs.Stop()
func (s *Services) Stop() {
	s.Scheduler.Stop()
}
