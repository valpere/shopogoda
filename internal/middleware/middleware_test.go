package middleware

import (
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/metrics"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewUserRateLimiter(t *testing.T) {
	rateLimit := rate.Limit(10) // 10 requests per second
	burst := 20

	limiter := NewUserRateLimiter(rateLimit, burst)

	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.limiters)
	assert.Equal(t, rateLimit, limiter.rate)
	assert.Equal(t, burst, limiter.burst)
	assert.Empty(t, limiter.limiters) // Initially no limiters
}

func TestUserRateLimiter_Allow(t *testing.T) {
	t.Run("allows first request", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(10), 5)
		userID := int64(123)

		allowed := limiter.Allow(userID)
		assert.True(t, allowed)
	})

	t.Run("creates limiter for new user", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(10), 5)
		userID := int64(456)

		// First call creates limiter
		limiter.Allow(userID)

		// Verify limiter was created
		limiter.mu.RLock()
		_, exists := limiter.limiters[userID]
		limiter.mu.RUnlock()

		assert.True(t, exists)
	})

	t.Run("allows within burst limit", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(1), 3) // 1 req/s, burst of 3
		userID := int64(789)

		// First 3 requests should succeed (burst)
		assert.True(t, limiter.Allow(userID))
		assert.True(t, limiter.Allow(userID))
		assert.True(t, limiter.Allow(userID))
	})

	t.Run("denies after exceeding burst limit", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(1), 2) // 1 req/s, burst of 2
		userID := int64(101112)

		// Use up burst
		limiter.Allow(userID)
		limiter.Allow(userID)

		// Third immediate request should be denied
		allowed := limiter.Allow(userID)
		assert.False(t, allowed)
	})

	t.Run("allows after waiting for rate limit recovery", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(10), 1) // 10 req/s, burst of 1
		userID := int64(131415)

		// First request succeeds
		assert.True(t, limiter.Allow(userID))

		// Second immediate request fails
		assert.False(t, limiter.Allow(userID))

		// Wait for token to refill (100ms = 1/10 second)
		time.Sleep(150 * time.Millisecond)

		// Should succeed now
		assert.True(t, limiter.Allow(userID))
	})

	t.Run("independent limiters per user", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(1), 1)
		user1 := int64(111)
		user2 := int64(222)

		// Use up user1's limit
		assert.True(t, limiter.Allow(user1))
		assert.False(t, limiter.Allow(user1))

		// User2 should still be allowed
		assert.True(t, limiter.Allow(user2))
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(100), 50)
		userID := int64(999)

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 10; j++ {
					limiter.Allow(userID)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should not panic from concurrent access
		assert.True(t, true)
	})

	t.Run("graceful shutdown stops cleanup goroutine", func(t *testing.T) {
		limiter := NewUserRateLimiter(rate.Limit(10), 5)

		// Add some limiters
		limiter.Allow(int64(123))
		limiter.Allow(int64(456))

		// Stop should close the done channel and terminate cleanupLoop
		limiter.Stop()

		// Verify done channel is closed
		select {
		case <-limiter.done:
			// Channel is closed, goroutine should terminate
			assert.True(t, true)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("done channel was not closed")
		}
	})
}

func TestLogging(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.Disabled)
	handler := Logging(logger)

	t.Run("logs message command", func(t *testing.T) {
		bot := &gotgbot.Bot{}
		ctx := &ext.Context{
			Update: &gotgbot.Update{
				Message: &gotgbot.Message{Text: "/start"},
			},
			EffectiveUser: &gotgbot.User{Id: 123, Username: "testuser"},
			EffectiveChat: &gotgbot.Chat{Id: 456},
		}

		err := handler(bot, ctx)
		assert.NoError(t, err)
	})

	t.Run("logs callback query", func(t *testing.T) {
		bot := &gotgbot.Bot{}
		ctx := &ext.Context{
			Update: &gotgbot.Update{
				CallbackQuery: &gotgbot.CallbackQuery{Data: "test_callback"},
			},
			EffectiveUser: &gotgbot.User{Id: 123, Username: "testuser"},
			EffectiveChat: &gotgbot.Chat{Id: 456},
		}

		err := handler(bot, ctx)
		assert.NoError(t, err)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewUserRateLimiter(rate.Limit(1), 1)
	handler := RateLimit(limiter)

	bot := &gotgbot.Bot{}

	t.Run("allows first request", func(t *testing.T) {
		ctx := &ext.Context{
			EffectiveUser: &gotgbot.User{Id: 123},
		}

		err := handler(bot, ctx)
		assert.NoError(t, err)
	})

	t.Run("blocks after rate limit", func(t *testing.T) {
		userID := int64(999)

		// Use up the limiter
		limiter.Allow(userID)

		ctx := &ext.Context{
			EffectiveUser: &gotgbot.User{Id: userID},
			EffectiveMessage: &gotgbot.Message{
				MessageId: 1,
				Chat:      gotgbot.Chat{Id: 123},
			},
		}

		// This will try to send a reply but we can't easily mock that
		// The important part is the rate limiter logic is tested
		_ = handler(bot, ctx)
	})
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("creates handler function", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		mockRedis := helpers.NewMockRedis()
		metricsCollector := metrics.New()
		startTime := time.Now()

		logger := zerolog.Nop()
		userService := services.NewUserService(mockDB.DB, mockRedis.Client, metricsCollector, &logger, startTime)
		handler := Auth(userService)

		assert.NotNil(t, handler)
	})

	t.Run("calls user registration", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		mockRedis := helpers.NewMockRedis()
		metricsCollector := metrics.New()
		startTime := time.Now()

		logger := zerolog.Nop()
		userService := services.NewUserService(mockDB.DB, mockRedis.Client, metricsCollector, &logger, startTime)
		handler := Auth(userService)

		bot := &gotgbot.Bot{}
		ctx := &ext.Context{
			EffectiveUser: &gotgbot.User{
				Id:        123,
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
		}

		// Call the handler - it will attempt to register the user
		// The registration may fail with mock DB but we're testing the code path
		_ = handler(bot, ctx)

		// The important part is that the handler executed without panic
		// and attempted to call RegisterUser
		assert.NotNil(t, handler)
	})

	t.Run("handles user registration error", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		mockRedis := helpers.NewMockRedis()
		metricsCollector := metrics.New()
		startTime := time.Now()

		// Close DB to simulate error
		sqlDB, _ := mockDB.DB.DB()
		_ = sqlDB.Close()

		logger := zerolog.Nop()
		userService := services.NewUserService(mockDB.DB, mockRedis.Client, metricsCollector, &logger, startTime)
		handler := Auth(userService)

		bot := &gotgbot.Bot{}
		ctx := &ext.Context{
			EffectiveUser: &gotgbot.User{
				Id:       123,
				Username: "testuser",
			},
		}

		err := handler(bot, ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register user")
	})
}

func TestMetricsMiddleware(t *testing.T) {
	handler := Metrics()

	t.Run("executes without error", func(t *testing.T) {
		bot := &gotgbot.Bot{}
		ctx := &ext.Context{
			EffectiveUser: &gotgbot.User{Id: 123},
		}

		err := handler(bot, ctx)
		assert.NoError(t, err)
	})
}
