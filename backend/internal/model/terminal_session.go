package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TerminalSession represents a WebSocket terminal session for container access
type TerminalSession struct {
	ID           int                `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID    string             `json:"session_id" gorm:"uniqueIndex;not null;size:100;index:idx_terminal_sessions_session_id"`
	ContainerID  int                `json:"container_id" gorm:"not null;index:idx_terminal_sessions_container_id"`
	UserID       *int               `json:"user_id,omitempty" gorm:"index:idx_terminal_sessions_user_id"`
	Status       TerminalStatus     `json:"status" gorm:"not null;default:'connecting';index:idx_terminal_sessions_status"`
	Command      string             `json:"command" gorm:"size:500;not null;default:'/bin/bash'"`
	WorkingDir   string             `json:"working_dir,omitempty" gorm:"size:500"`
	Environment  string             `json:"environment,omitempty" gorm:"type:jsonb;default:'[]'"`
	TTYSettings  *TTYSettings       `json:"tty_settings,omitempty" gorm:"type:jsonb"`
	Capabilities *SessionCapabilities `json:"capabilities,omitempty" gorm:"type:jsonb"`

	// Connection and lifecycle management
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	LastActivity   time.Time  `json:"last_activity"`
	ExpiresAt      time.Time  `json:"expires_at" gorm:"index:idx_terminal_sessions_expires_at"`
	TimeoutSeconds int        `json:"timeout_seconds" gorm:"not null;default:3600"` // 1 hour default

	// Session metadata and statistics
	BytesReceived     uint64             `json:"bytes_received" gorm:"default:0"`
	BytesSent         uint64             `json:"bytes_sent" gorm:"default:0"`
	CommandCount      int                `json:"command_count" gorm:"default:0"`
	ErrorCount        int                `json:"error_count" gorm:"default:0"`
	LastError         string             `json:"last_error,omitempty" gorm:"type:text"`
	SessionHistory    *SessionHistory    `json:"session_history,omitempty" gorm:"type:jsonb"`

	// Security and access control
	ClientIP         string    `json:"client_ip" gorm:"size:45;index:idx_terminal_sessions_client_ip"`
	UserAgent        string    `json:"user_agent,omitempty" gorm:"size:500"`
	AccessLevel      string    `json:"access_level" gorm:"size:20;not null;default:'read-write';index:idx_terminal_sessions_access_level"`
	AllowedCommands  string    `json:"allowed_commands,omitempty" gorm:"type:jsonb"`
	BlockedCommands  string    `json:"blocked_commands,omitempty" gorm:"type:jsonb"`

	// Process information
	ProcessID       int    `json:"process_id,omitempty"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	SignalReceived  string `json:"signal_received,omitempty" gorm:"size:20"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Container *Container `json:"container,omitempty" gorm:"foreignKey:ContainerID"`
	User      *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TerminalStatus defines terminal session status
type TerminalStatus string

const (
	TerminalStatusConnecting  TerminalStatus = "connecting"
	TerminalStatusActive      TerminalStatus = "active"
	TerminalStatusIdle        TerminalStatus = "idle"
	TerminalStatusDisconnected TerminalStatus = "disconnected"
	TerminalStatusExpired     TerminalStatus = "expired"
	TerminalStatusError       TerminalStatus = "error"
	TerminalStatusKilled      TerminalStatus = "killed"
)

// TTYSettings represents terminal TTY configuration
type TTYSettings struct {
	Columns int    `json:"columns" validate:"min=1,max=500"`
	Rows    int    `json:"rows" validate:"min=1,max=200"`
	Term    string `json:"term" validate:"required"`
}

// SessionCapabilities represents what operations the session can perform
type SessionCapabilities struct {
	CanExecuteCommands bool     `json:"can_execute_commands"`
	CanAccessFiles     bool     `json:"can_access_files"`
	CanInstallPackages bool     `json:"can_install_packages"`
	CanModifySystem    bool     `json:"can_modify_system"`
	AllowedPaths       []string `json:"allowed_paths,omitempty"`
	RestrictedPaths    []string `json:"restricted_paths,omitempty"`
}

// SessionHistory represents command history and statistics
type SessionHistory struct {
	Commands     []CommandEntry `json:"commands"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      *time.Time     `json:"end_time,omitempty"`
	TotalCommands int           `json:"total_commands"`
	UniqueCommands int          `json:"unique_commands"`
	Errors        []ErrorEntry  `json:"errors,omitempty"`
}

