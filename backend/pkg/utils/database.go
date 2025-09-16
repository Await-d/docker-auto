package utils

import (
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBManager manages database connections and operations
type DBManager struct {
	db     *gorm.DB
	config *config.Config
}

// NewDBManager creates a new database manager
func NewDBManager(cfg *config.Config) *DBManager {
	return &DBManager{
		config: cfg,
	}
}

// InitDB initializes database connection with configuration
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	manager := NewDBManager(cfg)
	return manager.Connect()
}

// Connect establishes database connection
func (dm *DBManager) Connect() (*gorm.DB, error) {
	var dialector gorm.Dialector

	// Determine database type based on configuration
	if dm.config.Database.Host == "sqlite" || dm.config.Database.Name == ":memory:" {
		// SQLite for testing or development
		dialector = sqlite.Open(dm.config.Database.Name)
	} else {
		// PostgreSQL for production
		dsn := dm.config.GetDSN()
		dialector = postgres.Open(dsn)
	}

	// Configure GORM logger with performance optimizations
	var logLevel logger.LogLevel
	if dm.config.Database.Debug {
		logLevel = logger.Info
	} else {
		logLevel = logger.Silent
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// Performance optimizations
		PrepareStmt: true,              // Enable prepared statement cache
		DisableForeignKeyConstraintWhenMigrating: true, // Faster migrations
		SkipDefaultTransaction: true,   // Skip default transaction for single queries
		QueryFields: true,              // Select all fields by name
		CreateBatchSize: 1000,          // Optimized batch size for bulk operations
	}

	// Open database connection
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	if err := dm.configureConnectionPool(db); err != nil {
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	dm.db = db
	logrus.Info("Database connection established successfully")
	return db, nil
}

// configureConnectionPool configures database connection pool settings with performance optimizations
func (dm *DBManager) configureConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Performance-optimized connection pool settings
	maxIdleConns := dm.config.Database.MaxIdleConns
	maxOpenConns := dm.config.Database.MaxOpenConns
	connMaxLifetime := time.Duration(dm.config.Database.ConnMaxLifetimeMin) * time.Minute

	// Apply optimized defaults if not configured
	if maxIdleConns <= 0 {
		maxIdleConns = 25 // Optimized default for better performance
	}
	if maxOpenConns <= 0 {
		maxOpenConns = 100 // Higher default for concurrent workloads
	}
	if connMaxLifetime <= 0 {
		connMaxLifetime = time.Hour // Longer lifetime for connection reuse
	}

	// Set optimized connection pool settings
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // Prevent idle connection buildup

	logrus.WithFields(logrus.Fields{
		"max_idle_conns":     maxIdleConns,
		"max_open_conns":     maxOpenConns,
		"conn_max_lifetime":  connMaxLifetime,
		"conn_max_idle_time": 30 * time.Minute,
	}).Info("Database connection pool configured with performance optimizations")

	return nil
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	logrus.Info("Starting database migration...")

	// Use the centralized migration from model package
	if err := model.AutoMigrate(db); err != nil {
		return fmt.Errorf("failed to run auto migration: %w", err)
	}

	// Create indexes
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

// createIndexes creates comprehensive database indexes for performance optimization
func createIndexes(db *gorm.DB) error {
	indexes := []struct {
		table   string
		columns []string
		name    string
		unique  bool
		partial string // Partial index condition for PostgreSQL
	}{
		// User indexes
		{
			table:   "users",
			columns: []string{"username"},
			name:    "idx_users_username",
			unique:  true,
		},
		{
			table:   "users",
			columns: []string{"email"},
			name:    "idx_users_email",
			unique:  true,
		},
		{
			table:   "users",
			columns: []string{"created_at"},
			name:    "idx_users_created_at",
		},
		// Container indexes - optimized for common queries
		{
			table:   "containers",
			columns: []string{"user_id", "status"},
			name:    "idx_containers_user_status",
		},
		{
			table:   "containers",
			columns: []string{"auto_update_enabled", "status"},
			name:    "idx_containers_auto_update_status",
			partial: "WHERE auto_update_enabled = true AND status IN ('running', 'restarting')",
		},
		{
			table:   "containers",
			columns: []string{"image_name", "image_tag"},
			name:    "idx_containers_image",
		},
		{
			table:   "containers",
			columns: []string{"updated_at"},
			name:    "idx_containers_updated_at",
		},
		// Update history indexes - optimized for time-series queries
		{
			table:   "update_histories",
			columns: []string{"container_id", "created_at"},
			name:    "idx_update_histories_container_created",
		},
		{
			table:   "update_histories",
			columns: []string{"status", "created_at"},
			name:    "idx_update_histories_status_created",
		},
		{
			table:   "update_histories",
			columns: []string{"created_at"},
			name:    "idx_update_histories_created_at",
		},
		// Notification indexes
		{
			table:   "notification_logs",
			columns: []string{"user_id", "status", "created_at"},
			name:    "idx_notification_logs_user_status_created",
		},
		{
			table:   "notification_logs",
			columns: []string{"created_at"},
			name:    "idx_notification_logs_created_at",
		},
		{
			table:   "notification_logs",
			columns: []string{"status"},
			name:    "idx_notification_logs_status",
			partial: "WHERE status IN ('failed', 'pending')",
		},
		// Image version indexes
		{
			table:   "image_versions",
			columns: []string{"image_name", "tag"},
			name:    "idx_image_versions_name_tag",
			unique:  true,
		},
		{
			table:   "image_versions",
			columns: []string{"image_name", "created_at"},
			name:    "idx_image_versions_name_created",
		},
		{
			table:   "image_versions",
			columns: []string{"created_at"},
			name:    "idx_image_versions_created_at",
		},
		// Scheduled task indexes
		{
			table:   "scheduled_tasks",
			columns: []string{"task_type", "status"},
			name:    "idx_scheduled_tasks_type_status",
		},
		{
			table:   "scheduled_tasks",
			columns: []string{"next_run_at", "status"},
			name:    "idx_scheduled_tasks_next_run_status",
			partial: "WHERE status = 'active' AND next_run_at IS NOT NULL",
		},
		{
			table:   "scheduled_tasks",
			columns: []string{"created_at"},
			name:    "idx_scheduled_tasks_created_at",
		},
	}

	// Create indexes with performance monitoring
	logrus.Info("Creating performance-optimized database indexes...")
	start := time.Now()
	createdCount := 0

	for _, idx := range indexes {
		indexStart := time.Now()

		// Build index creation SQL with proper syntax
		var sql string
		if idx.unique {
			sql = fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)",
				idx.name, idx.table, joinColumns(idx.columns))
		} else {
			sql = fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				idx.name, idx.table, joinColumns(idx.columns))
		}

		// Add partial index condition for PostgreSQL
		if idx.partial != "" && isPostgreSQL(db) {
			sql += " " + idx.partial
		}

		if err := db.Exec(sql).Error; err != nil {
			logrus.WithFields(logrus.Fields{
				"index_name": idx.name,
				"table":      idx.table,
				"error":      err,
			}).Warn("Failed to create index, continuing...")
			continue
		}

		createdCount++
		logrus.WithFields(logrus.Fields{
			"index_name": idx.name,
			"table":      idx.table,
			"duration":   time.Since(indexStart),
			"unique":     idx.unique,
			"partial":    idx.partial != "",
		}).Debug("Index created successfully")
	}

	logrus.WithFields(logrus.Fields{
		"total_indexes": len(indexes),
		"created":       createdCount,
		"duration":      time.Since(start),
	}).Info("Database index creation completed")

	return nil
}

