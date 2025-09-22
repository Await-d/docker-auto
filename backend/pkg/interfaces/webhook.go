package interfaces

import (
	"context"
	"time"

	"docker-auto/pkg/types"
)

// WebhookSender defines the interface for webhook delivery
type WebhookSender interface {
	// Core webhook operations
	Send(ctx context.Context, payload *types.WebhookPayload) (*types.WebhookDeliveryResult, error)
	SendBatch(ctx context.Context, payloads []*types.WebhookPayload) ([]*types.WebhookDeliveryResult, error)
	SendToEndpoint(ctx context.Context, endpointID string, payload *types.WebhookPayload) (*types.WebhookDeliveryResult, error)

	// Webhook management
	ValidateConfig() error
	GetName() string
	GetSupportedEvents() []types.WebhookEvent
	IsHealthy(ctx context.Context) (bool, error)

	// Configuration and lifecycle
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	UpdateConfig(config *types.WebhookConfig) error
}

// WebhookValidator defines the interface for webhook validation
type WebhookValidator interface {
	// Payload validation
	ValidatePayload(payload *types.WebhookPayload) (*WebhookValidationResult, error)
	ValidateSignature(payload []byte, signature string, secret string) error
	ValidateEndpoint(endpoint *types.WebhookEndpoint) (*EndpointValidationResult, error)

	// URL validation
	ValidateURL(url string) (*URLValidationResult, error)
	TestEndpoint(ctx context.Context, endpoint *types.WebhookEndpoint, testPayload *types.WebhookPayload) (*EndpointTestResult, error)

	// Security validation
	ValidateSecurityConfig(config *types.WebhookSecurityConfig) (*SecurityValidationResult, error)
	CheckRateLimit(endpointID string, currentTime time.Time) (*RateLimitResult, error)
}

// WebhookQueue defines the interface for webhook queue management
type WebhookQueue interface {
	// Queue operations
	Enqueue(ctx context.Context, webhook *types.WebhookQueuedMessage, priority types.WebhookPriority) error
	EnqueueBatch(ctx context.Context, webhooks []*types.WebhookQueuedMessage) error
	Dequeue(ctx context.Context, maxItems int) ([]*types.WebhookQueuedMessage, error)
	Requeue(ctx context.Context, webhookID string, delay time.Duration) error
	Remove(ctx context.Context, webhookID string) error

	// Queue monitoring
	GetQueueDepth(ctx context.Context) (int64, error)
	GetQueuedWebhooks(ctx context.Context, filter *QueuedWebhookFilter) ([]*types.WebhookQueuedMessage, error)
	GetQueueStats(ctx context.Context) (*WebhookQueueStats, error)

	// Queue processing
	StartProcessing(ctx context.Context) error
	StopProcessing(ctx context.Context) error
	PauseProcessing(ctx context.Context) error
	ResumeProcessing(ctx context.Context) error
	IsProcessing() bool

	// Queue management
	PurgeQueue(ctx context.Context) (int64, error)
	PurgeOldWebhooks(ctx context.Context, olderThan time.Duration) (int64, error)
	GetFailedWebhooks(ctx context.Context, filter *FailedWebhookFilter) ([]*types.WebhookQueuedMessage, error)
	RetryFailedWebhooks(ctx context.Context, webhookIDs []string) error
}

// WebhookEndpointManager defines the interface for managing webhook endpoints
type WebhookEndpointManager interface {
	// Endpoint lifecycle
	Create(ctx context.Context, endpoint *types.WebhookEndpoint) error
	Update(ctx context.Context, endpointID string, endpoint *types.WebhookEndpoint) error
	Delete(ctx context.Context, endpointID string) error
	Get(ctx context.Context, endpointID string) (*types.WebhookEndpoint, error)
	List(ctx context.Context, filter *WebhookEndpointFilter) ([]*types.WebhookEndpoint, error)

	// Endpoint configuration
	UpdateEndpointConfig(ctx context.Context, endpointID string, config *types.WebhookConfig) error
	GetEndpointConfig(ctx context.Context, endpointID string) (*types.WebhookConfig, error)
	TestEndpoint(ctx context.Context, endpointID string, testPayload *types.WebhookPayload) (*EndpointTestResult, error)

	// Endpoint monitoring
	GetEndpointStats(ctx context.Context, endpointID string) (*types.WebhookEndpointStats, error)
	GetEndpointHealth(ctx context.Context, endpointID string) (*EndpointHealthStatus, error)
	GetEndpointHistory(ctx context.Context, endpointID string, filter *EndpointHistoryFilter) ([]*WebhookDeliveryHistory, error)

	// Endpoint events
	EnableEndpoint(ctx context.Context, endpointID string) error
	DisableEndpoint(ctx context.Context, endpointID string) error
	AddEventToEndpoint(ctx context.Context, endpointID string, event types.WebhookEvent) error
	RemoveEventFromEndpoint(ctx context.Context, endpointID string, event types.WebhookEvent) error

	// Bulk operations
	BulkUpdate(ctx context.Context, filter *WebhookEndpointFilter, updates *BulkEndpointUpdate) (*BulkUpdateResult, error)
	BulkDisable(ctx context.Context, endpointIDs []string) (*BulkUpdateResult, error)
	BulkEnable(ctx context.Context, endpointIDs []string) (*BulkUpdateResult, error)
}

