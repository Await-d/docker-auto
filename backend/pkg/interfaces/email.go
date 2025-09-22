package interfaces

import (
	"context"
	"time"

	"docker-auto/pkg/types"
)

// EmailProvider defines the interface for email service providers
type EmailProvider interface {
	// Core email operations
	Send(ctx context.Context, message *types.EmailMessage) (*types.EmailSendResult, error)
	SendBatch(ctx context.Context, messages []*types.EmailMessage) ([]*types.EmailSendResult, error)
	SendTemplate(ctx context.Context, templateID string, data *types.EmailTemplateData, recipients []string) (*types.EmailSendResult, error)

	// Provider management
	ValidateConfig() error
	GetProviderName() string
	GetProviderType() string
	IsHealthy(ctx context.Context) (bool, error)
	GetLimits() *EmailProviderLimits
	GetStats(ctx context.Context) (*EmailProviderStats, error)

	// Configuration and lifecycle
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	UpdateConfig(config *types.EmailProviderConfig) error
}

// TemplateManager defines the interface for email template management
type TemplateManager interface {
	// Template lifecycle
	Create(ctx context.Context, template *types.EmailTemplate) error
	Update(ctx context.Context, templateID string, template *types.EmailTemplate) error
	Delete(ctx context.Context, templateID string) error
	Get(ctx context.Context, templateID string) (*types.EmailTemplate, error)
	List(ctx context.Context, filter *EmailTemplateFilter) ([]*types.EmailTemplate, error)

	// Template rendering
	Render(ctx context.Context, templateID string, data *types.EmailTemplateData) (*types.EmailMessage, error)
	RenderPreview(ctx context.Context, templateID string, data *types.EmailTemplateData) (*EmailPreview, error)
	ValidateTemplate(ctx context.Context, template *types.EmailTemplate) (*TemplateValidationResult, error)

	// Template versions
	CreateVersion(ctx context.Context, templateID string, template *types.EmailTemplate) error
	GetVersion(ctx context.Context, templateID string, version int) (*types.EmailTemplate, error)
	ListVersions(ctx context.Context, templateID string) ([]*types.EmailTemplate, error)
	SetActiveVersion(ctx context.Context, templateID string, version int) error

	// Template testing
	TestTemplate(ctx context.Context, templateID string, data *types.EmailTemplateData, testRecipients []string) (*TemplateTestResult, error)
	ValidateVariables(ctx context.Context, templateID string, variables map[string]interface{}) (*VariableValidationResult, error)

	// Template analytics
	GetTemplateUsage(ctx context.Context, templateID string, period time.Duration) (*TemplateUsageStats, error)
	GetTemplatePerformance(ctx context.Context, templateID string, period time.Duration) (*TemplatePerformanceStats, error)
}

// QueueManager defines the interface for email queue management
type QueueManager interface {
	// Queue operations
	Enqueue(ctx context.Context, message *types.EmailMessage, options *EnqueueOptions) error
	EnqueueBatch(ctx context.Context, messages []*types.EmailMessage, options *EnqueueOptions) error
	Dequeue(ctx context.Context, queueName string, maxItems int) ([]*types.EmailQueuedMessage, error)
	Requeue(ctx context.Context, messageID string, delay time.Duration) error
	Remove(ctx context.Context, messageID string) error

	// Queue management
	CreateQueue(ctx context.Context, queue *types.EmailQueue) error
	UpdateQueue(ctx context.Context, queueName string, queue *types.EmailQueue) error
	DeleteQueue(ctx context.Context, queueName string) error
	GetQueue(ctx context.Context, queueName string) (*types.EmailQueue, error)
	ListQueues(ctx context.Context) ([]*types.EmailQueue, error)

	// Queue monitoring
	GetQueueStats(ctx context.Context, queueName string) (*types.EmailQueueStats, error)
	GetQueueDepth(ctx context.Context, queueName string) (int64, error)
	GetQueuedMessages(ctx context.Context, queueName string, filter *QueuedMessageFilter) ([]*types.EmailQueuedMessage, error)

	// Queue processing
	StartProcessing(ctx context.Context, queueName string) error
	StopProcessing(ctx context.Context, queueName string) error
	PauseProcessing(ctx context.Context, queueName string) error
	ResumeProcessing(ctx context.Context, queueName string) error
	IsProcessing(ctx context.Context, queueName string) (bool, error)

	// Bulk operations
	BulkRequeue(ctx context.Context, filter *QueuedMessageFilter, delay time.Duration) (int64, error)
	BulkRemove(ctx context.Context, filter *QueuedMessageFilter) (int64, error)
	PurgeQueue(ctx context.Context, queueName string) (int64, error)

	// Dead letter queue
	GetFailedMessages(ctx context.Context, queueName string, filter *FailedMessageFilter) ([]*types.EmailQueuedMessage, error)
	RetryFailedMessages(ctx context.Context, queueName string, messageIDs []string) error
	PurgeFailedMessages(ctx context.Context, queueName string, olderThan time.Duration) (int64, error)
}