// joinColumns joins column names with commas
func joinColumns(columns []string) string {
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}

// HealthCheck performs database health check
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Ping database
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check if we can perform a simple query
	var result map[string]interface{}
	if err := db.Raw("SELECT 1 as test").Scan(&result).Error; err != nil {
		return fmt.Errorf("failed to execute test query: %w", err)
	}

	return nil
}

// GetDBStats returns database connection statistics
func GetDBStats(db *gorm.DB) (*DBStats, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	stats := sqlDB.Stats()

	return &DBStats{
		OpenConnections: stats.OpenConnections,
		InUse:          stats.InUse,
		Idle:           stats.Idle,
		WaitCount:      stats.WaitCount,
		WaitDuration:   stats.WaitDuration,
		MaxIdleClosed:  stats.MaxIdleClosed,
		MaxLifetimeClosed: stats.MaxLifetimeClosed,
	}, nil
}

// DBStats represents database connection statistics
type DBStats struct {
	OpenConnections   int           `json:"open_connections"`
	InUse            int           `json:"in_use"`
	Idle             int           `json:"idle"`
	WaitCount        int64         `json:"wait_count"`
	WaitDuration     time.Duration `json:"wait_duration"`
	MaxIdleClosed    int64         `json:"max_idle_closed"`
	MaxLifetimeClosed int64        `json:"max_lifetime_closed"`
}

