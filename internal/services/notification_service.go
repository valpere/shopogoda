package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/zerolog"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
)

type NotificationService struct {
	config *config.IntegrationsConfig
	logger *zerolog.Logger
	client *http.Client
	bot    *gotgbot.Bot // Telegram bot instance for sending notifications
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

// SetBot sets the Telegram bot instance for sending notifications
func (s *NotificationService) SetBot(bot *gotgbot.Bot) {
	s.bot = bot
}

// getTelegramChatID returns the chat ID for sending direct messages to a user
// For direct messages to users, chat ID is the same as user ID
// See: https://core.telegram.org/bots/api#chat
func (s *NotificationService) getTelegramChatID(user *models.User) int64 {
	return user.ID
}

func (s *NotificationService) SendSlackAlert(alert *models.EnvironmentalAlert, user *models.User) error {
	if s.config.SlackWebhookURL == "" {
		return nil // Slack not configured
	}

	color := s.getSeverityColor(alert.Severity)

	locationName := user.LocationName
	if locationName == "" {
		locationName = "Unknown Location"
	}

	message := SlackMessage{
		Text: "🚨 Weather Alert",
		Attachments: []SlackAttachment{
			{
				Color: color,
				Title: alert.Title,
				Text:  alert.Description,
				Fields: []SlackField{
					{Title: "Location", Value: locationName, Short: true},
					{Title: "User", Value: user.GetDisplayName(), Short: true},
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
					{Title: "Wind", Value: fmt.Sprintf("%.1f km/h %d°", weather.WindSpeed, weather.WindDirection), Short: true},
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

// Telegram notification methods

// SendTelegramAlert sends a Telegram alert notification to a user
func (s *NotificationService) SendTelegramAlert(alert *models.EnvironmentalAlert, user *models.User) error {
	if s.bot == nil {
		s.logger.Debug().Msg("Telegram bot not configured for notifications")
		return nil
	}

	severityEmoji := s.getSeverityEmoji(alert.Severity)
	locationName := user.LocationName
	if locationName == "" {
		locationName = "Unknown Location"
	}

	message := fmt.Sprintf(`%s *Weather Alert*

*%s*
%s

📍 *Location:* %s
👤 *User:* %s
🚨 *Severity:* %s
📊 *Current Value:* %.1f
⚠️ *Threshold:* %.1f`,
		severityEmoji,
		alert.Title,
		alert.Description,
		locationName,
		user.GetDisplayName(),
		s.getSeverityText(alert.Severity),
		alert.Value,
		alert.Threshold)

	// Assumption: For direct messages to users, chat ID is the same as user ID.
	// This is only valid if the user has already started a private chat with the bot.
	// If the user has not started a chat, this will fail with "bot was blocked by the user" or "chat not found".
	// See: https://core.telegram.org/bots/api#chat
	// 
	// Error handling: Callers of this function should expect and handle errors returned here,
	// as they may indicate that the user has not started a chat with the bot or has blocked the bot.
	// Possible recovery mechanisms include notifying the user through another channel,
	// prompting the user to start a chat with the bot, or logging the incident for further review.
	chatID := s.getTelegramChatID(user)
	_, err := s.bot.SendMessage(chatID, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	if err != nil {
		s.logger.Error().
			Err(err).
			Int64("user_id", user.ID).
			Int64("chat_id", chatID).
			Msg("Failed to send Telegram alert - user may have blocked bot or deleted chat")
		return fmt.Errorf("failed to send Telegram alert to user %d: %w", user.ID, err)
	}

	s.logger.Info().Int64("user_id", user.ID).Int64("chat_id", chatID).Msg("Telegram alert notification sent successfully")
	return nil
}

// SendTelegramWeatherUpdate sends a daily weather update to users via Telegram
func (s *NotificationService) SendTelegramWeatherUpdate(weather *WeatherData, user *models.User) error {
	if s.bot == nil {
		s.logger.Debug().Msg("Telegram bot not configured for notifications")
		return nil
	}

	message := fmt.Sprintf(`☀️ *Daily Weather Update*
📍 *%s*

🌡️ *Temperature:* %.1f°C
💧 *Humidity:* %d%%
💨 *Wind:* %.1f km/h %d°
🌿 *Air Quality:* AQI %d
👁️ *Visibility:* %.1f km
📅 *Updated:* %s`,
		weather.LocationName,
		weather.Temperature,
		weather.Humidity,
		weather.WindSpeed,
		weather.WindDirection,
		weather.AQI,
		weather.Visibility,
		weather.Timestamp.Format("15:04 UTC"))

	chatID := s.getTelegramChatID(user)
	_, err := s.bot.SendMessage(chatID, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	if err != nil {
		s.logger.Error().
			Err(err).
			Int64("user_id", user.ID).
			Int64("chat_id", chatID).
			Msg("Failed to send Telegram weather update - user may have blocked bot or deleted chat")
		return fmt.Errorf("failed to send Telegram weather update to user %d: %w", user.ID, err)
	}

	s.logger.Info().Int64("user_id", user.ID).Int64("chat_id", chatID).Msg("Telegram weather update sent successfully")
	return nil
}

// SendTelegramWeeklyUpdate sends a weekly weather summary to users via Telegram
func (s *NotificationService) SendTelegramWeeklyUpdate(user *models.User, summary string) error {
	if s.bot == nil {
		s.logger.Debug().Msg("Telegram bot not configured for notifications")
		return nil
	}

	message := fmt.Sprintf(`📅 *Weekly Weather Summary*
📍 *%s*

%s

Have a great week ahead! 🌟`, user.LocationName, summary)

	chatID := s.getTelegramChatID(user)
	_, err := s.bot.SendMessage(chatID, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	if err != nil {
		s.logger.Error().
			Err(err).
			Int64("user_id", user.ID).
			Int64("chat_id", chatID).
			Msg("Failed to send Telegram weekly update - user may have blocked bot or deleted chat")
		return fmt.Errorf("failed to send Telegram weekly update to user %d: %w", user.ID, err)
	}

	s.logger.Info().Int64("user_id", user.ID).Int64("chat_id", chatID).Msg("Telegram weekly update sent successfully")
	return nil
}

func (s *NotificationService) getSeverityEmoji(severity models.Severity) string {
	switch severity {
	case models.SeverityLow:
		return "⚠️"
	case models.SeverityMedium:
		return "🔶"
	case models.SeverityHigh:
		return "🚨"
	case models.SeverityCritical:
		return "🆘"
	default:
		return "ℹ️"
	}
}
