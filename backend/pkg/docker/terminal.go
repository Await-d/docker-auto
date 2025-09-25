package docker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// TerminalManager manages Docker exec terminal sessions with WebSocket integration
type TerminalManager struct {
	client          *DockerClient
	sessions        map[string]*TerminalSession
	mu              sync.RWMutex
	config          *TerminalConfig
	metrics         *TerminalMetrics
	logger          *logrus.Entry
}

// TerminalSession represents an active Docker exec terminal session
type TerminalSession struct {
	ID            string                 `json:"id"`
	ContainerID   string                 `json:"container_id"`
	ContainerName string                 `json:"container_name"`
	ExecID        string                 `json:"exec_id"`
	StartTime     time.Time              `json:"start_time"`
	LastActivity  time.Time              `json:"last_activity"`
	IsActive      bool                   `json:"is_active"`
	Command       []string               `json:"command"`
	WorkDir       string                 `json:"work_dir"`
	Env           []string               `json:"env"`
	User          string                 `json:"user"`
	Privileged    bool                   `json:"privileged"`
	TTY           bool                   `json:"tty"`
	websocket     *websocket.Conn
	execConn      types.HijackedResponse
	cancel        context.CancelFunc
	mu            sync.RWMutex
}

// TerminalConfig configures terminal session behavior
type TerminalConfig struct {
	SessionTimeout    time.Duration `json:"session_timeout"`
	IdleTimeout       time.Duration `json:"idle_timeout"`
	MaxSessions       int           `json:"max_sessions"`
	BufferSize        int           `json:"buffer_size"`
	PingInterval      time.Duration `json:"ping_interval"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	EnableCompression bool          `json:"enable_compression"`
	AllowPrivileged   bool          `json:"allow_privileged"`
}

// TerminalMetrics tracks terminal session performance and usage
type TerminalMetrics struct {
	mu                  sync.RWMutex
	ActiveSessions      int64                    `json:"active_sessions"`
	TotalSessions       int64                    `json:"total_sessions"`
	SessionsExpired     int64                    `json:"sessions_expired"`
	CommandsExecuted    int64                    `json:"commands_executed"`
	BytesTransferred    int64                    `json:"bytes_transferred"`
	ErrorsEncountered   int64                    `json:"errors_encountered"`
	AverageSessionTime  time.Duration            `json:"average_session_time"`
	LastActivity        time.Time                `json:"last_activity"`
	SessionStats        map[string]*SessionStats `json:"session_stats"`
}

// ExecRequest represents a request to create a terminal exec session
type ExecRequest struct {
	ContainerID string   `json:"container_id" validate:"required"`
	Command     []string `json:"command"`
	WorkDir     string   `json:"work_dir"`
	Env         []string `json:"env"`
	User        string   `json:"user"`
	Privileged  bool     `json:"privileged"`
	TTY         bool     `json:"tty"`
	Detach      bool     `json:"detach"`
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string    `json:"type"`
	Data      string    `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Message types for WebSocket communication
const (
	MessageTypeCommand = "command"
	MessageTypeOutput  = "output"
	MessageTypeError   = "error"
	MessageTypeResize  = "resize"
	MessageTypePing    = "ping"
	MessageTypePong    = "pong"
	MessageTypeClose   = "close"
)

// DefaultTerminalConfig returns the default terminal configuration
func DefaultTerminalConfig() *TerminalConfig {
	return &TerminalConfig{
		SessionTimeout:    30 * time.Minute,
		IdleTimeout:       10 * time.Minute,
		MaxSessions:       100,
		BufferSize:        8192,
		PingInterval:      30 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       10 * time.Second,
		EnableCompression: true,
		AllowPrivileged:   false,
	}
}

// NewTerminalManager creates a new terminal manager with production-grade features
func NewTerminalManager(client *DockerClient, config *TerminalConfig) *TerminalManager {
	if config == nil {
		config = DefaultTerminalConfig()
	}

	manager := &TerminalManager{
		client:   client,
		sessions: make(map[string]*TerminalSession),
		config:   config,
		logger:   logrus.WithField("component", "terminal_manager"),
		metrics: &TerminalMetrics{
			SessionStats:  make(map[string]*SessionStats),
			LastActivity: time.Now(),
		},
	}

	// Start cleanup worker for expired sessions
	go manager.sessionCleanupWorker()

	manager.logger.WithFields(logrus.Fields{
		"session_timeout": config.SessionTimeout,
		"idle_timeout":    config.IdleTimeout,
		"max_sessions":    config.MaxSessions,
	}).Info("Terminal manager initialized")

	return manager
}

// CreateExecSession creates a new Docker exec session
func (tm *TerminalManager) CreateExecSession(ctx context.Context, req *ExecRequest) (*TerminalSession, error) {
	if req == nil {
		return nil, fmt.Errorf("exec request cannot be nil")
	}

	if req.ContainerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	// Validate session limits
	tm.mu.RLock()
	if len(tm.sessions) >= tm.config.MaxSessions {
		tm.mu.RUnlock()
		return nil, fmt.Errorf("maximum number of sessions reached (%d)", tm.config.MaxSessions)
	}
	tm.mu.RUnlock()

	// Security check for privileged access
	if req.Privileged && !tm.config.AllowPrivileged {
		return nil, fmt.Errorf("privileged exec sessions are not allowed")
	}

	// Get container information
	containerInfo, err := tm.client.GetContainer(ctx, req.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	if !containerInfo.State.Running {
		return nil, fmt.Errorf("container %s is not running", req.ContainerID)
	}

	// Set default command if not specified
	command := req.Command
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}

	// Create exec configuration
	execConfig := types.ExecConfig{
		User:         req.User,
		Privileged:   req.Privileged,
		Tty:          req.TTY,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Detach:       req.Detach,
		Cmd:          command,
		WorkingDir:   req.WorkDir,
		Env:          req.Env,
	}

	// Create exec instance
	execResp, err := tm.client.client.ContainerExecCreate(ctx, req.ContainerID, execConfig)
	if err != nil {
		tm.updateMetrics(func(m *TerminalMetrics) {
			m.ErrorsEncountered++
		})
		return nil, fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Generate unique session ID
	sessionID := generateSessionID()

	// Create session context with timeout
	_, cancel := context.WithTimeout(ctx, tm.config.SessionTimeout)

	// Create terminal session
	session := &TerminalSession{
		ID:            sessionID,
		ContainerID:   req.ContainerID,
		ContainerName: containerInfo.Name,
		ExecID:        execResp.ID,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		IsActive:      true,
		Command:       command,
		WorkDir:       req.WorkDir,
		Env:           req.Env,
		User:          req.User,
		Privileged:    req.Privileged,
		TTY:           req.TTY,
		cancel:        cancel,
	}

	// Store session
	tm.mu.Lock()
	tm.sessions[sessionID] = session
	tm.mu.Unlock()

	// Update metrics
	tm.updateMetrics(func(m *TerminalMetrics) {
		m.ActiveSessions++
		m.TotalSessions++
		m.LastActivity = time.Now()
		m.SessionStats[sessionID] = &SessionStats{
			ContainerID:  req.ContainerID,
			SessionStart: time.Now(),
			IsActive:     true,
		}
	})

	tm.logger.WithFields(logrus.Fields{
		"session_id":     sessionID,
		"container_id":   req.ContainerID,
		"container_name": containerInfo.Name,
		"command":        command,
	}).Info("Created exec session")

	return session, nil
}

// AttachWebSocket attaches a WebSocket connection to an existing exec session
func (tm *TerminalManager) AttachWebSocket(sessionID string, ws *websocket.Conn) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if ws == nil {
		return fmt.Errorf("websocket connection cannot be nil")
	}

	tm.mu.Lock()
	session, exists := tm.sessions[sessionID]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	tm.mu.Unlock()

	session.mu.Lock()
	if !session.IsActive {
		session.mu.Unlock()
		return fmt.Errorf("session %s is not active", sessionID)
	}

	session.websocket = ws
	session.mu.Unlock()

	// Configure WebSocket
	ws.SetReadDeadline(time.Now().Add(tm.config.ReadTimeout))
	ws.SetWriteDeadline(time.Now().Add(tm.config.WriteTimeout))

	if tm.config.EnableCompression {
		ws.EnableWriteCompression(true)
	}

	// Start WebSocket handlers
	go tm.handleWebSocketConnection(session)

	tm.logger.WithField("session_id", sessionID).Info("WebSocket attached to exec session")

	return nil
}

// StartExecSession starts the Docker exec session and connects I/O streams
func (tm *TerminalManager) StartExecSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	tm.mu.RLock()
	session, exists := tm.sessions[sessionID]
	if !exists {
		tm.mu.RUnlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	tm.mu.RUnlock()

	session.mu.RLock()
	if !session.IsActive {
		session.mu.RUnlock()
		return fmt.Errorf("session %s is not active", sessionID)
	}
	execID := session.ExecID
	session.mu.RUnlock()

	// Attach to exec instance
	attachOptions := types.ExecStartCheck{
		Detach: false,
		Tty:    session.TTY,
	}

	execConn, err := tm.client.client.ContainerExecAttach(ctx, execID, attachOptions)
	if err != nil {
		tm.updateMetrics(func(m *TerminalMetrics) {
			m.ErrorsEncountered++
		})
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}

	// Store the connection
	session.mu.Lock()
	session.execConn = execConn
	session.mu.Unlock()

	// Start the exec instance
	err = tm.client.client.ContainerExecStart(ctx, execID, types.ExecStartCheck{
		Detach: false,
		Tty:    session.TTY,
	})
	if err != nil {
		execConn.Close()
		tm.updateMetrics(func(m *TerminalMetrics) {
			m.ErrorsEncountered++
		})
		return fmt.Errorf("failed to start exec instance: %w", err)
	}

	// Start I/O handlers
	go tm.handleExecIO(session)

	tm.updateMetrics(func(m *TerminalMetrics) {
		m.CommandsExecuted++
		m.LastActivity = time.Now()
	})

	tm.logger.WithField("session_id", sessionID).Info("Started exec session")

	return nil
}

// handleWebSocketConnection handles WebSocket communication for a session
func (tm *TerminalManager) handleWebSocketConnection(session *TerminalSession) {
	defer func() {
		tm.logger.WithField("session_id", session.ID).Debug("WebSocket handler finished")
	}()

	// Set up ping/pong handling
	session.websocket.SetPongHandler(func(appData string) error {
		session.mu.Lock()
		session.LastActivity = time.Now()
		session.mu.Unlock()
		session.websocket.SetReadDeadline(time.Now().Add(tm.config.ReadTimeout))
		return nil
	})

	// Start ping goroutine
	go tm.pingHandler(session)

	for {
		session.websocket.SetReadDeadline(time.Now().Add(tm.config.ReadTimeout))

		var msg WebSocketMessage
		err := session.websocket.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				tm.logger.WithError(err).WithField("session_id", session.ID).Warn("Unexpected WebSocket close")
			}
			break
		}

		// Update activity timestamp
		session.mu.Lock()
		session.LastActivity = time.Now()
		session.mu.Unlock()

		// Handle different message types
		switch msg.Type {
		case MessageTypeCommand:
			tm.handleCommand(session, msg.Data)
		case MessageTypeResize:
			tm.handleResize(session, msg.Data)
		case MessageTypePong:
			// Already handled by pong handler
		case MessageTypeClose:
			tm.logger.WithField("session_id", session.ID).Info("Received close message")
			return
		default:
			tm.logger.WithFields(logrus.Fields{
				"session_id":   session.ID,
				"message_type": msg.Type,
			}).Warn("Unknown message type received")
		}
	}
}

