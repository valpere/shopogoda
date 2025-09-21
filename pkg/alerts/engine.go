package alerts

import (
	"fmt"
	"time"
)

// AlertLevel represents the severity level of an alert
type AlertLevel int

const (
	AlertLevelLow AlertLevel = iota + 1
	AlertLevelMedium
	AlertLevelHigh
	AlertLevelCritical
)

// String returns the string representation of AlertLevel
func (a AlertLevel) String() string {
	switch a {
	case AlertLevelLow:
		return "Low"
	case AlertLevelMedium:
		return "Medium"
	case AlertLevelHigh:
		return "High"
	case AlertLevelCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// Color returns the color representation for UI display
func (a AlertLevel) Color() string {
	switch a {
	case AlertLevelLow:
		return "🟢"
	case AlertLevelMedium:
		return "🟡"
	case AlertLevelHigh:
		return "🟠"
	case AlertLevelCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

// AlertType represents different types of environmental alerts
type AlertType int

const (
	AlertTypeTemperature AlertType = iota + 1
	AlertTypeHumidity
	AlertTypePressure
	AlertTypeWindSpeed
	AlertTypeAirQuality
	AlertTypeUVIndex
	AlertTypeVisibility
)

// String returns the string representation of AlertType
func (a AlertType) String() string {
	switch a {
	case AlertTypeTemperature:
		return "Temperature"
	case AlertTypeHumidity:
		return "Humidity"
	case AlertTypePressure:
		return "Pressure"
	case AlertTypeWindSpeed:
		return "Wind Speed"
	case AlertTypeAirQuality:
		return "Air Quality"
	case AlertTypeUVIndex:
		return "UV Index"
	case AlertTypeVisibility:
		return "Visibility"
	default:
		return "Unknown"
	}
}

// AlertCondition represents the condition for triggering an alert
type AlertCondition struct {
	Operator  string  `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
}

// Alert represents a triggered environmental alert
type Alert struct {
	ID          string        `json:"id"`
	Type        AlertType     `json:"type"`
	Level       AlertLevel    `json:"level"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Value       float64       `json:"value"`
	Threshold   float64       `json:"threshold"`
	Location    string        `json:"location"`
	Timestamp   time.Time     `json:"timestamp"`
	IsResolved  bool          `json:"is_resolved"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

// Engine handles alert processing and evaluation
type Engine struct {
	cooldownPeriod time.Duration
	lastTriggered  map[string]time.Time
}

// NewEngine creates a new alert engine
func NewEngine() *Engine {
	return &Engine{
		cooldownPeriod: time.Hour, // 1-hour cooldown
		lastTriggered:  make(map[string]time.Time),
	}
}

// EvaluateCondition checks if a condition is met
func (e *Engine) EvaluateCondition(value float64, condition AlertCondition) bool {
	switch condition.Operator {
	case "gt":
		return value > condition.Value
	case "lt":
		return value < condition.Value
	case "gte":
		return value >= condition.Value
	case "lte":
		return value <= condition.Value
	case "eq":
		return value == condition.Value
	default:
		return false
	}
}

// CalculateLevel determines the severity level based on alert type and values
func (e *Engine) CalculateLevel(alertType AlertType, currentValue, threshold float64) AlertLevel {
	deviation := currentValue - threshold
	if deviation < 0 {
		deviation = -deviation
	}

	switch alertType {
	case AlertTypeTemperature:
		if deviation > 15 {
			return AlertLevelCritical
		} else if deviation > 10 {
			return AlertLevelHigh
		} else if deviation > 5 {
			return AlertLevelMedium
		}
		return AlertLevelLow

	case AlertTypeAirQuality:
		// AQI scale: 0-50 Good, 51-100 Moderate, 101-150 Unhealthy for sensitive, 151-200 Unhealthy, 201-300 Very unhealthy, 301+ Hazardous
		if currentValue > 300 {
			return AlertLevelCritical
		} else if currentValue > 200 {
			return AlertLevelHigh
		} else if currentValue > 150 {
			return AlertLevelMedium
		}
		return AlertLevelLow

	case AlertTypeWindSpeed:
		// Wind speed in km/h
		if currentValue > 80 { // Hurricane force
			return AlertLevelCritical
		} else if currentValue > 60 { // Storm force
			return AlertLevelHigh
		} else if currentValue > 40 { // Gale force
			return AlertLevelMedium
		}
		return AlertLevelLow

	case AlertTypeHumidity:
		// Humidity percentage
		if deviation > 30 {
			return AlertLevelHigh
		} else if deviation > 20 {
			return AlertLevelMedium
		}
		return AlertLevelLow

	case AlertTypeUVIndex:
		// UV Index scale: 0-2 Low, 3-5 Moderate, 6-7 High, 8-10 Very High, 11+ Extreme
		if currentValue > 10 {
			return AlertLevelCritical
		} else if currentValue > 7 {
			return AlertLevelHigh
		} else if currentValue > 5 {
			return AlertLevelMedium
		}
		return AlertLevelLow

	default:
		return AlertLevelMedium
	}
}

// CreateAlert creates a new alert with proper formatting
func (e *Engine) CreateAlert(alertType AlertType, currentValue, threshold float64, location string) *Alert {
	level := e.CalculateLevel(alertType, currentValue, threshold)

	title := fmt.Sprintf("%s Alert - %s", alertType.String(), level.String())

	var description string
	var unit string

	switch alertType {
	case AlertTypeTemperature:
		unit = "°C"
		if currentValue > threshold {
			description = fmt.Sprintf("Temperature is %.1f%s, above threshold of %.1f%s", currentValue, unit, threshold, unit)
		} else {
			description = fmt.Sprintf("Temperature is %.1f%s, below threshold of %.1f%s", currentValue, unit, threshold, unit)
		}
	case AlertTypeHumidity:
		unit = "%"
		if currentValue > threshold {
			description = fmt.Sprintf("Humidity is %.0f%s, above threshold of %.0f%s", currentValue, unit, threshold, unit)
		} else {
			description = fmt.Sprintf("Humidity is %.0f%s, below threshold of %.0f%s", currentValue, unit, threshold, unit)
		}
	case AlertTypeAirQuality:
		description = fmt.Sprintf("Air Quality Index is %.0f, threshold: %.0f", currentValue, threshold)
		if currentValue > 150 {
			description += " - Unhealthy for sensitive groups"
		}
	case AlertTypeWindSpeed:
		unit = "km/h"
		description = fmt.Sprintf("Wind speed is %.1f%s, threshold: %.1f%s", currentValue, unit, threshold, unit)
	case AlertTypeUVIndex:
		description = fmt.Sprintf("UV Index is %.1f, threshold: %.1f", currentValue, threshold)
		if currentValue > 7 {
			description += " - Very high UV exposure"
		}
	default:
		description = fmt.Sprintf("Value is %.2f, threshold: %.2f", currentValue, threshold)
	}

	return &Alert{
		ID:          generateAlertID(),
		Type:        alertType,
		Level:       level,
		Title:       title,
		Description: description,
		Value:       currentValue,
		Threshold:   threshold,
		Location:    location,
		Timestamp:   time.Now(),
		IsResolved:  false,
	}
}

// CanTrigger checks if an alert can be triggered based on cooldown period
func (e *Engine) CanTrigger(alertKey string) bool {
	if lastTime, exists := e.lastTriggered[alertKey]; exists {
		return time.Since(lastTime) >= e.cooldownPeriod
	}
	return true
}

// MarkTriggered marks an alert as triggered to enforce cooldown
func (e *Engine) MarkTriggered(alertKey string) {
	e.lastTriggered[alertKey] = time.Now()
}

// FormatAlertMessage formats an alert for display in Telegram
func (e *Engine) FormatAlertMessage(alert *Alert) string {
	return fmt.Sprintf(
		"%s %s\n\n"+
			"📍 Location: %s\n"+
			"📊 Current: %.2f\n"+
			"⚠️ Threshold: %.2f\n"+
			"🕐 Time: %s\n\n"+
			"%s",
		alert.Level.Color(),
		alert.Title,
		alert.Location,
		alert.Value,
		alert.Threshold,
		alert.Timestamp.Format("15:04 MST"),
		alert.Description,
	)
}

// FormatSlackMessage formats an alert for Slack/Teams notifications
func (e *Engine) FormatSlackMessage(alert *Alert) map[string]interface{} {
	color := "#36a64f" // Green (Low)
	switch alert.Level {
	case AlertLevelMedium:
		color = "#ffcc00" // Yellow
	case AlertLevelHigh:
		color = "#ff9900" // Orange
	case AlertLevelCritical:
		color = "#ff0000" // Red
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     alert.Title,
				"text":      alert.Description,
				"timestamp": alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Location",
						"value": alert.Location,
						"short": true,
					},
					{
						"title": "Current Value",
						"value": fmt.Sprintf("%.2f", alert.Value),
						"short": true,
					},
					{
						"title": "Threshold",
						"value": fmt.Sprintf("%.2f", alert.Threshold),
						"short": true,
					},
					{
						"title": "Severity",
						"value": alert.Level.String(),
						"short": true,
					},
				},
			},
		},
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// ValidateCondition validates an alert condition
func ValidateCondition(condition AlertCondition) error {
	validOperators := map[string]bool{
		"gt":  true,
		"lt":  true,
		"gte": true,
		"lte": true,
		"eq":  true,
	}

	if !validOperators[condition.Operator] {
		return fmt.Errorf("invalid operator: %s", condition.Operator)
	}

	return nil
}

// GetAirQualityDescription returns a human-readable description of AQI
func GetAirQualityDescription(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good - Air quality is satisfactory"
	case aqi <= 100:
		return "Moderate - Air quality is acceptable for most people"
	case aqi <= 150:
		return "Unhealthy for Sensitive Groups"
	case aqi <= 200:
		return "Unhealthy - Everyone may experience health effects"
	case aqi <= 300:
		return "Very Unhealthy - Health alert"
	default:
		return "Hazardous - Health warnings of emergency conditions"
	}
}

// GetUVIndexDescription returns a human-readable description of UV Index
func GetUVIndexDescription(uvIndex float64) string {
	switch {
	case uvIndex <= 2:
		return "Low - Minimal protection required"
	case uvIndex <= 5:
		return "Moderate - Take precautions when outside"
	case uvIndex <= 7:
		return "High - Protection required"
	case uvIndex <= 10:
		return "Very High - Extra protection required"
	default:
		return "Extreme - Avoid being outside"
	}
}