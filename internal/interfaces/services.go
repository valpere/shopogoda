package interfaces

import (
	"context"
	"time"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/weather"
)

//go:generate mockgen -source=services.go -destination=../tests/mocks/services_mock.go -package=mocks

// WeatherServiceInterface defines the interface for weather service operations
type WeatherServiceInterface interface {
	GeocodeLocation(ctx context.Context, location string) (*weather.Location, error)
	GetCurrentWeather(ctx context.Context, lat, lon float64) (*services.WeatherData, error)
	GetForecast(ctx context.Context, lat, lon float64) ([]services.WeatherData, error)
	GetAirQuality(ctx context.Context, lat, lon float64) (*weather.AirQualityData, error)
	SaveWeatherData(ctx context.Context, data *models.WeatherData) error
}

// UserServiceInterface defines the interface for user service operations
type UserServiceInterface interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, userID int64) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, userID int64) error
	SetUserLocation(ctx context.Context, userID int64, locationName, country, city string, lat, lon float64) error
	ClearUserLocation(ctx context.Context, userID int64) error
	GetUserLocation(ctx context.Context, userID int64) (string, float64, float64, error)
	SetUserLanguage(ctx context.Context, userID int64, language string) error
	GetUserLanguage(ctx context.Context, userID int64) string
	SetUserTimezone(ctx context.Context, userID int64, timezone string) error
	GetUserTimezone(ctx context.Context, userID int64) string
	ConvertToUserTime(ctx context.Context, userID int64, utcTime time.Time) time.Time
	ConvertToUTC(ctx context.Context, userID int64, localTime time.Time) time.Time
	GetActiveUsersWithLocations(ctx context.Context) ([]*models.User, error)
}

// AlertServiceInterface defines the interface for alert service operations
type AlertServiceInterface interface {
	CreateAlert(ctx context.Context, alert *models.EnvironmentalAlert) error
	GetAlert(ctx context.Context, alertID string) (*models.EnvironmentalAlert, error)
	GetUserAlerts(ctx context.Context, userID int64) ([]*models.EnvironmentalAlert, error)
	UpdateAlert(ctx context.Context, alert *models.EnvironmentalAlert) error
	DeleteAlert(ctx context.Context, alertID string) error
	CheckAlertsForUser(ctx context.Context, userID int64, weatherData *services.WeatherData) ([]*models.EnvironmentalAlert, error)
	TriggerAlert(ctx context.Context, alert *models.EnvironmentalAlert, value float64) error
}

// NotificationServiceInterface defines the interface for notification service operations
type NotificationServiceInterface interface {
	SendSlackAlert(alert *models.EnvironmentalAlert, user *models.User) error
	SendSlackWeatherUpdate(weatherData *services.WeatherData, subscribers []models.User) error
	SendTelegramAlert(alert *models.EnvironmentalAlert, user *models.User) error
	SendTelegramWeatherUpdate(weatherData *services.WeatherData, user *models.User) error
	SendTelegramWeeklyUpdate(weatherData *services.WeatherData, user *models.User) error
}

// SubscriptionServiceInterface defines the interface for subscription service operations
type SubscriptionServiceInterface interface {
	CreateSubscription(ctx context.Context, subscription *models.Subscription) error
	GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error)
	GetUserSubscriptions(ctx context.Context, userID int64) ([]*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) error
	DeleteSubscription(ctx context.Context, subscriptionID string) error
	GetActiveSubscriptionsByType(ctx context.Context, subscriptionType models.SubscriptionType) ([]*models.Subscription, error)
	GetActiveSubscriptionsByTime(ctx context.Context, hour int) ([]*models.Subscription, error)
}

// LocalizationServiceInterface defines the interface for localization service operations
type LocalizationServiceInterface interface {
	T(ctx context.Context, language, key string, args ...interface{}) string
	GetSupportedLanguages() []string
	IsLanguageSupported(language string) bool
}

// ExportServiceInterface defines the interface for export service operations
type ExportServiceInterface interface {
	ExportUserData(ctx context.Context, userID int64, format services.ExportFormat, dataType services.ExportType) ([]byte, error)
	ExportWeatherData(ctx context.Context, userID int64, format services.ExportFormat) ([]byte, error)
	ExportAlerts(ctx context.Context, userID int64, format services.ExportFormat) ([]byte, error)
	ExportSubscriptions(ctx context.Context, userID int64, format services.ExportFormat) ([]byte, error)
}

// SchedulerServiceInterface defines the interface for scheduler service operations
type SchedulerServiceInterface interface {
	Start(ctx context.Context) error
	Stop() error
	ProcessWeatherAlerts(ctx context.Context) error
	ProcessScheduledNotifications(ctx context.Context) error
}
