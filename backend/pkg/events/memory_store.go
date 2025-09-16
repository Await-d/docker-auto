package events

import (
	"sort"
	"sync"
	"time"
)

// MemoryPersistenceStore implements PersistenceStore using in-memory storage
type MemoryPersistenceStore struct {
	events    map[string]*Event
	mu        sync.RWMutex
	maxEvents int
}

// NewMemoryPersistenceStore creates a new memory-based persistence store
func NewMemoryPersistenceStore(maxEvents int) *MemoryPersistenceStore {
	if maxEvents <= 0 {
		maxEvents = 10000 // Default maximum
	}

	return &MemoryPersistenceStore{
		events:    make(map[string]*Event),
		maxEvents: maxEvents,
	}
}

// SaveEvent saves an event to memory
func (m *MemoryPersistenceStore) SaveEvent(event *Event) error {
	if event == nil {
		return ErrInvalidEvent
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.events[event.ID] = event

	// Clean up old events if we exceed the maximum
	if len(m.events) > m.maxEvents {
		m.cleanupOldEvents()
	}

	return nil
}

// GetEvents retrieves events based on filter criteria
func (m *MemoryPersistenceStore) GetEvents(filter EventFilter, limit, offset int) ([]*Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []*Event

	// Convert map to slice and filter
	for _, event := range m.events {
		if filter.Matches(event) {
			filtered = append(filtered, event)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply pagination
	if offset >= len(filtered) {
		return []*Event{}, nil
	}

	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end], nil
}

// GetEventByID retrieves a specific event by ID
func (m *MemoryPersistenceStore) GetEventByID(id string) (*Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	event, exists := m.events[id]
	if !exists {
		return nil, ErrInvalidEvent
	}

	return event, nil
}

// DeleteEvent removes an event from storage
func (m *MemoryPersistenceStore) DeleteEvent(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.events[id]; !exists {
		return ErrInvalidEvent
	}

	delete(m.events, id)
	return nil
}

// GetEventCount returns the total number of stored events
func (m *MemoryPersistenceStore) GetEventCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.events)
}

// GetEventsByTimeRange retrieves events within a time range
func (m *MemoryPersistenceStore) GetEventsByTimeRange(since, until time.Time, limit int) ([]*Event, error) {
	filter := EventFilter{
		Since: &since,
		Until: &until,
	}
	return m.GetEvents(filter, limit, 0)
}

// GetEventsByType retrieves events of specific types
func (m *MemoryPersistenceStore) GetEventsByType(eventTypes []EventType, limit int) ([]*Event, error) {
	filter := EventFilter{
		Types: eventTypes,
	}
	return m.GetEvents(filter, limit, 0)
}

// GetEventsBySeverity retrieves events of specific severities
func (m *MemoryPersistenceStore) GetEventsBySeverity(severities []EventSeverity, limit int) ([]*Event, error) {
	filter := EventFilter{
		Severities: severities,
	}
	return m.GetEvents(filter, limit, 0)
}

// GetEventsByUser retrieves events for a specific user
func (m *MemoryPersistenceStore) GetEventsByUser(userID string, limit int) ([]*Event, error) {
	filter := EventFilter{
		UserID: &userID,
	}
	return m.GetEvents(filter, limit, 0)
}

// GetEventsByResource retrieves events for a specific resource
func (m *MemoryPersistenceStore) GetEventsByResource(resourceType, resourceID string, limit int) ([]*Event, error) {
	filter := EventFilter{
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
	}
	return m.GetEvents(filter, limit, 0)
}

// ClearEvents removes all events from storage
func (m *MemoryPersistenceStore) ClearEvents() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = make(map[string]*Event)
	return nil
}

// GetStats returns storage statistics
func (m *MemoryPersistenceStore) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_events": len(m.events),
		"max_events":   m.maxEvents,
		"memory_usage": len(m.events) * 1024, // Rough estimate
	}
}

// cleanupOldEvents removes the oldest events to maintain the maximum limit
func (m *MemoryPersistenceStore) cleanupOldEvents() {
	if len(m.events) <= m.maxEvents {
		return
	}

	// Convert to slice for sorting
	events := make([]*Event, 0, len(m.events))
	for _, event := range m.events {
		events = append(events, event)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Calculate how many to remove
	toRemove := len(events) - m.maxEvents + (m.maxEvents / 10) // Remove 10% extra to avoid frequent cleanup

	// Remove oldest events
	for i := 0; i < toRemove && i < len(events); i++ {
		delete(m.events, events[i].ID)
	}
}

// ExportEvents exports all events (useful for backup or migration)
func (m *MemoryPersistenceStore) ExportEvents() []*Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]*Event, 0, len(m.events))
	for _, event := range m.events {
		events = append(events, event)
	}

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events
}

// ImportEvents imports events into the store
func (m *MemoryPersistenceStore) ImportEvents(events []*Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, event := range events {
		if event != nil {
			m.events[event.ID] = event
		}
	}

	// Clean up if necessary
	if len(m.events) > m.maxEvents {
		m.cleanupOldEvents()
	}

	return nil
}