// WebhookService defines the main interface for the webhook service
type WebhookService interface {
	// Core webhook operations
	SendWebhook(ctx context.Context, event types.WebhookEvent, data map[string]interface{}) error
	SendWebhookToEndpoint(ctx context.Context, endpointID string, payload *types.WebhookPayload) (*types.WebhookDeliveryResult, error)
	SendWebhookAsync(ctx context.Context, event types.WebhookEvent, data map[string]interface{}) error

	// Event management
	TriggerEvent(ctx context.Context, event types.WebhookEvent, data map[string]interface{}) error
	RegisterEventHandler(eventType types.WebhookEvent, handler WebhookEventHandler) error
	UnregisterEventHandler(eventType types.WebhookEvent) error
	GetSupportedEvents() []types.WebhookEvent

	// Webhook management
	CreateWebhook(ctx context.Context, endpoint *types.WebhookEndpoint) error
	UpdateWebhook(ctx context.Context, endpointID string, endpoint *types.WebhookEndpoint) error
	DeleteWebhook(ctx context.Context, endpointID string) error
	GetWebhook(ctx context.Context, endpointID string) (*types.WebhookEndpoint, error)
	ListWebhooks(ctx context.Context, filter *types.WebhookFilter) ([]*types.WebhookEndpoint, error)

	// Delivery tracking
	GetDeliveryStatus(ctx context.Context, deliveryID string) (*types.WebhookDeliveryResult, error)
	GetDeliveryHistory(ctx context.Context, endpointID string, filter *DeliveryHistoryFilter) ([]*types.WebhookDeliveryResult, error)
	GetFailedDeliveries(ctx context.Context, filter *FailedDeliveryFilter) ([]*types.WebhookDeliveryResult, error)
	RetryFailedDelivery(ctx context.Context, deliveryID string) error

	// Configuration
	UpdateNotificationConfig(ctx context.Context, config *types.WebhookNotificationConfig) error
	GetNotificationConfig(ctx context.Context) (*types.WebhookNotificationConfig, error)

	// Analytics and monitoring
	GetWebhookMetrics(ctx context.Context, period time.Duration) (*types.WebhookMetrics, error)
	GetEndpointMetrics(ctx context.Context, endpointID string, period time.Duration) (*types.WebhookEndpointStats, error)
	GenerateWebhookReport(ctx context.Context, options *WebhookReportOptions) (*WebhookReport, error)

	// Health and status
	HealthCheck(ctx context.Context) (*WebhookServiceHealth, error)
	GetSystemStatus(ctx context.Context) (*WebhookSystemStatus, error)
}

// WebhookEventHandler defines the interface for handling webhook events
type WebhookEventHandler interface {
	HandleEvent(ctx context.Context, event types.WebhookEvent, data map[string]interface{}) error
	GetHandlerName() string
	GetSupportedEvents() []types.WebhookEvent
	GetPriority() int
}

// WebhookSignatureGenerator defines the interface for generating webhook signatures
type WebhookSignatureGenerator interface {
	GenerateSignature(payload []byte, secret string, algorithm string) (string, error)
	ValidateSignature(payload []byte, signature string, secret string, algorithm string) error
	GetSupportedAlgorithms() []string
}

// WebhookRateLimiter defines the interface for webhook rate limiting
type WebhookRateLimiter interface {
	// Rate limiting
	CheckLimit(ctx context.Context, endpointID string) (*RateLimitResult, error)
	RecordRequest(ctx context.Context, endpointID string) error
	ResetLimit(ctx context.Context, endpointID string) error

	// Rate limit configuration
	SetRateLimit(ctx context.Context, endpointID string, limit *types.WebhookRateLimit) error
	GetRateLimit(ctx context.Context, endpointID string) (*types.WebhookRateLimit, error)
	RemoveRateLimit(ctx context.Context, endpointID string) error

	// Rate limit monitoring
	GetRateLimitStatus(ctx context.Context, endpointID string) (*RateLimitStatus, error)
	GetRateLimitStats(ctx context.Context, endpointID string, period time.Duration) (*RateLimitStats, error)
}

