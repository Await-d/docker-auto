package types

import (
	"context"
	"sync"
	"time"
)

// WebhookSender defines the interface for webhook delivery
type WebhookSender interface {
	Send(ctx context.Context, payload *WebhookPayload) error
	ValidateConfig() error
	GetName() string
}

// WebhookPayload represents a webhook payload
type WebhookPayload struct {
	ID          string                 `json:"id"`
	Event       string                 `json:"event"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	Priority    WebhookPriority        `json:"priority,omitempty"`
}

// WebhookPriority represents webhook priority levels
type WebhookPriority int

const (
	// WebhookPriorityLow represents low priority webhook
	WebhookPriorityLow WebhookPriority = iota
	// WebhookPriorityNormal represents normal priority webhook
	WebhookPriorityNormal
	// WebhookPriorityHigh represents high priority webhook
	WebhookPriorityHigh
	// WebhookPriorityUrgent represents urgent priority webhook
	WebhookPriorityUrgent
)

// String returns the string representation of WebhookPriority
func (p WebhookPriority) String() string {
	switch p {
	case WebhookPriorityLow:
		return "low"
	case WebhookPriorityNormal:
		return "normal"
	case WebhookPriorityHigh:
		return "high"
	case WebhookPriorityUrgent:
		return "urgent"
	default:
		return "normal"
	}
}

// WebhookDeliveryResult represents the result of webhook delivery
type WebhookDeliveryResult struct {
	ID              string            `json:"id"`
	URL             string            `json:"url"`
	StatusCode      int               `json:"status_code"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody    string            `json:"response_body,omitempty"`
	DeliveredAt     time.Time         `json:"delivered_at"`
	Duration        time.Duration     `json:"duration"`
	Success         bool              `json:"success"`
	Error           string            `json:"error,omitempty"`
	RetryCount      int               `json:"retry_count"`
}

// WebhookQueuedMessage represents a webhook in the delivery queue
type WebhookQueuedMessage struct {
	ID          string           `json:"id"`
	Payload     *WebhookPayload  `json:"payload"`
	URL         string           `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	ScheduledAt time.Time        `json:"scheduled_at"`
	Attempts    int              `json:"attempts"`
	LastError   string           `json:"last_error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL             string            `json:"url"`
	Secret          string            `json:"secret,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	SignatureHeader string            `json:"signature_header,omitempty"`
	VerifySSL       bool              `json:"verify_ssl"`
	Timeout         time.Duration     `json:"timeout"`
	RetryAttempts   int               `json:"retry_attempts"`
	RetryDelay      time.Duration     `json:"retry_delay"`

	// Rate limiting
	RateLimit       int           `json:"rate_limit,omitempty"`        // requests per minute
	RateLimitBurst  int           `json:"rate_limit_burst,omitempty"`  // burst size
	RateLimitWindow time.Duration `json:"rate_limit_window,omitempty"` // window size
}

// WebhookDeliveryStatus represents webhook delivery status
type WebhookDeliveryStatus string

const (
	// WebhookDeliveryStatusPending indicates webhook is pending delivery
	WebhookDeliveryStatusPending WebhookDeliveryStatus = "pending"
	// WebhookDeliveryStatusSent indicates webhook has been sent
	WebhookDeliveryStatusSent WebhookDeliveryStatus = "sent"
	// WebhookDeliveryStatusDelivered indicates webhook was successfully delivered
	WebhookDeliveryStatusDelivered WebhookDeliveryStatus = "delivered"
	// WebhookDeliveryStatusFailed indicates webhook delivery failed
	WebhookDeliveryStatusFailed WebhookDeliveryStatus = "failed"
	// WebhookDeliveryStatusRetrying indicates webhook is being retried
	WebhookDeliveryStatusRetrying WebhookDeliveryStatus = "retrying"
	// WebhookDeliveryStatusCancelled indicates webhook delivery was cancelled
	WebhookDeliveryStatusCancelled WebhookDeliveryStatus = "cancelled"
)

// WebhookEvent represents different webhook event types
type WebhookEvent string

const (
	// WebhookEventContainerUpdate represents container update events
	WebhookEventContainerUpdate WebhookEvent = "container.update"
	// WebhookEventContainerFailure represents container failure events
	WebhookEventContainerFailure WebhookEvent = "container.failure"
	// WebhookEventContainerStart represents container start events
	WebhookEventContainerStart WebhookEvent = "container.start"
	// WebhookEventContainerStop represents container stop events
	WebhookEventContainerStop WebhookEvent = "container.stop"
	// WebhookEventImageUpdate represents image update events
	WebhookEventImageUpdate WebhookEvent = "image.update"
	// WebhookEventSystemAlert represents system alert events
	WebhookEventSystemAlert WebhookEvent = "system.alert"
	// WebhookEventSecurityAlert represents security alert events
	WebhookEventSecurityAlert WebhookEvent = "security.alert"
	// WebhookEventUserAction represents user action events
	WebhookEventUserAction WebhookEvent = "user.action"
	// WebhookEventScheduledTask represents scheduled task events
	WebhookEventScheduledTask WebhookEvent = "task.scheduled"
	// WebhookEventTaskComplete represents task completion events
	WebhookEventTaskComplete WebhookEvent = "task.complete"
	// WebhookEventTaskFailed represents task failure events
	WebhookEventTaskFailed WebhookEvent = "task.failed"
)

// WebhookMetrics tracks webhook delivery metrics
// NOTE: Added missing mu field as specified in requirements
type WebhookMetrics struct {
	mu                  sync.RWMutex // Thread-safe access to metrics
	TotalSent           int64        `json:"total_sent"`
	TotalFailed         int64        `json:"total_failed"`
	TotalQueued         int64        `json:"total_queued"`
	CurrentQueueSize    int64        `json:"current_queue_size"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	SuccessRate         float64      `json:"success_rate"`
	LastSentAt          *time.Time   `json:"last_sent_at,omitempty"`
	LastFailureAt       *time.Time   `json:"last_failure_at,omitempty"`

	// Status code distribution
	StatusCodes map[int]int64 `json:"status_codes,omitempty"`

	// Response time percentiles
	ResponseTimeP50 time.Duration `json:"response_time_p50,omitempty"`
	ResponseTimeP95 time.Duration `json:"response_time_p95,omitempty"`
	ResponseTimeP99 time.Duration `json:"response_time_p99,omitempty"`
}

