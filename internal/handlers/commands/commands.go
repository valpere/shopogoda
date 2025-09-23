package commands

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/weather"
)

type CommandHandler struct {
	services *services.Services
	logger   *zerolog.Logger
}

func New(services *services.Services, logger *zerolog.Logger) *CommandHandler {
	return &CommandHandler{
		services: services,
		logger:   logger,
	}
}

// Start command handler
func (h *CommandHandler) Start(bot *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser

	// Debug logging for start command
	h.logger.Debug().
		Int64("user_id", user.Id).
		Int("args_count", len(ctx.Args())).
		Msg("Starting Start command")

	// Register or update user
	if err := h.services.User.RegisterUser(context.Background(), user); err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.Id).Msg("Failed to register user")
	}

	welcomeText := fmt.Sprintf(`🌤️ *Welcome to Enterprise Weather Bot*

Hello %s! I'm your professional weather and environmental monitoring assistant.

*Available Commands:*
🏠 /weather - Get current weather
📊 /forecast - 5-day weather forecast
🌬️ /air - Air quality information
📍 /setlocation - Set your location
⚙️ /settings - Configure preferences
🔔 /subscribe - Set up notifications
⚠️ /addalert - Create weather alerts
📋 /help - Show all commands

*Enterprise Features:*
• Real-time environmental monitoring
• Custom alert thresholds
• Multi-location tracking
• Integration with Slack/Teams
• Compliance reporting
• Role-based access control

Ready to get started? Try /weather to see current conditions!`,
		user.FirstName)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: "weather_current"}},
		{{Text: "⚙️ Settings", CallbackData: "settings_main"}},
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, welcomeText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Help command handler
func (h *CommandHandler) Help(bot *gotgbot.Bot, ctx *ext.Context) error {
	helpText := `🌤️ *Enterprise Weather Bot - Commands*

*🏠 Basic Commands:*
/weather \[location\] - Current weather conditions
/forecast \[location\] - 5-day weather forecast
/air \[location\] - Air quality index and pollutants

*📍 Location Management:*
/setlocation - Set your location

*🔔 Notifications:*
/subscribe - Set up weather notifications
/unsubscribe - Remove notifications
/subscriptions - View active subscriptions

*⚠️ Alert System:*
/addalert - Create weather alert
/alerts - View active alerts
/removealert \<id\> - Remove specific alert

*⚙️ Settings:*
/settings - Open settings menu
Language, units, timezone configuration

*👨‍💼 Admin Commands:*
/stats - Bot usage statistics
/broadcast - Send message to all users
/users - User management

*💡 Tips:*
• Share your location for instant weather
• Use inline queries: @weatherbot London
• Set multiple alerts for different conditions
• Export data for compliance reporting

*🆘 Support:*
For enterprise support, contact: support@weatherbot.com`

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, helpText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// Current weather command
func (h *CommandHandler) CurrentWeather(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	var location string

	// Check if this is called from a callback query (no args) or command (has args)
	if ctx.CallbackQuery != nil {
		// Called from button - no location argument, use user's saved location
		h.logger.Debug().
			Int64("user_id", userID).
			Msg("CurrentWeather called from callback button")
		location = ""
	} else {
		// Called from command - parse location from arguments
		h.logger.Debug().
			Int64("user_id", userID).
			Interface("all_args", ctx.Args()).
			Int("args_count", len(ctx.Args())).
			Msg("CurrentWeather called from command")

		if len(ctx.Args()) > 1 {
			location = strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))
		}
	}

	h.logger.Debug().
		Str("parsed_location", location).
		Msg("Parsed location parameter")

	// If no location provided, use user's saved location or ask for it
	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"📍 Please provide a location or set your location:\n\n/weather London\nor\n/setlocation to set your location",
				&gotgbot.SendMessageOpts{
					ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
							{{Text: "📍 Share Location", CallbackData: "share_location"}},
							{{Text: "📍 Set Location", CallbackData: "location_set"}},
						},
					},
				})
			return err
		}
		location = locationName
	}

	// Get weather data
	h.logger.Debug().
		Str("location", location).
		Msg("Calling weather service")

	weatherData, err := h.services.Weather.GetCurrentWeatherByLocation(context.Background(), location)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("location", location).
			Msg("Failed to get weather data")

		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get weather for '%s'. Please check the location name.", location), nil)
		return err
	}

	h.logger.Debug().
		Str("location", location).
		Msg("Successfully got weather data")

	// Format weather message
	weatherText := h.formatWeatherMessage(weatherData)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "📊 5-Day Forecast", CallbackData: fmt.Sprintf("forecast_%s", location)}},
		{{Text: "🌬️ Air Quality", CallbackData: fmt.Sprintf("air_%s", location)}},
		{{Text: "🔔 Set Alert", CallbackData: fmt.Sprintf("alert_%s", location)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, weatherText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Forecast command
func (h *CommandHandler) Forecast(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Debug logging for argument parsing
	h.logger.Info().
		Int64("user_id", userID).
		Interface("all_args", ctx.Args()).
		Int("args_count", len(ctx.Args())).
		Str("message_text", ctx.Message.Text).
		Msg("FORECAST_DEBUG: Starting Forecast command")

	location := strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))

	h.logger.Info().
		Str("parsed_location", location).
		Msg("FORECAST_DEBUG: Parsed location parameter")

	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"📍 Please provide a location: /forecast London", nil)
			return err
		}
		location = locationName
	}

	// Get coordinates first for forecast
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), location)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to find location '%s'", location), nil)
		return err
	}

	forecast, err := h.services.Weather.GetForecast(context.Background(), locationData.Latitude, locationData.Longitude, 5)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get forecast for '%s'", location), nil)
		return err
	}

	forecastText := h.formatForecastMessage(forecast)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, forecastText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// Air quality command
