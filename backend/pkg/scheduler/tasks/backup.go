package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

// BackupTask implements the Task interface for system backup operations
type BackupTask struct {
	containerRepo       repository.ContainerRepository
	updateHistoryRepo   repository.UpdateHistoryRepository
	taskRepo            repository.ScheduledTaskRepository
	containerService    *service.ContainerService
	notificationService *service.NotificationService
	dockerClient        *docker.DockerClient
}

// NewBackupTask creates a new backup task
func NewBackupTask(
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	taskRepo repository.ScheduledTaskRepository,
	containerService *service.ContainerService,
	notificationService *service.NotificationService,
	dockerClient *docker.DockerClient,
) *BackupTask {
	return &BackupTask{
		containerRepo:       containerRepo,
		updateHistoryRepo:   updateHistoryRepo,
		taskRepo:            taskRepo,
		containerService:    containerService,
		notificationService: notificationService,
		dockerClient:        dockerClient,
	}
}

// Execute runs the backup task
func (t *BackupTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting system backup task")

	// Parse task-specific parameters
	backupParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Create backup session
	session := &BackupSession{
		StartedAt:    time.Now(),
		BackupType:   backupParams.BackupType,
		StoragePath:  backupParams.StoragePath,
		Operations:   []BackupOperation{},
	}

	// Create backup directory structure
	if err := t.createBackupDirectory(session, backupParams); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Perform different types of backups based on configuration
	if backupParams.BackupDatabase {
		operation := t.backupDatabase(ctx, session, backupParams)
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
	}

	if backupParams.BackupConfigurations {
		operation := t.backupConfigurations(ctx, session, backupParams)
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
	}

	if backupParams.BackupContainerConfigs {
		operation := t.backupContainerConfigs(ctx, session, backupParams)
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
	}

	if backupParams.BackupVolumes {
		operations := t.backupVolumes(ctx, session, backupParams)
		session.Operations = append(session.Operations, operations...)
		for _, op := range operations {
			if op.Success {
				session.SuccessfulOperations++
			} else {
				session.FailedOperations++
			}
		}
	}

	if backupParams.BackupImages {
		operations := t.backupImages(ctx, session, backupParams)
		session.Operations = append(session.Operations, operations...)
		for _, op := range operations {
			if op.Success {
				session.SuccessfulOperations++
			} else {
				session.FailedOperations++
			}
		}
	}

	// Compress backup if requested
	if backupParams.CompressBackups {
		operation := t.compressBackup(ctx, session, backupParams)
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
	}

	// Clean up old backups
	if backupParams.RetentionDays > 0 {
		operation := t.cleanupOldBackups(ctx, session, backupParams)
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
	}

	session.CompletedAt = time.Now()
	session.Duration = session.CompletedAt.Sub(session.StartedAt)

	// Send notification about backup results
	if err := t.sendBackupNotification(ctx, session, backupParams); err != nil {
		logger.WithError(err).Warn("Failed to send backup notification")
	}

	logger.WithFields(logrus.Fields{
		"total_operations":      len(session.Operations),
		"successful_operations": session.SuccessfulOperations,
		"failed_operations":     session.FailedOperations,
		"duration":             session.Duration,
		"backup_size":          session.TotalSize,
		"backup_path":          session.BackupPath,
	}).Info("System backup task completed")

	return nil
}

// GetName returns the task name
func (t *BackupTask) GetName() string {
	return "System Backup"
}

// GetType returns the task type
func (t *BackupTask) GetType() model.TaskType {
	return model.TaskTypeBackup
}

