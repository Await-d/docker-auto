package model

import (
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

	// Relationships
	CreatedByUser   *User           `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
	UpdateHistories []UpdateHistory `json:"update_histories,omitempty" gorm:"foreignKey:ContainerID"`
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