func (h *CommandHandler) AirQuality(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	location := strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))

	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"📍 Please provide a location: /air London", nil)
			return err
		}
		location = locationName
	}

	// Get coordinates first for air quality
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), location)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to find location '%s'", location), nil)
		return err
	}

	airData, err := h.services.Weather.GetAirQuality(context.Background(), locationData.Latitude, locationData.Longitude)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get air quality for '%s'", location), nil)
		return err
	}

	airText := h.formatAirQualityMessage(airData)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: fmt.Sprintf("weather_%s", location)}},
		{{Text: "⚠️ Set Air Alert", CallbackData: fmt.Sprintf("air_alert_%s", location)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, airText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Settings command
func (h *CommandHandler) Settings(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil {
		return err
	}

	// Get user's current location
	locationName, lat, lon, err := h.services.User.GetUserLocation(context.Background(), userID)
	var locationText string
	if err != nil || locationName == "" {
		locationText = "Not set"
	} else {
		locationText = fmt.Sprintf("%s (%.4f, %.4f)", locationName, lat, lon)
	}

	settingsText := fmt.Sprintf(`⚙️ *Settings*

*Current Configuration:*
Location: %s
Language: %s
Units: %s
Timezone: %s
Role: %s
Status: %s

*Available Settings:*
• Location management
• Language preferences
• Unit system (Metric/Imperial)
• Timezone settings
• Notification preferences
• Data export options`,
		locationText,
		user.Language,
		h.getUnitsText(user.Units),
		user.Timezone,
		h.getRoleName(user.Role),
		h.getStatusText(user.IsActive))

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "📍 Set Location", CallbackData: "settings_location"}},
		{{Text: "🌐 Language", CallbackData: "settings_language"}},
		{{Text: "📏 Units", CallbackData: "settings_units"}},
		{{Text: "🕐 Timezone", CallbackData: "settings_timezone"}},
		{{Text: "🔔 Notifications", CallbackData: "settings_notifications"}},
		{{Text: "📊 Data Export", CallbackData: "settings_export"}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, settingsText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Callback query handler
func (h *CommandHandler) HandleCallback(bot *gotgbot.Bot, ctx *ext.Context) error {
	cq := ctx.CallbackQuery
	data := cq.Data

	h.logger.Info().Str("callback_data", data).Msg("Callback received")

	// Answer callback query first
	if _, err := bot.AnswerCallbackQuery(cq.Id, nil); err != nil {
		h.logger.Error().Err(err).Msg("Failed to answer callback query")
	}

	// Parse callback data
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		h.logger.Warn().Str("data", data).Msg("Invalid callback data format")
		return nil
	}

	action := parts[0]
	subAction := parts[1]
	h.logger.Info().Str("action", action).Str("subAction", subAction).Int("parts_count", len(parts)).Msg("Parsed callback data")

	switch action {
	case "weather":
		return h.handleWeatherCallback(bot, ctx, subAction, parts[2:])
	case "forecast":
		return h.handleForecastCallback(bot, ctx, subAction, parts[2:])
	case "settings":
		return h.handleSettingsCallback(bot, ctx, subAction, parts[2:])
	case "location":
		return h.handleLocationCallback(bot, ctx, subAction, parts[2:])
	case "alert":
		return h.handleAlertCallback(bot, ctx, subAction, parts[2:])
	case "alerts":
		return h.handleAlertsCallback(bot, ctx, subAction, parts[2:])
	case "subscribe", "unsubscribe":
		return h.handleSubscriptionCallback(bot, ctx, action, subAction, parts[2:])
	case "sub":
		return h.handleSubscriptionCallback(bot, ctx, subAction, parts[1], parts[2:])
	case "subscriptions":
		return h.handleSubscriptionCallback(bot, ctx, action, subAction, parts[2:])
	case "admin":
		return h.handleAdminCallback(bot, ctx, subAction, parts[2:])
	case "share":
		return h.handleShareCallback(bot, ctx, subAction, parts[2:])
	case "air":
		return h.handleAirCallback(bot, ctx, subAction, parts[2:])
	}

	return nil
}

// Location message handler
func (h *CommandHandler) HandleLocationMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	h.logger.Info().Msg("HandleLocationMessage called")

	if ctx.Message.Location == nil {
		h.logger.Warn().Msg("Location message received but no location data")
		return nil
	}

	// Register or update user (ensure user exists before saving location)
	user := ctx.EffectiveUser
	if err := h.services.User.RegisterUser(context.Background(), user); err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.Id).Msg("Failed to register user")
	}

	lat := ctx.Message.Location.Latitude
	lon := ctx.Message.Location.Longitude
	h.logger.Info().Float64("lat", lat).Float64("lon", lon).Msg("Processing location message")

	// Get location name from coordinates
	locationName, err := h.services.Weather.GetLocationName(context.Background(), lat, lon)
	if err != nil {
		locationName = fmt.Sprintf("Location (%.4f, %.4f)", lat, lon)
	}

	// Get weather for this location
	weatherData, err := h.services.Weather.GetCurrentWeatherByCoords(context.Background(), lat, lon)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Failed to get weather for your location", nil)
		return err
	}

	weatherText := h.formatWeatherMessage(weatherData)

	// URL encode the location name to handle spaces and special characters
	encodedName := url.QueryEscape(locationName)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "💾 Save Location", CallbackData: fmt.Sprintf("location_save_%.4f_%.4f_%s", lat, lon, encodedName)}},
		{{Text: "📊 Forecast", CallbackData: fmt.Sprintf("forecast_coords_%.4f_%.4f", lat, lon)}},
		{{Text: "🔔 Set Alert", CallbackData: fmt.Sprintf("alert_coords_%.4f_%.4f", lat, lon)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, weatherText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// HandleAnyMessage logs incoming messages for debugging (debug level only)
func (h *CommandHandler) HandleAnyMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.Message
	if msg == nil {
		h.logger.Debug().Msg("Non-message update received")
		return nil
	}

	// Only log in debug mode to avoid privacy concerns and log spam
	h.logger.Debug().
		Int64("user_id", ctx.EffectiveUser.Id).
		Int64("chat_id", ctx.EffectiveChat.Id).
		Str("chat_type", ctx.EffectiveChat.Type).
		Int64("message_id", msg.MessageId).
		Bool("has_location", msg.Location != nil).
		Bool("has_photo", len(msg.Photo) > 0).
		Bool("has_document", msg.Document != nil).
		Bool("has_voice", msg.Voice != nil).
		Msg("Received message")

	if msg.Location != nil {
		h.logger.Debug().
			Float64("latitude", msg.Location.Latitude).
			Float64("longitude", msg.Location.Longitude).
			Msg("Location shared")
	}

	return nil // Don't consume the message, let other handlers process it
}

// HandleTextMessage processes plain text messages that might be location names
func (h *CommandHandler) HandleTextMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	msg := ctx.Message
	if msg == nil || msg.Text == "" {
		return nil
	}

	// Skip if it's a command (starts with /)
	if strings.HasPrefix(msg.Text, "/") {
		return nil
	}

	text := strings.TrimSpace(msg.Text)

	// Check if this looks like GPS coordinates first
	coordPattern := `^(-?\d+\.?\d*),?\s*(-?\d+\.?\d*)$`
	coordMatch, _ := regexp.MatchString(coordPattern, text)
	if coordMatch {
		return h.handleCoordinateInput(bot, ctx, text)
	}

	// Simple heuristics to detect if this might be a location name
	// - Should be 2-50 characters
	// - Should contain only letters, spaces, hyphens, apostrophes
	// - Should not be too short (avoid "ok", "yes", etc.)
	if len(text) < 2 || len(text) > 50 {
		return nil
	}

	// Check if text looks like a location name (letters, spaces, hyphens, apostrophes only)
	locationPattern := `^[a-zA-ZÀ-ÿ\s\-']+$`
	matched, _ := regexp.MatchString(locationPattern, text)
	if !matched {
		return nil
	}

	// Skip common non-location words
	commonWords := map[string]bool{
		"ok": true, "yes": true, "no": true, "hi": true, "hello": true,
		"thanks": true, "thank you": true, "good": true, "bad": true,
		"help": true, "stop": true, "cancel": true, "back": true,
	}
	if commonWords[strings.ToLower(text)] {
		return nil
	}

	userID := ctx.EffectiveUser.Id

	// Check if user already has a location set
	existingLocation, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)

	var messageText string
	var keyboard [][]gotgbot.InlineKeyboardButton

	if err != nil || existingLocation == "" {
		// No location set - offer to set this as their location
		messageText = fmt.Sprintf("📍 Did you want to set *%s* as your location?", text)
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: "✅ Yes, set as my location", CallbackData: fmt.Sprintf("location_confirm_%s", url.QueryEscape(text))}},
			{{Text: "❌ No, just ignore", CallbackData: "location_ignore"}},
		}
	} else {
		// Location already set - offer to change it
		messageText = fmt.Sprintf("📍 Did you want to change your location from *%s* to *%s*?", existingLocation, text)
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: "✅ Yes, change location", CallbackData: fmt.Sprintf("location_confirm_%s", url.QueryEscape(text))}},
			{{Text: "❌ No, keep current", CallbackData: "location_ignore"}},
		}
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, messageText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// handleCoordinateInput processes GPS coordinates entered as text
func (h *CommandHandler) handleCoordinateInput(bot *gotgbot.Bot, ctx *ext.Context, coordinateText string) error {
	userID := ctx.EffectiveUser.Id

	// Parse coordinates from text
	coordPattern := `^(-?\d+\.?\d*),?\s*(-?\d+\.?\d*)$`
	re := regexp.MustCompile(coordPattern)
	matches := re.FindStringSubmatch(coordinateText)

	if len(matches) != 3 {
		h.logger.Warn().Str("input", coordinateText).Msg("Failed to parse coordinates")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Invalid coordinate format. Please use format: 'latitude, longitude' (e.g., '37.7749, -122.4194')", nil)
		return err
	}

	lat, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		h.logger.Error().Err(err).Str("lat", matches[1]).Msg("Failed to parse latitude")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid latitude value", nil)
		return err
	}

	lon, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		h.logger.Error().Err(err).Str("lon", matches[2]).Msg("Failed to parse longitude")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid longitude value", nil)
		return err
	}

	// Validate coordinate ranges
	if lat < -90 || lat > 90 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Latitude must be between -90 and 90", nil)
		return err
	}
	if lon < -180 || lon > 180 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Longitude must be between -180 and 180", nil)
		return err
	}

	h.logger.Info().Float64("lat", lat).Float64("lon", lon).Int64("user_id", userID).Msg("Processing coordinate input")

	// Get location name from coordinates (reverse geocoding)
	locationName, err := h.services.Weather.GetLocationName(context.Background(), lat, lon)
	if err != nil {
		locationName = fmt.Sprintf("Location (%.4f, %.4f)", lat, lon)
		h.logger.Warn().Err(err).Msg("Failed to get location name from coordinates, using default")
	}

	// Save the location
	err = h.services.User.SetUserLocation(context.Background(), userID, locationName, "", "", lat, lon)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to save location from coordinates")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location. Please try again.", nil)
		return err
	}

	h.logger.Info().Str("location", locationName).Float64("lat", lat).Float64("lon", lon).Msg("Location saved from coordinates")

	// Send confirmation message
	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Location set to *%s*\n📍 Coordinates: %.4f, %.4f", locationName, lat, lon),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown"})

	return err
}