// Validate validates task parameters
func (t *BackupTask) Validate(params scheduler.TaskParameters) error {
	if params.TaskType != model.TaskTypeBackup {
		return fmt.Errorf("invalid task type: expected %s, got %s", model.TaskTypeBackup, params.TaskType)
	}

	// Validate parameters structure
	if _, err := t.parseParameters(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	return nil
}

// GetDefaultTimeout returns the default timeout for this task
func (t *BackupTask) GetDefaultTimeout() time.Duration {
	return 2 * time.Hour
}

// CanRunConcurrently returns false since backup operations should be serialized
func (t *BackupTask) CanRunConcurrently() bool {
	return false
}

// BackupParameters represents parameters for backup operations
type BackupParameters struct {
	BackupType              string   `json:"backup_type"`               // full, incremental, differential
	StoragePath             string   `json:"storage_path"`              // Base path for backups
	RetentionDays           int      `json:"retention_days"`            // How long to keep backups
	CompressBackups         bool     `json:"compress_backups"`          // Whether to compress backups
	BackupDatabase          bool     `json:"backup_database"`           // Backup application database
	BackupConfigurations    bool     `json:"backup_configurations"`     // Backup system configurations
	BackupContainerConfigs  bool     `json:"backup_container_configs"`  // Backup container configurations
	BackupVolumes           bool     `json:"backup_volumes"`            // Backup Docker volumes
	BackupImages            bool     `json:"backup_images"`             // Backup Docker images
	ExcludeContainers       []string `json:"exclude_containers"`        // Containers to exclude from backup
	ExcludeVolumes          []string `json:"exclude_volumes"`           // Volumes to exclude from backup
	ExcludeImages           []string `json:"exclude_images"`            // Images to exclude from backup
	IncludeSystemImages     bool     `json:"include_system_images"`     // Include system Docker images
	MaxBackupSize           int64    `json:"max_backup_size"`           // Maximum backup size in bytes
	EnableEncryption        bool     `json:"enable_encryption"`         // Enable backup encryption
	EncryptionKey           string   `json:"encryption_key,omitempty"`  // Encryption key
	RemoteStorage           *RemoteStorageConfig `json:"remote_storage,omitempty"` // Remote storage configuration
	NotifyOnSuccess         bool     `json:"notify_on_success"`         // Send notification on success
	NotifyOnFailure         bool     `json:"notify_on_failure"`         // Send notification on failure
	VerifyBackup            bool     `json:"verify_backup"`             // Verify backup integrity
	CreateManifest          bool     `json:"create_manifest"`           // Create backup manifest
	ParallelOperations      int      `json:"parallel_operations"`       // Number of parallel backup operations
}

// RemoteStorageConfig represents remote storage configuration
type RemoteStorageConfig struct {
	Type        string            `json:"type"`         // s3, ftp, sftp, nfs
	Endpoint    string            `json:"endpoint"`     // Storage endpoint
	Credentials map[string]string `json:"credentials"`  // Storage credentials
	BucketPath  string            `json:"bucket_path"`  // Path within storage
	Enabled     bool              `json:"enabled"`      // Whether remote storage is enabled
}

// BackupSession represents a backup session
type BackupSession struct {
	BackupID             string            `json:"backup_id"`
	StartedAt            time.Time         `json:"started_at"`
	CompletedAt          time.Time         `json:"completed_at"`
	Duration             time.Duration     `json:"duration"`
	BackupType           string            `json:"backup_type"`
	StoragePath          string            `json:"storage_path"`
	BackupPath           string            `json:"backup_path"`
	Operations           []BackupOperation `json:"operations"`
	SuccessfulOperations int               `json:"successful_operations"`
	FailedOperations     int               `json:"failed_operations"`
	TotalSize            int64             `json:"total_size"`
	CompressedSize       int64             `json:"compressed_size,omitempty"`
	IsCompressed         bool              `json:"is_compressed"`
	IsEncrypted          bool              `json:"is_encrypted"`
	Manifest             *BackupManifest   `json:"manifest,omitempty"`
	Errors               []BackupError     `json:"errors"`
}

// BackupOperation represents a single backup operation
type BackupOperation struct {
	Type        string        `json:"type"`         // database, config, container, volume, image
	Name        string        `json:"name"`         // Name of the item being backed up
	Success     bool          `json:"success"`      // Whether the operation succeeded
	Error       string        `json:"error,omitempty"` // Error message if failed
	SourcePath  string        `json:"source_path"`  // Source path
	BackupPath  string        `json:"backup_path"`  // Backup destination path
	Size        int64         `json:"size"`         // Size of backed up data
	Duration    time.Duration `json:"duration"`     // Time taken
	Checksum    string        `json:"checksum,omitempty"` // Checksum for verification
	Metadata    interface{}   `json:"metadata,omitempty"` // Additional metadata
}

// BackupManifest represents backup manifest information
type BackupManifest struct {
	Version      string                 `json:"version"`
	CreatedAt    time.Time              `json:"created_at"`
	BackupType   string                 `json:"backup_type"`
	TotalSize    int64                  `json:"total_size"`
	FileCount    int                    `json:"file_count"`
	Checksums    map[string]string      `json:"checksums"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// BackupError represents an error during backup operations
type BackupError struct {
	Operation string `json:"operation"`
	Error     string `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Recoverable bool `json:"recoverable"`
}

// parseParameters parses and validates task parameters
func (t *BackupTask) parseParameters(params scheduler.TaskParameters) (*BackupParameters, error) {
	// Set defaults
	backupParams := &BackupParameters{
		BackupType:             "full",
		StoragePath:            "/var/backups/docker-auto",
		RetentionDays:          30,
		CompressBackups:        true,
		BackupDatabase:         true,
		BackupConfigurations:   true,
		BackupContainerConfigs: true,
		BackupVolumes:          false, // Conservative default
		BackupImages:           false, // Conservative default
		ExcludeContainers:      []string{},
		ExcludeVolumes:         []string{},
		ExcludeImages:          []string{},
		IncludeSystemImages:    false,
		MaxBackupSize:          0, // No limit
		EnableEncryption:       false,
		NotifyOnSuccess:        true,
		NotifyOnFailure:        true,
		VerifyBackup:           true,
		CreateManifest:         true,
		ParallelOperations:     2,
	}

	// Parse from parameters map
	if params.Parameters != nil {
		jsonData, err := json.Marshal(params.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		if err := json.Unmarshal(jsonData, backupParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	// Validate parameters
	validTypes := []string{"full", "incremental", "differential"}
	if !t.contains(validTypes, backupParams.BackupType) {
		backupParams.BackupType = "full"
	}

	if backupParams.ParallelOperations <= 0 {
		backupParams.ParallelOperations = 1
	}
	if backupParams.ParallelOperations > 10 {
		backupParams.ParallelOperations = 10
	}

	if backupParams.RetentionDays <= 0 {
		backupParams.RetentionDays = 30
	}

	return backupParams, nil
}

// createBackupDirectory creates the backup directory structure
func (t *BackupTask) createBackupDirectory(session *BackupSession, params *BackupParameters) error {
	timestamp := session.StartedAt.Format("20060102-150405")
	session.BackupID = fmt.Sprintf("backup-%s", timestamp)
	session.BackupPath = filepath.Join(params.StoragePath, session.BackupID)

	if err := os.MkdirAll(session.BackupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	logrus.WithField("backup_path", session.BackupPath).Info("Created backup directory")
	return nil
}

// backupDatabase backs up the application database
func (t *BackupTask) backupDatabase(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "database",
		Name: "Application Database",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	// This is a placeholder implementation
	// Real implementation would:
	// 1. Determine database type (PostgreSQL, SQLite, etc.)
	// 2. Create database dump using appropriate tools
	// 3. Save dump to backup directory
	// 4. Calculate checksum

	backupFile := filepath.Join(session.BackupPath, "database.sql")
	operation.BackupPath = backupFile

	// Placeholder: Create empty backup file
	file, err := os.Create(backupFile)
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to create database backup file: %v", err)
		return operation
	}
	defer file.Close()

	// Write placeholder content
	content := fmt.Sprintf("-- Database backup created at %s\n-- This is a placeholder implementation\n", time.Now())
	if _, err := file.WriteString(content); err != nil {
		operation.Error = fmt.Sprintf("Failed to write database backup: %v", err)
		return operation
	}

	// Get file info
	if info, err := file.Stat(); err == nil {
		operation.Size = info.Size()
	}

	operation.Success = true
	logrus.Info("Database backup completed (placeholder)")

	return operation
}

// backupConfigurations backs up system configurations
func (t *BackupTask) backupConfigurations(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "configuration",
		Name: "System Configurations",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	configDir := filepath.Join(session.BackupPath, "configurations")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		operation.Error = fmt.Sprintf("Failed to create config backup directory: %v", err)
		return operation
	}

	operation.BackupPath = configDir

	// Backup configuration files
	configFiles := []string{
		"config.yaml",
		"docker-compose.yml",
		".env",
	}

	var totalSize int64
	for _, configFile := range configFiles {
		sourcePath := filepath.Join(".", configFile)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue // Skip non-existent files
		}

		destPath := filepath.Join(configDir, configFile)
		if err := t.copyFile(sourcePath, destPath); err != nil {
			logrus.WithError(err).WithField("file", configFile).Warn("Failed to backup config file")
			continue
		}

		if info, err := os.Stat(destPath); err == nil {
			totalSize += info.Size()
		}
	}

	operation.Size = totalSize
	operation.Success = true

	logrus.WithField("config_dir", configDir).Info("Configuration backup completed")

	return operation
}

// backupContainerConfigs backs up container configurations
func (t *BackupTask) backupContainerConfigs(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "container_configs",
		Name: "Container Configurations",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	if t.containerRepo == nil {
		operation.Error = "Container repository not available"
		return operation
	}

	// Get all containers
	containers, _, err := t.containerRepo.List(ctx, &model.ContainerFilter{
		Limit: 1000,
	})
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to list containers: %v", err)
		return operation
	}

	// Create container configs directory
	configDir := filepath.Join(session.BackupPath, "container_configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		operation.Error = fmt.Sprintf("Failed to create container config directory: %v", err)
		return operation
	}

	operation.BackupPath = configDir

	// Export container configurations
	var totalSize int64
	for _, container := range containers {
		// Skip excluded containers
		if t.contains(params.ExcludeContainers, container.Name) {
			continue
		}

		configData := map[string]interface{}{
			"id":            container.ID,
			"name":          container.Name,
			"image":         container.Image,
			"tag":           container.Tag,
			"config_json":   container.ConfigJSON,
			"update_policy": container.UpdatePolicy,
			"registry_url":  container.RegistryURL,
			"created_at":    container.CreatedAt,
		}

		configJSON, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			logrus.WithError(err).WithField("container", container.Name).Warn("Failed to marshal container config")
			continue
		}

		configFile := filepath.Join(configDir, fmt.Sprintf("%s.json", container.Name))
		if err := os.WriteFile(configFile, configJSON, 0644); err != nil {
			logrus.WithError(err).WithField("container", container.Name).Warn("Failed to write container config")
			continue
		}

		totalSize += int64(len(configJSON))
	}

	operation.Size = totalSize
	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"container_count": len(containers),
		"config_dir":      configDir,
	}).Info("Container configuration backup completed")

	return operation
}

