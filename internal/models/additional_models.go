package models

import (
    "fmt"
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Additional model methods and constants

// String methods for enums
func (r UserRole) String() string {
    switch r {
    case RoleUser:
        return "User"
    case RoleModerator:
        return "Moderator"
    case RoleAdmin:
        return "Admin"
    default:
        return "Unknown"
    }
}

func (s SubscriptionType) String() string {
    switch s {
    case SubscriptionDaily:
        return "Daily"
    case SubscriptionWeekly:
        return "Weekly"
    case SubscriptionAlerts:
        return "Alerts"
    case SubscriptionExtreme:
        return "Extreme"
    default:
        return "Unknown"
    }
}

func (f Frequency) String() string {
    switch f {
    case FrequencyHourly:
        return "Hourly"
    case FrequencyEvery3Hours:
        return "Every 3 Hours"
    case FrequencyEvery6Hours:
        return "Every 6 Hours"
    case FrequencyDaily:
        return "Daily"
    case FrequencyWeekly:
        return "Weekly"
    default:
        return "Unknown"
    }
}

func (a AlertType) String() string {
    switch a {
    case AlertTemperature:
        return "Temperature"
    case AlertHumidity:
        return "Humidity"
    case AlertPressure:
        return "Pressure"
    case AlertWindSpeed:
        return "Wind Speed"
    case AlertUVIndex:
        return "UV Index"
    case AlertAirQuality:
        return "Air Quality"
    case AlertRain:
        return "Rain"
    case AlertSnow:
        return "Snow"
    case AlertStorm:
        return "Storm"
    default:
        return "Unknown"
    }
}

func (s Severity) String() string {
    switch s {
    case SeverityLow:
        return "Low"
    case SeverityMedium:
        return "Medium"
    case SeverityHigh:
        return "High"
    case SeverityCritical:
        return "Critical"
    default:
        return "Unknown"
    }
}

// Helper methods for models
func (u *User) GetDisplayName() string {
    if u.FirstName != "" && u.LastName != "" {
        return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
    } else if u.FirstName != "" {
        return u.FirstName
    } else if u.Username != "" {
        return "@" + u.Username
    }
    return fmt.Sprintf("User_%d", u.ID)
}

func (l *Location) GetCoordinatesString() string {
    return fmt.Sprintf("%.4f, %.4f", l.Latitude, l.Longitude)
}

func (w *WeatherData) GetTemperatureString() string {
    return fmt.Sprintf("%.1f°C", w.Temperature)
}

func (a *AlertConfig) IsRecentlyTriggered() bool {
    if a.LastTriggered == nil {
        return false
    }
    return time.Since(*a.LastTriggered) < time.Hour
}

func (ea *EnvironmentalAlert) GetSeverityColor() string {
    switch ea.Severity {
    case SeverityLow:
        return "🟢"
    case SeverityMedium:
        return "🟡"
    case SeverityHigh:
        return "🟠"
    case SeverityCritical:
        return "🔴"
    default:
        return "⚪"
    }
}

func (ea *EnvironmentalAlert) GetFormattedMessage() string {
    return fmt.Sprintf("%s %s\n📊 Value: %.1f (Threshold: %.1f)\n%s",
        ea.GetSeverityColor(),
        ea.Title,
        ea.Value,
        ea.Threshold,
        ea.Description)
}

// Validation methods
func (u *User) Validate() error {
    if u.ID <= 0 {
        return fmt.Errorf("invalid user ID")
    }
    if u.FirstName == "" {
        return fmt.Errorf("first name is required")
    }
    return nil
}

func (l *Location) Validate() error {
    if l.UserID <= 0 {
        return fmt.Errorf("invalid user ID")
    }
    if l.Name == "" {
        return fmt.Errorf("location name is required")
    }
    if l.Latitude < -90 || l.Latitude > 90 {
        return fmt.Errorf("invalid latitude: %f", l.Latitude)
    }
    if l.Longitude < -180 || l.Longitude > 180 {
        return fmt.Errorf("invalid longitude: %f", l.Longitude)
    }
    return nil
}

func (s *Subscription) Validate() error {
    if s.UserID <= 0 {
        return fmt.Errorf("invalid user ID")
    }
    if s.LocationID == uuid.Nil {
        return fmt.Errorf("location ID is required")
    }
    if s.SubscriptionType < 1 || s.SubscriptionType > 4 {
        return fmt.Errorf("invalid subscription type")
    }
    if s.Frequency < 1 || s.Frequency > 5 {
        return fmt.Errorf("invalid frequency")
    }
    return nil
}

func (a *AlertConfig) Validate() error {
    if a.UserID <= 0 {
        return fmt.Errorf("invalid user ID")
    }
    if a.LocationID == uuid.Nil {
        return fmt.Errorf("location ID is required")
    }
    if a.AlertType < 1 || a.AlertType > 9 {
        return fmt.Errorf("invalid alert type")
    }
    if a.Condition == "" {
        return fmt.Errorf("alert condition is required")
    }
    return nil
}

// Database hooks
func (u *User) BeforeCreate(tx *gorm.DB) error {
    return u.Validate()
}

func (l *Location) BeforeCreate(tx *gorm.DB) error {
    return l.Validate()
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
    return s.Validate()
}

func (a *AlertConfig) BeforeCreate(tx *gorm.DB) error {
    return a.Validate()
}