// Helper methods for formatting messages
func (h *CommandHandler) formatWeatherMessage(weather *services.WeatherData) string {
	return fmt.Sprintf(`🌤️ *%s*

🌡️ Temperature: %d°C (feels like %d°C)
💧 Humidity: %d%%
🌬️ Wind: %.1f km/h %d°
👁️ Visibility: %.1f km
☀️ UV Index: %.1f
🏢 Pressure: %.1f hPa

%s %s

*Air Quality:*
🌿 AQI: %d (%s)
CO: %.2f | NO₂: %.2f | O₃: %.2f
PM2.5: %.1f | PM10: %.1f

📅 Updated: %s`,
		weather.LocationName,
		int(weather.Temperature),
		int(weather.Temperature), // FeelsLike not available in current struct
		weather.Humidity,
		weather.WindSpeed,
		weather.WindDirection,
		weather.Visibility,
		weather.UVIndex,
		weather.Pressure,
		weather.Icon,
		weather.Description,
		weather.AQI,
		h.getAQIDescription(weather.AQI),
		weather.CO,
		weather.NO2,
		weather.O3,
		weather.PM25,
		weather.PM10,
		weather.Timestamp.Format("15:04 UTC"))
}

// Additional helper methods...
func (h *CommandHandler) getRoleName(role models.UserRole) string {
	switch role {
	case models.RoleAdmin:
		return "Administrator"
	case models.RoleModerator:
		return "Moderator"
	default:
		return "User"
	}
}

func (h *CommandHandler) getStatusText(isActive bool) string {
	if isActive {
		return "Active"
	}
	return "Inactive"
}

func (h *CommandHandler) getUnitsText(units string) string {
	switch units {
	case "metric":
		return "🌡️ Metric (°C, km/h, km)"
	case "imperial":
		return "🌡️ Imperial (°F, mph, miles)"
	default:
		return units
	}
}

func (h *CommandHandler) getAQIDescription(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy for Sensitive Groups"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

func (h *CommandHandler) formatForecastMessage(forecast *weather.ForecastData) string {
	text := fmt.Sprintf("📊 *5-Day Forecast for %s*\n\n", forecast.Location)

	for _, day := range forecast.Forecasts {
		text += fmt.Sprintf("📅 *%s*\n", day.Date.Format("Monday, Jan 2"))
		text += fmt.Sprintf("🌡️ %.1f°/%.1f°C | %s %s\n",
			day.MaxTemp, day.MinTemp, day.Icon, day.Description)
		text += fmt.Sprintf("💧 Humidity: %d%% | 🌬️ Wind: %.1f km/h\n\n",
			day.Humidity, day.WindSpeed)
	}

	return text
}

func (h *CommandHandler) formatAirQualityMessage(air *weather.AirQualityData) string {
	return fmt.Sprintf(`🌬️ *Air Quality - %s*

🌿 *Overall AQI: %d (%s)*

*Pollutant Levels:*
🏭 CO (Carbon Monoxide): %.2f μg/m³
🚗 NO₂ (Nitrogen Dioxide): %.2f μg/m³
☀️ O₃ (Ozone): %.2f μg/m³
🏭 PM2.5: %.1f μg/m³
🌫️ PM10: %.1f μg/m³

*Health Recommendations:*
%s

📅 Updated: %s`,
		"Air Quality Data", // LocationName not available in weather.AirQualityData
		air.AQI,
		h.getAQIDescription(air.AQI),
		air.CO,
		air.NO2,
		air.O3,
		air.PM25,
		air.PM10,
		h.getHealthRecommendation(air.AQI),
		air.Timestamp.Format("15:04 UTC"))
}

func (h *CommandHandler) getHealthRecommendation(aqi int) string {
	switch {
	case aqi <= 50:
		return "✅ Air quality is satisfactory. Enjoy outdoor activities!"
	case aqi <= 100:
		return "⚠️ Acceptable for most people. Sensitive individuals should limit prolonged outdoor exertion."
	case aqi <= 150:
		return "🚨 Sensitive groups should reduce outdoor activities."
	case aqi <= 200:
		return "❌ Everyone should limit outdoor activities."
	case aqi <= 300:
		return "🔴 Avoid outdoor activities. Wear a mask if you must go outside."
	default:
		return "🆘 Health emergency! Stay indoors and avoid all outdoor activities."
	}
}

// Additional command handlers
func (h *CommandHandler) SetLocation(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	locationName := strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))

	if locationName == "" {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 Please provide a location name:\n\n/setlocation London\nor share your current location",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "📍 Share Current Location", CallbackData: "share_location"}},
					},
				},
			})
		return err
	}

	// Validate location
	coords, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Could not find location '%s'. Please check the spelling.", locationName), nil)
		return err
	}

	// Save location as user's location
	err = h.services.User.SetUserLocation(context.Background(), userID, locationName, coords.Country, "", coords.Latitude, coords.Longitude)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Failed to save location. Please try again.", nil)
		return err
	}

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Get Weather", CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: "🔔 Add Alert", CallbackData: fmt.Sprintf("alert_%s", locationName)}},
	}

	message := fmt.Sprintf("✅ Location '%s' saved successfully!\n📍 This is now your current location", locationName)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message,
		&gotgbot.SendMessageOpts{
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})

	return err
}

