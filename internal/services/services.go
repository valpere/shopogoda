package services

import (
    "context"

    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog"
    "gorm.io/gorm"

    "github.com/valpere/shopogoda/internal/config"
)

type Services struct {
    User        *UserService
    Weather     *WeatherService
    Alert       *AlertService
    Subscription *SubscriptionService
    Notification *NotificationService
    Scheduler   *SchedulerService
}

func New(db *gorm.DB, redis *redis.Client, cfg *config.Config, logger *zerolog.Logger) *Services {
    userService := NewUserService(db, redis)
    weatherService := NewWeatherService(&cfg.Weather, redis)
    alertService := NewAlertService(db, redis)
    subscriptionService := NewSubscriptionService(db, redis)
    notificationService := NewNotificationService(&cfg.Integrations, logger)
    schedulerService := NewSchedulerService(db, redis, weatherService, alertService, notificationService, logger)

    return &Services{
        User:         userService,
        Weather:      weatherService,
        Alert:        alertService,
        Subscription: subscriptionService,
        Notification: notificationService,
        Scheduler:    schedulerService,
    }
}

func (s *Services) StartScheduler(ctx context.Context) {
    s.Scheduler.Start(ctx)
}

func (s *Services) Stop() {
    s.Scheduler.Stop()
}
