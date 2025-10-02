package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
)

// DemoService handles demo mode data seeding and management
type DemoService struct {
	db     *gorm.DB
	logger *zerolog.Logger
}

// NewDemoService creates a new demo service instance
func NewDemoService(db *gorm.DB, logger *zerolog.Logger) *DemoService {
	return &DemoService{
		db:     db,
		logger: logger,
	}
}

// DemoUserID is the Telegram user ID for the demo account
const DemoUserID int64 = 999999999

// SeedDemoData populates the database with demonstration data
func (s *DemoService) SeedDemoData(ctx context.Context) error {
	s.logger.Info().Msg("Seeding demo data...")

	// Create demo user
	if err := s.createDemoUser(ctx); err != nil {
		return err
	}

	// Create demo weather data
	if err := s.createDemoWeatherData(ctx); err != nil {
		return err
	}

	// Create demo alerts
	if err := s.createDemoAlerts(ctx); err != nil {
		return err
	}

	// Create demo subscriptions
	if err := s.createDemoSubscriptions(ctx); err != nil {
		return err
	}

	s.logger.Info().Msg("Demo data seeded successfully")
	return nil
}

// createDemoUser creates a demonstration user account
func (s *DemoService) createDemoUser(ctx context.Context) error {
	user := &models.User{
		ID:           DemoUserID,
		Username:     "demo_user",
		FirstName:    "Demo",
		LastName:     "User",
		Language:     "en",
		LocationName: "Kyiv, Ukraine",
		Latitude:     50.4501,
		Longitude:    30.5234,
		Country:      "Ukraine",
		City:         "Kyiv",
		Timezone:     "Europe/Kyiv",
		Units:        "metric",
		Role:         models.RoleUser,
		IsActive:     true,
	}

	// Use FirstOrCreate to avoid duplicates
	result := s.db.WithContext(ctx).Where("id = ?", DemoUserID).FirstOrCreate(user)
	if result.Error != nil {
		s.logger.Error().Err(result.Error).Msg("Failed to create demo user")
		return result.Error
	}

	s.logger.Info().Int64("user_id", DemoUserID).Msg("Demo user created")
	return nil
}

// createDemoWeatherData generates sample weather records
func (s *DemoService) createDemoWeatherData(ctx context.Context) error {
	now := time.Now()
	baseTime := now.Add(-24 * time.Hour) // Start from 24 hours ago

	// Generate hourly weather data for the last 24 hours
	for i := 0; i < 24; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Hour)

		// Realistic temperature variation (sine wave pattern)
		baseTemp := 15.0
		variation := 8.0
		hour := float64(timestamp.Hour())
		temperature := baseTemp + variation*0.5*(1+0.8*((hour-6)/12))

		weatherData := &models.WeatherData{
			ID:          uuid.New(),
			UserID:      DemoUserID,
			Temperature: temperature,
			Humidity:    60 + i%20,
			Pressure:    1013 + float64(i%5),
			WindSpeed:   5.0 + float64(i%8),
			WindDegree:  180 + i*15,
			Visibility:  10.0,
			UVIndex:     float64(i % 10),
			Description: getWeatherDescription(int(temperature)),
			Icon:        getWeatherIcon(int(temperature)),
			Timestamp:   timestamp,
		}

		// Include AQI data for more recent entries
		if i >= 18 { // Last 6 hours
			weatherData.AQI = 50 + i%30
			weatherData.PM25 = 15.0 + float64(i%10)
			weatherData.PM10 = 25.0 + float64(i%15)
			weatherData.CO = 200.0 + float64(i%50)
			weatherData.NO2 = 10.0 + float64(i%5)
			weatherData.O3 = 30.0 + float64(i%20)
		}

		if err := s.db.WithContext(ctx).Create(weatherData).Error; err != nil {
			s.logger.Error().Err(err).Msg("Failed to create demo weather data")
			return err
		}
	}

	s.logger.Info().Int("count", 24).Msg("Demo weather data created")
	return nil
}