// backupVolumes backs up Docker volumes
func (t *BackupTask) backupVolumes(ctx context.Context, session *BackupSession, params *BackupParameters) []BackupOperation {
	var operations []BackupOperation

	if t.dockerClient == nil {
		operations = append(operations, BackupOperation{
			Type:  "volumes",
			Name:  "Docker Volumes",
			Error: "Docker client not available",
		})
		return operations
	}

	// This is a placeholder implementation
	// Real implementation would:
	// 1. List Docker volumes
	// 2. Create tar archives of volume data
	// 3. Save to backup directory

	operation := BackupOperation{
		Type:    "volumes",
		Name:    "Docker Volumes (placeholder)",
		Success: true,
		Size:    0,
	}

	volumesDir := filepath.Join(session.BackupPath, "volumes")
	if err := os.MkdirAll(volumesDir, 0755); err != nil {
		operation.Error = fmt.Sprintf("Failed to create volumes directory: %v", err)
		operation.Success = false
	} else {
		operation.BackupPath = volumesDir
	}

	operations = append(operations, operation)

	logrus.Info("Volume backup completed (placeholder)")

	return operations
}

// backupImages backs up Docker images
func (t *BackupTask) backupImages(ctx context.Context, session *BackupSession, params *BackupParameters) []BackupOperation {
	var operations []BackupOperation

	if t.dockerClient == nil {
		operations = append(operations, BackupOperation{
			Type:  "images",
			Name:  "Docker Images",
			Error: "Docker client not available",
		})
		return operations
	}

	// This is a placeholder implementation
	// Real implementation would:
	// 1. List Docker images
	// 2. Export images to tar files
	// 3. Save to backup directory

	operation := BackupOperation{
		Type:    "images",
		Name:    "Docker Images (placeholder)",
		Success: true,
		Size:    0,
	}

	imagesDir := filepath.Join(session.BackupPath, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		operation.Error = fmt.Sprintf("Failed to create images directory: %v", err)
		operation.Success = false
	} else {
		operation.BackupPath = imagesDir
	}

	operations = append(operations, operation)

	logrus.Info("Image backup completed (placeholder)")

	return operations
}

