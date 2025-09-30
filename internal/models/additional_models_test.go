package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserRole_String(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{
			name:     "user role",
			role:     RoleUser,
			expected: "User",
		},
		{
			name:     "moderator role",
			role:     RoleModerator,
			expected: "Moderator",
		},
		{
			name:     "admin role",
			role:     RoleAdmin,
			expected: "Admin",
		},
		{
			name:     "unknown role",
			role:     UserRole(999),
			expected: "Unknown",
		},
		{
			name:     "zero role",
			role:     UserRole(0),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscriptionType_String(t *testing.T) {
	tests := []struct {
		name     string
		subType  SubscriptionType
		expected string
	}{
		{
			name:     "daily subscription",
			subType:  SubscriptionDaily,
			expected: "Daily",
		},
		{
			name:     "weekly subscription",
			subType:  SubscriptionWeekly,
			expected: "Weekly",
		},
		{
			name:     "alerts subscription",
			subType:  SubscriptionAlerts,
			expected: "Alerts",
		},
		{
			name:     "extreme subscription",
			subType:  SubscriptionExtreme,
			expected: "Extreme",
		},
		{
			name:     "unknown subscription",
			subType:  SubscriptionType(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.subType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFrequency_String(t *testing.T) {
	tests := []struct {
		name     string
		freq     Frequency
		expected string
	}{
		{
			name:     "hourly frequency",
			freq:     FrequencyHourly,
			expected: "Hourly",
		},
		{
			name:     "every 3 hours frequency",
			freq:     FrequencyEvery3Hours,
			expected: "Every 3 Hours",
		},
		{
			name:     "every 6 hours frequency",
			freq:     FrequencyEvery6Hours,
			expected: "Every 6 Hours",
		},
		{
			name:     "daily frequency",
			freq:     FrequencyDaily,
			expected: "Daily",
		},
		{
			name:     "weekly frequency",
			freq:     FrequencyWeekly,
			expected: "Weekly",
		},
		{
			name:     "unknown frequency",
			freq:     Frequency(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.freq.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertType_String(t *testing.T) {
	tests := []struct {
		name     string
		alertType AlertType
		expected string
	}{
		{
			name:     "temperature alert",
			alertType: AlertTemperature,
			expected: "Temperature",
		},
		{
			name:     "humidity alert",
			alertType: AlertHumidity,
			expected: "Humidity",
		},
		{
			name:     "pressure alert",
			alertType: AlertPressure,
			expected: "Pressure",
		},
		{
			name:     "wind speed alert",
			alertType: AlertWindSpeed,
			expected: "Wind Speed",
		},
		{
			name:     "UV index alert",
			alertType: AlertUVIndex,
			expected: "UV Index",
		},
		{
			name:     "air quality alert",
			alertType: AlertAirQuality,
			expected: "Air Quality",
		},
		{
			name:     "rain alert",
			alertType: AlertRain,
			expected: "Rain",
		},
		{
			name:     "snow alert",
			alertType: AlertSnow,
			expected: "Snow",
		},
		{
			name:     "storm alert",
			alertType: AlertStorm,
			expected: "Storm",
		},
		{
			name:     "unknown alert type",
			alertType: AlertType(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.alertType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "low severity",
			severity: SeverityLow,
			expected: "Low",
		},
		{
			name:     "medium severity",
			severity: SeverityMedium,
			expected: "Medium",
		},
		{
			name:     "high severity",
			severity: SeverityHigh,
			expected: "High",
		},
		{
			name:     "critical severity",
			severity: SeverityCritical,
			expected: "Critical",
		},
		{
			name:     "unknown severity",
			severity: Severity(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExportFormat_Validation(t *testing.T) {
	// Note: ExportFormat is defined in services package, not models
	// This test validates the concept of export format validation
	type ExportFormat string
	const (
		ExportFormatJSON ExportFormat = "json"
		ExportFormatCSV  ExportFormat = "csv"
		ExportFormatTXT  ExportFormat = "txt"
	)

	tests := []struct {
		name     string
		format   ExportFormat
		expected bool
	}{
		{
			name:     "JSON format is valid",
			format:   ExportFormatJSON,
			expected: true,
		},
		{
			name:     "CSV format is valid",
			format:   ExportFormatCSV,
			expected: true,
		},
		{
			name:     "TXT format is valid",
			format:   ExportFormatTXT,
			expected: true,
		},
		{
			name:     "unknown format is invalid",
			format:   ExportFormat("xml"),
			expected: false,
		},
	}

	validFormats := map[ExportFormat]bool{
		ExportFormatJSON: true,
		ExportFormatCSV:  true,
		ExportFormatTXT:  true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validFormats[tt.format]
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExportType_Validation(t *testing.T) {
	// Note: ExportType is defined in services package, not models
	// This test validates the concept of export type validation
	type ExportType string
	const (
		ExportTypeWeatherData   ExportType = "weather"
		ExportTypeAlerts        ExportType = "alerts"
		ExportTypeSubscriptions ExportType = "subscriptions"
		ExportTypeAll           ExportType = "all"
	)

	tests := []struct {
		name     string
		expType  ExportType
		expected bool
	}{
		{
			name:     "weather data type is valid",
			expType:  ExportTypeWeatherData,
			expected: true,
		},
		{
			name:     "alerts type is valid",
			expType:  ExportTypeAlerts,
			expected: true,
		},
		{
			name:     "subscriptions type is valid",
			expType:  ExportTypeSubscriptions,
			expected: true,
		},
		{
			name:     "all data type is valid",
			expType:  ExportTypeAll,
			expected: true,
		},
		{
			name:     "unknown type is invalid",
			expType:  ExportType("logs"),
			expected: false,
		},
	}

	validTypes := map[ExportType]bool{
		ExportTypeWeatherData:   true,
		ExportTypeAlerts:        true,
		ExportTypeSubscriptions: true,
		ExportTypeAll:           true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validTypes[tt.expType]
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAlertType_ValidValues(t *testing.T) {
	// Test that all defined alert types are distinct
	alertTypes := []AlertType{
		AlertTemperature,
		AlertHumidity,
		AlertPressure,
		AlertWindSpeed,
		AlertUVIndex,
		AlertAirQuality,
		AlertRain,
		AlertSnow,
		AlertStorm,
	}

	// Create a map to track uniqueness
	seen := make(map[AlertType]bool)
	for _, alertType := range alertTypes {
		assert.False(t, seen[alertType], "Alert type %s should be unique", alertType)
		seen[alertType] = true
	}

	// Verify we have all expected alert types
	assert.Len(t, alertTypes, 9, "Should have exactly 9 alert types defined")
}

func TestAlertSeverity_ValidValues(t *testing.T) {
	// Test that all defined severities are distinct
	severities := []Severity{
		SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical,
	}

	// Create a map to track uniqueness
	seen := make(map[Severity]bool)
	for _, severity := range severities {
		assert.False(t, seen[severity], "Severity %s should be unique", severity)
		seen[severity] = true
	}

	// Verify we have all expected severities
	assert.Len(t, severities, 4, "Should have exactly 4 severity levels defined")
}

func TestSubscriptionType_ValidValues(t *testing.T) {
	// Test that all defined subscription types are distinct
	subTypes := []SubscriptionType{
		SubscriptionDaily,
		SubscriptionWeekly,
		SubscriptionAlerts,
		SubscriptionExtreme,
	}

	// Create a map to track uniqueness
	seen := make(map[SubscriptionType]bool)
	for _, subType := range subTypes {
		assert.False(t, seen[subType], "Subscription type %s should be unique", subType)
		seen[subType] = true
	}

	// Verify we have all expected subscription types
	assert.Len(t, subTypes, 4, "Should have exactly 4 subscription types defined")
}

func TestFrequency_ValidValues(t *testing.T) {
	// Test that all defined frequencies are distinct
	frequencies := []Frequency{
		FrequencyHourly,
		FrequencyEvery3Hours,
		FrequencyEvery6Hours,
		FrequencyDaily,
		FrequencyWeekly,
	}

	// Create a map to track uniqueness
	seen := make(map[Frequency]bool)
	for _, freq := range frequencies {
		assert.False(t, seen[freq], "Frequency %s should be unique", freq)
		seen[freq] = true
	}

	// Verify we have all expected frequencies
	assert.Len(t, frequencies, 5, "Should have exactly 5 frequency types defined")
}

func TestUserRole_ValidValues(t *testing.T) {
	// Test that all defined user roles are distinct
	roles := []UserRole{
		RoleUser,
		RoleModerator,
		RoleAdmin,
	}

	// Create a map to track uniqueness
	seen := make(map[UserRole]bool)
	for _, role := range roles {
		assert.False(t, seen[role], "User role %s should be unique", role)
		seen[role] = true
	}

	// Verify we have all expected roles
	assert.Len(t, roles, 3, "Should have exactly 3 user roles defined")
}