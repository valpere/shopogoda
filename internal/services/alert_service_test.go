package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestAlertService_CreateAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("successful alert creation", func(t *testing.T) {
		userID := int64(123)
		alertType := models.AlertTemperature
		condition := AlertCondition{
			Operator: "gt",
			Value:    25.0,
		}

		// Mock database expectations - CREATE operation
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "alert_configs"`).
			WithArgs(userID, alertType, `{"operator":"gt","value":25}`, 25.0, true, nil, helpers.AnyTime{}, helpers.AnyTime{}).
			WillReturnRows(mockDB.Mock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mockDB.Mock.ExpectCommit()

		alertConfig, err := service.CreateAlert(context.Background(), userID, alertType, condition)

		assert.NoError(t, err)
		assert.NotNil(t, alertConfig)
		assert.Equal(t, userID, alertConfig.UserID)
		assert.Equal(t, alertType, alertConfig.AlertType)
		assert.Equal(t, condition.Value, alertConfig.Threshold)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error", func(t *testing.T) {
		userID := int64(123)
		alertType := models.AlertTemperature
		condition := AlertCondition{
			Operator: "gt",
			Value:    25.0,
		}

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "alert_configs"`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		alertConfig, err := service.CreateAlert(context.Background(), userID, alertType, condition)

		assert.Error(t, err)
		assert.Nil(t, alertConfig)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_GetAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("successful alert retrieval", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()
		expectedAlert := helpers.MockAlertConfig(userID)
		expectedAlert.ID = alertID

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "condition", "threshold", "is_active", "last_triggered", "created_at", "updated_at",
		}).AddRow(
			alertID, expectedAlert.UserID, expectedAlert.AlertType, expectedAlert.Condition,
			expectedAlert.Threshold, expectedAlert.IsActive, expectedAlert.LastTriggered,
			expectedAlert.CreatedAt, expectedAlert.UpdatedAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE id = \$1 AND user_id = \$2 ORDER BY "alert_configs"\."id" LIMIT 1`).
			WithArgs(alertID, userID).
			WillReturnRows(rows)

		alert, err := service.GetAlert(context.Background(), userID, alertID)

		assert.NoError(t, err)
		assert.NotNil(t, alert)
		assert.Equal(t, expectedAlert.UserID, alert.UserID)
		assert.Equal(t, expectedAlert.AlertType, alert.AlertType)
		assert.Equal(t, expectedAlert.Threshold, alert.Threshold)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE id = \$1 AND user_id = \$2 ORDER BY "alert_configs"\."id" LIMIT 1`).
			WithArgs(alertID, userID).
			WillReturnError(errors.New("record not found"))

		alert, err := service.GetAlert(context.Background(), userID, alertID)

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

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("successful user alerts retrieval", func(t *testing.T) {
		userID := int64(123)
		alert1 := helpers.MockAlertConfig(userID)
		alert2 := helpers.MockAlertConfig(userID)

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "condition", "threshold", "is_active", "last_triggered", "created_at", "updated_at",
		})

		rows.AddRow(
			alert1.ID, alert1.UserID, alert1.AlertType, alert1.Condition,
			alert1.Threshold, alert1.IsActive, alert1.LastTriggered,
			alert1.CreatedAt, alert1.UpdatedAt,
		)
		rows.AddRow(
			alert2.ID, alert2.UserID, alert2.AlertType, alert2.Condition,
			alert2.Threshold, alert2.IsActive, alert2.LastTriggered,
			alert2.CreatedAt, alert2.UpdatedAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
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

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(mockDB.Mock.NewRows([]string{
				"id", "user_id", "alert_type", "condition", "threshold", "is_active", "last_triggered", "created_at", "updated_at",
			}))

		result, err := service.GetUserAlerts(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_CheckAlerts(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	weatherData := helpers.MockWeatherData(123)

	t.Run("temperature alert triggered", func(t *testing.T) {
		userID := int64(123)
		alertConfig := helpers.MockAlertConfig(userID)

		// Weather data has temperature > threshold (25.0)
		weatherData.Temperature = 26.5

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "condition", "threshold", "is_active", "last_triggered", "created_at", "updated_at",
		}).AddRow(
			alertConfig.ID, alertConfig.UserID, alertConfig.AlertType, alertConfig.Condition,
			alertConfig.Threshold, alertConfig.IsActive, alertConfig.LastTriggered,
			alertConfig.CreatedAt, alertConfig.UpdatedAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		// Mock the INSERT for EnvironmentalAlert creation
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "environmental_alerts"`).
			WithArgs(userID, models.AlertTemperature, helpers.AnyValue{}, "Temperature Alert", "Temperature is 26.5°C", 26.5, 25.0, false, nil, helpers.AnyTime{}, helpers.AnyTime{}).
			WillReturnRows(mockDB.Mock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mockDB.Mock.ExpectCommit()

		// Mock the UPDATE for last_triggered time
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET "last_triggered"=\$1,"updated_at"=\$2 WHERE "id" = \$3`).
			WithArgs(helpers.AnyTime{}, helpers.AnyTime{}, alertConfig.ID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		triggeredAlerts, err := service.CheckAlerts(context.Background(), weatherData, userID)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, models.AlertTemperature, triggeredAlerts[0].AlertType)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no alerts triggered", func(t *testing.T) {
		userID := int64(123)
		alertConfig := helpers.MockAlertConfig(userID)
		alertConfig.Condition = `{"operator":"gt","value":30.0}` // Higher than weather data
		alertConfig.Threshold = 30.0

		// Weather data has temperature < threshold
		weatherData.Temperature = 20.5

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "alert_type", "condition", "threshold", "is_active", "last_triggered", "created_at", "updated_at",
		}).AddRow(
			alertConfig.ID, alertConfig.UserID, alertConfig.AlertType, alertConfig.Condition,
			alertConfig.Threshold, alertConfig.IsActive, alertConfig.LastTriggered,
			alertConfig.CreatedAt, alertConfig.UpdatedAt,
		)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "alert_configs" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		triggeredAlerts, err := service.CheckAlerts(context.Background(), weatherData, userID)

		assert.NoError(t, err)
		assert.Len(t, triggeredAlerts, 0)
		mockDB.ExpectationsWereMet(t)
	})

}

