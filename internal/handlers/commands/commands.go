package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/hbollon/go-edlib"
	"github.com/rs/zerolog"

	"github.com/valpere/shopogoda/internal"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/weather"
)

type CommandHandler struct {
	services *services.Services
	logger   *zerolog.Logger
}

// availableCommands lists all available bot commands
var availableCommands = []string{
	"start", "help", "settings", "language", "version",
	"weather", "forecast", "air",
	"setlocation",
	"subscribe", "unsubscribe", "subscriptions",
	"addalert", "alerts", "removealert",
	"stats", "broadcast", "users", "demoreset", "democlear",
}

func New(services *services.Services, logger *zerolog.Logger) *CommandHandler {
	return &CommandHandler{
		services: services,
		logger:   logger,
	}
}

// commandSuggestion represents a command with its edit distance
type commandSuggestion struct {
	command  string
	distance int
}

// suggestSimilarCommands finds commands similar to the input using edit distance
func suggestSimilarCommands(input string, maxSuggestions int) []string {
	// Remove leading slash if present
	input = strings.TrimPrefix(input, "/")

	suggestions := make([]commandSuggestion, 0)

	// Calculate edit distance for each command
	for _, cmd := range availableCommands {
		distance := edlib.LevenshteinDistance(input, cmd)
		// Only suggest commands with reasonable edit distance (less than half the command length)
		if distance <= len(cmd)/2+1 {
			suggestions = append(suggestions, commandSuggestion{
				command:  cmd,
				distance: distance,
			})
		}
	}

	// Sort by distance (ascending), then alphabetically
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].distance == suggestions[j].distance {
			return suggestions[i].command < suggestions[j].command
		}
		return suggestions[i].distance < suggestions[j].distance
	})

	// Limit to maxSuggestions
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	// Extract command names
	result := make([]string, len(suggestions))
	for i, s := range suggestions {
		result[i] = s.command
	}

	return result
}

// getUserLanguage gets the user's language preference or returns default
func (h *CommandHandler) getUserLanguage(ctx context.Context, userID int64) string {
	user, err := h.services.User.GetUser(ctx, userID)
	if err != nil || user == nil || user.Language == "" {
		return internal.DefaultLanguage // default to English
	}
	return user.Language
}

// ensureUserRegistered ensures the user is registered, auto-registering if needed
// Returns true if user was just registered (new user), false if already existed
func (h *CommandHandler) ensureUserRegistered(ctx context.Context, user *gotgbot.User) bool {
	dbUser, err := h.services.User.GetUser(ctx, user.Id)
	if err != nil || dbUser == nil {
		// User not found, auto-register
		if err := h.services.User.RegisterUser(ctx, user); err != nil {
			h.logger.Error().Err(err).Int64("user_id", user.Id).Msg("Failed to auto-register user")
		}
		return true // New user
	}
	return false // Existing user
}

// Start command handler
func (h *CommandHandler) Start(bot *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser

	// Track message in Redis
	if err := h.services.User.IncrementMessageCounter(context.Background()); err != nil {
		h.logger.Warn().Err(err).Msg("Failed to increment message counter")
	}

	// Debug logging for start command
	h.logger.Debug().
		Int64("user_id", user.Id).
		Int("args_count", len(ctx.Args())).
		Msg("Starting Start command")

	// Register or update user
	if err := h.services.User.RegisterUser(context.Background(), user); err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.Id).Msg("Failed to register user")
	}

	// Get user's language preference
	userLang := h.getUserLanguage(context.Background(), user.Id)

	// Get localized welcome message
	welcomeText := h.services.Localization.T(context.Background(), userLang, "welcome_message")

	// Get localized button texts
	weatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	settingsBtn := h.services.Localization.T(context.Background(), userLang, "button_settings")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: weatherBtn, CallbackData: "weather_current"}},
		{{Text: settingsBtn, CallbackData: "settings_main"}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Get localized help text components
	title := h.services.Localization.T(context.Background(), userLang, "help_title")
	basicCmd := h.services.Localization.T(context.Background(), userLang, "help_basic_commands")
	weather := h.services.Localization.T(context.Background(), userLang, "help_weather")
	forecast := h.services.Localization.T(context.Background(), userLang, "help_forecast")
	air := h.services.Localization.T(context.Background(), userLang, "help_air")

	locationMgmt := h.services.Localization.T(context.Background(), userLang, "help_location_management")
	setLocation := h.services.Localization.T(context.Background(), userLang, "help_setlocation")

	notifications := h.services.Localization.T(context.Background(), userLang, "help_notifications")
	subscribe := h.services.Localization.T(context.Background(), userLang, "help_subscribe")
	unsubscribe := h.services.Localization.T(context.Background(), userLang, "help_unsubscribe")
	subscriptions := h.services.Localization.T(context.Background(), userLang, "help_subscriptions")

	alerts := h.services.Localization.T(context.Background(), userLang, "help_alerts")
	addAlert := h.services.Localization.T(context.Background(), userLang, "help_addalert")
	viewAlerts := h.services.Localization.T(context.Background(), userLang, "help_view_alerts")
	removeAlert := h.services.Localization.T(context.Background(), userLang, "help_removealert")

	settings := h.services.Localization.T(context.Background(), userLang, "help_settings")
	settingsDesc := h.services.Localization.T(context.Background(), userLang, "help_settings_desc")
	dataExport := h.services.Localization.T(context.Background(), userLang, "help_data_export")

	exportFeatures := h.services.Localization.T(context.Background(), userLang, "help_export_features")
	weatherData := h.services.Localization.T(context.Background(), userLang, "help_export_weather")
	alertHistory := h.services.Localization.T(context.Background(), userLang, "help_export_alerts")
	notifSubs := h.services.Localization.T(context.Background(), userLang, "help_export_subscriptions")
	completeExport := h.services.Localization.T(context.Background(), userLang, "help_export_complete")

	adminCmd := h.services.Localization.T(context.Background(), userLang, "help_admin_commands")
	stats := h.services.Localization.T(context.Background(), userLang, "help_stats")
	broadcast := h.services.Localization.T(context.Background(), userLang, "help_broadcast")
	users := h.services.Localization.T(context.Background(), userLang, "help_users")

	proTips := h.services.Localization.T(context.Background(), userLang, "help_pro_tips")
	tip1 := h.services.Localization.T(context.Background(), userLang, "help_tip_location")
	tip2 := h.services.Localization.T(context.Background(), userLang, "help_tip_alerts")
	tip3 := h.services.Localization.T(context.Background(), userLang, "help_tip_export")
	tip4 := h.services.Localization.T(context.Background(), userLang, "help_tip_timezone")
	tip5 := h.services.Localization.T(context.Background(), userLang, "help_tip_separation")

	support := h.services.Localization.T(context.Background(), userLang, "help_support")

	// Build help text with localized content
	helpText := fmt.Sprintf(`🌤️ *%s*

*🏠 %s:*
/weather \[location] - %s
/forecast \[location] - %s
/air \[location] - %s

*📍 %s:*
/setlocation - %s

*🔔 %s:*
/subscribe - %s
/unsubscribe - %s
/subscriptions - %s

*⚠️ %s:*
/addalert - %s
/alerts - %s
/removealert <id> - %s

*⚙️ %s:*
/settings - %s
• %s

*📊 %s:*
%s:
• 🌤️ %s
• ⚠️ %s
• 📋 %s
• 📦 %s

*👨‍💼 %s:*
/stats - %s
/broadcast - %s
/users - %s

*💡 %s:*
• %s
• %s
• %s
• %s
• %s

*🆘 %s:*
https://github.com/valpere/shopogoda/issues`,
		title, basicCmd, weather, forecast, air,
		locationMgmt, setLocation,
		notifications, subscribe, unsubscribe, subscriptions,
		alerts, addAlert, viewAlerts, removeAlert,
		settings, settingsDesc, dataExport,
		exportFeatures, exportFeatures,
		weatherData, alertHistory, notifSubs, completeExport,
		adminCmd, stats, broadcast, users,
		proTips, tip1, tip2, tip3, tip4, tip5,
		support)

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

		location = h.parseLocationFromArgs(ctx)
	}

	h.logger.Debug().
		Str("parsed_location", location).
		Msg("Parsed location parameter")

	// If no location provided, use user's saved location or ask for it
	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			userLang := h.getUserLanguage(context.Background(), userID)
			message := h.services.Localization.T(context.Background(), userLang, "weather_location_needed")

			// Show 3-button dialog (same as /setlocation)
			setByNameBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_name")
			setCoordsBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_coords")
			backBtn := h.services.Localization.T(context.Background(), userLang, "button_back_to_start")

			keyboard := [][]gotgbot.InlineKeyboardButton{
				{{Text: setByNameBtn, CallbackData: "location_set_name"}},
				{{Text: setCoordsBtn, CallbackData: "location_set_coords"}},
				{{Text: backBtn, CallbackData: "back_to_start"}},
			}

			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				message,
				&gotgbot.SendMessageOpts{
					ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: keyboard,
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

		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "weather_error", location)

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	h.logger.Debug().
		Str("location", location).
		Msg("Successfully got weather data")

	// Track weather request in Redis
	if err := h.services.User.IncrementWeatherRequestCounter(context.Background()); err != nil {
		h.logger.Warn().Err(err).Msg("Failed to increment weather request counter")
	}

	// Format weather message
	userLang := h.getUserLanguage(context.Background(), userID)
	weatherText := h.formatWeatherMessage(weatherData, userLang)

	// Get localized button texts
	forecastBtn := h.services.Localization.T(context.Background(), userLang, "button_forecast")
	airQualityBtn := h.services.Localization.T(context.Background(), userLang, "button_air_quality")
	setAlertBtn := h.services.Localization.T(context.Background(), userLang, "button_set_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: forecastBtn, CallbackData: fmt.Sprintf("forecast_%s", location)}},
		{{Text: airQualityBtn, CallbackData: fmt.Sprintf("air_%s", location)}},
		{{Text: setAlertBtn, CallbackData: fmt.Sprintf("alert_%s", location)}},
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

	location := h.parseLocationFromArgs(ctx)

	h.logger.Info().
		Str("parsed_location", location).
		Msg("FORECAST_DEBUG: Parsed location parameter")

	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			userLang := h.getUserLanguage(context.Background(), userID)
			message := h.services.Localization.T(context.Background(), userLang, "forecast_location_needed")

			_, err := bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
			return err
		}
		location = locationName
	}

	// Get coordinates first for forecast
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), location)
	if err != nil {
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_not_found", location)

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	forecast, err := h.services.Weather.GetForecast(context.Background(), locationData.Latitude, locationData.Longitude, 5)
	if err != nil {
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "forecast_error", location)

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Track weather request in Redis
	if err := h.services.User.IncrementWeatherRequestCounter(context.Background()); err != nil {
		h.logger.Warn().Err(err).Msg("Failed to increment weather request counter")
	}

	userLang := h.getUserLanguage(context.Background(), userID)
	forecastText := h.formatForecastMessage(forecast, userLang)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, forecastText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// Air quality command
func (h *CommandHandler) AirQuality(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	location := h.parseLocationFromArgs(ctx)
	userLang := h.getUserLanguage(context.Background(), userID)

	if location == "" {
		locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
		if err != nil || locationName == "" {
			message := h.services.Localization.T(context.Background(), userLang, "air_location_needed")

			_, err := bot.SendMessage(ctx.EffectiveChat.Id, message, nil)
			return err
		}
		location = locationName
	}

	// Get coordinates first for air quality
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), location)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_not_found", location)

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	airData, err := h.services.Weather.GetAirQuality(context.Background(), locationData.Latitude, locationData.Longitude)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "air_quality_error", location)

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Track weather request in Redis
	if err := h.services.User.IncrementWeatherRequestCounter(context.Background()); err != nil {
		h.logger.Warn().Err(err).Msg("Failed to increment weather request counter")
	}

	airText := h.formatAirQualityMessage(airData, userLang)

	// Get localized button texts
	weatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	alertBtn := h.services.Localization.T(context.Background(), userLang, "button_set_air_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: weatherBtn, CallbackData: fmt.Sprintf("weather_%s", location)}},
		{{Text: alertBtn, CallbackData: fmt.Sprintf("air_alert_%s", location)}},
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

	userLang := h.getUserLanguage(context.Background(), userID)

	// Get user's current location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	var locationText string
	if err != nil || locationName == "" {
		locationText = h.services.Localization.T(context.Background(), userLang, "settings_not_set")
	} else {
		locationText = locationName
	}

	// Get localized strings
	title := h.services.Localization.T(context.Background(), userLang, "settings_title")
	currentConfig := h.services.Localization.T(context.Background(), userLang, "settings_current_config")
	location := h.services.Localization.T(context.Background(), userLang, "settings_location")
	language := h.services.Localization.T(context.Background(), userLang, "settings_language")
	units := h.services.Localization.T(context.Background(), userLang, "settings_units")
	timezone := h.services.Localization.T(context.Background(), userLang, "settings_timezone")
	role := h.services.Localization.T(context.Background(), userLang, "settings_role")
	status := h.services.Localization.T(context.Background(), userLang, "settings_status")

	availableSettings := h.services.Localization.T(context.Background(), userLang, "settings_available")
	locationMgmt := h.services.Localization.T(context.Background(), userLang, "settings_location_mgmt")
	langPrefs := h.services.Localization.T(context.Background(), userLang, "settings_lang_prefs")
	unitSystem := h.services.Localization.T(context.Background(), userLang, "settings_unit_system")
	timezoneSettings := h.services.Localization.T(context.Background(), userLang, "settings_timezone_settings")
	notifPrefs := h.services.Localization.T(context.Background(), userLang, "settings_notif_prefs")
	dataExport := h.services.Localization.T(context.Background(), userLang, "settings_data_export")

	// Get localized values
	unitsText := h.getLocalizedUnitsText(context.Background(), userLang, user.Units)
	roleText := h.getLocalizedRoleName(context.Background(), userLang, user.Role)
	statusText := h.getLocalizedStatusText(context.Background(), userLang, user.IsActive)

	settingsText := fmt.Sprintf(`⚙️ *%s*

*%s:*
%s: %s
%s: %s
%s: %s
%s: %s
%s: %s
%s: %s

*%s:*
• %s
• %s
• %s
• %s
• %s
• %s`,
		title,
		currentConfig,
		location, locationText,
		language, user.Language,
		units, unitsText,
		timezone, user.Timezone,
		role, roleText,
		status, statusText,
		availableSettings,
		locationMgmt, langPrefs, unitSystem, timezoneSettings, notifPrefs, dataExport)

	// Get localized button texts
	setLocationBtn := h.services.Localization.T(context.Background(), userLang, "button_set_location")
	languageBtn := h.services.Localization.T(context.Background(), userLang, "button_language")
	unitsBtn := h.services.Localization.T(context.Background(), userLang, "button_units")
	timezoneBtn := h.services.Localization.T(context.Background(), userLang, "button_timezone")
	notificationsBtn := h.services.Localization.T(context.Background(), userLang, "button_notifications")
	exportBtn := h.services.Localization.T(context.Background(), userLang, "button_data_export")
	backBtn := h.services.Localization.T(context.Background(), userLang, "button_back_to_start")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: setLocationBtn, CallbackData: "settings_location"}},
		{{Text: languageBtn, CallbackData: "settings_language"}},
		{{Text: unitsBtn, CallbackData: "settings_units"}},
		{{Text: timezoneBtn, CallbackData: "settings_timezone"}},
		{{Text: notificationsBtn, CallbackData: "settings_notifications"}},
		{{Text: exportBtn, CallbackData: "settings_export"}},
		{{Text: backBtn, CallbackData: "settings_start"}},
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
	case "timezone":
		return h.handleTimezoneCallback(bot, ctx, subAction, parts[2:])
	case "language":
		return h.handleLanguageCallback(bot, ctx, subAction, parts[2:])
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
	case "air":
		return h.handleAirCallback(bot, ctx, subAction, parts[2:])
	case "notifications":
		return h.handleNotificationCallback(bot, ctx, subAction, parts[2:])
	case "export":
		return h.handleExportCallback(bot, ctx, subAction, parts[2:])
	case "back":
		return h.handleBackCallback(bot, ctx, subAction, parts[2:])
	case "role":
		return h.handleRoleCallback(bot, ctx, subAction, parts[2:])
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
	userLang := h.getUserLanguage(context.Background(), user.Id)
	weatherData, err := h.services.Weather.GetCurrentWeatherByCoords(context.Background(), lat, lon)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_weather_location_failed")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	weatherText := h.formatWeatherMessage(weatherData, userLang)

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

	// Check if this looks like a timezone first - use very specific patterns to avoid conflicts
	// Only match specific timezone formats that are unlikely to be city names:
	// 1. Region/City format (Europe/London, America/New_York)
	// 2. UTC/GMT with offsets (UTC+1, GMT-5)
	// 3. Common timezone abbreviations (UTC, GMT, EST, PST, etc.)
	timezonePattern := `^(?:UTC|GMT|[A-Z]{3,4}(?:[+-]\d{1,2})?|[A-Za-z_]+/[A-Za-z_]+|(?:UTC|GMT)[+-]\d{1,2})$`
	timezoneRe := regexp.MustCompile(timezonePattern)
	if timezoneRe.MatchString(text) {
		// Additional check: if it contains a slash, it's likely a timezone (Europe/London)
		// If it's UTC, GMT, or other timezone abbreviations, it's definitely a timezone
		if strings.Contains(text, "/") ||
			strings.HasPrefix(text, "UTC") ||
			strings.HasPrefix(text, "GMT") ||
			regexp.MustCompile(`^[A-Z]{3,4}([+-]\d{1,2})?$`).MatchString(text) {
			h.logger.Info().Str("input", text).Msg("Detected potential timezone input from text message")
			return h.handleTimezoneInput(bot, ctx, text)
		}
	}

	// Simple heuristics to detect if this might be a location name
	// - Should be 2-50 characters
	// - Should contain only letters, spaces, hyphens, apostrophes
	// - Should not be too short (avoid "ok", "yes", etc.)
	if len(text) >= 2 && len(text) <= 50 {
		// Check if text looks like a location name (letters, spaces, hyphens, apostrophes only)
		locationPattern := `^[a-zA-ZÀ-ÿ\s\-']+$`
		matched, _ := regexp.MatchString(locationPattern, text)
		if matched {
			// Skip common non-location words
			commonWords := map[string]bool{
				"ok": true, "yes": true, "no": true, "hi": true, "hello": true,
				"thanks": true, "thank you": true, "good": true, "bad": true,
				"help": true, "stop": true, "cancel": true, "back": true,
			}
			if !commonWords[strings.ToLower(text)] {
				h.logger.Info().Str("input", text).Msg("Detected potential location input from text message")
				// Use shared confirmation logic
				return h.showLocationConfirmation(bot, ctx, text)
			}
		}
	}

	return nil
}