// createDemoAlerts generates sample alert configurations
func (s *DemoService) createDemoAlerts(ctx context.Context) error {
	alerts := []models.AlertConfig{
		{
			ID:        uuid.New(),
			UserID:    DemoUserID,
			AlertType: models.AlertTemperature,
			Condition: "greater_than",
			Threshold: 30.0,
			IsActive:  true,
		},
		{
			ID:        uuid.New(),
			UserID:    DemoUserID,
			AlertType: models.AlertHumidity,
			Condition: "greater_than",
			Threshold: 80.0,
			IsActive:  true,
		},
		{
			ID:        uuid.New(),
			UserID:    DemoUserID,
			AlertType: models.AlertAirQuality,
			Condition: "greater_than",
			Threshold: 100.0,
			IsActive:  false, // One inactive to show variation
		},
	}

	for _, alert := range alerts {
		if err := s.db.WithContext(ctx).Create(&alert).Error; err != nil {
			s.logger.Error().Err(err).Msg("Failed to create demo alert")
			return err
		}
	}

	s.logger.Info().Int("count", len(alerts)).Msg("Demo alerts created")
	return nil
}

// createDemoSubscriptions generates sample notification subscriptions
func (s *DemoService) createDemoSubscriptions(ctx context.Context) error {
	subscriptions := []models.Subscription{
		{
			ID:               uuid.New(),
			UserID:           DemoUserID,
			SubscriptionType: models.SubscriptionDaily,
			Frequency:        models.FrequencyDaily,
			IsActive:         true,
			TimeOfDay:        "08:00",
		},
		{
			ID:               uuid.New(),
			UserID:           DemoUserID,
			SubscriptionType: models.SubscriptionWeekly,
			Frequency:        models.FrequencyWeekly,
			IsActive:         true,
			TimeOfDay:        "09:00",
		},
		{
			ID:               uuid.New(),
			UserID:           DemoUserID,
			SubscriptionType: models.SubscriptionAlerts,
			Frequency:        models.FrequencyHourly,
			IsActive:         true,
		},
	}

	for _, sub := range subscriptions {
		if err := s.db.WithContext(ctx).Create(&sub).Error; err != nil {
			s.logger.Error().Err(err).Msg("Failed to create demo subscription")
			return err
		}
	}

	s.logger.Info().Int("count", len(subscriptions)).Msg("Demo subscriptions created")
	return nil
}

// ClearDemoData removes all demonstration data from the database
func (s *DemoService) ClearDemoData(ctx context.Context) error {
	s.logger.Info().Msg("Clearing demo data...")

	// Delete in reverse order of dependencies
	tables := []interface{}{
		&models.Subscription{},
		&models.AlertConfig{},
		&models.WeatherData{},
		&models.User{},
	}

	for _, table := range tables {
		if err := s.db.WithContext(ctx).Where("user_id = ?", DemoUserID).Delete(table).Error; err != nil {
			s.logger.Error().Err(err).Msgf("Failed to clear demo data for %T", table)
			return err
		}
	}

	s.logger.Info().Msg("Demo data cleared successfully")
	return nil
}

// ResetDemoData clears and re-seeds demonstration data
func (s *DemoService) ResetDemoData(ctx context.Context) error {
	if err := s.ClearDemoData(ctx); err != nil {
		return err
	}
	return s.SeedDemoData(ctx)
}

// IsDemoUser checks if a user ID belongs to the demo account
func (s *DemoService) IsDemoUser(userID int64) bool {
	return userID == DemoUserID
}

// Helper functions for weather data generation

func getWeatherDescription(temp int) string {
	switch {
	case temp <= 0:
		return "Freezing"
	case temp < 10:
		return "Cold"
	case temp < 20:
		return "Cool"
	case temp < 25:
		return "Mild"
	case temp < 30:
		return "Warm"
	default:
		return "Hot"
	}
}

func getWeatherIcon(temp int) string {
	switch {
	case temp <= 0:
		return "❄️"
	case temp < 10:
		return "🌧️"
	case temp < 20:
		return "⛅"
	case temp < 30:
		return "☀️"
	default:
		return "🔥"
	}
}
