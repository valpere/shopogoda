package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a Telegram user
type User struct {
	ID        int64    `gorm:"primaryKey" json:"id"`
	Username  string   `gorm:"index" json:"username"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Language  string   `gorm:"default:'en'" json:"language"`
	Units     string   `gorm:"default:'metric'" json:"units"`
	Timezone  string   `gorm:"default:'UTC'" json:"timezone"`
	Role      UserRole `gorm:"default:1" json:"role"`
	IsActive  bool     `gorm:"default:true" json:"is_active"`

	// User's single location
	LocationName string  `json:"location_name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Country      string  `json:"country"`
	City         string  `json:"city"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
	AlertConfigs  []AlertConfig  `json:"alert_configs,omitempty"`
}

type UserRole int

const (
	RoleUser UserRole = iota + 1
	RoleModerator
	RoleAdmin
)

// WeatherData stores weather information (timezone-aware, stored in UTC)
type WeatherData struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      int64     `gorm:"index" json:"user_id"`
	Temperature float64   `json:"temperature"`
	Humidity    int       `json:"humidity"`
	Pressure    float64   `json:"pressure"`
	WindSpeed   float64   `json:"wind_speed"`
	WindDegree  int       `json:"wind_degree"`
	Visibility  float64   `json:"visibility"`
	UVIndex     float64   `json:"uv_index"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	AQI         int       `json:"aqi"`                    // Air Quality Index
	CO          float64   `json:"co"`                     // Carbon Monoxide
	NO2         float64   `json:"no2"`                    // Nitrogen Dioxide
	O3          float64   `json:"o3"`                     // Ozone
	PM25        float64   `json:"pm25"`                   // PM2.5
	PM10        float64   `json:"pm10"`                   // PM10
	Timestamp   time.Time `gorm:"index" json:"timestamp"` // Always UTC
	CreatedAt   time.Time `json:"created_at"`

	// Relationships
	User User `json:"user,omitempty"`
}

// Subscription represents user notification preferences
type Subscription struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID           int64            `gorm:"index" json:"user_id"`
	SubscriptionType SubscriptionType `json:"subscription_type"`
	Frequency        Frequency        `json:"frequency"`
	TimeOfDay        string           `json:"time_of_day"` // HH:MM format in user timezone
	IsActive         bool             `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`

	// Relationships
	User User `json:"user,omitempty"`
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
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID        int64      `gorm:"index" json:"user_id"`
	AlertType     AlertType  `json:"alert_type"`
	Condition     string     `json:"condition"` // JSON condition
	Threshold     float64    `json:"threshold"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"` // UTC
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relationships
	User User `json:"user,omitempty"`
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
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      int64      `gorm:"index" json:"user_id"`
	AlertType   AlertType  `json:"alert_type"`
	Severity    Severity   `json:"severity"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Value       float64    `json:"value"`
	Threshold   float64    `json:"threshold"`
	IsResolved  bool       `gorm:"default:false" json:"is_resolved"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`   // UTC
	CreatedAt   time.Time  `gorm:"index" json:"created_at"` // UTC
	UpdatedAt   time.Time  `json:"updated_at"`              // UTC

	// Relationships
	User User `json:"user,omitempty"`
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
	UserID    int64     `gorm:"primaryKey" json:"user_id"`
	State     string    `json:"state"`
	Data      string    `json:"data"` // JSON data
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&WeatherData{},
		&Subscription{},
		&AlertConfig{},
		&EnvironmentalAlert{},
		&UserSession{},
	)
}