// WebhookRetryManager defines the interface for managing webhook retries
type WebhookRetryManager interface {
	// Retry management
	ScheduleRetry(ctx context.Context, deliveryID string, attempt int) error
	GetRetrySchedule(ctx context.Context, deliveryID string) (*RetrySchedule, error)
	CancelRetry(ctx context.Context, deliveryID string) error

	// Retry policy
	SetRetryPolicy(ctx context.Context, endpointID string, policy *types.WebhookRetryPolicy) error
	GetRetryPolicy(ctx context.Context, endpointID string) (*types.WebhookRetryPolicy, error)
	RemoveRetryPolicy(ctx context.Context, endpointID string) error

	// Retry monitoring
	GetRetryStats(ctx context.Context, endpointID string, period time.Duration) (*RetryStats, error)
	GetFailedRetries(ctx context.Context, filter *FailedRetryFilter) ([]*FailedRetry, error)
	PurgeOldRetries(ctx context.Context, olderThan time.Duration) (int64, error)
}

// WebhookMonitor defines the interface for webhook monitoring
type WebhookMonitor interface {
	// Monitoring
	StartMonitoring(ctx context.Context) error
	StopMonitoring(ctx context.Context) error
	IsMonitoring() bool

	// Health monitoring
	CheckEndpointHealth(ctx context.Context, endpointID string) (*EndpointHealthStatus, error)
	GetUnhealthyEndpoints(ctx context.Context) ([]*types.WebhookEndpoint, error)
	MonitorEndpointHealth(ctx context.Context, endpointID string, interval time.Duration) (<-chan *EndpointHealthStatus, error)

	// Performance monitoring
	GetPerformanceMetrics(ctx context.Context, endpointID string, period time.Duration) (*EndpointPerformanceMetrics, error)
	GetSystemPerformance(ctx context.Context, period time.Duration) (*WebhookSystemPerformance, error)

	// Alert management
	CreateAlert(ctx context.Context, alert *WebhookAlert) error
	UpdateAlert(ctx context.Context, alertID string, alert *WebhookAlert) error
	DeleteAlert(ctx context.Context, alertID string) error
	GetAlerts(ctx context.Context, filter *WebhookAlertFilter) ([]*WebhookAlert, error)
	TriggerAlert(ctx context.Context, alert *WebhookAlert, data map[string]interface{}) error

	// Event monitoring
	MonitorEvents(ctx context.Context, events []types.WebhookEvent) (<-chan *WebhookEventMonitor, error)
	GetEventStats(ctx context.Context, period time.Duration) (*WebhookEventStats, error)
}

// WebhookSecurity defines the interface for webhook security
type WebhookSecurity interface {
	// Security validation
	ValidateRequest(ctx context.Context, request *WebhookSecurityRequest) (*SecurityValidationResult, error)
	ValidateIP(ctx context.Context, endpointID string, clientIP string) error
	ValidateHTTPS(ctx context.Context, url string) error

	// Security configuration
	SetSecurityConfig(ctx context.Context, endpointID string, config *types.WebhookSecurityConfig) error
	GetSecurityConfig(ctx context.Context, endpointID string) (*types.WebhookSecurityConfig, error)
	RemoveSecurityConfig(ctx context.Context, endpointID string) error

	// Security monitoring
	GetSecurityEvents(ctx context.Context, filter *SecurityEventFilter) ([]*WebhookSecurityEvent, error)
	GetSecurityStats(ctx context.Context, period time.Duration) (*WebhookSecurityStats, error)
	DetectAnomalies(ctx context.Context, endpointID string, period time.Duration) ([]*SecurityAnomaly, error)

	// Threat protection
	BlacklistIP(ctx context.Context, ip string, reason string, duration time.Duration) error
	WhitelistIP(ctx context.Context, ip string) error
	GetBlacklist(ctx context.Context) ([]*BlacklistEntry, error)
	GetWhitelist(ctx context.Context) ([]*WhitelistEntry, error)
}

// Supporting types and structures

