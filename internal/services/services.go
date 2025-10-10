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

type Services struct {
	User         *UserService
	Weather      *WeatherService
	Alert        *AlertService
	Subscription *SubscriptionService
	Notification *NotificationService
	Scheduler    *SchedulerService
	Export       *ExportService
	Localization *LocalizationService
	Demo         *DemoService
	startTime    time.Time
}

func New(db *gorm.DB, redis *redis.Client, cfg *config.Config, logger *zerolog.Logger, metricsCollector *metrics.Metrics) *Services {
	startTime := time.Now()

	userService := NewUserService(db, redis, metricsCollector, startTime)
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

func (s *Services) StartScheduler(ctx context.Context) {
	s.Scheduler.Start(ctx)
}

func (s *Services) Stop() {
	s.Scheduler.Stop()
}