// EmailService defines the main interface for the email service
type EmailService interface {
	// Core email operations
	SendEmail(ctx context.Context, message *types.EmailMessage) (*types.EmailSendResult, error)
	SendEmailAsync(ctx context.Context, message *types.EmailMessage, options *AsyncSendOptions) error
	SendBulkEmail(ctx context.Context, messages []*types.EmailMessage) ([]*types.EmailSendResult, error)
	SendTemplateEmail(ctx context.Context, templateID string, data *types.EmailTemplateData, recipients []string) (*types.EmailSendResult, error)

	// Scheduling
	ScheduleEmail(ctx context.Context, message *types.EmailMessage, sendAt time.Time) error
	CancelScheduledEmail(ctx context.Context, messageID string) error
	GetScheduledEmails(ctx context.Context, filter *ScheduledEmailFilter) ([]*types.EmailQueuedMessage, error)

	// Status and tracking
	GetEmailStatus(ctx context.Context, messageID string) (*types.EmailDeliveryStatus, error)
	GetEmailHistory(ctx context.Context, filter *EmailHistoryFilter) ([]*EmailHistoryEntry, error)
	TrackEmailEvents(ctx context.Context, messageID string) ([]*types.EmailEvent, error)

	// Configuration
	UpdateNotificationConfig(ctx context.Context, config *types.EmailNotificationConfig) error
	GetNotificationConfig(ctx context.Context) (*types.EmailNotificationConfig, error)
	TestEmailConfig(ctx context.Context, testEmail string) error

	// Analytics and reporting
	GetEmailMetrics(ctx context.Context, period time.Duration) (*types.EmailMetrics, error)
	GetProviderMetrics(ctx context.Context, providerName string, period time.Duration) (*EmailProviderMetrics, error)
	GenerateEmailReport(ctx context.Context, options *EmailReportOptions) (*EmailReport, error)

	// Health and monitoring
	HealthCheck(ctx context.Context) (*EmailServiceHealth, error)
	GetSystemStatus(ctx context.Context) (*EmailSystemStatus, error)
}

// EmailNotificationService defines the interface for notification management
type EmailNotificationService interface {
	// Notification management
	RegisterEventHandler(eventType string, handler EmailEventHandler) error
	UnregisterEventHandler(eventType string) error
	TriggerNotification(ctx context.Context, eventType string, data map[string]interface{}) error

	// Notification rules
	CreateNotificationRule(ctx context.Context, rule *NotificationRule) error
	UpdateNotificationRule(ctx context.Context, ruleID string, rule *NotificationRule) error
	DeleteNotificationRule(ctx context.Context, ruleID string) error
	GetNotificationRule(ctx context.Context, ruleID string) (*NotificationRule, error)
	ListNotificationRules(ctx context.Context, filter *NotificationRuleFilter) ([]*NotificationRule, error)

	// Notification history
	GetNotificationHistory(ctx context.Context, filter *NotificationHistoryFilter) ([]*NotificationHistoryEntry, error)
	GetNotificationStats(ctx context.Context, period time.Duration) (*NotificationStats, error)

	// User preferences
	GetUserPreferences(ctx context.Context, userID string) (*UserEmailPreferences, error)
	UpdateUserPreferences(ctx context.Context, userID string, preferences *UserEmailPreferences) error
	UnsubscribeUser(ctx context.Context, userID string, notificationType string) error
}

// EmailEventHandler defines the interface for handling email events
type EmailEventHandler interface {
	HandleEvent(ctx context.Context, event *types.EmailEvent) error
	GetHandlerName() string
	GetSupportedEvents() []string
}

// EmailValidator defines the interface for email validation
type EmailValidator interface {
	// Email validation
	ValidateEmail(email string) (*EmailValidationResult, error)
	ValidateBulkEmails(emails []string) ([]*EmailValidationResult, error)
	ValidateEmailDomain(domain string) (*DomainValidationResult, error)

	// Message validation
	ValidateMessage(message *types.EmailMessage) (*MessageValidationResult, error)
	ValidateTemplate(template *types.EmailTemplate) (*TemplateValidationResult, error)

	// Content validation
	CheckSpamScore(message *types.EmailMessage) (*SpamCheckResult, error)
	ValidateLinks(content string) (*LinkValidationResult, error)
	ValidateImages(content string) (*ImageValidationResult, error)
}