// handleExecIO handles I/O streams between Docker exec and WebSocket
func (tm *TerminalManager) handleExecIO(session *TerminalSession) {
	defer func() {
		session.mu.Lock()
		if session.execConn.Conn != nil {
			session.execConn.Close()
		}
		session.mu.Unlock()
		tm.logger.WithField("session_id", session.ID).Debug("Exec I/O handler finished")
	}()

	session.mu.RLock()
	execConn := session.execConn
	ws := session.websocket
	session.mu.RUnlock()

	if ws == nil {
		tm.logger.WithField("session_id", session.ID).Error("WebSocket not attached")
		return
	}

	// Handle output from exec to WebSocket
	go func() {
		buffer := make([]byte, tm.config.BufferSize)
		for {
			n, err := execConn.Reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					tm.logger.WithError(err).WithField("session_id", session.ID).Error("Error reading from exec")
				}
				break
			}

			if n > 0 {
				msg := WebSocketMessage{
					Type:      MessageTypeOutput,
					Data:      string(buffer[:n]),
					Timestamp: time.Now(),
					SessionID: session.ID,
				}

				ws.SetWriteDeadline(time.Now().Add(tm.config.WriteTimeout))
				if err := ws.WriteJSON(msg); err != nil {
					tm.logger.WithError(err).WithField("session_id", session.ID).Error("Error writing to WebSocket")
					break
				}

				// Update metrics
				tm.updateMetrics(func(m *TerminalMetrics) {
					m.BytesTransferred += int64(n)
					m.LastActivity = time.Now()
				})

				session.mu.Lock()
				session.LastActivity = time.Now()
				session.mu.Unlock()
			}
		}
	}()
}

