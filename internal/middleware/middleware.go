package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/valpere/enterprise-weather-bot/internal/services"
	"github.com/valpere/enterprise-weather-bot/pkg/metrics"
)

// Logging middleware
func Logging(logger zerolog.Logger) ext.Handler {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		start := time.Now()
		
		user := ctx.EffectiveUser
		chat := ctx.EffectiveChat
		
		var command string
		if ctx.Message != nil && ctx.Message.Text != "" {
			command = ctx.Message.Text
		} else if ctx.CallbackQuery != nil {
			command = fmt.Sprintf("callback:%s", ctx.CallbackQuery.Data)
		}

		logger.Info().
			Int64("user_id", user.Id).
			Str("username", user.Username).
			Int64("chat_id", chat.Id).
			Str("command", command).
			Msg("Processing update")

		// Continue to next handler
		err := ext.ContinueHandling{}
		
		duration := time.Since(start)
		if err != nil {
			logger.Error().
				Err(err).
				Int64("user_id", user.Id).
				Str("command", command).
				Dur("duration", duration).
				Msg("Handler error")
		} else {
			logger.Debug().
				Int64("user_id", user.Id).
				Str("command", command).
				Dur("duration", duration).
				Msg("Handler completed")
		}

		return err
	}
}

// Metrics middleware
func Metrics(metrics *metrics.Metrics) ext.Handler {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		start := time.Now()
		
		var commandType string
		if ctx.Message != nil {
			commandType = "message"
		} else if ctx.CallbackQuery != nil {
			commandType = "callback"
		} else {
			commandType = "other"
		}

		metrics.IncrementCounter("bot_updates_total", "type", commandType)
		
		err := ext.ContinueHandling{}
		
		duration := time.Since(start).Seconds()
		metrics.ObserveHistogram("bot_handler_duration_seconds", duration, "type", commandType)
		
		if err != nil {
			metrics.IncrementCounter("bot_errors_total", "type", commandType)
		}

		return err
	}
}

// User registration middleware
func UserRegistration(services *services.Services) ext.Handler {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		user := ctx.EffectiveUser
		if user != nil {
			// Register or update user in background
			go func() {
				ctx := context.Background()
				if err := services.User.RegisterUser(ctx, user); err != nil {
					// Log error but don't fail the request
				}
			}()
		}
		
		return ext.ContinueHandling{}
	}
}

// Rate limiting middleware
func RateLimiting() ext.Handler {
	// Create rate limiter map for users
	limiters := make(map[int64]*rate.Limiter)
	
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		userID := ctx.EffectiveUser.Id
		
		// Get or create limiter for user (10 requests per minute)
		limiter, exists := limiters[userID]
		if !exists {
			limiter = rate.NewLimiter(rate.Every(6*time.Second), 10)
			limiters[userID] = limiter
		}
		
		if !limiter.Allow() {
			_, err := bot.SendMessage(ctx.EffectiveChat.Id, 
				"⏰ Please slow down! You're sending requests too quickly.", nil)
			return err
		}
		
		return ext.ContinueHandling{}
	}
}