// Transaction executes a function within a database transaction
func Transaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchInsert performs optimized batch insert operation with monitoring
func BatchInsert(db *gorm.DB, records interface{}, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000 // Default optimized batch size
	}

	start := time.Now()
	err := db.CreateInBatches(records, batchSize).Error

	logrus.WithFields(logrus.Fields{
		"batch_size": batchSize,
		"duration":   time.Since(start),
		"success":    err == nil,
	}).Debug("Batch insert operation completed")

	return err
}

// CloseDB safely closes database connection
func CloseDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	logrus.Info("Database connection closed successfully")
	return nil
}

// IsConnectionError checks if the error is a connection-related error
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Common connection error patterns
	errorMessages := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no connection",
		"database is closed",
		"invalid connection",
	}

	errMsg := err.Error()
	for _, msg := range errorMessages {
		if len(errMsg) >= len(msg) && errMsg[:len(msg)] == msg {
			return true
		}
	}

	return false
}

// RetryOperation retries database operation on connection errors with exponential backoff
func RetryOperation(operation func() error, maxRetries int, delay time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if IsConnectionError(err) && i < maxRetries-1 {
				retryDelay := delay * time.Duration(1<<uint(i)) // Exponential backoff
				logrus.WithFields(logrus.Fields{
					"attempt":     i + 1,
					"max_retries": maxRetries,
					"retry_delay": retryDelay,
					"error":       err,
				}).Warn("Database operation failed, retrying...")
				time.Sleep(retryDelay)
				continue
			}
			return err
		}
		return nil
	}

	return fmt.Errorf("operation failed after %d retries, last error: %w", maxRetries, lastErr)
}

// isPostgreSQL checks if the database is PostgreSQL
func isPostgreSQL(db *gorm.DB) bool {
	return db.Dialector.Name() == "postgres"
}

// BatchOperationConfig configures batch database operations
type BatchOperationConfig struct {
	BatchSize     int
	MaxRetries    int
	RetryDelay    time.Duration
	ProgressCallback func(processed, total int)
}

// DefaultBatchConfig returns default batch operation configuration
func DefaultBatchConfig() BatchOperationConfig {
	return BatchOperationConfig{
		BatchSize:  1000,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// BatchInsertWithConfig performs optimized batch insert with configuration
func BatchInsertWithConfig(db *gorm.DB, records interface{}, config BatchOperationConfig) error {
	start := time.Now()

	err := RetryOperation(func() error {
		return db.CreateInBatches(records, config.BatchSize).Error
	}, config.MaxRetries, config.RetryDelay)

	if err != nil {
		return fmt.Errorf("batch insert failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"batch_size": config.BatchSize,
		"duration":   time.Since(start),
	}).Debug("Batch insert completed successfully")

	return nil
}

// QueryPerformanceMetrics tracks query performance
type QueryPerformanceMetrics struct {
	QuerySQL     string        `json:"query_sql"`
	Duration     time.Duration `json:"duration"`
	RowsAffected int64         `json:"rows_affected"`
	Timestamp    time.Time     `json:"timestamp"`
	Error        string        `json:"error,omitempty"`
}

// WithQueryMetrics wraps database operations with performance metrics
func WithQueryMetrics(db *gorm.DB, queryName string, operation func(*gorm.DB) *gorm.DB) error {
	start := time.Now()
	var rowsAffected int64
	var querySQL string

	// Execute the operation
	result := operation(db.Session(&gorm.Session{DryRun: false}))
	if result.Error != nil {
		metrics := QueryPerformanceMetrics{
			QuerySQL:     querySQL,
			Duration:     time.Since(start),
			RowsAffected: result.RowsAffected,
			Timestamp:    time.Now(),
			Error:        result.Error.Error(),
		}

		logrus.WithFields(logrus.Fields{
			"query_name":    queryName,
			"duration":      metrics.Duration,
			"rows_affected": metrics.RowsAffected,
			"error":         metrics.Error,
		}).Warn("Query executed with error")

		return result.Error
	}

	rowsAffected = result.RowsAffected

	metrics := QueryPerformanceMetrics{
		QuerySQL:     querySQL,
		Duration:     time.Since(start),
		RowsAffected: rowsAffected,
		Timestamp:    time.Now(),
	}

	// Log slow queries (> 100ms)
	if metrics.Duration > 100*time.Millisecond {
		logrus.WithFields(logrus.Fields{
			"query_name":    queryName,
			"duration":      metrics.Duration,
			"rows_affected": metrics.RowsAffected,
		}).Warn("Slow query detected")
	} else {
		logrus.WithFields(logrus.Fields{
			"query_name":    queryName,
			"duration":      metrics.Duration,
			"rows_affected": metrics.RowsAffected,
		}).Debug("Query executed successfully")
	}

	return nil
}