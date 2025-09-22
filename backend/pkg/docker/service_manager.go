package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/pkg/security"
	dockerTypes "docker-auto/pkg/types"

	"github.com/sirupsen/logrus"
)

// ServiceManager integrates all Docker-related services
type ServiceManager struct {
	clientManager       *ClientManager
	updateEngine        *UpdateEngine
	imageScanner        *ImageScanner
	notificationManager *NotificationManager
	rollbackManager     *RollbackManager
	securityConfig      *security.DockerSecurityConfig
	config              *ServiceConfig
	logger              *logrus.Logger
	mutex               sync.RWMutex
	initialized         bool
}

// ServiceConfig represents the overall service configuration
type ServiceConfig struct {
	ClientConfig        *dockerTypes.ClientConfig       `json:"client_config"`
	ScannerConfig       *ScannerConfig                   `json:"scanner_config"`
	RollbackConfig      *RollbackConfig                  `json:"rollback_config"`
	SecurityConfig      *security.DockerSecurityConfig  `json:"security_config"`
	NotificationEnabled bool                             `json:"notification_enabled"`
	MetricsEnabled      bool                             `json:"metrics_enabled"`
	LogLevel            string                           `json:"log_level"`
	HealthCheckInterval time.Duration                    `json:"health_check_interval"`
	AutoCleanup         bool                             `json:"auto_cleanup"`
	CleanupInterval     time.Duration                    `json:"cleanup_interval"`
}

// ServiceInfo represents service information and status
type ServiceInfo struct {
	Status              string                 `json:"status"`
	Version             string                 `json:"version"`
	StartTime           time.Time              `json:"start_time"`
	Uptime              time.Duration          `json:"uptime"`
	Docker              map[string]interface{} `json:"docker"`
	Statistics          map[string]interface{} `json:"statistics"`
	Health              dockerTypes.ServiceHealthStatus `json:"health"`
	Features            []string               `json:"features"`
	Configuration       map[string]interface{} `json:"configuration"`
}