// parseLocationFromArgs extracts location from command arguments or returns empty string
func (h *CommandHandler) parseLocationFromArgs(ctx *ext.Context) string {
	if len(ctx.Args()) > 1 {
		return strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))
	}
	return ""
}

// showLocationConfirmation displays a confirmation dialog for setting/changing location
func (h *CommandHandler) showLocationConfirmation(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check if user already has a location set
	existingLocation, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)

	var messageText string
	var keyboard [][]gotgbot.InlineKeyboardButton

	backText := h.services.Localization.T(context.Background(), userLang, "button_back_to_settings")

	if err != nil || existingLocation == "" {
		// No location set - offer to set this as their location
		messageText = h.services.Localization.T(context.Background(), userLang, "location_confirm_set", locationName)
		yesText := h.services.Localization.T(context.Background(), userLang, "location_confirm_yes_set")
		noText := h.services.Localization.T(context.Background(), userLang, "location_confirm_no_ignore")
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: yesText, CallbackData: fmt.Sprintf("location_confirm_%s", url.QueryEscape(locationName))}},
			{{Text: noText, CallbackData: "location_ignore"}},
			{{Text: backText, CallbackData: "settings_main"}},
		}
	} else {
		// Location already set - offer to change it
		messageText = h.services.Localization.T(context.Background(), userLang, "location_confirm_change", existingLocation, locationName)
		yesText := h.services.Localization.T(context.Background(), userLang, "location_confirm_yes_change")
		noText := h.services.Localization.T(context.Background(), userLang, "location_confirm_no_keep")
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: yesText, CallbackData: fmt.Sprintf("location_confirm_%s", url.QueryEscape(locationName))}},
			{{Text: noText, CallbackData: "location_ignore"}},
			{{Text: backText, CallbackData: "settings_main"}},
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
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse coordinates from text
	coordPattern := `^(-?\d+\.?\d*),?\s*(-?\d+\.?\d*)$`
	re := regexp.MustCompile(coordPattern)
	matches := re.FindStringSubmatch(coordinateText)

	if len(matches) != 3 {
		h.logger.Warn().Str("input", coordinateText).Msg("Failed to parse coordinates")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_coordinate_format")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	lat, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		h.logger.Error().Err(err).Str("lat", matches[1]).Msg("Failed to parse latitude")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_latitude_invalid")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	lon, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		h.logger.Error().Err(err).Str("lon", matches[2]).Msg("Failed to parse longitude")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_longitude_invalid")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Validate coordinate ranges
	if lat < -90 || lat > 90 {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_latitude_range")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}
	if lon < -180 || lon > 180 {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_longitude_range")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	h.logger.Info().Float64("lat", lat).Float64("lon", lon).Int64("user_id", userID).Msg("Received coordinate input")

	// Create location name with embedded coordinates for confirmation
	// Don't process/save yet - only do that when user confirms
	locationName := fmt.Sprintf("coordinates (%.4f, %.4f)", lat, lon)

	h.logger.Info().Str("location", locationName).Float64("lat", lat).Float64("lon", lon).Msg("Showing confirmation for coordinates")

	// Use shared confirmation logic (coordinates will be processed and saved when user confirms)
	return h.showLocationConfirmation(bot, ctx, locationName)
}

// handleTimezoneInput processes timezone input entered as text
func (h *CommandHandler) handleTimezoneInput(bot *gotgbot.Bot, ctx *ext.Context, timezoneText string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	h.logger.Info().Str("timezone", timezoneText).Int64("user_id", userID).Msg("Processing timezone input")

	// Validate timezone
	if !h.isValidTimezone(timezoneText) {
		h.logger.Warn().Str("timezone", timezoneText).Msg("Invalid timezone")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_timezone_invalid", timezoneText)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	h.logger.Info().Str("timezone", timezoneText).Msg("Valid timezone, showing confirmation")

	// Use shared confirmation logic for timezones
	return h.showTimezoneConfirmation(bot, ctx, timezoneText)
}

// isValidTimezone validates if a timezone string is valid by attempting to load it
func (h *CommandHandler) isValidTimezone(timezone string) bool {
	_, err := time.LoadLocation(timezone)
	return err == nil
}

// showTimezoneConfirmation displays a confirmation dialog for setting/changing timezone
func (h *CommandHandler) showTimezoneConfirmation(bot *gotgbot.Bot, ctx *ext.Context, timezoneName string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check if user already has a timezone set (get current user settings)
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user for timezone confirmation")
		// Continue with default behavior
	}

	var messageText string
	var keyboard [][]gotgbot.InlineKeyboardButton

	backText := h.services.Localization.T(context.Background(), userLang, "button_back_to_settings")

	if err != nil || user.Timezone == "" {
		// No timezone set - offer to set this timezone
		messageText = h.services.Localization.T(context.Background(), userLang, "timezone_confirm_set", timezoneName)
		yesText := h.services.Localization.T(context.Background(), userLang, "timezone_confirm_yes_set")
		noText := h.services.Localization.T(context.Background(), userLang, "timezone_confirm_no_ignore")
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: yesText, CallbackData: fmt.Sprintf("timezone_confirm_%s", url.QueryEscape(timezoneName))}},
			{{Text: noText, CallbackData: "timezone_ignore"}},
			{{Text: backText, CallbackData: "settings_main"}},
		}
	} else {
		// Timezone already set (including UTC) - offer to change it
		messageText = h.services.Localization.T(context.Background(), userLang, "timezone_confirm_change", user.Timezone, timezoneName)
		yesText := h.services.Localization.T(context.Background(), userLang, "timezone_confirm_yes_change")
		noText := h.services.Localization.T(context.Background(), userLang, "timezone_confirm_no_keep")
		keyboard = [][]gotgbot.InlineKeyboardButton{
			{{Text: yesText, CallbackData: fmt.Sprintf("timezone_confirm_%s", url.QueryEscape(timezoneName))}},
			{{Text: noText, CallbackData: "timezone_ignore"}},
			{{Text: backText, CallbackData: "settings_main"}},
		}
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, messageText, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})

	return err
}

