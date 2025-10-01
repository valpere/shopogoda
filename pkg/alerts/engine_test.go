package alerts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    AlertLevel
		expected string
	}{
		{"low level", AlertLevelLow, "Low"},
		{"medium level", AlertLevelMedium, "Medium"},
		{"high level", AlertLevelHigh, "High"},
		{"critical level", AlertLevelCritical, "Critical"},
		{"unknown level", AlertLevel(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertLevel_Color(t *testing.T) {
	tests := []struct {
		name     string
		level    AlertLevel
		expected string
	}{
		{"low level color", AlertLevelLow, "🟢"},
		{"medium level color", AlertLevelMedium, "🟡"},
		{"high level color", AlertLevelHigh, "🟠"},
		{"critical level color", AlertLevelCritical, "🔴"},
		{"unknown level color", AlertLevel(999), "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.Color()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertType_String(t *testing.T) {
	tests := []struct {
		name      string
		alertType AlertType
		expected  string
	}{
		{"temperature", AlertTypeTemperature, "Temperature"},
		{"humidity", AlertTypeHumidity, "Humidity"},
		{"pressure", AlertTypePressure, "Pressure"},
		{"wind speed", AlertTypeWindSpeed, "Wind Speed"},
		{"air quality", AlertTypeAirQuality, "Air Quality"},
		{"uv index", AlertTypeUVIndex, "UV Index"},
		{"visibility", AlertTypeVisibility, "Visibility"},
		{"unknown type", AlertType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alertType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertLevelConstants(t *testing.T) {
	// Verify that alert levels are sequential starting from 1
	assert.Equal(t, AlertLevel(1), AlertLevelLow)
	assert.Equal(t, AlertLevel(2), AlertLevelMedium)
	assert.Equal(t, AlertLevel(3), AlertLevelHigh)
	assert.Equal(t, AlertLevel(4), AlertLevelCritical)
}

func TestAlertTypeConstants(t *testing.T) {
	// Verify that alert types are sequential starting from 1
	assert.Equal(t, AlertType(1), AlertTypeTemperature)
	assert.Equal(t, AlertType(2), AlertTypeHumidity)
	assert.Equal(t, AlertType(3), AlertTypePressure)
	assert.Equal(t, AlertType(4), AlertTypeWindSpeed)
	assert.Equal(t, AlertType(5), AlertTypeAirQuality)
	assert.Equal(t, AlertType(6), AlertTypeUVIndex)
	assert.Equal(t, AlertType(7), AlertTypeVisibility)
}

func TestNewEngine(t *testing.T) {
	engine := NewEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.lastTriggered)
	assert.Equal(t, int64(3600000000000), int64(engine.cooldownPeriod)) // 1 hour in nanoseconds
}

func TestEngine_EvaluateCondition(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name      string
		value     float64
		condition AlertCondition
		expected  bool
	}{
		{
			name:  "greater than - true",
			value: 25.0,
			condition: AlertCondition{
				Operator: "gt",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "greater than - false",
			value: 15.0,
			condition: AlertCondition{
				Operator: "gt",
				Value:    20.0,
			},
			expected: false,
		},
		{
			name:  "less than - true",
			value: 15.0,
			condition: AlertCondition{
				Operator: "lt",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "less than - false",
			value: 25.0,
			condition: AlertCondition{
				Operator: "lt",
				Value:    20.0,
			},
			expected: false,
		},
		{
			name:  "greater than or equal - true (greater)",
			value: 25.0,
			condition: AlertCondition{
				Operator: "gte",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "greater than or equal - true (equal)",
			value: 20.0,
			condition: AlertCondition{
				Operator: "gte",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "less than or equal - true (less)",
			value: 15.0,
			condition: AlertCondition{
				Operator: "lte",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "less than or equal - true (equal)",
			value: 20.0,
			condition: AlertCondition{
				Operator: "lte",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "equal - true",
			value: 20.0,
			condition: AlertCondition{
				Operator: "eq",
				Value:    20.0,
			},
			expected: true,
		},
		{
			name:  "equal - false",
			value: 19.9,
			condition: AlertCondition{
				Operator: "eq",
				Value:    20.0,
			},
			expected: false,
		},
		{
			name:  "invalid operator",
			value: 20.0,
			condition: AlertCondition{
				Operator: "invalid",
				Value:    20.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.EvaluateCondition(tt.value, tt.condition)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_CalculateLevel(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name         string
		alertType    AlertType
		currentValue float64
		threshold    float64
		expected     AlertLevel
	}{
		// Temperature tests
		{"temperature critical deviation", AlertTypeTemperature, 40.0, 20.0, AlertLevelCritical}, // deviation 20
		{"temperature high deviation", AlertTypeTemperature, 32.0, 20.0, AlertLevelHigh},         // deviation 12
		{"temperature medium deviation", AlertTypeTemperature, 28.0, 20.0, AlertLevelMedium},     // deviation 8
		{"temperature low deviation", AlertTypeTemperature, 24.0, 20.0, AlertLevelLow},           // deviation 4

		// Air Quality tests
		{"air quality critical", AlertTypeAirQuality, 350.0, 100.0, AlertLevelCritical},
		{"air quality high", AlertTypeAirQuality, 220.0, 100.0, AlertLevelHigh},
		{"air quality medium", AlertTypeAirQuality, 160.0, 100.0, AlertLevelMedium},
		{"air quality low", AlertTypeAirQuality, 140.0, 100.0, AlertLevelLow},

		// Wind Speed tests
		{"wind speed critical", AlertTypeWindSpeed, 90.0, 40.0, AlertLevelCritical},
		{"wind speed high", AlertTypeWindSpeed, 70.0, 40.0, AlertLevelHigh},
		{"wind speed medium", AlertTypeWindSpeed, 50.0, 40.0, AlertLevelMedium},
		{"wind speed low", AlertTypeWindSpeed, 38.0, 40.0, AlertLevelLow},

		// Humidity tests
		{"humidity high deviation", AlertTypeHumidity, 90.0, 50.0, AlertLevelHigh},     // deviation 40
		{"humidity medium deviation", AlertTypeHumidity, 75.0, 50.0, AlertLevelMedium}, // deviation 25
		{"humidity low deviation", AlertTypeHumidity, 60.0, 50.0, AlertLevelLow},       // deviation 10

		// UV Index tests
		{"uv index critical", AlertTypeUVIndex, 12.0, 5.0, AlertLevelCritical},
		{"uv index high", AlertTypeUVIndex, 9.0, 5.0, AlertLevelHigh},
		{"uv index medium", AlertTypeUVIndex, 6.5, 5.0, AlertLevelMedium},
		{"uv index low", AlertTypeUVIndex, 5.0, 5.0, AlertLevelLow},

		// Default case
		{"unknown type default", AlertType(999), 100.0, 50.0, AlertLevelMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.CalculateLevel(tt.alertType, tt.currentValue, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_CreateAlert(t *testing.T) {
	engine := NewEngine()

	t.Run("temperature alert above threshold", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeTemperature, 35.0, 25.0, "London")

		assert.NotNil(t, alert)
		assert.NotEmpty(t, alert.ID)
		assert.Equal(t, AlertTypeTemperature, alert.Type)
		assert.Equal(t, "London", alert.Location)
		assert.Equal(t, 35.0, alert.Value)
		assert.Equal(t, 25.0, alert.Threshold)
		assert.False(t, alert.IsResolved)
		assert.Nil(t, alert.ResolvedAt)
		assert.Contains(t, alert.Description, "above threshold")
	})

	t.Run("temperature alert below threshold", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeTemperature, 10.0, 20.0, "Moscow")

		assert.Contains(t, alert.Description, "below threshold")
	})

	t.Run("humidity alert", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeHumidity, 85.0, 60.0, "Singapore")

		assert.Equal(t, AlertTypeHumidity, alert.Type)
		assert.Contains(t, alert.Description, "Humidity")
		assert.Contains(t, alert.Description, "%")
	})

	t.Run("air quality alert unhealthy", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeAirQuality, 180.0, 100.0, "Beijing")

		assert.Contains(t, alert.Description, "Air Quality Index")
		assert.Contains(t, alert.Description, "Unhealthy for sensitive groups")
	})

	t.Run("wind speed alert", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeWindSpeed, 70.0, 50.0, "Miami")

		assert.Contains(t, alert.Description, "Wind speed")
		assert.Contains(t, alert.Description, "km/h")
	})

	t.Run("uv index alert high", func(t *testing.T) {
		alert := engine.CreateAlert(AlertTypeUVIndex, 9.0, 6.0, "Sydney")

		assert.Contains(t, alert.Description, "UV Index")
		assert.Contains(t, alert.Description, "Very high UV exposure")
	})

	t.Run("unknown alert type", func(t *testing.T) {
		alert := engine.CreateAlert(AlertType(999), 100.0, 50.0, "Unknown")

		assert.Contains(t, alert.Description, "Value is")
	})
}

func TestEngine_CanTrigger(t *testing.T) {
	engine := NewEngine()

	t.Run("can trigger when never triggered before", func(t *testing.T) {
		can := engine.CanTrigger("test_alert")
		assert.True(t, can)
	})

	t.Run("cannot trigger within cooldown period", func(t *testing.T) {
		engine.MarkTriggered("test_alert")
		can := engine.CanTrigger("test_alert")
		assert.False(t, can)
	})

	t.Run("can trigger after cooldown period", func(t *testing.T) {
		// Set last triggered to 2 hours ago (beyond 1 hour cooldown)
		engine.lastTriggered["old_alert"] = engine.lastTriggered["old_alert"].Add(-2 * 60 * 60 * 1000000000)
		can := engine.CanTrigger("old_alert")
		assert.True(t, can)
	})
}

func TestEngine_MarkTriggered(t *testing.T) {
	engine := NewEngine()

	engine.MarkTriggered("test_alert")

	assert.Contains(t, engine.lastTriggered, "test_alert")
	assert.False(t, engine.lastTriggered["test_alert"].IsZero())
}

func TestEngine_FormatAlertMessage(t *testing.T) {
	engine := NewEngine()
	alert := engine.CreateAlert(AlertTypeTemperature, 35.0, 25.0, "Test Location")

	message := engine.FormatAlertMessage(alert)

	assert.Contains(t, message, alert.Level.Color())
	assert.Contains(t, message, alert.Title)
	assert.Contains(t, message, "Test Location")
	assert.Contains(t, message, "35.00")
	assert.Contains(t, message, "25.00")
	assert.Contains(t, message, alert.Description)
}

func TestEngine_FormatSlackMessage(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name          string
		level         AlertLevel
		expectedColor string
	}{
		{"low severity", AlertLevelLow, "#36a64f"},
		{"medium severity", AlertLevelMedium, "#ffcc00"},
		{"high severity", AlertLevelHigh, "#ff9900"},
		{"critical severity", AlertLevelCritical, "#ff0000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := &Alert{
				Type:        AlertTypeTemperature,
				Level:       tt.level,
				Title:       "Test Alert",
				Description: "Test Description",
				Value:       35.0,
				Threshold:   25.0,
				Location:    "Test Location",
			}

			message := engine.FormatSlackMessage(alert)

			assert.NotNil(t, message)
			assert.Contains(t, message, "attachments")

			attachments := message["attachments"].([]map[string]interface{})
			assert.Len(t, attachments, 1)
			assert.Equal(t, tt.expectedColor, attachments[0]["color"])
			assert.Equal(t, alert.Title, attachments[0]["title"])
			assert.Equal(t, alert.Description, attachments[0]["text"])
		})
	}
}

func TestGenerateAlertID(t *testing.T) {
	id1 := generateAlertID()
	id2 := generateAlertID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "alert_")
	assert.Contains(t, id2, "alert_")
}

func TestValidateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition AlertCondition
		expectErr bool
	}{
		{"valid gt operator", AlertCondition{Operator: "gt", Value: 20.0}, false},
		{"valid lt operator", AlertCondition{Operator: "lt", Value: 20.0}, false},
		{"valid gte operator", AlertCondition{Operator: "gte", Value: 20.0}, false},
		{"valid lte operator", AlertCondition{Operator: "lte", Value: 20.0}, false},
		{"valid eq operator", AlertCondition{Operator: "eq", Value: 20.0}, false},
		{"invalid operator", AlertCondition{Operator: "invalid", Value: 20.0}, true},
		{"empty operator", AlertCondition{Operator: "", Value: 20.0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCondition(tt.condition)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid operator")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAirQualityDescription(t *testing.T) {
	tests := []struct {
		aqi      int
		expected string
	}{
		{30, "Good - Air quality is satisfactory"},
		{75, "Moderate - Air quality is acceptable for most people"},
		{125, "Unhealthy for Sensitive Groups"},
		{175, "Unhealthy - Everyone may experience health effects"},
		{250, "Very Unhealthy - Health alert"},
		{350, "Hazardous - Health warnings of emergency conditions"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetAirQualityDescription(tt.aqi)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUVIndexDescription(t *testing.T) {
	tests := []struct {
		uvIndex  float64
		expected string
	}{
		{1.5, "Low - Minimal protection required"},
		{3.5, "Moderate - Take precautions when outside"},
		{6.5, "High - Protection required"},
		{9.0, "Very High - Extra protection required"},
		{12.0, "Extreme - Avoid being outside"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetUVIndexDescription(tt.uvIndex)
			assert.Equal(t, tt.expected, result)
		})
	}
}