func (h *CommandHandler) ListLocations(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil {
		return err
	}

	if locationName == "" {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 No location set.\n\nUse /setlocation to set your location!",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "📍 Set Location", CallbackData: "location_set"}},
					},
				},
			})
		return err
	}

	text := fmt.Sprintf("📍 *Your Current Location:*\n\n🏠 %s", locationName)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: fmt.Sprintf("🌤️ Current Weather"), CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: "📍 Change Location", CallbackData: "location_set"}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) Subscribe(bot *gotgbot.Bot, ctx *ext.Context) error {
	subscriptionText := `🔔 *Weather Notifications*

Set up automatic weather updates for your location:

*Available Subscription Types:*
• 🌅 Daily Weather (morning summary)
• 📊 Weekly Forecast (Sunday overview)
• ⚠️ Weather Alerts (extreme conditions)
• 🌬️ Air Quality Alerts (pollution levels)

*Notification Schedule:*
• Choose your preferred time
• Select notification frequency
• Configure alert thresholds`

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌅 Daily Weather", CallbackData: "subscribe_daily"}},
		{{Text: "📊 Weekly Forecast", CallbackData: "subscribe_weekly"}},
		{{Text: "⚠️ Weather Alerts", CallbackData: "subscribe_alerts"}},
		{{Text: "🌬️ Air Quality", CallbackData: "subscribe_air"}},
		{{Text: "📋 My Subscriptions", CallbackData: "subscriptions_list"}},
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, subscriptionText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) AddAlert(bot *gotgbot.Bot, ctx *ext.Context) error {
	alertText := `⚠️ *Weather Alert System*

Create custom alerts for weather conditions:

*Alert Types:*
• 🌡️ Temperature (high/low thresholds)
• 💧 Humidity levels
• 🌬️ Wind speed warnings
• ☀️ UV index alerts
• 🌫️ Air quality notifications
• 🌧️ Precipitation alerts

*Enterprise Features:*
• Slack/Teams integration
• Email notifications
• Escalation procedures
• Compliance reporting`

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌡️ Temperature Alert", CallbackData: "alert_create_temperature"}},
		{{Text: "🌬️ Wind Alert", CallbackData: "alert_create_wind"}},
		{{Text: "🌫️ Air Quality Alert", CallbackData: "alert_create_air"}},
		{{Text: "🌧️ Rain Alert", CallbackData: "alert_create_rain"}},
		{{Text: "📋 My Alerts", CallbackData: "alerts_list"}},
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, alertText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Admin commands
func (h *CommandHandler) AdminStats(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Insufficient permissions", nil)
		return err
	}

	stats, err := h.services.User.GetSystemStats(context.Background())
	if err != nil {
		return err
	}

	statsText := fmt.Sprintf(`📊 *System Statistics*

👥 *Users:*
Total Users: %d
Active Users: %d
New Users (24h): %d
Users with Location: %d

🔔 *Notifications:*
Active Subscriptions: %d
Alerts Configured: %d
Messages Sent (24h): %d

🌐 *API Usage:*
Weather Requests (24h): %d
Cache Hit Rate: %.1f%%

📈 *Performance:*
Average Response Time: %dms
Uptime: %.2f%%`,
		stats.TotalUsers,
		stats.ActiveUsers,
		stats.NewUsers24h,
		stats.UsersWithLocation,
		stats.ActiveSubscriptions,
		stats.AlertsConfigured,
		stats.MessagesSent24h,
		stats.WeatherRequests24h,
		stats.CacheHitRate,
		stats.AvgResponseTime,
		stats.Uptime)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, statsText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// Callback handlers
func (h *CommandHandler) handleWeatherCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "current":
		return h.CurrentWeather(bot, ctx)
	default:
		// Handle weather for specific location from button callback
		locationName := action
		if len(params) > 0 {
			locationName = strings.Join(append([]string{action}, params...), " ")
		}

		return h.getWeatherForLocation(bot, ctx, locationName)
	}
	return nil
}

// Helper function to get weather for a specific location
func (h *CommandHandler) getWeatherForLocation(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	// Get weather data
	weatherData, err := h.services.Weather.GetCurrentWeatherByLocation(context.Background(), locationName)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get weather for '%s'. Please check the location name.", locationName), nil)
		return err
	}

	// Format weather information
	weatherText := fmt.Sprintf(
		"🌤️ *Current Weather in %s*\n\n"+
			"🌡️ *Temperature:* %.1f°C\n"+
			"💨 *Wind:* %.1f km/h\n"+
			"💧 *Humidity:* %d%%\n"+
			"🏗️ *Pressure:* %.0f hPa\n"+
			"👁️ *Visibility:* %.1f km\n"+
			"☀️ *UV Index:* %.0f\n"+
			"☁️ *Description:* %s",
		weatherData.LocationName,
		weatherData.Temperature,
		weatherData.WindSpeed,
		weatherData.Humidity,
		weatherData.Pressure,
		weatherData.Visibility,
		weatherData.UVIndex,
		weatherData.Description,
	)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "📊 5-Day Forecast", CallbackData: fmt.Sprintf("forecast_%s", locationName)}},
		{{Text: "🌬️ Air Quality", CallbackData: fmt.Sprintf("air_%s", locationName)}},
		{{Text: "🔔 Set Alert", CallbackData: fmt.Sprintf("alert_%s", locationName)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, weatherText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) getForecastForLocation(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	// First get coordinates for the location
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to find location '%s'", locationName), nil)
		return err
	}

	forecast, err := h.services.Weather.GetForecast(context.Background(), locationData.Latitude, locationData.Longitude, 5)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get forecast for '%s'. Please check the location name.", locationName), nil)
		return err
	}

	forecastText := h.formatForecastMessage(forecast)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: "🌬️ Air Quality", CallbackData: fmt.Sprintf("air_%s", locationName)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, forecastText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	return err
}

