package registry

import (
	"context"
	"time"

	"docker-auto/internal/model"
)

// Client defines the interface for registry clients
type Client interface {
	// Basic operations
	TestConnection(ctx context.Context) error
	GetRegistryInfo(ctx context.Context) (*RegistryInfo, error)

	// Image operations
	CheckImageUpdate(ctx context.Context, image, currentDigest string) (*UpdateCheckResult, error)
	GetLatestImageInfo(ctx context.Context, image string) (*model.ImageVersion, error)
	GetImageTags(ctx context.Context, repository string, options *TagListOptions) ([]*ImageTag, error)
	GetImageManifest(ctx context.Context, repository, tag string) (*ImageManifest, error)

	// Search operations
	SearchRepositories(ctx context.Context, options *SearchOptions) ([]*RepositorySearchResult, error)
	GetRepositoryInfo(ctx context.Context, repository string) (*RepositoryInfo, error)

	// Security scanning (optional)
	GetSecurityScanResult(ctx context.Context, repository, tag string) (*ScanResult, error)

	// Cleanup
	Close() error
}

// HarborClient defines extended interface for Harbor registries
type HarborClient interface {
	Client

	// Harbor-specific operations
	GetProjects(ctx context.Context) ([]*Project, error)
	GetRepositories(ctx context.Context, projectName string) ([]*Repository, error)
	GetArtifacts(ctx context.Context, projectName, repoName string) ([]*Artifact, error)
	GetImageScanResult(ctx context.Context, projectName, repoName, reference string) (*ScanResult, error)

	// Project management
	CreateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, projectID int) error
	UpdateProject(ctx context.Context, project *Project) error

	// Repository management
	DeleteRepository(ctx context.Context, projectName, repoName string) error
	GetRepositoryTags(ctx context.Context, projectName, repoName string) ([]*ArtifactTag, error)
}

// DockerHubClient defines interface for Docker Hub specific operations
type DockerHubClient interface {
	Client

	// Docker Hub specific operations
	GetOfficialImages(ctx context.Context) ([]*RepositorySearchResult, error)
	GetUserRepositories(ctx context.Context, username string) ([]*RepositorySearchResult, error)
	GetRepositoryBuildHistory(ctx context.Context, repository string) ([]BuildInfo, error)
}

// BuildInfo represents Docker Hub build information
type BuildInfo struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	BuildCode string `json:"build_code"`
	Tag       string `json:"tag"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}

// GenericClient defines interface for generic registry operations
type GenericClient interface {
	// Basic registry operations
	Ping(ctx context.Context) error
	GetVersion(ctx context.Context) (string, error)
	GetCatalog(ctx context.Context) ([]string, error)

	// Manifest operations
	GetManifest(ctx context.Context, repository, reference string) (*ImageManifest, error)
	PutManifest(ctx context.Context, repository, reference string, manifest *ImageManifest) error
	DeleteManifest(ctx context.Context, repository, reference string) error

	// Blob operations
	GetBlob(ctx context.Context, repository, digest string) ([]byte, error)
	PutBlob(ctx context.Context, repository string, data []byte) (string, error)
	DeleteBlob(ctx context.Context, repository, digest string) error

	// Tag operations
	ListTags(ctx context.Context, repository string) ([]string, error)
	DeleteTag(ctx context.Context, repository, tag string) error
}

// ClientFactory defines interface for creating registry clients
type ClientFactory interface {
	// Create clients for different registry types
	CreateDockerHubClient(auth *AuthConfig) (DockerHubClient, error)
	CreateHarborClient(baseURL string, auth *AuthConfig) (HarborClient, error)
	CreateGenericClient(baseURL string, auth *AuthConfig) (GenericClient, error)

	// Create client from configuration
	CreateClientFromConfig(config *ClientConfig) (Client, error)

	// Get supported registry types
	GetSupportedTypes() []string

	// Detect registry type from URL
	DetectRegistryType(baseURL string) string
}

// ImageChecker defines interface for image checking and comparison
type ImageChecker interface {
	// Check for updates
	CheckImageUpdate(ctx context.Context, image, currentDigest string, registryURL string) (*UpdateCheckResult, error)
	CheckAllImages(ctx context.Context, containers []*model.Container) ([]*UpdateCheckResult, error)

	// Version comparison
	CompareVersions(current, latest *model.ImageVersion) (*VersionComparisonResult, error)
	ShouldUpdate(comparison *VersionComparisonResult, policy string) bool
	GetUpdateStrategy(comparison *VersionComparisonResult) string

	// Registry management
	RegisterClient(registryType string, client Client)
	GetClient(registryURL string) (Client, error)
	GetSupportedRegistries() []string

	// Cache management
	CacheImageInfo(image string, info *model.ImageVersion, ttl time.Duration) error
	GetCachedImageInfo(image string) (*model.ImageVersion, bool)
	InvalidateCache(image string) error
	ClearCache() error

	// Batch operations
	CheckMultipleImages(ctx context.Context, images []string, registryURL string) ([]*UpdateCheckResult, error)
	RefreshAllCache(ctx context.Context) error

	// Configuration
	SetDefaultRegistry(registryURL string)
	GetDefaultRegistry() string
	SetCacheConfig(config *CacheConfig)
}

// CacheConfig represents cache configuration for image checker
type CacheConfig struct {
	TTL                time.Duration `json:"ttl"`
	MaxEntries         int           `json:"max_entries"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	EnablePersistence  bool          `json:"enable_persistence"`
	PersistencePath    string        `json:"persistence_path,omitempty"`
}