// Helper methods for formatting messages
func (h *CommandHandler) formatWeatherMessage(weather *services.WeatherData, userLang string) string {
	// Get localized strings
	temperature := h.services.Localization.T(context.Background(), userLang, "weather_temperature")
	feelsLike := h.services.Localization.T(context.Background(), userLang, "weather_feels_like")
	humidity := h.services.Localization.T(context.Background(), userLang, "weather_humidity")
	wind := h.services.Localization.T(context.Background(), userLang, "weather_wind")
	visibility := h.services.Localization.T(context.Background(), userLang, "weather_visibility")
	uvIndex := h.services.Localization.T(context.Background(), userLang, "weather_uv_index")
	pressure := h.services.Localization.T(context.Background(), userLang, "weather_pressure")
	airQuality := h.services.Localization.T(context.Background(), userLang, "weather_air_quality")
	aqi := h.services.Localization.T(context.Background(), userLang, "weather_aqi")
	updated := h.services.Localization.T(context.Background(), userLang, "weather_updated")

	// Get localized location name if available
	locationName := weather.LocationName
	if weather.Location != nil {
		locationName = h.services.Weather.GetLocalizedLocationName(weather.Location, userLang)
	}

	return fmt.Sprintf(`🌤️ *%s*

%s: %d°C (%s %d°C)
%s: %d%%
%s: %.1f km/h %d°
%s: %.1f km
%s: %.1f
%s: %.1f hPa

%s %s

*%s:*
%s: %d (%s)
CO: %.2f | NO₂: %.2f | O₃: %.2f
PM2.5: %.1f | PM10: %.1f

%s: %s`,
		locationName,
		temperature, int(weather.Temperature), feelsLike, int(weather.Temperature), // FeelsLike not available in current struct
		humidity, weather.Humidity,
		wind, weather.WindSpeed, weather.WindDirection,
		visibility, weather.Visibility,
		uvIndex, weather.UVIndex,
		pressure, weather.Pressure,
		weather.Icon,
		weather.Description,
		airQuality,
		aqi, weather.AQI, h.getAQIDescription(weather.AQI, userLang),
		weather.CO,
		weather.NO2,
		weather.O3,
		weather.PM25,
		weather.PM10,
		updated, weather.Timestamp.Format("15:04 UTC"))
}

func (h *CommandHandler) getAQIDescription(aqi int, language string) string {
	switch {
	case aqi <= 50:
		return h.services.Localization.T(context.Background(), language, "aqi_good")
	case aqi <= 100:
		return h.services.Localization.T(context.Background(), language, "aqi_moderate")
	case aqi <= 150:
		return h.services.Localization.T(context.Background(), language, "aqi_unhealthy_sensitive")
	case aqi <= 200:
		return h.services.Localization.T(context.Background(), language, "aqi_unhealthy")
	case aqi <= 300:
		return h.services.Localization.T(context.Background(), language, "aqi_very_unhealthy")
	default:
		return h.services.Localization.T(context.Background(), language, "aqi_hazardous")
	}
}

func (h *CommandHandler) formatForecastMessage(forecast *weather.ForecastData, language string) string {
	title := h.services.Localization.T(context.Background(), language, "forecast_title", forecast.Location)
	text := fmt.Sprintf("%s\n\n", title)

	humidityLabel := h.services.Localization.T(context.Background(), language, "forecast_humidity")
	windLabel := h.services.Localization.T(context.Background(), language, "forecast_wind")

	for _, day := range forecast.Forecasts {
		text += fmt.Sprintf("📅 *%s*\n", day.Date.Format("Monday, Jan 2"))
		text += fmt.Sprintf("🌡️ %.1f°/%.1f°C | %s %s\n",
			day.MaxTemp, day.MinTemp, day.Icon, day.Description)
		text += fmt.Sprintf("%s: %d%% | %s: %.1f km/h\n\n",
			humidityLabel, day.Humidity, windLabel, day.WindSpeed)
	}

	return text
}

func (h *CommandHandler) formatAirQualityMessage(air *weather.AirQualityData, language string) string {
	title := h.services.Localization.T(context.Background(), language, "air_quality_title", "Air Quality Data")
	overallAqi := h.services.Localization.T(context.Background(), language, "air_quality_overall_aqi", air.AQI, h.getAQIDescription(air.AQI, language))
	pollutantsLabel := h.services.Localization.T(context.Background(), language, "air_quality_pollutants")
	coLabel := h.services.Localization.T(context.Background(), language, "air_quality_co")
	no2Label := h.services.Localization.T(context.Background(), language, "air_quality_no2")
	o3Label := h.services.Localization.T(context.Background(), language, "air_quality_o3")
	pm25Label := h.services.Localization.T(context.Background(), language, "air_quality_pm25")
	pm10Label := h.services.Localization.T(context.Background(), language, "air_quality_pm10")
	healthLabel := h.services.Localization.T(context.Background(), language, "air_quality_health_recommendations")
	updatedLabel := h.services.Localization.T(context.Background(), language, "air_quality_updated")

	return fmt.Sprintf(`%s

%s

%s
%s: %.2f μg/m³
%s: %.2f μg/m³
%s: %.2f μg/m³
%s: %.1f μg/m³
%s: %.1f μg/m³

%s
%s

%s: %s`,
		title,
		overallAqi,
		pollutantsLabel,
		coLabel, air.CO,
		no2Label, air.NO2,
		o3Label, air.O3,
		pm25Label, air.PM25,
		pm10Label, air.PM10,
		healthLabel,
		h.getHealthRecommendation(air.AQI, language),
		updatedLabel, air.Timestamp.Format("15:04 UTC"))
}

func (h *CommandHandler) getHealthRecommendation(aqi int, language string) string {
	switch {
	case aqi <= 50:
		return h.services.Localization.T(context.Background(), language, "health_good")
	case aqi <= 100:
		return h.services.Localization.T(context.Background(), language, "health_moderate")
	case aqi <= 150:
		return h.services.Localization.T(context.Background(), language, "health_unhealthy_sensitive")
	case aqi <= 200:
		return h.services.Localization.T(context.Background(), language, "health_unhealthy")
	case aqi <= 300:
		return h.services.Localization.T(context.Background(), language, "health_very_unhealthy")
	default:
		return h.services.Localization.T(context.Background(), language, "health_hazardous")
	}
}

// Additional command handlers
func (h *CommandHandler) SetLocation(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)
	locationName := strings.TrimSpace(strings.Join(ctx.Args()[1:], " "))

	if locationName == "" {
		// Show location setting options (same as Settings → Set Location)
		title := h.services.Localization.T(context.Background(), userLang, "location_settings_title")
		optionsLabel := h.services.Localization.T(context.Background(), userLang, "location_settings_options")
		optionName := h.services.Localization.T(context.Background(), userLang, "location_settings_option_name")
		optionGPS := h.services.Localization.T(context.Background(), userLang, "location_settings_option_gps")

		promptText := fmt.Sprintf(`📍 *%s*

*%s:*
• %s
• %s`,
			title,
			optionsLabel,
			optionName,
			optionGPS)

		setByNameBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_name")
		setCoordsBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_coords")
		backBtn := h.services.Localization.T(context.Background(), userLang, "button_back_to_start")

		keyboard := [][]gotgbot.InlineKeyboardButton{
			{{Text: setByNameBtn, CallbackData: "location_set_name"}},
			{{Text: setCoordsBtn, CallbackData: "location_set_coords"}},
			{{Text: backBtn, CallbackData: "back_to_start"}},
		}

		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			promptText,
			&gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: keyboard,
				},
			})
		return err
	}

	// Validate location
	coords, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "setlocation_not_found", locationName)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Save location as user's location
	err = h.services.User.SetUserLocation(context.Background(), userID, locationName, coords.Country, "", coords.Latitude, coords.Longitude)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "setlocation_save_failed")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	weatherButtonText := h.services.Localization.T(context.Background(), userLang, "button_get_weather")
	alertButtonText := h.services.Localization.T(context.Background(), userLang, "button_add_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: weatherButtonText, CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: alertButtonText, CallbackData: fmt.Sprintf("alert_%s", locationName)}},
	}

	successMsg := h.services.Localization.T(context.Background(), userLang, "setlocation_success", locationName)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successMsg,
		&gotgbot.SendMessageOpts{
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})

	return err
}

