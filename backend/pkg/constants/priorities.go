package constants

// Priority level constants for tasks
const (
	// PriorityLow represents low priority
	PriorityLow = 1

	// PriorityNormal represents normal priority
	PriorityNormal = 5

	// PriorityHigh represents high priority
	PriorityHigh = 10

	// PriorityUrgent represents urgent priority
	PriorityUrgent = 15

	// PriorityCritical represents critical priority
	PriorityCritical = 20
)

// Priority name constants for string representation
const (
	// PriorityNameLow represents low priority name
	PriorityNameLow = "low"

	// PriorityNameNormal represents normal priority name
	PriorityNameNormal = "normal"

	// PriorityNameHigh represents high priority name
	PriorityNameHigh = "high"

	// PriorityNameUrgent represents urgent priority name
	PriorityNameUrgent = "urgent"

	// PriorityNameCritical represents critical priority name
	PriorityNameCritical = "critical"
)

// Email priority level constants
const (
	// EmailPriorityLow represents low priority email
	EmailPriorityLow = 0

	// EmailPriorityNormal represents normal priority email
	EmailPriorityNormal = 1

	// EmailPriorityHigh represents high priority email
	EmailPriorityHigh = 2

	// EmailPriorityUrgent represents urgent priority email
	EmailPriorityUrgent = 3
)

// Webhook priority level constants
const (
	// WebhookPriorityLow represents low priority webhook
	WebhookPriorityLow = 0

	// WebhookPriorityNormal represents normal priority webhook
	WebhookPriorityNormal = 1

	// WebhookPriorityHigh represents high priority webhook
	WebhookPriorityHigh = 2

	// WebhookPriorityUrgent represents urgent priority webhook
	WebhookPriorityUrgent = 3
)

// Alert severity level constants
const (
	// SeverityInfo represents informational alerts
	SeverityInfo = "info"

	// SeverityWarning represents warning alerts
	SeverityWarning = "warning"

	// SeverityError represents error alerts
	SeverityError = "error"

	// SeverityCritical represents critical alerts
	SeverityCritical = "critical"

	// SeverityFatal represents fatal alerts
	SeverityFatal = "fatal"
)

// Log level constants
const (
	// LogLevelTrace represents trace log level
	LogLevelTrace = "trace"

	// LogLevelDebug represents debug log level
	LogLevelDebug = "debug"

	// LogLevelInfo represents info log level
	LogLevelInfo = "info"

	// LogLevelWarn represents warn log level
	LogLevelWarn = "warn"

	// LogLevelError represents error log level
	LogLevelError = "error"

	// LogLevelFatal represents fatal log level
	LogLevelFatal = "fatal"

	// LogLevelPanic represents panic log level
	LogLevelPanic = "panic"
)

// Priority queue constants
const (
	// DefaultQueueCapacity is the default capacity for priority queues
	DefaultQueueCapacity = 1000

	// MaxQueueCapacity is the maximum capacity for priority queues
	MaxQueueCapacity = 10000

	// MinQueueCapacity is the minimum capacity for priority queues
	MinQueueCapacity = 10
)

// Priority threshold constants for different operations
const (
	// ThresholdImmediateProcessing represents threshold for immediate processing
	ThresholdImmediateProcessing = PriorityUrgent

	// ThresholdFastTrack represents threshold for fast track processing
	ThresholdFastTrack = PriorityHigh

	// ThresholdStandardProcessing represents threshold for standard processing
	ThresholdStandardProcessing = PriorityNormal

	// ThresholdBatchProcessing represents threshold for batch processing
	ThresholdBatchProcessing = PriorityLow
)

// Rate limiting constants based on priority
const (
	// RateLimitLowPriority represents rate limit for low priority items (per minute)
	RateLimitLowPriority = 10

	// RateLimitNormalPriority represents rate limit for normal priority items (per minute)
	RateLimitNormalPriority = 30

	// RateLimitHighPriority represents rate limit for high priority items (per minute)
	RateLimitHighPriority = 60

	// RateLimitUrgentPriority represents rate limit for urgent priority items (per minute)
	RateLimitUrgentPriority = 120

	// RateLimitCriticalPriority represents rate limit for critical priority items (per minute)
	RateLimitCriticalPriority = 300
)

// Timeout constants based on priority
const (
	// TimeoutLowPriority represents timeout for low priority operations
	TimeoutLowPriority = "5m"

	// TimeoutNormalPriority represents timeout for normal priority operations
	TimeoutNormalPriority = "3m"

	// TimeoutHighPriority represents timeout for high priority operations
	TimeoutHighPriority = "2m"

	// TimeoutUrgentPriority represents timeout for urgent priority operations
	TimeoutUrgentPriority = "1m"

	// TimeoutCriticalPriority represents timeout for critical priority operations
	TimeoutCriticalPriority = "30s"
)

// Retry constants based on priority
const (
	// RetriesLowPriority represents retry count for low priority operations
	RetriesLowPriority = 1

	// RetriesNormalPriority represents retry count for normal priority operations
	RetriesNormalPriority = 2

	// RetriesHighPriority represents retry count for high priority operations
	RetriesHighPriority = 3

	// RetriesUrgentPriority represents retry count for urgent priority operations
	RetriesUrgentPriority = 5

	// RetriesCriticalPriority represents retry count for critical priority operations
	RetriesCriticalPriority = 10
)

// Delay constants based on priority for retry operations
const (
	// RetryDelayLowPriority represents retry delay for low priority operations
	RetryDelayLowPriority = "5m"

	// RetryDelayNormalPriority represents retry delay for normal priority operations
	RetryDelayNormalPriority = "2m"

	// RetryDelayHighPriority represents retry delay for high priority operations
	RetryDelayHighPriority = "1m"

	// RetryDelayUrgentPriority represents retry delay for urgent priority operations
	RetryDelayUrgentPriority = "30s"

	// RetryDelayCriticalPriority represents retry delay for critical priority operations
	RetryDelayCriticalPriority = "10s"
)

// Notification priority constants
const (
	// NotificationPriorityLow for low priority notifications
	NotificationPriorityLow = "low"

	// NotificationPriorityNormal for normal priority notifications
	NotificationPriorityNormal = "normal"

	// NotificationPriorityHigh for high priority notifications
	NotificationPriorityHigh = "high"

	// NotificationPriorityUrgent for urgent priority notifications
	NotificationPriorityUrgent = "urgent"

	// NotificationPriorityCritical for critical priority notifications
	NotificationPriorityCritical = "critical"
)

// Priority escalation constants
const (
	// EscalationTimeThreshold is the time after which priority is escalated
	EscalationTimeThreshold = "1h"

	// EscalationFailureThreshold is the failure count after which priority is escalated
	EscalationFailureThreshold = 3

	// MaxEscalationLevel is the maximum priority level for escalation
	MaxEscalationLevel = PriorityCritical

	// EscalationIncrement is the increment value for priority escalation
	EscalationIncrement = 5
)

// Priority aging constants for dynamic priority adjustment
const (
	// AgingIntervalMinutes represents how often priority aging is applied
	AgingIntervalMinutes = 15

	// AgingIncrementPerInterval represents priority increment per aging interval
	AgingIncrementPerInterval = 1

	// MaxAgingIncrement represents maximum priority increase through aging
	MaxAgingIncrement = 10

	// AgingThresholdMinutes represents minimum age for priority aging
	AgingThresholdMinutes = 30
)