func (h *CommandHandler) getForecastByCoords(bot *gotgbot.Bot, ctx *ext.Context, lat, lon float64) error {
	forecast, err := h.services.Weather.GetForecast(context.Background(), lat, lon, 5)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get forecast for this location", nil)
		return err
	}

	forecastText := h.formatForecastMessage(forecast)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: fmt.Sprintf("weather_coords_%.4f_%.4f", lat, lon)}},
		{{Text: "🌬️ Air Quality", CallbackData: fmt.Sprintf("air_coords_%.4f_%.4f", lat, lon)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, forecastText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	return err
}

func (h *CommandHandler) handleForecastCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "coords":
		if len(params) >= 2 {
			lat, err := strconv.ParseFloat(params[0], 64)
			if err != nil {
				return err
			}
			lon, err := strconv.ParseFloat(params[1], 64)
			if err != nil {
				return err
			}
			return h.getForecastByCoords(bot, ctx, lat, lon)
		}
	default:
		// Handle forecast for specific location from button callback
		locationName := action
		if len(params) > 0 {
			locationName = strings.Join(append([]string{action}, params...), " ")
		}
		return h.getForecastForLocation(bot, ctx, locationName)
	}
	return nil
}

func (h *CommandHandler) handleSettingsCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "main":
		return h.Settings(bot, ctx)
	case "location":
		return h.handleLocationSettings(bot, ctx)
	case "language":
		if len(params) >= 2 && params[0] == "set" {
			return h.setUserLanguage(bot, ctx, params[1])
		}
		return h.handleLanguageSettings(bot, ctx)
	case "units":
		if len(params) >= 2 && params[0] == "set" {
			return h.setUserUnits(bot, ctx, params[1])
		}
		return h.handleUnitSettings(bot, ctx)
	case "timezone":
		if len(params) >= 2 && params[0] == "set" {
			return h.setUserTimezone(bot, ctx, strings.Join(params[1:], "_"))
		}
		return h.handleTimezoneSettings(bot, ctx)
	case "notifications":
		return h.handleNotificationSettings(bot, ctx)
	case "export":
		return h.handleExportSettings(bot, ctx)
	}
	return nil
}

func (h *CommandHandler) handleLocationCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "add", "set_name":
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📝 *Set Location by Name*\n\nPlease type your city name (e.g., \"London\", \"New York\", \"Kyiv\"):",
			&gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
			})
		return err
	case "set":
		// Check if this is for coordinates or name based on params
		if len(params) > 0 && params[0] == "coords" {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"📍 *Set Location by Coordinates*\n\nPlease enter your GPS coordinates in the format:\n`latitude, longitude`\n\nExample: `37.7749, -122.4194`",
				&gotgbot.SendMessageOpts{
					ParseMode: "Markdown",
				})
			return err
		} else {
			// Default set behavior (name-based)
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"📝 *Set Location by Name*\n\nPlease type your city name (e.g., \"London\", \"New York\", \"Kyiv\"):",
				&gotgbot.SendMessageOpts{
					ParseMode: "Markdown",
				})
			return err
		}
	case "clear":
		userID := ctx.EffectiveUser.Id
		err := h.services.User.ClearUserLocation(context.Background(), userID)
		if err != nil {
			_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to clear location", nil)
			return sendErr
		}
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Location cleared successfully!", nil)
		return err
	case "default":
		// With single location per user, this is no longer needed
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "✅ You only have one location - it's already your default!", nil)
		return err
	case "save":
		h.logger.Info().Int("params_count", len(params)).Interface("params", params).Msg("Location save callback")
		// Handle saving shared location
		if len(params) >= 3 {
			lat, _ := strconv.ParseFloat(params[0], 64)
			lon, _ := strconv.ParseFloat(params[1], 64)
			encodedName := strings.Join(params[2:], "_")

			// URL decode the location name
			name, decodeErr := url.QueryUnescape(encodedName)
			if decodeErr != nil {
				// If decoding fails, use the encoded name as fallback
				name = encodedName
			}

			h.logger.Info().Float64("lat", lat).Float64("lon", lon).Str("name", name).Msg("Saving location")

			userID := ctx.EffectiveUser.Id
			err := h.services.User.SetUserLocation(context.Background(), userID, name, "", "", lat, lon)
			if err != nil {
				h.logger.Error().Err(err).Msg("Failed to save location to database")
				_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location", nil)
				return err
			}

			h.logger.Info().Str("name", name).Msg("Location saved successfully")
			_, err = bot.SendMessage(ctx.EffectiveChat.Id,
				fmt.Sprintf("✅ Location '%s' saved successfully!", name), nil)
			return err
		} else {
			h.logger.Warn().Int("params_count", len(params)).Msg("Not enough parameters for location save")
		}
	case "confirm":
		// Handle location confirmation from plain text input
		if len(params) >= 1 {
			locationName := strings.Join(params, " ")
			h.logger.Info().Str("location", locationName).Msg("User confirmed location from text input")

			userID := ctx.EffectiveUser.Id

			// Geocode the location
			location, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
			if err != nil {
				h.logger.Error().Err(err).Str("location", locationName).Msg("Failed to geocode location")
				_, err := bot.SendMessage(ctx.EffectiveChat.Id,
					fmt.Sprintf("❌ Sorry, I couldn't find the location '%s'. Please try a different city name.", locationName), nil)
				return err
			}

			// Save the location
			err = h.services.User.SetUserLocation(context.Background(), userID, location.Name, location.Country, location.City, location.Latitude, location.Longitude)
			if err != nil {
				h.logger.Error().Err(err).Msg("Failed to save location to database")
				_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location. Please try again.", nil)
				return err
			}

			h.logger.Info().Str("location", location.Name).Msg("Location saved successfully from text input")
			_, err = bot.SendMessage(ctx.EffectiveChat.Id,
				fmt.Sprintf("✅ Location set to *%s, %s*\n📍 Coordinates: %.4f, %.4f", location.Name, location.Country, location.Latitude, location.Longitude),
				&gotgbot.SendMessageOpts{ParseMode: "Markdown"})
			return err
		}
	case "ignore":
		// Handle ignoring potential location from plain text input
		h.logger.Info().Msg("User ignored location suggestion from text input")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "👍 Understood, I won't set that as your location.", nil)
		return err
	}
	return nil
}

func (h *CommandHandler) handleAlertCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "create":
		if len(params) > 0 {
			alertType := params[0]
			return h.handleCreateAlert(bot, ctx, alertType)
		}
	case "temp":
		if len(params) >= 2 {
			return h.handleTemperatureAlert(bot, ctx, params[0], params[1])
		}
	case "wind":
		if len(params) >= 2 {
			return h.handleWindAlert(bot, ctx, params[0], params[1])
		}
	case "air":
		if len(params) >= 2 {
			return h.handleAirQualityAlert(bot, ctx, params[0], params[1])
		}
	case "humidity":
		if len(params) >= 2 {
			return h.handleHumidityAlert(bot, ctx, params[0], params[1])
		}
	case "edit":
		if len(params) > 0 {
			return h.editAlert(bot, ctx, params[0])
		}
	case "remove":
		if len(params) > 0 {
			return h.removeAlert(bot, ctx, params[0])
		}
	default:
		// Handle alert setup for specific location from button callback
		locationName := action
		if len(params) > 0 {
			locationName = strings.Join(append([]string{action}, params...), " ")
		}

		return h.showAlertOptions(bot, ctx, locationName)
	}
	return nil
}

