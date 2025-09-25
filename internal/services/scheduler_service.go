package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
)

const (
	// NotificationPlatformCount represents the number of notification platforms (Slack + Telegram)
	NotificationPlatformCount = 2
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
			// Track alert notification errors but don't fail processing
			var alertErrors []string

			// Send Slack alert
			if err := s.notification.SendSlackAlert(&alert, &user); err != nil {
				s.logger.Error().Err(err).Msg("Failed to send Slack alert")
				alertErrors = append(alertErrors, fmt.Sprintf("Slack: %v", err))
			}

			// Send Telegram alert
			if err := s.notification.SendTelegramAlert(&alert, &user); err != nil {
				s.logger.Error().Err(err).Msg("Failed to send Telegram alert")
				alertErrors = append(alertErrors, fmt.Sprintf("Telegram: %v", err))
			}

			// Log alert delivery status
			if len(alertErrors) == 0 {
				s.logger.Info().
					Str("alert_type", alert.AlertType.String()).
					Int64("user_id", user.ID).
					Msg("Alert notifications sent successfully to all platforms")
			} else if len(alertErrors) == NotificationPlatformCount {
				s.logger.Error().
					Strs("failed_platforms", alertErrors).
					Str("alert_type", alert.AlertType.String()).
					Int64("user_id", user.ID).
					Msg("Alert notification failed on all platforms")
			} else {
				s.logger.Warn().
					Strs("failed_platforms", alertErrors).
					Str("alert_type", alert.AlertType.String()).
					Int64("user_id", user.ID).
					Msg("Alert notification partially failed but at least one platform succeeded")
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
	s.logger.Debug().Time("utc_time", now).Msg("Processing scheduled notifications")

	// Get all active subscriptions with users who have locations set
	var subscriptions []models.Subscription
	if err := s.db.WithContext(ctx).
		Preload("User").
		Joins("JOIN users ON users.id = subscriptions.user_id").
		Where("subscriptions.is_active = ? AND users.location_name != '' AND users.location_name IS NOT NULL", true).
		Find(&subscriptions).Error; err != nil {
		s.logger.Error().Err(err).Msg("Failed to get active subscriptions")
		return
	}

	for _, subscription := range subscriptions {
		// Parse user's timezone
		userTimezone := subscription.User.Timezone
		if userTimezone == "" {
			userTimezone = "UTC"
		}

		location, err := time.LoadLocation(userTimezone)
		if err != nil {
			s.logger.Warn().Str("timezone", userTimezone).Err(err).Msg("Invalid timezone, using UTC")
			location = time.UTC
		}

		// Convert current time to user's timezone
		userTime := now.In(location)

		// Check if it's time to send the notification
		if s.shouldSendNotification(subscription, userTime) {
			s.logger.Info().
				Str("type", subscription.SubscriptionType.String()).
				Str("time", subscription.TimeOfDay).
				Str("user_timezone", userTimezone).
				Int64("user_id", subscription.UserID).
				Msg("Sending scheduled notification")

			if err := s.sendScheduledNotification(ctx, subscription); err != nil {
				s.logger.Error().Err(err).
					Int64("user_id", subscription.UserID).
					Str("type", subscription.SubscriptionType.String()).
					Msg("Failed to send scheduled notification")
			}
		}
	}
}

func (s *SchedulerService) shouldSendNotification(subscription models.Subscription, userTime time.Time) bool {
	// Parse the time of day (format: HH:MM)
	targetTime, err := time.Parse("15:04", subscription.TimeOfDay)
	if err != nil {
		s.logger.Warn().Str("time_of_day", subscription.TimeOfDay).Err(err).Msg("Invalid time format")
		return false
	}

	// Check if current time matches the target time (within 1 hour window to avoid missing)
	currentHour := userTime.Hour()
	currentMinute := userTime.Minute()
	targetHour := targetTime.Hour()
	targetMinute := targetTime.Minute()

	// Send if we're within a 5-minute window of the target time
	if currentHour == targetHour && math.Abs(float64(currentMinute-targetMinute)) <= 5 {
		switch subscription.SubscriptionType {
		case models.SubscriptionDaily:
			// Send daily notifications every day
			return true
		case models.SubscriptionWeekly:
			// Send weekly notifications only on Mondays
			return userTime.Weekday() == time.Monday
		case models.SubscriptionAlerts, models.SubscriptionExtreme:
			// Alert subscriptions are handled by checkAndProcessAlerts
			return false
		}
	}

	return false
}

func (s *SchedulerService) sendScheduledNotification(ctx context.Context, subscription models.Subscription) error {
	// Get weather for user's location
	weather, err := s.weather.GetCurrentWeatherByCoords(
		ctx,
		subscription.User.Latitude,
		subscription.User.Longitude,
	)
	if err != nil {
		return fmt.Errorf("failed to get weather for user %d: %w", subscription.UserID, err)
	}

	switch subscription.SubscriptionType {
	case models.SubscriptionDaily:
		// Send daily weather update
		users := []models.User{subscription.User}

		// Track notification errors but don't fail completely if one platform fails
		var notificationErrors []string

		// Send Slack notification
		if err := s.notification.SendSlackWeatherUpdate(weather, users); err != nil {
			s.logger.Error().Err(err).Msg("Failed to send Slack daily notification")
			notificationErrors = append(notificationErrors, fmt.Sprintf("Slack: %v", err))
		}

		// Send Telegram notification
		if err := s.notification.SendTelegramWeatherUpdate(weather, &subscription.User); err != nil {
			s.logger.Error().Err(err).Msg("Failed to send Telegram daily notification")
			notificationErrors = append(notificationErrors, fmt.Sprintf("Telegram: %v", err))
		}

		// Return error only if all platforms failed
		if len(notificationErrors) > 0 {
			if len(notificationErrors) == NotificationPlatformCount {
				return fmt.Errorf("all notification platforms failed: %v", strings.Join(notificationErrors, "; "))
			}
			// Log partial failure but don't return error
			s.logger.Warn().Strs("failed_platforms", notificationErrors).Msg("Some notification platforms failed but at least one succeeded")
		}

	case models.SubscriptionWeekly:
		// Send weekly summary (simplified for now)
		summary := fmt.Sprintf(`This week's weather overview:
🌡️ Current temperature: %.1f°C
💧 Humidity: %d%%
💨 Wind: %.1f km/h
🌿 Air quality: AQI %d

Stay weather-aware!`, weather.Temperature, weather.Humidity, weather.WindSpeed, weather.AQI)

		if err := s.notification.SendTelegramWeeklyUpdate(&subscription.User, summary); err != nil {
			return fmt.Errorf("failed to send weekly notification: %w", err)
		}
	}

	return nil
}

