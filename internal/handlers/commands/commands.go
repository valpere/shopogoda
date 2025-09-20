package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"

	"github.com/valpere/enterprise-weather-bot/internal/services"
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
	
	// Register or update user
	if err := h.services.User.RegisterUser(ctx.Context(), user); err != nil {
		h.logger.Error().Err(err).Int64("user_id", user.Id).Msg("Failed to register user")
	}

	welcomeText := fmt.Sprintf(`🌤️ *Welcome to Enterprise Weather Bot*

Hello %s! I'm your professional weather and environmental monitoring assistant.

*Available Commands:*
🏠 /weather - Get current weather
📊 /forecast - 5-day weather forecast
🌬️ /air - Air quality information
📍 /addlocation - Add monitoring location
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
		{{Text: "📍 Add Location", CallbackData: "location_add"}},
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
/addlocation - Add a new monitoring location
/locations - List all your saved locations
/setdefault \<location\> - Set default location

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
	location := strings.TrimSpace(ctx.Args()[1:])

	// If no location provided, use default or ask for it
	if location == "" {
		defaultLocation, err := h.services.Location.GetDefaultLocation(ctx.Context(), userID)
		if err != nil || defaultLocation == nil {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
				"📍 Please provide a location or share your current location:\n\n/weather London\nor\n/addlocation to set a default location", 
				&gotgbot.SendMessageOpts{
					ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
						InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
							{{Text: "📍 Share Location", CallbackData: "share_location"}},
							{{Text: "➕ Add Location", CallbackData: "location_add"}},
						},
					},
				})
			return err
		}
		location = defaultLocation.Name
	}

	// Get weather data
	weatherData, err := h.services.Weather.GetCurrentWeather(ctx.Context(), location)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
			fmt.Sprintf("❌ Failed to get weather for '%s'. Please check the location name.", location), nil)
		return err
	}

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
	location := strings.TrimSpace(ctx.Args()[1:])

	if location == "" {
		defaultLocation, err := h.services.Location.GetDefaultLocation(ctx.Context(), userID)
		if err != nil || defaultLocation == nil {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
				"📍 Please provide a location: /forecast London", nil)
			return err
		}
		location = defaultLocation.Name
	}

	forecast, err := h.services.Weather.GetForecast(ctx.Context(), location, 5)
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
	location := strings.TrimSpace(ctx.Args()[1:])

	if location == "" {
		defaultLocation, err := h.services.Location.GetDefaultLocation(ctx.Context(), userID)
		if err != nil || defaultLocation == nil {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
				"📍 Please provide a location: /air London", nil)
			return err
		}
		location = defaultLocation.Name
	}

	airData, err := h.services.Weather.GetAirQuality(ctx.Context(), location)
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
	
	user, err := h.services.User.GetUser(ctx.Context(), userID)
	if err != nil {
		return err
	}

	settingsText := fmt.Sprintf(`⚙️ *Settings*

*Current Configuration:*
Language: %s
Role: %s
Status: %s

*Available Settings:*
• Language preferences
• Unit system (Metric/Imperial)
• Timezone settings
• Notification preferences
• Alert thresholds
• Data export options`, 
		user.Language, 
		h.getRoleName(user.Role),
		h.getStatusText(user.IsActive))

	keyboard := [][]gotgbot.InlineKeyboardButton{
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

	// Answer callback query first
	if _, err := bot.AnswerCallbackQuery(cq.Id, nil); err != nil {
		h.logger.Error().Err(err).Msg("Failed to answer callback query")
	}

	// Parse callback data
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return nil
	}

	action := parts[0]
	subAction := parts[1]

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
	}

	return nil
}

// Location message handler
func (h *CommandHandler) HandleLocationMessage(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message.Location == nil {
		return nil
	}

	userID := ctx.EffectiveUser.Id
	lat := ctx.Message.Location.Latitude
	lon := ctx.Message.Location.Longitude

	// Get location name from coordinates
	locationName, err := h.services.Weather.GetLocationName(ctx.Context(), lat, lon)
	if err != nil {
		locationName = fmt.Sprintf("Location (%.4f, %.4f)", lat, lon)
	}

	// Get weather for this location
	weatherData, err := h.services.Weather.GetCurrentWeatherByCoords(ctx.Context(), lat, lon)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
			"❌ Failed to get weather for your location", nil)
		return err
	}

	weatherText := h.formatWeatherMessage(weatherData)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "💾 Save Location", CallbackData: fmt.Sprintf("location_save_%.4f_%.4f_%s", lat, lon, locationName)}},
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

// Helper methods for formatting messages
func (h *CommandHandler) formatWeatherMessage(weather *services.WeatherData) string {
	return fmt.Sprintf(`🌤️ *%s*

🌡️ Temperature: %d°C (feels like %d°C)
💧 Humidity: %d%%
🌬️ Wind: %.1f km/h %s
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
		int(weather.FeelsLike),
		weather.Humidity,
		weather.WindSpeed,
		weather.WindDirection,
		weather.Visibility,
		weather.UVIndex,
		weather.Pressure,
		weather.WeatherIcon,
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

func (h *CommandHandler) formatForecastMessage(forecast *services.ForecastData) string {
	text := fmt.Sprintf("📊 *5-Day Forecast for %s*\n\n", forecast.LocationName)
	
	for _, day := range forecast.Days {
		text += fmt.Sprintf("📅 *%s*\n", day.Date.Format("Monday, Jan 2"))
		text += fmt.Sprintf("🌡️ %d°/%d°C | %s %s\n", 
			int(day.TempMax), int(day.TempMin), day.Icon, day.Description)
		text += fmt.Sprintf("💧 Humidity: %d%% | 🌬️ Wind: %.1f km/h\n\n", 
			day.Humidity, day.WindSpeed)
	}
	
	return text
}

func (h *CommandHandler) formatAirQualityMessage(air *services.AirQualityData) string {
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
		air.LocationName,
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
func (h *CommandHandler) AddLocation(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	locationName := strings.TrimSpace(ctx.Args()[1:])

	if locationName == "" {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 Please provide a location name:\n\n/addlocation London\nor share your current location", 
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
	coords, err := h.services.Weather.GetCoordinates(ctx.Context(), locationName)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			fmt.Sprintf("❌ Could not find location '%s'. Please check the spelling.", locationName), nil)
		return err
	}

	// Save location
	location, err := h.services.Location.AddLocation(ctx.Context(), userID, locationName, coords.Lat, coords.Lon)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Failed to save location. Please try again.", nil)
		return err
	}

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "🏠 Set as Default", CallbackData: fmt.Sprintf("location_default_%s", location.ID)}},
		{{Text: "🌤️ Get Weather", CallbackData: fmt.Sprintf("weather_%s", locationName)}},
		{{Text: "🔔 Add Alert", CallbackData: fmt.Sprintf("alert_%s", locationName)}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		fmt.Sprintf("✅ Location '%s' saved successfully!\n📍 %s", locationName, location.Name),
		&gotgbot.SendMessageOpts{
			ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})

	return err
}

