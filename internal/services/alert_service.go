package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
)

type AlertService struct {
	db    *gorm.DB
	redis *redis.Client
}

type AlertCondition struct {
	Operator string  `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
	Value    float64 `json:"value"`
}

func NewAlertService(db *gorm.DB, redis *redis.Client) *AlertService {
	return &AlertService{
		db:    db,
		redis: redis,
	}
}

func (s *AlertService) CreateAlert(ctx context.Context, userID int64, alertType models.AlertType, condition AlertCondition) (*models.AlertConfig, error) {
	conditionJSON, _ := json.Marshal(condition)

	alert := &models.AlertConfig{
		UserID:    userID,
		AlertType: alertType,
		Condition: string(conditionJSON),
		Threshold: condition.Value,
		IsActive:  true,
	}

	if err := s.db.WithContext(ctx).Create(alert).Error; err != nil {
		return nil, err
	}

	return alert, nil
}

func (s *AlertService) GetUserAlerts(ctx context.Context, userID int64) ([]models.AlertConfig, error) {
	var alerts []models.AlertConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Find(&alerts).Error

	return alerts, err
}

func (s *AlertService) CheckAlerts(ctx context.Context, weatherData *models.WeatherData, userID int64) ([]models.EnvironmentalAlert, error) {
	// Get active alerts for this user
	var alertConfigs []models.AlertConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Find(&alertConfigs).Error

	if err != nil {
		return nil, err
	}

	var triggeredAlerts []models.EnvironmentalAlert

	for _, config := range alertConfigs {
		var condition AlertCondition
		json.Unmarshal([]byte(config.Condition), &condition)

		var currentValue float64
		var alertTitle, alertDescription string

		// Get current value based on alert type
		switch config.AlertType {
		case models.AlertTemperature:
			currentValue = weatherData.Temperature
			alertTitle = "Temperature Alert"
			alertDescription = fmt.Sprintf("Temperature is %.1f°C", currentValue)
		case models.AlertHumidity:
			currentValue = float64(weatherData.Humidity)
			alertTitle = "Humidity Alert"
			alertDescription = fmt.Sprintf("Humidity is %d%%", weatherData.Humidity)
		case models.AlertWindSpeed:
			currentValue = weatherData.WindSpeed
			alertTitle = "Wind Speed Alert"
			alertDescription = fmt.Sprintf("Wind speed is %.1f km/h", currentValue)
		case models.AlertAirQuality:
			currentValue = float64(weatherData.AQI)
			alertTitle = "Air Quality Alert"
			alertDescription = fmt.Sprintf("AQI is %d", weatherData.AQI)
		default:
			continue
		}

		// Check if condition is met
		if s.evaluateCondition(currentValue, condition) {
			// Check if alert was recently triggered (avoid spam)
			if config.LastTriggered != nil && time.Since(*config.LastTriggered) < time.Hour {
				continue
			}

			severity := s.calculateSeverity(config.AlertType, currentValue, condition.Value)

			alert := models.EnvironmentalAlert{
				UserID:      userID,
				AlertType:   config.AlertType,
				Severity:    severity,
				Title:       alertTitle,
				Description: alertDescription,
				Value:       currentValue,
				Threshold:   condition.Value,
				IsResolved:  false,
			}

			// Save alert
			if err := s.db.WithContext(ctx).Create(&alert).Error; err == nil {
				triggeredAlerts = append(triggeredAlerts, alert)

				// Update last triggered time
				now := time.Now().UTC()
				s.db.WithContext(ctx).Model(&config).Update("last_triggered", &now)
			}
		}
	}

	return triggeredAlerts, nil
}

func (s *AlertService) evaluateCondition(currentValue float64, condition AlertCondition) bool {
	switch condition.Operator {
	case "gt":
		return currentValue > condition.Value
	case "lt":
		return currentValue < condition.Value
	case "gte":
		return currentValue >= condition.Value
	case "lte":
		return currentValue <= condition.Value
	case "eq":
		return currentValue == condition.Value
	default:
		return false
	}
}

func (s *AlertService) calculateSeverity(alertType models.AlertType, currentValue, threshold float64) models.Severity {
	var deviation float64
	if currentValue > threshold {
		deviation = currentValue - threshold
	} else {
		deviation = threshold - currentValue
	}

	switch alertType {
	case models.AlertTemperature:
		if deviation > 15 {
			return models.SeverityCritical
		} else if deviation > 10 {
			return models.SeverityHigh
		} else if deviation > 5 {
			return models.SeverityMedium
		}
		return models.SeverityLow

	case models.AlertAirQuality:
		if currentValue > 300 {
			return models.SeverityCritical
		} else if currentValue > 200 {
			return models.SeverityHigh
		} else if currentValue > 150 {
			return models.SeverityMedium
		}
		return models.SeverityLow

	default:
		return models.SeverityMedium
	}
}

// DeleteAlert marks an alert as inactive
func (s *AlertService) DeleteAlert(ctx context.Context, userID int64, alertID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Model(&models.AlertConfig{}).
		Where("id = ? AND user_id = ?", alertID, userID).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}

// UpdateAlert updates an existing alert configuration
func (s *AlertService) UpdateAlert(ctx context.Context, userID int64, alertID uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UTC()

	result := s.db.WithContext(ctx).
		Model(&models.AlertConfig{}).
		Where("id = ? AND user_id = ?", alertID, userID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}

// GetAlert retrieves a specific alert by ID and user ID
func (s *AlertService) GetAlert(ctx context.Context, userID int64, alertID uuid.UUID) (*models.AlertConfig, error) {
	var alert models.AlertConfig
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", alertID, userID).
		First(&alert).Error

	if err != nil {
		return nil, err
	}
	return &alert, nil
}
