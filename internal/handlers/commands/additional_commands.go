package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"

	"github.com/valpere/shopogoda/internal"
	"github.com/valpere/shopogoda/internal/models"
)

// SetDefaultLocation command handler
func (h *CommandHandler) SetDefaultLocation(bot *gotgbot.Bot, ctx *ext.Context) error {
	args := ctx.Args()

	if len(args) < 2 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"Usage: /setdefault <location_id>\n\nUse /locations to see your saved locations with IDs", nil)
		return err
	}

	// This functionality is deprecated - use /setlocation instead
	_, err := bot.SendMessage(ctx.EffectiveChat.Id,
		"⚠️ This command is deprecated. Use /setlocation instead to set your location.", nil)
	return err
}

// Unsubscribe command handler
func (h *CommandHandler) Unsubscribe(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		return err
	}

	if len(subscriptions) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📋 You have no active subscriptions.", nil)
		return err
	}

	text := "📋 *Your Active Subscriptions:*\n\nSelect subscription to remove:\n\n"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, sub := range subscriptions {
		subTypeText := h.getSubscriptionTypeText(sub.SubscriptionType, userLang)
		text += fmt.Sprintf("%d. %s - %s\n", i+1, subTypeText, sub.User.LocationName)

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("🗑️ Remove %s", subTypeText),
				CallbackData: fmt.Sprintf("unsubscribe_%s", sub.ID)},
		})
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// ListSubscriptions command handler
func (h *CommandHandler) ListSubscriptions(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	subscriptions, err := h.services.Subscription.GetUserSubscriptions(context.Background(), userID)
	if err != nil {
		return err
	}

	if len(subscriptions) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📋 You have no active subscriptions.\n\nUse /subscribe to set up weather notifications!",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "🔔 Subscribe Now", CallbackData: "subscribe_daily"}},
					},
				},
			})
		return err
	}

	text := "📋 *Your Active Subscriptions:*\n\n"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, sub := range subscriptions {
		subTypeText := h.getSubscriptionTypeText(sub.SubscriptionType, userLang)
		freqText := h.getFrequencyText(sub.Frequency, userLang)

		text += fmt.Sprintf("%d. **%s**\n", i+1, subTypeText)
		text += fmt.Sprintf("   📍 Location: %s\n", sub.User.LocationName)
		text += fmt.Sprintf("   ⏰ Frequency: %s\n", freqText)
		text += fmt.Sprintf("   🕐 Time: %s\n", sub.TimeOfDay)
		text += "\n"

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("⚙️ Edit %s", subTypeText),
				CallbackData: fmt.Sprintf("sub_edit_%s", sub.ID)},
			{Text: "🗑️ Remove",
				CallbackData: fmt.Sprintf("sub_remove_%s", sub.ID)},
		})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: "➕ Add New Alert", CallbackData: "alert_create_temperature"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// ListAlerts command handler
func (h *CommandHandler) ListAlerts(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	alerts, err := h.services.Alert.GetUserAlerts(context.Background(), userID)
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"📋 You have no active alerts.\n\nUse /addalert to create weather alerts!",
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
						{{Text: "⚠️ Create Alert", CallbackData: "alert_create_temperature"}},
					},
				},
			})
		return err
	}

	text := "📋 *Your Active Alerts:*\n\n"
	var keyboard [][]gotgbot.InlineKeyboardButton

	for i, alert := range alerts {
		alertTypeText := h.getAlertTypeText(alert.AlertType)

		text += fmt.Sprintf("%d. **%s Alert**\n", i+1, alertTypeText)
		text += fmt.Sprintf("   📍 Location: %s\n", alert.User.LocationName)
		text += fmt.Sprintf("   ⚡ Condition: %s %.1f\n", alert.Condition, alert.Threshold)
		text += fmt.Sprintf("   🔔 Status: %s\n", h.getAlertStatusText(alert.IsActive))
		text += "\n"

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("⚙️ Edit %s", alertTypeText),
				CallbackData: fmt.Sprintf("alert_edit_%s", alert.ID)},
			{Text: "🗑️ Remove",
				CallbackData: fmt.Sprintf("alert_remove_%s", alert.ID)},
		})
	}

	keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
		{Text: "➕ Add New Alert", CallbackData: "alert_create_temperature"},
	})

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, text, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// RemoveAlert command handler
func (h *CommandHandler) RemoveAlert(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	args := ctx.Args()

	if len(args) < 2 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"Usage: /removealert <alert_id>\n\nUse /alerts to see your alert IDs", nil)
		return err
	}

	alertIDStr := args[1]
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Invalid alert ID format", nil)
		return err
	}

	err = h.services.Alert.DeleteAlert(context.Background(), userID, alertID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Failed to remove alert", nil)
		return err
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id,
		"✅ Alert removed successfully!", nil)
	return err
}

