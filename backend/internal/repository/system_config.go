package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"

	"gorm.io/gorm"
)

// systemConfigRepository implements SystemConfigRepository interface
type systemConfigRepository struct {
	db    *gorm.DB
	cache sync.Map // Simple in-memory cache for frequently accessed configs
}

// NewSystemConfigRepository creates a new system config repository
func NewSystemConfigRepository(db *gorm.DB) SystemConfigRepository {
	return &systemConfigRepository{db: db}
}

// Create creates a new system configuration
func (r *systemConfigRepository) Create(ctx context.Context, config *model.SystemConfig) error {
	if config == nil {
		return fmt.Errorf("system config cannot be nil")
	}

	// Validate required fields
	if config.ConfigKey == "" {
		return fmt.Errorf("config key is required")
	}
	if config.ConfigValue == "" {
		return fmt.Errorf("config value is required")
	}

	// Check for existing config with same key
	var existingConfig model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key = ?", config.ConfigKey).
		First(&existingConfig).Error

	if err == nil {
		return fmt.Errorf("configuration with key '%s' already exists", config.ConfigKey)
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create system config: %w", err)
	}

	// Invalidate cache for this key
	r.cache.Delete(config.ConfigKey)

	return nil
}

// GetByID retrieves a system configuration by ID
func (r *systemConfigRepository) GetByID(ctx context.Context, id int64) (*model.SystemConfig, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid config ID: %d", id)
	}

	var config model.SystemConfig
	err := r.db.WithContext(ctx).First(&config, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("config with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get config by ID: %w", err)
	}

	return &config, nil
}

// GetByKey retrieves a system configuration by key
func (r *systemConfigRepository) GetByKey(ctx context.Context, configKey string) (*model.SystemConfig, error) {
	if configKey == "" {
		return nil, fmt.Errorf("config key cannot be empty")
	}

	// Try cache first
	if cached, ok := r.cache.Load(configKey); ok {
		if config, ok := cached.(*model.SystemConfig); ok {
			return config, nil
		}
	}

	var config model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key = ?", configKey).
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("config with key '%s' not found", configKey)
		}
		return nil, fmt.Errorf("failed to get config by key: %w", err)
	}

	// Cache the result
	r.cache.Store(configKey, &config)

	return &config, nil
}

// Update updates an existing system configuration
func (r *systemConfigRepository) Update(ctx context.Context, config *model.SystemConfig) error {
	if config == nil {
		return fmt.Errorf("system config cannot be nil")
	}
	if config.ID <= 0 {
		return fmt.Errorf("invalid config ID: %d", config.ID)
	}

	// Check if config exists
	var existingConfig model.SystemConfig
	if err := r.db.WithContext(ctx).First(&existingConfig, config.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("config with ID %d not found", config.ID)
		}
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	// Prevent changing system configs if not allowed
	if existingConfig.IsSystem && !config.IsSystem {
		return fmt.Errorf("cannot modify system configuration")
	}

	// Update timestamp manually
	config.UpdatedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("failed to update system config: %w", err)
	}

	// Invalidate cache for this key
	r.cache.Delete(config.ConfigKey)

	return nil
}

// Delete deletes a system configuration by ID
func (r *systemConfigRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid config ID: %d", id)
	}

	// Check if config exists and is not a system config
	var config model.SystemConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("config with ID %d not found", id)
		}
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	if config.IsSystem {
		return fmt.Errorf("cannot delete system configuration")
	}

	result := r.db.WithContext(ctx).Delete(&model.SystemConfig{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete system config: %w", result.Error)
	}

	// Invalidate cache for this key
	r.cache.Delete(config.ConfigKey)

	return nil
}

// List retrieves system configurations with filtering and pagination
func (r *systemConfigRepository) List(ctx context.Context, filter *model.SystemConfigFilter) ([]*model.SystemConfig, int64, error) {
	var configs []*model.SystemConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SystemConfig{})

	// Apply filters
	if filter != nil {
		if filter.ConfigKey != "" {
			query = query.Where("config_key ILIKE ?", "%"+filter.ConfigKey+"%")
		}
		if filter.IsSystem != nil {
			query = query.Where("is_system = ?", *filter.IsSystem)
		}
		if filter.IsEncrypted != nil {
			query = query.Where("is_encrypted = ?", *filter.IsEncrypted)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count system configs: %w", err)
	}

	// Apply ordering
	orderBy := "config_key ASC"
	if filter != nil && filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	query = query.Order(orderBy)

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	if err := query.Find(&configs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list system configs: %w", err)
	}

	return configs, total, nil
}

// GetByCategory retrieves configurations by category (key prefix)
func (r *systemConfigRepository) GetByCategory(ctx context.Context, category string) ([]*model.SystemConfig, error) {
	if category == "" {
		return nil, fmt.Errorf("category cannot be empty")
	}

	var configs []*model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key LIKE ?", category+".%").
		Order("config_key ASC").
		Find(&configs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get configs by category: %w", err)
	}

	return configs, nil
}