func (h *CommandHandler) ListLocations(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	
	locations, err := h.services.Location.GetUserLocations(ctx.Context(), userID)
	if err != nil {
		return err
	}

	if len(locations) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 No saved locations found.\n\nUse /addlocation to add your first location!",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "➕ Add Location", CallbackData: "location_add"}},
					},
				},
			})
		return err
	}

	text := "📍 *Your Saved Locations:*\n\n"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, loc := range locations {
		defaultText := ""
		if loc.IsDefault {
			defaultText = " 🏠"
		}
		text += fmt.Sprintf("%d. %s%s\n", i+1, loc.Name, defaultText)
		
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("🌤️ %s", loc.Name), CallbackData: fmt.Sprintf("weather_%s", loc.Name)},
			{Text: "🗑️", CallbackData: fmt.Sprintf("location_delete_%s", loc.ID)},
		})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: "➕ Add New Location", CallbackData: "location_add"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

func (h *CommandHandler) Subscribe(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	
	subscriptionText := `🔔 *Weather Notifications*

Set up automatic weather updates for your locations:

*Available Subscription Types:*
• 🌅 Daily Weather (morning summary)
• 📊 Weekly Forecast (Sunday overview)
• ⚠️ Weather Alerts (extreme conditions)
• 🌬️ Air Quality Alerts (pollution levels)

*Notification Schedule:*
• Choose your preferred time
• Select specific locations
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
	user, err := h.services.User.GetUser(ctx.Context(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Insufficient permissions", nil)
		return err
	}

	stats, err := h.services.User.GetSystemStats(ctx.Context())
	if err != nil {
		return err
	}

	statsText := fmt.Sprintf(`📊 *System Statistics*

👥 *Users:*
Total Users: %d
Active Users: %d
New Users (24h): %d

📍 *Locations:*
Total Locations: %d
Active Monitoring: %d

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
		stats.TotalLocations,
		stats.ActiveMonitoring,
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
		if len(params) > 0 {
			// Handle weather for specific location
			location := strings.Join(params, "_")
			// Simulate args for existing handler
			ctx.Args = func() []string { return []string{"/weather", location} }
			return h.CurrentWeather(bot, ctx)
		}
	}
	return nil
}

func (h *CommandHandler) handleForecastCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	if len(params) > 0 {
		location := strings.Join(params, "_")
		ctx.Args = func() []string { return []string{"/forecast", location} }
		return h.Forecast(bot, ctx)
	}
	return nil
}

func (h *CommandHandler) handleSettingsCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "main":
		return h.Settings(bot, ctx)
	case "language":
		return h.handleLanguageSettings(bot, ctx)
	case "units":
		return h.handleUnitSettings(bot, ctx)
	case "timezone":
		return h.handleTimezoneSettings(bot, ctx)
	}
	return nil
}

func (h *CommandHandler) handleLocationCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "add":
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📍 Please send me a location name or share your current location:",
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
	case "save":
		// Handle saving shared location
		if len(params) >= 3 {
			lat, _ := strconv.ParseFloat(params[0], 64)
			lon, _ := strconv.ParseFloat(params[1], 64)
			name := strings.Join(params[2:], "_")
			
			userID := ctx.EffectiveUser.Id
			_, err := h.services.Location.AddLocation(ctx.Context(), userID, name, lat, lon)
			if err != nil {
				_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Failed to save location", nil)
				return err
			}
			
			_, err = bot.SendMessage(ctx.EffectiveChat.Id, 
				fmt.Sprintf("✅ Location '%s' saved successfully!", name), nil)
			return err
		}
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
	}
	return nil
}

// Additional helper methods for settings
func (h *CommandHandler) handleLanguageSettings(bot *gotgbot.Bot, ctx *ext.Context) error {
	languages := map[string]string{
		"en": "🇺🇸 English",
		"uk": "🇺🇦 Українська",
		"de": "🇩🇪 Deutsch",
		"fr": "🇫🇷 Français",
		"es": "🇪🇸 Español",
	}

	text := "🌐 *Choose your language:*"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for code, name := range languages {
		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: name, CallbackData: fmt.Sprintf("settings_language_set_%s", code)},
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
