package security

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseSecurityConfig represents database security configuration
type DatabaseSecurityConfig struct {
	// Connection security
	TLSEnabled         bool          `json:"tls_enabled"`
	TLSConfig          *tls.Config   `json:"-"`
	CertificateAuth    bool          `json:"certificate_auth"`
	ConnectionTimeout  time.Duration `json:"connection_timeout"`

	// Connection pooling
	MaxOpenConns       int           `json:"max_open_conns"`
	MaxIdleConns       int           `json:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime    time.Duration `json:"conn_max_idle_time"`

	// Query security
	PreparedStatements bool          `json:"prepared_statements"`
	QueryTimeout       time.Duration `json:"query_timeout"`
	MaxQueryLength     int           `json:"max_query_length"`

	// Audit logging
	AuditEnabled       bool          `json:"audit_enabled"`
	AuditLevel         AuditLevel    `json:"audit_level"`
	LogSlowQueries     bool          `json:"log_slow_queries"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`

	// Data encryption
	EncryptionEnabled  bool          `json:"encryption_enabled"`
	EncryptionKey      string        `json:"encryption_key"`

	// Access control
	ReadOnlyMode       bool          `json:"read_only_mode"`
	AllowedOperations  []string      `json:"allowed_operations"`
	RestrictedTables   []string      `json:"restricted_tables"`

	// Monitoring
	MonitorConnections bool          `json:"monitor_connections"`
	AlertOnFailedQueries bool        `json:"alert_on_failed_queries"`
	MaxFailedQueries   int           `json:"max_failed_queries"`
}

// AuditLevel represents the level of database auditing
type AuditLevel int

const (
	AuditNone AuditLevel = iota
	AuditErrors
	AuditWrites
	AuditReads
	AuditAll
)

// DefaultDatabaseSecurityConfig returns secure default configuration
func DefaultDatabaseSecurityConfig() *DatabaseSecurityConfig {
	return &DatabaseSecurityConfig{
		TLSEnabled:         true,
		ConnectionTimeout:  30 * time.Second,
		MaxOpenConns:       25,
		MaxIdleConns:       5,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    2 * time.Minute,
		PreparedStatements: true,
		QueryTimeout:       30 * time.Second,
		MaxQueryLength:     10000,
		AuditEnabled:       true,
		AuditLevel:         AuditWrites,
		LogSlowQueries:     true,
		SlowQueryThreshold: 2 * time.Second,
		EncryptionEnabled:  true,
		AllowedOperations:  []string{"SELECT", "INSERT", "UPDATE", "DELETE"},
		MonitorConnections: true,
		AlertOnFailedQueries: true,
		MaxFailedQueries:   10,
	}
}

// AuditRecord represents a database audit record
type AuditRecord struct {
	ID          int64     `json:"id" db:"id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	UserID      *int64    `json:"user_id,omitempty" db:"user_id"`
	Username    string    `json:"username" db:"username"`
	SessionID   string    `json:"session_id" db:"session_id"`
	Operation   string    `json:"operation" db:"operation"`
	TableName   string    `json:"table_name" db:"table_name"`
	Query       string    `json:"query" db:"query"`
	Parameters  string    `json:"parameters,omitempty" db:"parameters"`
	RowsAffected int64    `json:"rows_affected" db:"rows_affected"`
	Duration    time.Duration `json:"duration" db:"duration"`
	Success     bool      `json:"success" db:"success"`
	ErrorMsg    string    `json:"error_msg,omitempty" db:"error_msg"`
	ClientIP    string    `json:"client_ip" db:"client_ip"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
}

// SecureDatabase represents a secure database wrapper
type SecureDatabase struct {
	config      *DatabaseSecurityConfig
	db          *gorm.DB
	sqlDB       *sql.DB
	auditLogger *AuditLogger
	stats       *DatabaseStats
	mutex       sync.RWMutex
}

// DatabaseStats represents database security statistics
type DatabaseStats struct {
	TotalQueries     int64     `json:"total_queries"`
	FailedQueries    int64     `json:"failed_queries"`
	SlowQueries      int64     `json:"slow_queries"`
	BlockedQueries   int64     `json:"blocked_queries"`
	AuditRecords     int64     `json:"audit_records"`
	ActiveConnections int      `json:"active_connections"`
	LastUpdate       time.Time `json:"last_update"`
}

// AuditLogger handles database audit logging
type AuditLogger struct {
	config     *DatabaseSecurityConfig
	auditDB    *sqlx.DB
	auditQueue chan *AuditRecord
	stats      *DatabaseStats
	mutex      sync.RWMutex
}

// NewSecureDatabase creates a new secure database wrapper
func NewSecureDatabase(dsn string, config *DatabaseSecurityConfig) (*SecureDatabase, error) {
	if config == nil {
		config = DefaultDatabaseSecurityConfig()
	}

	// Determine database driver from DSN
	var dialector gorm.Dialector
	if strings.Contains(dsn, "postgres://") || strings.Contains(dsn, "postgresql://") {
		dialector = postgres.Open(dsn)
	} else if strings.Contains(dsn, "mysql://") || strings.Contains(dsn, "@tcp(") {
		dialector = mysql.Open(dsn)
	} else if strings.HasSuffix(dsn, ".db") || strings.Contains(dsn, "sqlite") {
		dialector = sqlite.Open(dsn)
	} else {
		return nil, fmt.Errorf("unsupported database driver")
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		PrepareStmt: config.PreparedStatements,
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{
				SlowThreshold:             config.SlowQueryThreshold,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	}

	// Open database connection
	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Initialize audit logger
	var auditLogger *AuditLogger
	if config.AuditEnabled {
		auditLogger, err = NewAuditLogger(dsn, config)
		if err != nil {
			logrus.WithError(err).Warn("Failed to initialize audit logger")
		}
	}

	secureDB := &SecureDatabase{
		config:      config,
		db:          db,
		sqlDB:       sqlDB,
		auditLogger: auditLogger,
		stats:       &DatabaseStats{LastUpdate: time.Now()},
	}

	// Start monitoring
	if config.MonitorConnections {
		go secureDB.startMonitoring()
	}

	// Create audit tables if needed
	if config.AuditEnabled && auditLogger != nil {
		if err := secureDB.createAuditTables(); err != nil {
			logrus.WithError(err).Warn("Failed to create audit tables")
		}
	}

	return secureDB, nil
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(dsn string, config *DatabaseSecurityConfig) (*AuditLogger, error) {
	// Use separate connection for audit logging to avoid conflicts
	auditDB, err := sqlx.Open("postgres", dsn) // Adjust driver as needed
	if err != nil {
		return nil, fmt.Errorf("failed to open audit database: %w", err)
	}

	auditLogger := &AuditLogger{
		config:     config,
		auditDB:    auditDB,
		auditQueue: make(chan *AuditRecord, 1000),
		stats:      &DatabaseStats{},
	}

	// Start audit processing goroutine
	go auditLogger.processAuditQueue()

	return auditLogger, nil
}

// SecureQuery executes a query with security checks and auditing
func (sd *SecureDatabase) SecureQuery(ctx context.Context, query string, args []interface{}, userContext *QueryUserContext) (*gorm.DB, error) {
	startTime := time.Now()
	sd.stats.TotalQueries++

	// Validate query
	if err := sd.validateQuery(query, userContext); err != nil {
		sd.stats.FailedQueries++
		sd.stats.BlockedQueries++
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Add query timeout
	queryCtx := ctx
	if sd.config.QueryTimeout > 0 {
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithTimeout(ctx, sd.config.QueryTimeout)
		defer cancel()
	}

	// Execute query
	result := sd.db.WithContext(queryCtx)
	if len(args) > 0 {
		result = result.Raw(query, args...)
	} else {
		result = result.Raw(query)
	}

	// Check for errors
	var success = true
	var errorMsg string
	if result.Error != nil {
		success = false
		errorMsg = result.Error.Error()
		sd.stats.FailedQueries++
	}

	duration := time.Since(startTime)

	// Check for slow queries
	if duration > sd.config.SlowQueryThreshold {
		sd.stats.SlowQueries++
		logrus.WithFields(logrus.Fields{
			"query":    query[:min(len(query), 200)],
			"duration": duration,
			"user_id":  userContext.UserID,
		}).Warn("Slow database query detected")
	}

	// Audit logging
	if sd.config.AuditEnabled && sd.auditLogger != nil && sd.shouldAuditQuery(query, success) {
		auditRecord := &AuditRecord{
			Timestamp:    startTime,
			UserID:       &userContext.UserID,
			Username:     userContext.Username,
			SessionID:    userContext.SessionID,
			Operation:    sd.extractOperation(query),
			TableName:    sd.extractTableName(query),
			Query:        query,
			Duration:     duration,
			Success:      success,
			ErrorMsg:     errorMsg,
			ClientIP:     userContext.ClientIP,
			UserAgent:    userContext.UserAgent,
		}

		if result.Error == nil && result.RowsAffected >= 0 {
			auditRecord.RowsAffected = result.RowsAffected
		}

		sd.auditLogger.LogAudit(auditRecord)
	}

	if !success {
		return nil, result.Error
	}

	return result, nil
}

// QueryUserContext represents the user context for database queries
type QueryUserContext struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	Role      string `json:"role"`
}

// validateQuery performs security validation on database queries
func (sd *SecureDatabase) validateQuery(query string, userContext *QueryUserContext) error {
	query = strings.TrimSpace(strings.ToUpper(query))

	// Check query length
	if sd.config.MaxQueryLength > 0 && len(query) > sd.config.MaxQueryLength {
		return fmt.Errorf("query too long: %d characters, max: %d", len(query), sd.config.MaxQueryLength)
	}

	// Check read-only mode
	if sd.config.ReadOnlyMode {
		if !strings.HasPrefix(query, "SELECT") && !strings.HasPrefix(query, "SHOW") && !strings.HasPrefix(query, "DESCRIBE") {
			return fmt.Errorf("database is in read-only mode")
		}
	}

	// Check allowed operations
	if len(sd.config.AllowedOperations) > 0 {
		operation := sd.extractOperation(query)
		allowed := false
		for _, allowedOp := range sd.config.AllowedOperations {
			if strings.EqualFold(operation, allowedOp) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("operation %s not allowed", operation)
		}
	}

	// Check restricted tables
	if len(sd.config.RestrictedTables) > 0 {
		tableName := sd.extractTableName(query)
		for _, restrictedTable := range sd.config.RestrictedTables {
			if strings.EqualFold(tableName, restrictedTable) {
				// Check if user has access to restricted table
				if !sd.hasTableAccess(userContext, restrictedTable) {
					return fmt.Errorf("access denied to table %s", restrictedTable)
				}
			}
		}
	}

	// Check for SQL injection patterns
	if sd.containsSQLInjection(query) {
		return fmt.Errorf("potential SQL injection detected")
	}

	return nil
}

// extractOperation extracts the SQL operation from a query
func (sd *SecureDatabase) extractOperation(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))
	parts := strings.Split(query, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return "UNKNOWN"
}

// extractTableName extracts the table name from a query
func (sd *SecureDatabase) extractTableName(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))

	// Simple table extraction - can be improved with proper SQL parsing
	if strings.HasPrefix(query, "SELECT") {
		if fromIndex := strings.Index(query, "FROM "); fromIndex != -1 {
			parts := strings.Split(query[fromIndex+5:], " ")
			if len(parts) > 0 {
				return strings.Trim(parts[0], "`\"[]")
			}
		}
	} else if strings.HasPrefix(query, "INSERT INTO ") {
		parts := strings.Split(query[12:], " ")
		if len(parts) > 0 {
			return strings.Trim(parts[0], "`\"[]")
		}
	} else if strings.HasPrefix(query, "UPDATE ") {
		parts := strings.Split(query[7:], " ")
		if len(parts) > 0 {
			return strings.Trim(parts[0], "`\"[]")
		}
	} else if strings.HasPrefix(query, "DELETE FROM ") {
		parts := strings.Split(query[12:], " ")
		if len(parts) > 0 {
			return strings.Trim(parts[0], "`\"[]")
		}
	}

	return "UNKNOWN"
}

