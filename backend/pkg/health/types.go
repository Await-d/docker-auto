package health

import (
	"time"
)

// HealthStatus represents the status of a health check
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthResult represents the result of a health check
type HealthResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

// AggregateHealth represents the overall health of the system
type AggregateHealth struct {
	Status        HealthStatus               `json:"status"`
	Message       string                     `json:"message"`
	Checks        map[string]HealthResult    `json:"checks"`
	Dependencies  map[string]HealthResult    `json:"dependencies"`
	Components    map[string]HealthResult    `json:"components"`
	Timestamp     time.Time                  `json:"timestamp"`
	Duration      time.Duration              `json:"duration"`
	Version       string                     `json:"version,omitempty"`
	Uptime        time.Duration              `json:"uptime,omitempty"`
	Environment   string                     `json:"environment,omitempty"`
}

// HealthConfig represents configuration for health checks
type HealthConfig struct {
	Enabled           bool          `json:"enabled"`
	CheckInterval     time.Duration `json:"check_interval"`
	Timeout           time.Duration `json:"timeout"`
	FailureThreshold  int           `json:"failure_threshold"`
	SuccessThreshold  int           `json:"success_threshold"`
	GracePeriod       time.Duration `json:"grace_period"`
	RetryAttempts     int           `json:"retry_attempts"`
	RetryDelay        time.Duration `json:"retry_delay"`
	EnableRecovery    bool          `json:"enable_recovery"`
	RecoveryActions   []string      `json:"recovery_actions"`
}

// HealthCheckConfig represents configuration for a specific health check
type HealthCheckConfig struct {
	Name             string        `json:"name"`
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	Critical         bool          `json:"critical"`
	Tags             []string      `json:"tags,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DatabaseHealthConfig represents database health check configuration
type DatabaseHealthConfig struct {
	HealthCheckConfig
	ConnectionString string        `json:"connection_string,omitempty"`
	QueryTimeout     time.Duration `json:"query_timeout"`
	TestQuery        string        `json:"test_query"`
	MaxConnections   int           `json:"max_connections"`
	MaxIdleTime      time.Duration `json:"max_idle_time"`
}

// DockerHealthConfig represents Docker health check configuration
type DockerHealthConfig struct {
	HealthCheckConfig
	DockerHost     string        `json:"docker_host,omitempty"`
	APIVersion     string        `json:"api_version,omitempty"`
	RequestTimeout time.Duration `json:"request_timeout"`
	CheckContainers bool         `json:"check_containers"`
	CheckImages     bool         `json:"check_images"`
	CheckNetworks   bool         `json:"check_networks"`
}

// RegistryHealthConfig represents container registry health check configuration
type RegistryHealthConfig struct {
	HealthCheckConfig
	URL           string            `json:"url"`
	Username      string            `json:"username,omitempty"`
	Password      string            `json:"password,omitempty"`
	Token         string            `json:"token,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	TestEndpoint  string            `json:"test_endpoint"`
}

// HTTPHealthConfig represents HTTP endpoint health check configuration
type HTTPHealthConfig struct {
	HealthCheckConfig
	URL               string            `json:"url"`
	Method            string            `json:"method"`
	Headers           map[string]string `json:"headers,omitempty"`
	Body              string            `json:"body,omitempty"`
	ExpectedStatus    []int             `json:"expected_status"`
	ExpectedBody      string            `json:"expected_body,omitempty"`
	TLSSkipVerify     bool              `json:"tls_skip_verify"`
	FollowRedirects   bool              `json:"follow_redirects"`
}

// FileSystemHealthConfig represents filesystem health check configuration
type FileSystemHealthConfig struct {
	HealthCheckConfig
	Path            string  `json:"path"`
	MinFreeSpace    uint64  `json:"min_free_space"`    // bytes
	MinFreePercent  float64 `json:"min_free_percent"`  // percentage
	CheckWritable   bool    `json:"check_writable"`
	CheckReadable   bool    `json:"check_readable"`
	TestFilePath    string  `json:"test_file_path,omitempty"`
}