func (h *CommandHandler) ListLocations(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil {
		return err
	}

	if locationName == "" {
		noLocationText := h.services.Localization.T(context.Background(), userLang, "listlocations_no_location")
		setLocationBtnText := h.services.Localization.T(context.Background(), userLang, "listlocations_set_location_btn")

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, noLocationText,
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: setLocationBtnText, CallbackData: "location_set"}},
					},
				},
			})
		return err
	}

	titleText := h.services.Localization.T(context.Background(), userLang, "listlocations_title", locationName)
	currentWeatherBtnText := h.services.Localization.T(context.Background(), userLang, "listlocations_current_weather_btn")
	changeLocationBtnText := h.services.Localization.T(context.Background(), userLang, "listlocations_change_location_btn")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: currentWeatherBtnText, CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: changeLocationBtnText, CallbackData: "location_set"}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, titleText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) Subscribe(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	subscriptionText := h.services.Localization.T(context.Background(), userLang, "subscribe_text")

	dailyBtn := h.services.Localization.T(context.Background(), userLang, "subscribe_daily_btn")
	weeklyBtn := h.services.Localization.T(context.Background(), userLang, "subscribe_weekly_btn")
	alertsBtn := h.services.Localization.T(context.Background(), userLang, "subscribe_alerts_btn")
	airBtn := h.services.Localization.T(context.Background(), userLang, "subscribe_air_btn")
	mySubsBtn := h.services.Localization.T(context.Background(), userLang, "subscribe_my_subs_btn")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: dailyBtn, CallbackData: "subscribe_daily"}},
		{{Text: weeklyBtn, CallbackData: "subscribe_weekly"}},
		{{Text: alertsBtn, CallbackData: "subscribe_alerts"}},
		{{Text: airBtn, CallbackData: "subscribe_air"}},
		{{Text: mySubsBtn, CallbackData: "subscriptions_list"}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	alertText := h.services.Localization.T(context.Background(), userLang, "addalert_text")

	tempBtn := h.services.Localization.T(context.Background(), userLang, "addalert_temp_btn")
	windBtn := h.services.Localization.T(context.Background(), userLang, "addalert_wind_btn")
	airBtn := h.services.Localization.T(context.Background(), userLang, "addalert_air_btn")
	rainBtn := h.services.Localization.T(context.Background(), userLang, "addalert_rain_btn")
	myAlertsBtn := h.services.Localization.T(context.Background(), userLang, "addalert_my_alerts_btn")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: tempBtn, CallbackData: "alert_create_temperature"}},
		{{Text: windBtn, CallbackData: "alert_create_wind"}},
		{{Text: airBtn, CallbackData: "alert_create_air"}},
		{{Text: rainBtn, CallbackData: "alert_create_rain"}},
		{{Text: myAlertsBtn, CallbackData: "alerts_list"}},
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

	userLang := h.getUserLanguage(context.Background(), userID)

	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "unauthorized")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	stats, err := h.services.User.GetSystemStats(context.Background())
	if err != nil {
		return err
	}

	title := h.services.Localization.T(context.Background(), userLang, "admin_stats_title")
	usersSection := h.services.Localization.T(context.Background(), userLang, "admin_stats_users_section")
	totalUsers := h.services.Localization.T(context.Background(), userLang, "admin_stats_total_users", stats.TotalUsers)
	activeUsers := h.services.Localization.T(context.Background(), userLang, "admin_stats_active_users", stats.ActiveUsers)
	newUsers := h.services.Localization.T(context.Background(), userLang, "admin_stats_new_users", stats.NewUsers24h)
	usersWithLocation := h.services.Localization.T(context.Background(), userLang, "admin_stats_users_with_location", stats.UsersWithLocation)

	notificationsSection := h.services.Localization.T(context.Background(), userLang, "admin_stats_notifications_section")
	activeSubscriptions := h.services.Localization.T(context.Background(), userLang, "admin_stats_active_subscriptions", stats.ActiveSubscriptions)
	alertsConfigured := h.services.Localization.T(context.Background(), userLang, "admin_stats_alerts_configured", stats.AlertsConfigured)
	messagesSent := h.services.Localization.T(context.Background(), userLang, "admin_stats_messages_sent", stats.MessagesSent24h)

	apiSection := h.services.Localization.T(context.Background(), userLang, "admin_stats_api_section")
	weatherRequests := h.services.Localization.T(context.Background(), userLang, "admin_stats_weather_requests", stats.WeatherRequests24h)
	cacheHitRate := h.services.Localization.T(context.Background(), userLang, "admin_stats_cache_hit_rate", stats.CacheHitRate)

	performanceSection := h.services.Localization.T(context.Background(), userLang, "admin_stats_performance_section")
	avgResponseTime := h.services.Localization.T(context.Background(), userLang, "admin_stats_avg_response_time", stats.AvgResponseTime)
	uptime := h.services.Localization.T(context.Background(), userLang, "admin_stats_uptime", stats.Uptime)

	statsText := fmt.Sprintf(`%s

%s
%s
%s
%s
%s

%s
%s
%s
%s

%s
%s
%s

%s
%s
%s`,
		title,
		usersSection, totalUsers, activeUsers, newUsers, usersWithLocation,
		notificationsSection, activeSubscriptions, alertsConfigured, messagesSent,
		apiSection, weatherRequests, cacheHitRate,
		performanceSection, avgResponseTime, uptime)

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
}

// Helper function to get weather for a specific location
func (h *CommandHandler) getWeatherForLocation(bot *gotgbot.Bot, ctx *ext.Context, locationName string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Get weather data
	weatherData, err := h.services.Weather.GetCurrentWeatherByLocation(context.Background(), locationName)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_weather_get_failed", locationName)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Format weather information using localized template
	weatherFormat := h.services.Localization.T(context.Background(), userLang, "weather_current_format")
	weatherText := fmt.Sprintf(weatherFormat,
		weatherData.LocationName,
		weatherData.Temperature,
		weatherData.WindSpeed,
		weatherData.Humidity,
		weatherData.Pressure,
		weatherData.Visibility,
		weatherData.UVIndex,
		weatherData.Description,
	)

	forecastBtn := h.services.Localization.T(context.Background(), userLang, "button_forecast")
	airQualityBtn := h.services.Localization.T(context.Background(), userLang, "button_air_quality")
	setAlertBtn := h.services.Localization.T(context.Background(), userLang, "button_set_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: forecastBtn, CallbackData: fmt.Sprintf("forecast_%s", locationName)}},
		{{Text: airQualityBtn, CallbackData: fmt.Sprintf("air_%s", locationName)}},
		{{Text: setAlertBtn, CallbackData: fmt.Sprintf("alert_%s", locationName)}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// First get coordinates for the location
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_location_not_found", locationName)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	forecast, err := h.services.Weather.GetForecast(context.Background(), locationData.Latitude, locationData.Longitude, 5)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_forecast_get_failed", locationName)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	forecastText := h.formatForecastMessage(forecast, userLang)

	currentWeatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	airQualityBtn := h.services.Localization.T(context.Background(), userLang, "button_air_quality")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: currentWeatherBtn, CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: airQualityBtn, CallbackData: fmt.Sprintf("air_%s", locationName)}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	forecast, err := h.services.Weather.GetForecast(context.Background(), lat, lon, 5)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "forecast_error")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	forecastText := h.formatForecastMessage(forecast, userLang)

	currentWeatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	airQualityBtn := h.services.Localization.T(context.Background(), userLang, "button_air_quality")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: currentWeatherBtn, CallbackData: fmt.Sprintf("weather_coords_%.4f_%.4f", lat, lon)}},
		{{Text: airQualityBtn, CallbackData: fmt.Sprintf("air_coords_%.4f_%.4f", lat, lon)}},
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
	case "start":
		return h.Start(bot, ctx)
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	switch action {
	case "add":
		promptText := h.services.Localization.T(context.Background(), userLang, "location_input_name_prompt")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, promptText, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		return err
	case "set":
		// Check if this is for coordinates or name based on params
		if len(params) > 0 && params[0] == "coords" {
			promptText := h.services.Localization.T(context.Background(), userLang, "location_input_coords_prompt")
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, promptText, &gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
			})
			return err
		} else if len(params) > 0 && params[0] == "name" {
			promptText := h.services.Localization.T(context.Background(), userLang, "location_input_name_prompt")
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, promptText, &gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
			})
			return err
		} else {
			// Default set behavior (name-based) for "location_set" without params
			promptText := h.services.Localization.T(context.Background(), userLang, "location_input_name_prompt")
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, promptText, &gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
			})
			return err
		}
	case "clear":
		userID := ctx.EffectiveUser.Id
		err := h.services.User.ClearUserLocation(context.Background(), userID)
		if err != nil {
			errorText := h.services.Localization.T(context.Background(), userLang, "location_clear_failed")
			_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorText, nil)
			return sendErr
		}
		successText := h.services.Localization.T(context.Background(), userLang, "location_clear_success")
		_, err = bot.SendMessage(ctx.EffectiveChat.Id, successText, nil)
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
			encodedLocationName := strings.Join(params, " ")
			// URL decode the location name
			locationName, decodeErr := url.QueryUnescape(encodedLocationName)
			if decodeErr != nil {
				// If decoding fails, use the encoded name as fallback
				locationName = encodedLocationName
				h.logger.Warn().Err(decodeErr).Str("encoded", encodedLocationName).Msg("Failed to decode location name")
			}
			h.logger.Info().Str("location", locationName).Msg("User confirmed location from text input")

			userID := ctx.EffectiveUser.Id
			userLang := h.getUserLanguage(context.Background(), userID)

			// Check if this is raw coordinates input that needs processing
			coordPattern := `^coordinates \((-?\d+\.?\d*),\s*(-?\d+\.?\d*)\)$`
			coordRe := regexp.MustCompile(coordPattern)
			coordMatches := coordRe.FindStringSubmatch(locationName)

			if len(coordMatches) == 3 {
				// This is raw coordinates input - process it now
				lat, _ := strconv.ParseFloat(coordMatches[1], 64)
				lon, _ := strconv.ParseFloat(coordMatches[2], 64)

				h.logger.Info().Float64("lat", lat).Float64("lon", lon).Msg("Processing raw coordinates on confirmation")

				// Get location name from coordinates (reverse geocoding)
				baseName, err := h.services.Weather.GetLocationName(context.Background(), lat, lon)
				if err != nil {
					baseName = "Location"
					h.logger.Warn().Err(err).Msg("Failed to get location name from coordinates, using default")
				}

				// Create formatted location name with coordinates
				finalLocationName := fmt.Sprintf("%s (%.4f, %.4f)", baseName, lat, lon)

				// Save the location with coordinates
				err = h.services.User.SetUserLocation(context.Background(), userID, finalLocationName, "", "", lat, lon)
				if err != nil {
					h.logger.Error().Err(err).Msg("Failed to save location to database")
					_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location. Please try again.", nil)
					return err
				}

				h.logger.Info().Str("location", finalLocationName).Msg("Location with coordinates saved successfully")
				_, err = bot.SendMessage(ctx.EffectiveChat.Id,
					fmt.Sprintf("✅ Location set to *%s*", finalLocationName),
					&gotgbot.SendMessageOpts{ParseMode: "Markdown"})
				return err
			}

			// Check if this location name already contains coordinates (from reverse geocoding)
			// Pattern: "London (51.5073, -0.1276)" or "near London (51.5073, -0.1276)"
			coordInNamePattern := `^(.+?)\s*\((-?\d+\.?\d*),\s*(-?\d+\.?\d*)\)$`
			re := regexp.MustCompile(coordInNamePattern)
			matches := re.FindStringSubmatch(locationName)

			if len(matches) == 4 {
				// Location name already contains coordinates (coordinate-based input)
				displayName := strings.TrimSpace(matches[1])
				lat, _ := strconv.ParseFloat(matches[2], 64)
				lon, _ := strconv.ParseFloat(matches[3], 64)

				h.logger.Info().Str("display_name", displayName).Float64("lat", lat).Float64("lon", lon).Msg("Using coordinates from location name")

				// Save the location with coordinates
				err := h.services.User.SetUserLocation(context.Background(), userID, locationName, "", "", lat, lon)
				if err != nil {
					h.logger.Error().Err(err).Msg("Failed to save location to database")
					_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location. Please try again.", nil)
					return err
				}

				h.logger.Info().Str("location", locationName).Msg("Location with coordinates saved successfully")
				_, err = bot.SendMessage(ctx.EffectiveChat.Id,
					fmt.Sprintf("✅ Location set to *%s*", locationName),
					&gotgbot.SendMessageOpts{ParseMode: "Markdown"})
				return err
			} else {
				// Regular location name (name-based input) - needs geocoding
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
				successText := h.services.Localization.T(context.Background(), userLang, "location_save_success_with_coords", location.Name, location.Country, location.Latitude, location.Longitude)
				_, err = bot.SendMessage(ctx.EffectiveChat.Id, successText, &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
				return err
			}
		}
	case "ignore":
		// Handle ignoring potential location from plain text input
		h.logger.Info().Msg("User ignored location suggestion from text input")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "👍 Understood, I won't set that as your location.", nil)
		return err
	}
	return nil
}

