package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewNotificationService(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{
		SlackWebhookURL: "https://hooks.slack.com/test",
	}

	service := NewNotificationService(cfg, logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.logger)
	assert.Equal(t, cfg, service.config)
	assert.Nil(t, service.bot) // Bot is initially nil until set
}

func TestNotificationService_SetBot(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{}
	service := NewNotificationService(cfg, logger)

	// Create a mock bot (just for testing SetBot functionality)
	bot := &gotgbot.Bot{}
	service.SetBot(bot)

	assert.NotNil(t, service.bot)
	assert.Equal(t, bot, service.bot)
}

func TestNotificationService_GetTelegramChatID(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{}
	service := NewNotificationService(cfg, logger)

	user := &models.User{
		ID:        123456,
		FirstName: "Test",
	}

	chatID := service.getTelegramChatID(user)
	assert.Equal(t, user.ID, chatID)
}

func TestNotificationService_GetSeverityColor(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name     string
		severity models.Severity
		expected string
	}{
		{"low severity", models.SeverityLow, "good"},
		{"medium severity", models.SeverityMedium, "warning"},
		{"high severity", models.SeverityHigh, "danger"},
		{"critical severity", models.SeverityCritical, "#ff0000"},
		{"unknown severity", models.Severity(999), "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := service.getSeverityColor(tt.severity)
			assert.Equal(t, tt.expected, color)
		})
	}
}

func TestNotificationService_GetSeverityText(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name     string
		severity models.Severity
		expected string
	}{
		{"low severity", models.SeverityLow, "Low"},
		{"medium severity", models.SeverityMedium, "Medium"},
		{"high severity", models.SeverityHigh, "High"},
		{"critical severity", models.SeverityCritical, "Critical"},
		{"unknown severity", models.Severity(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := service.getSeverityText(tt.severity)
			assert.Equal(t, tt.expected, text)
		})
	}
}

func TestNotificationService_GetSeverityEmoji(t *testing.T) {
	service := &NotificationService{}

	tests := []struct {
		name     string
		severity models.Severity
		expected string
	}{
		{"low severity", models.SeverityLow, "⚠️"},
		{"medium severity", models.SeverityMedium, "🔶"},
		{"high severity", models.SeverityHigh, "🚨"},
		{"critical severity", models.SeverityCritical, "🆘"},
		{"unknown severity", models.Severity(999), "ℹ️"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emoji := service.getSeverityEmoji(tt.severity)
			assert.Equal(t, tt.expected, emoji)
		})
	}
}

func TestNotificationService_SendSlackAlert(t *testing.T) {
	logger := helpers.NewSilentTestLogger()

	t.Run("successful slack alert", func(t *testing.T) {
		// Create mock Slack server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var message SlackMessage
			err := json.NewDecoder(r.Body).Decode(&message)
			require.NoError(t, err)

			assert.Equal(t, "🚨 Weather Alert", message.Text)
			assert.Len(t, message.Attachments, 1)
			assert.Equal(t, "danger", message.Attachments[0].Color)
			assert.Equal(t, "High Temperature Alert", message.Attachments[0].Title)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: server.URL,
		}
		service := NewNotificationService(cfg, logger)

		alert := &models.EnvironmentalAlert{
			ID:          uuid.New(),
			AlertType:   models.AlertTemperature,
			Title:       "High Temperature Alert",
			Description: "Temperature exceeds threshold",
			Severity:    models.SeverityHigh,
			Value:       35.0,
			Threshold:   30.0,
		}

		user := &models.User{
			ID:           123,
			FirstName:    "John",
			LocationName: "London",
		}

		err := service.SendSlackAlert(alert, user)
		assert.NoError(t, err)
	})

	t.Run("slack not configured", func(t *testing.T) {
		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: "",
		}
		service := NewNotificationService(cfg, logger)

		alert := &models.EnvironmentalAlert{
			ID:       uuid.New(),
			Severity: models.SeverityMedium,
		}
		user := helpers.MockUser(123)

		err := service.SendSlackAlert(alert, user)
		assert.NoError(t, err) // Should return nil if not configured
	})

	t.Run("slack webhook error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: server.URL,
		}
		service := NewNotificationService(cfg, logger)

		alert := &models.EnvironmentalAlert{
			ID:       uuid.New(),
			Severity: models.SeverityHigh,
		}
		user := helpers.MockUser(123)

		err := service.SendSlackAlert(alert, user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slack webhook returned status 500")
	})

	t.Run("user without location", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var message SlackMessage
			_ = json.NewDecoder(r.Body).Decode(&message)

			// Check that default location is used
			locationField := message.Attachments[0].Fields[0]
			assert.Equal(t, "Unknown Location", locationField.Value)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: server.URL,
		}
		service := NewNotificationService(cfg, logger)

		alert := &models.EnvironmentalAlert{
			ID:       uuid.New(),
			Severity: models.SeverityMedium,
		}
		user := &models.User{
			ID:           123,
			FirstName:    "John",
			LocationName: "", // No location set
		}

		err := service.SendSlackAlert(alert, user)
		assert.NoError(t, err)
	})
}

