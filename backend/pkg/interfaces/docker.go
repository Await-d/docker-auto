// Package interfaces provides unified interface definitions for the docker-auto system.
// This package centralizes all interface definitions to ensure consistency and enable dependency injection.
package interfaces

import (
	"context"
	"io"
	"time"

	"docker-auto/pkg/types"
)

// DockerClient defines the interface for Docker client operations
type DockerClient interface {
	// Container operations
	ListContainers(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	CreateContainer(ctx context.Context, config *types.ContainerCreateConfig) (*types.Container, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error
	RestartContainer(ctx context.Context, containerID string, timeout *time.Duration) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	PauseContainer(ctx context.Context, containerID string) error
	UnpauseContainer(ctx context.Context, containerID string) error
	KillContainer(ctx context.Context, containerID string, signal string) error
	GetContainer(ctx context.Context, containerID string) (*types.Container, error)
	GetContainerStatus(ctx context.Context, containerID string) (*types.ContainerStatus, error)
	GetContainerLogs(ctx context.Context, containerID string, options LogOptions) (io.ReadCloser, error)
	GetContainerStats(ctx context.Context, containerID string, stream bool) (*types.ContainerMetrics, error)
	UpdateContainer(ctx context.Context, containerID string, config *types.ContainerUpdateConfig) error
	ExecInContainer(ctx context.Context, containerID string, cmd []string, options ExecOptions) (*ExecResult, error)

	// Image operations
	ListImages(ctx context.Context, options types.ImageListOptions) ([]types.Image, error)
	PullImage(ctx context.Context, imageName string, options PullOptions) error
	RemoveImage(ctx context.Context, imageID string, options types.ImageRemoveOptions) error
	GetImage(ctx context.Context, imageID string) (*types.Image, error)
	BuildImage(ctx context.Context, buildContext io.Reader, options BuildOptions) error
	TagImage(ctx context.Context, sourceImage, targetImage string) error
	InspectImage(ctx context.Context, imageID string) (*ImageInspect, error)

	// Network operations
	ListNetworks(ctx context.Context, options NetworkListOptions) ([]Network, error)
	CreateNetwork(ctx context.Context, name string, options NetworkCreateOptions) (*Network, error)
	RemoveNetwork(ctx context.Context, networkID string) error
	ConnectContainerToNetwork(ctx context.Context, networkID, containerID string, config *NetworkConfig) error
	DisconnectContainerFromNetwork(ctx context.Context, networkID, containerID string, force bool) error

	// Volume operations
	ListVolumes(ctx context.Context, options VolumeListOptions) ([]Volume, error)
	CreateVolume(ctx context.Context, name string, options VolumeCreateOptions) (*Volume, error)
	RemoveVolume(ctx context.Context, volumeName string, force bool) error
	InspectVolume(ctx context.Context, volumeName string) (*Volume, error)

	// System operations
	GetSystemInfo(ctx context.Context) (*SystemInfo, error)
	GetVersion(ctx context.Context) (*Version, error)
	Ping(ctx context.Context) error
	GetEvents(ctx context.Context, options EventOptions) (<-chan Event, <-chan error)
	GetDiskUsage(ctx context.Context) (*DiskUsage, error)

	// Cleanup operations
	PruneContainers(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	PruneImages(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	PruneNetworks(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	PruneVolumes(ctx context.Context, filters map[string][]string) (*PruneResult, error)

	// Configuration and lifecycle
	ValidateConfig() error
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	GetConfig() *types.ClientConfig
}

// ImageManager defines the interface for Docker image management
type ImageManager interface {
	// Image lifecycle
	Pull(ctx context.Context, imageName string, options PullOptions) error
	Push(ctx context.Context, imageName string, options PushOptions) error
	Build(ctx context.Context, buildContext io.Reader, options BuildOptions) (*BuildResult, error)
	Tag(ctx context.Context, sourceImage, targetImage string) error
	Remove(ctx context.Context, imageID string, force bool) error

	// Image queries
	List(ctx context.Context, filters map[string][]string) ([]types.Image, error)
	Get(ctx context.Context, imageID string) (*types.Image, error)
	Inspect(ctx context.Context, imageID string) (*ImageInspect, error)
	History(ctx context.Context, imageID string) ([]ImageHistoryEntry, error)
	Search(ctx context.Context, term string, limit int) ([]ImageSearchResult, error)

	// Image operations
	Prune(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	Export(ctx context.Context, imageID string) (io.ReadCloser, error)
	Import(ctx context.Context, source io.Reader, ref string) error
	Save(ctx context.Context, imageNames []string) (io.ReadCloser, error)
	Load(ctx context.Context, input io.Reader) error

	// Registry operations
	Login(ctx context.Context, registry, username, password string) error
	Logout(ctx context.Context, registry string) error

	// Image metadata
	GetLayers(ctx context.Context, imageID string) ([]ImageLayer, error)
	GetManifest(ctx context.Context, imageID string) (*ImageManifest, error)
	GetSize(ctx context.Context, imageID string) (int64, error)
	GetDigest(ctx context.Context, imageID string) (string, error)

	// Cleanup and maintenance
	CleanupDangling(ctx context.Context) (*PruneResult, error)
	CleanupOld(ctx context.Context, olderThan time.Duration) (*PruneResult, error)
	OptimizeStorage(ctx context.Context) error
}

// ContainerManager defines the interface for Docker container management
type ContainerManager interface {
	// Container lifecycle
	Create(ctx context.Context, config *types.ContainerCreateConfig) (*types.Container, error)
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string, timeout *time.Duration) error
	Restart(ctx context.Context, containerID string, timeout *time.Duration) error
	Remove(ctx context.Context, containerID string, force bool) error
	Pause(ctx context.Context, containerID string) error
	Unpause(ctx context.Context, containerID string) error
	Kill(ctx context.Context, containerID string, signal string) error

	// Container queries
	List(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error)
	Get(ctx context.Context, containerID string) (*types.Container, error)
	GetStatus(ctx context.Context, containerID string) (*types.ContainerStatus, error)
	GetProcesses(ctx context.Context, containerID string) ([]ContainerProcess, error)
	GetChanges(ctx context.Context, containerID string) ([]ContainerChange, error)

	// Container operations
	Update(ctx context.Context, containerID string, config *types.ContainerUpdateConfig) error
	Rename(ctx context.Context, containerID, newName string) error
	Resize(ctx context.Context, containerID string, height, width uint) error
	Export(ctx context.Context, containerID string) (io.ReadCloser, error)
	Commit(ctx context.Context, containerID string, options CommitOptions) (*types.Image, error)

	// Container execution
	Exec(ctx context.Context, containerID string, cmd []string, options ExecOptions) (*ExecResult, error)
	AttachToContainer(ctx context.Context, containerID string, options AttachOptions) (*AttachResult, error)

	// Container monitoring
	GetLogs(ctx context.Context, containerID string, options LogOptions) (io.ReadCloser, error)
	GetStats(ctx context.Context, containerID string, stream bool) (*types.ContainerMetrics, error)
	StreamStats(ctx context.Context, containerID string) (<-chan *types.ContainerMetrics, error)

	// Bulk operations
	BulkStart(ctx context.Context, containerIDs []string, config types.BulkOperationConfig) ([]types.ParallelOperationResult, error)
	BulkStop(ctx context.Context, containerIDs []string, config types.BulkOperationConfig) ([]types.ParallelOperationResult, error)
	BulkRemove(ctx context.Context, containerIDs []string, config types.BulkOperationConfig) ([]types.ParallelOperationResult, error)
	BulkUpdate(ctx context.Context, updates map[string]*types.ContainerUpdateConfig, config types.BulkOperationConfig) ([]types.ParallelOperationResult, error)

	// Cleanup operations
	Prune(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	CleanupExited(ctx context.Context) (*PruneResult, error)
	CleanupUnused(ctx context.Context, unusedFor time.Duration) (*PruneResult, error)
}

// NetworkManager defines the interface for Docker network management
type NetworkManager interface {
	// Network lifecycle
	Create(ctx context.Context, name string, options NetworkCreateOptions) (*Network, error)
	Remove(ctx context.Context, networkID string) error
	Connect(ctx context.Context, networkID, containerID string, config *NetworkConfig) error
	Disconnect(ctx context.Context, networkID, containerID string, force bool) error

	// Network queries
	List(ctx context.Context, options NetworkListOptions) ([]Network, error)
	Get(ctx context.Context, networkID string) (*Network, error)
	Inspect(ctx context.Context, networkID string) (*NetworkInspect, error)

	// Network operations
	Prune(ctx context.Context, filters map[string][]string) (*PruneResult, error)
}

// VolumeManager defines the interface for Docker volume management
type VolumeManager interface {
	// Volume lifecycle
	Create(ctx context.Context, name string, options VolumeCreateOptions) (*Volume, error)
	Remove(ctx context.Context, volumeName string, force bool) error

	// Volume queries
	List(ctx context.Context, options VolumeListOptions) ([]Volume, error)
	Get(ctx context.Context, volumeName string) (*Volume, error)
	Inspect(ctx context.Context, volumeName string) (*Volume, error)

	// Volume operations
	Prune(ctx context.Context, filters map[string][]string) (*PruneResult, error)
	CleanupOrphaned(ctx context.Context) (*PruneResult, error)
}

// EventMonitor defines the interface for Docker event monitoring
type EventMonitor interface {
	// Event streaming
	StartMonitoring(ctx context.Context, options EventOptions) (<-chan Event, <-chan error)
	StopMonitoring() error

	// Event filtering and processing
	AddEventHandler(eventType string, handler EventHandler) error
	RemoveEventHandler(eventType string) error
	GetEventHistory(ctx context.Context, since time.Time, until time.Time) ([]Event, error)

	// Event statistics
	GetEventStats(ctx context.Context) (*EventStats, error)
}

// HealthChecker defines the interface for container health checking
type HealthChecker interface {
	// Health checks
	CheckHealth(ctx context.Context, containerID string) (*HealthCheckResult, error)
	GetHealthStatus(ctx context.Context, containerID string) (*types.ContainerHealthStatus, error)
	GetHealthHistory(ctx context.Context, containerID string, limit int) ([]types.HealthLogEntry, error)

	// Health monitoring
	StartHealthMonitoring(ctx context.Context, containerID string, interval time.Duration) (<-chan *HealthCheckResult, error)
	StopHealthMonitoring(containerID string) error

	// Health configuration
	SetHealthCheck(ctx context.Context, containerID string, config *types.HealthCheckConfig) error
	RemoveHealthCheck(ctx context.Context, containerID string) error
}

// MetricsCollector defines the interface for collecting Docker metrics
type MetricsCollector interface {
	// Container metrics
	GetContainerMetrics(ctx context.Context, containerID string) (*types.ContainerMetrics, error)
	GetAllContainerMetrics(ctx context.Context) (map[string]*types.ContainerMetrics, error)
	StreamContainerMetrics(ctx context.Context, containerID string) (<-chan *types.ContainerMetrics, error)

	// System metrics
	GetSystemMetrics(ctx context.Context) (*SystemMetrics, error)
	GetResourceUsage(ctx context.Context) (*ResourceUsage, error)

	// Historical metrics
	GetMetricsHistory(ctx context.Context, containerID string, since time.Time) ([]*types.ContainerMetrics, error)
	GetAggregatedMetrics(ctx context.Context, containerIDs []string, period time.Duration) (*AggregatedMetrics, error)

	// Metrics configuration
	SetMetricsInterval(interval time.Duration) error
	EnableMetricsCollection(containerID string) error
	DisableMetricsCollection(containerID string) error
}

// Supporting types and interfaces

// EventHandler defines the interface for handling Docker events
type EventHandler interface {
	HandleEvent(ctx context.Context, event Event) error
}

// LogOptions represents options for container log retrieval
type LogOptions struct {
	ShowStdout bool
	ShowStderr bool
	Since      string
	Until      string
	Timestamps bool
	Follow     bool
	Tail       string
	Details    bool
}

// ExecOptions represents options for container execution
type ExecOptions struct {
	AttachStdout bool
	AttachStderr bool
	AttachStdin  bool
	Tty          bool
	Env          []string
	WorkingDir   string
	User         string
	Privileged   bool
	Detach       bool
}

// ExecResult represents the result of container execution
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
}

// PullOptions represents options for image pulling
type PullOptions struct {
	RegistryAuth string
	Platform     string
	All          bool
	PrivilegeFunc func() (string, error)
}

// PushOptions represents options for image pushing
type PushOptions struct {
	RegistryAuth string
	Platform     string
	PrivilegeFunc func() (string, error)
}

// BuildOptions represents options for image building
type BuildOptions struct {
	Dockerfile   string
	Tags         []string
	BuildArgs    map[string]*string
	Target       string
	Platform     string
	CacheFrom    []string
	PullParent   bool
	NoCache      bool
	Remove       bool
	ForceRemove  bool
	Squash       bool
	NetworkMode  string
	Memory       int64
	MemorySwap   int64
	CPUShares    int64
	CPUQuota     int64
	CPUPeriod    int64
	CPUSetCPUs   string
	CPUSetMems   string
	Labels       map[string]string
	ShmSize      int64
	Ulimits      []UlimitConfig
}

// BuildResult represents the result of image building
type BuildResult struct {
	ImageID   string
	Warnings  []string
	Size      int64
	Duration  time.Duration
	BuildLogs []BuildLogEntry
}

// BuildLogEntry represents a single build log entry
type BuildLogEntry struct {
	Stream    string    `json:"stream,omitempty"`
	Error     string    `json:"error,omitempty"`
	Progress  string    `json:"progress,omitempty"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// UlimitConfig represents ulimit configuration
type UlimitConfig struct {
	Name string `json:"name"`
	Soft int64  `json:"soft"`
	Hard int64  `json:"hard"`
}

// Additional supporting types would be defined here...
// (Network, Volume, Event, SystemInfo, etc.)

// These types would contain the actual structure definitions
// but are omitted here for brevity. They should be defined based on
// the specific requirements of the Docker API integration.

// Network represents a Docker network
type Network struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Driver  string                 `json:"driver"`
	Options map[string]string      `json:"options,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty"`
	Created time.Time              `json:"created"`
	IPAM    NetworkIPAM            `json:"ipam,omitempty"`
	Scope   string                 `json:"scope"`
	Internal bool                  `json:"internal"`
	Ingress bool                   `json:"ingress"`
	ConfigOnly bool                `json:"config_only"`
	Containers map[string]NetworkEndpoint `json:"containers,omitempty"`
}

// NetworkIPAM represents IP Address Management for networks
type NetworkIPAM struct {
	Driver  string             `json:"driver"`
	Config  []NetworkIPAMConfig `json:"config,omitempty"`
	Options map[string]string   `json:"options,omitempty"`
}

// NetworkIPAMConfig represents IPAM configuration
type NetworkIPAMConfig struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
	IPRange string `json:"ip_range,omitempty"`
}

// NetworkEndpoint represents a network endpoint
type NetworkEndpoint struct {
	Name        string `json:"name"`
	EndpointID  string `json:"endpoint_id"`
	MacAddress  string `json:"mac_address"`
	IPv4Address string `json:"ipv4_address"`
	IPv6Address string `json:"ipv6_address"`
}

// NetworkListOptions represents options for listing networks
type NetworkListOptions struct {
	Filters map[string][]string `json:"filters,omitempty"`
}

// NetworkCreateOptions represents options for creating networks
type NetworkCreateOptions struct {
	Driver     string                 `json:"driver,omitempty"`
	Options    map[string]string      `json:"options,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	IPAM       *NetworkIPAM           `json:"ipam,omitempty"`
	Internal   bool                   `json:"internal"`
	Attachable bool                   `json:"attachable"`
	Ingress    bool                   `json:"ingress"`
	ConfigOnly bool                   `json:"config_only"`
	Scope      string                 `json:"scope,omitempty"`
}

// NetworkConfig represents network configuration for containers
type NetworkConfig struct {
	IPAMConfig *NetworkEndpointIPAMConfig `json:"ipam_config,omitempty"`
	Links      []string                   `json:"links,omitempty"`
	Aliases    []string                   `json:"aliases,omitempty"`
}

// NetworkEndpointIPAMConfig represents IPAM configuration for endpoints
type NetworkEndpointIPAMConfig struct {
	IPv4Address string `json:"ipv4_address,omitempty"`
	IPv6Address string `json:"ipv6_address,omitempty"`
}

// NetworkInspect represents detailed network information
type NetworkInspect struct {
	Network
	Options map[string]string `json:"options,omitempty"`
}

// Volume represents a Docker volume
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	Scope      string            `json:"scope"`
	CreatedAt  time.Time         `json:"created_at"`
	Status     map[string]interface{} `json:"status,omitempty"`
	UsageData  *VolumeUsageData  `json:"usage_data,omitempty"`
}

// VolumeUsageData represents volume usage statistics
type VolumeUsageData struct {
	Size     int64 `json:"size"`
	RefCount int64 `json:"ref_count"`
}

// VolumeListOptions represents options for listing volumes
type VolumeListOptions struct {
	Filters map[string][]string `json:"filters,omitempty"`
}

// VolumeCreateOptions represents options for creating volumes
type VolumeCreateOptions struct {
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// Event represents a Docker event
type Event struct {
	Type   string            `json:"type"`
	Action string            `json:"action"`
	Actor  EventActor        `json:"actor"`
	Time   int64             `json:"time"`
	TimeNano int64           `json:"time_nano"`
	Scope  string            `json:"scope,omitempty"`
}

// EventActor represents the actor in a Docker event
type EventActor struct {
	ID         string            `json:"id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// EventOptions represents options for event monitoring
type EventOptions struct {
	Since   string              `json:"since,omitempty"`
	Until   string              `json:"until,omitempty"`
	Filters map[string][]string `json:"filters,omitempty"`
}

// EventStats represents event statistics
type EventStats struct {
	TotalEvents     int64                `json:"total_events"`
	EventsByType    map[string]int64     `json:"events_by_type"`
	EventsByAction  map[string]int64     `json:"events_by_action"`
	LastEventTime   time.Time            `json:"last_event_time"`
	EventRate       float64              `json:"event_rate"` // events per minute
}

// Additional types for completeness...
type SystemInfo struct{}
type Version struct{}
type DiskUsage struct{}
type PruneResult struct{}
type ImageInspect struct{}
type ImageHistoryEntry struct{}
type ImageSearchResult struct{}
type ImageLayer struct{}
type ImageManifest struct{}
type ContainerProcess struct{}
type ContainerChange struct{}
type CommitOptions struct{}
type AttachOptions struct{}
type AttachResult struct{}
type HealthCheckResult struct{}
type SystemMetrics struct{}
type ResourceUsage struct{}
type AggregatedMetrics struct{}