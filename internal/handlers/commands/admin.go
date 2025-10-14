package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/valpere/shopogoda/internal/models"
)

// Promote command handler - promotes a user to a higher role
func (h *CommandHandler) Promote(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check admin permissions
	adminUser, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || adminUser.Role != models.RoleAdmin {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_insufficient_permissions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	args := ctx.Args()
	if len(args) < 2 {
		usageMsg := `*Usage:* /promote <user_id> [role]

*Arguments:*
• user_id - Telegram user ID
• role (optional) - Target role: moderator (default) or admin

*Examples:*
/promote 123456789 - Promote user to Moderator
/promote 123456789 moderator - Promote user to Moderator
/promote 123456789 admin - Promote user to Admin

*Role Progression:*
User → Moderator → Admin`

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, usageMsg, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		return err
	}

	// Parse target user ID
	targetUserIDStr := args[1]
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid user ID format", nil)
		return err
	}

	// Get target user
	targetUser, err := h.services.User.GetUser(context.Background(), targetUserID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ User not found", nil)
		return err
	}

	// Determine target role (default: Moderator, or parse from args)
	targetRole := models.RoleModerator
	if len(args) >= 3 {
		roleArg := strings.ToLower(args[2])
		switch roleArg {
		case "moderator", "mod":
			targetRole = models.RoleModerator
		case "admin":
			targetRole = models.RoleAdmin
		default:
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"❌ Invalid role. Use 'moderator' or 'admin'", nil)
			return err
		}
	}

	// Validate promotion path based on current role
	var newRole models.UserRole
	switch targetUser.Role {
	case models.RoleUser:
		if targetRole == models.RoleAdmin {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"❌ Cannot promote User directly to Admin. Promote to Moderator first.", nil)
			return err
		}
		newRole = models.RoleModerator
	case models.RoleModerator:
		if targetRole == models.RoleModerator {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id,
				"ℹ️ User is already a Moderator", nil)
			return err
		}
		newRole = models.RoleAdmin
	case models.RoleAdmin:
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"ℹ️ User is already an Admin (highest role)", nil)
		return err
	}

	// Send confirmation dialog
	currentRoleName := h.services.User.GetRoleName(targetUser.Role)
	newRoleName := h.services.User.GetRoleName(newRole)

	username := targetUser.Username
	if username == "" {
		username = fmt.Sprintf("%s %s", targetUser.FirstName, targetUser.LastName)
	}

	confirmMsg := fmt.Sprintf(`*Confirm Role Change*

👤 *User:* %s (ID: %d)
📊 *Current Role:* %s
⬆️ *New Role:* %s

Are you sure you want to promote this user?`,
		username, targetUserID, currentRoleName, newRoleName)

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "✅ Confirm", CallbackData: fmt.Sprintf("role_confirm_promote_%d_%d", targetUserID, newRole)},
			{Text: "❌ Cancel", CallbackData: "role_cancel"},
		},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, confirmMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// Demote command handler - demotes a user to a lower role
func (h *CommandHandler) Demote(bot *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	userLang := h.getUserLanguage(context.Background(), userID)

	// Check admin permissions
	adminUser, err := h.services.User.GetUser(context.Background(), userID)
	if err != nil || adminUser.Role != models.RoleAdmin {
		errorMsg := h.services.Localization.T(context.Background(), userLang, "admin_broadcast_insufficient_permissions")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	args := ctx.Args()
	if len(args) < 2 {
		usageMsg := `*Usage:* /demote <user_id>

*Arguments:*
• user_id - Telegram user ID

*Examples:*
/demote 123456789 - Demote Admin to Moderator, or Moderator to User

*Role Progression:*
Admin → Moderator → User`

		_, err := bot.SendMessage(ctx.EffectiveChat.Id, usageMsg, &gotgbot.SendMessageOpts{
			ParseMode: "Markdown",
		})
		return err
	}

	// Parse target user ID
	targetUserIDStr := args[1]
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid user ID format", nil)
		return err
	}

	// Get target user
	targetUser, err := h.services.User.GetUser(context.Background(), targetUserID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ User not found", nil)
		return err
	}

	// Determine new role based on current role
	var newRole models.UserRole
	switch targetUser.Role {
	case models.RoleUser:
		_, err := bot.SendMessage(ctx.EffectiveChat.Id,
			"ℹ️ User already has the lowest role (User)", nil)
		return err
	case models.RoleModerator:
		newRole = models.RoleUser
	case models.RoleAdmin:
		newRole = models.RoleModerator
	}

	// Send confirmation dialog
	currentRoleName := h.services.User.GetRoleName(targetUser.Role)
	newRoleName := h.services.User.GetRoleName(newRole)

	username := targetUser.Username
	if username == "" {
		username = fmt.Sprintf("%s %s", targetUser.FirstName, targetUser.LastName)
	}

	confirmMsg := fmt.Sprintf(`*Confirm Role Change*

👤 *User:* %s (ID: %d)
📊 *Current Role:* %s
⬇️ *New Role:* %s

Are you sure you want to demote this user?`,
		username, targetUserID, currentRoleName, newRoleName)

	// Special warning for demoting admin
	if targetUser.Role == models.RoleAdmin {
		confirmMsg += "\n\n⚠️ *Warning:* Demoting an Admin is a significant action."
	}

	keyboard := [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "✅ Confirm", CallbackData: fmt.Sprintf("role_confirm_demote_%d_%d", targetUserID, newRole)},
			{Text: "❌ Cancel", CallbackData: "role_cancel"},
		},
	}

	_, err = bot.SendMessage(ctx.EffectiveChat.Id, confirmMsg, &gotgbot.SendMessageOpts{
		ParseMode: "Markdown",
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})

	return err
}