// VersionComparisonResult represents the result of version comparison
type VersionComparisonResult struct {
	CurrentVersion string                 `json:"current_version"`
	LatestVersion  string                 `json:"latest_version"`
	CompareResult  int                    `json:"compare_result"` // -1: current < latest, 0: equal, 1: current > latest
	VersionType    string                 `json:"version_type"`   // semantic, date, hash, unknown
	Changes        []VersionChange        `json:"changes,omitempty"`
	SecurityIssues []SecurityVulnerability `json:"security_issues,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Confidence     float64                `json:"confidence,omitempty"` // 0.0-1.0
}

// VersionChange represents a change between versions
type VersionChange struct {
	Type        string    `json:"type"`        // feature, bugfix, security, breaking
	Description string    `json:"description"`
	Impact      string    `json:"impact"`      // low, medium, high, critical
	Date        time.Time `json:"date,omitempty"`
}

// Scheduler defines interface for scheduling image checks
type Scheduler interface {
	// Schedule operations
	ScheduleImageCheck(image string, interval time.Duration) error
	ScheduleContainerCheck(containerID int64, interval time.Duration) error
	UnscheduleImageCheck(image string) error
	UnscheduleContainerCheck(containerID int64) error

	// Bulk scheduling
	ScheduleAllContainers(ctx context.Context, interval time.Duration) error
	UnscheduleAll() error

	// Status and management
	GetScheduledJobs() []ScheduledJob
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// Configuration
	SetDefaultInterval(interval time.Duration)
	GetDefaultInterval() time.Duration
}

// ScheduledJob represents a scheduled image check job
type ScheduledJob struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`         // image, container
	Target       string        `json:"target"`       // image name or container ID
	Interval     time.Duration `json:"interval"`
	NextRun      time.Time     `json:"next_run"`
	LastRun      time.Time     `json:"last_run,omitempty"`
	LastResult   string        `json:"last_result,omitempty"` // success, error
	LastError    string        `json:"last_error,omitempty"`
	Enabled      bool          `json:"enabled"`
	CreatedAt    time.Time     `json:"created_at"`
}

// NotificationManager defines interface for managing update notifications
type NotificationManager interface {
	// Send notifications
	NotifyUpdateAvailable(ctx context.Context, update *UpdateCheckResult) error
	NotifyUpdateCompleted(ctx context.Context, containerID int64, success bool, message string) error
	NotifyUpdateFailed(ctx context.Context, containerID int64, error string) error

	// Batch notifications
	NotifyMultipleUpdates(ctx context.Context, updates []*UpdateCheckResult) error

	// Configuration
	SetNotificationChannels(channels []NotificationChannel) error
	GetNotificationChannels() []NotificationChannel
	EnableChannel(channelType string) error
	DisableChannel(channelType string) error

	// Templates
	SetMessageTemplate(eventType string, template string) error
	GetMessageTemplate(eventType string) string
}

