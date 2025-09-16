package events

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Publisher interface defines methods for publishing events
type Publisher interface {
	Publish(event *Event) error
	PublishAsync(event *Event)
	Subscribe(filter EventFilter) *Subscription
	SubscribeWithUser(filter EventFilter, userID string) *Subscription
	Unsubscribe(subscriptionID string) error
	GetSubscription(subscriptionID string) (*Subscription, bool)
	Close() error
}

// EventPublisher implements the Publisher interface
type EventPublisher struct {
	subscribers    map[string]*Subscription
	mu             sync.RWMutex
	eventQueue     chan *Event
	shutdownCh     chan struct{}
	wg             sync.WaitGroup
	logger         *logrus.Logger
	persistence    PersistenceStore
	maxQueueSize   int
	workerCount    int
	eventHistory   []*Event
	maxHistorySize int
	historyMu      sync.RWMutex
}

// PersistenceStore interface for persisting events
type PersistenceStore interface {
	SaveEvent(event *Event) error
	GetEvents(filter EventFilter, limit, offset int) ([]*Event, error)
	GetEventByID(id string) (*Event, error)
	DeleteEvent(id string) error
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(logger *logrus.Logger, persistence PersistenceStore) *EventPublisher {
	if logger == nil {
		logger = logrus.New()
	}

	return &EventPublisher{
		subscribers:    make(map[string]*Subscription),
		eventQueue:     make(chan *Event, 1000), // Buffered channel for async publishing
		shutdownCh:     make(chan struct{}),
		logger:         logger,
		persistence:    persistence,
		maxQueueSize:   1000,
		workerCount:    5,
		eventHistory:   make([]*Event, 0),
		maxHistorySize: 1000,
	}
}

// Start initializes the event publisher and starts worker goroutines
func (p *EventPublisher) Start(ctx context.Context) {
	p.logger.Info("Starting event publisher")

	// Start worker goroutines for processing events
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Start cleanup goroutine
	p.wg.Add(1)
	go p.cleanupWorker(ctx)
}

// Publish publishes an event synchronously
func (p *EventPublisher) Publish(event *Event) error {
	if event == nil {
		return ErrInvalidEvent
	}

	// Add to history
	p.addToHistory(event)

	// Persist event if persistence store is available
	if p.persistence != nil {
		if err := p.persistence.SaveEvent(event); err != nil {
			p.logger.WithError(err).Error("Failed to persist event")
		}
	}

	// Send to subscribers
	p.sendToSubscribers(event)

	p.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"severity":   event.Severity,
		"source":     event.Source,
	}).Debug("Event published")

	return nil
}

// PublishAsync publishes an event asynchronously
func (p *EventPublisher) PublishAsync(event *Event) {
	if event == nil {
		p.logger.Error("Attempted to publish nil event")
		return
	}

	select {
	case p.eventQueue <- event:
		// Event queued successfully
	default:
		// Queue is full, drop the event and log warning
		p.logger.WithFields(logrus.Fields{
			"event_id":   event.ID,
			"event_type": event.Type,
			"queue_size": len(p.eventQueue),
		}).Warn("Event queue full, dropping event")
	}
}

// Subscribe creates a new subscription with the given filter
func (p *EventPublisher) Subscribe(filter EventFilter) *Subscription {
	subscription := NewSubscription(filter)

	p.mu.Lock()
	p.subscribers[subscription.ID] = subscription
	p.mu.Unlock()

	p.logger.WithFields(logrus.Fields{
		"subscription_id": subscription.ID,
		"filter_types":    filter.Types,
		"filter_sources":  filter.Sources,
	}).Debug("New subscription created")

	return subscription
}

// SubscribeWithUser creates a new subscription for a specific user
func (p *EventPublisher) SubscribeWithUser(filter EventFilter, userID string) *Subscription {
	subscription := NewSubscription(filter).WithUser(userID)

	p.mu.Lock()
	p.subscribers[subscription.ID] = subscription
	p.mu.Unlock()

	p.logger.WithFields(logrus.Fields{
		"subscription_id": subscription.ID,
		"user_id":         userID,
		"filter_types":    filter.Types,
		"filter_sources":  filter.Sources,
	}).Debug("New user subscription created")

	return subscription
}

// Unsubscribe removes a subscription
func (p *EventPublisher) Unsubscribe(subscriptionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	subscription, exists := p.subscribers[subscriptionID]
	if !exists {
		return ErrSubscriptionNotFound
	}

	subscription.Active = false
	close(subscription.Channel)
	delete(p.subscribers, subscriptionID)

	p.logger.WithField("subscription_id", subscriptionID).Debug("Subscription removed")
	return nil
}

