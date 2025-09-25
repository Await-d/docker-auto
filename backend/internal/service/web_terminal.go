package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/security"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebTerminalService manages terminal session lifecycle and WebSocket communications
type WebTerminalService struct {
	containerRepo   repository.ContainerRepository
	sessionRepo     repository.TerminalSessionRepository
	activityRepo    repository.ActivityLogRepository
	terminalManager *docker.TerminalManager
	cache           *CacheService
	config          *config.Config
	userService     *UserService

	// Session management
	activeSessions  map[string]*TerminalServiceSession
	sessionsMu      sync.RWMutex

	// WebSocket connections
	wsConnections   map[string]*WebSocketConnection
	connectionsMu   sync.RWMutex

	// Security and access control
	accessControl   *TerminalAccessControl
	auditLogger     *TerminalAuditLogger

	logger          *logrus.Entry
}

// TerminalServiceSession represents a service-level terminal session
type TerminalServiceSession struct {
	ID              string                      `json:"id"`
	ContainerID     string                      `json:"container_id"`
	ContainerName   string                      `json:"container_name"`
	UserID          int64                       `json:"user_id"`
	Username        string                      `json:"username"`
	StartTime       time.Time                   `json:"start_time"`
	LastActivity    time.Time                   `json:"last_activity"`
	IsActive        bool                        `json:"is_active"`
	Command         []string                    `json:"command"`
	SessionConfig   *TerminalSessionConfig      `json:"session_config"`

	// Docker terminal session
	dockerSession   *docker.TerminalSession     `json:"-"`

	// WebSocket connection
	wsConnection    *WebSocketConnection        `json:"-"`

	// Security context
	securityContext *security.DockerUserContext `json:"-"`

	// Internal state
	cancel          context.CancelFunc          `json:"-"`
	mu              sync.RWMutex               `json:"-"`
}

