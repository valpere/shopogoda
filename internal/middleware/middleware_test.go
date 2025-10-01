package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
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
}
