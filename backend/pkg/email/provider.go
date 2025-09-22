package email

import (
	"docker-auto/pkg/email/providers"
)

// Type aliases for backward compatibility
type EmailProvider = providers.EmailProvider
type Message = providers.Message
type Attachment = providers.Attachment
type Priority = providers.Priority
type TrackingOptions = providers.TrackingOptions
type SendResult = providers.SendResult
type QueuedMessage = providers.QueuedMessage
type DeliveryStatus = providers.DeliveryStatus

// Constants for backward compatibility
const (
	PriorityLow    = providers.PriorityLow
	PriorityNormal = providers.PriorityNormal
	PriorityHigh   = providers.PriorityHigh
	PriorityUrgent = providers.PriorityUrgent
)