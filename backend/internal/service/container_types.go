package service

import (
	"fmt"
	"time"

	"docker-auto/internal/model"
)

// Container service request types

// CreateContainerRequest represents a request to create a new container
type CreateContainerRequest struct {
	Name         string                 `json:"name" binding:"required" validate:"required,min=3,max=100"`
	Image        string                 `json:"image" binding:"required" validate:"required,min=3,max=255"`
	Tag          string                 `json:"tag" validate:"max=100"`
	Config       map[string]interface{} `json:"config"`
	UpdatePolicy string                 `json:"update_policy" validate:"oneof=auto manual scheduled disabled"`
	RegistryURL  string                 `json:"registry_url,omitempty" validate:"omitempty,url"`
	RegistryAuth *RegistryAuth          `json:"registry_auth,omitempty"`
}

// UpdateContainerRequest represents a request to update container configuration
type UpdateContainerRequest struct {
	Config       map[string]interface{} `json:"config,omitempty"`
	UpdatePolicy *string                `json:"update_policy,omitempty" validate:"omitempty,oneof=auto manual scheduled disabled"`
	RegistryURL  *string                `json:"registry_url,omitempty" validate:"omitempty,url"`
	RegistryAuth *RegistryAuth          `json:"registry_auth,omitempty"`
}

// UpdateImageRequest represents a request to update container image
type UpdateImageRequest struct {
	Strategy string `json:"strategy,omitempty" validate:"omitempty,oneof=recreate rolling blue_green"`
	Force    bool   `json:"force,omitempty"`
	Backup   bool   `json:"backup,omitempty"`
}

// BulkUpdateRequest represents a request for bulk container updates
type BulkUpdateRequest struct {
	ContainerIDs []int64              `json:"container_ids" binding:"required" validate:"required,min=1"`
	Action       string               `json:"action" binding:"required" validate:"required,oneof=start stop restart update"`
	UpdateImage  *UpdateImageRequest  `json:"update_image,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// RegistryAuth represents registry authentication information
type RegistryAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	AuthType string `json:"auth_type" validate:"oneof=basic token oauth"`
}

// Container service response types

// ContainerDetail represents detailed container information
type ContainerDetail struct {
	*model.Container
	DockerStatus *model.ContainerStatus `json:"docker_status,omitempty"`
	Metrics      *ContainerMetrics       `json:"metrics,omitempty"`
	UpdateInfo   *UpdateInfo             `json:"update_info,omitempty"`
	LogsSample   []string                `json:"logs_sample,omitempty"`
}

// ContainerSummary represents container summary for list views
type ContainerSummary struct {
	ID           int64                   `json:"id"`
	Name         string                  `json:"name"`
	Image        string                  `json:"image"`
	Tag          string                  `json:"tag"`
	Status       model.ContainerStatus   `json:"status"`
	DockerStatus string                  `json:"docker_status"`
	UpdatePolicy model.UpdatePolicy      `json:"update_policy"`
	HasUpdate    bool                    `json:"has_update"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// ContainerListResponse represents paginated container list response
type ContainerListResponse struct {
	Containers []*ContainerSummary `json:"containers"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	HasNext    bool                `json:"has_next"`
	HasPrev    bool                `json:"has_prev"`
}

// ContainerMetrics represents container performance metrics
type ContainerMetrics struct {
	CPUPercent    float64           `json:"cpu_percent"`
	MemoryUsage   int64             `json:"memory_usage"`
	MemoryLimit   int64             `json:"memory_limit"`
	MemoryPercent float64           `json:"memory_percent"`
	NetworkIO     *NetworkIOMetrics `json:"network_io,omitempty"`
	BlockIO       *BlockIOMetrics   `json:"block_io,omitempty"`
	PIDs          int               `json:"pids"`
	Timestamp     time.Time         `json:"timestamp"`
}

// NetworkIOMetrics represents network I/O metrics
type NetworkIOMetrics struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
}

// BlockIOMetrics represents disk I/O metrics
type BlockIOMetrics struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`
}

// Container operation types

// ContainerStatus represents current container runtime status
type ContainerStatus struct {
	ID           int64                   `json:"id"`
	Name         string                  `json:"name"`
	Status       model.ContainerStatus   `json:"status"`
	DockerStatus *model.ContainerStatus `json:"docker_status,omitempty"`
	Health       string                  `json:"health,omitempty"`
	Uptime       time.Duration           `json:"uptime"`
	RestartCount int                     `json:"restart_count"`
	LastRestart  *time.Time              `json:"last_restart,omitempty"`
	Timestamp    time.Time               `json:"timestamp"`
}

// ContainerStats represents container resource statistics
type ContainerStats struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Metrics   *ContainerMetrics `json:"metrics"`
	Timestamp time.Time         `json:"timestamp"`
}