// handleRoleCallback processes role change confirmation callbacks
func (h *CommandHandler) handleRoleCallback(bot *gotgbot.Bot, ctx *ext.Context, action string, params []string) error {
	switch action {
	case "confirm":
		return h.confirmRoleChange(bot, ctx, params)
	case "cancel":
		return h.cancelRoleChange(bot, ctx)
	default:
		h.logger.Warn().Str("action", action).Msg("Unknown role callback action")
		return nil
	}
}

// confirmRoleChange executes the confirmed role change
func (h *CommandHandler) confirmRoleChange(bot *gotgbot.Bot, ctx *ext.Context, params []string) error {
	adminID := ctx.EffectiveUser.Id

	// Parse params: [action, targetUserID, newRole]
	// Callback format: role_confirm_<action>_<targetUserID>_<newRole>
	if len(params) < 2 {
		h.logger.Error().Int("params_len", len(params)).Msg("Invalid role confirmation callback params")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid callback data", nil)
		return err
	}

	// params[0] is promote/demote action (for logging)
	targetUserIDStr := params[0]
	newRoleStr := params[1]

	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("target_user_id", targetUserIDStr).Msg("Failed to parse target user ID")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid user ID", nil)
		return err
	}

	newRoleInt, err := strconv.Atoi(newRoleStr)
	if err != nil {
		h.logger.Error().Err(err).Str("new_role", newRoleStr).Msg("Failed to parse new role")
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ Invalid role", nil)
		return err
	}
	newRole := models.UserRole(newRoleInt)

	// Get target user before change for comparison
	targetUser, err := h.services.User.GetUser(context.Background(), targetUserID)
	if err != nil {
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, "❌ User not found", nil)
		return err
	}

	oldRoleName := h.services.User.GetRoleName(targetUser.Role)
	newRoleName := h.services.User.GetRoleName(newRole)

	// Execute role change
	err = h.services.User.ChangeUserRole(context.Background(), adminID, targetUserID, newRole)
	if err != nil {
		h.logger.Error().Err(err).Int64("admin_id", adminID).Int64("target_user_id", targetUserID).Msg("Failed to change user role")

		errorMsg := fmt.Sprintf("❌ Failed to change role: %v", err)
		_, err := bot.SendMessage(ctx.EffectiveChat.Id, errorMsg, nil)
		return err
	}

	username := targetUser.Username
	if username == "" {
		username = fmt.Sprintf("%s %s", targetUser.FirstName, targetUser.LastName)
	}

	successMsg := fmt.Sprintf(`✅ *Role Changed Successfully*

👤 *User:* %s (ID: %d)
📊 *Previous Role:* %s
⭐ *New Role:* %s

The user's permissions have been updated.`,
		username, targetUserID, oldRoleName, newRoleName)

	// Edit the original message to show success
	_, _, err = bot.EditMessageText(successMsg, &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.CallbackQuery.Message.GetMessageId(),
		ParseMode: "Markdown",
	})

	return err
}

// cancelRoleChange cancels the role change confirmation
func (h *CommandHandler) cancelRoleChange(bot *gotgbot.Bot, ctx *ext.Context) error {
	cancelMsg := "❌ Role change cancelled."

	// Edit the original message
	_, _, err := bot.EditMessageText(cancelMsg, &gotgbot.EditMessageTextOpts{
		ChatId:    ctx.EffectiveChat.Id,
		MessageId: ctx.CallbackQuery.Message.GetMessageId(),
	})

	return err
}