// Lock provides thread-safe write access to webhook metrics
func (wm *WebhookMetrics) Lock() {
	wm.mu.Lock()
}

// Unlock releases the write lock on webhook metrics
func (wm *WebhookMetrics) Unlock() {
	wm.mu.Unlock()
}

// RLock provides thread-safe read access to webhook metrics
func (wm *WebhookMetrics) RLock() {
	wm.mu.RLock()
}

// RUnlock releases the read lock on webhook metrics
func (wm *WebhookMetrics) RUnlock() {
	wm.mu.RUnlock()
}

// WebhookFilter represents filters for webhook queries
type WebhookFilter struct {
	Event      WebhookEvent          `json:"event,omitempty"`
	Status     WebhookDeliveryStatus `json:"status,omitempty"`
	URL        string                `json:"url,omitempty"`
	From       *time.Time            `json:"from,omitempty"`
	To         *time.Time            `json:"to,omitempty"`
	Limit      int                   `json:"limit,omitempty"`
	Offset     int                   `json:"offset,omitempty"`
}

// WebhookRequest represents an HTTP request for webhook delivery
type WebhookRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        []byte            `json:"body"`
	Timeout     time.Duration     `json:"timeout"`
	RetryCount  int               `json:"retry_count"`
	Signature   string            `json:"signature,omitempty"`
	ContentType string            `json:"content_type"`
}

// WebhookResponse represents an HTTP response from webhook delivery
type WebhookResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        []byte            `json:"body"`
	Duration    time.Duration     `json:"duration"`
	Error       string            `json:"error,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// WebhookEndpoint represents a webhook endpoint configuration
type WebhookEndpoint struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	URL         string               `json:"url"`
	Secret      string               `json:"secret,omitempty"`
	Events      []WebhookEvent       `json:"events"`
	Headers     map[string]string    `json:"headers,omitempty"`
	IsActive    bool                 `json:"is_active"`
	VerifySSL   bool                 `json:"verify_ssl"`
	Timeout     time.Duration        `json:"timeout"`
	MaxRetries  int                  `json:"max_retries"`
	RetryDelay  time.Duration        `json:"retry_delay"`
	RateLimit   *WebhookRateLimit    `json:"rate_limit,omitempty"`
	Filters     *WebhookEventFilters `json:"filters,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	LastUsed    *time.Time           `json:"last_used,omitempty"`
	Stats       *WebhookEndpointStats `json:"stats,omitempty"`
}

// WebhookRateLimit represents rate limiting configuration for webhooks
type WebhookRateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	BurstSize         int           `json:"burst_size"`
	WindowSize        time.Duration `json:"window_size"`
}

// WebhookEventFilters represents filters for webhook events
type WebhookEventFilters struct {
	IncludePatterns []string               `json:"include_patterns,omitempty"`
	ExcludePatterns []string               `json:"exclude_patterns,omitempty"`
	Conditions      map[string]interface{} `json:"conditions,omitempty"`
}

// WebhookEndpointStats represents statistics for a webhook endpoint
type WebhookEndpointStats struct {
	TotalDeliveries    int64         `json:"total_deliveries"`
	SuccessfulDeliveries int64       `json:"successful_deliveries"`
	FailedDeliveries   int64         `json:"failed_deliveries"`
	LastDeliveryAt     *time.Time    `json:"last_delivery_at,omitempty"`
	LastSuccessAt      *time.Time    `json:"last_success_at,omitempty"`
	LastFailureAt      *time.Time    `json:"last_failure_at,omitempty"`
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	SuccessRate        float64       `json:"success_rate"`
	StatusCodeCounts   map[int]int64 `json:"status_code_counts,omitempty"`
}