// hasTableAccess checks if user has access to a restricted table
func (sd *SecureDatabase) hasTableAccess(userContext *QueryUserContext, tableName string) bool {
	// Implement table-level access control based on user role
	// This is a simplified version - implement according to your needs
	switch strings.ToLower(userContext.Role) {
	case "admin":
		return true
	case "user":
		// Users can access non-sensitive tables
		sensitiveTables := []string{"users", "audit_logs", "system_config"}
		for _, sensitive := range sensitiveTables {
			if strings.EqualFold(tableName, sensitive) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// containsSQLInjection checks for SQL injection patterns
func (sd *SecureDatabase) containsSQLInjection(query string) bool {
	query = strings.ToLower(query)

	// Common SQL injection patterns
	injectionPatterns := []string{
		"union select",
		"'; drop table",
		"'; delete from",
		"or 1=1",
		"or '1'='1'",
		"and 1=1",
		"/*",
		"*/",
		"xp_",
		"sp_",
		"exec(",
		"execute(",
	}

	for _, pattern := range injectionPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	return false
}

// shouldAuditQuery determines if a query should be audited
func (sd *SecureDatabase) shouldAuditQuery(query string, success bool) bool {
	operation := strings.ToUpper(sd.extractOperation(query))

	switch sd.config.AuditLevel {
	case AuditNone:
		return false
	case AuditErrors:
		return !success
	case AuditWrites:
		return !success || operation == "INSERT" || operation == "UPDATE" || operation == "DELETE"
	case AuditReads:
		return !success || operation == "SELECT" || operation == "INSERT" || operation == "UPDATE" || operation == "DELETE"
	case AuditAll:
		return true
	default:
		return false
	}
}

// LogAudit logs an audit record
func (al *AuditLogger) LogAudit(record *AuditRecord) {
	select {
	case al.auditQueue <- record:
		// Successfully queued
	default:
		// Queue is full, log error
		logrus.Warn("Audit queue is full, dropping audit record")
	}
}

// processAuditQueue processes the audit queue
func (al *AuditLogger) processAuditQueue() {
	for record := range al.auditQueue {
		if err := al.insertAuditRecord(record); err != nil {
			logrus.WithError(err).Error("Failed to insert audit record")
		} else {
			al.mutex.Lock()
			al.stats.AuditRecords++
			al.mutex.Unlock()
		}
	}
}

// insertAuditRecord inserts an audit record into the database
func (al *AuditLogger) insertAuditRecord(record *AuditRecord) error {
	query := `
		INSERT INTO audit_logs (
			timestamp, user_id, username, session_id, operation, table_name,
			query, parameters, rows_affected, duration, success, error_msg,
			client_ip, user_agent
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := al.auditDB.Exec(query,
		record.Timestamp,
		record.UserID,
		record.Username,
		record.SessionID,
		record.Operation,
		record.TableName,
		record.Query,
		record.Parameters,
		record.RowsAffected,
		int64(record.Duration),
		record.Success,
		record.ErrorMsg,
		record.ClientIP,
		record.UserAgent,
	)

	return err
}

// createAuditTables creates the audit tables if they don't exist
func (sd *SecureDatabase) createAuditTables() error {
	auditTableQuery := `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			user_id BIGINT,
			username VARCHAR(255),
			session_id VARCHAR(255),
			operation VARCHAR(50) NOT NULL,
			table_name VARCHAR(255),
			query TEXT,
			parameters TEXT,
			rows_affected BIGINT DEFAULT 0,
			duration BIGINT DEFAULT 0,
			success BOOLEAN NOT NULL DEFAULT false,
			error_msg TEXT,
			client_ip INET,
			user_agent TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`

	if err := sd.db.Exec(auditTableQuery).Error; err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_operation ON audit_logs(operation)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_table_name ON audit_logs(table_name)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_success ON audit_logs(success)",
	}

	for _, indexQuery := range indexes {
		if err := sd.db.Exec(indexQuery).Error; err != nil {
			logrus.WithError(err).WithField("query", indexQuery).Warn("Failed to create audit index")
		}
	}

	return nil
}

// startMonitoring starts database monitoring
func (sd *SecureDatabase) startMonitoring() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sd.updateStats()
	}
}

// updateStats updates database statistics
func (sd *SecureDatabase) updateStats() {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()

	stats := sd.sqlDB.Stats()
	sd.stats.ActiveConnections = stats.InUse
	sd.stats.LastUpdate = time.Now()
}

// GetStats returns database security statistics
func (sd *SecureDatabase) GetStats() map[string]interface{} {
	sd.mutex.RLock()
	defer sd.mutex.RUnlock()

	stats := map[string]interface{}{
		"queries": map[string]interface{}{
			"total":   sd.stats.TotalQueries,
			"failed":  sd.stats.FailedQueries,
			"slow":    sd.stats.SlowQueries,
			"blocked": sd.stats.BlockedQueries,
		},
		"connections": map[string]interface{}{
			"active":      sd.stats.ActiveConnections,
			"max_open":    sd.config.MaxOpenConns,
			"max_idle":    sd.config.MaxIdleConns,
		},
		"audit": map[string]interface{}{
			"enabled": sd.config.AuditEnabled,
			"level":   sd.config.AuditLevel,
			"records": sd.stats.AuditRecords,
		},
		"security": map[string]interface{}{
			"tls_enabled":         sd.config.TLSEnabled,
			"prepared_statements": sd.config.PreparedStatements,
			"encryption_enabled":  sd.config.EncryptionEnabled,
			"read_only_mode":      sd.config.ReadOnlyMode,
		},
		"last_update": sd.stats.LastUpdate,
	}

	return stats
}

// GetAuditRecords retrieves audit records with filtering
func (sd *SecureDatabase) GetAuditRecords(filters AuditFilters) ([]AuditRecord, error) {
	if !sd.config.AuditEnabled || sd.auditLogger == nil {
		return nil, fmt.Errorf("audit logging not enabled")
	}

	query := "SELECT * FROM audit_logs WHERE 1=1"
	args := make([]interface{}, 0)
	argCount := 0

	if filters.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filters.UserID)
	}

	if filters.Operation != "" {
		argCount++
		query += fmt.Sprintf(" AND operation = $%d", argCount)
		args = append(args, filters.Operation)
	}

	if filters.TableName != "" {
		argCount++
		query += fmt.Sprintf(" AND table_name = $%d", argCount)
		args = append(args, filters.TableName)
	}

	if !filters.StartTime.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, filters.StartTime)
	}

	if !filters.EndTime.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, filters.EndTime)
	}

	if filters.SuccessOnly != nil {
		argCount++
		query += fmt.Sprintf(" AND success = $%d", argCount)
		args = append(args, *filters.SuccessOnly)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
	}

	var records []AuditRecord
	err := sd.auditLogger.auditDB.Select(&records, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audit records: %w", err)
	}

	return records, nil
}

// AuditFilters represents filters for audit record queries
type AuditFilters struct {
	UserID      *int64    `json:"user_id,omitempty"`
	Operation   string    `json:"operation,omitempty"`
	TableName   string    `json:"table_name,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	SuccessOnly *bool     `json:"success_only,omitempty"`
	Limit       int       `json:"limit,omitempty"`
}

// Close closes the database connections
func (sd *SecureDatabase) Close() error {
	if sd.auditLogger != nil {
		close(sd.auditLogger.auditQueue)
		sd.auditLogger.auditDB.Close()
	}

	return sd.sqlDB.Close()
}

// BackupAuditLogs creates a backup of audit logs
func (sd *SecureDatabase) BackupAuditLogs(filePath string, filters AuditFilters) error {
	if !sd.config.AuditEnabled {
		return fmt.Errorf("audit logging not enabled")
	}

	records, err := sd.GetAuditRecords(filters)
	if err != nil {
		return fmt.Errorf("failed to get audit records: %w", err)
	}

	// Export to JSON file (could be enhanced to support other formats)
	return sd.exportAuditRecords(records, filePath)
}

// exportAuditRecords exports audit records to a file
func (sd *SecureDatabase) exportAuditRecords(records []AuditRecord, filePath string) error {
	// Implementation for exporting audit records
	// This is a placeholder - implement according to your needs
	logrus.WithFields(logrus.Fields{
		"record_count": len(records),
		"file_path":    filePath,
	}).Info("Exporting audit records")

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}