// EmailAnalytics defines the interface for email analytics
type EmailAnalytics interface {
	// Delivery analytics
	GetDeliveryRate(ctx context.Context, period time.Duration) (*DeliveryRateStats, error)
	GetBounceRate(ctx context.Context, period time.Duration) (*BounceRateStats, error)
	GetOpenRate(ctx context.Context, period time.Duration) (*OpenRateStats, error)
	GetClickRate(ctx context.Context, period time.Duration) (*ClickRateStats, error)

	// Campaign analytics
	GetCampaignStats(ctx context.Context, campaignID string) (*CampaignStats, error)
	CompareCampaigns(ctx context.Context, campaignIDs []string) (*CampaignComparison, error)

	// Template analytics
	GetTemplatePerformance(ctx context.Context, templateID string, period time.Duration) (*TemplatePerformanceStats, error)
	GetPopularTemplates(ctx context.Context, period time.Duration, limit int) ([]*TemplateUsageStats, error)

	// Provider analytics
	GetProviderPerformance(ctx context.Context, providerName string, period time.Duration) (*EmailProviderMetrics, error)
	CompareProviders(ctx context.Context, providerNames []string, period time.Duration) (*ProviderComparison, error)

	// User analytics
	GetUserEngagement(ctx context.Context, userID string, period time.Duration) (*UserEngagementStats, error)
	GetActiveUsers(ctx context.Context, period time.Duration) (*ActiveUserStats, error)

	// Trend analysis
	GetTrends(ctx context.Context, metric string, period time.Duration) (*TrendAnalysis, error)
	PredictDeliveryVolume(ctx context.Context, period time.Duration) (*VolumePredicition, error)
}

// Supporting types and structures

// EmailProviderLimits represents limits for an email provider
type EmailProviderLimits struct {
	DailyLimit         int                    `json:"daily_limit"`
	HourlyLimit        int                    `json:"hourly_limit"`
	RateLimit          int                    `json:"rate_limit"`          // per minute
	MaxAttachmentSize  int64                  `json:"max_attachment_size"` // bytes
	MaxMessageSize     int64                  `json:"max_message_size"`    // bytes
	MaxRecipients      int                    `json:"max_recipients"`
	SupportedMimeTypes []string               `json:"supported_mime_types"`
	Features           map[string]bool        `json:"features"`
}

// EmailProviderStats represents statistics for an email provider
type EmailProviderStats struct {
	TotalSent      int64                 `json:"total_sent"`
	TotalFailed    int64                 `json:"total_failed"`
	TotalDelivered int64                 `json:"total_delivered"`
	TotalBounced   int64                 `json:"total_bounced"`
	SuccessRate    float64               `json:"success_rate"`
	AverageLatency time.Duration         `json:"average_latency"`
	StatusCodes    map[int]int64         `json:"status_codes"`
	ErrorTypes     map[string]int64      `json:"error_types"`
	LastUsed       *time.Time            `json:"last_used"`
	Uptime         time.Duration         `json:"uptime"`
}

// EmailTemplateFilter represents filters for template queries
type EmailTemplateFilter struct {
	Category    string     `json:"category,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	CreatedBy   *int       `json:"created_by,omitempty"`
	UpdatedBy   *int       `json:"updated_by,omitempty"`
	FromTime    *time.Time `json:"from_time,omitempty"`
	ToTime      *time.Time `json:"to_time,omitempty"`
	NamePattern string     `json:"name_pattern,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
	SortBy      string     `json:"sort_by,omitempty"`
	SortOrder   string     `json:"sort_order,omitempty"`
}

// EmailPreview represents a preview of rendered email
type EmailPreview struct {
	Subject     string                 `json:"subject"`
	TextBody    string                 `json:"text_body,omitempty"`
	HTMLBody    string                 `json:"html_body,omitempty"`
	Variables   map[string]interface{} `json:"variables"`
	RenderTime  time.Duration          `json:"render_time"`
	Warnings    []string               `json:"warnings,omitempty"`
}

// Additional supporting types would be defined here...
// These are abbreviated for space but would include full definitions

type TemplateValidationResult struct{}
type TemplateTestResult struct{}
type VariableValidationResult struct{}
type TemplateUsageStats struct{}
type TemplatePerformanceStats struct{}
type EnqueueOptions struct{}
type QueuedMessageFilter struct{}
type FailedMessageFilter struct{}
type AsyncSendOptions struct{}
type ScheduledEmailFilter struct{}
type EmailHistoryFilter struct{}
type EmailHistoryEntry struct{}
type EmailReportOptions struct{}
type EmailReport struct{}
type EmailServiceHealth struct{}
type EmailSystemStatus struct{}
type EmailProviderMetrics struct{}
type NotificationRule struct{}
type NotificationRuleFilter struct{}
type NotificationHistoryFilter struct{}
type NotificationHistoryEntry struct{}
type NotificationStats struct{}
type UserEmailPreferences struct{}
type EmailValidationResult struct{}
type DomainValidationResult struct{}
type MessageValidationResult struct{}
type SpamCheckResult struct{}
type LinkValidationResult struct{}
type ImageValidationResult struct{}
type DeliveryRateStats struct{}
type BounceRateStats struct{}
type OpenRateStats struct{}
type ClickRateStats struct{}
type CampaignStats struct{}
type CampaignComparison struct{}
type ProviderComparison struct{}
type UserEngagementStats struct{}
type ActiveUserStats struct{}
type TrendAnalysis struct{}
type VolumePredicition struct{}