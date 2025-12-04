package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewSchedulerService(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	// Create mock services
	weatherService := &WeatherService{}
	alertService := &AlertService{}
	notificationService := &NotificationService{}

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		weatherService,
		alertService,
		notificationService,
		logger,
	)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.redis)
	assert.NotNil(t, service.weather)
	assert.NotNil(t, service.alert)
	assert.NotNil(t, service.notification)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.stopChan)
}

func TestSchedulerService_ShouldSendNotification(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		&WeatherService{},
		&AlertService{},
		&NotificationService{},
		logger,
	)

	// Test time in user's timezone: 08:00
	userTime := time.Date(2025, 1, 15, 8, 2, 0, 0, time.UTC) // Wednesday

	t.Run("daily subscription at correct time", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "08:00",
		}
		assert.True(t, service.shouldSendNotification(subscription, userTime))
	})

	t.Run("daily subscription within 5 minute window", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "08:00",
		}
		// 4 minutes after target time - should still work
		timeWithin := time.Date(2025, 1, 15, 8, 4, 0, 0, time.UTC)
		assert.True(t, service.shouldSendNotification(subscription, timeWithin))
	})

	t.Run("daily subscription outside 5 minute window", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "08:00",
		}
		// 6 minutes after target time - too late
		timeOutside := time.Date(2025, 1, 15, 8, 6, 0, 0, time.UTC)
		assert.False(t, service.shouldSendNotification(subscription, timeOutside))
	})

	t.Run("daily subscription before target time", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "08:00",
		}
		// 2 minutes before target time - too early
		timeBefore := time.Date(2025, 1, 15, 7, 58, 0, 0, time.UTC)
		assert.False(t, service.shouldSendNotification(subscription, timeBefore))
	})

	t.Run("weekly subscription on Monday", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionWeekly,
			TimeOfDay:        "08:00",
		}
		// January 13, 2025 is a Monday
		mondayTime := time.Date(2025, 1, 13, 8, 0, 0, 0, time.UTC)
		assert.True(t, service.shouldSendNotification(subscription, mondayTime))
	})

	t.Run("weekly subscription not on Monday", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionWeekly,
			TimeOfDay:        "08:00",
		}
		// January 15, 2025 is a Wednesday
		wednesdayTime := time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC)
		assert.False(t, service.shouldSendNotification(subscription, wednesdayTime))
	})

	t.Run("alert subscription should not trigger scheduled check", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionAlerts,
			TimeOfDay:        "08:00",
		}
		assert.False(t, service.shouldSendNotification(subscription, userTime))
	})

	t.Run("extreme subscription should not trigger scheduled check", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionExtreme,
			TimeOfDay:        "08:00",
		}
		assert.False(t, service.shouldSendNotification(subscription, userTime))
	})

	t.Run("invalid time format", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "invalid",
		}
		assert.False(t, service.shouldSendNotification(subscription, userTime))
	})

	t.Run("wrong hour", func(t *testing.T) {
		subscription := models.Subscription{
			SubscriptionType: models.SubscriptionDaily,
			TimeOfDay:        "09:00", // User time is 08:00
		}
		assert.False(t, service.shouldSendNotification(subscription, userTime))
	})
}

func TestSchedulerService_Stop(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		&WeatherService{},
		&AlertService{},
		&NotificationService{},
		logger,
	)

	// Stop should close the stop channel
	service.Stop()

	// Verify channel is closed by trying to receive from it
	select {
	case <-service.stopChan:
		// Channel is closed as expected
	default:
		t.Fatal("stopChan was not closed")
	}
}

func TestSchedulerService_NotificationPlatformCount(t *testing.T) {
	// Verify the constant is correct (Slack + Telegram = 2)
	assert.Equal(t, 2, NotificationPlatformCount)
}

func TestSchedulerService_CheckAndProcessAlerts(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	// Create mock services
	weatherService := &WeatherService{}
	alertService := &AlertService{}
	notificationService := &NotificationService{}

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		weatherService,
		alertService,
		notificationService,
		logger,
	)

	// Create test context
	ctx := context.Background()

	t.Run("no active users with locations", func(t *testing.T) {
		// Clear any existing users
		mockDB.DB.Exec("DELETE FROM users")

		// Call checkAndProcessAlerts - should not error
		service.checkAndProcessAlerts(ctx)
	})

	t.Run("user without location is skipped", func(t *testing.T) {
		// Clear users
		mockDB.DB.Exec("DELETE FROM users")

		// Create user without location
		user := models.User{
			ID:           12345,
			Username:     "testuser",
			IsActive:     true,
			LocationName: "", // No location
		}
		mockDB.DB.Create(&user)

		// Call checkAndProcessAlerts - should skip user
		service.checkAndProcessAlerts(ctx)
	})
}

