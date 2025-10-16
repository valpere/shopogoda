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
			result := handler.getAQIDescription(tt.aqi, "en-US")
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
			result := handler.getHealthRecommendation(tt.aqi, "en-US")
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
			result := handler.getLocalizedUnitsText(context.Background(), "en-US", tt.units)
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
			result := handler.getLocalizedRoleName(context.Background(), "en-US", tt.role)
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
			result := handler.getLocalizedStatusText(context.Background(), "en-US", tt.isActive)
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
			result := handler.getSubscriptionTypeText(tt.subType, "en-US")
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
			result := handler.getFrequencyText(tt.freq, "en-US")
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

func TestGetThresholdOptions(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name         string
		alertType    models.AlertType
		currentValue float64
		validate     func(*testing.T, []float64)
	}{
		{
			name:         "Temperature alert - returns fixed range -20 to 40",
			alertType:    models.AlertTemperature,
			currentValue: 25.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Should generate values from -20 to 40 in 5° steps around current
				assert.Contains(t, options, 25.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, -20.0)
					assert.LessOrEqual(t, opt, 40.0)
				}
			},
		},
		{
			name:         "Humidity alert - returns fixed range 20 to 90",
			alertType:    models.AlertHumidity,
			currentValue: 60.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 60.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 20.0)
					assert.LessOrEqual(t, opt, 90.0)
				}
			},
		},
		{
			name:         "Pressure alert - returns fixed range 960 to 1040",
			alertType:    models.AlertPressure,
			currentValue: 1013.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 1013.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 960.0)
					assert.LessOrEqual(t, opt, 1040.0)
				}
			},
		},
		{
			name:         "Wind Speed alert - returns fixed range 5 to 50",
			alertType:    models.AlertWindSpeed,
			currentValue: 25.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 25.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 5.0)
					assert.LessOrEqual(t, opt, 50.0)
				}
			},
		},
		{
			name:         "UV Index alert - returns fixed range 1 to 11",
			alertType:    models.AlertUVIndex,
			currentValue: 6.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 6.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 1.0)
					assert.LessOrEqual(t, opt, 11.0)
				}
			},
		},
		{
			name:         "Air Quality alert - returns fixed range 50 to 300",
			alertType:    models.AlertAirQuality,
			currentValue: 150.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 150.0) // current value
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 50.0)
					assert.LessOrEqual(t, opt, 300.0)
				}
			},
		},
		{
			name:         "Default alert type - uses defaultMinFactor (0.8) and defaultMaxFactor (1.2)",
			alertType:    models.AlertType(999), // Unknown type
			currentValue: 100.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Should generate ±20% range: 80-120
				assert.Contains(t, options, 100.0) // current value
				for _, opt := range options {
					// min = 100 * 0.8 = 80, max = 100 * 1.2 = 120
					assert.GreaterOrEqual(t, opt, 80.0)
					assert.LessOrEqual(t, opt, 120.0)
				}
			},
		},
		{
			name:         "Rain alert type",
			alertType:    models.AlertRain,
			currentValue: 50.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Uses default case: ±20%
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 40.0) // 50 * 0.8
					assert.LessOrEqual(t, opt, 60.0)    // 50 * 1.2
				}
			},
		},
		{
			name:         "Snow alert type",
			alertType:    models.AlertSnow,
			currentValue: 30.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Uses default case: ±20%
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 24.0) // 30 * 0.8
					assert.LessOrEqual(t, opt, 36.0)    // 30 * 1.2
				}
			},
		},
		{
			name:         "Storm alert type",
			alertType:    models.AlertStorm,
			currentValue: 75.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Uses default case: ±20%
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 60.0) // 75 * 0.8
					assert.LessOrEqual(t, opt, 90.0)    // 75 * 1.2
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := handler.getThresholdOptions(tt.alertType, tt.currentValue)
			tt.validate(t, options)
		})
	}
}

