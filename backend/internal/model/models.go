package model

import (
	"gorm.io/gorm"
)

// AllModels returns a slice of all model structs for auto-migration
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&UserSession{},
		&ActivityLog{},
		&Container{},
		&RegistryCredentials{},
		&UpdateHistory{},
		&ImageVersion{},
		&SystemConfig{},
		&NotificationTemplate{},
		&NotificationLog{},
		&ScheduledTask{},
		&TaskExecutionLog{},
	}
}

// AutoMigrate runs auto-migration for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// DatabaseSeeder represents the interface for seeding database with initial data
type DatabaseSeeder interface {
	Seed(db *gorm.DB) error
}

// DefaultSeeder implements DatabaseSeeder for default data
type DefaultSeeder struct{}

// Seed seeds the database with default data
func (ds *DefaultSeeder) Seed(db *gorm.DB) error {
	// Seed default system configurations
	for _, config := range GetDefaultConfigs() {
		var existing SystemConfig
		if err := db.Where("config_key = ?", config.ConfigKey).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&config).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Seed default notification templates
	for _, template := range GetDefaultNotificationTemplates() {
		var existing NotificationTemplate
		if err := db.Where("name = ?", template.Name).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&template).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

// CreateDefaultAdminUser creates the default admin user if it doesn't exist
func CreateDefaultAdminUser(db *gorm.DB, username, email, passwordHash string) error {
	var existing User
	if err := db.Where("username = ? OR email = ?", username, email).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			admin := User{
				Username:     username,
				Email:        email,
				PasswordHash: passwordHash,
				Role:         UserRoleAdmin,
				IsActive:     true,
			}
			return db.Create(&admin).Error
		}
		return err
	}
	return nil
}

// ValidateModelConstraints validates that all model constraints are properly set
func ValidateModelConstraints() []string {
	var errors []string

	// Check UserRole constraints
	validRoles := GetValidRoles()
	if len(validRoles) == 0 {
		errors = append(errors, "No valid user roles defined")
	}

	// Check ContainerStatus constraints
	validStatuses := GetValidContainerStatuses()
	if len(validStatuses) == 0 {
		errors = append(errors, "No valid container statuses defined")
	}

	// Check UpdatePolicy constraints
	validPolicies := GetValidUpdatePolicies()
	if len(validPolicies) == 0 {
		errors = append(errors, "No valid update policies defined")
	}

	// Check NotificationType constraints
	validNotificationTypes := GetValidNotificationTypes()
	if len(validNotificationTypes) == 0 {
		errors = append(errors, "No valid notification types defined")
	}

	// Check TaskType constraints
	validTaskTypes := GetValidTaskTypes()
	if len(validTaskTypes) == 0 {
		errors = append(errors, "No valid task types defined")
	}

	return errors
}

// GetModelTableNames returns a map of model names to table names
func GetModelTableNames() map[string]string {
	return map[string]string{
		"User":                 User{}.TableName(),
		"UserSession":          UserSession{}.TableName(),
		"ActivityLog":          ActivityLog{}.TableName(),
		"Container":            Container{}.TableName(),
		"RegistryCredentials":  RegistryCredentials{}.TableName(),
		"UpdateHistory":        UpdateHistory{}.TableName(),
		"ImageVersion":         ImageVersion{}.TableName(),
		"SystemConfig":         SystemConfig{}.TableName(),
		"NotificationTemplate": NotificationTemplate{}.TableName(),
		"NotificationLog":      NotificationLog{}.TableName(),
		"ScheduledTask":        ScheduledTask{}.TableName(),
		"TaskExecutionLog":     TaskExecutionLog{}.TableName(),
	}
}

// CleanupExpiredData removes expired data from various tables
func CleanupExpiredData(db *gorm.DB) error {
	// Clean up expired user sessions
	if err := db.Where("expires_at < ?", "NOW()").Delete(&UserSession{}).Error; err != nil {
		return err
	}

	// Clean up old activity logs (older than 30 days)
	if err := db.Where("created_at < ?", "NOW() - INTERVAL '30 days'").Delete(&ActivityLog{}).Error; err != nil {
		return err
	}

	// Clean up old notification logs (older than 30 days)
	if err := db.Where("created_at < ?", "NOW() - INTERVAL '30 days'").Delete(&NotificationLog{}).Error; err != nil {
		return err
	}

	// Clean up old task execution logs (older than 30 days)
	if err := db.Where("started_at < ?", "NOW() - INTERVAL '30 days'").Delete(&TaskExecutionLog{}).Error; err != nil {
		return err
	}

	return nil
}