package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestAlertService_CreateAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful alert creation", func(t *testing.T) {
		alert := helpers.MockAlert(123)

		// Mock database expectations
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "environmental_alerts"`).
			WillReturnRows(mockDB.Mock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow("alert-123", alert.CreatedAt, alert.CreatedAt))
		mockDB.Mock.ExpectCommit()

		err := service.CreateAlert(context.Background(), alert)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("invalid alert type", func(t *testing.T) {
		alert := helpers.MockAlert(123)
		alert.Type = models.AlertType("invalid")

		err := service.CreateAlert(context.Background(), alert)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid alert type")
	})

	t.Run("invalid threshold", func(t *testing.T) {
		alert := helpers.MockAlert(123)
		alert.Threshold = -999 // Invalid threshold

		err := service.CreateAlert(context.Background(), alert)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid threshold")
	})

	t.Run("database error", func(t *testing.T) {
		alert := helpers.MockAlert(123)

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "environmental_alerts"`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		err := service.CreateAlert(context.Background(), alert)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_GetAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful alert retrieval", func(t *testing.T) {
		alertID := "alert-123"
		expectedAlert := helpers.MockAlert(123)

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		}).AddRow(
			alertID, expectedAlert.UserID, expectedAlert.Type, expectedAlert.Threshold,
			expectedAlert.Condition, expectedAlert.Value, expectedAlert.Severity,
			expectedAlert.Title, expectedAlert.Description, expectedAlert.IsActive,
			expectedAlert.CreatedAt, expectedAlert.TriggeredAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE "environmental_alerts"\."id" = \$1`).
			WithArgs(alertID).
			WillReturnRows(rows)

		alert, err := service.GetAlert(context.Background(), alertID)

		assert.NoError(t, err)
		assert.NotNil(t, alert)
		assert.Equal(t, expectedAlert.UserID, alert.UserID)
		assert.Equal(t, expectedAlert.Type, alert.Type)
		assert.Equal(t, expectedAlert.Threshold, alert.Threshold)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		alertID := "nonexistent"

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE "environmental_alerts"\."id" = \$1`).
			WithArgs(alertID).
			WillReturnError(errors.New("record not found"))

		alert, err := service.GetAlert(context.Background(), alertID)

		assert.Error(t, err)
		assert.Nil(t, alert)
		assert.Contains(t, err.Error(), "record not found")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_GetUserAlerts(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful user alerts retrieval", func(t *testing.T) {
		userID := int64(123)
		alerts := []*models.EnvironmentalAlert{
			helpers.MockAlert(userID),
			helpers.MockAlert(userID),
		}

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		})

		for _, alert := range alerts {
			rows.AddRow(
				"alert-"+string(rune(alert.UserID)), alert.UserID, alert.Type, alert.Threshold,
				alert.Condition, alert.Value, alert.Severity,
				alert.Title, alert.Description, alert.IsActive,
				alert.CreatedAt, alert.TriggeredAt,
			)
		}

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := service.GetUserAlerts(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, userID, result[0].UserID)
		assert.Equal(t, userID, result[1].UserID)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no alerts found", func(t *testing.T) {
		userID := int64(999)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1`).
			WithArgs(userID).
			WillReturnRows(mockDB.Mock.NewRows([]string{
				"id", "user_id", "type", "threshold", "condition", "value", "severity",
				"title", "description", "is_active", "created_at", "triggered_at",
			}))

		result, err := service.GetUserAlerts(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_CheckAlertsForUser(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	weatherData := helpers.MockWeatherData(123)

	t.Run("temperature alert triggered", func(t *testing.T) {
		userID := int64(123)
		alert := helpers.MockAlert(userID)
		alert.Type = models.AlertTemperature
		alert.Threshold = 25.0
		alert.Condition = "greater_than"

		// Weather data has temperature > threshold
		weatherData.Temperature = 26.5

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		}).AddRow(
			"alert-123", alert.UserID, alert.Type, alert.Threshold,
			alert.Condition, alert.Value, alert.Severity,
			alert.Title, alert.Description, alert.IsActive,
			alert.CreatedAt, alert.TriggeredAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		triggeredAlerts, err := service.CheckAlertsForUser(context.Background(), userID, weatherData)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, models.AlertTemperature, triggeredAlerts[0].Type)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no alerts triggered", func(t *testing.T) {
		userID := int64(123)
		alert := helpers.MockAlert(userID)
		alert.Type = models.AlertTemperature
		alert.Threshold = 30.0 // Higher than weather data
		alert.Condition = "greater_than"

		// Weather data has temperature < threshold
		weatherData.Temperature = 20.5

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		}).AddRow(
			"alert-123", alert.UserID, alert.Type, alert.Threshold,
			alert.Condition, alert.Value, alert.Severity,
			alert.Title, alert.Description, alert.IsActive,
			alert.CreatedAt, alert.TriggeredAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		triggeredAlerts, err := service.CheckAlertsForUser(context.Background(), userID, weatherData)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 0)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("humidity alert with less_than condition", func(t *testing.T) {
		userID := int64(123)
		alert := helpers.MockAlert(userID)
		alert.Type = models.AlertHumidity
		alert.Threshold = 70.0
		alert.Condition = "less_than"

		// Weather data has humidity < threshold
		weatherData.Humidity = 65

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		}).AddRow(
			"alert-123", alert.UserID, alert.Type, alert.Threshold,
			alert.Condition, alert.Value, alert.Severity,
			alert.Title, alert.Description, alert.IsActive,
			alert.CreatedAt, alert.TriggeredAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		triggeredAlerts, err := service.CheckAlertsForUser(context.Background(), userID, weatherData)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, models.AlertHumidity, triggeredAlerts[0].Type)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("air quality alert", func(t *testing.T) {
		userID := int64(123)
		alert := helpers.MockAlert(userID)
		alert.Type = models.AlertAirQuality
		alert.Threshold = 3.0
		alert.Condition = "greater_than"

		// Weather data has AQI > threshold
		weatherData.AQI = 4

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "type", "threshold", "condition", "value", "severity",
			"title", "description", "is_active", "created_at", "triggered_at",
		}).AddRow(
			"alert-123", alert.UserID, alert.Type, alert.Threshold,
			alert.Condition, alert.Value, alert.Severity,
			alert.Title, alert.Description, alert.IsActive,
			alert.CreatedAt, alert.TriggeredAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "environmental_alerts" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		triggeredAlerts, err := service.CheckAlertsForUser(context.Background(), userID, weatherData)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, models.AlertAirQuality, triggeredAlerts[0].Type)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_TriggerAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful alert trigger", func(t *testing.T) {
		alert := helpers.MockAlert(123)
		value := 26.5

		// Mock update expectations
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "environmental_alerts" SET`).
			WithArgs(helpers.AnyTime{}, value, helpers.AnyTime{}, alert.ID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.TriggerAlert(context.Background(), alert, value)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error during trigger", func(t *testing.T) {
		alert := helpers.MockAlert(123)
		value := 26.5

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "environmental_alerts" SET`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		err := service.TriggerAlert(context.Background(), alert, value)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_UpdateAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful alert update", func(t *testing.T) {
		alert := helpers.MockAlert(123)
		alert.Threshold = 30.0 // Changed threshold

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "environmental_alerts" SET`).
			WithArgs(helpers.AnyTime{}, alert.ID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.UpdateAlert(context.Background(), alert)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		alert := helpers.MockAlert(123)

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "environmental_alerts" SET`).
			WillReturnResult(helpers.NewResult(0, 0)) // No rows affected
		mockDB.Mock.ExpectCommit()

		err := service.UpdateAlert(context.Background(), alert)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert not found")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_DeleteAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	t.Run("successful alert deletion", func(t *testing.T) {
		alertID := "alert-123"

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`DELETE FROM "environmental_alerts" WHERE "environmental_alerts"\."id" = \$1`).
			WithArgs(alertID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteAlert(context.Background(), alertID)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		alertID := "nonexistent"

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`DELETE FROM "environmental_alerts" WHERE "environmental_alerts"\."id" = \$1`).
			WithArgs(alertID).
			WillReturnResult(helpers.NewResult(0, 0))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteAlert(context.Background(), alertID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alert not found")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error during deletion", func(t *testing.T) {
		alertID := "alert-123"

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`DELETE FROM "environmental_alerts" WHERE "environmental_alerts"\."id" = \$1`).
			WithArgs(alertID).
			WillReturnError(errors.New("foreign key constraint"))
		mockDB.Mock.ExpectRollback()

		err := service.DeleteAlert(context.Background(), alertID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "foreign key constraint")
		mockDB.ExpectationsWereMet(t)
	})
}