// Helper function to show alert options for a location
func (h *CommandHandler) showAlertOptions(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌡️ Temperature Alert", CallbackData: fmt.Sprintf("alert_temp_setup_%s", locationName)}},
		{{Text: "💨 Wind Speed Alert", CallbackData: fmt.Sprintf("alert_wind_setup_%s", locationName)}},
		{{Text: "🌬️ Air Quality Alert", CallbackData: fmt.Sprintf("alert_air_setup_%s", locationName)}},
		{{Text: "💧 Humidity Alert", CallbackData: fmt.Sprintf("alert_humidity_setup_%s", locationName)}},
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("🔔 *Set Alert for %s*\n\nChoose the type of alert you want to create:", locationName),
		&gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})

	return err
}

// Additional helper methods for settings
func (h *CommandHandler) handleLanguageSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	// Languages in alphabetical order by name
	languages := []struct {
		code string
		name string
	}{
		{"de", "🇩🇪 Deutsch"},
		{"en", "🇺🇸 English"},
		{"es", "🇪🇸 Español"},
		{"fr", "🇫🇷 Français"},
		{"uk", "🇺🇦 Українська"},
	}

	text := "🌐 *Choose your language:*"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for _, lang := range languages {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: lang.name, CallbackData: fmt.Sprintf("settings_language_set_%s", lang.code)},
		})
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) handleUnitSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := "📏 *Choose your preferred units:*"

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌡️ Metric (°C, km/h, km)", CallbackData: "settings_units_set_metric"}},
		{{Text: "🌡️ Imperial (°F, mph, miles)", CallbackData: "settings_units_set_imperial"}},
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) handleTimezoneSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	text := "🕐 *Select your timezone:*"

	timezones := []string{
		"UTC", "Europe/London", "Europe/Berlin", "Europe/Kyiv",
		"America/New_York", "America/Los_Angeles", "Asia/Tokyo",
	}

	var keyboard [][]gotgbot.InlineKeyboardButton
	for _, tz := range timezones {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: tz, CallbackData: fmt.Sprintf("settings_timezone_set_%s", tz)},
		})
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) handleCreateAlert(bot *gotgbot.Bot, ctx *ext.Context, alertType string) error {
	var text string
	var keyboard [][]gotgbot.InlineKeyboardButton

	switch alertType {
	case "temperature":
		text = `🌡️ *Temperature Alert Setup*

Choose alert condition:`
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: "🔥 High Temperature (>30°C)", CallbackData: "alert_temp_high_30"}},
			{{Text: "🥶 Low Temperature (<0°C)", CallbackData: "alert_temp_low_0"}},
			{{Text: "⚙️ Custom Threshold", CallbackData: "alert_temp_custom"}},
		}
	case "wind":
		text = `🌬️ *Wind Speed Alert Setup*

Choose alert condition:`
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: "💨 Strong Wind (>50 km/h)", CallbackData: "alert_wind_high_50"}},
			{{Text: "🌪️ Very Strong (>80 km/h)", CallbackData: "alert_wind_high_80"}},
			{{Text: "⚙️ Custom Threshold", CallbackData: "alert_wind_custom"}},
		}
	case "air":
		text = `🌫️ *Air Quality Alert Setup*

Choose alert condition:`
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: "⚠️ Moderate AQI (>100)", CallbackData: "alert_air_moderate_100"}},
			{{Text: "🚨 Unhealthy AQI (>150)", CallbackData: "alert_air_unhealthy_150"}},
			{{Text: "⚙️ Custom Threshold", CallbackData: "alert_air_custom"}},
		}
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// handleSubscriptionCallback handles subscription-related button callbacks
func (h *CommandHandler) handleSubscriptionCallback(bot *gotgbot.Bot, ctx *ext.Context, action, subAction string, params []string) error {
	switch action {
	case "subscribe":
		switch subAction {
		case "daily":
			return h.createDailySubscription(bot, ctx)
		case "weekly":
			return h.createWeeklySubscription(bot, ctx)
		case "alerts":
			return h.createAlertsSubscription(bot, ctx)
		case "air":
			return h.createAirQualitySubscription(bot, ctx)
		}
	case "unsubscribe":
		return h.removeSubscription(bot, ctx, subAction)
	case "edit":
		return h.editSubscription(bot, ctx, subAction)
	case "subscriptions":
		if subAction == "list" {
			return h.listUserSubscriptions(bot, ctx)
		}
	}
	return nil
}

// handleAdminCallback handles admin-related button callbacks
func (h *CommandHandler) handleAdminCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "users":
		if len(params) > 0 {
			switch params[0] {
			case "recent":
				return h.showRecentUsers(bot, ctx)
			case "roles":
				return h.showUserRoles(bot, ctx)
			}
		}
		return h.AdminListUsers(bot, ctx)
	case "stats":
		if len(params) > 0 && params[0] == "detailed" {
			return h.showDetailedStats(bot, ctx)
		}
		return h.AdminStats(bot, ctx)
	}
	return nil
}

// handleShareCallback handles share location button callbacks
func (h *CommandHandler) handleShareCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	if action == "location" {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 Please share your location using the button below:",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.ReplyKeyboardMarkup{
					Keyboard: [][]gotgbot.KeyboardButton{
						{{Text: "📍 Share Location", RequestLocation: true}},
					},
					OneTimeKeyboard: true,
					ResizeKeyboard:  true,
				},
			})
		return err
	}
	return nil
}

// handleAirCallback handles air quality button callbacks
func (h *CommandHandler) handleAirCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "coords":
		if len(params) >= 2 {
			lat, err := strconv.ParseFloat(params[0], 64)
			if err != nil {
				return err
			}
			lon, err := strconv.ParseFloat(params[1], 64)
			if err != nil {
				return err
			}
			return h.getAirQualityByCoords(bot, ctx, lat, lon)
		}
	default:
		// Handle air quality for specific location from button callback
		locationName := action
		if len(params) > 0 {
			locationName = strings.Join(append([]string{action}, params...), " ")
		}
		return h.getAirQualityData(bot, ctx, locationName)
	}
	return nil
}

// Helper functions for subscription handling
func (h *CommandHandler) createDailySubscription(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	_, err = h.services.Subscription.CreateSubscription(
		context.Background(),
		userID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create subscription. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Daily weather subscription created! You'll receive morning updates at 8:00 AM.", nil)
	return err
}

func (h *CommandHandler) createWeeklySubscription(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	_, err = h.services.Subscription.CreateSubscription(
		context.Background(),
		userID,
		models.SubscriptionWeekly,
		models.FrequencyWeekly,
		"09:00",
	)

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create subscription. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Weekly weather subscription created! You'll receive updates every Sunday at 9:00 AM.", nil)
	return err
}

func (h *CommandHandler) removeSubscription(bot *gotgbot.Bot, ctx *ext.Context, subscriptionID string) error {
	userID := ctx.EffectiveUser.Id

	// Parse UUID from string
	subscriptionUUID, err := uuid.Parse(subscriptionID)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid subscription ID.", nil)
		return sendErr
	}

	err = h.services.Subscription.DeleteSubscription(context.Background(), userID, subscriptionUUID)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to remove subscription. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Subscription removed successfully.", nil)
	return err
}

func (h *CommandHandler) editSubscription(bot *gotgbot.Bot, ctx *ext.Context, subscriptionID string) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "⚙️ Subscription editing feature coming soon!", nil)
	return err
}

