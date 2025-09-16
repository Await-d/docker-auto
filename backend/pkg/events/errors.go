package events

import "errors"

var (
	// ErrInvalidEvent is returned when an event is invalid
	ErrInvalidEvent = errors.New("invalid event")

	// ErrSubscriptionNotFound is returned when a subscription is not found
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrPublisherClosed is returned when trying to use a closed publisher
	ErrPublisherClosed = errors.New("publisher is closed")

	// ErrInvalidFilter is returned when an event filter is invalid
	ErrInvalidFilter = errors.New("invalid event filter")

	// ErrPersistenceFailed is returned when event persistence fails
	ErrPersistenceFailed = errors.New("event persistence failed")

	// ErrChannelClosed is returned when trying to send to a closed channel
	ErrChannelClosed = errors.New("event channel is closed")

	// ErrMaxSubscriptionsExceeded is returned when max subscriptions limit is reached
	ErrMaxSubscriptionsExceeded = errors.New("maximum subscriptions limit exceeded")

	// ErrRateLimitExceeded is returned when event publishing rate limit is exceeded
	ErrRateLimitExceeded = errors.New("event publishing rate limit exceeded")
)