func (h *CommandHandler) handleTimezoneCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "confirm":
		// Handle timezone confirmation from text input
		if len(params) >= 1 {
			encodedTimezoneName := strings.Join(params, " ")
			// URL decode the timezone name
			timezoneName, decodeErr := url.QueryUnescape(encodedTimezoneName)
			if decodeErr != nil {
				// If decoding fails, use the encoded name as fallback
				timezoneName = encodedTimezoneName
				h.logger.Warn().Err(decodeErr).Str("encoded", encodedTimezoneName).Msg("Failed to decode timezone name")
			}
			h.logger.Info().Str("timezone", timezoneName).Msg("User confirmed timezone from text input")

			// Validate timezone again before saving
			if !h.isValidTimezone(timezoneName) {
				h.logger.Error().Str("timezone", timezoneName).Msg("Invalid timezone during confirmation")
				userID := ctx.EffectiveUser.Id
				userLang := h.getUserLanguage(context.Background(), userID)
				errorMsg := h.services.Localization.T(context.Background(), userLang, "error_timezone_invalid_simple")
				_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
				return err
			}

			// Save the timezone using existing setUserTimezone method
			err := h.setUserTimezone(bot, ctx, timezoneName)
			if err != nil {
				return err
			}

			return nil
		} else {
			h.logger.Warn().Int("params_count", len(params)).Msg("Not enough parameters for timezone confirmation")
		}
	case "ignore":
		// Handle ignoring timezone setting
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "✅ Timezone setting cancelled", nil)
		return err
	}
	return nil
}

func (h *CommandHandler) handleLanguageCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	userID := ctx.EffectiveUser.Id

	switch action {
	case "set":
		if len(params) < 1 {
			return fmt.Errorf("language code not provided")
		}

		languageCode := params[0]

		// Validate language code
		if !h.services.Localization.IsLanguageSupported(languageCode) {
			h.logger.Error().Str("language", languageCode).Msg("Invalid language code")
			return fmt.Errorf("unsupported language: %s", languageCode)
		}

		// Update user language preference
		err := h.services.User.UpdateUserLanguage(context.Background(), userID, languageCode)
		if err != nil {
			h.logger.Error().Err(err).Int64("user_id", userID).Str("language", languageCode).Msg("Failed to update user language")
			errorMsg := h.services.Localization.T(context.Background(), languageCode, "language_set_error")
			_, _, sendErr := bot.EditMessageText(errorMsg, &gotgbot.EditMessageTextOpts{
				ChatId:    ctx.EffectiveChat.Id,
				MessageId: ctx.CallbackQuery.Message.GetMessageId(),
				ParseMode: "Markdown",
			})
			if sendErr != nil {
				h.logger.Error().Err(sendErr).Msg("Failed to send error message")
			}
			return err
		}

		// Get language info for confirmation
		langInfo, _ := h.services.Localization.GetLanguageByCode(languageCode)

		// Send success message in the new language
		successMsg := h.services.Localization.T(context.Background(), languageCode, "language_set_success", langInfo.Flag, langInfo.Name)

		_, _, err = bot.EditMessageText(successMsg, &gotgbot.EditMessageTextOpts{
			ChatId:    ctx.EffectiveChat.Id,
			MessageId: ctx.CallbackQuery.Message.GetMessageId(),
			ParseMode: "Markdown",
		})

		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to send success message")
		}

		return err

	default:
		return fmt.Errorf("unknown language action: %s", action)
	}
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
		{"de-DE", "🇩🇪 Deutsch"},
		{"en-US", "🇺🇸 English"},
		{"es-ES", "🇪🇸 Español"},
		{"fr-FR", "🇫🇷 Français"},
		{"uk-UA", "🇺🇦 Українська"},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	text := h.services.Localization.T(context.Background(), userLang, "units_choose_prompt")

	metricText := h.services.Localization.T(context.Background(), userLang, "units_metric")
	imperialText := h.services.Localization.T(context.Background(), userLang, "units_imperial")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: metricText, CallbackData: "settings_units_set_metric"}},
		{{Text: imperialText, CallbackData: "settings_units_set_imperial"}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	text := h.services.Localization.T(context.Background(), userLang, "timezone_input_prompt")
	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Get coordinates first for air quality
	locationData, err := h.services.Weather.GeocodeLocation(context.Background(), locationName)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "error_location_not_found", locationName)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	airData, err := h.services.Weather.GetAirQuality(context.Background(), locationData.Latitude, locationData.Longitude)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "air_quality_error")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	airText := h.formatAirQualityMessage(airData, userLang)

	currentWeatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	forecastBtn := h.services.Localization.T(context.Background(), userLang, "button_forecast")
	setAlertBtn := h.services.Localization.T(context.Background(), userLang, "button_set_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: currentWeatherBtn, CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: forecastBtn, CallbackData: fmt.Sprintf("forecast_%s", locationName)}},
		{{Text: setAlertBtn, CallbackData: fmt.Sprintf("alert_%s", locationName)}},
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	airData, err := h.services.Weather.GetAirQuality(context.Background(), lat, lon)
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "air_quality_error")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	airText := h.formatAirQualityMessage(airData, userLang)

	currentWeatherBtn := h.services.Localization.T(context.Background(), userLang, "button_current_weather")
	forecastBtn := h.services.Localization.T(context.Background(), userLang, "button_forecast")
	setAlertBtn := h.services.Localization.T(context.Background(), userLang, "button_set_alert")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: currentWeatherBtn, CallbackData: fmt.Sprintf("weather_coords_%.4f_%.4f", lat, lon)}},
		{{Text: forecastBtn, CallbackData: fmt.Sprintf("forecast_coords_%.4f_%.4f", lat, lon)}},
		{{Text: setAlertBtn, CallbackData: fmt.Sprintf("alert_coords_%.4f_%.4f", lat, lon)}},
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get user statistics. Please try again.", nil)
		return sendErr
	}

	title := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_title")
	newUsers := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_new_users", stats.NewUsers24h)
	totalActive := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_total_active", stats.ActiveUsers)
	totalUsers := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_total_users", stats.TotalUsers)
	locations := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_locations", stats.LocationsSaved)
	alerts := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_alerts", stats.ActiveAlerts)
	messages := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_messages", stats.Messages24h)
	weatherReq := h.services.Localization.T(context.Background(), userLang, "admin_recent_activity_weather_requests", stats.WeatherRequests24h)

	text := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		title, newUsers, totalActive, totalUsers, locations, alerts, messages, weatherReq)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})
	return err
}

func (h *CommandHandler) showUserRoles(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get user statistics. Please try again.", nil)
		return sendErr
	}

	title := h.services.Localization.T(context.Background(), userLang, "admin_roles_overview_title")
	admins := h.services.Localization.T(context.Background(), userLang, "admin_roles_overview_admins", stats.AdminCount)
	moderators := h.services.Localization.T(context.Background(), userLang, "admin_roles_overview_moderators", stats.ModeratorCount)
	users := h.services.Localization.T(context.Background(), userLang, "admin_roles_overview_users", stats.TotalUsers-stats.AdminCount-stats.ModeratorCount)
	total := h.services.Localization.T(context.Background(), userLang, "admin_roles_overview_total", stats.TotalUsers)

	text := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s", title, admins, moderators, users, total)

	// Add interactive buttons for Promote/Demote
	promoteBtn := h.services.Localization.T(context.Background(), userLang, "admin_roles_promote_btn")
	demoteBtn := h.services.Localization.T(context.Background(), userLang, "admin_roles_demote_btn")

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: promoteBtn, CallbackData: "admin_role_promote"},
				{Text: demoteBtn, CallbackData: "admin_role_demote"},
			},
		},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})
	return err
}

func (h *CommandHandler) showDetailedStats(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	systemStats, err := h.services.User.GetSystemStats(context.Background())
	if err != nil {
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to get system statistics. Please try again.", nil)
		return sendErr
	}

	title := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_title")
	usersSection := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_users_section")
	usersTotal := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_users_total", systemStats.TotalUsers)
	usersActive := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_users_active", systemStats.ActiveUsers)
	usersNew := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_users_new", systemStats.NewUsers24h)
	locationsSection := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_locations_section")
	locationsCount := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_locations_count", systemStats.UsersWithLocation)
	subsSection := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_subscriptions_section")
	subsActive := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_subscriptions_active", systemStats.ActiveSubscriptions)
	alertsConfigured := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_alerts_configured", systemStats.AlertsConfigured)
	perfSection := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_performance_section")
	messagesSent := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_messages_sent", systemStats.MessagesSent24h)
	weatherReqs := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_weather_requests", systemStats.WeatherRequests24h)
	cacheHitRate := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_cache_hit_rate", systemStats.CacheHitRate)
	avgRespTime := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_avg_response_time", systemStats.AvgResponseTime)
	uptime := h.services.Localization.T(context.Background(), userLang, "admin_detailed_stats_uptime", systemStats.Uptime)

	text := fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n%s\n\n%s\n%s\n%s\n%s\n%s\n%s",
		title, usersSection, usersTotal, usersActive, usersNew,
		locationsSection, locationsCount,
		subsSection, subsActive, alertsConfigured,
		perfSection, messagesSent, weatherReqs, cacheHitRate, avgRespTime, uptime)

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
		"en-US": "🇺🇸 English",
		"uk-UA": "🇺🇦 Українська",
		"de-DE": "🇩🇪 Deutsch",
		"fr-FR": "🇫🇷 Français",
		"es-ES": "🇪🇸 Español",
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
	userLang := h.getUserLanguage(context.Background(), userID)

	err := h.services.User.UpdateUserSettings(context.Background(), userID, map[string]interface{}{
		"units": units,
	})

	if err != nil {
		errorText := h.services.Localization.T(context.Background(), userLang, "units_update_failed")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorText, nil)
		return sendErr
	}

	unitName := h.getLocalizedUnitsText(context.Background(), userLang, units)
	successText := h.services.Localization.T(context.Background(), userLang, "units_update_success", unitName)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successText, nil)
	return err
}

func (h *CommandHandler) setUserTimezone(bot *gotgbot.Bot, ctx *ext.Context, timezone string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	err := h.services.User.UpdateUserSettings(context.Background(), userID, map[string]interface{}{
		"timezone": timezone,
	})

	if err != nil {
		errorText := h.services.Localization.T(context.Background(), userLang, "timezone_update_failed")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorText, nil)
		return sendErr
	}

	successText := h.services.Localization.T(context.Background(), userLang, "timezone_update_success", timezone)
	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successText, nil)
	return err
}

func (h *CommandHandler) handleNotificationSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get user's current subscriptions
	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get user subscriptions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error loading your notification settings. Please try again.", nil)
		return err
	}

	// Build the notification settings message
	message := "🔔 *Notification Settings*\n\n"

	if len(subscriptions) == 0 {
		message += "You don't have any active notifications.\n\n"
	} else {
		message += "*Your Active Notifications:*\n"
		for i, sub := range subscriptions {
			status := "✅"
			if !sub.IsActive {
				status = "❌"
			}
			message += fmt.Sprintf("%d. %s %s - %s at %s\n",
				i+1, status, sub.SubscriptionType.String(), sub.Frequency.String(), sub.TimeOfDay)
		}
		message += "\n"
	}

	message += "_Choose an option below:_"

	// Create keyboard with notification options
	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "➕ Add Daily Weather", CallbackData: "notifications_add_daily"},
			},
			{
				{Text: "⚡ Add Weather Alerts", CallbackData: "notifications_add_alerts"},
			},
			{
				{Text: "🌪️ Add Extreme Weather", CallbackData: "notifications_add_extreme"},
			},
			{
				{Text: "📅 Add Weekly Summary", CallbackData: "notifications_add_weekly"},
			},
		},
	}

	// Add manage options if user has subscriptions
	if len(subscriptions) > 0 {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			[]gotgbot.InlineKeyboardButton{
				{Text: "⚙️ Manage Existing", CallbackData: "notifications_manage"},
			},
		)
	}

	// Add back button
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
		[]gotgbot.InlineKeyboardButton{
			{Text: "🔙 Back to Settings", CallbackData: "settings_main"},
		},
	)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})

	return err
}