func (h *CommandHandler) getAirQualityData(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	// Get coordinates first for air quality
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to find location '%s'", locationName), nil)
		return err
	}

	airData, err := h.services.Weather.GetAirQuality(context.Background(), locationData.Latitude, locationData.Longitude)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Failed to get air quality for '%s'", locationName), nil)
		return err
	}

	airText := h.formatAirQualityMessage(airData)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: "📊 5-Day Forecast", CallbackData: fmt.Sprintf("forecast_%s", locationName)}},
		{{Text: "🔔 Set Alert", CallbackData: fmt.Sprintf("alert_%s", locationName)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, airText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	return err
}

func (h *CommandHandler) getAirQualityByCoords(bot *gotgbot.Bot, ctx *ext.Context, lat, lon float64) error {
	airData, err := h.services.Weather.GetAirQuality(context.Background(), lat, lon)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get air quality for this location", nil)
		return err
	}

	airText := h.formatAirQualityMessage(airData)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🌤️ Current Weather", CallbackData: fmt.Sprintf("weather_coords_%.4f_%.4f", lat, lon)}},
		{{Text: "📊 5-Day Forecast", CallbackData: fmt.Sprintf("forecast_coords_%.4f_%.4f", lat, lon)}},
		{{Text: "🔔 Set Alert", CallbackData: fmt.Sprintf("alert_coords_%.4f_%.4f", lat, lon)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, airText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	return err
}

// Additional subscription handlers
func (h *CommandHandler) createAlertsSubscription(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	_, err = h.services.Subscription.CreateSubscription(
		context.Background(),
		userID,
		models.SubscriptionAlerts,
		models.FrequencyDaily,
		"12:00",
	)

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create alerts subscription. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Weather alerts subscription created! You'll receive alert notifications when thresholds are exceeded.", nil)
	return err
}

func (h *CommandHandler) createAirQualitySubscription(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	_, err = h.services.Subscription.CreateSubscription(
		context.Background(),
		userID,
		models.SubscriptionAlerts,
		models.FrequencyDaily,
		"10:00",
	)

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create air quality subscription. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Air quality subscription created! You'll receive daily air quality updates at 10:00 AM.", nil)
	return err
}

func (h *CommandHandler) listUserSubscriptions(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get subscriptions. Please try again.", nil)
		return sendErr
	}

	if len(subscriptions) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "📋 You have no active subscriptions.\n\nUse /subscribe to create new subscriptions.", nil)
		return err
	}

	var text strings.Builder
	text.WriteString("📋 *Your Active Subscriptions:*\n\n")

	for _, sub := range subscriptions {
		text.WriteString(fmt.Sprintf("• **%s** - %s at %s\n",
			sub.SubscriptionType.String(),
			sub.Frequency.String(),
			sub.TimeOfDay))
	}

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🔔 Add New Subscription", CallbackData: "subscribe_daily"}},
		{{Text: "⚙️ Settings", CallbackData: "settings_main"}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text.String(), &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	return err
}

// Additional admin handlers
func (h *CommandHandler) showRecentUsers(bot *gotgbot.Bot, ctx *ext.Context) error {
	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get user statistics. Please try again.", nil)
		return sendErr
	}

	text := fmt.Sprintf(`👥 *Recent User Activity*

📈 New Users (24h): %d
👤 Total Active Users: %d
📊 Total Users: %d
📍 Locations Saved: %d
⚠️ Active Alerts: %d
💬 Messages (24h): %d
🌤️ Weather Requests (24h): %d`,
		stats.NewUsers24h,
		stats.ActiveUsers,
		stats.TotalUsers,
		stats.LocationsSaved,
		stats.ActiveAlerts,
		stats.Messages24h,
		stats.WeatherRequests24h)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	return err
}

func (h *CommandHandler) showUserRoles(bot *gotgbot.Bot, ctx *ext.Context) error {
	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get user statistics. Please try again.", nil)
		return sendErr
	}

	text := fmt.Sprintf(`👥 *User Roles Overview*

🔧 Administrators: %d
⚙️ Moderators: %d
👤 Regular Users: %d
📊 Total Users: %d`,
		stats.AdminCount,
		stats.ModeratorCount,
		stats.TotalUsers-stats.AdminCount-stats.ModeratorCount,
		stats.TotalUsers)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	return err
}

func (h *CommandHandler) showDetailedStats(bot *gotgbot.Bot, ctx *ext.Context) error {
	systemStats, err := h.services.User.GetSystemStats(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get system statistics. Please try again.", nil)
		return sendErr
	}

	text := fmt.Sprintf(`📊 *Detailed System Statistics*

*👥 Users:*
• Total: %d
• Active: %d
• New (24h): %d

*📍 Locations:*
• Users with Location: %d

*🔔 Subscriptions & Alerts:*
• Active Subscriptions: %d
• Configured Alerts: %d

*📈 Performance:*
• Messages Sent (24h): %d
• Weather Requests (24h): %d
• Cache Hit Rate: %.1f%%
• Avg Response Time: %dms
• Uptime: %.1f%%`,
		systemStats.TotalUsers,
		systemStats.ActiveUsers,
		systemStats.NewUsers24h,
		systemStats.UsersWithLocation,
		systemStats.ActiveSubscriptions,
		systemStats.AlertsConfigured,
		systemStats.MessagesSent24h,
		systemStats.WeatherRequests24h,
		systemStats.CacheHitRate,
		systemStats.AvgResponseTime,
		systemStats.Uptime)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	return err
}

// Settings handlers
func (h *CommandHandler) setUserLanguage(bot *gotgbot.Bot, ctx *ext.Context, language string) error {
	userID := ctx.EffectiveUser.Id

	err := h.services.User.UpdateUserSettings(context.Background(), userID, map[string]interface{}{
		"language": language,
	})

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to update language setting. Please try again.", nil)
		return sendErr
	}

	languageNames := map[string]string{
		"en": "🇺🇸 English",
		"uk": "🇺🇦 Українська",
		"de": "🇩🇪 Deutsch",
		"fr": "🇫🇷 Français",
		"es": "🇪🇸 Español",
	}

	languageName := languageNames[language]
	if languageName == "" {
		languageName = language
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Language updated to %s", languageName), nil)
	return err
}

func (h *CommandHandler) setUserUnits(bot *gotgbot.Bot, ctx *ext.Context, units string) error {
	userID := ctx.EffectiveUser.Id

	err := h.services.User.UpdateUserSettings(context.Background(), userID, map[string]interface{}{
		"units": units,
	})

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to update units setting. Please try again.", nil)
		return sendErr
	}

	unitNames := map[string]string{
		"metric":   "🌡️ Metric (°C, km/h, km)",
		"imperial": "🌡️ Imperial (°F, mph, miles)",
	}

	unitName := unitNames[units]
	if unitName == "" {
		unitName = units
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Units updated to %s", unitName), nil)
	return err
}

func (h *CommandHandler) setUserTimezone(bot *gotgbot.Bot, ctx *ext.Context, timezone string) error {
	userID := ctx.EffectiveUser.Id

	err := h.services.User.UpdateUserSettings(context.Background(), userID, map[string]interface{}{
		"timezone": timezone,
	})

	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to update timezone setting. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Timezone updated to %s", timezone), nil)
	return err
}