func TestNotificationService_SendSlackWeatherUpdate(t *testing.T) {
	logger := helpers.NewSilentTestLogger()

	t.Run("successful slack weather update", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var message SlackMessage
			err := json.NewDecoder(r.Body).Decode(&message)
			require.NoError(t, err)

			assert.Contains(t, message.Text, "Daily Weather Update")
			assert.Contains(t, message.Text, "London")
			assert.Len(t, message.Attachments, 1)
			assert.Equal(t, "good", message.Attachments[0].Color)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: server.URL,
		}
		service := NewNotificationService(cfg, logger)

		weather := &WeatherData{
			LocationName:  "London",
			Temperature:   20.5,
			Humidity:      65,
			WindSpeed:     10.2,
			WindDirection: 180,
			AQI:           2,
			Timestamp:     time.Now(),
		}

		users := []models.User{
			{ID: 123, FirstName: "User1"},
			{ID: 456, FirstName: "User2"},
		}

		err := service.SendSlackWeatherUpdate(weather, users)
		assert.NoError(t, err)
	})

	t.Run("slack not configured", func(t *testing.T) {
		cfg := &config.IntegrationsConfig{
			SlackWebhookURL: "",
		}
		service := NewNotificationService(cfg, logger)

		weather := &WeatherData{LocationName: "London"}
		users := []models.User{}

		err := service.SendSlackWeatherUpdate(weather, users)
		assert.NoError(t, err)
	})
}

func TestNotificationService_SendTelegramAlert(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{}

	t.Run("bot not configured", func(t *testing.T) {
		service := NewNotificationService(cfg, logger)
		// Don't set bot

		alert := &models.EnvironmentalAlert{
			ID:       uuid.New(),
			Severity: models.SeverityHigh,
		}
		user := helpers.MockUser(123)

		err := service.SendTelegramAlert(alert, user)
		assert.NoError(t, err) // Should return nil if bot not configured
	})

	t.Run("user without location", func(t *testing.T) {
		service := NewNotificationService(cfg, logger)
		// Note: We can't easily test actual message sending without mocking gotgbot.Bot
		// which would require an interface. For now, we just test that the method
		// handles nil bot correctly.

		user := &models.User{
			ID:           123,
			FirstName:    "John",
			LocationName: "", // No location
		}

		alert := &models.EnvironmentalAlert{
			ID:          uuid.New(),
			AlertType:   models.AlertTemperature,
			Title:       "Test Alert",
			Description: "Test Description",
			Severity:    models.SeverityMedium,
			Value:       25.0,
			Threshold:   20.0,
		}

		err := service.SendTelegramAlert(alert, user)
		assert.NoError(t, err) // Should be nil since bot is not configured
	})
}

func TestNotificationService_SendTelegramWeatherUpdate(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{}

	t.Run("bot not configured", func(t *testing.T) {
		service := NewNotificationService(cfg, logger)

		weather := &WeatherData{
			LocationName:  "London",
			Temperature:   20.5,
			Humidity:      65,
			WindSpeed:     10.2,
			WindDirection: 180,
			AQI:           2,
			Visibility:    10.0,
			Timestamp:     time.Now(),
		}
		user := helpers.MockUser(123)

		err := service.SendTelegramWeatherUpdate(weather, user)
		assert.NoError(t, err)
	})
}

func TestNotificationService_SendTelegramWeeklyUpdate(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	cfg := &config.IntegrationsConfig{}

	t.Run("bot not configured", func(t *testing.T) {
		service := NewNotificationService(cfg, logger)

		user := &models.User{
			ID:           123,
			FirstName:    "John",
			LocationName: "London",
		}

		summary := "Week summary: Mostly sunny with occasional clouds"

		err := service.SendTelegramWeeklyUpdate(user, summary)
		assert.NoError(t, err)
	})
}