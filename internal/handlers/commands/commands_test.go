package commands

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/tests/helpers"
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

func TestIsValidTimezone(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		timezone string
		expected bool
	}{
		{"Valid UTC", "UTC", true},
		{"Valid America/New_York", "America/New_York", true},
		{"Valid Europe/London", "Europe/London", true},
		{"Valid Europe/Kyiv", "Europe/Kyiv", true},
		{"Valid Asia/Tokyo", "Asia/Tokyo", true},
		{"Valid America/Los_Angeles", "America/Los_Angeles", true},
		{"Invalid timezone", "Invalid/Timezone", false},
		{"Empty string defaults to Local", "", true}, // Empty string is valid - defaults to Local
		{"Random string", "RandomString123", false},
		{"Partial timezone", "America", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isValidTimezone(tt.timezone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAQIDescription(t *testing.T) {
	logger := helpers.NewSilentTestLogger()

	// Create minimal services with localization
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		aqi      int
		expected string
	}{
		{"Good air quality - 0", 0, "aqi_good"},
		{"Good air quality - 25", 25, "aqi_good"},
		{"Good air quality - 50", 50, "aqi_good"},
		{"Moderate - 51", 51, "aqi_moderate"},
		{"Moderate - 75", 75, "aqi_moderate"},
		{"Moderate - 100", 100, "aqi_moderate"},
		{"Unhealthy for sensitive - 101", 101, "aqi_unhealthy_sensitive"},
		{"Unhealthy for sensitive - 150", 150, "aqi_unhealthy_sensitive"},
		{"Unhealthy - 151", 151, "aqi_unhealthy"},
		{"Unhealthy - 200", 200, "aqi_unhealthy"},
		{"Very unhealthy - 201", 201, "aqi_very_unhealthy"},
		{"Very unhealthy - 300", 300, "aqi_very_unhealthy"},
		{"Hazardous - 301", 301, "aqi_hazardous"},
		{"Hazardous - 500", 500, "aqi_hazardous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getAQIDescription(tt.aqi, "en")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHealthRecommendation(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		aqi      int
		expected string
	}{
		{"Good health - 0", 0, "health_good"},
		{"Good health - 50", 50, "health_good"},
		{"Moderate health - 51", 51, "health_moderate"},
		{"Moderate health - 100", 100, "health_moderate"},
		{"Unhealthy for sensitive - 101", 101, "health_unhealthy_sensitive"},
		{"Unhealthy for sensitive - 150", 150, "health_unhealthy_sensitive"},
		{"Unhealthy - 151", 151, "health_unhealthy"},
		{"Unhealthy - 200", 200, "health_unhealthy"},
		{"Very unhealthy - 201", 201, "health_very_unhealthy"},
		{"Very unhealthy - 300", 300, "health_very_unhealthy"},
		{"Hazardous - 301", 301, "health_hazardous"},
		{"Hazardous - 500", 500, "health_hazardous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getHealthRecommendation(tt.aqi, "en")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocalizedUnitsText(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		units    string
		expected string
	}{
		{"Metric units", "metric", "units_metric"},
		{"Imperial units", "imperial", "units_imperial"},
		{"Unknown units", "kelvin", "kelvin"}, // Returns input as-is
		{"Empty units", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getLocalizedUnitsText(context.Background(), "en", tt.units)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocalizedRoleName(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		role     models.UserRole
		expected string
	}{
		{"Admin role", models.RoleAdmin, "role_admin"},
		{"Moderator role", models.RoleModerator, "role_moderator"},
		{"User role", models.RoleUser, "role_user"},
		{"Unknown role defaults to user", models.UserRole(999), "role_user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getLocalizedRoleName(context.Background(), "en", tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocalizedStatusText(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		isActive bool
		expected string
	}{
		{"Active status", true, "status_active"},
		{"Inactive status", false, "status_inactive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getLocalizedStatusText(context.Background(), "en", tt.isActive)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSubscriptionTypeText(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		subType  models.SubscriptionType
		expected string
	}{
		{"Daily subscription", models.SubscriptionDaily, "subscription_type_daily"},
		{"Weekly subscription", models.SubscriptionWeekly, "subscription_type_weekly"},
		{"Alerts subscription", models.SubscriptionAlerts, "subscription_type_alerts"},
		{"Extreme subscription", models.SubscriptionExtreme, "subscription_type_extreme"},
		{"Unknown subscription type", models.SubscriptionType(999), "subscription_type_unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getSubscriptionTypeText(tt.subType, "en")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFrequencyText(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		freq     models.Frequency
		expected string
	}{
		{"Hourly frequency", models.FrequencyHourly, "frequency_hourly"},
		{"Every 3 hours frequency", models.FrequencyEvery3Hours, "frequency_every_3_hours"},
		{"Every 6 hours frequency", models.FrequencyEvery6Hours, "frequency_every_6_hours"},
		{"Daily frequency", models.FrequencyDaily, "frequency_daily"},
		{"Weekly frequency", models.FrequencyWeekly, "frequency_weekly"},
		{"Unknown frequency", models.Frequency(999), "frequency_unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getFrequencyText(tt.freq, "en")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseLocationFromArgs(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"No args", []string{"/weather"}, ""},
		{"Single word location", []string{"/weather", "London"}, "London"},
		{"Multi-word location", []string{"/weather", "New", "York"}, "New York"},
		{"Location with comma", []string{"/weather", "Paris,", "France"}, "Paris, France"},
		{"Location with extra spaces", []string{"/weather", " ", "Berlin", " "}, "Berlin"},
		{"Empty args after command", []string{"/weather", ""}, ""},
		{"Three word location", []string{"/weather", "Los", "Angeles", "USA"}, "Los Angeles USA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
				Args: tt.args,
			})

			result := handler.parseLocationFromArgs(mockCtx.Context)
			assert.Equal(t, tt.expected, result)
		})
	}
}
