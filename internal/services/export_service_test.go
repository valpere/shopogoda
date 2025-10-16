package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewExportService(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	localization := NewLocalizationService(logger)

	service := NewExportService(mockDB.DB, logger, localization)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.localization)
}

func TestExportService_GetWeatherData(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	service := NewExportService(mockDB.DB, logger, nil)

	userID := int64(123)
	now := time.Now().UTC()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	t.Run("get weather data successfully", func(t *testing.T) {
		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "location_name", "temperature", "humidity", "pressure",
			"wind_speed", "wind_degree", "visibility", "uv_index", "description",
			"aqi", "co", "no", "no2", "o3", "so2", "pm25", "pm10", "nh3", "timestamp",
		})
		rows.AddRow(
			uuid.New(), userID, "London", 20.5, 65, 1013.0,
			10.2, 180, 10.0, 3.5, "Clear", 2, 0.3, 0.1, 15.2, 45.0, 2.1, 8.5, 12.0, 1.2,
			now.Add(-10*time.Hour),
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "weather_data" WHERE user_id = \$1 AND timestamp >= \$2 ORDER BY timestamp DESC LIMIT \$3`).
			WithArgs(userID, helpers.AnyTime{}, 1000).
			WillReturnRows(rows)

		weatherData, err := service.getWeatherData(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, weatherData, 1)
		assert.Equal(t, userID, weatherData[0].UserID)
		assert.Equal(t, 20.5, weatherData[0].Temperature)
		assert.True(t, weatherData[0].Timestamp.After(thirtyDaysAgo))
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no weather data found", func(t *testing.T) {
		rows := mockDB.Mock.NewRows([]string{"id", "user_id", "location_name", "temperature"})

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "weather_data" WHERE user_id = \$1 AND timestamp >= \$2 ORDER BY timestamp DESC LIMIT \$3`).
			WithArgs(userID, helpers.AnyTime{}, 1000).
			WillReturnRows(rows)

		weatherData, err := service.getWeatherData(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, weatherData, 0)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestExportService_GetAlertConfigs(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	service := NewExportService(mockDB.DB, logger, nil)

	userID := int64(123)

	t.Run("get alert configs successfully", func(t *testing.T) {
		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "condition", "threshold", "is_active", "created_at", "updated_at",
		})
		rows.AddRow(
			uuid.New(), userID, models.AlertTemperature, "greater_than", 30.0, true,
			time.Now(), time.Now(),
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(rows)

		alertConfigs, err := service.getAlertConfigs(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, alertConfigs, 1)
		assert.Equal(t, userID, alertConfigs[0].UserID)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestExportService_GetTriggeredAlerts(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	service := NewExportService(mockDB.DB, logger, nil)

	userID := int64(123)

	t.Run("get triggered alerts successfully", func(t *testing.T) {
		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "severity", "title", "description",
			"value", "threshold", "is_resolved", "created_at", "updated_at",
		})
		rows.AddRow(
			uuid.New(), userID, models.AlertTemperature, models.SeverityHigh,
			"High Temperature", "Temperature exceeded threshold",
			35.0, 30.0, false, time.Now(), time.Now(),
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1 AND created_at >= \$2 ORDER BY created_at DESC`).
			WithArgs(userID, helpers.AnyTime{}).
			WillReturnRows(rows)

		triggeredAlerts, err := service.getTriggeredAlerts(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, userID, triggeredAlerts[0].UserID)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestExportService_GetSubscriptions(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	service := NewExportService(mockDB.DB, logger, nil)

	userID := int64(123)

	t.Run("get subscriptions successfully", func(t *testing.T) {
		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "subscription_type", "frequency", "time_of_day", "is_active", "created_at", "updated_at",
		})
		rows.AddRow(
			uuid.New(), userID, models.SubscriptionDaily, models.FrequencyDaily,
			"08:00", true, time.Now(), time.Now(),
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(rows)

		subscriptions, err := service.getSubscriptions(context.Background(), userID)

		require.NoError(t, err)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, userID, subscriptions[0].UserID)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestExportService_ExportToJSON(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	service := NewExportService(nil, logger, nil)

	user := &models.User{
		ID:       123,
		Username: "testuser",
	}

	weatherData := []models.WeatherData{
		{
			UserID:      123,
			Temperature: 20.5,
			Humidity:    65,
			Timestamp:   time.Now().UTC(),
		},
	}

	exportData := &ExportData{
		User:        user,
		WeatherData: weatherData,
		ExportedAt:  time.Now().UTC(),
		Format:      ExportFormatJSON,
		Type:        ExportTypeWeatherData,
	}

	t.Run("export to JSON successfully", func(t *testing.T) {
		buffer, filename, err := service.exportToJSON(exportData, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_weather_testuser")
		assert.Contains(t, filename, ".json")

		// Verify JSON is valid
		var result ExportData
		err = json.Unmarshal(buffer.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, user.ID, result.User.ID)
		assert.Len(t, result.WeatherData, 1)
	})
}

func TestExportService_ExportToCSV(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	mockLocalization := NewLocalizationService(logger)
	service := NewExportService(nil, logger, mockLocalization)

	user := &models.User{
		ID:       123,
		Username: "testuser",
	}

	weatherData := []models.WeatherData{
		{
			UserID:      123,
			Temperature: 20.5,
			Humidity:    65,
			Pressure:    1013.0,
			Timestamp:   time.Now().UTC(),
		},
	}

	exportData := &ExportData{
		User:        user,
		WeatherData: weatherData,
		ExportedAt:  time.Now().UTC(),
		Format:      ExportFormatCSV,
		Type:        ExportTypeWeatherData,
	}

	t.Run("export to CSV successfully", func(t *testing.T) {
		buffer, filename, err := service.exportToCSV(exportData, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_weather_testuser")
		assert.Contains(t, filename, ".csv")

		// Verify CSV contains expected data
		csvContent := buffer.String()
		assert.Contains(t, csvContent, "testuser")
		assert.Contains(t, csvContent, "20.5") // Temperature
	})

	t.Run("export to CSV with subscriptions", func(t *testing.T) {
		subscriptions := []models.Subscription{
			{
				UserID:           123,
				SubscriptionType: models.SubscriptionDaily,
				Frequency:        models.FrequencyDaily,
				TimeOfDay:        "08:00",
				IsActive:         true,
				CreatedAt:        time.Now().UTC(),
			},
		}
		exportDataWithSubs := &ExportData{
			User:          user,
			Subscriptions: subscriptions,
			ExportedAt:    time.Now().UTC(),
			Format:        ExportFormatCSV,
			Type:          ExportTypeSubscriptions,
		}

		buffer, filename, err := service.exportToCSV(exportDataWithSubs, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".csv")

		csvContent := buffer.String()
		assert.Contains(t, csvContent, "Daily")
		assert.Contains(t, csvContent, "08:00")
	})

	t.Run("export to CSV with alert configs and triggered alerts", func(t *testing.T) {
		alertConfigs := []models.AlertConfig{
			{
				UserID:    123,
				AlertType: models.AlertTemperature,
				Condition: `{"operator":"gt","value":30}`,
				Threshold: 30.0,
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
			},
		}
		triggeredAlerts := []models.EnvironmentalAlert{
			{
				UserID:    123,
				AlertType: models.AlertTemperature,
				Severity:  models.SeverityHigh,
				Title:     "High Temperature",
				Value:     35.0,
				Threshold: 30.0,
				CreatedAt: time.Now().UTC(),
			},
		}
		exportDataWithAlerts := &ExportData{
			User:            user,
			AlertConfigs:    alertConfigs,
			TriggeredAlerts: triggeredAlerts,
			ExportedAt:      time.Now().UTC(),
			Format:          ExportFormatCSV,
			Type:            ExportTypeAlerts,
		}

		buffer, filename, err := service.exportToCSV(exportDataWithAlerts, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".csv")

		csvContent := buffer.String()
		assert.Contains(t, csvContent, "Temperature")
		assert.Contains(t, csvContent, "30.0")
		assert.Contains(t, csvContent, "35.0")
		assert.Contains(t, csvContent, "High")
	})
}

func TestExportService_ExportToTXT(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	mockLocalization := NewLocalizationService(logger)
	service := NewExportService(nil, logger, mockLocalization)

	user := &models.User{
		ID:        123,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Language:  "en-US",
		Units:     "metric",
		Timezone:  "UTC",
	}

	weatherData := []models.WeatherData{
		{
			UserID:      123,
			Temperature: 20.5,
			Humidity:    65,
			Description: "Clear sky",
			Timestamp:   time.Now().UTC(),
		},
	}

	subscriptions := []models.Subscription{
		{
			UserID:           123,
			SubscriptionType: models.SubscriptionDaily,
			Frequency:        models.FrequencyDaily,
			TimeOfDay:        "08:00",
			IsActive:         true,
			CreatedAt:        time.Now().UTC(),
		},
	}

	exportData := &ExportData{
		User:          user,
		WeatherData:   weatherData,
		Subscriptions: subscriptions,
		ExportedAt:    time.Now().UTC(),
		Format:        ExportFormatTXT,
		Type:          ExportTypeAll,
	}

	t.Run("export to TXT successfully", func(t *testing.T) {
		buffer, filename, err := service.exportToTXT(exportData, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_all_testuser")
		assert.Contains(t, filename, ".txt")

		// Verify TXT contains expected sections
		txtContent := buffer.String()
		assert.Contains(t, txtContent, "testuser")
		assert.Contains(t, txtContent, "Temperature: 20.5")
		assert.Contains(t, txtContent, "Clear sky")
		assert.Contains(t, txtContent, "Subscriptions")
		assert.Contains(t, txtContent, "End of Export")
	})

	t.Run("export to TXT with alert configs", func(t *testing.T) {
		alertConfigs := []models.AlertConfig{
			{
				UserID:    123,
				AlertType: models.AlertTemperature,
				Condition: `{"operator":"gt","value":30}`,
				Threshold: 30.0,
				IsActive:  true,
				CreatedAt: time.Now().UTC(),
			},
		}
		exportDataWithAlerts := &ExportData{
			User:         user,
			AlertConfigs: alertConfigs,
			ExportedAt:   time.Now().UTC(),
			Format:       ExportFormatTXT,
			Type:         ExportTypeAlerts,
		}

		buffer, filename, err := service.exportToTXT(exportDataWithAlerts, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".txt")

		txtContent := buffer.String()
		assert.Contains(t, txtContent, "Alert Configurations")
		assert.Contains(t, txtContent, "Temperature")
		assert.Contains(t, txtContent, "Threshold: 30.0")
	})

	t.Run("export to TXT with triggered alerts", func(t *testing.T) {
		triggeredAlerts := []models.EnvironmentalAlert{
			{
				UserID:      123,
				AlertType:   models.AlertTemperature,
				Severity:    models.SeverityHigh,
				Title:       "High Temperature Alert",
				Description: "Temperature exceeded threshold",
				Value:       35.0,
				Threshold:   30.0,
				IsResolved:  false,
				CreatedAt:   time.Now().UTC(),
			},
		}
		exportDataWithTriggered := &ExportData{
			User:            user,
			TriggeredAlerts: triggeredAlerts,
			ExportedAt:      time.Now().UTC(),
			Format:          ExportFormatTXT,
			Type:            ExportTypeAlerts,
		}

		buffer, filename, err := service.exportToTXT(exportDataWithTriggered, "en")

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".txt")

		txtContent := buffer.String()
		assert.Contains(t, txtContent, "Triggered Alerts")
		assert.Contains(t, txtContent, "High Temperature Alert")
		assert.Contains(t, txtContent, "Value: 35.0")
		assert.Contains(t, txtContent, "Threshold: 30.0")
		assert.Contains(t, txtContent, "Resolved: false")
	})
}

func TestExportService_ExportUserData(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	logger := helpers.NewSilentTestLogger()
	mockLocalization := NewLocalizationService(logger)
	service := NewExportService(mockDB.DB, logger, mockLocalization)

	userID := int64(123)

	t.Run("export weather data as JSON", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		// Mock weather data query
		weatherRows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "location_name", "temperature", "humidity", "pressure",
			"wind_speed", "wind_degree", "visibility", "uv_index", "description",
			"aqi", "co", "no", "no2", "o3", "so2", "pm25", "pm10", "nh3", "timestamp",
		})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "weather_data"`).
			WillReturnRows(weatherRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeWeatherData,
			ExportFormatJSON,
			"en",
		)

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".json")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("unsupported export type", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportType("invalid"),
			ExportFormatJSON,
			"en",
		)

		assert.Error(t, err)
		assert.Nil(t, buffer)
		assert.Empty(t, filename)
		assert.Contains(t, err.Error(), "unsupported export type")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("unsupported export format", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		// Mock weather data query
		weatherRows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "location_name", "temperature",
		})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "weather_data"`).
			WillReturnRows(weatherRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeWeatherData,
			ExportFormat("invalid"),
			"en",
		)

		assert.Error(t, err)
		assert.Nil(t, buffer)
		assert.Empty(t, filename)
		assert.Contains(t, err.Error(), "unsupported export format")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("export alerts as CSV", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		// Mock alert configs query
		alertRows := mockDB.Mock.NewRows([]string{"id", "user_id", "alert_type", "condition", "threshold", "is_active"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs"`).
			WillReturnRows(alertRows)

		// Mock triggered alerts query
		triggeredRows := mockDB.Mock.NewRows([]string{"id", "user_id", "alert_type", "severity", "title", "description"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts"`).
			WillReturnRows(triggeredRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeAlerts,
			ExportFormatCSV,
			"en",
		)

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".csv")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("export subscriptions as TXT", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		// Mock subscriptions query
		subRows := mockDB.Mock.NewRows([]string{"id", "user_id", "notification_type", "frequency"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions"`).
			WillReturnRows(subRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeSubscriptions,
			ExportFormatTXT,
			"en",
		)

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".txt")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("export all data as JSON", func(t *testing.T) {
		// Mock user query
		userRows := mockDB.Mock.NewRows([]string{"id", "username"})
		userRows.AddRow(userID, "testuser")
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1 ORDER BY "users"\."id" LIMIT \$2`).
			WithArgs(userID, 1).
			WillReturnRows(userRows)

		// Mock weather data query
		weatherRows := mockDB.Mock.NewRows([]string{"id", "user_id", "location_name", "temperature"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "weather_data"`).
			WillReturnRows(weatherRows)

		// Mock alert configs query
		alertRows := mockDB.Mock.NewRows([]string{"id", "user_id", "alert_type", "condition", "threshold"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs"`).
			WillReturnRows(alertRows)

		// Mock triggered alerts query
		triggeredRows := mockDB.Mock.NewRows([]string{"id", "user_id", "alert_type", "severity"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts"`).
			WillReturnRows(triggeredRows)

		// Mock subscriptions query
		subRows := mockDB.Mock.NewRows([]string{"id", "user_id", "notification_type"})
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions"`).
			WillReturnRows(subRows)

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeAll,
			ExportFormatJSON,
			"en",
		)

		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".json")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("getUserData error handling", func(t *testing.T) {
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WillReturnError(errors.New("user not found"))

		buffer, filename, err := service.ExportUserData(
			context.Background(),
			userID,
			ExportTypeWeatherData,
			ExportFormatJSON,
			"en",
		)

		assert.Error(t, err)
		assert.Nil(t, buffer)
		assert.Empty(t, filename)
		assert.Contains(t, err.Error(), "failed to get user data")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestExportConstants(t *testing.T) {
	t.Run("export formats", func(t *testing.T) {
		assert.Equal(t, ExportFormat("json"), ExportFormatJSON)
		assert.Equal(t, ExportFormat("csv"), ExportFormatCSV)
		assert.Equal(t, ExportFormat("txt"), ExportFormatTXT)
	})

	t.Run("export types", func(t *testing.T) {
		assert.Equal(t, ExportType("weather"), ExportTypeWeatherData)
		assert.Equal(t, ExportType("alerts"), ExportTypeAlerts)
		assert.Equal(t, ExportType("subscriptions"), ExportTypeSubscriptions)
		assert.Equal(t, ExportType("all"), ExportTypeAll)
	})
}
