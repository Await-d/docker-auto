package email

import (
	"context"
	"errors"
	"sync"
	"time"

	"docker-auto/pkg/email/providers"

	"github.com/sirupsen/logrus"
)

var (
	ErrQueueEmpty = errors.New("queue is empty")
	ErrQueueFull  = errors.New("queue is full")
)

// Queue manages email message queuing with priority and delay support
type Queue struct {
	items      []*providers.QueuedMessage
	capacity   int
	mu         sync.RWMutex
	notEmpty   *sync.Cond
	logger     *logrus.Logger
}

// NewQueue creates a new email queue
func NewQueue(capacity int, logger *logrus.Logger) *Queue {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	q := &Queue{
		items:    make([]*providers.QueuedMessage, 0, capacity),
		capacity: capacity,
		logger:   logger,
	}
	q.notEmpty = sync.NewCond(&q.mu)

	return q
}

// Enqueue adds a message to the queue
func (q *Queue) Enqueue(message *providers.QueuedMessage) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if queue is full
	if len(q.items) >= q.capacity {
		return ErrQueueFull
	}

	// Insert message in order of scheduled time (earliest first)
	inserted := false
	for i, item := range q.items {
		if message.ScheduledAt.Before(item.ScheduledAt) {
			// Insert at position i
			q.items = append(q.items[:i], append([]*providers.QueuedMessage{message}, q.items[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		// Add to end
		q.items = append(q.items, message)
	}

	q.logger.WithFields(logrus.Fields{
		"message_id":   message.ID,
		"queue_size":   len(q.items),
		"scheduled_at": message.ScheduledAt,
	}).Debug("Message enqueued")

	// Signal waiting dequeuers
	q.notEmpty.Signal()

	return nil
}

// Dequeue removes and returns the next ready message from the queue
func (q *Queue) Dequeue(ctx context.Context) (*providers.QueuedMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Look for a ready message
		now := time.Now()
		for i, item := range q.items {
			if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
				// Remove item from queue
				q.items = append(q.items[:i], q.items[i+1:]...)

				q.logger.WithFields(logrus.Fields{
					"message_id": item.ID,
					"queue_size": len(q.items),
					"attempts":   item.Attempts,
				}).Debug("Message dequeued")

				return item, nil
			}
		}

		// No ready messages, wait for signal or timeout
		if len(q.items) == 0 {
			// Queue is empty, wait for enqueue
			q.notEmpty.Wait()
		} else {
			// Messages exist but not ready, wait until the next one is ready
			nextReady := q.items[0].ScheduledAt
			waitTime := time.Until(nextReady)
			if waitTime > 0 {
				// Unlock and wait, then relock
				q.mu.Unlock()
				select {
				case <-ctx.Done():
					q.mu.Lock()
					return nil, ctx.Err()
				case <-time.After(waitTime):
					q.mu.Lock()
				}
			}
		}
	}
}

// Peek returns the next message without removing it
func (q *Queue) Peek() (*providers.QueuedMessage, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}

	now := time.Now()
	for _, item := range q.items {
		if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
			return item, nil
		}
	}

	// No ready messages
	return nil, ErrQueueEmpty
}

// Size returns the current queue size
func (q *Queue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

// Clear removes all messages from the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = q.items[:0]
	q.logger.Info("Queue cleared")
}

// GetStats returns queue statistics
func (q *Queue) GetStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &QueueStats{
		Size:     len(q.items),
		Capacity: q.capacity,
	}

	if len(q.items) > 0 {
		// Count ready messages
		now := time.Now()
		for _, item := range q.items {
			if item.ScheduledAt.Before(now) || item.ScheduledAt.Equal(now) {
				stats.ReadyCount++
			} else {
				stats.DelayedCount++
			}

			// Track oldest message
			if stats.OldestMessage == nil || item.CreatedAt.Before(*stats.OldestMessage) {
				stats.OldestMessage = &item.CreatedAt
			}

			// Track by attempts
			if item.Attempts == 0 {
				stats.FirstAttemptCount++
			} else {
				stats.RetryCount++
			}
		}

		// Next ready time
		for _, item := range q.items {
			if item.ScheduledAt.After(now) {
				stats.NextReadyAt = &item.ScheduledAt
				break
			}
		}
	}

	stats.IsFull = len(q.items) >= q.capacity

	return stats
}

