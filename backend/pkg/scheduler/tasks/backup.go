package tasks

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"hash"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/scheduler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/volume"
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
	dbConnection        *sql.DB
	backupWorkers       chan struct{}
	mutex               sync.RWMutex
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
		backupWorkers:       make(chan struct{}, 10), // Max 10 concurrent workers
	}
}

// Execute runs the backup task
func (t *BackupTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting comprehensive system backup task")

	// Parse task-specific parameters
	backupParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Create backup session with unique ID
	session := &BackupSession{
		BackupID:     t.generateBackupID(),
		StartedAt:    time.Now(),
		BackupType:   backupParams.BackupType,
		StoragePath:  backupParams.StoragePath,
		Operations:   []BackupOperation{},
		Status:       "running",
		Metadata:     make(map[string]interface{}),
	}

	// Initialize backup workers
	for i := 0; i < backupParams.ParallelOperations; i++ {
		t.backupWorkers <- struct{}{}
	}

	// Create backup directory structure
	if err := t.createBackupDirectory(session, backupParams); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup manifest
	manifest := &BackupManifest{
		Version:    "2.0",
		CreatedAt:  session.StartedAt,
		BackupType: backupParams.BackupType,
		Checksums:  make(map[string]string),
		Metadata:   make(map[string]interface{}),
	}

	// Perform backup operations based on configuration
	var wg sync.WaitGroup
	operationChan := make(chan BackupOperation, 100)

	// Start operation processor
	go t.processOperations(ctx, session, operationChan, &wg)

	// Database backup
	if backupParams.BackupDatabase {
		wg.Add(1)
		go func() {
			defer wg.Done()
			operation := t.backupDatabase(ctx, session, backupParams)
			operationChan <- operation
			if operation.Success && operation.Checksum != "" {
				manifest.Checksums[operation.BackupPath] = operation.Checksum
			}
		}()
	}

	// Configuration backup
	if backupParams.BackupConfigurations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			operation := t.backupConfigurations(ctx, session, backupParams)
			operationChan <- operation
			if operation.Success && operation.Checksum != "" {
				manifest.Checksums[operation.BackupPath] = operation.Checksum
			}
		}()
	}

	// Container configurations backup
	if backupParams.BackupContainerConfigs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			operation := t.backupContainerConfigs(ctx, session, backupParams)
			operationChan <- operation
			if operation.Success && operation.Checksum != "" {
				manifest.Checksums[operation.BackupPath] = operation.Checksum
			}
		}()
	}

	// Docker volumes backup
	if backupParams.BackupVolumes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			operations := t.backupVolumes(ctx, session, backupParams)
			for _, operation := range operations {
				operationChan <- operation
				if operation.Success && operation.Checksum != "" {
					manifest.Checksums[operation.BackupPath] = operation.Checksum
				}
			}
		}()
	}

	// Docker images backup
	if backupParams.BackupImages {
		wg.Add(1)
		go func() {
			defer wg.Done()
			operations := t.backupImages(ctx, session, backupParams)
			for _, operation := range operations {
				operationChan <- operation
				if operation.Success && operation.Checksum != "" {
					manifest.Checksums[operation.BackupPath] = operation.Checksum
				}
			}
		}()
	}

	// Wait for all backup operations to complete
	wg.Wait()
	close(operationChan)

	// Finalize manifest
	manifest.TotalSize = session.TotalSize
	manifest.FileCount = len(session.Operations)
	session.Manifest = manifest

	// Save manifest
	if backupParams.CreateManifest {
		t.saveManifest(session, manifest)
	}

	// Compress backup if requested
	if backupParams.CompressBackups {
		operation := t.compressBackup(ctx, session, backupParams)
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
			session.CompressedSize = operation.Size
			session.IsCompressed = true
		} else {
			session.FailedOperations++
		}
		t.mutex.Unlock()
	}

	// Encrypt backup if requested
	if backupParams.EnableEncryption && backupParams.EncryptionKey != "" {
		operation := t.encryptBackup(ctx, session, backupParams)
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
			session.IsEncrypted = true
		} else {
			session.FailedOperations++
		}
		t.mutex.Unlock()
	}

	// Upload to remote storage if configured
	if backupParams.RemoteStorage != nil && backupParams.RemoteStorage.Enabled {
		operation := t.uploadToRemoteStorage(ctx, session, backupParams)
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
		t.mutex.Unlock()
	}

	// Verify backup integrity if requested
	if backupParams.VerifyBackup {
		operation := t.verifyBackup(ctx, session, backupParams)
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
		t.mutex.Unlock()
	}

	// Clean up old backups
	if backupParams.RetentionDays > 0 {
		operation := t.cleanupOldBackups(ctx, session, backupParams)
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
		}
		t.mutex.Unlock()
	}

	session.CompletedAt = time.Now()
	session.Duration = session.CompletedAt.Sub(session.StartedAt)
	session.Status = "completed"

	// Calculate compression ratio
	if session.IsCompressed && session.TotalSize > 0 {
		session.CompressionRatio = float64(session.CompressedSize) / float64(session.TotalSize)
	}

	// Send notification about backup results
	if err := t.sendBackupNotification(ctx, session, backupParams); err != nil {
		logger.WithError(err).Warn("Failed to send backup notification")
	}

	logger.WithFields(logrus.Fields{
		"total_operations":      len(session.Operations),
		"successful_operations": session.SuccessfulOperations,
		"failed_operations":     session.FailedOperations,
		"duration":              session.Duration,
		"backup_size":           session.TotalSize,
		"compressed_size":       session.CompressedSize,
		"compression_ratio":     session.CompressionRatio,
		"backup_path":           session.BackupPath,
		"is_compressed":         session.IsCompressed,
		"is_encrypted":          session.IsEncrypted,
	}).Info("Comprehensive system backup task completed")

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
	return 4 * time.Hour // Extended timeout for comprehensive backups
}

// CanRunConcurrently returns false since backup operations should be serialized
func (t *BackupTask) CanRunConcurrently() bool {
	return false
}