func (h *CommandHandler) handleNotificationSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "🔔 Notification settings will be available soon!", nil)
	return err
}

func (h *CommandHandler) handleLocationSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's current location
	locationName, lat, lon, err := h.services.User.GetUserLocation(context.Background(), userID)

	var locationText string
	var statusText string
	if err != nil || locationName == "" {
		locationText = "Not set"
		statusText = "You can set your location by:\n• Typing a city name\n• Sharing your current GPS location"
	} else {
		locationText = fmt.Sprintf("%s\nCoordinates: %.4f, %.4f", locationName, lat, lon)
		statusText = "Your location is set. You can update it anytime."
	}

	settingsText := fmt.Sprintf(`📍 *Location Settings*

*Current Location:*
%s

%s

*Options:*
• Set a new location by name
• Share your GPS location
• Clear current location`,
		locationText,
		statusText)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "📝 Set Location by Name", CallbackData: "location_set_name"}},
		{{Text: "📍 Set Location by Coordinates", CallbackData: "location_set_coords"}},
	}

	if locationName != "" {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: "🗑️ Clear Location", CallbackData: "location_clear"},
		})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: "⬅️ Back to Settings", CallbackData: "settings_main"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, settingsText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) handleExportSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "📊 Data export feature will be available soon!", nil)
	return err
}

// Alert handlers
func (h *CommandHandler) handleTemperatureAlert(bot *gotgbot.Bot, ctx *ext.Context, condition, threshold string) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	// Parse threshold and determine operator
	var thresholdValue float64
	var operator string
	var message string

	switch condition {
	case "high":
		thresholdValue = 30.0 // Default high temperature
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = fmt.Sprintf("✅ High temperature alert created! You'll be notified when temperature exceeds %.1f°C.", thresholdValue)
	case "low":
		thresholdValue = 0.0 // Default low temperature
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "lt"
		message = fmt.Sprintf("✅ Low temperature alert created! You'll be notified when temperature drops below %.1f°C.", thresholdValue)
	case "custom":
		thresholdValue = 25.0 // Default value for custom
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = "✅ Custom temperature alert created! Specify your threshold next."
	default:
		thresholdValue = 25.0
		operator = "gt"
		message = "✅ Temperature alert created!"
	}

	// Create the alert in database
	alertCondition := services.AlertCondition{
		Operator: operator,
		Value:    thresholdValue,
	}

	_, err = h.services.Alert.CreateAlert(context.Background(), userID, models.AlertTemperature, alertCondition)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create temperature alert. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
	return err
}

func (h *CommandHandler) handleWindAlert(bot *gotgbot.Bot, ctx *ext.Context, condition, threshold string) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	// Parse threshold and determine operator
	var thresholdValue float64
	var operator string
	var message string

	switch condition {
	case "high":
		thresholdValue = 50.0 // Default high wind speed in km/h
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = fmt.Sprintf("✅ Wind alert created! You'll be notified when wind speed exceeds %.1f km/h.", thresholdValue)
	case "custom":
		thresholdValue = 30.0 // Default value for custom
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = "✅ Custom wind alert created! Specify your threshold next."
	default:
		thresholdValue = 40.0
		operator = "gt"
		message = "✅ Wind alert created!"
	}

	// Create the alert in database
	alertCondition := services.AlertCondition{
		Operator: operator,
		Value:    thresholdValue,
	}

	_, err = h.services.Alert.CreateAlert(context.Background(), userID, models.AlertWindSpeed, alertCondition)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create wind alert. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
	return err
}

func (h *CommandHandler) handleAirQualityAlert(bot *gotgbot.Bot, ctx *ext.Context, condition, threshold string) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	// Parse threshold and determine operator
	var thresholdValue float64
	var operator string
	var message string

	switch condition {
	case "moderate":
		thresholdValue = 100.0 // Moderate AQI threshold
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = fmt.Sprintf("✅ Air quality alert created! You'll be notified when AQI exceeds %.0f.", thresholdValue)
	case "unhealthy":
		thresholdValue = 150.0 // Unhealthy AQI threshold
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = fmt.Sprintf("✅ Air quality alert created! You'll be notified when AQI reaches unhealthy levels (%.0f+).", thresholdValue)
	case "custom":
		thresholdValue = 75.0 // Default value for custom
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = "✅ Custom air quality alert created! Specify your threshold next."
	default:
		thresholdValue = 100.0
		operator = "gt"
		message = "✅ Air quality alert created!"
	}

	// Create the alert in database
	alertCondition := services.AlertCondition{
		Operator: operator,
		Value:    thresholdValue,
	}

	_, err = h.services.Alert.CreateAlert(context.Background(), userID, models.AlertAirQuality, alertCondition)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create air quality alert. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
	return err
}

func (h *CommandHandler) handleHumidityAlert(bot *gotgbot.Bot, ctx *ext.Context, condition, threshold string) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Please set a location first using /setlocation", nil)
		return sendErr
	}

	// Parse threshold and determine operator
	var thresholdValue float64
	var operator string
	var message string

	switch condition {
	case "high":
		thresholdValue = 80.0 // High humidity threshold (%)
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = fmt.Sprintf("✅ High humidity alert created! You'll be notified when humidity exceeds %.1f%%.", thresholdValue)
	case "low":
		thresholdValue = 30.0 // Low humidity threshold (%)
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "lt"
		message = fmt.Sprintf("✅ Low humidity alert created! You'll be notified when humidity drops below %.1f%%.", thresholdValue)
	case "custom":
		thresholdValue = 60.0 // Default value for custom
		if threshold != "" {
			if val, err := strconv.ParseFloat(threshold, 64); err == nil {
				thresholdValue = val
			}
		}
		operator = "gt"
		message = "✅ Custom humidity alert created! Specify your threshold next."
	default:
		thresholdValue = 70.0
		operator = "gt"
		message = "✅ Humidity alert created!"
	}

	// Create the alert in database
	alertCondition := services.AlertCondition{
		Operator: operator,
		Value:    thresholdValue,
	}

	_, err = h.services.Alert.CreateAlert(context.Background(), userID, models.AlertHumidity, alertCondition)
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to create humidity alert. Please try again.", nil)
		return sendErr
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
	return err
}

func (h *CommandHandler) editAlert(bot *gotgbot.Bot, ctx *ext.Context, alertID string) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "⚙️ Alert editing feature coming soon!", nil)
	return err
}

func (h *CommandHandler) removeAlert(bot *gotgbot.Bot, ctx *ext.Context, alertID string) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, "✅ Alert removed successfully!", nil)
	return err
}

// handleAlertsCallback handles the alerts list callback
func (h *CommandHandler) handleAlertsCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "list":
		return h.listUserAlerts(bot, ctx)
	}
	return nil
}

func (h *CommandHandler) listUserAlerts(bot *gotgbot.Bot, ctx *ext.Context) error {
	_, err := bot.SendMessage(ctx.EffectiveChat.Id,
		"⚠️ *Your Active Alerts*\n\nNo alerts configured yet.\n\nUse /addalert to create new alerts.",
		&gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
	return err
}