// CommandEntry represents a single command execution
type CommandEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Command     string    `json:"command"`
	ExitCode    int       `json:"exit_code"`
	Duration    int64     `json:"duration"` // milliseconds
	OutputSize  int       `json:"output_size"`
}

// ErrorEntry represents an error that occurred during the session
type ErrorEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	Context   string    `json:"context,omitempty"`
}

// TerminalSessionFilter represents filters for querying terminal sessions
type TerminalSessionFilter struct {
	ContainerID *int           `json:"container_id,omitempty"`
	UserID      *int           `json:"user_id,omitempty"`
	Status      TerminalStatus `json:"status,omitempty"`
	ClientIP    string         `json:"client_ip,omitempty"`
	AccessLevel string         `json:"access_level,omitempty"`
	ActiveOnly  bool           `json:"active_only,omitempty"`
	ExpiredOnly bool           `json:"expired_only,omitempty"`
	Limit       int            `json:"limit,omitempty"`
	Offset      int            `json:"offset,omitempty"`
	OrderBy     string         `json:"order_by,omitempty"`
}

// TableName returns the table name for TerminalSession model
func (TerminalSession) TableName() string {
	return "terminal_sessions"
}

// GORM hooks

// BeforeCreate hook for TerminalSession model
func (ts *TerminalSession) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()

	// Set default values
	if ts.Command == "" {
		ts.Command = "/bin/bash"
	}
	if ts.TimeoutSeconds == 0 {
		ts.TimeoutSeconds = 3600 // 1 hour
	}
	if ts.AccessLevel == "" {
		ts.AccessLevel = "read-write"
	}

	// Set timestamps
	ts.ConnectedAt = now
	ts.LastActivity = now
	ts.ExpiresAt = now.Add(time.Duration(ts.TimeoutSeconds) * time.Second)

	// Initialize session history
	if ts.SessionHistory == nil {
		ts.SessionHistory = &SessionHistory{
			StartTime:   now,
			Commands:    make([]CommandEntry, 0),
			Errors:      make([]ErrorEntry, 0),
		}
	}

	return nil
}

// BeforeSave hook for TerminalSession model
func (ts *TerminalSession) BeforeSave(tx *gorm.DB) error {
	// Validate JSON fields
	if ts.TTYSettings != nil {
		if _, err := json.Marshal(ts.TTYSettings); err != nil {
			return fmt.Errorf("invalid tty_settings JSON: %w", err)
		}
		// Validate TTY settings
		if ts.TTYSettings.Columns < 1 || ts.TTYSettings.Columns > 500 {
			return fmt.Errorf("invalid TTY columns: %d", ts.TTYSettings.Columns)
		}
		if ts.TTYSettings.Rows < 1 || ts.TTYSettings.Rows > 200 {
			return fmt.Errorf("invalid TTY rows: %d", ts.TTYSettings.Rows)
		}
	}

	if ts.Capabilities != nil {
		if _, err := json.Marshal(ts.Capabilities); err != nil {
			return fmt.Errorf("invalid capabilities JSON: %w", err)
		}
	}

	if ts.SessionHistory != nil {
		if _, err := json.Marshal(ts.SessionHistory); err != nil {
			return fmt.Errorf("invalid session_history JSON: %w", err)
		}
	}

	return nil
}

// BeforeUpdate hook for TerminalSession model
func (ts *TerminalSession) BeforeUpdate(tx *gorm.DB) error {
	ts.LastActivity = time.Now()
	return nil
}

