package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
)

type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatTXT  ExportFormat = "txt"
)

type ExportType string

const (
	ExportTypeWeatherData    ExportType = "weather"
	ExportTypeAlerts         ExportType = "alerts"
	ExportTypeSubscriptions  ExportType = "subscriptions"
	ExportTypeAll            ExportType = "all"
)

type ExportService struct {
	db     *gorm.DB
	logger *zerolog.Logger
}

type ExportData struct {
	User            *models.User                     `json:"user,omitempty"`
	WeatherData     []models.WeatherData             `json:"weather_data,omitempty"`
	Subscriptions   []models.Subscription            `json:"subscriptions,omitempty"`
	AlertConfigs    []models.AlertConfig             `json:"alert_configs,omitempty"`
	TriggeredAlerts []models.EnvironmentalAlert      `json:"triggered_alerts,omitempty"`
	ExportedAt      time.Time                        `json:"exported_at"`
	Format          ExportFormat                     `json:"format"`
	Type            ExportType                       `json:"type"`
}

func NewExportService(db *gorm.DB, logger *zerolog.Logger) *ExportService {
	return &ExportService{
		db:     db,
		logger: logger,
	}
}

// ExportUserData exports user's data in the specified format
func (s *ExportService) ExportUserData(ctx context.Context, userID int64, exportType ExportType, format ExportFormat) (*bytes.Buffer, string, error) {
	s.logger.Info().
		Int64("user_id", userID).
		Str("type", string(exportType)).
		Str("format", string(format)).
		Msg("Starting data export")

	// Get user data
	user, err := s.getUserData(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user data: %w", err)
	}

	exportData := &ExportData{
		User:       user,
		ExportedAt: time.Now().UTC(),
		Format:     format,
		Type:       exportType,
	}

	// Get specific data based on export type
	switch exportType {
	case ExportTypeWeatherData:
		exportData.WeatherData, err = s.getWeatherData(ctx, userID)
	case ExportTypeAlerts:
		exportData.AlertConfigs, err = s.getAlertConfigs(ctx, userID)
		exportData.TriggeredAlerts, err = s.getTriggeredAlerts(ctx, userID)
	case ExportTypeSubscriptions:
		exportData.Subscriptions, err = s.getSubscriptions(ctx, userID)
	case ExportTypeAll:
		exportData.WeatherData, _ = s.getWeatherData(ctx, userID)
		exportData.AlertConfigs, _ = s.getAlertConfigs(ctx, userID)
		exportData.TriggeredAlerts, _ = s.getTriggeredAlerts(ctx, userID)
		exportData.Subscriptions, _ = s.getSubscriptions(ctx, userID)
	default:
		return nil, "", fmt.Errorf("unsupported export type: %s", exportType)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to get %s data: %w", exportType, err)
	}

	// Generate export based on format
	var buffer *bytes.Buffer
	var filename string

	switch format {
	case ExportFormatJSON:
		buffer, filename, err = s.exportToJSON(exportData)
	case ExportFormatCSV:
		buffer, filename, err = s.exportToCSV(exportData)
	case ExportFormatTXT:
		buffer, filename, err = s.exportToTXT(exportData)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to generate %s export: %w", format, err)
	}

	s.logger.Info().
		Int64("user_id", userID).
		Str("filename", filename).
		Int("size_bytes", buffer.Len()).
		Msg("Data export completed")

	return buffer, filename, nil
}

func (s *ExportService) getUserData(ctx context.Context, userID int64) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *ExportService) getWeatherData(ctx context.Context, userID int64) ([]models.WeatherData, error) {
	var weatherData []models.WeatherData
	// Get last 30 days of weather data
	thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30)

	err := s.db.WithContext(ctx).
		Where("user_id = ? AND timestamp >= ?", userID, thirtyDaysAgo).
		Order("timestamp DESC").
		Limit(1000). // Reasonable limit
		Find(&weatherData).Error

	return weatherData, err
}

func (s *ExportService) getAlertConfigs(ctx context.Context, userID int64) ([]models.AlertConfig, error) {
	var alertConfigs []models.AlertConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&alertConfigs).Error
	return alertConfigs, err
}

func (s *ExportService) getTriggeredAlerts(ctx context.Context, userID int64) ([]models.EnvironmentalAlert, error) {
	var triggeredAlerts []models.EnvironmentalAlert
	// Get last 90 days of triggered alerts
	ninetyDaysAgo := time.Now().UTC().AddDate(0, 0, -90)

	err := s.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, ninetyDaysAgo).
		Order("created_at DESC").
		Find(&triggeredAlerts).Error

	return triggeredAlerts, err
}

func (s *ExportService) getSubscriptions(ctx context.Context, userID int64) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&subscriptions).Error
	return subscriptions, err
}