func (h *CommandHandler) handleLocationSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Get user's current location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)

	var locationText string
	var statusText string
	if err != nil || locationName == "" {
		locationText = h.services.Localization.T(context.Background(), userLang, "settings_not_set")
		statusText = h.services.Localization.T(context.Background(), userLang, "location_settings_not_set_help")
	} else {
		// Location name already includes coordinates from reverse geocoding
		locationText = locationName
		statusText = h.services.Localization.T(context.Background(), userLang, "location_settings_is_set")
	}

	title := h.services.Localization.T(context.Background(), userLang, "location_settings_title")
	currentLabel := h.services.Localization.T(context.Background(), userLang, "location_settings_current")
	optionsLabel := h.services.Localization.T(context.Background(), userLang, "location_settings_options")
	optionName := h.services.Localization.T(context.Background(), userLang, "location_settings_option_name")
	optionGPS := h.services.Localization.T(context.Background(), userLang, "location_settings_option_gps")
	optionClear := h.services.Localization.T(context.Background(), userLang, "location_settings_option_clear")

	settingsText := fmt.Sprintf(`📍 *%s*

*%s:*
%s

%s

*%s:*
• %s
• %s
• %s`,
		title,
		currentLabel,
		locationText,
		statusText,
		optionsLabel,
		optionName,
		optionGPS,
		optionClear)

	setByNameBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_name")
	setCoordsBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_set_coords")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: setByNameBtn, CallbackData: "location_set_name"}},
		{{Text: setCoordsBtn, CallbackData: "location_set_coords"}},
	}

	if locationName != "" {
		clearBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_clear")
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: clearBtn, CallbackData: "location_clear"},
		})
	}

	backBtn := h.services.Localization.T(context.Background(), userLang, "location_settings_btn_back")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: backBtn, CallbackData: "settings_main"},
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
	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "🌤️ Weather Data", CallbackData: "export_weather"},
				{Text: "⚠️ Alerts", CallbackData: "export_alerts"},
			},
			{
				{Text: "📋 Subscriptions", CallbackData: "export_subscriptions"},
				{Text: "📦 All Data", CallbackData: "export_all"},
			},
			{
				{Text: "🔙 Back to Settings", CallbackData: "settings_main"},
			},
		},
	}

	text := "📊 *Data Export*\n\n" +
		"Choose what data you want to export:\n\n" +
		"🌤️ *Weather Data* - Last 30 days of weather records\n" +
		"⚠️ *Alerts* - Your alert configurations and triggered alerts\n" +
		"📋 *Subscriptions* - Your notification preferences\n" +
		"📦 *All Data* - Complete export of all your data\n\n" +
		"Exported data will be sent to you as a file."

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})

	return err
}

// Alert handlers
func (h *CommandHandler) handleTemperatureAlert(bot *gotgbot.Bot, ctx *ext.Context, condition, threshold string) error {
	userID := ctx.EffectiveUser.Id

	// Get user's location
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_setlocation")
		_, sendErr := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
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
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse the alert UUID
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Invalid alert UUID")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_invalid_id")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Get the alert
	alert, err := h.services.Alert.GetAlert(context.Background(), userID, alertUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Int64("user_id", userID).Msg("Failed to get alert")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_fetch_failed")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Parse the current condition
	var condition services.AlertCondition
	if err := json.Unmarshal([]byte(alert.Condition), &condition); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse alert condition")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_parse_error")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Get alert type text
	alertTypeText := h.getAlertTypeTextLocalized(alert.AlertType, userLang)
	operatorSymbol := h.getOperatorSymbol(condition.Operator)

	// Build the edit message
	titleText := h.services.Localization.T(context.Background(), userLang, "alerts_edit_title")
	currentText := h.services.Localization.T(context.Background(), userLang, "alerts_edit_current", alertTypeText, operatorSymbol, alert.Threshold)
	instructionText := h.services.Localization.T(context.Background(), userLang, "alerts_edit_instruction")

	message := fmt.Sprintf("*%s*\n\n%s\n\n%s", titleText, currentText, instructionText)

	// Create keyboard with threshold options
	var keyboard [][]gotgbot.InlineKeyboardButton

	// Generate threshold options based on alert type
	thresholds := h.getThresholdOptions(alert.AlertType, alert.Threshold)

	for _, threshold := range thresholds {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %.1f", operatorSymbol, threshold),
				CallbackData: fmt.Sprintf("alerts_update_%s_%.1f", alert.ID, threshold)},
		})
	}

	// Add operator change options
	changeOperatorText := h.services.Localization.T(context.Background(), userLang, "alerts_change_operator")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: changeOperatorText, CallbackData: fmt.Sprintf("alerts_operator_%s", alert.ID)},
	})

	// Add toggle active/inactive button
	toggleText := h.services.Localization.T(context.Background(), userLang, "alerts_toggle")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: toggleText, CallbackData: fmt.Sprintf("alerts_toggle_%s", alert.ID)},
	})

	// Add back button
	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back_to_list")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: backBtnText, CallbackData: "alerts_list"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	// Answer the callback query
	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, nil)
	}

	return err
}

// getThresholdOptions generates threshold options based on alert type and current value
func (h *CommandHandler) getThresholdOptions(alertType models.AlertType, currentValue float64) []float64 {
	switch alertType {
	case models.AlertTemperature:
		// Temperature: -20 to 40°C in 5° increments
		return h.generateRangeOptions(-20, 40, 5, currentValue)
	case models.AlertHumidity:
		// Humidity: 20 to 90% in 10% increments
		return h.generateRangeOptions(20, 90, 10, currentValue)
	case models.AlertPressure:
		// Pressure: 960 to 1040 hPa in 10 hPa increments
		return h.generateRangeOptions(960, 1040, 10, currentValue)
	case models.AlertWindSpeed:
		// Wind: 5 to 50 km/h in 5 km/h increments
		return h.generateRangeOptions(5, 50, 5, currentValue)
	case models.AlertUVIndex:
		// UV: 1 to 11 in 1 increment
		return h.generateRangeOptions(1, 11, 1, currentValue)
	case models.AlertAirQuality:
		// AQI: 50 to 300 in 50 increments
		return h.generateRangeOptions(50, 300, 50, currentValue)
	default:
		// Default: show ±20% around current value
		min := currentValue * 0.8
		max := currentValue * 1.2
		step := (max - min) / 5
		return h.generateRangeOptions(min, max, step, currentValue)
	}
}

// generateRangeOptions generates a range of values around the current value
func (h *CommandHandler) generateRangeOptions(min, max, step, current float64) []float64 {
	var options []float64

	// Add 3 values below current, current, and 3 values above current
	for i := -3; i <= 3; i++ {
		value := current + float64(i)*step
		if value >= min && value <= max {
			options = append(options, value)
		}
	}

	// Ensure we have at least 5 options
	if len(options) < 5 {
		options = []float64{}
		for v := min; v <= max && len(options) < 7; v += step {
			options = append(options, v)
		}
	}

	return options
}

// updateAlertThreshold updates the threshold value of an alert
func (h *CommandHandler) updateAlertThreshold(bot *gotgbot.Bot, ctx *ext.Context, alertID string, thresholdStr string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse alert UUID
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Invalid alert UUID")
		return err
	}

	// Parse threshold value
	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("threshold", thresholdStr).Msg("Invalid threshold value")
		return err
	}

	// Update the alert
	updates := map[string]interface{}{
		"threshold": threshold,
	}

	err = h.services.Alert.UpdateAlert(context.Background(), userID, alertUUID, updates)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Failed to update alert")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_update_failed")
		_, _ = bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Send success message
	successMsg := h.services.Localization.T(context.Background(), userLang, "alerts_update_success", threshold)
	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back_to_list")

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: backBtnText, CallbackData: "alerts_list"}},
			},
		},
	})

	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: h.services.Localization.T(context.Background(), userLang, "alerts_update_success", threshold),
		})
	}

	return err
}

// showOperatorOptions shows operator change options for an alert
func (h *CommandHandler) showOperatorOptions(bot *gotgbot.Bot, ctx *ext.Context, alertID string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	titleText := h.services.Localization.T(context.Background(), userLang, "alerts_operator_title")
	message := fmt.Sprintf("*%s*\n\n", titleText)

	// Create keyboard with operator options
	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "> (Greater than)", CallbackData: fmt.Sprintf("alerts_setoperator_%s_gt", alertID)}},
		{{Text: "≥ (Greater than or equal)", CallbackData: fmt.Sprintf("alerts_setoperator_%s_gte", alertID)}},
		{{Text: "< (Less than)", CallbackData: fmt.Sprintf("alerts_setoperator_%s_lt", alertID)}},
		{{Text: "≤ (Less than or equal)", CallbackData: fmt.Sprintf("alerts_setoperator_%s_lte", alertID)}},
		{{Text: "= (Equal to)", CallbackData: fmt.Sprintf("alerts_setoperator_%s_eq", alertID)}},
	}

	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: backBtnText, CallbackData: fmt.Sprintf("alerts_edit_%s", alertID)},
	})

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, nil)
	}

	return err
}

// updateAlertOperator updates the operator of an alert
func (h *CommandHandler) updateAlertOperator(bot *gotgbot.Bot, ctx *ext.Context, alertID string, operator string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse alert UUID
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Invalid alert UUID")
		return err
	}

	// Get the current alert to update condition
	alert, err := h.services.Alert.GetAlert(context.Background(), userID, alertUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get alert")
		return err
	}

	// Parse current condition
	var condition services.AlertCondition
	if err := json.Unmarshal([]byte(alert.Condition), &condition); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse alert condition")
		return err
	}

	// Update operator
	condition.Operator = operator
	conditionJSON, err := json.Marshal(condition)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal condition")
		return err
	}

	// Update the alert
	updates := map[string]interface{}{
		"condition": string(conditionJSON),
	}

	err = h.services.Alert.UpdateAlert(context.Background(), userID, alertUUID, updates)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Failed to update alert operator")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_update_failed")
		_, _ = bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Send success message
	operatorSymbol := h.getOperatorSymbol(operator)
	successMsg := h.services.Localization.T(context.Background(), userLang, "alerts_operator_update_success", operatorSymbol)
	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back_to_list")

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: backBtnText, CallbackData: "alerts_list"}},
			},
		},
	})

	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: h.services.Localization.T(context.Background(), userLang, "alerts_operator_update_success", operatorSymbol),
		})
	}

	return err
}

// toggleAlert toggles an alert active/inactive state
func (h *CommandHandler) toggleAlert(bot *gotgbot.Bot, ctx *ext.Context, alertID string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse alert UUID
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Invalid alert UUID")
		return err
	}

	// Get the current alert
	alert, err := h.services.Alert.GetAlert(context.Background(), userID, alertUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get alert")
		return err
	}

	// Toggle the active state
	newState := !alert.IsActive
	updates := map[string]interface{}{
		"is_active": newState,
	}

	err = h.services.Alert.UpdateAlert(context.Background(), userID, alertUUID, updates)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Failed to toggle alert")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_toggle_failed")
		_, _ = bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Send success message
	var successMsg string
	if newState {
		successMsg = h.services.Localization.T(context.Background(), userLang, "alerts_activated")
	} else {
		successMsg = h.services.Localization.T(context.Background(), userLang, "alerts_deactivated")
	}

	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back_to_list")

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: backBtnText, CallbackData: "alerts_list"}},
			},
		},
	})

	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: successMsg,
		})
	}

	return err
}

func (h *CommandHandler) removeAlert(bot *gotgbot.Bot, ctx *ext.Context, alertID string) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Parse the alert UUID
	alertUUID, err := uuid.Parse(alertID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Msg("Invalid alert UUID")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_invalid_id")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Delete the alert
	err = h.services.Alert.DeleteAlert(context.Background(), userID, alertUUID)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", alertID).Int64("user_id", userID).Msg("Failed to delete alert")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_delete_failed")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Send success message
	successMsg := h.services.Localization.T(context.Background(), userLang, "alerts_delete_success")

	// Provide option to go back to alerts list
	backBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_back_to_list")
	addNewBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_add_new_btn")

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, successMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{{Text: backBtnText, CallbackData: "alerts_list"}},
				{{Text: addNewBtnText, CallbackData: "alert_create_temperature"}},
			},
		},
	})

	// Answer the callback query to remove the loading state
	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: h.services.Localization.T(context.Background(), userLang, "alerts_delete_success"),
		})
	}

	return err
}