// OperationResult represents the result of a container operation
type OperationResult struct {
	ContainerID int64  `json:"container_id"`
	Name        string `json:"name"`
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

// LogOptions represents options for retrieving container logs
type LogOptions struct {
	Since      time.Time `json:"since,omitempty"`
	Until      time.Time `json:"until,omitempty"`
	Tail       int       `json:"tail,omitempty"`
	Follow     bool      `json:"follow,omitempty"`
	Timestamps bool      `json:"timestamps,omitempty"`
}

// LogResponse represents container logs response
type LogResponse struct {
	ContainerID int64             `json:"container_id"`
	Name        string            `json:"name"`
	Logs        []LogEntry        `json:"logs"`
	Truncated   bool              `json:"truncated"`
	Since       time.Time         `json:"since"`
	Until       time.Time         `json:"until"`
	Count       int               `json:"count"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // stdout, stderr
	Message   string    `json:"message"`
}

// Container export/import types

// ContainerExport represents exported container configuration
type ContainerExport struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Tag          string                 `json:"tag"`
	Config       map[string]interface{} `json:"config"`
	UpdatePolicy string                 `json:"update_policy"`
	RegistryURL  string                 `json:"registry_url,omitempty"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Environment  map[string]string      `json:"environment,omitempty"`
	Ports        []PortMapping          `json:"ports,omitempty"`
	Volumes      []VolumeMapping        `json:"volumes,omitempty"`
	ExportedAt   time.Time              `json:"exported_at"`
	Version      string                 `json:"version"`
}

// PortMapping represents port mapping configuration
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port,omitempty"`
	Protocol      string `json:"protocol,omitempty"` // tcp, udp
	HostIP        string `json:"host_ip,omitempty"`
}

// VolumeMapping represents volume mapping configuration
type VolumeMapping struct {
	Source      string `json:"source"`      // host path or volume name
	Target      string `json:"target"`      // container path
	Type        string `json:"type"`        // bind, volume, tmpfs
	ReadOnly    bool   `json:"read_only"`
	Consistency string `json:"consistency,omitempty"` // default, consistent, cached, delegated
}

// Update related types

// UpdateInfo represents information about available updates
type UpdateInfo struct {
	ContainerID     int64                  `json:"container_id"`
	Name            string                 `json:"name"`
	CurrentImage    string                 `json:"current_image"`
	CurrentTag      string                 `json:"current_tag"`
	LatestImage     string                 `json:"latest_image,omitempty"`
	LatestTag       string                 `json:"latest_tag,omitempty"`
	UpdateAvailable bool                   `json:"update_available"`
	UpdateType      string                 `json:"update_type,omitempty"` // major, minor, patch, unknown
	LastChecked     time.Time              `json:"last_checked"`
	VersionInfo     *VersionComparisonResult `json:"version_info,omitempty"`
}