// handleCommand handles command input from WebSocket
func (tm *TerminalManager) handleCommand(session *TerminalSession, command string) {
	session.mu.RLock()
	execConn := session.execConn
	session.mu.RUnlock()

	if execConn.Conn == nil {
		tm.logger.WithField("session_id", session.ID).Error("Exec connection not available")
		return
	}

	// Write command to exec stdin
	_, err := execConn.Conn.Write([]byte(command))
	if err != nil {
		tm.logger.WithError(err).WithField("session_id", session.ID).Error("Error writing command to exec")

		// Send error to WebSocket
		if session.websocket != nil {
			errorMsg := WebSocketMessage{
				Type:      MessageTypeError,
				Error:     fmt.Sprintf("Failed to execute command: %v", err),
				Timestamp: time.Now(),
				SessionID: session.ID,
			}
			session.websocket.WriteJSON(errorMsg)
		}
		return
	}

	// Update metrics
	tm.updateMetrics(func(m *TerminalMetrics) {
		m.BytesTransferred += int64(len(command))
		m.CommandsExecuted++
		m.LastActivity = time.Now()
	})

	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()
}

// handleResize handles terminal resize requests
func (tm *TerminalManager) handleResize(session *TerminalSession, resizeData string) {
	// Parse resize data (expecting "cols,rows" format)
	// This is a simplified implementation - in production, you'd want proper JSON parsing
	tm.logger.WithFields(logrus.Fields{
		"session_id":   session.ID,
		"resize_data": resizeData,
	}).Debug("Terminal resize requested")

	// Update last activity
	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()
}

