package webhook

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrWebhookQueueEmpty = errors.New("webhook queue is empty")
	ErrWebhookQueueFull  = errors.New("webhook queue is full")
)

// WebhookQueue manages webhook delivery queuing with priority and rate limiting
type WebhookQueue struct {
	items     []*QueuedWebhook
	capacity  int
	mu        sync.RWMutex
	notEmpty  *sync.Cond
	logger    *logrus.Logger

	// Rate limiting
	rateLimiter *RateLimiter
}

// NewWebhookQueue creates a new webhook queue
func NewWebhookQueue(capacity int, rateLimitConfig *RateLimitConfig, logger *logrus.Logger) *WebhookQueue {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	q := &WebhookQueue{
		items:    make([]*QueuedWebhook, 0, capacity),
		capacity: capacity,
		logger:   logger,
	}
	q.notEmpty = sync.NewCond(&q.mu)

	// Initialize rate limiter if configured
	if rateLimitConfig != nil {
		q.rateLimiter = NewRateLimiter(*rateLimitConfig)
	}

	return q
}

// Enqueue adds a webhook to the queue
func (q *WebhookQueue) Enqueue(webhook *QueuedWebhook) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if queue is full
	if len(q.items) >= q.capacity {
		return ErrWebhookQueueFull
	}

	// Insert webhook in order of scheduled time and priority
	inserted := false
	for i, item := range q.items {
		if q.shouldInsertBefore(webhook, item) {
			// Insert at position i
			q.items = append(q.items[:i], append([]*QueuedWebhook{webhook}, q.items[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		// Add to end
		q.items = append(q.items, webhook)
	}

	q.logger.WithFields(logrus.Fields{
		"webhook_id":   webhook.ID,
		"queue_size":   len(q.items),
		"scheduled_at": webhook.ScheduledAt,
		"event":        webhook.Payload.Event,
	}).Debug("Webhook enqueued")

	// Signal waiting dequeuers
	q.notEmpty.Signal()

	return nil
}

// Dequeue removes and returns the next ready webhook from the queue
func (q *WebhookQueue) Dequeue(ctx context.Context) (*QueuedWebhook, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check rate limit
		if q.rateLimiter != nil && !q.rateLimiter.Allow() {
			// Rate limited, wait a bit
			q.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
			q.mu.Lock()
			continue
		}

		// Look for a ready webhook
		now := time.Now()
		for i, item := range q.items {
			if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
				// Remove item from queue
				q.items = append(q.items[:i], q.items[i+1:]...)

				q.logger.WithFields(logrus.Fields{
					"webhook_id": item.ID,
					"queue_size": len(q.items),
					"attempts":   item.Attempts,
					"event":      item.Payload.Event,
				}).Debug("Webhook dequeued")

				return item, nil
			}
		}

		// No ready webhooks, wait for signal or timeout
		if len(q.items) == 0 {
			// Queue is empty, wait for enqueue
			q.notEmpty.Wait()
		} else {
			// Webhooks exist but not ready, wait until the next one is ready
			nextReady := q.items[0].ScheduledAt
			waitTime := time.Until(nextReady)
			if waitTime > 0 && waitTime < time.Hour { // Reasonable wait time
				// Unlock and wait, then relock
				q.mu.Unlock()
				select {
				case <-ctx.Done():
					q.mu.Lock()
					return nil, ctx.Err()
				case <-time.After(waitTime):
					q.mu.Lock()
				}
			} else {
				// Wait time too long or invalid, wait for signal
				q.notEmpty.Wait()
			}
		}
	}
}

// shouldInsertBefore determines if webhook a should be inserted before webhook b
func (q *WebhookQueue) shouldInsertBefore(a, b *QueuedWebhook) bool {
	// First, compare by priority
	if a.Payload.Priority != b.Payload.Priority {
		return a.Payload.Priority > b.Payload.Priority // Higher priority first
	}

	// Then by scheduled time
	return a.ScheduledAt.Before(b.ScheduledAt)
}

// Peek returns the next webhook without removing it
func (q *WebhookQueue) Peek() (*QueuedWebhook, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.items) == 0 {
		return nil, ErrWebhookQueueEmpty
	}

	now := time.Now()
	for _, item := range q.items {
		if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
			return item, nil
		}
	}

	// No ready webhooks
	return nil, ErrWebhookQueueEmpty
}