// BackupParameters represents parameters for backup operations
type BackupParameters struct {
	BackupType              string              `json:"backup_type"`               // full, incremental, differential
	StoragePath             string              `json:"storage_path"`              // Base path for backups
	RetentionDays           int                 `json:"retention_days"`            // How long to keep backups
	CompressBackups         bool                `json:"compress_backups"`          // Whether to compress backups
	CompressionLevel        int                 `json:"compression_level"`         // Compression level (1-9)
	BackupDatabase          bool                `json:"backup_database"`           // Backup application database
	BackupConfigurations    bool                `json:"backup_configurations"`     // Backup system configurations
	BackupContainerConfigs  bool                `json:"backup_container_configs"`  // Backup container configurations
	BackupVolumes           bool                `json:"backup_volumes"`            // Backup Docker volumes
	BackupImages            bool                `json:"backup_images"`             // Backup Docker images
	ExcludeContainers       []string            `json:"exclude_containers"`        // Containers to exclude from backup
	ExcludeVolumes          []string            `json:"exclude_volumes"`           // Volumes to exclude from backup
	ExcludeImages           []string            `json:"exclude_images"`            // Images to exclude from backup
	IncludeSystemImages     bool                `json:"include_system_images"`     // Include system Docker images
	MaxBackupSize           int64               `json:"max_backup_size"`           // Maximum backup size in bytes
	EnableEncryption        bool                `json:"enable_encryption"`         // Enable backup encryption
	EncryptionKey           string              `json:"encryption_key,omitempty"`  // Encryption key
	RemoteStorage           *RemoteStorageConfig `json:"remote_storage,omitempty"` // Remote storage configuration
	NotifyOnSuccess         bool                `json:"notify_on_success"`         // Send notification on success
	NotifyOnFailure         bool                `json:"notify_on_failure"`         // Send notification on failure
	VerifyBackup            bool                `json:"verify_backup"`             // Verify backup integrity
	CreateManifest          bool                `json:"create_manifest"`           // Create backup manifest
	ParallelOperations      int                 `json:"parallel_operations"`       // Number of parallel backup operations
	DatabaseType            string              `json:"database_type"`             // sqlite, postgres, mysql
	DatabaseConnection      string              `json:"database_connection"`       // Database connection string
	IncrementalBasePath     string              `json:"incremental_base_path"`     // Base path for incremental backups
	DifferentialBasePath    string              `json:"differential_base_path"`    // Base path for differential backups
	ChecksumAlgorithm       string              `json:"checksum_algorithm"`        // sha256, md5
}

// RemoteStorageConfig represents remote storage configuration
type RemoteStorageConfig struct {
	Type        string            `json:"type"`         // s3, ftp, sftp, nfs
	Endpoint    string            `json:"endpoint"`     // Storage endpoint
	Credentials map[string]string `json:"credentials"`  // Storage credentials
	BucketPath  string            `json:"bucket_path"`  // Path within storage
	Enabled     bool              `json:"enabled"`      // Whether remote storage is enabled
	Region      string            `json:"region"`       // AWS region for S3
	Bucket      string            `json:"bucket"`       // S3 bucket name
}