// MemoryHealthConfig represents memory health check configuration
type MemoryHealthConfig struct {
	HealthCheckConfig
	MaxMemoryPercent float64 `json:"max_memory_percent"`
	MaxMemoryBytes   uint64  `json:"max_memory_bytes"`
	CheckSwap        bool    `json:"check_swap"`
	MaxSwapPercent   float64 `json:"max_swap_percent"`
}

// CPUHealthConfig represents CPU health check configuration
type CPUHealthConfig struct {
	HealthCheckConfig
	MaxCPUPercent    float64 `json:"max_cpu_percent"`
	MaxLoadAverage   float64 `json:"max_load_average"`
	SampleDuration   time.Duration `json:"sample_duration"`
}

// ServiceHealthConfig represents service health check configuration
type ServiceHealthConfig struct {
	HealthCheckConfig
	ServiceName     string        `json:"service_name"`
	Port            int           `json:"port,omitempty"`
	ProcessName     string        `json:"process_name,omitempty"`
	CheckPorts      []int         `json:"check_ports,omitempty"`
	CheckProcesses  []string      `json:"check_processes,omitempty"`
	PingTimeout     time.Duration `json:"ping_timeout"`
}

// HealthCheckHistory represents historical health check data
type HealthCheckHistory struct {
	CheckName string              `json:"check_name"`
	Results   []HealthHistoryItem `json:"results"`
	Summary   HealthSummary       `json:"summary"`
}

// HealthHistoryItem represents a single historical health check result
type HealthHistoryItem struct {
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Error     string        `json:"error,omitempty"`
}

// HealthSummary represents a summary of health check history
type HealthSummary struct {
	TotalChecks     int           `json:"total_checks"`
	SuccessfulChecks int          `json:"successful_checks"`
	FailedChecks    int           `json:"failed_checks"`
	SuccessRate     float64       `json:"success_rate"`
	AverageDuration time.Duration `json:"average_duration"`
	LastSuccess     *time.Time    `json:"last_success,omitempty"`
	LastFailure     *time.Time    `json:"last_failure,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	ConsecutiveSuccesses int      `json:"consecutive_successes"`
	UpTime          time.Duration `json:"uptime"`
	DownTime        time.Duration `json:"downtime"`
}

// HealthTrend represents trending health data
type HealthTrend struct {
	Period     time.Duration     `json:"period"`
	DataPoints []HealthDataPoint `json:"data_points"`
	Trend      string            `json:"trend"` // improving, degrading, stable
	Analysis   string            `json:"analysis"`
}

// HealthDataPoint represents a single data point in health trends
type HealthDataPoint struct {
	Timestamp   time.Time     `json:"timestamp"`
	Status      HealthStatus  `json:"status"`
	Duration    time.Duration `json:"duration"`
	Value       float64       `json:"value,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RecoveryAction represents an automated recovery action
type RecoveryAction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Condition   string                 `json:"condition"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Cooldown    time.Duration          `json:"cooldown"`
	MaxRetries  int                    `json:"max_retries"`
	Enabled     bool                   `json:"enabled"`
}

// HealthAlert represents a health-based alert
type HealthAlert struct {
	ID          string                 `json:"id"`
	CheckName   string                 `json:"check_name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// HealthMetrics represents health check metrics
type HealthMetrics struct {
	CheckName          string        `json:"check_name"`
	TotalChecks        uint64        `json:"total_checks"`
	SuccessfulChecks   uint64        `json:"successful_checks"`
	FailedChecks       uint64        `json:"failed_checks"`
	SuccessRate        float64       `json:"success_rate"`
	AverageDuration    time.Duration `json:"average_duration"`
	MinDuration        time.Duration `json:"min_duration"`
	MaxDuration        time.Duration `json:"max_duration"`
	CurrentStatus      HealthStatus  `json:"current_status"`
	LastChecked        time.Time     `json:"last_checked"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	ConsecutiveSuccesses int         `json:"consecutive_successes"`
}