// GetSubscription retrieves a subscription by ID
func (p *EventPublisher) GetSubscription(subscriptionID string) (*Subscription, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subscription, exists := p.subscribers[subscriptionID]
	return subscription, exists
}

// GetActiveSubscriptions returns all active subscriptions
func (p *EventPublisher) GetActiveSubscriptions() []*Subscription {
	p.mu.RLock()
	defer p.mu.RUnlock()

	subscriptions := make([]*Subscription, 0, len(p.subscribers))
	for _, sub := range p.subscribers {
		if sub.Active {
			subscriptions = append(subscriptions, sub)
		}
	}
	return subscriptions
}

// GetEventHistory returns recent events from memory
func (p *EventPublisher) GetEventHistory(filter EventFilter, limit int) []*Event {
	p.historyMu.RLock()
	defer p.historyMu.RUnlock()

	var filtered []*Event
	count := 0

	// Iterate through history in reverse order (most recent first)
	for i := len(p.eventHistory) - 1; i >= 0 && count < limit; i-- {
		event := p.eventHistory[i]
		if filter.Matches(event) {
			filtered = append(filtered, event)
			count++
		}
	}

	return filtered
}

// Close shuts down the event publisher
func (p *EventPublisher) Close() error {
	p.logger.Info("Shutting down event publisher")

	close(p.shutdownCh)

	// Close all subscription channels
	p.mu.Lock()
	for _, subscription := range p.subscribers {
		if subscription.Active {
			subscription.Active = false
			close(subscription.Channel)
		}
	}
	p.mu.Unlock()

	// Wait for all workers to finish
	p.wg.Wait()

	p.logger.Info("Event publisher shut down complete")
	return nil
}

// worker processes events from the async queue
func (p *EventPublisher) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	logger := p.logger.WithField("worker_id", workerID)
	logger.Debug("Event worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Event worker stopping due to context cancellation")
			return
		case <-p.shutdownCh:
			logger.Debug("Event worker stopping due to shutdown signal")
			return
		case event := <-p.eventQueue:
			if err := p.Publish(event); err != nil {
				logger.WithError(err).Error("Failed to publish event")
			}
		}
	}
}

// cleanupWorker periodically cleans up inactive subscriptions
func (p *EventPublisher) cleanupWorker(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdownCh:
			return
		case <-ticker.C:
			p.cleanupInactiveSubscriptions()
		}
	}
}

// cleanupInactiveSubscriptions removes inactive subscriptions
func (p *EventPublisher) cleanupInactiveSubscriptions() {
	p.mu.Lock()
	defer p.mu.Unlock()

	var toDelete []string
	for id, subscription := range p.subscribers {
		if !subscription.Active {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(p.subscribers, id)
	}

	if len(toDelete) > 0 {
		p.logger.WithField("count", len(toDelete)).Debug("Cleaned up inactive subscriptions")
	}
}

// sendToSubscribers sends an event to all matching subscribers
func (p *EventPublisher) sendToSubscribers(event *Event) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, subscription := range p.subscribers {
		if !subscription.Active {
			continue
		}

		if subscription.Filter.Matches(event) {
			select {
			case subscription.Channel <- event:
				// Event sent successfully
			default:
				// Channel is full, log warning but don't block
				p.logger.WithFields(logrus.Fields{
					"subscription_id": subscription.ID,
					"event_id":        event.ID,
				}).Warn("Subscription channel full, dropping event")
			}
		}
	}
}

// addToHistory adds an event to the in-memory history
func (p *EventPublisher) addToHistory(event *Event) {
	p.historyMu.Lock()
	defer p.historyMu.Unlock()

	p.eventHistory = append(p.eventHistory, event)

	// Keep only the most recent events
	if len(p.eventHistory) > p.maxHistorySize {
		// Remove oldest events
		copy(p.eventHistory, p.eventHistory[len(p.eventHistory)-p.maxHistorySize:])
		p.eventHistory = p.eventHistory[:p.maxHistorySize]
	}
}

// GetStats returns publisher statistics
func (p *EventPublisher) GetStats() map[string]interface{} {
	p.mu.RLock()
	activeSubscriptions := len(p.subscribers)
	p.mu.RUnlock()

	p.historyMu.RLock()
	historySize := len(p.eventHistory)
	p.historyMu.RUnlock()

	return map[string]interface{}{
		"active_subscriptions": activeSubscriptions,
		"queue_size":          len(p.eventQueue),
		"queue_capacity":      cap(p.eventQueue),
		"history_size":        historySize,
		"max_history_size":    p.maxHistorySize,
		"worker_count":        p.workerCount,
	}
}