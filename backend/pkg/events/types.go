package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	// Container lifecycle events
	EventContainerStarted   EventType = "container.started"
	EventContainerStopped   EventType = "container.stopped"
	EventContainerUpdated   EventType = "container.updated"
	EventContainerError     EventType = "container.error"
	EventContainerCreated   EventType = "container.created"
	EventContainerDeleted   EventType = "container.deleted"
	EventContainerRestarted EventType = "container.restarted"

	// Image update events
	EventImageUpdateAvailable EventType = "image.update_available"
	EventImageUpdateStarted   EventType = "image.update_started"
	EventImageUpdateCompleted EventType = "image.update_completed"
	EventImageUpdateFailed    EventType = "image.update_failed"
	EventImagePulled          EventType = "image.pulled"
	EventImageDeleted         EventType = "image.deleted"

	// System events
	EventSystemHealthChanged EventType = "system.health_changed"
	EventSystemResourceAlert EventType = "system.resource_alert"
	EventSystemStarted       EventType = "system.started"
	EventSystemStopped       EventType = "system.stopped"
	EventSystemError         EventType = "system.error"

	// User events
	EventUserLoggedIn        EventType = "user.logged_in"
	EventUserLoggedOut       EventType = "user.logged_out"
	EventUserPermissionChanged EventType = "user.permission_changed"
	EventUserCreated         EventType = "user.created"
	EventUserUpdated         EventType = "user.updated"
	EventUserDeleted         EventType = "user.deleted"

	// Scheduler events
	EventTaskStarted    EventType = "task.started"
	EventTaskCompleted  EventType = "task.completed"
	EventTaskFailed     EventType = "task.failed"
	EventTaskScheduled  EventType = "task.scheduled"
	EventTaskCancelled  EventType = "task.cancelled"

	// Notification events
	EventNotificationCreated EventType = "notification.created"
	EventNotificationRead    EventType = "notification.read"
	EventNotificationDeleted EventType = "notification.deleted"
)

// EventSeverity represents the severity level of an event
type EventSeverity string

const (
	SeverityInfo    EventSeverity = "info"
	SeverityWarning EventSeverity = "warning"
	SeverityError   EventSeverity = "error"
	SeveritySuccess EventSeverity = "success"
	SeverityDebug   EventSeverity = "debug"
)

// Event represents a system event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Severity    EventSeverity          `json:"severity"`
	Source      string                 `json:"source"`
	UserID      *string                `json:"user_id,omitempty"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Tags        []string               `json:"tags,omitempty"`
	ResourceID  *string                `json:"resource_id,omitempty"`
	ResourceType *string               `json:"resource_type,omitempty"`
}

// NewEvent creates a new event with generated ID and timestamp
func NewEvent(eventType EventType, severity EventSeverity, source, title, message string) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Severity:  severity,
		Source:    source,
		Title:     title,
		Message:   message,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
		Tags:      make([]string, 0),
	}
}

// WithUserID sets the user ID for the event
func (e *Event) WithUserID(userID string) *Event {
	e.UserID = &userID
	return e
}

// WithData sets custom data for the event
func (e *Event) WithData(key string, value interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// WithResource sets resource information for the event
func (e *Event) WithResource(resourceType, resourceID string) *Event {
	e.ResourceType = &resourceType
	e.ResourceID = &resourceID
	return e
}

// WithTags adds tags to the event
func (e *Event) WithTags(tags ...string) *Event {
	e.Tags = append(e.Tags, tags...)
	return e
}

// ToJSON serializes the event to JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// EventFilter represents criteria for filtering events
type EventFilter struct {
	Types        []EventType     `json:"types,omitempty"`
	Severities   []EventSeverity `json:"severities,omitempty"`
	Sources      []string        `json:"sources,omitempty"`
	UserID       *string         `json:"user_id,omitempty"`
	ResourceType *string         `json:"resource_type,omitempty"`
	ResourceID   *string         `json:"resource_id,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	Since        *time.Time      `json:"since,omitempty"`
	Until        *time.Time      `json:"until,omitempty"`
}

// Matches checks if an event matches the filter criteria
func (f *EventFilter) Matches(event *Event) bool {
	// Check event types
	if len(f.Types) > 0 {
		found := false
		for _, t := range f.Types {
			if t == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severities
	if len(f.Severities) > 0 {
		found := false
		for _, s := range f.Severities {
			if s == event.Severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check sources
	if len(f.Sources) > 0 {
		found := false
		for _, s := range f.Sources {
			if s == event.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check user ID
	if f.UserID != nil {
		if event.UserID == nil || *event.UserID != *f.UserID {
			return false
		}
	}

	// Check resource type
	if f.ResourceType != nil {
		if event.ResourceType == nil || *event.ResourceType != *f.ResourceType {
			return false
		}
	}

	// Check resource ID
	if f.ResourceID != nil {
		if event.ResourceID == nil || *event.ResourceID != *f.ResourceID {
			return false
		}
	}

	// Check tags
	if len(f.Tags) > 0 {
		for _, filterTag := range f.Tags {
			found := false
			for _, eventTag := range event.Tags {
				if eventTag == filterTag {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check time range
	if f.Since != nil && event.Timestamp.Before(*f.Since) {
		return false
	}
	if f.Until != nil && event.Timestamp.After(*f.Until) {
		return false
	}

	return true
}

// Subscription represents an event subscription
type Subscription struct {
	ID       string       `json:"id"`
	UserID   *string      `json:"user_id,omitempty"`
	Filter   EventFilter  `json:"filter"`
	Channel  chan *Event  `json:"-"`
	Active   bool         `json:"active"`
	Created  time.Time    `json:"created"`
}

// NewSubscription creates a new event subscription
func NewSubscription(filter EventFilter) *Subscription {
	return &Subscription{
		ID:      uuid.New().String(),
		Filter:  filter,
		Channel: make(chan *Event, 100), // Buffered channel
		Active:  true,
		Created: time.Now(),
	}
}

// WithUser sets the user ID for the subscription
func (s *Subscription) WithUser(userID string) *Subscription {
	s.UserID = &userID
	return s
}