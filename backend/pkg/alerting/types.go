package alerting

import (
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityFatal    AlertSeverity = "fatal"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
)

// Alert represents a single alert
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Source      string                 `json:"source"`
	Component   string                 `json:"component"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	Value       float64                `json:"value,omitempty"`
	Threshold   float64                `json:"threshold,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	AckedAt     *time.Time             `json:"acked_at,omitempty"`
	AckedBy     string                 `json:"acked_by,omitempty"`
	Count       int                    `json:"count"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
}

// AlertRule represents a rule for triggering alerts
type AlertRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Enabled          bool                   `json:"enabled"`
	Expression       string                 `json:"expression"`
	Condition        AlertCondition         `json:"condition"`
	Threshold        float64                `json:"threshold"`
	Severity         AlertSeverity          `json:"severity"`
	Duration         time.Duration          `json:"duration"`
	EvaluationInterval time.Duration        `json:"evaluation_interval"`
	Labels           map[string]string      `json:"labels"`
	Annotations      map[string]interface{} `json:"annotations"`
	Channels         []string               `json:"channels"`
	Cooldown         time.Duration          `json:"cooldown"`
	MaxAlerts        int                    `json:"max_alerts"`
	GroupBy          []string               `json:"group_by"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// AlertCondition represents the condition for triggering an alert
type AlertCondition string

const (
	AlertConditionGreaterThan    AlertCondition = ">"
	AlertConditionGreaterOrEqual AlertCondition = ">="
	AlertConditionLessThan       AlertCondition = "<"
	AlertConditionLessOrEqual    AlertCondition = "<="
	AlertConditionEqual          AlertCondition = "=="
	AlertConditionNotEqual       AlertCondition = "!="
	AlertConditionContains       AlertCondition = "contains"
	AlertConditionNotContains    AlertCondition = "not_contains"
	AlertConditionRegex          AlertCondition = "regex"
	AlertConditionNotRegex       AlertCondition = "not_regex"
)

// AlertChannel represents a notification channel
type AlertChannel interface {
	Name() string
	Type() string
	Send(alert Alert) error
	Config() AlertChannelConfig
	Test() error
}

// AlertChannelConfig represents configuration for an alert channel
type AlertChannelConfig struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Settings    map[string]interface{} `json:"settings"`
	Filters     []AlertFilter          `json:"filters"`
	Template    string                 `json:"template,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	RetryCount  int                    `json:"retry_count"`
	RetryDelay  time.Duration          `json:"retry_delay"`
	RateLimit   RateLimit              `json:"rate_limit"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertFilter represents a filter for alert channels
type AlertFilter struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	Negate    bool        `json:"negate"`
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
	Count    int           `json:"count"`
	Burst    int           `json:"burst"`
}

// EmailChannelConfig represents email channel configuration
type EmailChannelConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	From         string   `json:"from"`
	To           []string `json:"to"`
	CC           []string `json:"cc,omitempty"`
	BCC          []string `json:"bcc,omitempty"`
	Subject      string   `json:"subject"`
	Template     string   `json:"template"`
	TLS          bool     `json:"tls"`
	InsecureSkip bool     `json:"insecure_skip"`
}

// SlackChannelConfig represents Slack channel configuration
type SlackChannelConfig struct {
	WebhookURL  string            `json:"webhook_url"`
	Channel     string            `json:"channel"`
	Username    string            `json:"username"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Template    string            `json:"template"`
	Mentions    map[string]string `json:"mentions,omitempty"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
}

// WebhookChannelConfig represents webhook channel configuration
type WebhookChannelConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	Template    string            `json:"template"`
	ContentType string            `json:"content_type"`
	Secret      string            `json:"secret,omitempty"`
	SignHeader  string            `json:"sign_header,omitempty"`
}

// DiscordChannelConfig represents Discord channel configuration
type DiscordChannelConfig struct {
	WebhookURL string            `json:"webhook_url"`
	Username   string            `json:"username"`
	AvatarURL  string            `json:"avatar_url,omitempty"`
	Template   string            `json:"template"`
	Mentions   map[string]string `json:"mentions,omitempty"`
	Embeds     bool              `json:"embeds"`
}