// pingHandler sends periodic ping messages to keep WebSocket alive
func (tm *TerminalManager) pingHandler(session *TerminalSession) {
	ticker := time.NewTicker(tm.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			session.mu.RLock()
			ws := session.websocket
			isActive := session.IsActive
			session.mu.RUnlock()

			if !isActive || ws == nil {
				return
			}

			ws.SetWriteDeadline(time.Now().Add(tm.config.WriteTimeout))
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				tm.logger.WithError(err).WithField("session_id", session.ID).Error("Error sending ping")
				return
			}
		}
	}
}

// CloseSession closes a terminal session and cleans up resources
func (tm *TerminalManager) CloseSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	tm.mu.Lock()
	session, exists := tm.sessions[sessionID]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(tm.sessions, sessionID)
	tm.mu.Unlock()

	// Close session resources
	session.mu.Lock()
	session.IsActive = false

	// Cancel context
	if session.cancel != nil {
		session.cancel()
	}

	// Close WebSocket
	if session.websocket != nil {
		session.websocket.Close()
	}

	// Close exec connection
	if session.execConn.Conn != nil {
		session.execConn.Close()
	}
	session.mu.Unlock()

	// Update metrics
	sessionDuration := time.Since(session.StartTime)
	tm.updateMetrics(func(m *TerminalMetrics) {
		m.ActiveSessions--
		if m.AverageSessionTime == 0 {
			m.AverageSessionTime = sessionDuration
		} else {
			m.AverageSessionTime = (m.AverageSessionTime + sessionDuration) / 2
		}
		if stats, exists := m.SessionStats[sessionID]; exists {
			stats.IsActive = false
		}
	})

	tm.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"duration":   sessionDuration,
	}).Info("Closed terminal session")

	return nil
}

