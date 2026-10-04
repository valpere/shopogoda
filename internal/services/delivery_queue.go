package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/rs/zerolog"
)

var (
	// ErrDeliveryQueueFull is returned by Enqueue when the buffer has no room.
	ErrDeliveryQueueFull = errors.New("delivery queue is full")
	// ErrDeliveryQueueStopped is returned by Enqueue after Stop.
	ErrDeliveryQueueStopped = errors.New("delivery queue is stopped")
)

// RetryPolicy controls exponential backoff for failed deliveries.
type RetryPolicy struct {
	MaxAttempts int           // total attempts including the first
	BaseDelay   time.Duration // delay before the second attempt
	MaxDelay    time.Duration // upper bound for any single delay
}

// DefaultRetryPolicy retries for roughly a minute: 2s, 4s, 8s, 16s.
var DefaultRetryPolicy = RetryPolicy{MaxAttempts: 5, BaseDelay: 2 * time.Second, MaxDelay: time.Minute}

// DeliveryJob is one notification delivery attempt-able unit.
type DeliveryJob struct {
	Label string       // used in logs only
	Send  func() error // must be safe to call repeatedly
}

type queuedJob struct {
	DeliveryJob
	attempt int // completed attempts
}

// DeliveryQueue delivers jobs on background workers and retries transient
// failures with exponential backoff. Delivery is at-least-once: a timeout
// after the server accepted a message is retried and may duplicate it. It is in-memory: jobs still queued or
// waiting for a retry are lost on shutdown.
type DeliveryQueue struct {
	logger  *zerolog.Logger
	policy  RetryPolicy
	workers int
	jobs    chan queuedJob

	mu      sync.Mutex
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// seams for tests
	afterFunc func(d time.Duration, f func())
	jitter    func(d time.Duration) time.Duration
}

func NewDeliveryQueue(logger *zerolog.Logger, policy RetryPolicy, workers, capacity int) *DeliveryQueue {
	return &DeliveryQueue{
		logger:    logger,
		policy:    policy,
		workers:   workers,
		jobs:      make(chan queuedJob, capacity),
		afterFunc: func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		jitter: func(d time.Duration) time.Duration {
			// ±20% so retries from one outage don't synchronize
			return time.Duration(float64(d) * (0.8 + 0.4*rand.Float64())) // #nosec G404 -- jitter, not security
		},
	}
}

// Start launches the workers. They run until ctx is cancelled or Stop is called.
func (q *DeliveryQueue) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	q.mu.Lock()
	q.cancel = cancel
	q.mu.Unlock()

	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx)
	}

	// If the parent context dies the workers exit; mark the queue stopped so
	// Enqueue rejects and callers fall back instead of buffering into the void.
	go func() {
		<-ctx.Done()
		q.Stop()
	}()
}

// Stop halts the workers and drops anything not yet delivered.
func (q *DeliveryQueue) Stop() {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		return
	}
	q.stopped = true
	cancel := q.cancel
	q.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	q.wg.Wait()
	if dropped := len(q.jobs); dropped > 0 {
		q.logger.Warn().Int("dropped", dropped).Msg("Delivery queue stopped with undelivered notifications")
	}
}

// Enqueue schedules a job without blocking.
func (q *DeliveryQueue) Enqueue(job DeliveryJob) error {
	return q.push(queuedJob{DeliveryJob: job})
}

func (q *DeliveryQueue) push(j queuedJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return ErrDeliveryQueueStopped
	}
	select {
	case q.jobs <- j:
		return nil
	default:
		return ErrDeliveryQueueFull
	}
}

func (q *DeliveryQueue) worker(ctx context.Context) {
	defer q.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-q.jobs:
			q.process(j)
		}
	}
}

func (q *DeliveryQueue) process(j queuedJob) {
	err := j.Send()
	j.attempt++
	if err == nil {
		if j.attempt > 1 {
			q.logger.Info().Str("job", j.Label).Int("attempts", j.attempt).Msg("Notification delivered after retry")
		}
		return
	}

	switch {
	case isPermanentDeliveryError(err):
		q.logger.Error().Err(err).Str("job", j.Label).Msg("Notification delivery failed permanently, not retrying")
		return
	case j.attempt >= q.policy.MaxAttempts:
		q.logger.Error().Err(err).Str("job", j.Label).Int("attempts", j.attempt).Msg("Notification delivery failed, giving up")
		return
	}

	delay := q.retryDelay(j.attempt, err)
	q.logger.Warn().Err(err).Str("job", j.Label).Int("attempt", j.attempt).Dur("retry_in", delay).Msg("Notification delivery failed, will retry")
	q.afterFunc(delay, func() {
		switch err := q.push(j); {
		case errors.Is(err, ErrDeliveryQueueStopped):
			q.logger.Debug().Str("job", j.Label).Msg("Queue stopped, dropping pending retry")
		case err != nil:
			q.logger.Error().Err(err).Str("job", j.Label).Msg("Dropping notification retry")
		}
	})
}

// retryDelay returns the wait before the next attempt. A server-provided
// Retry-After (Telegram 429) wins over computed backoff.
func (q *DeliveryQueue) retryDelay(attempt int, err error) time.Duration {
	var tgErr *gotgbot.TelegramError
	if errors.As(err, &tgErr) && tgErr.Code == http.StatusTooManyRequests &&
		tgErr.ResponseParams != nil && tgErr.ResponseParams.RetryAfter > 0 {
		return time.Duration(tgErr.ResponseParams.RetryAfter) * time.Second
	}

	delay := q.policy.BaseDelay << (attempt - 1)
	if delay <= 0 || delay > q.policy.MaxDelay { // <=0: shift overflow
		delay = q.policy.MaxDelay
	}
	return q.jitter(delay)
}

// slackStatusError reports a non-200 response from the Slack webhook.
type slackStatusError struct{ Code int }

func (e *slackStatusError) Error() string {
	return fmt.Sprintf("slack webhook returned status %d", e.Code)
}

// isPermanentDeliveryError reports errors a retry cannot fix: the target
// rejected the request itself (blocked bot, unknown chat, bad webhook).
func isPermanentDeliveryError(err error) bool {
	var tgErr *gotgbot.TelegramError
	if errors.As(err, &tgErr) {
		return isPermanentStatus(tgErr.Code)
	}
	var slackErr *slackStatusError
	if errors.As(err, &slackErr) {
		return isPermanentStatus(slackErr.Code)
	}
	return false
}

func isPermanentStatus(code int) bool {
	return code >= 400 && code < 500 && code != http.StatusTooManyRequests
}