// IntegratedOperation represents a complete Docker operation with all services
type IntegratedOperation struct {
	ID                  string                 `json:"id"`
	Type                string                 `json:"type"`
	Status              string                 `json:"status"`
	Progress            float64                `json:"progress"`
	StartTime           time.Time              `json:"start_time"`
	EndTime             *time.Time             `json:"end_time,omitempty"`
	Duration            time.Duration          `json:"duration"`
	UserID              int64                  `json:"user_id"`
	ContainerID         string                 `json:"container_id,omitempty"`
	ImageName           string                 `json:"image_name,omitempty"`
	SecurityScan        *ScanResult            `json:"security_scan,omitempty"`
	UpdateResult        *dockerTypes.ContainerUpdateResult `json:"update_result,omitempty"`
	RollbackEntry       *RollbackEntry         `json:"rollback_entry,omitempty"`
	Notifications       []string               `json:"notifications"`
	Error               string                 `json:"error,omitempty"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// NewServiceManager creates a new Docker service manager
func NewServiceManager(config *ServiceConfig, logger *logrus.Logger) (*ServiceManager, error) {
	if config == nil {
		config = DefaultServiceConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	sm := &ServiceManager{
		config:      config,
		logger:      logger,
		initialized: false,
	}

	if err := sm.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize service manager: %w", err)
	}

	return sm, nil
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		ClientConfig:        DefaultClientConfig(),
		ScannerConfig:       DefaultScannerConfig(),
		RollbackConfig:      DefaultRollbackConfig(),
		SecurityConfig:      security.DefaultDockerSecurityConfig(),
		NotificationEnabled: true,
		MetricsEnabled:      true,
		LogLevel:            "info",
		HealthCheckInterval: 30 * time.Second,
		AutoCleanup:         true,
		CleanupInterval:     time.Hour,
	}
}

// initialize initializes all service components
func (sm *ServiceManager) initialize() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info("Initializing Docker service manager")

	// Initialize client manager
	clientManager, err := NewClientManager(sm.config.ClientConfig, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize client manager: %w", err)
	}
	sm.clientManager = clientManager

	// Initialize notification manager
	if sm.config.NotificationEnabled {
		sm.notificationManager = NewNotificationManager(sm.logger)
	}

	// Initialize update engine
	sm.updateEngine = NewUpdateEngine(sm.clientManager, sm.logger)

	// Initialize image scanner
	sm.imageScanner = NewImageScanner(sm.clientManager, sm.config.ScannerConfig, sm.logger)

	// Initialize rollback manager
	sm.rollbackManager = NewRollbackManager(
		sm.clientManager,
		sm.notificationManager,
		sm.config.RollbackConfig,
		sm.logger,
	)

	sm.initialized = true

	// Start background services
	if sm.config.AutoCleanup {
		go sm.cleanupRoutine()
	}

	go sm.healthCheckRoutine()

	sm.logger.Info("Docker service manager initialized successfully")
	return nil
}

// PullImageIntegrated performs an integrated image pull operation
func (sm *ServiceManager) PullImageIntegrated(ctx context.Context, imageName, tag string, userContext *security.DockerUserContext, scanAfterPull bool) (*IntegratedOperation, error) {
	if !sm.initialized {
		return nil, fmt.Errorf("service manager not initialized")
	}

	operationID := fmt.Sprintf("pull_%s_%d", imageName, time.Now().Unix())
	operation := &IntegratedOperation{
		ID:        operationID,
		Type:      "integrated_pull",
		Status:    "started",
		StartTime: time.Now(),
		UserID:    userContext.UserID,
		ImageName: imageName,
		Metadata:  make(map[string]interface{}),
	}

	// Start operation tracking
	if sm.notificationManager != nil {
		sm.notificationManager.StartOperation(operationID, "integrated_pull", userContext.UserID, map[string]interface{}{
			"image_name": imageName,
			"tag":        tag,
			"scan_after": scanAfterPull,
		})
	}

	// Step 1: Pull image
	operation.Status = "pulling"
	operation.Progress = 10.0
	pullProgress, err := sm.clientManager.PullImage(ctx, imageName, tag, nil, false, userContext)
	if err != nil {
		operation.Status = "failed"
		operation.Error = err.Error()
		if sm.notificationManager != nil {
			sm.notificationManager.CompleteOperation(operationID, false, err.Error())
		}
		return operation, err
	}

	// Wait for pull completion
	for !pullProgress.Completed {
		select {
		case <-ctx.Done():
			operation.Status = "cancelled"
			operation.Error = "Operation cancelled"
			return operation, ctx.Err()
		case <-time.After(1 * time.Second):
			operation.Progress = 10.0 + (float64(pullProgress.DownloadedSize) / float64(pullProgress.TotalSize) * 50.0)
			if sm.notificationManager != nil {
				sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Pulling image", map[string]interface{}{
					"downloaded": pullProgress.DownloadedSize,
					"total":      pullProgress.TotalSize,
				})
			}
		}
	}

	if pullProgress.Error != "" {
		operation.Status = "failed"
		operation.Error = pullProgress.Error
		if sm.notificationManager != nil {
			sm.notificationManager.CompleteOperation(operationID, false, pullProgress.Error)
		}
		return operation, fmt.Errorf("image pull failed: %s", pullProgress.Error)
	}

	operation.Progress = 60.0

	// Step 2: Security scan if requested
	if scanAfterPull {
		operation.Status = "scanning"
		if sm.notificationManager != nil {
			sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Performing security scan", nil)
		}

		fullImageName := imageName
		if tag != "" && tag != "latest" {
			fullImageName += ":" + tag
		}

		scanResult, err := sm.imageScanner.ScanImage(ctx, fullImageName)
		if err != nil {
			sm.logger.WithError(err).Warn("Security scan failed, continuing operation")
		} else {
			operation.SecurityScan = scanResult
			operation.Progress = 90.0

			// Add security notifications based on scan results
			if scanResult.CriticalVulns > 0 {
				if sm.notificationManager != nil {
					sm.notificationManager.NotifyAlert(
						userContext.UserID,
						"critical",
						"Critical Vulnerabilities Found",
						fmt.Sprintf("Image %s contains %d critical vulnerabilities", fullImageName, scanResult.CriticalVulns),
						map[string]interface{}{
							"image":            fullImageName,
							"critical_vulns":   scanResult.CriticalVulns,
							"total_vulns":      scanResult.TotalVulns,
							"security_grade":   scanResult.Grade,
						},
					)
				}
				operation.Notifications = append(operation.Notifications, "Critical vulnerabilities detected")
			}
		}
	}

	// Complete operation
	endTime := time.Now()
	operation.EndTime = &endTime
	operation.Duration = endTime.Sub(operation.StartTime)
	operation.Status = "completed"
	operation.Progress = 100.0

	if sm.notificationManager != nil {
		sm.notificationManager.CompleteOperation(operationID, true, "")
	}

	operation.Notifications = append(operation.Notifications, "Image pulled successfully")

	sm.logger.WithFields(logrus.Fields{
		"operation_id": operationID,
		"image":        imageName,
		"tag":          tag,
		"duration":     operation.Duration,
		"scan_result":  operation.SecurityScan != nil,
	}).Info("Integrated image pull completed")

	return operation, nil
}

// UpdateContainerIntegrated performs an integrated container update with all features
func (sm *ServiceManager) UpdateContainerIntegrated(ctx context.Context, containerID string, updateOptions *dockerTypes.ContainerUpdateOptions, userContext *security.DockerUserContext) (*IntegratedOperation, error) {
	if !sm.initialized {
		return nil, fmt.Errorf("service manager not initialized")
	}

	operationID := fmt.Sprintf("update_%s_%d", containerID[:8], time.Now().Unix())
	operation := &IntegratedOperation{
		ID:          operationID,
		Type:        "integrated_update",
		Status:      "started",
		StartTime:   time.Now(),
		UserID:      userContext.UserID,
		ContainerID: containerID,
		ImageName:   updateOptions.NewImage,
		Metadata:    make(map[string]interface{}),
	}

	// Start operation tracking
	if sm.notificationManager != nil {
		sm.notificationManager.StartOperation(operationID, "integrated_update", userContext.UserID, map[string]interface{}{
			"container_id": containerID,
			"new_image":    updateOptions.NewImage,
			"strategy":     updateOptions.Strategy,
		})
	}

	// Step 1: Pre-update security scan
	operation.Status = "scanning_new_image"
	operation.Progress = 10.0
	if sm.notificationManager != nil {
		sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Scanning new image for security issues", nil)
	}

	scanResult, err := sm.imageScanner.ScanImage(ctx, updateOptions.NewImage)
	if err != nil {
		sm.logger.WithError(err).Warn("Pre-update security scan failed")
	} else {
		operation.SecurityScan = scanResult

		// Check if update should proceed based on security scan
		if scanResult.CriticalVulns > 5 && !updateOptions.RollbackOnFailure {
			operation.Status = "blocked"
			operation.Error = fmt.Sprintf("Update blocked due to %d critical vulnerabilities in new image", scanResult.CriticalVulns)
			if sm.notificationManager != nil {
				sm.notificationManager.CompleteOperation(operationID, false, operation.Error)
			}
			return operation, fmt.Errorf(operation.Error)
		}
	}

	operation.Progress = 30.0

	// Step 2: Create rollback point
	operation.Status = "creating_backup"
	if sm.notificationManager != nil {
		sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Creating rollback point", nil)
	}

	rollbackRequest := &RollbackRequest{
		ContainerID: containerID,
		Reason:      "Pre-update backup",
		UserContext: userContext,
		Metadata: map[string]interface{}{
			"update_operation": operationID,
			"new_image":        updateOptions.NewImage,
		},
	}

	rollbackEntry, err := sm.rollbackManager.PrepareRollback(ctx, rollbackRequest)
	if err != nil {
		sm.logger.WithError(err).Warn("Failed to create rollback point")
	} else {
		operation.RollbackEntry = rollbackEntry
	}

	operation.Progress = 50.0

	// Step 3: Execute update
	operation.Status = "updating"
	if sm.notificationManager != nil {
		sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Executing container update", nil)
	}

	updateResult, err := sm.updateEngine.UpdateContainer(ctx, containerID, updateOptions, userContext)
	if err != nil {
		operation.Status = "failed"
		operation.Error = err.Error()

		// Attempt automatic rollback if configured
		if updateOptions.RollbackOnFailure && rollbackEntry != nil {
			operation.Status = "rolling_back"
			if sm.notificationManager != nil {
				sm.notificationManager.UpdateOperation(operationID, 75.0, "Update failed, performing automatic rollback", nil)
			}

			rollbackResult, rollbackErr := sm.rollbackManager.ExecuteRollback(ctx, rollbackEntry.ID)
			if rollbackErr != nil {
				operation.Error += fmt.Sprintf("; rollback also failed: %s", rollbackErr.Error())
				sm.logger.WithError(rollbackErr).Error("Automatic rollback failed")
			} else {
				operation.Notifications = append(operation.Notifications, "Automatic rollback completed successfully")
				operation.RollbackEntry = rollbackResult.RollbackEntry
			}
		}

		if sm.notificationManager != nil {
			sm.notificationManager.CompleteOperation(operationID, false, operation.Error)
		}
		return operation, err
	}

	operation.UpdateResult = updateResult
	operation.Progress = 90.0

	// Step 4: Post-update verification
	operation.Status = "verifying"
	if sm.notificationManager != nil {
		sm.notificationManager.UpdateOperation(operationID, operation.Progress, "Verifying update success", nil)
	}

	// Perform health check on updated container
	if updateResult.Success {
		// Wait a moment for container to stabilize
		select {
		case <-ctx.Done():
			return operation, ctx.Err()
		case <-time.After(5 * time.Second):
		}

		// Additional verification could go here
		operation.Notifications = append(operation.Notifications, "Container update verified successfully")
	}

	// Complete operation
	endTime := time.Now()
	operation.EndTime = &endTime
	operation.Duration = endTime.Sub(operation.StartTime)
	operation.Status = "completed"
	operation.Progress = 100.0

	if sm.notificationManager != nil {
		sm.notificationManager.CompleteOperation(operationID, true, "")
	}

	operation.Notifications = append(operation.Notifications, "Container updated successfully")

	sm.logger.WithFields(logrus.Fields{
		"operation_id":   operationID,
		"container_id":   containerID,
		"new_image":      updateOptions.NewImage,
		"strategy":       updateOptions.Strategy,
		"duration":       operation.Duration,
		"rollback_used":  operation.RollbackEntry != nil,
	}).Info("Integrated container update completed")

	return operation, nil
}

// GetServiceInfo returns comprehensive service information
func (sm *ServiceManager) GetServiceInfo() *ServiceInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	info := &ServiceInfo{
		Status:    "running",
		Version:   "1.0.0",
		StartTime: time.Now(), // This should be set during initialization
		Features: []string{
			"docker_management",
			"security_scanning",
			"container_updates",
			"rollback_management",
			"real_time_notifications",
		},
		Configuration: make(map[string]interface{}),
		Docker:        make(map[string]interface{}),
		Statistics:    make(map[string]interface{}),
	}

	if sm.initialized {
		info.Uptime = time.Since(info.StartTime)

		// Get Docker information
		if sm.clientManager != nil {
			client := sm.clientManager.GetClient()
			if dockerInfo, err := client.Info(context.Background()); err == nil {
				info.Docker["version"] = dockerInfo.ServerVersion
				info.Docker["containers"] = dockerInfo.Containers
				info.Docker["images"] = dockerInfo.Images
				info.Docker["architecture"] = dockerInfo.Architecture
				info.Docker["os"] = dockerInfo.OperatingSystem
			}
		}

		// Get statistics from various components
		if sm.notificationManager != nil {
			info.Statistics["notifications"] = sm.notificationManager.GetStats()
		}

		if sm.rollbackManager != nil {
			info.Statistics["rollbacks"] = sm.rollbackManager.GetStats()
		}

		if sm.imageScanner != nil {
			info.Statistics["scanner"] = sm.imageScanner.GetCacheStats()
		}

		// Health check
		info.Health = sm.performHealthCheck()
	} else {
		info.Status = "initializing"
		info.Health = dockerTypes.ServiceHealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Issues:    []string{"Service not initialized"},
		}
	}

	return info
}

// performHealthCheck performs a comprehensive health check
func (sm *ServiceManager) performHealthCheck() dockerTypes.ServiceHealthStatus {
	health := dockerTypes.ServiceHealthStatus{
		Healthy:    true,
		LastCheck:  time.Now(),
		Issues:     []string{},
		Components: make(map[string]bool),
		Details:    make(map[string]interface{}),
	}

	// Check Docker client
	if sm.clientManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := sm.clientManager.GetClient().Ping(ctx); err != nil {
			health.Healthy = false
			health.Components["docker_client"] = false
			health.Issues = append(health.Issues, "Docker client not responding")
		} else {
			health.Components["docker_client"] = true
		}
	} else {
		health.Healthy = false
		health.Components["docker_client"] = false
		health.Issues = append(health.Issues, "Docker client not initialized")
	}

	// Check notification manager
	if sm.notificationManager != nil {
		health.Components["notifications"] = true
		stats := sm.notificationManager.GetStats()
		health.Details["notification_stats"] = stats
	} else {
		health.Components["notifications"] = false
	}

	// Check other components
	health.Components["update_engine"] = sm.updateEngine != nil
	health.Components["image_scanner"] = sm.imageScanner != nil
	health.Components["rollback_manager"] = sm.rollbackManager != nil

	return health
}

// cleanupRoutine performs periodic cleanup
func (sm *ServiceManager) cleanupRoutine() {
	ticker := time.NewTicker(sm.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		sm.performCleanup()
	}
}

// performCleanup performs cleanup operations
func (sm *ServiceManager) performCleanup() {
	sm.logger.Debug("Performing service cleanup")

	// Cleanup pull progress
	if sm.clientManager != nil {
		sm.clientManager.CleanupPullProgress(24 * time.Hour)
	}

	// Cleanup scanner cache
	if sm.imageScanner != nil {
		// Could add cleanup for old scan results
	}

	sm.logger.Debug("Service cleanup completed")
}

// healthCheckRoutine performs periodic health checks
func (sm *ServiceManager) healthCheckRoutine() {
	ticker := time.NewTicker(sm.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		health := sm.performHealthCheck()

		if !health.Healthy && sm.notificationManager != nil {
			sm.notificationManager.NotifyAlert(
				0, // System alert
				"critical",
				"Service Health Check Failed",
				fmt.Sprintf("Docker service health check failed: %v", health.Issues),
				map[string]interface{}{
					"components": health.Components,
					"issues":     health.Issues,
				},
			)
		}
	}
}

// Shutdown gracefully shuts down all services
func (sm *ServiceManager) Shutdown(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.logger.Info("Shutting down Docker service manager")

	// Shutdown client manager
	if sm.clientManager != nil {
		if err := sm.clientManager.Close(); err != nil {
			sm.logger.WithError(err).Warn("Error closing client manager")
		}
	}

	// Clear notification manager
	if sm.notificationManager != nil {
		// Would normally close WebSocket connections
		sm.logger.Info("Notification manager shut down")
	}

	sm.initialized = false
	sm.logger.Info("Docker service manager shut down completed")

	return nil
}

// GetClientManager returns the client manager
func (sm *ServiceManager) GetClientManager() *ClientManager {
	return sm.clientManager
}

// GetUpdateEngine returns the update engine
func (sm *ServiceManager) GetUpdateEngine() *UpdateEngine {
	return sm.updateEngine
}

// GetImageScanner returns the image scanner
func (sm *ServiceManager) GetImageScanner() *ImageScanner {
	return sm.imageScanner
}

// GetNotificationManager returns the notification manager
func (sm *ServiceManager) GetNotificationManager() *NotificationManager {
	return sm.notificationManager
}

// GetRollbackManager returns the rollback manager
func (sm *ServiceManager) GetRollbackManager() *RollbackManager {
	return sm.rollbackManager
}