// AdminBroadcast command handler
func (h *CommandHandler) AdminBroadcast(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_insufficient_permissions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	args := ctx.Args()
	if len(args) < 2 {
		usageMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_usage")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, usageMsg, nil)
		return err
	}

	message := strings.Join(args[1:], " ")

	// Get all active users
	users, err := h.services.User.GetActiveUsers(context.Background())
	if err != nil {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_failed_get_users")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	successCount := 0
	failCount := 0

	// Send message to all users
	for _, targetUser := range users {
		if targetUser.ID == userID {
			continue // Skip sender
		}

		broadcastHeader := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_message_header", message)
		_, err := bot.SendMessage(targetUser.ID, broadcastHeader, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})

		if err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	resultMessage := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_results", successCount, failCount, len(users)-1)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, resultMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// AdminListUsers command handler
func (h *CommandHandler) AdminListUsers(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_insufficient_permissions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	// Get user statistics
	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		return err
	}

	title := h.services.Localization.T(context.Background(), userLang, "admin_users_title")
	statisticsSection := h.services.Localization.T(context.Background(), userLang, "admin_users_statistics_section")
	totalUsers := h.services.Localization.T(context.Background(), userLang, "admin_users_total_users", stats.TotalUsers)
	activeUsers := h.services.Localization.T(context.Background(), userLang, "admin_users_active_users", stats.ActiveUsers)
	newUsers := h.services.Localization.T(context.Background(), userLang, "admin_users_new_users", stats.NewUsers24h)
	admins := h.services.Localization.T(context.Background(), userLang, "admin_users_admins", stats.AdminCount)
	moderators := h.services.Localization.T(context.Background(), userLang, "admin_users_moderators", stats.ModeratorCount)

	activitySection := h.services.Localization.T(context.Background(), userLang, "admin_users_activity_section")
	messages := h.services.Localization.T(context.Background(), userLang, "admin_users_messages", stats.Messages24h)
	weatherRequests := h.services.Localization.T(context.Background(), userLang, "admin_users_weather_requests", stats.WeatherRequests24h)
	locationsSaved := h.services.Localization.T(context.Background(), userLang, "admin_users_locations_saved", stats.LocationsSaved)
	activeAlerts := h.services.Localization.T(context.Background(), userLang, "admin_users_active_alerts", stats.ActiveAlerts)

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
%s`,
		title,
		statisticsSection, totalUsers, activeUsers, newUsers, admins, moderators,
		activitySection, messages, weatherRequests, locationsSaved, activeAlerts)

	recentUsersBtn := h.services.Localization.T(context.Background(), userLang, "admin_users_recent_btn")
	rolesBtn := h.services.Localization.T(context.Background(), userLang, "admin_users_roles_btn")
	detailedStatsBtn := h.services.Localization.T(context.Background(), userLang, "admin_users_detailed_stats_btn")

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: recentUsersBtn, CallbackData: "admin_users_recent"}},
		{{Text: rolesBtn, CallbackData: "admin_users_roles"}},
		{{Text: detailedStatsBtn, CallbackData: "admin_stats_detailed"}},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, statsText, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Helper methods for text formatting
func (h *CommandHandler) getSubscriptionTypeText(subType models.SubscriptionType, language string) string {
	switch subType {
	case models.SubscriptionDaily:
		return h.services.Localization.T(context.Background(), language, "subscription_type_daily")
	case models.SubscriptionWeekly:
		return h.services.Localization.T(context.Background(), language, "subscription_type_weekly")
	case models.SubscriptionAlerts:
		return h.services.Localization.T(context.Background(), language, "subscription_type_alerts")
	case models.SubscriptionExtreme:
		return h.services.Localization.T(context.Background(), language, "subscription_type_extreme")
	default:
		return h.services.Localization.T(context.Background(), language, "subscription_type_unknown")
	}
}

func (h *CommandHandler) getFrequencyText(freq models.Frequency, language string) string {
	switch freq {
	case models.FrequencyHourly:
		return h.services.Localization.T(context.Background(), language, "frequency_hourly")
	case models.FrequencyEvery3Hours:
		return h.services.Localization.T(context.Background(), language, "frequency_every_3_hours")
	case models.FrequencyEvery6Hours:
		return h.services.Localization.T(context.Background(), language, "frequency_every_6_hours")
	case models.FrequencyDaily:
		return h.services.Localization.T(context.Background(), language, "frequency_daily")
	case models.FrequencyWeekly:
		return h.services.Localization.T(context.Background(), language, "frequency_weekly")
	default:
		return h.services.Localization.T(context.Background(), language, "frequency_unknown")
	}
}

func (h *CommandHandler) getAlertTypeText(alertType models.AlertType) string {
	switch alertType {
	case models.AlertTemperature:
		return "Temperature"
	case models.AlertHumidity:
		return "Humidity"
	case models.AlertPressure:
		return "Pressure"
	case models.AlertWindSpeed:
		return "Wind Speed"
	case models.AlertUVIndex:
		return "UV Index"
	case models.AlertAirQuality:
		return "Air Quality"
	case models.AlertRain:
		return "Rain"
	case models.AlertSnow:
		return "Snow"
	case models.AlertStorm:
		return "Storm"
	default:
		return "Unknown"
	}
}

func (h *CommandHandler) getAlertStatusText(isActive bool) string {
	if isActive {
		return "Active"
	}
	return "Inactive"
}

// Language command handler
func (h *CommandHandler) Language(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id

	// Get current user language
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil {
		h.logger.Error().Err(err).Int64("user_id", userID).Msg("Failed to get user")
		return err
	}

	currentLang := internal.DefaultLanguage // default
	if user != nil && user.Language != "" {
		currentLang = user.Language
	}

	// Get current language info
	langInfo, _ := h.services.Localization.GetLanguageByCode(currentLang)

	// Translate messages using current language
	title := h.services.Localization.T(context.Background(), currentLang, "language_select")
	currentText := h.services.Localization.T(context.Background(), currentLang, "language_current", langInfo.Flag, langInfo.Name)

	message := fmt.Sprintf("%s\n\n%s", title, currentText)

	// Create language selection keyboard
	supportedLanguages := h.services.Localization.GetSupportedLanguages()
	var keyboard [][]gotgbot.InlineKeyboardButton

	for _, lang := range supportedLanguages {
		// Add checkmark if this is the current language
		text := fmt.Sprintf("%s %s", lang.Flag, lang.Name)
		if lang.Code == currentLang {
			text += " ✅"
		}

		keyboard = append(keyboard, []gotgbot.InlineKeyboardButton{
			{Text: text, CallbackData: fmt.Sprintf("language_set_%s", lang.Code)},
		})
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, message, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}
