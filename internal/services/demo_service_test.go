package services

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewDemoService(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := zerolog.Nop()

	service := NewDemoService(mockDB.DB, &logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.logger)
}

func TestIsDemoUser(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := zerolog.Nop()
	service := NewDemoService(mockDB.DB, &logger)

	tests := []struct {
		name     string
		userID   int64
		expected bool
	}{
		{"Demo user", DemoUserID, true},
		{"Regular user", 12345, false},
		{"Another user", 67890, false},
		{"Zero ID", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsDemoUser(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetWeatherDescription(t *testing.T) {
	tests := []struct {
		name     string
		temp     int
		expected string
	}{
		{"Freezing", -5, "Freezing"},
		{"Cold", 5, "Cold"},
		{"Cool", 15, "Cool"},
		{"Mild", 22, "Mild"},
		{"Warm", 27, "Warm"},
		{"Hot", 35, "Hot"},
		{"Zero", 0, "Freezing"},
		{"Exactly 10", 10, "Cool"},
		{"Exactly 20", 20, "Mild"},
		{"Exactly 25", 25, "Warm"},
		{"Exactly 30", 30, "Hot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeatherDescription(tt.temp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetWeatherIcon(t *testing.T) {
	tests := []struct {
		name     string
		temp     int
		expected string
	}{
		{"Freezing", -5, "❄️"},
		{"Cold", 5, "🌧️"},
		{"Cool", 15, "⛅"},
		{"Warm", 25, "☀️"},
		{"Hot", 35, "🔥"},
		{"Zero", 0, "❄️"},
		{"Exactly 10", 10, "⛅"},
		{"Exactly 20", 20, "☀️"},
		{"Exactly 30", 30, "🔥"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWeatherIcon(tt.temp)
			assert.Equal(t, tt.expected, result)
		})
	}
}