// WebhookValidationResult represents the result of webhook validation
type WebhookValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// EndpointValidationResult represents the result of endpoint validation
type EndpointValidationResult struct {
	IsValid       bool                    `json:"is_valid"`
	Errors        []string                `json:"errors,omitempty"`
	Warnings      []string                `json:"warnings,omitempty"`
	ResponseTime  time.Duration           `json:"response_time"`
	StatusCode    int                     `json:"status_code"`
	SSLInfo       *SSLValidationInfo      `json:"ssl_info,omitempty"`
	SecurityIssues []SecurityIssue        `json:"security_issues,omitempty"`
}

// URLValidationResult represents the result of URL validation
type URLValidationResult struct {
	IsValid     bool          `json:"is_valid"`
	IsReachable bool          `json:"is_reachable"`
	StatusCode  int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	Errors      []string      `json:"errors,omitempty"`
	Warnings    []string      `json:"warnings,omitempty"`
}

// EndpointTestResult represents the result of endpoint testing
type EndpointTestResult struct {
	Success        bool                    `json:"success"`
	StatusCode     int                     `json:"status_code"`
	ResponseTime   time.Duration           `json:"response_time"`
	ResponseBody   string                  `json:"response_body,omitempty"`
	ResponseHeaders map[string]string      `json:"response_headers,omitempty"`
	Error          string                  `json:"error,omitempty"`
	Timestamp      time.Time               `json:"timestamp"`
}

// SecurityValidationResult represents the result of security validation
type SecurityValidationResult struct {
	IsSecure     bool              `json:"is_secure"`
	Issues       []SecurityIssue   `json:"issues,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
	SecurityScore   int            `json:"security_score"` // 0-100
}

// RateLimitResult represents the result of rate limit checking
type RateLimitResult struct {
	Allowed       bool          `json:"allowed"`
	Remaining     int           `json:"remaining"`
	ResetTime     time.Time     `json:"reset_time"`
	RetryAfter    time.Duration `json:"retry_after,omitempty"`
}

// QueuedWebhookFilter represents filters for queued webhook queries
type QueuedWebhookFilter struct {
	EndpointID   string                        `json:"endpoint_id,omitempty"`
	Status       types.WebhookDeliveryStatus   `json:"status,omitempty"`
	Priority     types.WebhookPriority         `json:"priority,omitempty"`
	Event        types.WebhookEvent            `json:"event,omitempty"`
	FromTime     *time.Time                    `json:"from_time,omitempty"`
	ToTime       *time.Time                    `json:"to_time,omitempty"`
	MaxAttempts  *int                          `json:"max_attempts,omitempty"`
	Limit        int                           `json:"limit,omitempty"`
	Offset       int                           `json:"offset,omitempty"`
}

// WebhookQueueStats represents statistics for the webhook queue
type WebhookQueueStats struct {
	QueueDepth        int64                                          `json:"queue_depth"`
	ProcessingCount   int64                                          `json:"processing_count"`
	CompletedCount    int64                                          `json:"completed_count"`
	FailedCount       int64                                          `json:"failed_count"`
	AverageProcessTime time.Duration                                 `json:"average_process_time"`
	QueueDepthByPriority map[types.WebhookPriority]int64            `json:"queue_depth_by_priority"`
	QueueDepthByEvent map[types.WebhookEvent]int64                  `json:"queue_depth_by_event"`
	LastProcessed     *time.Time                                    `json:"last_processed,omitempty"`
}

// Additional supporting types (abbreviated for space)
type FailedWebhookFilter struct{}
type WebhookEndpointFilter struct{}
type EndpointHealthStatus struct{}
type EndpointHistoryFilter struct{}
type WebhookDeliveryHistory struct{}
type BulkEndpointUpdate struct{}
type BulkUpdateResult struct{}
type DeliveryHistoryFilter struct{}
type FailedDeliveryFilter struct{}
type WebhookReportOptions struct{}
type WebhookReport struct{}
type WebhookServiceHealth struct{}
type WebhookSystemStatus struct{}
type RateLimitStatus struct{}
type RateLimitStats struct{}
type RetrySchedule struct{}
type RetryStats struct{}
type FailedRetryFilter struct{}
type FailedRetry struct{}
type EndpointPerformanceMetrics struct{}
type WebhookSystemPerformance struct{}
type WebhookAlert struct{}
type WebhookAlertFilter struct{}
type WebhookEventMonitor struct{}
type WebhookEventStats struct{}
type WebhookSecurityRequest struct{}
type SecurityEventFilter struct{}
type WebhookSecurityEvent struct{}
type WebhookSecurityStats struct{}
type SecurityAnomaly struct{}
type BlacklistEntry struct{}
type WhitelistEntry struct{}
type SSLValidationInfo struct{}
type SecurityIssue struct{}