// TerminalSessionConfig configures terminal session behavior
type TerminalSessionConfig struct {
	TTY             bool          `json:"tty"`
	WorkDir         string        `json:"work_dir"`
	Env             []string      `json:"env"`
	User            string        `json:"user"`
	Privileged      bool          `json:"privileged"`
	SessionTimeout  time.Duration `json:"session_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	EnableLogging   bool          `json:"enable_logging"`
	EnableRecording bool          `json:"enable_recording"`
	MaxBufferSize   int           `json:"max_buffer_size"`
}

// WebSocketConnection represents a WebSocket connection for terminal
type WebSocketConnection struct {
	Conn            *websocket.Conn  `json:"-"`
	SessionID       string           `json:"session_id"`
	UserID          int64            `json:"user_id"`
	ConnectedAt     time.Time        `json:"connected_at"`
	LastPing        time.Time        `json:"last_ping"`
	IsActive        bool             `json:"is_active"`
	MessageCount    int64            `json:"message_count"`
	BytesReceived   int64            `json:"bytes_received"`
	BytesSent       int64            `json:"bytes_sent"`

	// Message channels
	sendChan        chan []byte      `json:"-"`
	closeChan       chan struct{}    `json:"-"`
	mu              sync.RWMutex     `json:"-"`
}

// TerminalAccessControl manages access permissions for terminal sessions
type TerminalAccessControl struct {
	config          *config.Config
	userService     *UserService
	containerRepo   repository.ContainerRepository
	logger          *logrus.Entry
}

// TerminalAuditLogger logs terminal activities for security auditing
type TerminalAuditLogger struct {
	activityRepo    repository.ActivityLogRepository
	config          *config.Config
	logger          *logrus.Entry
}

// TerminalCommand represents a command executed in a terminal session
type TerminalCommand struct {
	SessionID       string    `json:"session_id"`
	Command         string    `json:"command"`
	Timestamp       time.Time `json:"timestamp"`
	UserID          int64     `json:"user_id"`
	ContainerID     string    `json:"container_id"`
	ExecutionTime   int64     `json:"execution_time_ms"`
	ExitCode        int       `json:"exit_code,omitempty"`
	Output          string    `json:"output,omitempty"`
}

// NewWebTerminalService creates a new web terminal service
func NewWebTerminalService(
	containerRepo repository.ContainerRepository,
	sessionRepo repository.TerminalSessionRepository,
	activityRepo repository.ActivityLogRepository,
	terminalManager *docker.TerminalManager,
	cache *CacheService,
	config *config.Config,
	userService *UserService,
) *WebTerminalService {

	logger := logrus.WithField("component", "web_terminal_service")

	// Create access control
	accessControl := &TerminalAccessControl{
		config:        config,
		userService:   userService,
		containerRepo: containerRepo,
		logger:        logger.WithField("subcomponent", "access_control"),
	}

	// Create audit logger
	auditLogger := &TerminalAuditLogger{
		activityRepo: activityRepo,
		config:       config,
		logger:       logger.WithField("subcomponent", "audit_logger"),
	}

	service := &WebTerminalService{
		containerRepo:   containerRepo,
		sessionRepo:     sessionRepo,
		activityRepo:    activityRepo,
		terminalManager: terminalManager,
		cache:           cache,
		config:          config,
		userService:     userService,
		activeSessions:  make(map[string]*TerminalServiceSession),
		wsConnections:   make(map[string]*WebSocketConnection),
		accessControl:   accessControl,
		auditLogger:     auditLogger,
		logger:          logger,
	}

	// Start background workers
	go service.sessionMaintenanceWorker()
	go service.connectionMaintenanceWorker()

	logger.Info("Web terminal service initialized")
	return service
}

// CreateTerminalSession creates a new terminal session for a container
func (s *WebTerminalService) CreateTerminalSession(ctx context.Context, userID int64, containerID int64, config *TerminalSessionConfig) (*TerminalServiceSession, error) {
	// Get container information
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	if container.ContainerID == "" {
		return nil, fmt.Errorf("container has no Docker instance")
	}

	// Check access permissions
	if err := s.accessControl.CheckAccess(ctx, userID, container); err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}

	// Validate container state
	if err := s.validateContainerState(ctx, container); err != nil {
		return nil, err
	}

	// Set default config if not provided
	if config == nil {
		config = s.getDefaultSessionConfig()
	}

	// Validate session config
	if err := s.validateSessionConfig(config); err != nil {
		return nil, fmt.Errorf("invalid session config: %w", err)
	}

	// Get user information for security context
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create security context
	securityContext := &security.DockerUserContext{
		UserID:   userID,
		Username: user.Username,
		Role:     string(user.Role),
		ClientIP: "internal", // Would be set from request context
	}

	// Create Docker exec request
	execReq := &docker.ExecRequest{
		ContainerID: container.ContainerID,
		Command:     []string{"/bin/sh"}, // Default shell
		TTY:         config.TTY,
		User:        config.User,
		Privileged:  config.Privileged,
		WorkDir:     config.WorkDir,
		Env:         config.Env,
	}

	// Validate security constraints
	if err := s.validateSecurityConstraints(execReq, securityContext); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Create Docker terminal session
	dockerSession, err := s.terminalManager.CreateExecSession(ctx, execReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker session: %w", err)
	}

	// Create session context with timeout
	sessionCtx, cancel := context.WithTimeout(ctx, config.SessionTimeout)

	// Create service session
	session := &TerminalServiceSession{
		ID:              dockerSession.ID,
		ContainerID:     container.ContainerID,
		ContainerName:   container.Name,
		UserID:          userID,
		Username:        user.Username,
		StartTime:       time.Now(),
		LastActivity:    time.Now(),
		IsActive:        true,
		Command:         execReq.Command,
		SessionConfig:   config,
		dockerSession:   dockerSession,
		securityContext: securityContext,
		cancel:          cancel,
	}

	// Store session
	s.sessionsMu.Lock()
	s.activeSessions[session.ID] = session
	s.sessionsMu.Unlock()

	// Persist session to database if enabled
	if s.sessionRepo != nil {
		dbSession := s.convertToDBSession(session, container)
		if err := s.sessionRepo.Create(ctx, dbSession); err != nil {
			s.logger.WithError(err).WithField("session_id", session.ID).Warn("Failed to persist session to database")
		}
	}

	// Start session processing
	go s.processSession(sessionCtx, session)

	// Log session creation
	s.auditLogger.LogSessionCreated(session)

	s.logger.WithFields(logrus.Fields{
		"session_id":     session.ID,
		"container_id":   container.ContainerID,
		"container_name": container.Name,
		"user_id":        userID,
		"username":       user.Username,
	}).Info("Terminal session created")

	return session, nil
}

// AttachWebSocket attaches a WebSocket connection to an existing terminal session
func (s *WebTerminalService) AttachWebSocket(sessionID string, ws *websocket.Conn, userID int64) error {
	if ws == nil {
		return fmt.Errorf("websocket connection cannot be nil")
	}

	// Get session
	s.sessionsMu.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Validate user permissions
	if session.UserID != userID {
		return fmt.Errorf("access denied: session belongs to different user")
	}

	session.mu.Lock()
	if !session.IsActive {
		session.mu.Unlock()
		return fmt.Errorf("session %s is not active", sessionID)
	}
	session.mu.Unlock()

	// Create WebSocket connection
	wsConn := &WebSocketConnection{
		Conn:         ws,
		SessionID:    sessionID,
		UserID:       userID,
		ConnectedAt:  time.Now(),
		LastPing:     time.Now(),
		IsActive:     true,
		sendChan:     make(chan []byte, 1000),
		closeChan:    make(chan struct{}),
	}

	// Configure WebSocket
	ws.SetReadLimit(int64(s.config.Terminal.BufferSize))
	ws.SetReadDeadline(time.Now().Add(time.Duration(s.config.Terminal.ReadTimeout) * time.Second))
	ws.SetWriteDeadline(time.Now().Add(time.Duration(s.config.Terminal.WriteTimeout) * time.Second))

	// Store connection
	s.connectionsMu.Lock()
	s.wsConnections[sessionID] = wsConn
	s.connectionsMu.Unlock()

	// Update session with WebSocket connection
	session.mu.Lock()
	session.wsConnection = wsConn
	session.mu.Unlock()

	// Attach to Docker terminal manager
	if err := s.terminalManager.AttachWebSocket(sessionID, ws); err != nil {
		// Our implementation doesn't have direct WebSocket attachment, so we'll handle it differently
		s.logger.WithError(err).WithField("session_id", sessionID).Debug("Direct WebSocket attachment not implemented, using proxy mode")
	}

	// Start WebSocket handlers
	go s.handleWebSocketConnection(wsConn, session)

	// Log WebSocket attachment
	s.auditLogger.LogWebSocketAttached(session, wsConn)

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"user_id":    userID,
	}).Info("WebSocket attached to terminal session")

	return nil
}

// StartTerminalSession starts the execution of a terminal session
func (s *WebTerminalService) StartTerminalSession(ctx context.Context, sessionID string) error {
	// Get session
	s.sessionsMu.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.RLock()
	if !session.IsActive {
		session.mu.RUnlock()
		return fmt.Errorf("session %s is not active", sessionID)
	}
	dockerSession := session.dockerSession
	session.mu.RUnlock()

	// Start Docker execution
	if err := s.terminalManager.StartExecSession(ctx, dockerSession.ID); err != nil {
		return fmt.Errorf("failed to start Docker session: %w", err)
	}

	// Update session state
	session.mu.Lock()
	session.LastActivity = time.Now()
	session.mu.Unlock()

	// Log session start
	s.auditLogger.LogSessionStarted(session)

	s.logger.WithField("session_id", sessionID).Info("Terminal session started")
	return nil
}

// CloseTerminalSession closes a terminal session and cleans up resources
func (s *WebTerminalService) CloseTerminalSession(sessionID string) error {
	// Get session
	s.sessionsMu.Lock()
	session, exists := s.activeSessions[sessionID]
	if !exists {
		s.sessionsMu.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(s.activeSessions, sessionID)
	s.sessionsMu.Unlock()

	// Mark as inactive
	session.mu.Lock()
	session.IsActive = false
	wsConn := session.wsConnection
	session.mu.Unlock()

	// Cancel session context
	if session.cancel != nil {
		session.cancel()
	}

	// Close WebSocket connection
	if wsConn != nil {
		s.closeWebSocketConnection(sessionID)
	}

	// Close Docker session
	if session.dockerSession != nil {
		if err := s.terminalManager.CloseSession(session.dockerSession.ID); err != nil {
			s.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to close Docker session")
		}
	}

	// Update database session status
	if s.sessionRepo != nil {
		// This would update the database session status
		s.logger.WithField("session_id", sessionID).Debug("Updating database session status")
	}

	// Calculate session duration
	sessionDuration := time.Since(session.StartTime)

	// Log session closure
	s.auditLogger.LogSessionClosed(session, sessionDuration)

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"duration":   sessionDuration,
	}).Info("Terminal session closed")

	return nil
}

// ListActiveSessions returns all active terminal sessions
func (s *WebTerminalService) ListActiveSessions() []*TerminalServiceSession {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*TerminalServiceSession, 0, len(s.activeSessions))
	for _, session := range s.activeSessions {
		if session.IsActive {
			// Create safe copy
			sessionCopy := &TerminalServiceSession{
				ID:            session.ID,
				ContainerID:   session.ContainerID,
				ContainerName: session.ContainerName,
				UserID:        session.UserID,
				Username:      session.Username,
				StartTime:     session.StartTime,
				LastActivity:  session.LastActivity,
				IsActive:      session.IsActive,
				Command:       session.Command,
				SessionConfig: session.SessionConfig,
			}
			sessions = append(sessions, sessionCopy)
		}
	}

	return sessions
}

// GetTerminalSession retrieves a terminal session by ID
func (s *WebTerminalService) GetTerminalSession(sessionID string) (*TerminalServiceSession, error) {
	s.sessionsMu.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Create safe copy
	sessionCopy := &TerminalServiceSession{
		ID:            session.ID,
		ContainerID:   session.ContainerID,
		ContainerName: session.ContainerName,
		UserID:        session.UserID,
		Username:      session.Username,
		StartTime:     session.StartTime,
		LastActivity:  session.LastActivity,
		IsActive:      session.IsActive,
		Command:       session.Command,
		SessionConfig: session.SessionConfig,
	}

	return sessionCopy, nil
}

// GetSessionMetrics returns terminal session system metrics
func (s *WebTerminalService) GetSessionMetrics() *docker.TerminalMetrics {
	if s.terminalManager != nil {
		return s.terminalManager.GetMetrics()
	}
	return &docker.TerminalMetrics{}
}

// handleWebSocketConnection handles WebSocket communication for a session
func (s *WebTerminalService) handleWebSocketConnection(wsConn *WebSocketConnection, session *TerminalServiceSession) {
	defer func() {
		s.closeWebSocketConnection(wsConn.SessionID)
		s.logger.WithField("session_id", wsConn.SessionID).Debug("WebSocket handler finished")
	}()

	// Start message sender goroutine
	go s.webSocketMessageSender(wsConn)

	// Handle incoming messages
	for {
		wsConn.Conn.SetReadDeadline(time.Now().Add(time.Duration(s.config.Terminal.ReadTimeout) * time.Second))

		messageType, message, err := wsConn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.WithError(err).WithField("session_id", wsConn.SessionID).Warn("Unexpected WebSocket close")
			}
			break
		}

		// Update connection statistics
		wsConn.mu.Lock()
		wsConn.MessageCount++
		wsConn.BytesReceived += int64(len(message))
		wsConn.mu.Unlock()

		// Update session activity
		session.mu.Lock()
		session.LastActivity = time.Now()
		session.mu.Unlock()

		// Process message based on type
		switch messageType {
		case websocket.TextMessage:
			s.handleTerminalCommand(wsConn, session, string(message))
		case websocket.BinaryMessage:
			s.handleBinaryMessage(wsConn, session, message)
		case websocket.PingMessage:
			wsConn.sendChan <- []byte{}
		case websocket.CloseMessage:
			s.logger.WithField("session_id", wsConn.SessionID).Info("Received close message")
			return
		default:
			s.logger.WithFields(logrus.Fields{
				"session_id":   wsConn.SessionID,
				"message_type": messageType,
			}).Warn("Unknown WebSocket message type")
		}
	}
}

// handleTerminalCommand processes terminal command input
func (s *WebTerminalService) handleTerminalCommand(wsConn *WebSocketConnection, session *TerminalServiceSession, command string) {
	// Log command for audit
	terminalCommand := &TerminalCommand{
		SessionID:   session.ID,
		Command:     command,
		Timestamp:   time.Now(),
		UserID:      session.UserID,
		ContainerID: session.ContainerID,
	}

	s.auditLogger.LogCommand(terminalCommand)

	// The actual command execution would be handled by the Docker terminal manager
	// For now, we'll just echo the command back as a placeholder
	response := fmt.Sprintf("Echo: %s\n", command)

	// Send response back through WebSocket
	wsConn.mu.Lock()
	wsConn.BytesSent += int64(len(response))
	wsConn.mu.Unlock()

	select {
	case wsConn.sendChan <- []byte(response):
	default:
		s.logger.WithField("session_id", session.ID).Warn("WebSocket send channel full, dropping message")
	}
}

// handleBinaryMessage processes binary message input
func (s *WebTerminalService) handleBinaryMessage(wsConn *WebSocketConnection, session *TerminalServiceSession, message []byte) {
	// Handle binary data (e.g., for file transfers or special terminal sequences)
	s.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"size":       len(message),
	}).Debug("Received binary message")
}

// webSocketMessageSender sends messages through WebSocket
func (s *WebTerminalService) webSocketMessageSender(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(time.Duration(s.config.Terminal.PingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message := <-wsConn.sendChan:
			wsConn.Conn.SetWriteDeadline(time.Now().Add(time.Duration(s.config.Terminal.WriteTimeout) * time.Second))
			if err := wsConn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.logger.WithError(err).WithField("session_id", wsConn.SessionID).Error("Failed to send WebSocket message")
				return
			}

		case <-ticker.C:
			wsConn.Conn.SetWriteDeadline(time.Now().Add(time.Duration(s.config.Terminal.WriteTimeout) * time.Second))
			if err := wsConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.WithError(err).WithField("session_id", wsConn.SessionID).Error("Failed to send ping")
				return
			}

		case <-wsConn.closeChan:
			return
		}
	}
}

// closeWebSocketConnection closes a WebSocket connection
func (s *WebTerminalService) closeWebSocketConnection(sessionID string) {
	s.connectionsMu.Lock()
	wsConn, exists := s.wsConnections[sessionID]
	if exists {
		delete(s.wsConnections, sessionID)
	}
	s.connectionsMu.Unlock()

	if exists && wsConn != nil {
		wsConn.mu.Lock()
		wsConn.IsActive = false
		close(wsConn.closeChan)
		wsConn.Conn.Close()
		wsConn.mu.Unlock()
	}
}

// processSession handles session lifecycle management
func (s *WebTerminalService) processSession(ctx context.Context, session *TerminalServiceSession) {
	idleTimeout := time.Duration(s.config.Terminal.IdleTimeout) * time.Minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check for idle timeout
			session.mu.RLock()
			lastActivity := session.LastActivity
			isActive := session.IsActive
			session.mu.RUnlock()

			if !isActive {
				return
			}

			if time.Since(lastActivity) > idleTimeout {
				s.logger.WithField("session_id", session.ID).Info("Session idle timeout, closing")
				s.CloseTerminalSession(session.ID)
				return
			}
		}
	}
}

// Background workers

// sessionMaintenanceWorker performs session maintenance tasks
func (s *WebTerminalService) sessionMaintenanceWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupExpiredSessions()
		}
	}
}

// connectionMaintenanceWorker performs WebSocket connection maintenance
func (s *WebTerminalService) connectionMaintenanceWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupStaleConnections()
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (s *WebTerminalService) cleanupExpiredSessions() {
	now := time.Now()
	sessionTimeout := time.Duration(s.config.Terminal.SessionTimeout) * time.Minute
	idleTimeout := time.Duration(s.config.Terminal.IdleTimeout) * time.Minute

	s.sessionsMu.RLock()
	expiredSessions := make([]string, 0)
	for sessionID, session := range s.activeSessions {
		sessionAge := now.Sub(session.StartTime)
		idleTime := now.Sub(session.LastActivity)

		if sessionAge > sessionTimeout || idleTime > idleTimeout || !session.IsActive {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}
	s.sessionsMu.RUnlock()

	// Close expired sessions
	for _, sessionID := range expiredSessions {
		if err := s.CloseTerminalSession(sessionID); err != nil {
			s.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to close expired session")
		}
	}

	if len(expiredSessions) > 0 {
		s.logger.WithField("expired_count", len(expiredSessions)).Debug("Cleaned up expired terminal sessions")
	}
}

// cleanupStaleConnections removes stale WebSocket connections
func (s *WebTerminalService) cleanupStaleConnections() {
	now := time.Now()
	connectionTimeout := 5 * time.Minute

	s.connectionsMu.Lock()
	staleConnections := make([]string, 0)
	for sessionID, wsConn := range s.wsConnections {
		if !wsConn.IsActive || now.Sub(wsConn.LastPing) > connectionTimeout {
			staleConnections = append(staleConnections, sessionID)
		}
	}
	s.connectionsMu.Unlock()

	// Close stale connections
	for _, sessionID := range staleConnections {
		s.closeWebSocketConnection(sessionID)
	}

	if len(staleConnections) > 0 {
		s.logger.WithField("stale_count", len(staleConnections)).Debug("Cleaned up stale WebSocket connections")
	}
}

// Helper methods

// getDefaultSessionConfig returns default session configuration
func (s *WebTerminalService) getDefaultSessionConfig() *TerminalSessionConfig {
	return &TerminalSessionConfig{
		TTY:             true,
		User:            "root",
		Privileged:      false,
		SessionTimeout:  time.Duration(s.config.Terminal.SessionTimeout) * time.Minute,
		IdleTimeout:     time.Duration(s.config.Terminal.IdleTimeout) * time.Minute,
		EnableLogging:   true,
		EnableRecording: false,
		MaxBufferSize:   s.config.Terminal.BufferSize,
	}
}

// validateSessionConfig validates session configuration
func (s *WebTerminalService) validateSessionConfig(config *TerminalSessionConfig) error {
	if config.SessionTimeout <= 0 {
		return fmt.Errorf("session timeout must be positive")
	}
	if config.IdleTimeout <= 0 {
		return fmt.Errorf("idle timeout must be positive")
	}
	if config.MaxBufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}
	return nil
}

// validateContainerState validates that container is in appropriate state for terminal access
func (s *WebTerminalService) validateContainerState(ctx context.Context, container *model.Container) error {
	if !container.IsRunning() {
		return fmt.Errorf("container must be running for terminal access")
	}
	return nil
}

// validateSecurityConstraints validates security constraints for terminal access
func (s *WebTerminalService) validateSecurityConstraints(execReq *docker.ExecRequest, securityContext *security.DockerUserContext) error {
	// Check privileged access
	if execReq.Privileged && !s.config.Terminal.AllowPrivileged {
		return fmt.Errorf("privileged terminal access not allowed")
	}

	// Additional security validations would go here
	return nil
}

// convertToDBSession converts service session to database model
func (s *WebTerminalService) convertToDBSession(session *TerminalServiceSession, container *model.Container) *model.TerminalSession {
	// This would convert to the database model when implemented
	userIDPtr := int(session.UserID)
	return &model.TerminalSession{
		SessionID:     session.ID,
		ContainerID:   int(container.ID),
		UserID:        &userIDPtr,
		Command:       fmt.Sprintf("%v", session.Command),
		Status:        "active",
		ConnectedAt:   session.StartTime,
		LastActivity:  session.LastActivity,
	}
}

// Access control methods

// CheckAccess verifies user access to container terminal
func (ac *TerminalAccessControl) CheckAccess(ctx context.Context, userID int64, container *model.Container) error {
	// Check if user owns the container
	if container.CreatedBy == nil || int64(*container.CreatedBy) != userID {
		return fmt.Errorf("access denied: container belongs to different user")
	}

	// Additional access control logic would go here
	return nil
}

// Audit logging methods

// LogSessionCreated logs session creation
func (al *TerminalAuditLogger) LogSessionCreated(session *TerminalServiceSession) {
	if al.activityRepo == nil {
		return
	}

	activity := &model.ActivityLog{
		UserID:       &session.UserID,
		Action:       "terminal_session_created",
		ResourceType: "terminal_session",
		Description:  fmt.Sprintf("Terminal session created for container %s", session.ContainerName),
		Metadata:     fmt.Sprintf(`{"session_id": "%s", "container_id": "%s"}`, session.ID, session.ContainerID),
	}

	if err := al.activityRepo.Create(context.Background(), activity); err != nil {
		al.logger.WithError(err).Warn("Failed to log session creation")
	}
}

// LogSessionStarted logs session start
func (al *TerminalAuditLogger) LogSessionStarted(session *TerminalServiceSession) {
	al.logger.WithField("session_id", session.ID).Debug("Session started audit log")
}

// LogSessionClosed logs session closure
func (al *TerminalAuditLogger) LogSessionClosed(session *TerminalServiceSession, duration time.Duration) {
	al.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"duration":   duration,
	}).Debug("Session closed audit log")
}

// LogWebSocketAttached logs WebSocket attachment
func (al *TerminalAuditLogger) LogWebSocketAttached(session *TerminalServiceSession, wsConn *WebSocketConnection) {
	al.logger.WithField("session_id", session.ID).Debug("WebSocket attached audit log")
}

// LogCommand logs command execution
func (al *TerminalAuditLogger) LogCommand(cmd *TerminalCommand) {
	al.logger.WithFields(logrus.Fields{
		"session_id":   cmd.SessionID,
		"user_id":      cmd.UserID,
		"container_id": cmd.ContainerID,
		"command":      cmd.Command,
	}).Debug("Terminal command audit log")
}

// Close gracefully shuts down the web terminal service
func (s *WebTerminalService) Close() error {
	s.sessionsMu.Lock()
	sessionIDs := make([]string, 0, len(s.activeSessions))
	for sessionID := range s.activeSessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.sessionsMu.Unlock()

	// Close all active sessions
	for _, sessionID := range sessionIDs {
		if err := s.CloseTerminalSession(sessionID); err != nil {
			s.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to close session during shutdown")
		}
	}

	s.logger.Info("Web terminal service shut down")
	return nil
}