// VersionComparisonResult represents the result of version comparison
type VersionComparisonResult struct {
	CurrentVersion string                 `json:"current_version"`
	LatestVersion  string                 `json:"latest_version"`
	CompareResult  int                    `json:"compare_result"` // -1: current < latest, 0: equal, 1: current > latest
	VersionType    string                 `json:"version_type"`   // semantic, date, hash, unknown
	Changes        []VersionChange        `json:"changes,omitempty"`
	SecurityIssues []SecurityIssue        `json:"security_issues,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
}

// VersionChange represents a change between versions
type VersionChange struct {
	Type        string    `json:"type"`        // feature, bugfix, security, breaking
	Description string    `json:"description"`
	Impact      string    `json:"impact"`      // low, medium, high, critical
	Date        time.Time `json:"date,omitempty"`
}

// SecurityIssue represents a security issue found in the current version
type SecurityIssue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	CVSS        float64   `json:"cvss,omitempty"`
	FixedIn     string    `json:"fixed_in,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// Filter types

// ContainerFilter represents filters for container queries (extends model.ContainerFilter)
type ContainerFilter struct {
	*model.ContainerFilter
	HasUpdate    *bool     `json:"has_update,omitempty"`
	LastUpdated  time.Time `json:"last_updated,omitempty"`
	SearchQuery  string    `json:"search_query,omitempty"`
	SortBy       string    `json:"sort_by,omitempty"`       // name, created_at, updated_at, status
	SortOrder    string    `json:"sort_order,omitempty"`    // asc, desc
}

// Sync and maintenance types

// SyncResult represents the result of container status synchronization
type SyncResult struct {
	TotalContainers    int                    `json:"total_containers"`
	SyncedContainers   int                    `json:"synced_containers"`
	ErrorContainers    int                    `json:"error_containers"`
	StatusChanges      []ContainerStatusChange `json:"status_changes,omitempty"`
	Errors             []SyncError            `json:"errors,omitempty"`
	Duration           time.Duration          `json:"duration"`
	Timestamp          time.Time              `json:"timestamp"`
}

// ContainerStatusChange represents a status change detected during sync
type ContainerStatusChange struct {
	ContainerID int64                 `json:"container_id"`
	Name        string                `json:"name"`
	OldStatus   model.ContainerStatus `json:"old_status"`
	NewStatus   model.ContainerStatus `json:"new_status"`
	Reason      string                `json:"reason,omitempty"`
}

// SyncError represents an error that occurred during sync
type SyncError struct {
	ContainerID int64  `json:"container_id"`
	Name        string `json:"name"`
	Error       string `json:"error"`
	Recoverable bool   `json:"recoverable"`
}

// Validation helpers

// Validate validates CreateContainerRequest
func (r *CreateContainerRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("container name is required")
	}
	if r.Image == "" {
		return fmt.Errorf("container image is required")
	}
	if r.Tag == "" {
		r.Tag = "latest"
	}
	if r.UpdatePolicy == "" {
		r.UpdatePolicy = "manual"
	}
	return nil
}

// Validate validates UpdateContainerRequest
func (r *UpdateContainerRequest) Validate() error {
	if r.UpdatePolicy != nil && *r.UpdatePolicy != "" {
		validPolicies := []string{"auto", "manual", "scheduled", "disabled"}
		valid := false
		for _, policy := range validPolicies {
			if *r.UpdatePolicy == policy {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid update policy")
		}
	}
	return nil
}

// Helper functions

// GetSortableFields returns list of fields that can be used for sorting
func GetSortableFields() []string {
	return []string{"name", "created_at", "updated_at", "status", "image"}
}

// IsValidSortField checks if the field is valid for sorting
func IsValidSortField(field string) bool {
	validFields := GetSortableFields()
	for _, validField := range validFields {
		if field == validField {
			return true
		}
	}
	return false
}

// GetValidUpdateStrategies returns list of valid update strategies
func GetValidUpdateStrategies() []string {
	return []string{"recreate", "rolling", "blue_green"}
}

// IsValidUpdateStrategy checks if the strategy is valid
func IsValidUpdateStrategy(strategy string) bool {
	validStrategies := GetValidUpdateStrategies()
	for _, validStrategy := range validStrategies {
		if strategy == validStrategy {
			return true
		}
	}
	return false
}