// WebhookNotificationConfig represents configuration for webhook notifications
type WebhookNotificationConfig struct {
	Enabled          bool                     `json:"enabled"`
	DefaultEndpoint  string                   `json:"default_endpoint"`
	FallbackEndpoints []string                `json:"fallback_endpoints,omitempty"`
	EventMappings    map[string][]string      `json:"event_mappings"`    // event -> endpoints mapping
	RetryPolicy      *WebhookRetryPolicy      `json:"retry_policy,omitempty"`
	RateLimits       map[string]int           `json:"rate_limits,omitempty"` // per-event rate limits
	Batching         *WebhookBatchConfig      `json:"batching,omitempty"`
	Security         *WebhookSecurityConfig   `json:"security,omitempty"`
	Monitoring       *WebhookMonitoringConfig `json:"monitoring,omitempty"`
}

// WebhookRetryPolicy represents retry policy for failed webhook deliveries
type WebhookRetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableStatus []int         `json:"retryable_status,omitempty"` // HTTP status codes to retry
}

// WebhookBatchConfig represents configuration for webhook batching
type WebhookBatchConfig struct {
	Enabled       bool          `json:"enabled"`
	MaxBatchSize  int           `json:"max_batch_size"`
	MaxWaitTime   time.Duration `json:"max_wait_time"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// WebhookSecurityConfig represents security configuration for webhooks
type WebhookSecurityConfig struct {
	SignatureAlgorithm string   `json:"signature_algorithm"` // hmac-sha256, hmac-sha1, etc.
	SignatureHeader    string   `json:"signature_header"`
	AllowedIPs         []string `json:"allowed_ips,omitempty"`
	RequireHTTPS       bool     `json:"require_https"`
	VerifySSL          bool     `json:"verify_ssl"`
	MaxPayloadSize     int64    `json:"max_payload_size"` // bytes
}

// WebhookMonitoringConfig represents monitoring configuration for webhooks
type WebhookMonitoringConfig struct {
	MetricsEnabled   bool          `json:"metrics_enabled"`
	LoggingEnabled   bool          `json:"logging_enabled"`
	AlertsEnabled    bool          `json:"alerts_enabled"`
	HealthCheckURL   string        `json:"health_check_url,omitempty"`
	HealthCheckInterval time.Duration `json:"health_check_interval,omitempty"`
	FailureThreshold int           `json:"failure_threshold"`      // failures before alert
	ResponseTimeThreshold time.Duration `json:"response_time_threshold"` // alert if response time exceeds
}

// GetPriorityValue returns the numeric value of the priority
func (p WebhookPriority) GetPriorityValue() int {
	return int(p)
}

// IsHigherPriority checks if this priority is higher than another
func (p WebhookPriority) IsHigherPriority(other WebhookPriority) bool {
	return p > other
}

// IsRetryable checks if a status code is retryable based on retry policy
func (r *WebhookRetryPolicy) IsRetryable(statusCode int) bool {
	if r == nil || len(r.RetryableStatus) == 0 {
		// Default retryable status codes (5xx errors)
		return statusCode >= 500 && statusCode < 600
	}

	for _, code := range r.RetryableStatus {
		if code == statusCode {
			return true
		}
	}
	return false
}

// CalculateDelay calculates the retry delay based on attempt number
func (r *WebhookRetryPolicy) CalculateDelay(attempt int) time.Duration {
	if r == nil {
		return time.Second
	}

	delay := r.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * r.BackoffFactor)
		if delay > r.MaxDelay {
			return r.MaxDelay
		}
	}
	return delay
}

// HasEvent checks if the endpoint is configured for a specific event
func (e *WebhookEndpoint) HasEvent(event WebhookEvent) bool {
	for _, e := range e.Events {
		if e == event {
			return true
		}
	}
	return false
}

// IsHealthy checks if the endpoint is healthy based on recent stats
func (e *WebhookEndpoint) IsHealthy() bool {
	if e.Stats == nil {
		return true // No stats available, assume healthy
	}

	// Consider healthy if success rate > 90% and last success within 24 hours
	return e.Stats.SuccessRate > 0.9 &&
		e.Stats.LastSuccessAt != nil &&
		time.Since(*e.Stats.LastSuccessAt) < 24*time.Hour
}

// EstimateSize estimates the size of the webhook payload in bytes
func (p *WebhookPayload) EstimateSize() int64 {
	// Basic estimation - should be enhanced for accurate calculation
	size := int64(len(p.ID) + len(p.Event) + len(p.Source))

	// Estimate data size (rough approximation)
	for k, v := range p.Data {
		size += int64(len(k))
		if str, ok := v.(string); ok {
			size += int64(len(str))
		} else {
			size += 50 // rough estimate for other types
		}
	}

	for k, v := range p.Metadata {
		size += int64(len(k) + len(v))
	}

	return size
}