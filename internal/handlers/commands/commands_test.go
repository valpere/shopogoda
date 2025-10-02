package commands

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/models"
)

func TestNew(t *testing.T) {
	logger := zerolog.Nop()

	handler := New(nil, &logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.logger)
}

func TestGetAlertTypeText(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name      string
		alertType models.AlertType
		expected  string
	}{
		{"Temperature", models.AlertTemperature, "Temperature"},
		{"Humidity", models.AlertHumidity, "Humidity"},
		{"Pressure", models.AlertPressure, "Pressure"},
		{"Wind Speed", models.AlertWindSpeed, "Wind Speed"},
		{"UV Index", models.AlertUVIndex, "UV Index"},
		{"Air Quality", models.AlertAirQuality, "Air Quality"},
		{"Rain", models.AlertRain, "Rain"},
		{"Snow", models.AlertSnow, "Snow"},
		{"Storm", models.AlertStorm, "Storm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getAlertTypeText(tt.alertType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAlertTypeText_Unknown(t *testing.T) {
	handler := &CommandHandler{}

	// Test with unknown alert type (use invalid numeric value)
	unknownType := models.AlertType(999)
	result := handler.getAlertTypeText(unknownType)
	assert.Equal(t, "Unknown", result)
}

func TestGetAlertStatusText(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		isActive bool
		expected string
	}{
		{"Active", true, "Active"},
		{"Inactive", false, "Inactive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getAlertStatusText(tt.isActive)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNotificationEmoji(t *testing.T) {
	tests := []struct {
		name     string
		subType  models.SubscriptionType
		expected string
	}{
		{"Daily", models.SubscriptionDaily, "☀️"},
		{"Weekly", models.SubscriptionWeekly, "📅"},
		{"Alerts", models.SubscriptionAlerts, "⚡"},
		{"Extreme", models.SubscriptionExtreme, "🌪️"},
		{"Unknown", models.SubscriptionType(999), "🔔"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNotificationEmoji(tt.subType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNotificationFrequency(t *testing.T) {
	tests := []struct {
		name     string
		notType  string
		expected string
	}{
		{"Daily", "daily", "daily"},
		{"Weekly", "weekly", "weekly"},
		{"Alerts", "alerts", "alerts"},
		{"Extreme", "extreme", "alerts"},
		{"Unknown", "unknown_type", "daily"},
		{"Empty", "", "daily"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNotificationFrequency(tt.notType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
