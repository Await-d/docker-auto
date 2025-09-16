package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SystemConfig represents system configuration settings
type SystemConfig struct {
	ID          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ConfigKey   string    `json:"config_key" gorm:"uniqueIndex:idx_system_configs_key;not null;size:100"`
	ConfigValue string    `json:"config_value" gorm:"type:jsonb;not null"`
	Description string    `json:"description,omitempty" gorm:"type:text"`
	IsEncrypted bool      `json:"is_encrypted" gorm:"not null;default:false"`
	IsSystem    bool      `json:"is_system" gorm:"not null;default:false;index:idx_system_configs_is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SystemConfigFilter represents filters for querying system configurations
type SystemConfigFilter struct {
	ConfigKey   string `json:"config_key,omitempty"`
	IsSystem    *bool  `json:"is_system,omitempty"`
	IsEncrypted *bool  `json:"is_encrypted,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	OrderBy     string `json:"order_by,omitempty"`
}

// ConfigValue represents a typed configuration value
type ConfigValue struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	IsSystem    bool        `json:"is_system"`
	IsEncrypted bool        `json:"is_encrypted"`
}

// Default configuration keys
const (
	// Application settings
	ConfigKeyAppVersion           = "app.version"
	ConfigKeyAppInitialized       = "app.initialized"
	ConfigKeyAppMaintenanceMode   = "app.maintenance_mode"

	// Image check settings
	ConfigKeyImageCheckInterval      = "image_check.interval"
	ConfigKeyImageCheckMaxConcurrent = "image_check.max_concurrent"
	ConfigKeyImageCheckTimeout       = "image_check.timeout"
	ConfigKeyImageCheckRetryCount    = "image_check.retry_count"

	// Update settings
	ConfigKeyUpdateDefaultStrategy   = "update.default_strategy"
	ConfigKeyUpdateMaxConcurrent     = "update.max_concurrent"
	ConfigKeyUpdateTimeout           = "update.timeout"
	ConfigKeyUpdateRollbackEnabled   = "update.rollback_enabled"

	// Notification settings
	ConfigKeyNotificationEnabled     = "notification.enabled"
	ConfigKeyNotificationEmail       = "notification.email"
	ConfigKeyNotificationWebhook     = "notification.webhook"
	ConfigKeyNotificationSlack       = "notification.slack"

	// Cleanup settings
	ConfigKeyCleanupLogRetentionDays     = "cleanup.log_retention_days"
	ConfigKeyCleanupHistoryRetentionCount = "cleanup.history_retention_count"
	ConfigKeyCleanupImageCacheRetentionDays = "cleanup.image_cache_retention_days"

	// Security settings
	ConfigKeySecurityJWTSecret          = "security.jwt_secret"
	ConfigKeySecurityJWTExpirationTime  = "security.jwt_expiration_time"
	ConfigKeySecurityPasswordMinLength  = "security.password_min_length"
	ConfigKeySecuritySessionTimeout     = "security.session_timeout"

	// Docker settings
	ConfigKeyDockerHost           = "docker.host"
	ConfigKeyDockerAPIVersion     = "docker.api_version"
	ConfigKeyDockerTLSVerify      = "docker.tls_verify"
	ConfigKeyDockerCertPath       = "docker.cert_path"
)

// TableName returns the table name for SystemConfig model
func (SystemConfig) TableName() string {
	return "system_configs"
}