// GetSession retrieves a terminal session by ID
func (tm *TerminalManager) GetSession(sessionID string) (*TerminalSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	tm.mu.RLock()
	session, exists := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

// ListActiveSessions returns all active terminal sessions
func (tm *TerminalManager) ListActiveSessions() []*TerminalSession {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	sessions := make([]*TerminalSession, 0, len(tm.sessions))
	for _, session := range tm.sessions {
		session.mu.RLock()
		if session.IsActive {
			sessions = append(sessions, session)
		}
		session.mu.RUnlock()
	}

	return sessions
}

// GetMetrics returns terminal manager metrics
func (tm *TerminalManager) GetMetrics() *TerminalMetrics {
	tm.metrics.mu.RLock()
	defer tm.metrics.mu.RUnlock()

	// Create deep copy of metrics
	metrics := &TerminalMetrics{
		ActiveSessions:     tm.metrics.ActiveSessions,
		TotalSessions:      tm.metrics.TotalSessions,
		SessionsExpired:    tm.metrics.SessionsExpired,
		CommandsExecuted:   tm.metrics.CommandsExecuted,
		BytesTransferred:   tm.metrics.BytesTransferred,
		ErrorsEncountered:  tm.metrics.ErrorsEncountered,
		AverageSessionTime: tm.metrics.AverageSessionTime,
		LastActivity:       tm.metrics.LastActivity,
		SessionStats:       make(map[string]*SessionStats),
	}

	// Copy session stats
	for k, v := range tm.metrics.SessionStats {
		metrics.SessionStats[k] = &SessionStats{
			ContainerID:  v.ContainerID,
			SessionStart: v.SessionStart,
			MetricsCount: v.MetricsCount,
			ErrorCount:   v.ErrorCount,
			LastUpdate:   v.LastUpdate,
			IsActive:     v.IsActive,
		}
	}

	return metrics
}

// sessionCleanupWorker periodically cleans up expired sessions
func (tm *TerminalManager) sessionCleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired and idle sessions
func (tm *TerminalManager) cleanupExpiredSessions() {
	now := time.Now()
	expiredSessions := make([]string, 0)

	tm.mu.RLock()
	for sessionID, session := range tm.sessions {
		session.mu.RLock()
		sessionAge := now.Sub(session.StartTime)
		idleTime := now.Sub(session.LastActivity)

		isExpired := sessionAge > tm.config.SessionTimeout ||
			idleTime > tm.config.IdleTimeout ||
			!session.IsActive

		session.mu.RUnlock()

		if isExpired {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	tm.mu.RUnlock()

	// Clean up expired sessions
	for _, sessionID := range expiredSessions {
		if err := tm.CloseSession(sessionID); err != nil {
			tm.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to close expired session")
		}

		tm.updateMetrics(func(m *TerminalMetrics) {
			m.SessionsExpired++
		})
	}

	if len(expiredSessions) > 0 {
		tm.logger.WithField("expired_count", len(expiredSessions)).Debug("Cleaned up expired terminal sessions")
	}
}

// updateMetrics safely updates metrics with a function
func (tm *TerminalManager) updateMetrics(updateFunc func(*TerminalMetrics)) {
	tm.metrics.mu.Lock()
	defer tm.metrics.mu.Unlock()
	updateFunc(tm.metrics)
}

// Close gracefully shuts down the terminal manager
func (tm *TerminalManager) Close() error {
	tm.mu.Lock()
	sessionIDs := make([]string, 0, len(tm.sessions))
	for sessionID := range tm.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	tm.mu.Unlock()

	// Close all active sessions
	for _, sessionID := range sessionIDs {
		if err := tm.CloseSession(sessionID); err != nil {
			tm.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to close session during shutdown")
		}
	}

	tm.logger.Info("Terminal manager shut down")
	return nil
}

// generateSessionID generates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("term_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}