// Status and lifecycle methods

// IsActive checks if the terminal session is currently active
func (ts *TerminalSession) IsActive() bool {
	return ts.Status == TerminalStatusActive || ts.Status == TerminalStatusIdle
}

// IsExpired checks if the terminal session has expired
func (ts *TerminalSession) IsExpired() bool {
	return time.Now().After(ts.ExpiresAt) || ts.Status == TerminalStatusExpired
}

// IsConnected checks if the terminal session is connected
func (ts *TerminalSession) IsConnected() bool {
	return ts.Status == TerminalStatusActive || ts.Status == TerminalStatusIdle || ts.Status == TerminalStatusConnecting
}

// GetDuration returns the duration of the session
func (ts *TerminalSession) GetDuration() time.Duration {
	if ts.DisconnectedAt != nil {
		return ts.DisconnectedAt.Sub(ts.ConnectedAt)
	}
	return time.Since(ts.ConnectedAt)
}

// GetFormattedDuration returns formatted session duration
func (ts *TerminalSession) GetFormattedDuration() string {
	duration := ts.GetDuration()
	return formatDuration(duration)
}

// GetActivityStatus returns human-readable activity status
func (ts *TerminalSession) GetActivityStatus() string {
	if !ts.IsConnected() {
		return string(ts.Status)
	}

	idleTime := time.Since(ts.LastActivity)
	if idleTime > 30*time.Minute {
		return "idle (30+ min)"
	} else if idleTime > 5*time.Minute {
		return "idle (5+ min)"
	}
	return "active"
}

// Session management methods

// UpdateActivity updates the last activity timestamp
func (ts *TerminalSession) UpdateActivity() {
	ts.LastActivity = time.Now()
	// Extend expiry if session is active
	if ts.IsActive() {
		ts.ExpiresAt = time.Now().Add(time.Duration(ts.TimeoutSeconds) * time.Second)
	}
}

// UpdateStatus updates the session status and handles state transitions
func (ts *TerminalSession) UpdateStatus(status TerminalStatus) {
	now := time.Now()
	ts.Status = status
	ts.LastActivity = now

	switch status {
	case TerminalStatusDisconnected, TerminalStatusExpired, TerminalStatusError, TerminalStatusKilled:
		if ts.DisconnectedAt == nil {
			ts.DisconnectedAt = &now
		}
		if ts.SessionHistory != nil && ts.SessionHistory.EndTime == nil {
			ts.SessionHistory.EndTime = &now
		}
	case TerminalStatusActive:
		// Reset disconnection time if reconnecting
		ts.DisconnectedAt = nil
		ts.ExpiresAt = now.Add(time.Duration(ts.TimeoutSeconds) * time.Second)
	}
}

// AddCommand adds a command to the session history
func (ts *TerminalSession) AddCommand(command string, exitCode int, duration time.Duration, outputSize int) {
	if ts.SessionHistory == nil {
		ts.SessionHistory = &SessionHistory{
			StartTime:   ts.ConnectedAt,
			Commands:    make([]CommandEntry, 0),
			Errors:      make([]ErrorEntry, 0),
		}
	}

	entry := CommandEntry{
		Timestamp:   time.Now(),
		Command:     command,
		ExitCode:    exitCode,
		Duration:    duration.Milliseconds(),
		OutputSize:  outputSize,
	}

	ts.SessionHistory.Commands = append(ts.SessionHistory.Commands, entry)
	ts.SessionHistory.TotalCommands++
	ts.CommandCount++

	// Keep only last 100 commands to prevent unbounded growth
	if len(ts.SessionHistory.Commands) > 100 {
		ts.SessionHistory.Commands = ts.SessionHistory.Commands[len(ts.SessionHistory.Commands)-100:]
	}

	ts.UpdateActivity()
}