// GetStringValue returns the configuration value as string
func (sc *SystemConfig) GetStringValue() (string, error) {
	var value string
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// GetIntValue returns the configuration value as integer
func (sc *SystemConfig) GetIntValue() (int, error) {
	var value int
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// GetBoolValue returns the configuration value as boolean
func (sc *SystemConfig) GetBoolValue() (bool, error) {
	var value bool
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// GetFloatValue returns the configuration value as float64
func (sc *SystemConfig) GetFloatValue() (float64, error) {
	var value float64
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// GetMapValue returns the configuration value as map[string]interface{}
func (sc *SystemConfig) GetMapValue() (map[string]interface{}, error) {
	var value map[string]interface{}
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// GetSliceValue returns the configuration value as []interface{}
func (sc *SystemConfig) GetSliceValue() ([]interface{}, error) {
	var value []interface{}
	err := json.Unmarshal([]byte(sc.ConfigValue), &value)
	return value, err
}

// SetValue sets the configuration value from any type
func (sc *SystemConfig) SetValue(value interface{}) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	sc.ConfigValue = string(jsonBytes)
	return nil
}

// IsEditable checks if the configuration can be edited by users
func (sc *SystemConfig) IsEditable() bool {
	return !sc.IsSystem
}

// GetValueType returns the type of the configuration value
func (sc *SystemConfig) GetValueType() string {
	var value interface{}
	if err := json.Unmarshal([]byte(sc.ConfigValue), &value); err != nil {
		return "unknown"
	}

	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// ToConfigValue converts SystemConfig to ConfigValue
func (sc *SystemConfig) ToConfigValue() (*ConfigValue, error) {
	var value interface{}
	if err := json.Unmarshal([]byte(sc.ConfigValue), &value); err != nil {
		return nil, err
	}

	return &ConfigValue{
		Key:         sc.ConfigKey,
		Value:       value,
		Type:        sc.GetValueType(),
		Description: sc.Description,
		IsSystem:    sc.IsSystem,
		IsEncrypted: sc.IsEncrypted,
	}, nil
}

// GetDefaultConfigs returns default system configurations
func GetDefaultConfigs() []SystemConfig {
	return []SystemConfig{
		{
			ConfigKey:   ConfigKeyAppVersion,
			ConfigValue: `"1.0.0"`,
			Description: "Application version",
			IsSystem:    true,
		},
		{
			ConfigKey:   ConfigKeyAppInitialized,
			ConfigValue: `true`,
			Description: "Application initialization status",
			IsSystem:    true,
		},
		{
			ConfigKey:   ConfigKeyAppMaintenanceMode,
			ConfigValue: `false`,
			Description: "Application maintenance mode",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyImageCheckInterval,
			ConfigValue: `60`,
			Description: "Default image check interval in minutes",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyImageCheckMaxConcurrent,
			ConfigValue: `10`,
			Description: "Maximum concurrent image checks",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyImageCheckTimeout,
			ConfigValue: `300`,
			Description: "Image check timeout in seconds",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyUpdateDefaultStrategy,
			ConfigValue: `"recreate"`,
			Description: "Default update strategy",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyUpdateMaxConcurrent,
			ConfigValue: `5`,
			Description: "Maximum concurrent updates",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyNotificationEnabled,
			ConfigValue: `true`,
			Description: "Enable notifications",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyCleanupLogRetentionDays,
			ConfigValue: `30`,
			Description: "Log retention period in days",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeyCleanupHistoryRetentionCount,
			ConfigValue: `1000`,
			Description: "Update history retention count",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeySecurityPasswordMinLength,
			ConfigValue: `8`,
			Description: "Minimum password length",
			IsSystem:    false,
		},
		{
			ConfigKey:   ConfigKeySecuritySessionTimeout,
			ConfigValue: `3600`,
			Description: "Session timeout in seconds",
			IsSystem:    false,
		},
	}
}

// ValidateConfigKey checks if the configuration key is valid
func ValidateConfigKey(key string) error {
	if key == "" {
		return fmt.Errorf("configuration key cannot be empty")
	}
	if len(key) > 100 {
		return fmt.Errorf("configuration key too long (max 100 characters)")
	}
	return nil
}

// ValidateConfigValue checks if the configuration value is valid JSON
func ValidateConfigValue(value string) error {
	var v interface{}
	return json.Unmarshal([]byte(value), &v)
}

// BeforeCreate hook for SystemConfig model
func (sc *SystemConfig) BeforeCreate(tx *gorm.DB) error {
	if err := ValidateConfigKey(sc.ConfigKey); err != nil {
		return err
	}
	if err := ValidateConfigValue(sc.ConfigValue); err != nil {
		return err
	}
	return nil
}

// BeforeUpdate hook for SystemConfig model
func (sc *SystemConfig) BeforeUpdate(tx *gorm.DB) error {
	if err := ValidateConfigValue(sc.ConfigValue); err != nil {
		return err
	}
	return nil
}