// GetPublicConfigs retrieves all non-system configurations
func (r *systemConfigRepository) GetPublicConfigs(ctx context.Context) ([]*model.SystemConfig, error) {
	var configs []*model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("is_system = ?", false).
		Order("config_key ASC").
		Find(&configs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get public configs: %w", err)
	}

	return configs, nil
}

// SetValue sets the value for a configuration key
func (r *systemConfigRepository) SetValue(ctx context.Context, configKey, value string) error {
	if configKey == "" {
		return fmt.Errorf("config key cannot be empty")
	}

	// Validate JSON value
	var v interface{}
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		return fmt.Errorf("invalid JSON value: %w", err)
	}

	// Check if config exists
	var config model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key = ?", configKey).
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("config with key '%s' not found", configKey)
		}
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Check if it's a system config
	if config.IsSystem {
		return fmt.Errorf("cannot modify system configuration")
	}

	// Update the value
	result := r.db.WithContext(ctx).
		Model(&model.SystemConfig{}).
		Where("config_key = ?", configKey).
		Updates(map[string]interface{}{
			"config_value": value,
			"updated_at":   time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to set config value: %w", result.Error)
	}

	// Invalidate cache for this key
	r.cache.Delete(configKey)

	return nil
}

// GetValue retrieves the value for a configuration key
func (r *systemConfigRepository) GetValue(ctx context.Context, configKey string) (string, error) {
	if configKey == "" {
		return "", fmt.Errorf("config key cannot be empty")
	}

	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return "", err
	}

	return config.ConfigValue, nil
}

// GetValueWithDefault retrieves the value for a configuration key or returns default
func (r *systemConfigRepository) GetValueWithDefault(ctx context.Context, configKey, defaultValue string) (string, error) {
	value, err := r.GetValue(ctx, configKey)
	if err != nil {
		// If key not found, return default
		if strings.Contains(err.Error(), "not found") {
			return defaultValue, nil
		}
		return "", err
	}
	return value, nil
}

// SetValues sets multiple configuration values in a single transaction
func (r *systemConfigRepository) SetValues(ctx context.Context, configs map[string]string) error {
	if len(configs) == 0 {
		return fmt.Errorf("configs map cannot be empty")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range configs {
			// Validate JSON value
			var v interface{}
			if err := json.Unmarshal([]byte(value), &v); err != nil {
				return fmt.Errorf("invalid JSON value for key '%s': %w", key, err)
			}

			// Check if config exists and is not system
			var config model.SystemConfig
			err := tx.Where("config_key = ?", key).First(&config).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("config with key '%s' not found", key)
				}
				return fmt.Errorf("failed to get config '%s': %w", key, err)
			}

			if config.IsSystem {
				return fmt.Errorf("cannot modify system configuration '%s'", key)
			}

			// Update the value
			result := tx.Model(&model.SystemConfig{}).
				Where("config_key = ?", key).
				Updates(map[string]interface{}{
					"config_value": value,
					"updated_at":   time.Now().UTC(),
				})

			if result.Error != nil {
				return fmt.Errorf("failed to set config value for '%s': %w", key, result.Error)
			}

			// Invalidate cache for this key
			r.cache.Delete(key)
		}

		return nil
	})
}

// GetValues retrieves values for multiple configuration keys
func (r *systemConfigRepository) GetValues(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}

	var configs []*model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key IN ?", keys).
		Find(&configs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get config values: %w", err)
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.ConfigKey] = config.ConfigValue
		// Cache the result
		r.cache.Store(config.ConfigKey, config)
	}

	return result, nil
}

// RefreshConfigCache clears the internal cache
func (r *systemConfigRepository) RefreshConfigCache(ctx context.Context) error {
	// Clear all cached entries
	r.cache.Range(func(key, value interface{}) bool {
		r.cache.Delete(key)
		return true
	})

	return nil
}

// GetCachedValue retrieves a cached configuration value
func (r *systemConfigRepository) GetCachedValue(ctx context.Context, configKey string) (string, bool, error) {
	if configKey == "" {
		return "", false, fmt.Errorf("config key cannot be empty")
	}

	if cached, ok := r.cache.Load(configKey); ok {
		if config, ok := cached.(*model.SystemConfig); ok {
			return config.ConfigValue, true, nil
		}
	}

	return "", false, nil
}

// ValidateConfig validates a configuration key-value pair
func (r *systemConfigRepository) ValidateConfig(ctx context.Context, configKey, value string) error {
	if configKey == "" {
		return fmt.Errorf("config key cannot be empty")
	}

	// Validate JSON format
	var v interface{}
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Check if config exists
	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Check if it's editable
	if config.IsSystem {
		return fmt.Errorf("system configurations cannot be modified")
	}

	// Add specific validation rules based on config key
	switch configKey {
	case model.ConfigKeyImageCheckInterval:
		var interval int
		if err := json.Unmarshal([]byte(value), &interval); err != nil {
			return fmt.Errorf("image check interval must be a number")
		}
		if interval < 1 || interval > 1440 {
			return fmt.Errorf("image check interval must be between 1 and 1440 minutes")
		}

	case model.ConfigKeyImageCheckMaxConcurrent:
		var maxConcurrent int
		if err := json.Unmarshal([]byte(value), &maxConcurrent); err != nil {
			return fmt.Errorf("max concurrent checks must be a number")
		}
		if maxConcurrent < 1 || maxConcurrent > 100 {
			return fmt.Errorf("max concurrent checks must be between 1 and 100")
		}

	case model.ConfigKeySecurityPasswordMinLength:
		var minLength int
		if err := json.Unmarshal([]byte(value), &minLength); err != nil {
			return fmt.Errorf("password min length must be a number")
		}
		if minLength < 4 || minLength > 128 {
			return fmt.Errorf("password min length must be between 4 and 128")
		}
	}

	return nil
}