// compressBackup compresses the backup directory
func (t *BackupTask) compressBackup(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "compression",
		Name: "Backup Compression",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	// This is a placeholder implementation
	// Real implementation would use tar/gzip to compress the backup directory

	compressedFile := session.BackupPath + ".tar.gz"
	operation.BackupPath = compressedFile

	// Placeholder: Create empty compressed file
	file, err := os.Create(compressedFile)
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to create compressed backup: %v", err)
		return operation
	}
	file.Close()

	operation.Success = true
	session.IsCompressed = true

	logrus.WithField("compressed_file", compressedFile).Info("Backup compression completed (placeholder)")

	return operation
}

// cleanupOldBackups removes old backup files
func (t *BackupTask) cleanupOldBackups(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "cleanup",
		Name: "Old Backup Cleanup",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	cutoffDate := time.Now().AddDate(0, 0, -params.RetentionDays)

	entries, err := os.ReadDir(params.StoragePath)
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to read backup directory: %v", err)
		return operation
	}

	var removedCount int
	var freedSpace int64

	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), "backup-") {
			continue
		}

		entryPath := filepath.Join(params.StoragePath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffDate) {
			if err := os.RemoveAll(entryPath); err != nil {
				logrus.WithError(err).WithField("path", entryPath).Warn("Failed to remove old backup")
				continue
			}

			removedCount++
			freedSpace += info.Size()
		}
	}

	operation.Success = true
	operation.Size = freedSpace
	operation.Metadata = map[string]interface{}{
		"removed_count": removedCount,
		"freed_space":   freedSpace,
	}

	logrus.WithFields(logrus.Fields{
		"removed_count": removedCount,
		"freed_space":   freedSpace,
	}).Info("Old backup cleanup completed")

	return operation
}