func (s *ExportService) exportToJSON(data *ExportData) (*bytes.Buffer, string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, "", err
	}

	buffer := bytes.NewBuffer(jsonData)
	filename := fmt.Sprintf("shopogoda_%s_%s_%s.json",
		data.Type,
		data.User.Username,
		data.ExportedAt.Format("2006-01-02"))

	return buffer, filename, nil
}

func (s *ExportService) exportToCSV(data *ExportData) (*bytes.Buffer, string, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write user info header
	writer.Write([]string{"Export Type", string(data.Type)})
	writer.Write([]string{"User ID", strconv.FormatInt(data.User.ID, 10)})
	writer.Write([]string{"Username", data.User.Username})
	writer.Write([]string{"Exported At", data.ExportedAt.Format(time.RFC3339)})
	writer.Write([]string{}) // Empty line

	// Export weather data if present
	if len(data.WeatherData) > 0 {
		writer.Write([]string{"Weather Data"})
		writer.Write([]string{"Timestamp", "Temperature", "Humidity", "Pressure", "Wind Speed", "Wind Degree", "Visibility", "UV Index", "Description", "AQI"})

		for _, weather := range data.WeatherData {
			writer.Write([]string{
				weather.Timestamp.Format(time.RFC3339),
				fmt.Sprintf("%.1f", weather.Temperature),
				strconv.Itoa(weather.Humidity),
				fmt.Sprintf("%.1f", weather.Pressure),
				fmt.Sprintf("%.1f", weather.WindSpeed),
				strconv.Itoa(weather.WindDegree),
				fmt.Sprintf("%.1f", weather.Visibility),
				fmt.Sprintf("%.1f", weather.UVIndex),
				weather.Description,
				strconv.Itoa(weather.AQI),
			})
		}
		writer.Write([]string{}) // Empty line
	}

	// Export subscriptions if present
	if len(data.Subscriptions) > 0 {
		writer.Write([]string{"Subscriptions"})
		writer.Write([]string{"Type", "Frequency", "Time Of Day", "Is Active", "Created At"})

		for _, sub := range data.Subscriptions {
			writer.Write([]string{
				sub.SubscriptionType.String(),
				sub.Frequency.String(),
				sub.TimeOfDay,
				strconv.FormatBool(sub.IsActive),
				sub.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Write([]string{}) // Empty line
	}

	// Export alert configs if present
	if len(data.AlertConfigs) > 0 {
		writer.Write([]string{"Alert Configurations"})
		writer.Write([]string{"Alert Type", "Condition", "Threshold", "Is Active", "Created At"})

		for _, alert := range data.AlertConfigs {
			writer.Write([]string{
				alert.AlertType.String(),
				alert.Condition,
				fmt.Sprintf("%.1f", alert.Threshold),
				strconv.FormatBool(alert.IsActive),
				alert.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Write([]string{}) // Empty line
	}

	// Export triggered alerts if present
	if len(data.TriggeredAlerts) > 0 {
		writer.Write([]string{"Triggered Alerts"})
		writer.Write([]string{"Alert Type", "Severity", "Title", "Value", "Threshold", "Is Resolved", "Created At"})

		for _, alert := range data.TriggeredAlerts {
			writer.Write([]string{
				alert.AlertType.String(),
				alert.Severity.String(),
				alert.Title,
				fmt.Sprintf("%.1f", alert.Value),
				fmt.Sprintf("%.1f", alert.Threshold),
				strconv.FormatBool(alert.IsResolved),
				alert.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("shopogoda_%s_%s_%s.csv",
		data.Type,
		data.User.Username,
		data.ExportedAt.Format("2006-01-02"))

	return &buffer, filename, nil
}

func (s *ExportService) exportToTXT(data *ExportData) (*bytes.Buffer, string, error) {
	var buffer bytes.Buffer

	// Header
	buffer.WriteString("ShoPogoda Data Export\n")
	buffer.WriteString("=====================\n\n")
	buffer.WriteString(fmt.Sprintf("Export Type: %s\n", data.Type))
	buffer.WriteString(fmt.Sprintf("User: %s (ID: %d)\n", data.User.Username, data.User.ID))
	buffer.WriteString(fmt.Sprintf("Exported At: %s\n\n", data.ExportedAt.Format(time.RFC3339)))

	// User information
	buffer.WriteString("User Information:\n")
	buffer.WriteString("-----------------\n")
	buffer.WriteString(fmt.Sprintf("Name: %s %s\n", data.User.FirstName, data.User.LastName))
	buffer.WriteString(fmt.Sprintf("Language: %s\n", data.User.Language))
	buffer.WriteString(fmt.Sprintf("Units: %s\n", data.User.Units))
	buffer.WriteString(fmt.Sprintf("Timezone: %s\n", data.User.Timezone))
	if data.User.LocationName != "" {
		buffer.WriteString(fmt.Sprintf("Location: %s (%s, %s)\n", data.User.LocationName, data.User.City, data.User.Country))
		buffer.WriteString(fmt.Sprintf("Coordinates: %.4f, %.4f\n", data.User.Latitude, data.User.Longitude))
	}
	buffer.WriteString("\n")

	// Weather data
	if len(data.WeatherData) > 0 {
		buffer.WriteString(fmt.Sprintf("Weather Data (%d records):\n", len(data.WeatherData)))
		buffer.WriteString("----------------------------\n")
		for _, weather := range data.WeatherData {
			buffer.WriteString(fmt.Sprintf("Date: %s\n", weather.Timestamp.Format("2006-01-02 15:04:05 UTC")))
			buffer.WriteString(fmt.Sprintf("  Temperature: %.1f°C, Humidity: %d%%\n", weather.Temperature, weather.Humidity))
			buffer.WriteString(fmt.Sprintf("  Pressure: %.1fhPa, Wind: %.1fkm/h at %d°\n", weather.Pressure, weather.WindSpeed, weather.WindDegree))
			buffer.WriteString(fmt.Sprintf("  Visibility: %.1fkm, UV Index: %.1f\n", weather.Visibility, weather.UVIndex))
			buffer.WriteString(fmt.Sprintf("  Conditions: %s, AQI: %d\n\n", weather.Description, weather.AQI))
		}
		buffer.WriteString("\n")
	}

	// Subscriptions
	if len(data.Subscriptions) > 0 {
		buffer.WriteString(fmt.Sprintf("Subscriptions (%d active):\n", len(data.Subscriptions)))
		buffer.WriteString("---------------------------\n")
		for _, sub := range data.Subscriptions {
			buffer.WriteString(fmt.Sprintf("Type: %s\n", sub.SubscriptionType.String()))
			buffer.WriteString(fmt.Sprintf("  Frequency: %s\n", sub.Frequency.String()))
			buffer.WriteString(fmt.Sprintf("  Time: %s\n", sub.TimeOfDay))
			buffer.WriteString(fmt.Sprintf("  Active: %t\n", sub.IsActive))
			buffer.WriteString(fmt.Sprintf("  Created: %s\n\n", sub.CreatedAt.Format("2006-01-02 15:04:05 UTC")))
		}
		buffer.WriteString("\n")
	}

	// Alert configs
	if len(data.AlertConfigs) > 0 {
		buffer.WriteString(fmt.Sprintf("Alert Configurations (%d configured):\n", len(data.AlertConfigs)))
		buffer.WriteString("---------------------------------------\n")
		for _, alert := range data.AlertConfigs {
			buffer.WriteString(fmt.Sprintf("Type: %s\n", alert.AlertType.String()))
			buffer.WriteString(fmt.Sprintf("  Condition: %s\n", alert.Condition))
			buffer.WriteString(fmt.Sprintf("  Threshold: %.1f\n", alert.Threshold))
			buffer.WriteString(fmt.Sprintf("  Active: %t\n", alert.IsActive))
			buffer.WriteString(fmt.Sprintf("  Created: %s\n\n", alert.CreatedAt.Format("2006-01-02 15:04:05 UTC")))
		}
		buffer.WriteString("\n")
	}

	// Triggered alerts
	if len(data.TriggeredAlerts) > 0 {
		buffer.WriteString(fmt.Sprintf("Triggered Alerts (%d alerts):\n", len(data.TriggeredAlerts)))
		buffer.WriteString("-------------------------------\n")
		for _, alert := range data.TriggeredAlerts {
			buffer.WriteString(fmt.Sprintf("Alert: %s\n", alert.Title))
			buffer.WriteString(fmt.Sprintf("  Type: %s, Severity: %s\n", alert.AlertType.String(), alert.Severity.String()))
			buffer.WriteString(fmt.Sprintf("  Value: %.1f (Threshold: %.1f)\n", alert.Value, alert.Threshold))
			buffer.WriteString(fmt.Sprintf("  Description: %s\n", alert.Description))
			buffer.WriteString(fmt.Sprintf("  Resolved: %t\n", alert.IsResolved))
			buffer.WriteString(fmt.Sprintf("  Triggered: %s\n\n", alert.CreatedAt.Format("2006-01-02 15:04:05 UTC")))
		}
	}

	buffer.WriteString("End of Export\n")

	filename := fmt.Sprintf("shopogoda_%s_%s_%s.txt",
		data.Type,
		data.User.Username,
		data.ExportedAt.Format("2006-01-02"))

	return &buffer, filename, nil
}