func TestGenerateRangeOptions(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		min      float64
		max      float64
		step     float64
		current  float64
		validate func(*testing.T, []float64)
	}{
		{
			name:    "Basic range centered on current - uses thresholdOptionsRange (3)",
			min:     0.0,
			max:     100.0,
			step:    10.0,
			current: 50.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// Should have thresholdOptionsRange (3) values below, current, and 3 above
				assert.Contains(t, options, 50.0) // current
				assert.Contains(t, options, 40.0) // -1 step
				assert.Contains(t, options, 30.0) // -2 steps
				assert.Contains(t, options, 20.0) // -3 steps
				assert.Contains(t, options, 60.0) // +1 step
				assert.Contains(t, options, 70.0) // +2 steps
				assert.Contains(t, options, 80.0) // +3 steps
				assert.Equal(t, 7, len(options))  // 3 below + current + 3 above
			},
		},
		{
			name:    "Range near minimum boundary",
			min:     10.0,
			max:     100.0,
			step:    5.0,
			current: 15.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 15.0) // current
				// Should include 10.0 (min) but not go below
				assert.Contains(t, options, 10.0)
				// All values should be within bounds
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 10.0)
					assert.LessOrEqual(t, opt, 100.0)
				}
			},
		},
		{
			name:    "Range near maximum boundary",
			min:     0.0,
			max:     100.0,
			step:    10.0,
			current: 95.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// With current=95, step=10, we'd want: 65, 75, 85, 95, 105, 115, 125
				// But only 65-100 are valid (within min-max bounds)
				// This gives us only 4 values (65, 75, 85, 95), which is < minThresholdOptions (5)
				// So fallback triggers: generates from min in steps until maxThresholdOptions
				assert.GreaterOrEqual(t, len(options), minThresholdOptions)
				assert.LessOrEqual(t, len(options), maxThresholdOptions)
				// All values should be within bounds
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 0.0)
					assert.LessOrEqual(t, opt, 100.0)
				}
			},
		},
		{
			name:    "Small range triggers fallback - uses minThresholdOptions (5) and maxThresholdOptions (7)",
			min:     10.0,
			max:     20.0,
			step:    10.0,
			current: 15.0,
			validate: func(t *testing.T, options []float64) {
				// With step=10, only 10, 20 would be in range around 15
				// Should fallback to generating minThresholdOptions (5) from min
				assert.NotEmpty(t, options)
				assert.GreaterOrEqual(t, len(options), 2)
				assert.LessOrEqual(t, len(options), maxThresholdOptions)
				// Should include both bounds
				assert.Contains(t, options, 10.0)
				assert.Contains(t, options, 20.0)
			},
		},
		{
			name:    "Very small range with 1.0 step",
			min:     1.0,
			max:     5.0,
			step:    1.0,
			current: 3.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				// With step=1, should have 1,2,3,4,5 (5 values)
				assert.GreaterOrEqual(t, len(options), minThresholdOptions)
				assert.Contains(t, options, 3.0)
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 1.0)
					assert.LessOrEqual(t, opt, 5.0)
				}
			},
		},
		{
			name:    "Large step size",
			min:     0.0,
			max:     200.0,
			step:    50.0,
			current: 100.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 100.0) // current
				// With step=50: 0, 50, 100, 150, 200 possible
				// Around 100: 50(-1), 100(0), 150(+1) at minimum
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 0.0)
					assert.LessOrEqual(t, opt, 200.0)
				}
			},
		},
		{
			name:    "Fractional steps",
			min:     0.0,
			max:     10.0,
			step:    0.5,
			current: 5.0,
			validate: func(t *testing.T, options []float64) {
				assert.NotEmpty(t, options)
				assert.Contains(t, options, 5.0) // current
				// Should have 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5
				assert.GreaterOrEqual(t, len(options), 5)
				for _, opt := range options {
					assert.GreaterOrEqual(t, opt, 0.0)
					assert.LessOrEqual(t, opt, 10.0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := handler.generateRangeOptions(tt.min, tt.max, tt.step, tt.current)
			tt.validate(t, options)
		})
	}
}

func TestGetOperatorSymbol(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		operator string
		expected string
	}{
		{"Greater than", "gt", ">"},
		{"Greater than or equal", "gte", "≥"},
		{"Less than", "lt", "<"},
		{"Less than or equal", "lte", "≤"},
		{"Equal", "eq", "="},
		{"Unknown operator returns as-is", "unknown", "unknown"},
		{"Empty string returns as-is", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getOperatorSymbol(tt.operator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAlertTypeTextLocalized(t *testing.T) {
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
		name      string
		alertType models.AlertType
		language  string
		expected  string
	}{
		{"Temperature in English", models.AlertTemperature, "en-US", "alert_type_temperature"},
		{"Temperature in Ukrainian", models.AlertTemperature, "uk-UA", "alert_type_temperature"},
		{"Humidity in English", models.AlertHumidity, "en-US", "alert_type_humidity"},
		{"Pressure in German", models.AlertPressure, "de-DE", "alert_type_pressure"},
		{"Wind Speed in French", models.AlertWindSpeed, "fr-FR", "alert_type_wind_speed"},
		{"UV Index in Spanish", models.AlertUVIndex, "es-ES", "alert_type_uv_index"},
		{"Air Quality in English", models.AlertAirQuality, "en-US", "alert_type_air_quality"},
		{"Rain in English", models.AlertRain, "en-US", "alert_type_rain"},
		{"Snow in English", models.AlertSnow, "en-US", "alert_type_snow"},
		{"Storm in English", models.AlertStorm, "en-US", "alert_type_storm"},
		{"Unknown type in English", models.AlertType(999), "en-US", "alert_type_unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getAlertTypeTextLocalized(tt.alertType, tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}
