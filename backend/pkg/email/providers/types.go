package providers

import (
	"context"
	"time"
)

// EmailProvider defines the interface for email service providers
type EmailProvider interface {
	Send(ctx context.Context, message *Message) error
	ValidateConfig() error
	GetProviderName() string
}

// Message represents an email message
type Message struct {
	To          []string          `json:"to"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	From        string            `json:"from"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	Subject     string            `json:"subject"`
	TextBody    string            `json:"text_body,omitempty"`
	HTMLBody    string            `json:"html_body,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Priority    Priority          `json:"priority,omitempty"`
	Tracking    *TrackingOptions  `json:"tracking,omitempty"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
	Inline      bool   `json:"inline,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// Priority represents email priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// TrackingOptions represents email tracking options
type TrackingOptions struct {
	OpenTracking  bool `json:"open_tracking,omitempty"`
	ClickTracking bool `json:"click_tracking,omitempty"`
	Unsubscribe   bool `json:"unsubscribe,omitempty"`
}

// SendResult represents the result of sending an email
type SendResult struct {
	MessageID   string    `json:"message_id"`
	Status      string    `json:"status"`
	ProviderID  string    `json:"provider_id,omitempty"`
	SentAt      time.Time `json:"sent_at"`
	Error       string    `json:"error,omitempty"`
	RetryCount  int       `json:"retry_count"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// QueuedMessage represents a message in the send queue
type QueuedMessage struct {
	ID          string    `json:"id"`
	Message     *Message  `json:"message"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DeliveryStatus represents email delivery status
type DeliveryStatus struct {
	MessageID   string                 `json:"message_id"`
	Status      string                 `json:"status"` // sent, delivered, bounced, failed, etc.
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}