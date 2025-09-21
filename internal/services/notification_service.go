package services

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/rs/zerolog"

    "github.com/valpere/shopogoda/internal/config"
    "github.com/valpere/shopogoda/internal/models"
)

type NotificationService struct {
    config *config.IntegrationsConfig
    logger *zerolog.Logger
    client *http.Client
}

type SlackMessage struct {
    Text        string            `json:"text"`
    Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
    Color  string       `json:"color"`
    Title  string       `json:"title"`
    Text   string       `json:"text"`
    Fields []SlackField `json:"fields,omitempty"`
}

type SlackField struct {
    Title string `json:"title"`
    Value string `json:"value"`
    Short bool   `json:"short"`
}

func NewNotificationService(config *config.IntegrationsConfig, logger *zerolog.Logger) *NotificationService {
    return &NotificationService{
        config: config,
        logger: logger,
        client: &http.Client{},
    }
}

func (s *NotificationService) SendSlackAlert(alert *models.EnvironmentalAlert, location *models.Location) error {
    if s.config.SlackWebhookURL == "" {
        return nil // Slack not configured
    }

    color := s.getSeverityColor(alert.Severity)

    message := SlackMessage{
        Text: "🚨 Weather Alert",
        Attachments: []SlackAttachment{
            {
                Color: color,
                Title: alert.Title,
                Text:  alert.Description,
                Fields: []SlackField{
                    {Title: "Location", Value: location.Name, Short: true},
                    {Title: "Severity", Value: s.getSeverityText(alert.Severity), Short: true},
                    {Title: "Current Value", Value: fmt.Sprintf("%.1f", alert.Value), Short: true},
                    {Title: "Threshold", Value: fmt.Sprintf("%.1f", alert.Threshold), Short: true},
                },
            },
        },
    }

    return s.sendSlackMessage(message)
}

func (s *NotificationService) SendSlackWeatherUpdate(weather *WeatherData, subscribers []models.User) error {
    if s.config.SlackWebhookURL == "" {
        return nil
    }

    message := SlackMessage{
        Text: fmt.Sprintf("🌤️ Daily Weather Update for %s", weather.LocationName),
        Attachments: []SlackAttachment{
            {
                Color: "good",
                Title: "Current Conditions",
                Fields: []SlackField{
                    {Title: "Temperature", Value: fmt.Sprintf("%.1f°C", weather.Temperature), Short: true},
                    {Title: "Humidity", Value: fmt.Sprintf("%d%%", weather.Humidity), Short: true},
                    {Title: "Wind", Value: fmt.Sprintf("%.1f km/h %s", weather.WindSpeed, weather.WindDirection), Short: true},
                    {Title: "Air Quality", Value: fmt.Sprintf("AQI %d", weather.AQI), Short: true},
                },
            },
        },
    }

    return s.sendSlackMessage(message)
}

func (s *NotificationService) sendSlackMessage(message SlackMessage) error {
    jsonData, err := json.Marshal(message)
    if err != nil {
        return err
    }

    resp, err := s.client.Post(s.config.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
    }

    s.logger.Info().Msg("Slack notification sent successfully")
    return nil
}

func (s *NotificationService) getSeverityColor(severity models.Severity) string {
    switch severity {
    case models.SeverityLow:
        return "good"
    case models.SeverityMedium:
        return "warning"
    case models.SeverityHigh:
        return "danger"
    case models.SeverityCritical:
        return "#ff0000"
    default:
        return "warning"
    }
}

func (s *NotificationService) getSeverityText(severity models.Severity) string {
    switch severity {
    case models.SeverityLow:
        return "Low"
    case models.SeverityMedium:
        return "Medium"
    case models.SeverityHigh:
        return "High"
    case models.SeverityCritical:
        return "Critical"
    default:
        return "Unknown"
    }
}
