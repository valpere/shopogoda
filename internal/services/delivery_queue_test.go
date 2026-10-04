package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestQueue builds a queue whose retry timer fires immediately and records
// the requested delays, so tests run without real waiting.
func newTestQueue(t *testing.T, policy RetryPolicy, capacity int) (*DeliveryQueue, *[]time.Duration) {
	t.Helper()
	logger := zerolog.Nop()
	q := NewDeliveryQueue(&logger, policy, 1, capacity)

	var mu sync.Mutex
	delays := &[]time.Duration{}
	q.afterFunc = func(d time.Duration, f func()) {
		mu.Lock()
		*delays = append(*delays, d)
		mu.Unlock()
		go f()
	}
	q.jitter = func(d time.Duration) time.Duration { return d }
	return q, delays
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	require.Eventually(t, cond, 2*time.Second, 5*time.Millisecond)
}

func TestDeliveryQueue_SuccessNoRetry(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: time.Minute}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "ok", Send: func() error { calls.Add(1); return nil }}))

	waitFor(t, func() bool { return calls.Load() == 1 })
	assert.Empty(t, *delays)
}

func TestDeliveryQueue_RetriesTransientThenSucceeds(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 4, BaseDelay: time.Second, MaxDelay: time.Minute}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "flaky", Send: func() error {
		if calls.Add(1) < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}}))

	waitFor(t, func() bool { return calls.Load() == 3 })
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, *delays, "exponential backoff")
}

func TestDeliveryQueue_GivesUpAfterMaxAttempts(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: time.Minute}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "down", Send: func() error { calls.Add(1); return errors.New("down") }}))

	waitFor(t, func() bool { return calls.Load() == 3 })
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 3, calls.Load(), "no attempts beyond MaxAttempts")
	assert.Len(t, *delays, 2)
}

func TestDeliveryQueue_BackoffCappedAtMaxDelay(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 5, BaseDelay: time.Second, MaxDelay: 3 * time.Second}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "cap", Send: func() error { calls.Add(1); return errors.New("x") }}))

	waitFor(t, func() bool { return calls.Load() == 5 })
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 3 * time.Second, 3 * time.Second}, *delays)
}

func TestDeliveryQueue_PermanentErrorNotRetried(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 4, BaseDelay: time.Second, MaxDelay: time.Minute}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	blocked := fmt.Errorf("send failed: %w", &gotgbot.TelegramError{Code: 403, Description: "Forbidden: bot was blocked by the user"})
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "blocked", Send: func() error { calls.Add(1); return blocked }}))

	waitFor(t, func() bool { return calls.Load() == 1 })
	time.Sleep(50 * time.Millisecond)
	assert.EqualValues(t, 1, calls.Load())
	assert.Empty(t, *delays)
}

func TestDeliveryQueue_RateLimitHonorsRetryAfter(t *testing.T) {
	q, delays := newTestQueue(t, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: time.Minute}, 4)
	q.Start(context.Background())
	defer q.Stop()

	var calls atomic.Int32
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "429", Send: func() error {
		if calls.Add(1) == 1 {
			return &gotgbot.TelegramError{Code: 429, ResponseParams: &gotgbot.ResponseParameters{RetryAfter: 7}}
		}
		return nil
	}}))

	waitFor(t, func() bool { return calls.Load() == 2 })
	assert.Equal(t, []time.Duration{7 * time.Second}, *delays)
}

func TestDeliveryQueue_FullQueueRejects(t *testing.T) {
	q, _ := newTestQueue(t, RetryPolicy{MaxAttempts: 1}, 1)
	// Not started: nothing drains the buffer.
	require.NoError(t, q.Enqueue(DeliveryJob{Label: "a", Send: func() error { return nil }}))
	assert.ErrorIs(t, q.Enqueue(DeliveryJob{Label: "b", Send: func() error { return nil }}), ErrDeliveryQueueFull)
}

func TestDeliveryQueue_EnqueueAfterStopRejects(t *testing.T) {
	q, _ := newTestQueue(t, RetryPolicy{MaxAttempts: 1}, 2)
	q.Start(context.Background())
	q.Stop()
	assert.ErrorIs(t, q.Enqueue(DeliveryJob{Label: "late", Send: func() error { return nil }}), ErrDeliveryQueueStopped)
}

func TestIsPermanentDeliveryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"telegram 403 blocked", &gotgbot.TelegramError{Code: 403}, true},
		{"telegram 400 chat not found", &gotgbot.TelegramError{Code: 400}, true},
		{"telegram 429 rate limit", &gotgbot.TelegramError{Code: 429}, false},
		{"telegram 500", &gotgbot.TelegramError{Code: 500}, false},
		{"slack 404", &slackStatusError{Code: 404}, true},
		{"slack 429", &slackStatusError{Code: 429}, false},
		{"slack 503", &slackStatusError{Code: 503}, false},
		{"network error", errors.New("connection reset"), false},
		{"wrapped telegram 403", fmt.Errorf("wrap: %w", &gotgbot.TelegramError{Code: 403}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPermanentDeliveryError(tt.err))
		})
	}
}

func TestNotificationService_DeliverWithoutQueueRunsOnce(t *testing.T) {
	logger := zerolog.Nop()
	s := NewNotificationService(nil, &logger)

	var calls int
	err := s.Deliver("sync", func() error { calls++; return errors.New("boom") })
	assert.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestDeliveryQueue_ParentContextCancelStopsQueue(t *testing.T) {
	q, _ := newTestQueue(t, RetryPolicy{MaxAttempts: 1}, 2)
	ctx, cancel := context.WithCancel(context.Background())
	q.Start(ctx)
	cancel()

	require.Eventually(t, func() bool {
		return errors.Is(q.Enqueue(DeliveryJob{Label: "x", Send: func() error { return nil }}), ErrDeliveryQueueStopped)
	}, 2*time.Second, 5*time.Millisecond, "Enqueue must reject once workers are gone so Deliver can fall back to sync")
}
