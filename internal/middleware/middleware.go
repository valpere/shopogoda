package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"

	"github.com/valpere/shopogoda/internal/services"
)

// UserRateLimiter manages rate limits per user
type UserRateLimiter struct {
	limiters map[int64]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func NewUserRateLimiter(r rate.Limit, b int) *UserRateLimiter {
	return &UserRateLimiter{
		limiters: make(map[int64]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

func (rl *UserRateLimiter) Allow(userID int64) bool {
	rl.mu.RLock()
	limiter, exists := rl.limiters[userID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[userID] = limiter
		rl.mu.Unlock()
	}

	return limiter.Allow()
}

// Logging creates a logging handler function
func Logging(logger zerolog.Logger) func(bot *gotgbot.Bot, ctx *ext.Context) error {
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
			Dur("duration", time.Since(start)).
			Msg("Request processed")

		return nil
	}
}

// RateLimit creates a rate limiting handler function
func RateLimit(rateLimiter *UserRateLimiter) func(bot *gotgbot.Bot, ctx *ext.Context) error {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		userID := ctx.EffectiveUser.Id

		if !rateLimiter.Allow(userID) {
			_, err := ctx.EffectiveMessage.Reply(bot, "Rate limit exceeded. Please try again later.", nil)
			return err
		}

		return nil
	}
}

// Auth creates an authentication handler function
func Auth(userService *services.UserService) func(bot *gotgbot.Bot, ctx *ext.Context) error {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		user := ctx.EffectiveUser

		// Register or update user - need to create a proper context
		bgCtx := context.Background()
		err := userService.RegisterUser(bgCtx, user)
		if err != nil {
			return fmt.Errorf("failed to register user: %w", err)
		}

		return nil
	}
}

// Metrics creates a metrics collection handler function for basic tracking
func Metrics() func(bot *gotgbot.Bot, ctx *ext.Context) error {
	return func(bot *gotgbot.Bot, ctx *ext.Context) error {
		// Basic metrics tracking - would integrate with actual metrics system
		// For now, just a placeholder that does nothing
		return nil
	}
}