func TestAlertService_UpdateAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("successful alert update", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()
		updates := map[string]interface{}{
			"threshold": 30.0,
			"is_active": false,
		}

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET`).
			WithArgs(false, 30.0, helpers.AnyTime{}, alertID, userID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.UpdateAlert(context.Background(), userID, alertID, updates)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()
		updates := map[string]interface{}{
			"threshold": 30.0,
		}

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET`).
			WillReturnResult(helpers.NewResult(0, 0)) // No rows affected
		mockDB.Mock.ExpectCommit()

		err := service.UpdateAlert(context.Background(), userID, alertID, updates)

		assert.Error(t, err)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_DeleteAlert(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("successful alert deletion", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET "is_active"=\$1,"updated_at"=\$2 WHERE id = \$3 AND user_id = \$4`).
			WithArgs(false, helpers.AnyTime{}, alertID, userID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteAlert(context.Background(), userID, alertID)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("alert not found", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET "is_active"=\$1,"updated_at"=\$2 WHERE id = \$3 AND user_id = \$4`).
			WithArgs(false, helpers.AnyTime{}, alertID, userID).
			WillReturnResult(helpers.NewResult(0, 0))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteAlert(context.Background(), userID, alertID)

		assert.Error(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error during deletion", func(t *testing.T) {
		userID := int64(123)
		alertID := uuid.New()

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "alert_configs" SET "is_active"=\$1,"updated_at"=\$2 WHERE id = \$3 AND user_id = \$4`).
			WithArgs(false, helpers.AnyTime{}, alertID, userID).
			WillReturnError(errors.New("foreign key constraint"))
		mockDB.Mock.ExpectRollback()

		err := service.DeleteAlert(context.Background(), userID, alertID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "foreign key constraint")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestAlertService_EvaluateCondition(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	testCases := []struct {
		name         string
		currentValue float64
		condition    AlertCondition
		expected     bool
	}{
		{
			name:         "greater than - true",
			currentValue: 30.0,
			condition:    AlertCondition{Operator: "gt", Value: 25.0},
			expected:     true,
		},
		{
			name:         "greater than - false",
			currentValue: 20.0,
			condition:    AlertCondition{Operator: "gt", Value: 25.0},
			expected:     false,
		},
		{
			name:         "less than - true",
			currentValue: 20.0,
			condition:    AlertCondition{Operator: "lt", Value: 25.0},
			expected:     true,
		},
		{
			name:         "less than - false",
			currentValue: 30.0,
			condition:    AlertCondition{Operator: "lt", Value: 25.0},
			expected:     false,
		},
		{
			name:         "greater than or equal - equal",
			currentValue: 25.0,
			condition:    AlertCondition{Operator: "gte", Value: 25.0},
			expected:     true,
		},
		{
			name:         "greater than or equal - greater",
			currentValue: 30.0,
			condition:    AlertCondition{Operator: "gte", Value: 25.0},
			expected:     true,
		},
		{
			name:         "greater than or equal - less",
			currentValue: 20.0,
			condition:    AlertCondition{Operator: "gte", Value: 25.0},
			expected:     false,
		},
		{
			name:         "less than or equal - equal",
			currentValue: 25.0,
			condition:    AlertCondition{Operator: "lte", Value: 25.0},
			expected:     true,
		},
		{
			name:         "less than or equal - less",
			currentValue: 20.0,
			condition:    AlertCondition{Operator: "lte", Value: 25.0},
			expected:     true,
		},
		{
			name:         "less than or equal - greater",
			currentValue: 30.0,
			condition:    AlertCondition{Operator: "lte", Value: 25.0},
			expected:     false,
		},
		{
			name:         "equal - true",
			currentValue: 25.0,
			condition:    AlertCondition{Operator: "eq", Value: 25.0},
			expected:     true,
		},
		{
			name:         "equal - false",
			currentValue: 26.0,
			condition:    AlertCondition{Operator: "eq", Value: 25.0},
			expected:     false,
		},
		{
			name:         "unknown operator",
			currentValue: 25.0,
			condition:    AlertCondition{Operator: "unknown", Value: 25.0},
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.evaluateCondition(tc.currentValue, tc.condition)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAlertService_CalculateSeverity(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()
	mockRedis := helpers.NewMockRedis()
	service := NewAlertService(mockDB.DB, mockRedis.Client)

	t.Run("temperature alert severities", func(t *testing.T) {
		testCases := []struct {
			name         string
			currentValue float64
			threshold    float64
			expected     models.Severity
		}{
			{
				name:         "critical - deviation > 15",
				currentValue: 40.0,
				threshold:    20.0,
				expected:     models.SeverityCritical,
			},
			{
				name:         "high - deviation > 10",
				currentValue: 35.0,
				threshold:    20.0,
				expected:     models.SeverityHigh,
			},
			{
				name:         "medium - deviation > 5",
				currentValue: 27.0,
				threshold:    20.0,
				expected:     models.SeverityMedium,
			},
			{
				name:         "low - deviation <= 5",
				currentValue: 24.0,
				threshold:    20.0,
				expected:     models.SeverityLow,
			},
			{
				name:         "critical - negative deviation > 15",
				currentValue: 5.0,
				threshold:    25.0,
				expected:     models.SeverityCritical,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := service.calculateSeverity(models.AlertTemperature, tc.currentValue, tc.threshold)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("air quality alert severities", func(t *testing.T) {
		testCases := []struct {
			name         string
			currentValue float64
			expected     models.Severity
		}{
			{
				name:         "critical - AQI > 300",
				currentValue: 350.0,
				expected:     models.SeverityCritical,
			},
			{
				name:         "high - AQI > 200",
				currentValue: 250.0,
				expected:     models.SeverityHigh,
			},
			{
				name:         "medium - AQI > 150",
				currentValue: 180.0,
				expected:     models.SeverityMedium,
			},
			{
				name:         "low - AQI <= 150",
				currentValue: 100.0,
				expected:     models.SeverityLow,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := service.calculateSeverity(models.AlertAirQuality, tc.currentValue, 0)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("default alert type - medium severity", func(t *testing.T) {
		result := service.calculateSeverity(models.AlertHumidity, 80.0, 50.0)
		assert.Equal(t, models.SeverityMedium, result)

		result = service.calculateSeverity(models.AlertWindSpeed, 60.0, 30.0)
		assert.Equal(t, models.SeverityMedium, result)
	})
}
