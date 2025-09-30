package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name: "full name with both first and last name",
			user: User{
				FirstName: "John",
				LastName:  "Doe",
				Username:  "johndoe",
			},
			expected: "John Doe",
		},
		{
			name: "only first name",
			user: User{
				FirstName: "John",
				Username:  "johndoe",
			},
			expected: "John",
		},
		{
			name: "only username when no names",
			user: User{
				Username: "johndoe",
			},
			expected: "@johndoe",
		},
		{
			name: "fallback to user ID when no names or username",
			user: User{
				ID: 12345,
			},
			expected: "User_12345",
		},
		{
			name: "empty last name should not affect display",
			user: User{
				FirstName: "John",
				LastName:  "",
				Username:  "johndoe",
			},
			expected: "John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.GetDisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_HasLocation(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "user with complete location",
			user: User{
				LocationName: "London",
				Latitude:     51.5074,
				Longitude:    -0.1278,
			},
			expected: true,
		},
		{
			name: "user with coordinates but no name",
			user: User{
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			expected: false,
		},
		{
			name: "user with name but no coordinates",
			user: User{
				LocationName: "London",
			},
			expected: true,
		},
		{
			name: "user with empty location",
			user: User{
				LocationName: "",
				Latitude:     0,
				Longitude:    0,
			},
			expected: false,
		},
		{
			name: "user with partial coordinates",
			user: User{
				LocationName: "London",
				Latitude:     51.5074,
				Longitude:    0, // Missing longitude
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.HasLocation()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name:     "admin user",
			user:     User{Role: RoleAdmin},
			expected: true,
		},
		{
			name:     "moderator user",
			user:     User{Role: RoleModerator},
			expected: false,
		},
		{
			name:     "regular user",
			user:     User{Role: RoleUser},
			expected: false,
		},
		{
			name:     "user with undefined role",
			user:     User{Role: UserRole(999)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.IsAdmin()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_IsModerator(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name:     "moderator user",
			user:     User{Role: RoleModerator},
			expected: true,
		},
		{
			name:     "admin user (also moderator)",
			user:     User{Role: RoleAdmin},
			expected: true,
		},
		{
			name:     "regular user",
			user:     User{Role: RoleUser},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.user.IsModerator()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWeatherData_IsRecent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		weather  WeatherData
		expected bool
	}{
		{
			name: "recent weather data (5 minutes ago)",
			weather: WeatherData{
				Timestamp: now.Add(-5 * time.Minute),
			},
			expected: true,
		},
		{
			name: "old weather data (2 hours ago)",
			weather: WeatherData{
				Timestamp: now.Add(-2 * time.Hour),
			},
			expected: false,
		},
		{
			name: "weather data from future",
			weather: WeatherData{
				Timestamp: now.Add(1 * time.Hour),
			},
			expected: true, // Future data is considered recent
		},
		{
			name: "weather data exactly 1 hour ago",
			weather: WeatherData{
				Timestamp: now.Add(-1 * time.Hour),
			},
			expected: false, // Exactly 1 hour is not recent
		},
		{
			name: "weather data 59 minutes ago",
			weather: WeatherData{
				Timestamp: now.Add(-59 * time.Minute),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.weather.IsRecent()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentalAlert_IsTriggered(t *testing.T) {
	tests := []struct {
		name     string
		alert    EnvironmentalAlert
		value    float64
		expected bool
	}{
		{
			name: "temperature greater than threshold - triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertTemperature,
				Threshold: 25.0,
			},
			value:    26.5,
			expected: true,
		},
		{
			name: "temperature greater than threshold - not triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertTemperature,
				Threshold: 25.0,
			},
			value:    24.5,
			expected: false,
		},
		{
			name: "humidity less than threshold - triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertHumidity,
				Threshold: 70.0,
			},
			value:    65.0,
			expected: true,
		},
		{
			name: "humidity less than threshold - not triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertHumidity,
				Threshold: 70.0,
			},
			value:    75.0,
			expected: false,
		},
		{
			name: "pressure equal to threshold - triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertPressure,
				Threshold: 1013.0,
			},
			value:    1013.0,
			expected: true,
		},
		{
			name: "pressure equal to threshold - not triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertPressure,
				Threshold: 1013.0,
			},
			value:    1012.0,
			expected: false,
		},
		{
			name: "invalid condition - not triggered",
			alert: EnvironmentalAlert{
				AlertType: AlertType(999), // Invalid alert type
				Threshold: 25.0,
			},
			value:    26.5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alert.IsTriggered(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentalAlert_GetSeverityText(t *testing.T) {
	tests := []struct {
		name     string
		alert    EnvironmentalAlert
		expected string
	}{
		{
			name:     "low severity",
			alert:    EnvironmentalAlert{Severity: SeverityLow},
			expected: "Low",
		},
		{
			name:     "medium severity",
			alert:    EnvironmentalAlert{Severity: SeverityMedium},
			expected: "Medium",
		},
		{
			name:     "high severity",
			alert:    EnvironmentalAlert{Severity: SeverityHigh},
			expected: "High",
		},
		{
			name:     "critical severity",
			alert:    EnvironmentalAlert{Severity: SeverityCritical},
			expected: "Critical",
		},
		{
			name:     "unknown severity",
			alert:    EnvironmentalAlert{Severity: Severity(999)},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alert.GetSeverityText()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscription_IsActiveTime(t *testing.T) {
	// Mock current time
	mockTime := time.Date(2023, 1, 1, 8, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		subscription Subscription
		currentTime  time.Time
		expected     bool
	}{
		{
			name: "exact time match",
			subscription: Subscription{
				TimeOfDay: "08:30",
				IsActive:  true,
			},
			currentTime: mockTime,
			expected:    true,
		},
		{
			name: "time doesn't match",
			subscription: Subscription{
				TimeOfDay: "09:30",
				IsActive:  true,
			},
			currentTime: mockTime,
			expected:    false,
		},
		{
			name: "inactive subscription",
			subscription: Subscription{
				TimeOfDay: "08:30",
				IsActive:  false,
			},
			currentTime: mockTime,
			expected:    false,
		},
		{
			name: "invalid time format",
			subscription: Subscription{
				TimeOfDay: "invalid",
				IsActive:  true,
			},
			currentTime: mockTime,
			expected:    false,
		},
		{
			name: "empty time",
			subscription: Subscription{
				TimeOfDay: "",
				IsActive:  true,
			},
			currentTime: mockTime,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.subscription.IsActiveTime(tt.currentTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscription_ShouldNotify(t *testing.T) {
	mockTime := time.Date(2023, 1, 1, 8, 0, 0, 0, time.UTC) // Sunday

	tests := []struct {
		name         string
		subscription Subscription
		currentTime  time.Time
		expected     bool
	}{
		{
			name: "daily subscription - should notify",
			subscription: Subscription{
				SubscriptionType: SubscriptionDaily,
				Frequency:        FrequencyDaily,
				TimeOfDay:        "08:00",
				IsActive:         true,
			},
			currentTime: mockTime,
			expected:    true,
		},
		{
			name: "weekly subscription on Sunday - should notify",
			subscription: Subscription{
				SubscriptionType: SubscriptionWeekly,
				Frequency:        FrequencyWeekly,
				TimeOfDay:        "08:00",
				IsActive:         true,
			},
			currentTime: mockTime, // Sunday
			expected:    true,
		},
		{
			name: "weekly subscription on Monday - should not notify",
			subscription: Subscription{
				SubscriptionType: SubscriptionWeekly,
				Frequency:        FrequencyWeekly,
				TimeOfDay:        "08:00",
				IsActive:         true,
			},
			currentTime: mockTime.AddDate(0, 0, 1), // Monday
			expected:    false,
		},
		{
			name: "inactive subscription - should not notify",
			subscription: Subscription{
				SubscriptionType: SubscriptionDaily,
				Frequency:        FrequencyDaily,
				TimeOfDay:        "08:00",
				IsActive:         false,
			},
			currentTime: mockTime,
			expected:    false,
		},
		{
			name: "wrong time - should not notify",
			subscription: Subscription{
				SubscriptionType: SubscriptionDaily,
				Frequency:        FrequencyDaily,
				TimeOfDay:        "09:00",
				IsActive:         true,
			},
			currentTime: mockTime,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.subscription.ShouldNotify(tt.currentTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}