// handleAlertsCallback handles the alerts list callback
func (h *CommandHandler) handleAlertsCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "list":
		// List all user alerts
		return h.listUserAlerts(bot, ctx)

	case "edit":
		// Edit an alert by ID
		if len(params) > 0 {
			return h.editAlert(bot, ctx, params[0])
		}

	case "remove":
		// Remove an alert by ID
		if len(params) > 0 {
			return h.removeAlert(bot, ctx, params[0])
		}

	case "update":
		// Update alert threshold: alerts_update_{alertID}_{threshold}
		if len(params) >= 2 {
			return h.updateAlertThreshold(bot, ctx, params[0], params[1])
		}

	case "operator":
		// Show operator change options: alerts_operator_{alertID}
		if len(params) > 0 {
			return h.showOperatorOptions(bot, ctx, params[0])
		}

	case "setoperator":
		// Set new operator: alerts_setoperator_{alertID}_{operator}
		if len(params) >= 2 {
			return h.updateAlertOperator(bot, ctx, params[0], params[1])
		}

	case "toggle":
		// Toggle alert active/inactive: alerts_toggle_{alertID}
		if len(params) > 0 {
			return h.toggleAlert(bot, ctx, params[0])
		}
	}

	return nil
}

func (h *CommandHandler) listUserAlerts(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	alerts, err := h.services.Alert.GetUserAlerts(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get user alerts")
		errorMsg := h.services.Localization.T(context.Background(), userLang, "alerts_fetch_failed")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	if len(alerts) == 0 {
		noAlertsText := h.services.Localization.T(context.Background(), userLang, "alerts_none")
		createBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_create_btn")

		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			noAlertsText,
			&gotgbot.SendMessageOpts{
				ParseMode: "Markdown",
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: createBtnText, CallbackData: "alert_create_temperature"}},
					},
				},
			})
		return err
	}

	// Get user info for location
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get user")
		return err
	}

	titleText := h.services.Localization.T(context.Background(), userLang, "alerts_list_title")
	text := fmt.Sprintf("*%s*\n\n", titleText)
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, alert := range alerts {
		alertTypeText := h.getAlertTypeTextLocalized(alert.AlertType, userLang)

		// Parse condition JSON to get operator
		var condition services.AlertCondition
		if err := json.Unmarshal([]byte(alert.Condition), &condition); err == nil {
			operatorSymbol := h.getOperatorSymbol(condition.Operator)

			text += fmt.Sprintf("%d. *%s*\n", i+1, alertTypeText)
			if user.LocationName != "" {
				text += fmt.Sprintf("   📍 %s\n", user.LocationName)
			}
			text += fmt.Sprintf("   ⚡ %s %.1f\n", operatorSymbol, alert.Threshold)
			statusText := h.services.Localization.T(context.Background(), userLang, "alerts_status_active")
			if !alert.IsActive {
				statusText = h.services.Localization.T(context.Background(), userLang, "alerts_status_inactive")
			}
			text += fmt.Sprintf("   🔔 %s\n\n", statusText)
		}

		editBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_edit_btn")
		removeBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_remove_btn")

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s", editBtnText, alertTypeText),
				CallbackData: fmt.Sprintf("alerts_edit_%s", alert.ID)},
			{Text: removeBtnText,
				CallbackData: fmt.Sprintf("alerts_remove_%s", alert.ID)},
		})
	}

	addNewBtnText := h.services.Localization.T(context.Background(), userLang, "alerts_add_new_btn")
	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: addNewBtnText, CallbackData: "alert_create_temperature"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	// Answer the callback query
	if ctx.CallbackQuery != nil {
		_, _ = ctx.CallbackQuery.Answer(bot, nil)
	}

	return err
}

func (h *CommandHandler) handleNotificationCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "add":
		if len(params) > 0 {
			return h.handleAddNotification(bot, ctx, params[0])
		}
	case "manage":
		return h.handleManageNotifications(bot, ctx)
	case "create":
		if len(params) >= 3 {
			return h.createNotification(bot, ctx, params[0], params[1], params[2])
		}
	case "toggle":
		if len(params) > 0 {
			return h.toggleNotification(bot, ctx, params[0])
		}
	case "delete":
		if len(params) > 0 {
			return h.deleteNotification(bot, ctx, params[0])
		}
	case "info":
		// Handle info display button - this is just for display, acknowledge the callback
		if len(params) > 0 && params[0] == "display" {
			_, err := ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
				Text: "This shows your notification type. Use the buttons next to it to manage this notification.",
			})
			return err
		}
	}
	return nil
}

func (h *CommandHandler) handleAddNotification(bot *gotgbot.Bot, ctx *ext.Context, notificationType string) error {
	userID := ctx.EffectiveUser.Id
	h.logger.Info().Str("type", notificationType).Int64("user_id", userID).Msg("Adding notification")

	// Check if user has a location set
	locationName, _, _, err := h.services.User.GetUserLocation(context.Background(), userID)
	if err != nil || locationName == "" {
		userLang := h.getUserLanguage(context.Background(), userID)
		errorMsg := h.services.Localization.T(context.Background(), userLang, "location_required_notifications")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			errorMsg,
			&gotgbot.SendMessageOpts{
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "📍 Set Location", CallbackData: "settings_location"}},
						{{Text: "🔙 Back", CallbackData: "notifications_manage"}},
					},
				},
			})
		return err
	}

	var description string
	var emoji string

	switch notificationType {
	case "daily":
		description = "daily weather updates"
		emoji = "☀️"
	case "weekly":
		description = "weekly weather summaries"
		emoji = "📅"
	case "alerts":
		description = "weather alerts and warnings"
		emoji = "⚡"
	case "extreme":
		description = "extreme weather notifications"
		emoji = "🌪️"
	default:
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid notification type.", nil)
		return err
	}

	message := fmt.Sprintf(`%s *Setup %s*

You're setting up %s for *%s*.

Choose your preferred time:`, emoji, description, description, locationName)

	// Determine the frequency based on notification type
	frequency := getNotificationFrequency(notificationType)

	// Create time selection buttons
	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "🌅 06:00", CallbackData: fmt.Sprintf("notifications_create_%s_06:00_%s", notificationType, frequency)},
				{Text: "🌞 08:00", CallbackData: fmt.Sprintf("notifications_create_%s_08:00_%s", notificationType, frequency)},
			},
			{
				{Text: "☀️ 12:00", CallbackData: fmt.Sprintf("notifications_create_%s_12:00_%s", notificationType, frequency)},
				{Text: "🌅 18:00", CallbackData: fmt.Sprintf("notifications_create_%s_18:00_%s", notificationType, frequency)},
			},
			{
				{Text: "🌙 20:00", CallbackData: fmt.Sprintf("notifications_create_%s_20:00_%s", notificationType, frequency)},
				{Text: "🌃 22:00", CallbackData: fmt.Sprintf("notifications_create_%s_22:00_%s", notificationType, frequency)},
			},
			{
				{Text: "🔙 Back", CallbackData: "settings_notifications"},
			},
		},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})

	return err
}

func (h *CommandHandler) createNotification(bot *gotgbot.Bot, ctx *ext.Context, notificationType, timeOfDay, frequency string) error {
	userID := ctx.EffectiveUser.Id

	var subscriptionType models.SubscriptionType
	switch notificationType {
	case "daily":
		subscriptionType = models.SubscriptionDaily
	case "weekly":
		subscriptionType = models.SubscriptionWeekly
	case "alerts":
		subscriptionType = models.SubscriptionAlerts
	case "extreme":
		subscriptionType = models.SubscriptionExtreme
	default:
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid notification type.", nil)
		return err
	}

	var freq models.Frequency
	switch frequency {
	case "daily":
		freq = models.FrequencyDaily
	case "weekly":
		freq = models.FrequencyWeekly
	default:
		freq = models.FrequencyDaily
	}

	// Create the subscription
	_, err := h.services.Subscription.CreateSubscription(context.Background(), userID, subscriptionType, freq, timeOfDay)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to create subscription")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error creating notification. Please try again.", nil)
		return err
	}

	message := fmt.Sprintf("✅ *Notification Created!*\n\n%s %s notifications will be sent at %s every day.\n\nYou can manage all your notifications in Settings → Notifications.",
		getNotificationEmoji(subscriptionType), subscriptionType.String(), timeOfDay)

	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{{Text: "🔔 Manage Notifications", CallbackData: "settings_notifications"}},
			{{Text: "⚙️ Settings", CallbackData: "settings_main"}},
		},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})

	return err
}

func (h *CommandHandler) handleManageNotifications(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get user subscriptions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error loading your notifications. Please try again.", nil)
		return err
	}

	if len(subscriptions) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"🔔 You don't have any active notifications.\n\nUse the buttons below to add some!",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "➕ Add Notifications", CallbackData: "settings_notifications"}},
					},
				},
			})
		return err
	}

	message := "⚙️ *Manage Your Notifications*\n\n*Active Notifications:*\n"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, sub := range subscriptions {
		status := "✅"
		if !sub.IsActive {
			status = "❌"
		}
		message += fmt.Sprintf("%d. %s %s %s - %s at %s\n",
			i+1, getNotificationEmoji(sub.SubscriptionType), status,
			sub.SubscriptionType.String(), sub.Frequency.String(), sub.TimeOfDay)

		// Add toggle button
		toggleText := "❌ Disable"
		toggleAction := "toggle"
		if !sub.IsActive {
			toggleText = "✅ Enable"
		}

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s", getNotificationEmoji(sub.SubscriptionType), sub.SubscriptionType.String()), CallbackData: "notifications_info_display"},
			{Text: toggleText, CallbackData: fmt.Sprintf("notifications_%s_%s", toggleAction, sub.ID.String())},
			{Text: "🗑️", CallbackData: fmt.Sprintf("notifications_delete_%s", sub.ID.String())},
		})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: "🔙 Back", CallbackData: "settings_notifications"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) toggleNotification(bot *gotgbot.Bot, ctx *ext.Context, subscriptionID string) error {
	userID := ctx.EffectiveUser.Id

	// Parse UUID
	subID, err := uuid.Parse(subscriptionID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid subscription ID.", nil)
		return err
	}

	// Get current subscription to toggle its state
	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error loading subscription.", nil)
		return err
	}

	var currentSub *models.Subscription
	for _, sub := range subscriptions {
		if sub.ID == subID {
			currentSub = &sub
			break
		}
	}

	if currentSub == nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Subscription not found.", nil)
		return err
	}

	// Toggle the active state
	newState := !currentSub.IsActive
	err = h.services.Subscription.UpdateSubscription(context.Background(), userID, subID, map[string]interface{}{
		"is_active": newState,
	})

	if err != nil {
		h.logger.Error().Err(err).Str("subscription_id", subscriptionID).Msg("Failed to toggle subscription")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error updating subscription.", nil)
		return err
	}

	status := "enabled"
	if !newState {
		status = "disabled"
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ %s %s notifications %s successfully!",
			getNotificationEmoji(currentSub.SubscriptionType), currentSub.SubscriptionType.String(), status),
		&gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{{Text: "⚙️ Manage Notifications", CallbackData: "notifications_manage"}},
				},
			},
		})

	return err
}