// Size returns the current queue size
func (q *WebhookQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// Clear removes all webhooks from the queue
func (q *WebhookQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = q.items[:0]
	q.logger.Info("Webhook queue cleared")
}

// GetStats returns queue statistics
func (q *WebhookQueue) GetStats() *WebhookQueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &WebhookQueueStats{
		Size:     len(q.items),
		Capacity: q.capacity,
	}

	if len(q.items) > 0 {
		// Count ready webhooks and analyze priority distribution
		now := time.Now()
		priorityCount := make(map[Priority]int)

		for _, item := range q.items {
			if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
				stats.ReadyCount++
			} else {
				stats.DelayedCount++
			}

			// Track oldest webhook
			if stats.OldestWebhook == nil || item.CreatedAt.Before(*stats.OldestWebhook) {
				stats.OldestWebhook = &item.CreatedAt
			}

			// Count by attempts
			if item.Attempts == 0 {
				stats.FirstAttemptCount++
			} else {
				stats.RetryCount++
			}

			// Count by priority
			priorityCount[item.Payload.Priority]++
		}

		stats.PriorityDistribution = priorityCount

		// Next ready time
		for _, item := range q.items {
			if item.ScheduledAt.After(now) {
				stats.NextReadyAt = &item.ScheduledAt
				break
			}
		}
	}

	stats.IsFull = len(q.items) >= q.capacity

	// Rate limiter stats
	if q.rateLimiter != nil {
		stats.RateLimiterStats = q.rateLimiter.GetStats()
	}

	return stats
}

// ListWebhooks returns a copy of all queued webhooks (for debugging)
func (q *WebhookQueue) ListWebhooks() []*QueuedWebhook {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Return a copy to avoid race conditions
	webhooks := make([]*QueuedWebhook, len(q.items))
	copy(webhooks, q.items)

	return webhooks
}

// RemoveWebhook removes a specific webhook from the queue
func (q *WebhookQueue) RemoveWebhook(webhookID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, item := range q.items {
		if item.ID == webhookID {
			// Remove item from queue
			q.items = append(q.items[:i], q.items[i+1:]...)

			q.logger.WithFields(logrus.Fields{
				"webhook_id": webhookID,
				"queue_size": len(q.items),
			}).Debug("Webhook removed from queue")

			return true
		}
	}

	return false
}

// WebhookQueueStats represents webhook queue statistics
type WebhookQueueStats struct {
	Size                  int                  `json:"size"`
	Capacity              int                  `json:"capacity"`
	ReadyCount            int                  `json:"ready_count"`
	DelayedCount          int                  `json:"delayed_count"`
	FirstAttemptCount     int                  `json:"first_attempt_count"`
	RetryCount            int                  `json:"retry_count"`
	IsFull                bool                 `json:"is_full"`
	OldestWebhook         *time.Time           `json:"oldest_webhook,omitempty"`
	NextReadyAt           *time.Time           `json:"next_ready_at,omitempty"`
	PriorityDistribution  map[Priority]int     `json:"priority_distribution,omitempty"`
	RateLimiterStats      *RateLimiterStats    `json:"rate_limiter_stats,omitempty"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	Burst             int           `json:"burst"`
	Window            time.Duration `json:"window"`
}

// RateLimiter implements token bucket rate limiting for webhooks
type RateLimiter struct {
	config     RateLimitConfig
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:     config,
		tokens:     config.Burst,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Refill tokens based on time elapsed
	elapsed := now.Sub(rl.lastRefill)
	if elapsed >= time.Minute {
		minutes := int(elapsed.Minutes())
		tokensToAdd := minutes * rl.config.RequestsPerMinute
		rl.tokens += tokensToAdd

		// Cap at burst limit
		if rl.tokens > rl.config.Burst {
			rl.tokens = rl.config.Burst
		}

		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() *RateLimiterStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return &RateLimiterStats{
		RequestsPerMinute: rl.config.RequestsPerMinute,
		Burst:             rl.config.Burst,
		AvailableTokens:   rl.tokens,
		LastRefill:        rl.lastRefill,
	}
}

// RateLimiterStats represents rate limiter statistics
type RateLimiterStats struct {
	RequestsPerMinute int       `json:"requests_per_minute"`
	Burst             int       `json:"burst"`
	AvailableTokens   int       `json:"available_tokens"`
	LastRefill        time.Time `json:"last_refill"`
}