// AddError adds an error to the session history
func (ts *TerminalSession) AddError(error, context string) {
	if ts.SessionHistory == nil {
		ts.SessionHistory = &SessionHistory{
			StartTime:   ts.ConnectedAt,
			Commands:    make([]CommandEntry, 0),
			Errors:      make([]ErrorEntry, 0),
		}
	}

	entry := ErrorEntry{
		Timestamp: time.Now(),
		Error:     error,
		Context:   context,
	}

	ts.SessionHistory.Errors = append(ts.SessionHistory.Errors, entry)
	ts.ErrorCount++
	ts.LastError = error

	// Keep only last 50 errors
	if len(ts.SessionHistory.Errors) > 50 {
		ts.SessionHistory.Errors = ts.SessionHistory.Errors[len(ts.SessionHistory.Errors)-50:]
	}

	ts.UpdateActivity()
}

// UpdateTrafficStats updates bytes sent/received statistics
func (ts *TerminalSession) UpdateTrafficStats(bytesSent, bytesReceived uint64) {
	ts.BytesSent += bytesSent
	ts.BytesReceived += bytesReceived
	ts.UpdateActivity()
}

// ResizeTTY updates the TTY dimensions
func (ts *TerminalSession) ResizeTTY(columns, rows int) error {
	if columns < 1 || columns > 500 {
		return fmt.Errorf("invalid columns: %d", columns)
	}
	if rows < 1 || rows > 200 {
		return fmt.Errorf("invalid rows: %d", rows)
	}

	if ts.TTYSettings == nil {
		ts.TTYSettings = &TTYSettings{}
	}

	ts.TTYSettings.Columns = columns
	ts.TTYSettings.Rows = rows
	ts.UpdateActivity()

	return nil
}

// TerminateSession terminates the session with the given reason
func (ts *TerminalSession) TerminateSession(reason string, exitCode *int, signal string) {
	now := time.Now()
	ts.UpdateStatus(TerminalStatusKilled)
	ts.DisconnectedAt = &now
	ts.ExitCode = exitCode
	ts.SignalReceived = signal
	ts.LastError = reason
}

// Security and validation methods

// CanExecuteCommand checks if a command can be executed based on access control
func (ts *TerminalSession) CanExecuteCommand(command string) bool {
	if ts.Capabilities == nil {
		return true // Default allow all
	}

	if !ts.Capabilities.CanExecuteCommands {
		return false
	}

	// Check blocked commands
	var blockedCommands []string
	if ts.BlockedCommands != "" {
		json.Unmarshal([]byte(ts.BlockedCommands), &blockedCommands)
		for _, blocked := range blockedCommands {
			if blocked == command {
				return false
			}
		}
	}

	// Check allowed commands (if specified)
	var allowedCommands []string
	if ts.AllowedCommands != "" {
		json.Unmarshal([]byte(ts.AllowedCommands), &allowedCommands)
		if len(allowedCommands) > 0 {
			for _, allowed := range allowedCommands {
				if allowed == command {
					return true
				}
			}
			return false
		}
	}

	return true
}

// GetValidStatuses returns all valid terminal session statuses
func GetValidTerminalStatuses() []TerminalStatus {
	return []TerminalStatus{
		TerminalStatusConnecting,
		TerminalStatusActive,
		TerminalStatusIdle,
		TerminalStatusDisconnected,
		TerminalStatusExpired,
		TerminalStatusError,
		TerminalStatusKilled,
	}
}

// GetValidAccessLevels returns all valid access levels
func GetValidAccessLevels() []string {
	return []string{
		"read-only",
		"read-write",
		"restricted",
		"admin",
	}
}

// CleanupExpiredSessions returns a query to find expired sessions for cleanup
func CleanupExpiredSessions(db *gorm.DB) *gorm.DB {
	return db.Where("expires_at < ? OR status IN ?",
		time.Now(),
		[]string{string(TerminalStatusExpired), string(TerminalStatusDisconnected), string(TerminalStatusError)})
}

// formatDuration formats duration in human-readable format (defined in container.go)