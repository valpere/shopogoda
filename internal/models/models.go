package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a Telegram user
type User struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"index" json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Language  string    `gorm:"default:'en'" json:"language"`
	Role      UserRole  `gorm:"default:1" json:"role"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Locations     []Location     `json:"locations,omitempty"`
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
	AlertConfigs  []AlertConfig  `json:"alert_configs,omitempty"`
}

type UserRole int

const (
	RoleUser UserRole = iota + 1
	RoleModerator
	RoleAdmin
)

// Location represents a monitored location
type Location struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      int64     `gorm:"index" json:"user_id"`
	Name        string    `gorm:"not null" json:"name"`
	Latitude    float64   `gorm:"not null" json:"latitude"`
	Longitude   float64   `gorm:"not null" json:"longitude"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	IsDefault   bool      `gorm:"default:false" json:"is_default"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	User                User                 `json:"user,omitempty"`
	WeatherData         []WeatherData        `json:"weather_data,omitempty"`
	Subscriptions       []Subscription       `json:"subscriptions,omitempty"`
	AlertConfigs        []AlertConfig        `json:"alert_configs,omitempty"`
	EnvironmentalAlerts []EnvironmentalAlert `json:"environmental_alerts,omitempty"`
}

// WeatherData stores weather information
type WeatherData struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LocationID   uuid.UUID `gorm:"index" json:"location_id"`
	Temperature  float64   `json:"temperature"`
	Humidity     int       `json:"humidity"`
	Pressure     float64   `json:"pressure"`
	WindSpeed    float64   `json:"wind_speed"`
	WindDegree   int       `json:"wind_degree"`
	Visibility   float64   `json:"visibility"`
	UVIndex      float64   `json:"uv_index"`
	Description  string    `json:"description"`
	Icon         string    `json:"icon"`
	AQI          int       `json:"aqi"` // Air Quality Index
	CO           float64   `json:"co"`  // Carbon Monoxide
	NO2          float64   `json:"no2"` // Nitrogen Dioxide
	O3           float64   `json:"o3"`  // Ozone
	PM25         float64   `json:"pm25"` // PM2.5
	PM10         float64   `json:"pm10"` // PM10
	Timestamp    time.Time `gorm:"index" json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`

	// Relationships
	Location Location `json:"location,omitempty"`
}

// Subscription represents user notification preferences
type Subscription struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID           int64           `gorm:"index" json:"user_id"`
	LocationID       uuid.UUID       `gorm:"index" json:"location_id"`
	SubscriptionType SubscriptionType `json:"subscription_type"`
	Frequency        Frequency        `json:"frequency"`
	TimeOfDay        string          `json:"time_of_day"` // HH:MM format
	IsActive         bool            `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`

	// Relationships
	User     User     `json:"user,omitempty"`
	Location Location `json:"location,omitempty"`
}

type SubscriptionType int

const (
	SubscriptionDaily SubscriptionType = iota + 1
	SubscriptionWeekly
	SubscriptionAlerts
	SubscriptionExtreme
)

type Frequency int

const (
	FrequencyHourly Frequency = iota + 1
	FrequencyEvery3Hours
	FrequencyEvery6Hours
	FrequencyDaily
	FrequencyWeekly
)

// AlertConfig represents alert configuration
type AlertConfig struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      int64     `gorm:"index" json:"user_id"`
	LocationID  uuid.UUID `gorm:"index" json:"location_id"`
	AlertType   AlertType `json:"alert_type"`
	Condition   string    `json:"condition"` // JSON condition
	Threshold   float64   `json:"threshold"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	User     User     `json:"user,omitempty"`
	Location Location `json:"location,omitempty"`
}

type AlertType int

const (
	AlertTemperature AlertType = iota + 1
	AlertHumidity
	AlertPressure
	AlertWindSpeed
	AlertUVIndex
	AlertAirQuality
	AlertRain
	AlertSnow
	AlertStorm
)

// EnvironmentalAlert represents triggered alerts
type EnvironmentalAlert struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	LocationID  uuid.UUID `gorm:"index" json:"location_id"`
	AlertType   AlertType `json:"alert_type"`
	Severity    Severity  `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	IsResolved  bool      `gorm:"default:false" json:"is_resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Location Location `json:"location,omitempty"`
}

type Severity int

const (
	SeverityLow Severity = iota + 1
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// UserSession stores user session data
type UserSession struct {
	UserID      int64     `gorm:"primaryKey" json:"user_id"`
	State       string    `json:"state"`
	Data        string    `json:"data"` // JSON data
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Location{},
		&WeatherData{},
		&Subscription{},
		&AlertConfig{},
		&EnvironmentalAlert{},
		&UserSession{},
	)
}