// SMSChannelConfig represents SMS channel configuration
type SMSChannelConfig struct {
	Provider    string   `json:"provider"`
	APIKey      string   `json:"api_key"`
	APISecret   string   `json:"api_secret"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	Template    string   `json:"template"`
	MaxLength   int      `json:"max_length"`
}

// AlertGroup represents a group of related alerts
type AlertGroup struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Alerts    []Alert           `json:"alerts"`
	Status    AlertStatus       `json:"status"`
	Severity  AlertSeverity     `json:"severity"`
	Count     int               `json:"count"`
	FirstSeen time.Time         `json:"first_seen"`
	LastSeen  time.Time         `json:"last_seen"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EscalationPolicy represents an escalation policy for alerts
type EscalationPolicy struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Enabled      bool                 `json:"enabled"`
	Rules        []EscalationRule     `json:"rules"`
	DefaultRule  *EscalationRule      `json:"default_rule,omitempty"`
	Filters      []AlertFilter        `json:"filters"`
	OnCallGroups []string             `json:"on_call_groups,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// EscalationRule represents a single escalation rule
type EscalationRule struct {
	Level        int           `json:"level"`
	Delay        time.Duration `json:"delay"`
	Channels     []string      `json:"channels"`
	Conditions   []AlertFilter `json:"conditions,omitempty"`
	RepeatCount  int           `json:"repeat_count"`
	RepeatDelay  time.Duration `json:"repeat_delay"`
	StopOnAck    bool          `json:"stop_on_ack"`
	StopOnResolve bool         `json:"stop_on_resolve"`
}

// SuppressionRule represents a rule for suppressing alerts
type SuppressionRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
	Matchers    []AlertFilter          `json:"matchers"`
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	Reason      string                 `json:"reason"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// AlertingConfig represents the main alerting configuration
type AlertingConfig struct {
	Enabled              bool                          `json:"enabled"`
	EvaluationInterval   time.Duration                 `json:"evaluation_interval"`
	AlertTimeout         time.Duration                 `json:"alert_timeout"`
	ResolveTimeout       time.Duration                 `json:"resolve_timeout"`
	GroupWait            time.Duration                 `json:"group_wait"`
	GroupInterval        time.Duration                 `json:"group_interval"`
	RepeatInterval       time.Duration                 `json:"repeat_interval"`
	MaxAlerts            int                           `json:"max_alerts"`
	Storage              AlertStorageConfig            `json:"storage"`
	Routes               []AlertRoute                  `json:"routes"`
	Receivers            []AlertReceiver               `json:"receivers"`
	InhibitRules         []InhibitRule                 `json:"inhibit_rules"`
	GlobalConfig         map[string]interface{}        `json:"global_config"`
}

// AlertStorageConfig represents storage configuration for alerts
type AlertStorageConfig struct {
	Type       string        `json:"type"` // memory, file, database
	Path       string        `json:"path,omitempty"`
	Retention  time.Duration `json:"retention"`
	MaxSize    int64         `json:"max_size,omitempty"`
	Compress   bool          `json:"compress"`
	Backup     bool          `json:"backup"`
}

// AlertRoute represents routing configuration for alerts
type AlertRoute struct {
	Match         map[string]string `json:"match,omitempty"`
	MatchRE       map[string]string `json:"match_re,omitempty"`
	Receiver      string            `json:"receiver"`
	GroupBy       []string          `json:"group_by,omitempty"`
	GroupWait     *time.Duration    `json:"group_wait,omitempty"`
	GroupInterval *time.Duration    `json:"group_interval,omitempty"`
	RepeatInterval *time.Duration   `json:"repeat_interval,omitempty"`
	Routes        []AlertRoute      `json:"routes,omitempty"`
	Continue      bool              `json:"continue"`
}

// AlertReceiver represents a receiver configuration
type AlertReceiver struct {
	Name            string                   `json:"name"`
	EmailConfigs    []EmailChannelConfig     `json:"email_configs,omitempty"`
	SlackConfigs    []SlackChannelConfig     `json:"slack_configs,omitempty"`
	WebhookConfigs  []WebhookChannelConfig   `json:"webhook_configs,omitempty"`
	DiscordConfigs  []DiscordChannelConfig   `json:"discord_configs,omitempty"`
	SMSConfigs      []SMSChannelConfig       `json:"sms_configs,omitempty"`
}

// InhibitRule represents a rule for inhibiting alerts
type InhibitRule struct {
	SourceMatch   map[string]string `json:"source_match,omitempty"`
	SourceMatchRE map[string]string `json:"source_match_re,omitempty"`
	TargetMatch   map[string]string `json:"target_match,omitempty"`
	TargetMatchRE map[string]string `json:"target_match_re,omitempty"`
	Equal         []string          `json:"equal,omitempty"`
}

// AlertMetrics represents metrics for the alerting system
type AlertMetrics struct {
	TotalAlerts        uint64            `json:"total_alerts"`
	ActiveAlerts       uint64            `json:"active_alerts"`
	ResolvedAlerts     uint64            `json:"resolved_alerts"`
	AlertsByStatus     map[string]uint64 `json:"alerts_by_status"`
	AlertsBySeverity   map[string]uint64 `json:"alerts_by_severity"`
	AlertsByComponent  map[string]uint64 `json:"alerts_by_component"`
	NotificationsSent  uint64            `json:"notifications_sent"`
	NotificationsFailed uint64           `json:"notifications_failed"`
	EvaluationTime     time.Duration     `json:"evaluation_time"`
	LastEvaluation     time.Time         `json:"last_evaluation"`
}

// AlertStats represents statistics for alerts
type AlertStats struct {
	Rule         string        `json:"rule"`
	Component    string        `json:"component"`
	TotalCount   int           `json:"total_count"`
	ActiveCount  int           `json:"active_count"`
	ResolvedCount int          `json:"resolved_count"`
	MTTR         time.Duration `json:"mttr"` // Mean Time To Resolution
	MTTA         time.Duration `json:"mtta"` // Mean Time To Acknowledgment
	Frequency    float64       `json:"frequency"` // alerts per hour
	LastOccurrence time.Time   `json:"last_occurrence"`
}

// AlertNotification represents a notification to be sent
type AlertNotification struct {
	ID        string                 `json:"id"`
	AlertID   string                 `json:"alert_id"`
	Channel   string                 `json:"channel"`
	Recipient string                 `json:"recipient"`
	Subject   string                 `json:"subject"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	SentAt    *time.Time             `json:"sent_at,omitempty"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	RetryCount int                   `json:"retry_count"`
}