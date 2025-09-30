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
		name     string
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