func TestSchedulerService_ProcessDailyNotifications(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	// Create mock services
	weatherService := &WeatherService{}
	alertService := &AlertService{}
	notificationService := &NotificationService{}

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		weatherService,
		alertService,
		notificationService,
		logger,
	)

	ctx := context.Background()

	t.Run("no active subscriptions", func(t *testing.T) {
		// Clear data
		mockDB.DB.Exec("DELETE FROM subscriptions")
		mockDB.DB.Exec("DELETE FROM users")

		// Call processDailyNotifications - should not error
		service.processDailyNotifications(ctx)
	})

	t.Run("subscription for user without location is skipped", func(t *testing.T) {
		// Clear data
		mockDB.DB.Exec("DELETE FROM subscriptions")
		mockDB.DB.Exec("DELETE FROM users")

		// Create user without location
		user := models.User{
			ID:           12345,
			Username:     "testuser",
			IsActive:     true,
			LocationName: "", // No location
			Timezone:     "UTC",
		}
		mockDB.DB.Create(&user)

		// Create subscription
		subscription := models.Subscription{
			UserID:           user.ID,
			SubscriptionType: models.SubscriptionDaily,
			Frequency:        models.FrequencyDaily,
			TimeOfDay:        "08:00",
			IsActive:         true,
		}
		mockDB.DB.Create(&subscription)

		// Call processDailyNotifications - should skip subscription
		service.processDailyNotifications(ctx)
	})

	t.Run("processes active subscriptions with locations", func(t *testing.T) {
		// Clear data
		mockDB.DB.Exec("DELETE FROM subscriptions")
		mockDB.DB.Exec("DELETE FROM users")

		// Create user with location
		user := models.User{
			ID:           12345,
			Username:     "testuser",
			IsActive:     true,
			LocationName: "Test City",
			Latitude:     50.45,
			Longitude:    30.52,
			Timezone:     "UTC",
		}
		mockDB.DB.Create(&user)

		// Create subscription with a time that won't match current time
		subscription := models.Subscription{
			UserID:           user.ID,
			SubscriptionType: models.SubscriptionDaily,
			Frequency:        models.FrequencyDaily,
			TimeOfDay:        "23:59", // Set to a time that won't match
			IsActive:         true,
		}
		mockDB.DB.Create(&subscription)

		// Call processDailyNotifications - should process but not send (wrong time)
		service.processDailyNotifications(ctx)
	})
}

func TestSchedulerService_SendScheduledNotification(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	logger := helpers.NewSilentTestLogger()

	// Create config for weather service
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-api-key",
		UserAgent:         "test-user-agent",
	}

	// Create properly initialized weather service
	weatherService := NewWeatherService(cfg, mockRedis.Client, logger)
	alertService := &AlertService{}
	notificationService := &NotificationService{}

	service := NewSchedulerService(
		mockDB.DB,
		mockRedis.Client,
		weatherService,
		alertService,
		notificationService,
		logger,
	)

	ctx := context.Background()

	t.Run("daily subscription without weather data fails", func(t *testing.T) {
		user := models.User{
			ID:           12345,
			Username:     "testuser",
			LocationName: "Test City",
			Latitude:     50.45,
			Longitude:    30.52,
		}

		subscription := models.Subscription{
			UserID:           user.ID,
			SubscriptionType: models.SubscriptionDaily,
			User:             user,
		}

		// Call should fail because weather API will fail
		err := service.sendScheduledNotification(ctx, subscription)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get weather")
	})

	t.Run("weekly subscription without weather data fails", func(t *testing.T) {
		user := models.User{
			ID:           12345,
			Username:     "testuser",
			LocationName: "Test City",
			Latitude:     50.45,
			Longitude:    30.52,
		}

		subscription := models.Subscription{
			UserID:           user.ID,
			SubscriptionType: models.SubscriptionWeekly,
			User:             user,
		}

		// Call should fail because weather API will fail
		err := service.sendScheduledNotification(ctx, subscription)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get weather")
	})
}