// ListMessages returns a copy of all queued messages (for debugging)
func (q *Queue) ListMessages() []*providers.QueuedMessage {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Return a copy to avoid race conditions
	messages := make([]*providers.QueuedMessage, len(q.items))
	copy(messages, q.items)

	return messages
}

// RemoveMessage removes a specific message from the queue
func (q *Queue) RemoveMessage(messageID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, item := range q.items {
		if item.ID == messageID {
			// Remove item from queue
			q.items = append(q.items[:i], q.items[i+1:]...)

			q.logger.WithFields(logrus.Fields{
				"message_id": messageID,
				"queue_size": len(q.items),
			}).Debug("Message removed from queue")

			return true
		}
	}

	return false
}

// PriorityQueue implements a priority-based email queue
type PriorityQueue struct {
	high   *Queue
	normal *Queue
	low    *Queue
	logger *logrus.Logger
}

// NewPriorityQueue creates a new priority-based email queue
func NewPriorityQueue(capacity int, logger *logrus.Logger) *PriorityQueue {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Distribute capacity across priority levels
	highCap := capacity / 2     // 50% for high priority
	normalCap := capacity / 3   // 33% for normal priority
	lowCap := capacity - highCap - normalCap // Remaining for low priority

	return &PriorityQueue{
		high:   NewQueue(highCap, logger),
		normal: NewQueue(normalCap, logger),
		low:    NewQueue(lowCap, logger),
		logger: logger,
	}
}

// Enqueue adds a message to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(message *providers.QueuedMessage) error {
	priority := message.Message.Priority

	switch priority {
	case providers.PriorityHigh, providers.PriorityUrgent:
		return pq.high.Enqueue(message)
	case providers.PriorityLow:
		return pq.low.Enqueue(message)
	default:
		return pq.normal.Enqueue(message)
	}
}

// Dequeue removes and returns the next message from the highest priority queue
func (pq *PriorityQueue) Dequeue(ctx context.Context) (*providers.QueuedMessage, error) {
	// Try high priority first
	if message, err := pq.high.Dequeue(ctx); err == nil {
		return message, nil
	} else if err != ErrQueueEmpty && err != context.DeadlineExceeded {
		return nil, err
	}

	// Try normal priority
	if message, err := pq.normal.Dequeue(ctx); err == nil {
		return message, nil
	} else if err != ErrQueueEmpty && err != context.DeadlineExceeded {
		return nil, err
	}

	// Try low priority
	return pq.low.Dequeue(ctx)
}

// Size returns the total size across all priority queues
func (pq *PriorityQueue) Size() int {
	return pq.high.Size() + pq.normal.Size() + pq.low.Size()
}

// GetStats returns combined statistics from all priority queues
func (pq *PriorityQueue) GetStats() *PriorityQueueStats {
	return &PriorityQueueStats{
		High:   pq.high.GetStats(),
		Normal: pq.normal.GetStats(),
		Low:    pq.low.GetStats(),
		Total:  pq.Size(),
	}
}

// Clear clears all priority queues
func (pq *PriorityQueue) Clear() {
	pq.high.Clear()
	pq.normal.Clear()
	pq.low.Clear()
	pq.logger.Info("All priority queues cleared")
}

// QueueStats represents queue statistics
type QueueStats struct {
	Size               int        `json:"size"`
	Capacity           int        `json:"capacity"`
	ReadyCount         int        `json:"ready_count"`
	DelayedCount       int        `json:"delayed_count"`
	FirstAttemptCount  int        `json:"first_attempt_count"`
	RetryCount         int        `json:"retry_count"`
	IsFull             bool       `json:"is_full"`
	OldestMessage      *time.Time `json:"oldest_message,omitempty"`
	NextReadyAt        *time.Time `json:"next_ready_at,omitempty"`
}

// PriorityQueueStats represents priority queue statistics
type PriorityQueueStats struct {
	High   *QueueStats `json:"high"`
	Normal *QueueStats `json:"normal"`
	Low    *QueueStats `json:"low"`
	Total  int         `json:"total"`
}