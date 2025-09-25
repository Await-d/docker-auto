package model

import "fmt"

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Container represents a Docker container managed by the system
type Container struct {
	ID            int             `json:"id" gorm:"primaryKey;autoIncrement"`
	Name          string          `json:"name" gorm:"uniqueIndex;not null;size:255;index:idx_containers_name"`
	Image         string          `json:"image" gorm:"not null;size:255;index:idx_containers_image"`
	Tag           string          `json:"tag" gorm:"not null;size:100;default:'latest'"`
	ContainerID   string          `json:"container_id,omitempty" gorm:"uniqueIndex:idx_containers_container_id;size:64"`
	Status        ContainerStatus `json:"status" gorm:"not null;default:'stopped';index:idx_containers_status"`
	ConfigJSON    string          `json:"config_json" gorm:"type:jsonb;not null;default:'{}'"`
	UpdatePolicy  UpdatePolicy    `json:"update_policy" gorm:"not null;default:'auto';index:idx_containers_update_policy"`
	RegistryURL   string          `json:"registry_url,omitempty" gorm:"size:255"`
	RegistryAuth  string          `json:"registry_auth,omitempty" gorm:"type:jsonb"`
	HealthCheck   string          `json:"health_check,omitempty" gorm:"type:jsonb"`
	Labels        string          `json:"labels" gorm:"type:jsonb;default:'{}'"`
	Environment   string          `json:"environment" gorm:"type:jsonb;default:'{}'"`
	Ports         string          `json:"ports" gorm:"type:jsonb;default:'[]'"`
	Volumes       string          `json:"volumes" gorm:"type:jsonb;default:'[]'"`
	RestartPolicy string          `json:"restart_policy" gorm:"size:20;default:'unless-stopped'"`
	CreatedBy     *int            `json:"created_by,omitempty" gorm:"index:idx_containers_created_by"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`

	// Runtime data fields for Docker integration
	RuntimeState      RuntimeState    `json:"runtime_state,omitempty" gorm:"embedded;embeddedPrefix:runtime_"`
	LastMetrics       *ContainerMetricsSnapshot `json:"last_metrics,omitempty" gorm:"type:jsonb"`
	LastHealthCheck   time.Time       `json:"last_health_check"`
	HealthStatus      string          `json:"health_status" gorm:"size:20;default:'unknown';index:idx_containers_health_status"`
	ResourceUsage     *ResourceUsageData `json:"resource_usage,omitempty" gorm:"type:jsonb"`
	NetworkSettings   string          `json:"network_settings,omitempty" gorm:"type:jsonb"`
	MountInfo         string          `json:"mount_info,omitempty" gorm:"type:jsonb"`
	ProcessInfo       *ProcessInfo    `json:"process_info,omitempty" gorm:"type:jsonb"`
	LastSyncAt        time.Time       `json:"last_sync_at"`

	// Relationships
	CreatedByUser      *User              `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
	UpdateHistories    []UpdateHistory    `json:"update_histories,omitempty" gorm:"foreignKey:ContainerID"`
	MonitoringMetrics  []MonitoringMetrics `json:"monitoring_metrics,omitempty" gorm:"foreignKey:ContainerID"`
	TerminalSessions   []TerminalSession  `json:"terminal_sessions,omitempty" gorm:"foreignKey:ContainerID"`
}

// ContainerStatus defines container status
type ContainerStatus string

const (
	ContainerStatusRunning    ContainerStatus = "running"
	ContainerStatusStopped    ContainerStatus = "stopped"
	ContainerStatusPaused     ContainerStatus = "paused"
	ContainerStatusRestarting ContainerStatus = "restarting"
	ContainerStatusRemoving   ContainerStatus = "removing"
	ContainerStatusExited     ContainerStatus = "exited"
	ContainerStatusDead       ContainerStatus = "dead"
	ContainerStatusUnknown    ContainerStatus = "unknown"
)

// UpdatePolicy defines update policies
type UpdatePolicy string

const (
	UpdatePolicyAuto      UpdatePolicy = "auto"
	UpdatePolicyManual    UpdatePolicy = "manual"
	UpdatePolicyScheduled UpdatePolicy = "scheduled"
	UpdatePolicyDisabled  UpdatePolicy = "disabled"
)

// RegistryCredentials represents registry authentication
type RegistryCredentials struct {
	ID              int                    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string                 `json:"name" gorm:"uniqueIndex;not null;size:100"`
	RegistryURL     string                 `json:"registry_url" gorm:"not null;size:255;index:idx_registry_credentials_registry_url"`
	Username        string                 `json:"username,omitempty" gorm:"size:100"`
	PasswordEncrypted string               `json:"-" gorm:"type:text"`
	TokenEncrypted  string                 `json:"-" gorm:"type:text"`
	AuthType        RegistryAuthType       `json:"auth_type" gorm:"not null;default:'basic'"`
	IsDefault       bool                   `json:"is_default" gorm:"not null;default:false;index:idx_registry_credentials_is_default"`
	IsActive        bool                   `json:"is_active" gorm:"not null;default:true;index:idx_registry_credentials_is_active"`
	Metadata        string                 `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedBy       *int                   `json:"created_by,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	// Relationships
	CreatedByUser *User `json:"-" gorm:"foreignKey:CreatedBy"`
}

// RegistryAuthType defines registry authentication types
type RegistryAuthType string

const (
	RegistryAuthTypeBasic  RegistryAuthType = "basic"
	RegistryAuthTypeToken  RegistryAuthType = "token"
	RegistryAuthTypeOAuth  RegistryAuthType = "oauth"
)


// ContainerFilter represents filters for querying containers
type ContainerFilter struct {
	CreatedBy    *int            `json:"created_by,omitempty"`
	Name         string          `json:"name,omitempty"`
	Image        string          `json:"image,omitempty"`
	Status       ContainerStatus `json:"status,omitempty"`
	UpdatePolicy UpdatePolicy    `json:"update_policy,omitempty"`
	Limit        int             `json:"limit,omitempty"`
	Offset       int             `json:"offset,omitempty"`
	OrderBy      string          `json:"order_by,omitempty"`
}

// RegistryCredentialsFilter represents filters for querying registry credentials
type RegistryCredentialsFilter struct {
	Name        string           `json:"name,omitempty"`
	RegistryURL string           `json:"registry_url,omitempty"`
	AuthType    RegistryAuthType `json:"auth_type,omitempty"`
	IsDefault   *bool            `json:"is_default,omitempty"`
	IsActive    *bool            `json:"is_active,omitempty"`
	CreatedBy   *int             `json:"created_by,omitempty"`
	Limit       int              `json:"limit,omitempty"`
	Offset      int              `json:"offset,omitempty"`
	OrderBy     string           `json:"order_by,omitempty"`
}

// TableName returns the table name for Container model
func (Container) TableName() string {
	return "containers"
}

// TableName returns the table name for RegistryCredentials model
func (RegistryCredentials) TableName() string {
	return "registry_credentials"
}

// IsRunning checks if container is running
func (c *Container) IsRunning() bool {
	return c.Status == ContainerStatusRunning
}

// IsAutoUpdateEnabled checks if auto update is enabled
func (c *Container) IsAutoUpdateEnabled() bool {
	return c.UpdatePolicy == UpdatePolicyAuto
}

// GetFullImageName returns full image name with tag
func (c *Container) GetFullImageName() string {
	if c.Tag == "" {
		return c.Image + ":latest"
	}
	return c.Image + ":" + c.Tag
}

// GetValidStatuses returns all valid container statuses
func GetValidContainerStatuses() []ContainerStatus {
	return []ContainerStatus{
		ContainerStatusRunning,
		ContainerStatusStopped,
		ContainerStatusPaused,
		ContainerStatusRestarting,
		ContainerStatusRemoving,
		ContainerStatusExited,
		ContainerStatusDead,
		ContainerStatusUnknown,
	}
}

// GetValidUpdatePolicies returns all valid update policies
func GetValidUpdatePolicies() []UpdatePolicy {
	return []UpdatePolicy{
		UpdatePolicyAuto,
		UpdatePolicyManual,
		UpdatePolicyScheduled,
		UpdatePolicyDisabled,
	}
}

// RuntimeState represents the current runtime state of a container
type RuntimeState struct {
	ActualStatus      string    `json:"actual_status" gorm:"column:actual_status;size:50"`
	Pid               int       `json:"pid" gorm:"column:pid"`
	ExitCode          int       `json:"exit_code" gorm:"column:exit_code"`
	StartedAt         time.Time `json:"started_at" gorm:"column:started_at"`
	FinishedAt        time.Time `json:"finished_at" gorm:"column:finished_at"`
	RestartCount      int       `json:"restart_count" gorm:"column:restart_count"`
	OOMKilled         bool      `json:"oom_killed" gorm:"column:oom_killed"`
	Error             string    `json:"error,omitempty" gorm:"column:error;type:text"`
	Platform          string    `json:"platform" gorm:"column:platform;size:100"`
	ImageDigest       string    `json:"image_digest" gorm:"column:image_digest;size:200"`
	LogPath           string    `json:"log_path" gorm:"column:log_path;size:500"`
}

// ContainerMetricsSnapshot represents a snapshot of container metrics for caching
type ContainerMetricsSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	CPUPercent      float64   `json:"cpu_percent"`
	MemoryPercent   float64   `json:"memory_percent"`
	MemoryUsage     uint64    `json:"memory_usage"`
	MemoryLimit     uint64    `json:"memory_limit"`
	NetworkRx       uint64    `json:"network_rx"`
	NetworkTx       uint64    `json:"network_tx"`
	BlockRead       uint64    `json:"block_read"`
	BlockWrite      uint64    `json:"block_write"`
	PIDs            uint64    `json:"pids"`
	OverallHealth   string    `json:"overall_health"`
}

// ResourceUsageData represents resource usage data for the container
type ResourceUsageData struct {
	CPUShares       int64  `json:"cpu_shares,omitempty"`
	MemoryLimit     int64  `json:"memory_limit,omitempty"`
	MemorySwapLimit int64  `json:"memory_swap_limit,omitempty"`
	CPUQuota        int64  `json:"cpu_quota,omitempty"`
	CPUPeriod       int64  `json:"cpu_period,omitempty"`
	BlkioWeight     uint16 `json:"blkio_weight,omitempty"`
	OOMKillDisable  bool   `json:"oom_kill_disable"`
	PidsLimit       int64  `json:"pids_limit,omitempty"`
}

// ProcessInfo represents process information inside the container
type ProcessInfo struct {
	Pid       int      `json:"pid"`
	Ppid      int      `json:"ppid"`
	Name      string   `json:"name"`
	Cmdline   string   `json:"cmdline"`
	Cwd       string   `json:"cwd"`
	Exe       string   `json:"exe"`
	Children  []int    `json:"children,omitempty"`
	Threads   int      `json:"threads"`
}

// GetValidRegistryAuthTypes returns all valid registry auth types
func GetValidRegistryAuthTypes() []RegistryAuthType {
	return []RegistryAuthType{
		RegistryAuthTypeBasic,
		RegistryAuthTypeToken,
		RegistryAuthTypeOAuth,
	}
}

// BeforeCreate hook for Container model
func (c *Container) BeforeCreate(tx *gorm.DB) error {
	if c.Tag == "" {
		c.Tag = "latest"
	}
	if c.UpdatePolicy == "" {
		c.UpdatePolicy = UpdatePolicyManual
	}
	if c.RestartPolicy == "" {
		c.RestartPolicy = "unless-stopped"
	}
	return nil
}

// BeforeCreate hook for RegistryCredentials model
func (rc *RegistryCredentials) BeforeCreate(tx *gorm.DB) error {
	if rc.AuthType == "" {
		rc.AuthType = RegistryAuthTypeBasic
	}
	return nil
}

// GORM hooks for Container runtime state synchronization

// BeforeUpdate hook for Container model
func (c *Container) BeforeUpdate(tx *gorm.DB) error {
	c.LastSyncAt = time.Now()
	return nil
}

// AfterFind hook for Container model - parse JSON fields
func (c *Container) AfterFind(tx *gorm.DB) error {
	// Parse JSON fields if needed
	return nil
}

// BeforeSave hook for Container model - validate JSON fields
func (c *Container) BeforeSave(tx *gorm.DB) error {
	// Validate JSON fields
	if c.LastMetrics != nil {
		if _, err := json.Marshal(c.LastMetrics); err != nil {
			return fmt.Errorf("invalid last_metrics JSON: %w", err)
		}
	}
	if c.ResourceUsage != nil {
		if _, err := json.Marshal(c.ResourceUsage); err != nil {
			return fmt.Errorf("invalid resource_usage JSON: %w", err)
		}
	}
	if c.ProcessInfo != nil {
		if _, err := json.Marshal(c.ProcessInfo); err != nil {
			return fmt.Errorf("invalid process_info JSON: %w", err)
		}
	}
	return nil
}

// Runtime state utility methods

// IsActuallyRunning checks if container is actually running (from Docker)
func (c *Container) IsActuallyRunning() bool {
	return c.RuntimeState.ActualStatus == "running" && c.RuntimeState.Pid > 0
}

// HasHealthCheck checks if container has health check configured
func (c *Container) HasHealthCheck() bool {
	return c.HealthCheck != "" && c.HealthCheck != "{}"
}

// IsHealthy checks if container is healthy based on last metrics
func (c *Container) IsHealthy() bool {
	if c.LastMetrics == nil {
		return c.HealthStatus == "healthy"
	}
	return c.LastMetrics.OverallHealth == "healthy"
}

// GetCPUUsagePercent returns current CPU usage percentage
func (c *Container) GetCPUUsagePercent() float64 {
	if c.LastMetrics == nil {
		return 0
	}
	return c.LastMetrics.CPUPercent
}

// GetMemoryUsagePercent returns current memory usage percentage
func (c *Container) GetMemoryUsagePercent() float64 {
	if c.LastMetrics == nil {
		return 0
	}
	return c.LastMetrics.MemoryPercent
}

// GetMemoryUsageFormatted returns formatted memory usage
func (c *Container) GetMemoryUsageFormatted() string {
	if c.LastMetrics == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.1f%% (%s / %s)",
		c.LastMetrics.MemoryPercent,
		formatBytes(c.LastMetrics.MemoryUsage),
		formatBytes(c.LastMetrics.MemoryLimit))
}

// GetNetworkUsageFormatted returns formatted network usage
func (c *Container) GetNetworkUsageFormatted() string {
	if c.LastMetrics == nil {
		return "N/A"
	}
	return fmt.Sprintf("RX: %s, TX: %s",
		formatBytes(c.LastMetrics.NetworkRx),
		formatBytes(c.LastMetrics.NetworkTx))
}

// GetUptimeFormatted returns formatted uptime
func (c *Container) GetUptimeFormatted() string {
	if c.RuntimeState.StartedAt.IsZero() || !c.IsActuallyRunning() {
		return "N/A"
	}
	duration := time.Since(c.RuntimeState.StartedAt)
	return formatDuration(duration)
}

// NeedsMetricsUpdate checks if metrics need updating
func (c *Container) NeedsMetricsUpdate() bool {
	if c.LastMetrics == nil {
		return true
	}
	// Update if metrics are older than 30 seconds
	return time.Since(c.LastMetrics.Timestamp) > 30*time.Second
}

// NeedsHealthCheck checks if health check is due
func (c *Container) NeedsHealthCheck() bool {
	if !c.HasHealthCheck() {
		return false
	}
	// Health check every 60 seconds
	return time.Since(c.LastHealthCheck) > 60*time.Second
}

// UpdateRuntimeState updates the runtime state from Docker container info
func (c *Container) UpdateRuntimeState(dockerStatus, error string, pid int, exitCode int, startedAt, finishedAt time.Time, restartCount int, oomKilled bool, platform, imageDigest, logPath string) {
	c.RuntimeState.ActualStatus = dockerStatus
	c.RuntimeState.Pid = pid
	c.RuntimeState.ExitCode = exitCode
	c.RuntimeState.StartedAt = startedAt
	c.RuntimeState.FinishedAt = finishedAt
	c.RuntimeState.RestartCount = restartCount
	c.RuntimeState.OOMKilled = oomKilled
	c.RuntimeState.Error = error
	c.RuntimeState.Platform = platform
	c.RuntimeState.ImageDigest = imageDigest
	c.RuntimeState.LogPath = logPath
	c.LastSyncAt = time.Now()
}

// UpdateMetricsSnapshot updates the cached metrics snapshot
func (c *Container) UpdateMetricsSnapshot(timestamp time.Time, cpuPercent, memoryPercent float64, memoryUsage, memoryLimit, networkRx, networkTx, blockRead, blockWrite, pids uint64, overallHealth string) {
	c.LastMetrics = &ContainerMetricsSnapshot{
		Timestamp:     timestamp,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		MemoryUsage:   memoryUsage,
		MemoryLimit:   memoryLimit,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          pids,
		OverallHealth: overallHealth,
	}
	c.LastSyncAt = time.Now()
}

// UpdateHealthStatus updates the health status
func (c *Container) UpdateHealthStatus(status string) {
	c.HealthStatus = status
	c.LastHealthCheck = time.Now()
	c.LastSyncAt = time.Now()
}

// Utility functions
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(duration time.Duration) string {
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}