// BackupSession represents a backup session
type BackupSession struct {
	BackupID             string              `json:"backup_id"`
	StartedAt            time.Time           `json:"started_at"`
	CompletedAt          time.Time           `json:"completed_at"`
	Duration             time.Duration       `json:"duration"`
	BackupType           string              `json:"backup_type"`
	StoragePath          string              `json:"storage_path"`
	BackupPath           string              `json:"backup_path"`
	Operations           []BackupOperation   `json:"operations"`
	SuccessfulOperations int                 `json:"successful_operations"`
	FailedOperations     int                 `json:"failed_operations"`
	TotalSize            int64               `json:"total_size"`
	CompressedSize       int64               `json:"compressed_size,omitempty"`
	CompressionRatio     float64             `json:"compression_ratio,omitempty"`
	IsCompressed         bool                `json:"is_compressed"`
	IsEncrypted          bool                `json:"is_encrypted"`
	Manifest             *BackupManifest     `json:"manifest,omitempty"`
	Errors               []BackupError       `json:"errors"`
	Status               string              `json:"status"`
	Metadata             map[string]interface{} `json:"metadata"`
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
	RetryCount  int           `json:"retry_count"`        // Number of retries attempted
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
	Operation   string    `json:"operation"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// generateBackupID generates a unique backup ID
func (t *BackupTask) generateBackupID() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("backup-%s-%d", timestamp, time.Now().UnixNano()%10000)
}

// parseParameters parses and validates task parameters
func (t *BackupTask) parseParameters(params scheduler.TaskParameters) (*BackupParameters, error) {
	// Set production-grade defaults
	backupParams := &BackupParameters{
		BackupType:             "full",
		StoragePath:            "/var/backups/docker-auto",
		RetentionDays:          30,
		CompressBackups:        true,
		CompressionLevel:       6,
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
		ParallelOperations:     3,
		DatabaseType:           "sqlite",
		ChecksumAlgorithm:      "sha256",
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

	// Validate and sanitize parameters
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

	if backupParams.CompressionLevel < 1 {
		backupParams.CompressionLevel = 1
	}
	if backupParams.CompressionLevel > 9 {
		backupParams.CompressionLevel = 9
	}

	validDbTypes := []string{"sqlite", "postgres", "mysql"}
	if !t.contains(validDbTypes, backupParams.DatabaseType) {
		backupParams.DatabaseType = "sqlite"
	}

	validChecksums := []string{"sha256", "md5"}
	if !t.contains(validChecksums, backupParams.ChecksumAlgorithm) {
		backupParams.ChecksumAlgorithm = "sha256"
	}

	return backupParams, nil
}

// processOperations processes backup operations from the channel
func (t *BackupTask) processOperations(ctx context.Context, session *BackupSession, operationChan chan BackupOperation, wg *sync.WaitGroup) {
	for operation := range operationChan {
		t.mutex.Lock()
		session.Operations = append(session.Operations, operation)
		session.TotalSize += operation.Size
		if operation.Success {
			session.SuccessfulOperations++
		} else {
			session.FailedOperations++
			session.Errors = append(session.Errors, BackupError{
				Operation:   operation.Type,
				Error:       operation.Error,
				Timestamp:   time.Now(),
				Recoverable: operation.RetryCount < 3,
			})
		}
		t.mutex.Unlock()
	}
}

// createBackupDirectory creates the backup directory structure
func (t *BackupTask) createBackupDirectory(session *BackupSession, params *BackupParameters) error {
	session.BackupPath = filepath.Join(params.StoragePath, session.BackupID)

	if err := os.MkdirAll(session.BackupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create subdirectories for different backup types
	subdirs := []string{"database", "configurations", "containers", "volumes", "images", "manifests"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(session.BackupPath, subdir), 0755); err != nil {
			return fmt.Errorf("failed to create backup subdirectory %s: %w", subdir, err)
		}
	}

	logrus.WithField("backup_path", session.BackupPath).Info("Created comprehensive backup directory structure")
	return nil
}

// backupDatabase backs up the application database with real implementation
func (t *BackupTask) backupDatabase(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "database",
		Name: fmt.Sprintf("%s Database", strings.Title(params.DatabaseType)),
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	backupDir := filepath.Join(session.BackupPath, "database")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("database_%s.sql", session.StartedAt.Format("20060102_150405")))
	operation.BackupPath = backupFile

	var err error
	switch params.DatabaseType {
	case "postgres":
		err = t.backupPostgresDatabase(ctx, params, backupFile)
	case "mysql":
		err = t.backupMySQLDatabase(ctx, params, backupFile)
	case "sqlite":
		err = t.backupSQLiteDatabase(ctx, params, backupFile)
	default:
		operation.Error = fmt.Sprintf("Unsupported database type: %s", params.DatabaseType)
		return operation
	}

	if err != nil {
		operation.Error = fmt.Sprintf("Database backup failed: %v", err)
		return operation
	}

	// Calculate file size and checksum
	if info, err := os.Stat(backupFile); err == nil {
		operation.Size = info.Size()
	}

	if checksum, err := t.calculateChecksum(backupFile, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	// Verify backup by attempting to read it
	if t.verifyDatabaseBackup(backupFile, params.DatabaseType) {
		operation.Success = true
		logrus.WithFields(logrus.Fields{
			"db_type":     params.DatabaseType,
			"backup_file": backupFile,
			"size":        operation.Size,
			"checksum":    operation.Checksum,
		}).Info("Database backup completed successfully")
	} else {
		operation.Error = "Database backup verification failed"
	}

	return operation
}

// backupPostgresDatabase performs PostgreSQL database backup using pg_dump
func (t *BackupTask) backupPostgresDatabase(ctx context.Context, params *BackupParameters, backupFile string) error {
	var cmd *exec.Cmd

	if params.DatabaseConnection != "" {
		// Use connection string
		cmd = exec.CommandContext(ctx, "pg_dump", params.DatabaseConnection, "--file", backupFile, "--verbose")
	} else {
		// Use environment variables or defaults
		cmd = exec.CommandContext(ctx, "pg_dump", "--file", backupFile, "--verbose", "--no-password")
	}

	// Set environment variables for PostgreSQL
	cmd.Env = append(os.Environ(),
		"PGPASSWORD="+os.Getenv("DB_PASSWORD"),
		"PGHOST="+os.Getenv("DB_HOST"),
		"PGPORT="+os.Getenv("DB_PORT"),
		"PGUSER="+os.Getenv("DB_USER"),
		"PGDATABASE="+os.Getenv("DB_NAME"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w, output: %s", err, output)
	}

	return nil
}

// backupMySQLDatabase performs MySQL database backup using mysqldump
func (t *BackupTask) backupMySQLDatabase(ctx context.Context, params *BackupParameters, backupFile string) error {
	var cmd *exec.Cmd

	if params.DatabaseConnection != "" {
		// Parse connection string and build mysqldump command
		cmd = exec.CommandContext(ctx, "mysqldump", "--result-file", backupFile, "--single-transaction", "--routines", "--triggers")
	} else {
		// Use environment variables
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "root"
		}
		database := os.Getenv("DB_NAME")

		cmd = exec.CommandContext(ctx, "mysqldump",
			"--host", host,
			"--port", port,
			"--user", user,
			"--password="+os.Getenv("DB_PASSWORD"),
			"--result-file", backupFile,
			"--single-transaction",
			"--routines",
			"--triggers",
			database,
		)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mysqldump failed: %w, output: %s", err, output)
	}

	return nil
}

// backupSQLiteDatabase performs SQLite database backup
func (t *BackupTask) backupSQLiteDatabase(ctx context.Context, params *BackupParameters, backupFile string) error {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./app.db" // Default SQLite database path
	}

	// Check if source database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("SQLite database not found at %s", dbPath)
	}

	// Use SQLite .backup command for consistent backup
	cmd := exec.CommandContext(ctx, "sqlite3", dbPath, ".backup '"+backupFile+"'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to file copy if .backup fails
		logrus.WithError(err).Warn("SQLite .backup failed, falling back to file copy")
		return t.copyFile(dbPath, backupFile)
	}

	if len(output) > 0 {
		logrus.WithField("output", string(output)).Debug("SQLite backup output")
	}

	return nil
}

// verifyDatabaseBackup verifies that the database backup is valid
func (t *BackupTask) verifyDatabaseBackup(backupFile string, dbType string) bool {
	switch dbType {
	case "postgres":
		return t.verifyPostgresBackup(backupFile)
	case "mysql":
		return t.verifyMySQLBackup(backupFile)
	case "sqlite":
		return t.verifySQLiteBackup(backupFile)
	}
	return false
}

// verifyPostgresBackup verifies PostgreSQL backup file
func (t *BackupTask) verifyPostgresBackup(backupFile string) bool {
	file, err := os.Open(backupFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasContent := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "PostgreSQL database dump") ||
		   strings.Contains(line, "CREATE TABLE") ||
		   strings.Contains(line, "INSERT INTO") {
			hasContent = true
			break
		}
	}
	return hasContent && scanner.Err() == nil
}

// verifyMySQLBackup verifies MySQL backup file
func (t *BackupTask) verifyMySQLBackup(backupFile string) bool {
	file, err := os.Open(backupFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hasContent := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "MySQL dump") ||
		   strings.Contains(line, "CREATE TABLE") ||
		   strings.Contains(line, "INSERT INTO") {
			hasContent = true
			break
		}
	}
	return hasContent && scanner.Err() == nil
}

// verifySQLiteBackup verifies SQLite backup file
func (t *BackupTask) verifySQLiteBackup(backupFile string) bool {
	// Try to open the SQLite file and verify it's a valid database
	db, err := sql.Open("sqlite3", backupFile)
	if err != nil {
		return false
	}
	defer db.Close()

	// Try to query the sqlite_master table
	_, err = db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	return err == nil
}

// backupConfigurations backs up system configurations with real implementation
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
	operation.BackupPath = configDir

	// Configuration files to backup
	configFiles := []struct {
		source string
		dest   string
	}{
		{"config.yaml", "config.yaml"},
		{"docker-compose.yml", "docker-compose.yml"},
		{"docker-compose.yaml", "docker-compose.yaml"},
		{".env", ".env"},
		{".env.production", ".env.production"},
		{".env.development", ".env.development"},
		{"Dockerfile", "Dockerfile"},
		{"nginx.conf", "nginx.conf"},
		{"/etc/docker/daemon.json", "docker-daemon.json"},
	}

	var totalSize int64
	var backupCount int

	for _, config := range configFiles {
		sourcePath := config.source
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(".", sourcePath)
		}

		// Check if file exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		destPath := filepath.Join(configDir, config.dest)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			logrus.WithError(err).WithField("dir", filepath.Dir(destPath)).Warn("Failed to create config directory")
			continue
		}

		if err := t.copyFile(sourcePath, destPath); err != nil {
			logrus.WithError(err).WithField("file", config.source).Warn("Failed to backup config file")
			continue
		}

		if info, err := os.Stat(destPath); err == nil {
			totalSize += info.Size()
			backupCount++
		}
	}

	// Backup additional system files
	systemFiles := []struct {
		source string
		dest   string
	}{
		{"/etc/hosts", "system/hosts"},
		{"/etc/resolv.conf", "system/resolv.conf"},
		{"/proc/version", "system/version"},
		{"/proc/meminfo", "system/meminfo"},
		{"/proc/cpuinfo", "system/cpuinfo"},
	}

	for _, sysFile := range systemFiles {
		if _, err := os.Stat(sysFile.source); os.IsNotExist(err) {
			continue
		}

		destPath := filepath.Join(configDir, sysFile.dest)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		if err := t.copyFile(sysFile.source, destPath); err != nil {
			continue
		}

		if info, err := os.Stat(destPath); err == nil {
			totalSize += info.Size()
			backupCount++
		}
	}

	operation.Size = totalSize
	operation.Metadata = map[string]interface{}{
		"files_backed_up": backupCount,
		"total_files":     len(configFiles) + len(systemFiles),
	}

	// Calculate checksum for the entire configuration directory
	if checksum, err := t.calculateDirectoryChecksum(configDir, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	operation.Success = backupCount > 0

	logrus.WithFields(logrus.Fields{
		"config_dir":      configDir,
		"files_backed_up": backupCount,
		"total_size":      totalSize,
	}).Info("Configuration backup completed")

	return operation
}

// backupContainerConfigs backs up container configurations with real implementation
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

	// Get all containers from database
	containers, _, err := t.containerRepo.List(ctx, &model.ContainerFilter{
		Limit: 1000,
	})
	if err != nil {
		operation.Error = fmt.Sprintf("Failed to list containers: %v", err)
		return operation
	}

	// Create container configs directory
	configDir := filepath.Join(session.BackupPath, "containers")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		operation.Error = fmt.Sprintf("Failed to create container config directory: %v", err)
		return operation
	}

	operation.BackupPath = configDir

	// Export container configurations
	var totalSize int64
	var backupCount int

	for _, container := range containers {
		// Skip excluded containers
		if t.contains(params.ExcludeContainers, container.Name) {
			continue
		}

		// Get live container info from Docker if client is available
		var liveContainerInfo map[string]interface{}
		if t.dockerClient != nil {
			if containerInfo, err := t.dockerClient.GetContainer(ctx, container.ContainerID); err == nil {
				liveContainerInfo = map[string]interface{}{
					"config":          containerInfo.Config,
					"host_config":     containerInfo.HostConfig,
					"network_settings": containerInfo.NetworkSettings,
					"mounts":          containerInfo.Mounts,
					"state":           containerInfo.State,
				}
			}
		}

		configData := map[string]interface{}{
			"database_info": map[string]interface{}{
				"id":            container.ID,
				"name":          container.Name,
				"image":         container.Image,
				"tag":           container.Tag,
				"config_json":   container.ConfigJSON,
				"update_policy": container.UpdatePolicy,
				"registry_url":  container.RegistryURL,
				"created_at":    container.CreatedAt,
				"updated_at":    container.UpdatedAt,
			},
			"live_container_info": liveContainerInfo,
			"backup_timestamp":    time.Now(),
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
		backupCount++
	}

	// Create summary file
	summary := map[string]interface{}{
		"total_containers":   len(containers),
		"backed_up":         backupCount,
		"excluded":          len(params.ExcludeContainers),
		"backup_timestamp":  time.Now(),
		"excluded_containers": params.ExcludeContainers,
	}

	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	summaryFile := filepath.Join(configDir, "_summary.json")
	os.WriteFile(summaryFile, summaryJSON, 0644)
	totalSize += int64(len(summaryJSON))

	operation.Size = totalSize
	operation.Metadata = map[string]interface{}{
		"total_containers": len(containers),
		"backed_up":       backupCount,
		"excluded":        len(params.ExcludeContainers),
	}

	// Calculate checksum for the entire container configs directory
	if checksum, err := t.calculateDirectoryChecksum(configDir, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	operation.Success = backupCount > 0

	logrus.WithFields(logrus.Fields{
		"container_count": len(containers),
		"backed_up":      backupCount,
		"config_dir":     configDir,
		"total_size":     totalSize,
	}).Info("Container configuration backup completed")

	return operation
}

// backupVolumes backs up Docker volumes with real implementation
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

	// List Docker volumes
	volumeList, err := t.dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		operations = append(operations, BackupOperation{
			Type:  "volumes",
			Name:  "Docker Volumes",
			Error: fmt.Sprintf("Failed to list volumes: %v", err),
		})
		return operations
	}

	volumesDir := filepath.Join(session.BackupPath, "volumes")
	if err := os.MkdirAll(volumesDir, 0755); err != nil {
		operations = append(operations, BackupOperation{
			Type:  "volumes",
			Name:  "Docker Volumes",
			Error: fmt.Sprintf("Failed to create volumes directory: %v", err),
		})
		return operations
	}

	// Create worker pool for parallel volume backups
	semaphore := make(chan struct{}, params.ParallelOperations)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, vol := range volumeList.Volumes {
		// Skip excluded volumes
		if t.contains(params.ExcludeVolumes, vol.Name) {
			continue
		}

		wg.Add(1)
		go func(volume *volume.Volume) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			operation := t.backupSingleVolume(ctx, volume, volumesDir, params)

			mutex.Lock()
			operations = append(operations, operation)
			mutex.Unlock()
		}(vol)
	}

	wg.Wait()

	logrus.WithFields(logrus.Fields{
		"total_volumes":   len(volumeList.Volumes),
		"backup_operations": len(operations),
		"volumes_dir":     volumesDir,
	}).Info("Docker volumes backup completed")

	return operations
}

// backupSingleVolume backs up a single Docker volume
func (t *BackupTask) backupSingleVolume(ctx context.Context, vol *volume.Volume, volumesDir string, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "volume",
		Name: vol.Name,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	// Create tar archive for the volume
	volumeBackupFile := filepath.Join(volumesDir, fmt.Sprintf("%s.tar", vol.Name))
	operation.BackupPath = volumeBackupFile

	// Create a temporary container to access the volume
	tempContainerName := fmt.Sprintf("backup-volume-%s-%d", vol.Name, time.Now().UnixNano())

	// Use alpine image to create temporary container
	pullCmd := exec.CommandContext(ctx, "docker", "pull", "alpine:latest")
	if err := pullCmd.Run(); err != nil {
		operation.Error = fmt.Sprintf("Failed to pull alpine image: %v", err)
		return operation
	}

	// Create container with volume mounted
	createCmd := exec.CommandContext(ctx, "docker", "create",
		"--name", tempContainerName,
		"-v", vol.Name+":/backup-source:ro",
		"alpine:latest",
		"tar", "-czf", "/backup.tar.gz", "-C", "/backup-source", ".")

	if err := createCmd.Run(); err != nil {
		operation.Error = fmt.Sprintf("Failed to create backup container: %v", err)
		return operation
	}

	// Ensure cleanup of temporary container
	defer func() {
		removeCmd := exec.CommandContext(context.Background(), "docker", "rm", "-f", tempContainerName)
		removeCmd.Run()
	}()

	// Start the container to create the tar file
	startCmd := exec.CommandContext(ctx, "docker", "start", "-a", tempContainerName)
	if err := startCmd.Run(); err != nil {
		operation.Error = fmt.Sprintf("Failed to run backup container: %v", err)
		return operation
	}

	// Copy the tar file from the container
	copyCmd := exec.CommandContext(ctx, "docker", "cp", tempContainerName+":/backup.tar.gz", volumeBackupFile+".gz")
	if err := copyCmd.Run(); err != nil {
		operation.Error = fmt.Sprintf("Failed to copy volume backup: %v", err)
		return operation
	}

	// Get file size
	if info, err := os.Stat(volumeBackupFile + ".gz"); err == nil {
		operation.Size = info.Size()

		// Rename to final name
		os.Rename(volumeBackupFile+".gz", volumeBackupFile)
		operation.BackupPath = volumeBackupFile
	} else {
		operation.Error = fmt.Sprintf("Failed to get backup file info: %v", err)
		return operation
	}

	// Calculate checksum
	if checksum, err := t.calculateChecksum(volumeBackupFile, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	operation.Success = true
	operation.Metadata = map[string]interface{}{
		"volume_driver":     vol.Driver,
		"volume_mountpoint": vol.Mountpoint,
		"volume_labels":     vol.Labels,
		"volume_options":    vol.Options,
	}

	logrus.WithFields(logrus.Fields{
		"volume":      vol.Name,
		"backup_file": volumeBackupFile,
		"size":        operation.Size,
		"checksum":    operation.Checksum,
	}).Info("Volume backup completed")

	return operation
}

// backupImages backs up Docker images with real implementation
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

	// List Docker images
	images, err := t.dockerClient.ListImages(ctx, types.ImageListOptions{
		All: false, // Only show tagged images by default
	})
	if err != nil {
		operations = append(operations, BackupOperation{
			Type:  "images",
			Name:  "Docker Images",
			Error: fmt.Sprintf("Failed to list images: %v", err),
		})
		return operations
	}

	imagesDir := filepath.Join(session.BackupPath, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		operations = append(operations, BackupOperation{
			Type:  "images",
			Name:  "Docker Images",
			Error: fmt.Sprintf("Failed to create images directory: %v", err),
		})
		return operations
	}

	// Create worker pool for parallel image backups
	semaphore := make(chan struct{}, params.ParallelOperations)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, img := range images {
		// Skip images without repository tags (dangling images)
		if len(img.RepoTags) == 0 {
			continue
		}

		// Skip excluded images
		shouldSkip := false
		for _, tag := range img.RepoTags {
			if t.contains(params.ExcludeImages, tag) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Skip system images unless explicitly included
		if !params.IncludeSystemImages {
			shouldSkip = false
			for _, tag := range img.RepoTags {
				if strings.HasPrefix(tag, "docker.io/library/") ||
				   strings.HasPrefix(tag, "registry.k8s.io/") ||
				   strings.Contains(tag, "pause") ||
				   strings.Contains(tag, "system") {
					shouldSkip = true
					break
				}
			}
			if shouldSkip {
				continue
			}
		}

		for _, tag := range img.RepoTags {
			wg.Add(1)
			go func(imageTag string, imageID string) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				operation := t.backupSingleImage(ctx, imageTag, imageID, imagesDir, params)

				mutex.Lock()
				operations = append(operations, operation)
				mutex.Unlock()
			}(tag, img.ID)
		}
	}

	wg.Wait()

	logrus.WithFields(logrus.Fields{
		"total_images":     len(images),
		"backup_operations": len(operations),
		"images_dir":       imagesDir,
	}).Info("Docker images backup completed")

	return operations
}

// backupSingleImage backs up a single Docker image
func (t *BackupTask) backupSingleImage(ctx context.Context, imageTag string, imageID string, imagesDir string, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "image",
		Name: imageTag,
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	// Sanitize image tag for filename
	safeTag := strings.ReplaceAll(imageTag, "/", "_")
	safeTag = strings.ReplaceAll(safeTag, ":", "_")

	imageBackupFile := filepath.Join(imagesDir, fmt.Sprintf("%s.tar", safeTag))
	operation.BackupPath = imageBackupFile

	// Use docker save to export the image
	saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", imageBackupFile, imageTag)
	if err := saveCmd.Run(); err != nil {
		operation.Error = fmt.Sprintf("Failed to save image %s: %v", imageTag, err)
		return operation
	}

	// Get file size
	if info, err := os.Stat(imageBackupFile); err == nil {
		operation.Size = info.Size()
	} else {
		operation.Error = fmt.Sprintf("Failed to get backup file info: %v", err)
		return operation
	}

	// Calculate checksum
	if checksum, err := t.calculateChecksum(imageBackupFile, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	// Compress if requested
	if params.CompressBackups {
		compressedFile := imageBackupFile + ".gz"
		if err := t.compressFile(imageBackupFile, compressedFile, params.CompressionLevel); err == nil {
			// Remove original file and update operation
			os.Remove(imageBackupFile)
			operation.BackupPath = compressedFile

			if info, err := os.Stat(compressedFile); err == nil {
				operation.Size = info.Size()
			}

			// Recalculate checksum for compressed file
			if checksum, err := t.calculateChecksum(compressedFile, params.ChecksumAlgorithm); err == nil {
				operation.Checksum = checksum
			}
		}
	}

	operation.Success = true
	operation.Metadata = map[string]interface{}{
		"image_id":  imageID,
		"image_tag": imageTag,
	}

	logrus.WithFields(logrus.Fields{
		"image":       imageTag,
		"backup_file": operation.BackupPath,
		"size":        operation.Size,
		"checksum":    operation.Checksum,
	}).Info("Image backup completed")

	return operation
}

// compressBackup compresses the backup directory with real implementation
func (t *BackupTask) compressBackup(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "compression",
		Name: "Backup Compression",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	compressedFile := session.BackupPath + ".tar.gz"
	operation.BackupPath = compressedFile

	// Create compressed archive
	if err := t.createTarGzArchive(session.BackupPath, compressedFile, params.CompressionLevel); err != nil {
		operation.Error = fmt.Sprintf("Failed to create compressed backup: %v", err)
		return operation
	}

	// Get compressed file size
	if info, err := os.Stat(compressedFile); err == nil {
		operation.Size = info.Size()
		session.CompressedSize = operation.Size
	} else {
		operation.Error = fmt.Sprintf("Failed to get compressed file info: %v", err)
		return operation
	}

	// Calculate checksum
	if checksum, err := t.calculateChecksum(compressedFile, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	// Remove original directory if compression successful
	if operation.Size > 0 {
		if err := os.RemoveAll(session.BackupPath); err != nil {
			logrus.WithError(err).Warn("Failed to remove original backup directory after compression")
		}
		session.BackupPath = compressedFile
	}

	operation.Success = true
	session.IsCompressed = true

	logrus.WithFields(logrus.Fields{
		"compressed_file":   compressedFile,
		"size":             operation.Size,
		"compression_level": params.CompressionLevel,
		"checksum":         operation.Checksum,
	}).Info("Backup compression completed")

	return operation
}

// encryptBackup encrypts the backup with real implementation
func (t *BackupTask) encryptBackup(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "encryption",
		Name: "Backup Encryption",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	sourceFile := session.BackupPath
	encryptedFile := sourceFile + ".enc"
	operation.BackupPath = encryptedFile

	// Encrypt the backup file
	if err := t.encryptFile(sourceFile, encryptedFile, params.EncryptionKey); err != nil {
		operation.Error = fmt.Sprintf("Failed to encrypt backup: %v", err)
		return operation
	}

	// Get encrypted file size
	if info, err := os.Stat(encryptedFile); err == nil {
		operation.Size = info.Size()
	} else {
		operation.Error = fmt.Sprintf("Failed to get encrypted file info: %v", err)
		return operation
	}

	// Calculate checksum
	if checksum, err := t.calculateChecksum(encryptedFile, params.ChecksumAlgorithm); err == nil {
		operation.Checksum = checksum
	}

	// Remove original file if encryption successful
	if operation.Size > 0 {
		if err := os.Remove(sourceFile); err != nil {
			logrus.WithError(err).Warn("Failed to remove original file after encryption")
		}
		session.BackupPath = encryptedFile
	}

	operation.Success = true
	session.IsEncrypted = true

	logrus.WithFields(logrus.Fields{
		"encrypted_file": encryptedFile,
		"size":          operation.Size,
		"checksum":      operation.Checksum,
	}).Info("Backup encryption completed")

	return operation
}

// uploadToRemoteStorage uploads backup to remote storage with real implementation
func (t *BackupTask) uploadToRemoteStorage(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "remote_upload",
		Name: "Remote Storage Upload",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	switch params.RemoteStorage.Type {
	case "s3":
		err := t.uploadToS3(ctx, session.BackupPath, params.RemoteStorage)
		if err != nil {
			operation.Error = fmt.Sprintf("S3 upload failed: %v", err)
			return operation
		}
	default:
		operation.Error = fmt.Sprintf("Unsupported remote storage type: %s", params.RemoteStorage.Type)
		return operation
	}

	// Get backup file size for upload verification
	if info, err := os.Stat(session.BackupPath); err == nil {
		operation.Size = info.Size()
	}

	operation.Success = true
	operation.BackupPath = fmt.Sprintf("%s://%s/%s",
		params.RemoteStorage.Type,
		params.RemoteStorage.Bucket,
		filepath.Base(session.BackupPath))

	logrus.WithFields(logrus.Fields{
		"storage_type": params.RemoteStorage.Type,
		"remote_path":  operation.BackupPath,
		"size":         operation.Size,
	}).Info("Remote storage upload completed")

	return operation
}

// uploadToS3 uploads file to AWS S3
func (t *BackupTask) uploadToS3(ctx context.Context, filePath string, config *RemoteStorageConfig) error {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(
			config.Credentials["access_key"],
			config.Credentials["secret_key"],
			"",
		),
		Endpoint: aws.String(config.Endpoint),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 service client
	svc := s3.New(sess)

	// Open file for upload
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Prepare S3 key
	key := filepath.Join(config.BucketPath, filepath.Base(filePath))

	// Upload file
	_, err = svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// verifyBackup verifies backup integrity with real implementation
func (t *BackupTask) verifyBackup(ctx context.Context, session *BackupSession, params *BackupParameters) BackupOperation {
	operation := BackupOperation{
		Type: "verification",
		Name: "Backup Verification",
	}

	startTime := time.Now()
	defer func() {
		operation.Duration = time.Since(startTime)
	}()

	verificationErrors := []string{}
	verifiedFiles := 0

	// Verify backup file exists and is readable
	if _, err := os.Stat(session.BackupPath); os.IsNotExist(err) {
		operation.Error = fmt.Sprintf("Backup file does not exist: %s", session.BackupPath)
		return operation
	}

	// Verify file integrity using checksums from manifest
	if session.Manifest != nil && len(session.Manifest.Checksums) > 0 {
		for filePath, expectedChecksum := range session.Manifest.Checksums {
			if actualChecksum, err := t.calculateChecksum(filePath, params.ChecksumAlgorithm); err == nil {
				if actualChecksum == expectedChecksum {
					verifiedFiles++
				} else {
					verificationErrors = append(verificationErrors,
						fmt.Sprintf("Checksum mismatch for %s: expected %s, got %s",
							filePath, expectedChecksum, actualChecksum))
				}
			} else {
				verificationErrors = append(verificationErrors,
					fmt.Sprintf("Failed to calculate checksum for %s: %v", filePath, err))
			}
		}
	}

	// Additional verification for compressed backups
	if session.IsCompressed {
		if err := t.verifyTarGzArchive(session.BackupPath); err != nil {
			verificationErrors = append(verificationErrors,
				fmt.Sprintf("Compressed archive verification failed: %v", err))
		} else {
			verifiedFiles++
		}
	}

	// Additional verification for encrypted backups
	if session.IsEncrypted {
		if err := t.verifyEncryptedFile(session.BackupPath); err != nil {
			verificationErrors = append(verificationErrors,
				fmt.Sprintf("Encrypted file verification failed: %v", err))
		} else {
			verifiedFiles++
		}
	}

	operation.Metadata = map[string]interface{}{
		"verified_files":       verifiedFiles,
		"verification_errors":  verificationErrors,
		"total_files_checked":  len(session.Manifest.Checksums),
	}

	if len(verificationErrors) > 0 {
		operation.Error = fmt.Sprintf("Verification failed with %d errors: %v",
			len(verificationErrors), strings.Join(verificationErrors, "; "))
		return operation
	}

	operation.Success = true

	logrus.WithFields(logrus.Fields{
		"verified_files":      verifiedFiles,
		"total_files_checked": len(session.Manifest.Checksums),
	}).Info("Backup verification completed successfully")

	return operation
}

// cleanupOldBackups removes old backup files with real implementation
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
	var errors []string

	// Sort entries by modification time to keep newest files
	type entryInfo struct {
		entry os.DirEntry
		info  os.FileInfo
	}

	var entryInfos []entryInfo
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil {
			entryInfos = append(entryInfos, entryInfo{entry: entry, info: info})
		}
	}

	// Sort by modification time (oldest first)
	sort.Slice(entryInfos, func(i, j int) bool {
		return entryInfos[i].info.ModTime().Before(entryInfos[j].info.ModTime())
	})

	for _, ei := range entryInfos {
		entry := ei.entry
		info := ei.info

		// Only remove backup directories/files
		if !strings.HasPrefix(entry.Name(), "backup-") {
			continue
		}

		entryPath := filepath.Join(params.StoragePath, entry.Name())

		// Skip current backup
		if entryPath == session.BackupPath ||
		   strings.Contains(session.BackupPath, entry.Name()) {
			continue
		}

		if info.ModTime().Before(cutoffDate) {
			// Calculate size before removal
			size := t.getDirectorySize(entryPath)

			if err := os.RemoveAll(entryPath); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to remove %s: %v", entryPath, err))
				logrus.WithError(err).WithField("path", entryPath).Warn("Failed to remove old backup")
				continue
			}

			removedCount++
			freedSpace += size

			logrus.WithFields(logrus.Fields{
				"path":         entryPath,
				"mod_time":     info.ModTime(),
				"size":         size,
			}).Info("Removed old backup")
		}
	}

	operation.Success = true
	operation.Size = freedSpace
	operation.Metadata = map[string]interface{}{
		"removed_count": removedCount,
		"freed_space":   freedSpace,
		"retention_days": params.RetentionDays,
		"cutoff_date":   cutoffDate,
		"errors":        errors,
	}

	logrus.WithFields(logrus.Fields{
		"removed_count": removedCount,
		"freed_space":   freedSpace,
		"retention_days": params.RetentionDays,
		"errors":        len(errors),
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

	title := "System Backup Completed Successfully"
	priority := model.NotificationPriorityNormal

	if session.FailedOperations > 0 {
		title = "System Backup Completed with Errors"
		priority = model.NotificationPriorityHigh
	}

	message := fmt.Sprintf("Backup completed: %d successful, %d failed operations\n",
		session.SuccessfulOperations, session.FailedOperations)

	if session.TotalSize > 0 {
		message += fmt.Sprintf("Total size: %s", t.formatBytes(session.TotalSize))
		if session.IsCompressed {
			message += fmt.Sprintf(", compressed: %s (ratio: %.2f)",
				t.formatBytes(session.CompressedSize), session.CompressionRatio)
		}
		message += "\n"
	}

	message += fmt.Sprintf("Duration: %s\n", session.Duration.Round(time.Second))
	message += fmt.Sprintf("Backup ID: %s", session.BackupID)

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
			"compressed_size":      session.CompressedSize,
			"compression_ratio":    session.CompressionRatio,
			"duration":             session.Duration.String(),
			"backup_type":          session.BackupType,
			"is_compressed":        session.IsCompressed,
			"is_encrypted":         session.IsEncrypted,
			"status":               session.Status,
		},
	}

	return t.notificationService.SendNotification(ctx, notification)
}

// saveManifest saves the backup manifest
func (t *BackupTask) saveManifest(session *BackupSession, manifest *BackupManifest) error {
	manifestPath := filepath.Join(session.BackupPath, "manifests", "backup-manifest.json")

	// Create manifests directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifests directory: %w", err)
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestJSON, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	logrus.WithField("manifest_path", manifestPath).Info("Backup manifest saved")
	return nil
}

// Helper methods with real implementations

// copyFile copies a file from source to destination with real implementation
func (t *BackupTask) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Copy file permissions
	if sourceInfo, err := sourceFile.Stat(); err == nil {
		if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
			logrus.WithError(err).Warn("Failed to copy file permissions")
		}
	}

	return nil
}

// calculateChecksum calculates file checksum
func (t *BackupTask) calculateChecksum(filePath string, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var hash hash.Hash
	switch algorithm {
	case "sha256":
		hash = sha256.New()
	default:
		return "", fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// calculateDirectoryChecksum calculates checksum for entire directory
func (t *BackupTask) calculateDirectoryChecksum(dirPath string, algorithm string) (string, error) {
	var hash hash.Hash
	switch algorithm {
	case "sha256":
		hash = sha256.New()
	default:
		return "", fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(hash, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// createTarGzArchive creates a compressed tar.gz archive
func (t *BackupTask) createTarGzArchive(sourceDir, targetFile string, compressionLevel int) error {
	file, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer file.Close()

	// Create gzip writer with specified compression level
	gzipWriter, err := gzip.NewWriterLevel(file, compressionLevel)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Update header name to be relative to source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write the content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}

// compressFile compresses a single file
func (t *BackupTask) compressFile(sourceFile, targetFile string, compressionLevel int) error {
	source, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	target, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	gzipWriter, err := gzip.NewWriterLevel(target, compressionLevel)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, source)
	return err
}

// encryptFile encrypts a file using AES-256-GCM
func (t *BackupTask) encryptFile(sourceFile, targetFile, key string) error {
	// Create AES cipher
	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Open source file
	source, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Create target file
	target, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer target.Close()

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce to target file
	if _, err := target.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}

	// Read and encrypt source file in chunks
	buffer := make([]byte, 64*1024) // 64KB chunks
	for {
		n, err := source.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		if n == 0 {
			break
		}

		// Encrypt chunk
		encrypted := gcm.Seal(nil, nonce, buffer[:n], nil)

		// Write chunk size and encrypted data
		sizeBytes := make([]byte, 4)
		sizeBytes[0] = byte(len(encrypted) >> 24)
		sizeBytes[1] = byte(len(encrypted) >> 16)
		sizeBytes[2] = byte(len(encrypted) >> 8)
		sizeBytes[3] = byte(len(encrypted))

		if _, err := target.Write(sizeBytes); err != nil {
			return fmt.Errorf("failed to write chunk size: %w", err)
		}

		if _, err := target.Write(encrypted); err != nil {
			return fmt.Errorf("failed to write encrypted data: %w", err)
		}

		// Generate new nonce for next chunk
		if _, err := rand.Read(nonce); err != nil {
			return fmt.Errorf("failed to generate new nonce: %w", err)
		}
	}

	return nil
}

// verifyTarGzArchive verifies a tar.gz archive
func (t *BackupTask) verifyTarGzArchive(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Read through all entries to verify archive integrity
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("archive verification failed: %w", err)
		}

		// Try to read file content to verify it's not corrupted
		if header.Typeflag == tar.TypeReg {
			buffer := make([]byte, 1024)
			for {
				_, err := tarReader.Read(buffer)
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("failed to read file %s from archive: %w", header.Name, err)
				}
			}
		}
	}

	return nil
}

// verifyEncryptedFile verifies an encrypted file can be read
func (t *BackupTask) verifyEncryptedFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer file.Close()

	// Read nonce
	nonce := make([]byte, 12) // GCM nonce size
	if _, err := file.Read(nonce); err != nil {
		return fmt.Errorf("failed to read nonce: %w", err)
	}

	// Try to read at least one chunk to verify file structure
	sizeBytes := make([]byte, 4)
	if _, err := file.Read(sizeBytes); err != nil {
		return fmt.Errorf("failed to read chunk size: %w", err)
	}

	chunkSize := int(sizeBytes[0])<<24 | int(sizeBytes[1])<<16 | int(sizeBytes[2])<<8 | int(sizeBytes[3])
	if chunkSize <= 0 || chunkSize > 1024*1024 { // Sanity check
		return fmt.Errorf("invalid chunk size in encrypted file: %d", chunkSize)
	}

	return nil
}

// getDirectorySize calculates the total size of a directory
func (t *BackupTask) getDirectorySize(dirPath string) int64 {
	var size int64

	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size
}

// formatBytes formats byte count as human readable string
func (t *BackupTask) formatBytes(bytes int64) string {
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

// contains checks if a slice contains a specific string
func (t *BackupTask) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}