func (h *CommandHandler) deleteNotification(bot *gotgbot.Bot, ctx *ext.Context, subscriptionID string) error {
	userID := ctx.EffectiveUser.Id

	subID, err := uuid.Parse(subscriptionID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid subscription ID.", nil)
		return err
	}

	err = h.services.Subscription.DeleteSubscription(context.Background(), userID, subID)
	if err != nil {
		h.logger.Error().Err(err).Str("subscription_id", subscriptionID).Msg("Failed to delete subscription")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Error deleting notification.", nil)
		return err
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, "✅ Notification deleted successfully!",
		&gotgbot.SendMessageOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{{Text: "⚙️ Manage Notifications", CallbackData: "notifications_manage"}},
				},
			},
		})

	return err
}

// Export callback handler
func (h *CommandHandler) handleExportCallback(bot *gotgbot.Bot, ctx *ext.Context, subAction string, parts []string) error {
	switch subAction {
	case "weather", "alerts", "subscriptions", "all":
		return h.showExportFormatOptions(bot, ctx, subAction)
	case "format":
		if len(parts) < 2 {
			return fmt.Errorf("invalid export format callback: missing parameters")
		}
		exportType := parts[0]
		format := parts[1]
		return h.processExportRequest(bot, ctx, exportType, format)
	default:
		h.logger.Warn().Str("sub_action", subAction).Msg("Unknown export callback subaction")
		return nil
	}
}

func (h *CommandHandler) showExportFormatOptions(bot *gotgbot.Bot, ctx *ext.Context, exportType string) error {
	keyboard := gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "📄 JSON", CallbackData: fmt.Sprintf("export_format_%s_json", exportType)},
				{Text: "📊 CSV", CallbackData: fmt.Sprintf("export_format_%s_csv", exportType)},
			},
			{
				{Text: "📝 TXT", CallbackData: fmt.Sprintf("export_format_%s_txt", exportType)},
			},
			{
				{Text: "🔙 Back", CallbackData: "settings_export"},
			},
		},
	}

	var dataTypeText string
	switch exportType {
	case "weather":
		dataTypeText = "🌤️ Weather Data"
	case "alerts":
		dataTypeText = "⚠️ Alerts"
	case "subscriptions":
		dataTypeText = "📋 Subscriptions"
	case "all":
		dataTypeText = "📦 All Data"
	default:
		dataTypeText = "Data"
	}

	text := fmt.Sprintf("📊 *Export %s*\n\n"+
		"Choose the export format:\n\n"+
		"📄 *JSON* - Machine-readable format for technical use\n"+
		"📊 *CSV* - Spreadsheet-compatible format\n"+
		"📝 *TXT* - Human-readable text format\n\n"+
		"The exported file will be sent to you via Telegram.", dataTypeText)

	_, _, err := bot.EditMessageText(text, &gotgbot.EditMessageTextOpts{
		ChatId:      ctx.EffectiveChat.Id,
		MessageId:   ctx.CallbackQuery.Message.GetMessageId(),
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	})

	return err
}

func (h *CommandHandler) processExportRequest(bot *gotgbot.Bot, ctx *ext.Context, exportType, format string) error {
	userID := ctx.EffectiveUser.Id

	// Show processing message
	_, _, err := bot.EditMessageText("🔄 *Preparing your data export...*\n\nThis may take a few moments.", &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.CallbackQuery.Message.GetMessageId(),
		ParseMode: "Markdown",
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to show processing message")
	}

	// Map export type string to service enum
	var serviceExportType services.ExportType
	switch exportType {
	case "weather":
		serviceExportType = services.ExportTypeWeatherData
	case "alerts":
		serviceExportType = services.ExportTypeAlerts
	case "subscriptions":
		serviceExportType = services.ExportTypeSubscriptions
	case "all":
		serviceExportType = services.ExportTypeAll
	default:
		return fmt.Errorf("invalid export type: %s", exportType)
	}

	// Map format string to service enum
	var serviceFormat services.ExportFormat
	switch format {
	case "json":
		serviceFormat = services.ExportFormatJSON
	case "csv":
		serviceFormat = services.ExportFormatCSV
	case "txt":
		serviceFormat = services.ExportFormatTXT
	default:
		return fmt.Errorf("invalid export format: %s", format)
	}

	// Generate export
	userLang := h.getUserLanguage(context.Background(), userID)
	buffer, filename, err := h.services.Export.ExportUserData(context.Background(), userID, serviceExportType, serviceFormat, userLang)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to export user data")

		_, _, editErr := bot.EditMessageText("❌ *Export Failed*\n\nSorry, there was an error generating your export. Please try again later.", &gotgbot.EditMessageTextOpts{
			ChatId:    ctx.EffectiveChat.Id,
			MessageId: ctx.CallbackQuery.Message.GetMessageId(),
			ParseMode: "Markdown",
		})
		if editErr != nil {
			h.logger.Error().Err(editErr).Msg("Failed to update error message")
		}
		return err
	}

	// Create a temporary file for sending
	tempFile, err := os.CreateTemp("", filename)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create temporary file for export")
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write buffer to temporary file
	if _, err := buffer.WriteTo(tempFile); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write export data to temporary file")
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Reset file pointer to beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		h.logger.Error().Err(err).Msg("Failed to reset file pointer")
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Send the file
	namedFile := gotgbot.NamedFile{
		File:     tempFile,
		FileName: filename,
	}
	_, err = bot.SendDocument(ctx.EffectiveChat.Id, namedFile, &gotgbot.SendDocumentOpts{
		Caption:   fmt.Sprintf("📊 Your %s export in %s format", exportType, format),
		ParseMode: "Markdown",
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to send export file")

		_, _, editErr := bot.EditMessageText("❌ *Export Failed*\n\nSorry, there was an error sending your export file. Please try again later.", &gotgbot.EditMessageTextOpts{
			ChatId:    ctx.EffectiveChat.Id,
			MessageId: ctx.CallbackQuery.Message.GetMessageId(),
			ParseMode: "Markdown",
		})
		if editErr != nil {
			h.logger.Error().Err(editErr).Msg("Failed to update error message")
		}
		return err
	}

	// Update the message to show success
	_, _, err = bot.EditMessageText("✅ *Export Complete*\n\nYour data export has been sent as a file above.", &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.CallbackQuery.Message.GetMessageId(),
		ParseMode: "Markdown",
	})

	h.logger.Info().
		Int64("user_id", userID).
		Str("export_type", exportType).
		Str("format", format).
		Str("filename", filename).
		Int("file_size", buffer.Len()).
		Msg("Data export completed successfully")

	return err
}

func (h *CommandHandler) handleBackCallback(bot *gotgbot.Bot, ctx *ext.Context, subAction string, parts []string) error {
	switch subAction {
	case "to":
		if len(parts) < 1 {
			h.logger.Warn().Msg("Invalid back_to callback: missing destination")
			return nil
		}
		destination := parts[0]
		switch destination {
		case "start":
			// Re-run the Start command to show main menu
			return h.Start(bot, ctx)
		default:
			h.logger.Warn().Str("destination", destination).Msg("Unknown back destination")
			return nil
		}
	default:
		h.logger.Warn().Str("sub_action", subAction).Msg("Unknown back callback subaction")
		return nil
	}
}

func getNotificationEmoji(subscriptionType models.SubscriptionType) string {
	switch subscriptionType {
	case models.SubscriptionDaily:
		return "☀️"
	case models.SubscriptionWeekly:
		return "📅"
	case models.SubscriptionAlerts:
		return "⚡"
	case models.SubscriptionExtreme:
		return "🌪️"
	default:
		return "🔔"
	}
}

// getNotificationFrequency returns the frequency string for callback data based on notification type
func getNotificationFrequency(notificationType string) string {
	switch notificationType {
	case "daily":
		return "daily"
	case "weekly":
		return "weekly"
	case "alerts", "extreme":
		return "alerts" // Alerts don't use frequency but need consistent callback format
	default:
		return "daily" // Default to daily for unknown types
	}
}

// Localized helper methods
func (h *CommandHandler) getLocalizedUnitsText(ctx context.Context, language, units string) string {
	switch units {
	case "metric":
		return h.services.Localization.T(ctx, language, "units_metric")
	case "imperial":
		return h.services.Localization.T(ctx, language, "units_imperial")
	default:
		return units
	}
}

func (h *CommandHandler) getLocalizedRoleName(ctx context.Context, language string, role models.UserRole) string {
	switch role {
	case models.RoleAdmin:
		return h.services.Localization.T(ctx, language, "role_admin")
	case models.RoleModerator:
		return h.services.Localization.T(ctx, language, "role_moderator")
	default:
		return h.services.Localization.T(ctx, language, "role_user")
	}
}

func (h *CommandHandler) getLocalizedStatusText(ctx context.Context, language string, isActive bool) string {
	if isActive {
		return h.services.Localization.T(ctx, language, "status_active")
	}
	return h.services.Localization.T(ctx, language, "status_inactive")
}

// UnknownCommand handles unknown commands and suggests similar ones
func (h *CommandHandler) UnknownCommand(bot *gotgbot.Bot, ctx *ext.Context) error {
	user := ctx.EffectiveUser

	// Auto-register new users and show welcome message
	isNewUser := h.ensureUserRegistered(context.Background(), user)
	if isNewUser {
		welcomeMsg := "👋 Welcome to ShoPogoda Weather Bot!\n\n"
		welcomeMsg += "I see this is your first time here. Let's get started!\n\n"
		welcomeMsg += "Use /start to begin or /help to see all available commands."

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, welcomeMsg, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		return err
	}

	// Get the unknown command from the message text
	commandText := ctx.EffectiveMessage.Text
	if commandText == "" {
		return nil
	}

	// Extract command (first word)
	parts := strings.Fields(commandText)
	if len(parts) == 0 {
		return nil
	}

	unknownCmd := strings.TrimPrefix(parts[0], "/")

	// Get command suggestions
	suggestions := suggestSimilarCommands(unknownCmd, 5)

	// Build response message
	var message string
	if len(suggestions) > 0 {
		message = fmt.Sprintf("❓ Unknown command: `/%s`\n\n", unknownCmd)
		message += "Did you mean:\n"
		for _, cmd := range suggestions {
			message += fmt.Sprintf("• /%s\n", cmd)
		}
		message += "\nUse /help to see all available commands."
	} else {
		message = fmt.Sprintf("❓ Unknown command: `/%s`\n\n", unknownCmd)
		message += "Use /help to see all available commands."
	}

	_, err := bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// getAlertTypeTextLocalized returns localized alert type text
func (h *CommandHandler) getAlertTypeTextLocalized(alertType models.AlertType, language string) string {
	switch alertType {
	case models.AlertTemperature:
		return h.services.Localization.T(context.Background(), language, "alert_type_temperature")
	case models.AlertHumidity:
		return h.services.Localization.T(context.Background(), language, "alert_type_humidity")
	case models.AlertPressure:
		return h.services.Localization.T(context.Background(), language, "alert_type_pressure")
	case models.AlertWindSpeed:
		return h.services.Localization.T(context.Background(), language, "alert_type_wind_speed")
	case models.AlertUVIndex:
		return h.services.Localization.T(context.Background(), language, "alert_type_uv_index")
	case models.AlertAirQuality:
		return h.services.Localization.T(context.Background(), language, "alert_type_air_quality")
	case models.AlertRain:
		return h.services.Localization.T(context.Background(), language, "alert_type_rain")
	case models.AlertSnow:
		return h.services.Localization.T(context.Background(), language, "alert_type_snow")
	case models.AlertStorm:
		return h.services.Localization.T(context.Background(), language, "alert_type_storm")
	default:
		return h.services.Localization.T(context.Background(), language, "alert_type_unknown")
	}
}

// getOperatorSymbol converts operator codes to user-friendly symbols
func (h *CommandHandler) getOperatorSymbol(operator string) string {
	switch operator {
	case "gt":
		return ">"
	case "gte":
		return "≥"
	case "lt":
		return "<"
	case "lte":
		return "≤"
	case "eq":
		return "="
	default:
		return operator
	}
}
