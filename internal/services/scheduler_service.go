package services

import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/rs/zerolog"
    "gorm.io/gorm"

    "github.com/valpere/shopogoda/internal/models"
)

type SchedulerService struct {
    db           *gorm.DB
    redis        *redis.Client
    weather      *WeatherService
    alert        *AlertService
    notification *NotificationService
    logger       *zerolog.Logger
    stopChan     chan struct{}
}

func NewSchedulerService(
    db *gorm.DB,
    redis *redis.Client,
    weather *WeatherService,
    alert *AlertService,
    notification *NotificationService,
    logger *zerolog.Logger,
) *SchedulerService {
    return &SchedulerService{
        db:           db,
        redis:        redis,
        weather:      weather,
        alert:        alert,
        notification: notification,
        logger:       logger,
        stopChan:     make(chan struct{}),
    }
}

func (s *SchedulerService) Start(ctx context.Context) {
    s.logger.Info().Msg("Starting scheduler service")

    // Check alerts every 10 minutes
    alertTicker := time.NewTicker(10 * time.Minute)
    defer alertTicker.Stop()

    // Send daily weather updates at 8 AM
    dailyTicker := time.NewTicker(time.Hour)
    defer dailyTicker.Stop()

    for {
        select {
        case <-ctx.Done():
            s.logger.Info().Msg("Scheduler context cancelled")
            return
        case <-s.stopChan:
            s.logger.Info().Msg("Scheduler stop signal received")
            return
        case <-alertTicker.C:
            s.checkAndProcessAlerts(ctx)
        case <-dailyTicker.C:
            s.processDailyNotifications(ctx)
        }
    }
}

func (s *SchedulerService) Stop() {
    close(s.stopChan)
}

func (s *SchedulerService) checkAndProcessAlerts(ctx context.Context) {
    s.logger.Debug().Msg("Checking weather alerts")

    // Get all active users with locations and alerts configured
    var users []models.User
    if err := s.db.WithContext(ctx).
        Where("is_active = ? AND location_name != '' AND location_name IS NOT NULL", true).
        Find(&users).Error; err != nil {
        s.logger.Error().Err(err).Msg("Failed to get active users with locations")
        return
    }

    for _, user := range users {
        // Get current weather for user's location
        weather, err := s.weather.GetCurrentWeatherByCoords(ctx, user.Latitude, user.Longitude)
        if err != nil {
            s.logger.Error().Err(err).
                Str("location", user.LocationName).
                Int64("user_id", user.ID).
                Msg("Failed to get weather data")
            continue
        }

        // Check for triggered alerts
        alerts, err := s.alert.CheckAlerts(ctx, weather.ToModelWeatherData(), user.ID)
        if err != nil {
            s.logger.Error().Err(err).
                Str("location", user.LocationName).
                Int64("user_id", user.ID).
                Msg("Failed to check alerts")
            continue
        }

        // Send notifications for triggered alerts
        for _, alert := range alerts {
            if err := s.notification.SendSlackAlert(&alert, &user); err != nil {
                s.logger.Error().Err(err).Msg("Failed to send Slack alert")
            }
        }

        if len(alerts) > 0 {
            s.logger.Info().
                Int("count", len(alerts)).
                Str("location", user.LocationName).
                Int64("user_id", user.ID).
                Msg("Processed weather alerts")
        }
    }
}

func (s *SchedulerService) processDailyNotifications(ctx context.Context) {
    now := time.Now().UTC()

    // Only send daily notifications at 8 AM
    if now.Hour() != 8 || now.Minute() != 0 {
        return
    }

    s.logger.Info().Msg("Processing daily weather notifications")

    // Get users with daily subscriptions who have locations set
    var subscriptions []models.Subscription
    if err := s.db.WithContext(ctx).
        Preload("User").
        Joins("JOIN users ON users.id = subscriptions.user_id").
        Where("subscriptions.subscription_type = ? AND subscriptions.is_active = ? AND users.location_name != '' AND users.location_name IS NOT NULL", models.SubscriptionDaily, true).
        Find(&subscriptions).Error; err != nil {
        s.logger.Error().Err(err).Msg("Failed to get daily subscriptions")
        return
    }

    for _, subscription := range subscriptions {
        // Get weather for user's location
        weather, err := s.weather.GetCurrentWeatherByCoords(
            ctx,
            subscription.User.Latitude,
            subscription.User.Longitude,
        )
        if err != nil {
            s.logger.Error().Err(err).
                Str("location", subscription.User.LocationName).
                Int64("user_id", subscription.UserID).
                Msg("Failed to get weather for daily notification")
            continue
        }

        // Send notification
        users := []models.User{subscription.User}
        if err := s.notification.SendSlackWeatherUpdate(weather, users); err != nil {
            s.logger.Error().Err(err).
                Int64("user_id", subscription.UserID).
                Msg("Failed to send daily weather notification")
        }
    }
}