// NotificationChannel represents a notification channel
type NotificationChannel struct {
	Type     string                 `json:"type"`     // email, slack, webhook, telegram
	Enabled  bool                   `json:"enabled"`
	Config   map[string]interface{} `json:"config"`
	Filters  []NotificationFilter   `json:"filters,omitempty"`
}

// NotificationFilter represents a filter for notifications
type NotificationFilter struct {
	Field    string      `json:"field"`    // severity, image, container
	Operator string      `json:"operator"` // equals, contains, starts_with, regex
	Value    interface{} `json:"value"`
}

// Metrics and monitoring interfaces

// MetricsCollector defines interface for collecting registry operation metrics
type MetricsCollector interface {
	// Request metrics
	RecordRequest(registryType, operation string, duration time.Duration, success bool)
	RecordCacheHit(image string, hit bool)
	RecordUpdateCheck(image string, updateAvailable bool)

	// Performance metrics
	GetRequestMetrics() *RequestMetrics
	GetCacheMetrics() *CacheMetrics
	GetUpdateMetrics() *UpdateMetrics

	// Reset metrics
	Reset()
}

// RequestMetrics represents request-related metrics
type RequestMetrics struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	RequestsByType  map[string]int64 `json:"requests_by_type"`
}

// CacheMetrics represents cache-related metrics
type CacheMetrics struct {
	TotalLookups int64   `json:"total_lookups"`
	CacheHits    int64   `json:"cache_hits"`
	CacheMisses  int64   `json:"cache_misses"`
	HitRatio     float64 `json:"hit_ratio"`
	CacheSize    int64   `json:"cache_size"`
}

// UpdateMetrics represents update check metrics
type UpdateMetrics struct {
	TotalChecks        int64            `json:"total_checks"`
	UpdatesAvailable   int64            `json:"updates_available"`
	UpdatesApplied     int64            `json:"updates_applied"`
	ChecksByType       map[string]int64 `json:"checks_by_type"`
	UpdatesByType      map[string]int64 `json:"updates_by_type"`
	LastCheckTimestamp time.Time        `json:"last_check_timestamp"`
}

// HealthChecker defines interface for health checking
type HealthChecker interface {
	// Health check operations
	CheckHealth(ctx context.Context) (*HealthStatus, error)
	CheckRegistryHealth(ctx context.Context, registryURL string) (*RegistryHealthStatus, error)
	CheckAllRegistries(ctx context.Context) ([]*RegistryHealthStatus, error)

	// Monitoring
	StartHealthMonitoring(ctx context.Context, interval time.Duration) error
	StopHealthMonitoring() error
	GetHealthHistory() []*HealthStatus
}

// HealthStatus represents overall health status
type HealthStatus struct {
	Status      string                   `json:"status"` // healthy, degraded, unhealthy
	Timestamp   time.Time                `json:"timestamp"`
	Checks      []*HealthCheck           `json:"checks"`
	Registries  []*RegistryHealthStatus  `json:"registries,omitempty"`
	Summary     *HealthSummary           `json:"summary"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name      string        `json:"name"`
	Status    string        `json:"status"` // pass, warn, fail
	Duration  time.Duration `json:"duration"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// RegistryHealthStatus represents health status of a specific registry
type RegistryHealthStatus struct {
	RegistryURL string        `json:"registry_url"`
	Type        string        `json:"type"`
	Status      string        `json:"status"` // online, offline, degraded
	Latency     time.Duration `json:"latency"`
	Error       string        `json:"error,omitempty"`
	LastCheck   time.Time     `json:"last_check"`
	Features    []string      `json:"features,omitempty"`
}

// HealthSummary represents summary of health status
type HealthSummary struct {
	TotalChecks   int `json:"total_checks"`
	PassingChecks int `json:"passing_checks"`
	WarningChecks int `json:"warning_checks"`
	FailingChecks int `json:"failing_checks"`
	HealthyRegistries int `json:"healthy_registries"`
	TotalRegistries   int `json:"total_registries"`
}

// Checker is an alias for ImageChecker to maintain compatibility
type Checker = ImageChecker