package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"

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
		subTypeText := h.getSubscriptionTypeText(sub.SubscriptionType)
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
		subTypeText := h.getSubscriptionTypeText(sub.SubscriptionType)
		freqText := h.getFrequencyText(sub.Frequency)
		
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
	
	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Insufficient permissions", nil)
		return err
	}

	args := ctx.Args()
	if len(args) < 2 {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"Usage: /broadcast <message>\n\nSends a message to all active users", nil)
		return err
	}

	message := strings.Join(args[1:], " ")
	
	// Get all active users
	users, err := h.services.User.GetActiveUsers(context.Background())
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"❌ Failed to get user list", nil)
		return err
	}

	successCount := 0
	failCount := 0

	// Send message to all users
	for _, targetUser := range users {
		if targetUser.ID == userID {
			continue // Skip sender
		}

		broadcastMessage := fmt.Sprintf("📢 *Admin Broadcast*\n\n%s", message)
		_, err := bot.SendMessage(targetUser.ID, broadcastMessage, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		
		if err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	resultMessage := fmt.Sprintf("📊 *Broadcast Results*\n\n✅ Successful: %d\n❌ Failed: %d\n👥 Total: %d",
		successCount, failCount, len(users)-1)

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, resultMessage, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
	})

	return err
}

// AdminListUsers command handler
func (h *CommandHandler) AdminListUsers(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	
	// Check admin permissions
	user, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || user.Role != models.RoleAdmin {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Insufficient permissions", nil)
		return err
	}

	// Get user statistics
	stats, err := h.services.User.GetUserStatistics(context.Background())
	if err != nil {
		return err
	}

	statsText := fmt.Sprintf(`👥 *User Management*

📊 *Statistics:*
Total Users: %d
Active Users: %d
New Users (24h): %d
Admins: %d
Moderators: %d

📈 *Activity:*
Messages (24h): %d
Weather Requests: %d
Locations Saved: %d
Active Alerts: %d`,
		stats.TotalUsers,
		stats.ActiveUsers,
		stats.NewUsers24h,
		stats.AdminCount,
		stats.ModeratorCount,
		stats.Messages24h,
		stats.WeatherRequests24h,
		stats.LocationsSaved,
		stats.ActiveAlerts)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{{Text: "👤 Recent Users", CallbackData: "admin_users_recent"}},
		{{Text: "🔒 Manage Roles", CallbackData: "admin_users_roles"}},
		{{Text: "📊 Detailed Stats", CallbackData: "admin_stats_detailed"}},
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
func (h *CommandHandler) getSubscriptionTypeText(subType models.SubscriptionType) string {
	switch subType {
	case models.SubscriptionDaily:
		return "Daily Weather"
	case models.SubscriptionWeekly:
		return "Weekly Forecast"
	case models.SubscriptionAlerts:
		return "Weather Alerts"
	case models.SubscriptionExtreme:
		return "Extreme Weather"
	default:
		return "Unknown"
	}
}

func (h *CommandHandler) getFrequencyText(freq models.Frequency) string {
	switch freq {
	case models.FrequencyHourly:
		return "Every Hour"
	case models.FrequencyEvery3Hours:
		return "Every 3 Hours"
	case models.FrequencyEvery6Hours:
		return "Every 6 Hours"
	case models.FrequencyDaily:
		return "Daily"
	case models.FrequencyWeekly:
		return "Weekly"
	default:
		return "Unknown"
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