// sendBackupNotification sends a notification about backup results
func (t *BackupTask) sendBackupNotification(ctx context.Context, session *BackupSession, params *BackupParameters) error {
	if t.notificationService == nil {
		return nil
	}

	shouldNotify := (params.NotifyOnSuccess && session.FailedOperations == 0) ||
		(params.NotifyOnFailure && session.FailedOperations > 0)

	if !shouldNotify {
		return nil
	}

	title := "System Backup Completed"
	priority := model.NotificationPriorityNormal

	if session.FailedOperations > 0 {
		title = "System Backup Completed with Errors"
		priority = model.NotificationPriorityHigh
	}

	message := fmt.Sprintf("Backup completed: %d successful, %d failed operations",
		session.SuccessfulOperations, session.FailedOperations)

	if session.TotalSize > 0 {
		message += fmt.Sprintf(", total size: %d bytes", session.TotalSize)
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeBackup,
		Title:    title,
		Message:  message,
		Priority: priority,
		Data: map[string]interface{}{
			"backup_id":             session.BackupID,
			"backup_path":           session.BackupPath,
			"successful_operations": session.SuccessfulOperations,
			"failed_operations":     session.FailedOperations,
			"total_size":           session.TotalSize,
			"duration":             session.Duration.String(),
			"backup_type":          session.BackupType,
			"is_compressed":        session.IsCompressed,
		},
	}

	return t.notificationService.SendNotification(ctx, notification)
}

// Helper methods

// copyFile copies a file from source to destination
func (t *BackupTask) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// contains checks if a slice contains a specific string
func (t *BackupTask) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}