// ResetToDefault resets a configuration to its default value
func (r *systemConfigRepository) ResetToDefault(ctx context.Context, configKey string) error {
	if configKey == "" {
		return fmt.Errorf("config key cannot be empty")
	}

	// Get default configurations
	defaults := model.GetDefaultConfigs()
	var defaultConfig *model.SystemConfig

	for _, config := range defaults {
		if config.ConfigKey == configKey {
			defaultConfig = &config
			break
		}
	}

	if defaultConfig == nil {
		return fmt.Errorf("no default value found for config key '%s'", configKey)
	}

	// Check if config exists and is not system
	var config model.SystemConfig
	err := r.db.WithContext(ctx).
		Where("config_key = ?", configKey).
		First(&config).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("config with key '%s' not found", configKey)
		}
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config.IsSystem {
		return fmt.Errorf("cannot reset system configuration")
	}

	// Reset to default value
	result := r.db.WithContext(ctx).
		Model(&model.SystemConfig{}).
		Where("config_key = ?", configKey).
		Updates(map[string]interface{}{
			"config_value": defaultConfig.ConfigValue,
			"updated_at":   time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to reset config to default: %w", result.Error)
	}

	// Invalidate cache for this key
	r.cache.Delete(configKey)

	return nil
}

// GetConfigSchema returns a schema describing available configurations
func (r *systemConfigRepository) GetConfigSchema(ctx context.Context) (map[string]interface{}, error) {
	schema := map[string]interface{}{
		"categories": map[string]interface{}{
			"app": map[string]interface{}{
				"title":       "Application Settings",
				"description": "General application configuration",
			},
			"image_check": map[string]interface{}{
				"title":       "Image Check Settings",
				"description": "Configuration for Docker image checking",
			},
			"update": map[string]interface{}{
				"title":       "Update Settings",
				"description": "Configuration for container updates",
			},
			"notification": map[string]interface{}{
				"title":       "Notification Settings",
				"description": "Configuration for notifications",
			},
			"cleanup": map[string]interface{}{
				"title":       "Cleanup Settings",
				"description": "Configuration for data cleanup and retention",
			},
			"security": map[string]interface{}{
				"title":       "Security Settings",
				"description": "Security and authentication configuration",
			},
			"docker": map[string]interface{}{
				"title":       "Docker Settings",
				"description": "Docker daemon connection configuration",
			},
		},
		"fields": map[string]interface{}{
			model.ConfigKeyImageCheckInterval: map[string]interface{}{
				"type":        "number",
				"title":       "Check Interval",
				"description": "Image check interval in minutes",
				"minimum":     1,
				"maximum":     1440,
				"default":     60,
			},
			model.ConfigKeyImageCheckMaxConcurrent: map[string]interface{}{
				"type":        "number",
				"title":       "Max Concurrent Checks",
				"description": "Maximum number of concurrent image checks",
				"minimum":     1,
				"maximum":     100,
				"default":     10,
			},
			model.ConfigKeyNotificationEnabled: map[string]interface{}{
				"type":        "boolean",
				"title":       "Enable Notifications",
				"description": "Enable or disable notifications",
				"default":     true,
			},
			model.ConfigKeySecurityPasswordMinLength: map[string]interface{}{
				"type":        "number",
				"title":       "Minimum Password Length",
				"description": "Minimum required password length",
				"minimum":     4,
				"maximum":     128,
				"default":     8,
			},
		},
	}

	return schema, nil
}

// Helper functions for typed value access

// GetStringConfig retrieves a string configuration value
func (r *systemConfigRepository) GetStringConfig(ctx context.Context, configKey string) (string, error) {
	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return "", err
	}
	return config.GetStringValue()
}

// GetIntConfig retrieves an integer configuration value
func (r *systemConfigRepository) GetIntConfig(ctx context.Context, configKey string) (int, error) {
	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return 0, err
	}
	return config.GetIntValue()
}

// GetBoolConfig retrieves a boolean configuration value
func (r *systemConfigRepository) GetBoolConfig(ctx context.Context, configKey string) (bool, error) {
	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return false, err
	}
	return config.GetBoolValue()
}

// GetFloatConfig retrieves a float configuration value
func (r *systemConfigRepository) GetFloatConfig(ctx context.Context, configKey string) (float64, error) {
	config, err := r.GetByKey(ctx, configKey)
	if err != nil {
		return 0, err
	}
	return config.GetFloatValue()
}