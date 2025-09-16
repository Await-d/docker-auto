package logging

import (
	"time"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry with structured fields
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Duration  *time.Duration         `json:"duration,omitempty"`
	Error     *string                `json:"error,omitempty"`
	Stack     *string                `json:"stack,omitempty"`
}

// SecurityEvent represents a security-related log entry
type SecurityEvent struct {
	LogEntry
	EventType    string `json:"event_type"`
	Severity     string `json:"severity"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	Outcome      string `json:"outcome"`
	ThreatLevel  string `json:"threat_level,omitempty"`
}

// PerformanceEvent represents a performance monitoring log entry
type PerformanceEvent struct {
	LogEntry
	Operation    string        `json:"operation"`
	Duration     time.Duration `json:"duration"`
	ResponseSize int64         `json:"response_size,omitempty"`
	QueryCount   int           `json:"query_count,omitempty"`
	CacheHit     *bool         `json:"cache_hit,omitempty"`
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	LogEntry
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Changes    map[string]interface{} `json:"changes,omitempty"`
	Outcome    string                 `json:"outcome"`
}

// LogConfig represents configuration for the logger
type LogConfig struct {
	Level       LogLevel `json:"level"`
	Format      string   `json:"format"` // json or text
	Output      string   `json:"output"` // stdout, stderr, or file path
	Rotation    bool     `json:"rotation"`
	MaxSize     int      `json:"max_size"`     // MB
	MaxBackups  int      `json:"max_backups"`
	MaxAge      int      `json:"max_age"`      // days
	Compress    bool     `json:"compress"`
	BufferSize  int      `json:"buffer_size"`
}

// LoggingMiddlewareConfig represents configuration for HTTP logging middleware
type LoggingMiddlewareConfig struct {
	Enabled          bool     `json:"enabled"`
	LogRequestBody   bool     `json:"log_request_body"`
	LogResponseBody  bool     `json:"log_response_body"`
	MaxBodySize      int      `json:"max_body_size"`
	SkipPaths        []string `json:"skip_paths"`
	SensitiveHeaders []string `json:"sensitive_headers"`
}