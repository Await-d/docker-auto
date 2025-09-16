package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketSecurityConfig represents WebSocket security configuration
type WebSocketSecurityConfig struct {
	// Authentication
	RequireAuth         bool          `json:"require_auth"`
	TokenValidation     bool          `json:"token_validation"`
	SessionValidation   bool          `json:"session_validation"`

	// Origin validation
	OriginValidation    bool          `json:"origin_validation"`
	AllowedOrigins      []string      `json:"allowed_origins"`
	StrictOriginCheck   bool          `json:"strict_origin_check"`

	// Connection limits
	MaxConnections      int           `json:"max_connections"`
	MaxConnectionsPerIP int           `json:"max_connections_per_ip"`
	MaxConnectionsPerUser int         `json:"max_connections_per_user"`

	// Message validation
	MessageValidation   bool          `json:"message_validation"`
	MaxMessageSize      int64         `json:"max_message_size"`
	MaxMessagesPerSecond int          `json:"max_messages_per_second"`
	AllowedMessageTypes []string      `json:"allowed_message_types"`

	// Rate limiting
	EnableRateLimit     bool          `json:"enable_rate_limit"`
	MessagesPerMinute   int           `json:"messages_per_minute"`
	BurstLimit          int           `json:"burst_limit"`

	// Security features
	HeartbeatInterval   time.Duration `json:"heartbeat_interval"`
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	IdleTimeout         time.Duration `json:"idle_timeout"`
	EnableCompression   bool          `json:"enable_compression"`
	SubProtocols        []string      `json:"sub_protocols"`

	// Monitoring
	EnableLogging       bool          `json:"enable_logging"`
	LogAllMessages      bool          `json:"log_all_messages"`
	MonitorConnections  bool          `json:"monitor_connections"`
}

// DefaultWebSocketSecurityConfig returns secure default configuration
func DefaultWebSocketSecurityConfig() *WebSocketSecurityConfig {
	return &WebSocketSecurityConfig{
		RequireAuth:           true,
		TokenValidation:       true,
		SessionValidation:     true,
		OriginValidation:      true,
		AllowedOrigins:        []string{"http://localhost:3000", "https://localhost:3000"},
		StrictOriginCheck:     true,
		MaxConnections:        1000,
		MaxConnectionsPerIP:   10,
		MaxConnectionsPerUser: 5,
		MessageValidation:     true,
		MaxMessageSize:        64 * 1024, // 64KB
		MaxMessagesPerSecond:  10,
		AllowedMessageTypes:   []string{"ping", "pong", "data", "command", "subscribe", "unsubscribe"},
		EnableRateLimit:       true,
		MessagesPerMinute:     60,
		BurstLimit:           10,
		HeartbeatInterval:    30 * time.Second,
		ConnectionTimeout:    60 * time.Second,
		IdleTimeout:          300 * time.Second, // 5 minutes
		EnableCompression:    false, // Disable to prevent compression attacks
		SubProtocols:         []string{"v1.docker-auto.protocol"},
		EnableLogging:        true,
		LogAllMessages:       false,
		MonitorConnections:   true,
	}
}

// WebSocketMessage represents a WebSocket message structure
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id,omitempty"`
}

// ConnectionInfo represents WebSocket connection information
type ConnectionInfo struct {
	ID            string                 `json:"id"`
	UserID        int64                  `json:"user_id"`
	Username      string                 `json:"username"`
	IP            string                 `json:"ip"`
	UserAgent     string                 `json:"user_agent"`
	Origin        string                 `json:"origin"`
	ConnectedAt   time.Time              `json:"connected_at"`
	LastActivity  time.Time              `json:"last_activity"`
	MessageCount  int64                  `json:"message_count"`
	SecurityLevel int                    `json:"security_level"`
	SessionID     string                 `json:"session_id"`
	Connection    *websocket.Conn        `json:"-"`
	Claims        *EnhancedClaims        `json:"-"`
	RateLimit     *ConnectionRateLimit   `json:"-"`
}

// ConnectionRateLimit represents rate limiting for a WebSocket connection
type ConnectionRateLimit struct {
	MessageCount   int       `json:"message_count"`
	WindowStart    time.Time `json:"window_start"`
	LastMessage    time.Time `json:"last_message"`
	ViolationCount int       `json:"violation_count"`
	Blocked        bool      `json:"blocked"`
}

// WebSocketSecurityManager manages WebSocket security
type WebSocketSecurityManager struct {
	config       *WebSocketSecurityConfig
	jwtManager   *EnhancedJWTManager
	connections  map[string]*ConnectionInfo
	ipConnections map[string][]string // IP -> ConnectionIDs
	userConnections map[int64][]string // UserID -> ConnectionIDs
	upgrader     websocket.Upgrader
	mutex        sync.RWMutex
	stats        *WebSocketStats
}

// WebSocketStats represents WebSocket statistics
type WebSocketStats struct {
	TotalConnections    int64     `json:"total_connections"`
	ActiveConnections   int       `json:"active_connections"`
	TotalMessages       int64     `json:"total_messages"`
	BlockedMessages     int64     `json:"blocked_messages"`
	RateLimitViolations int64     `json:"rate_limit_violations"`
	AuthFailures        int64     `json:"auth_failures"`
	OriginViolations    int64     `json:"origin_violations"`
	LastUpdate          time.Time `json:"last_update"`
}

// NewWebSocketSecurityManager creates a new WebSocket security manager
func NewWebSocketSecurityManager(config *WebSocketSecurityConfig, jwtManager *EnhancedJWTManager) *WebSocketSecurityManager {
	if config == nil {
		config = DefaultWebSocketSecurityConfig()
	}

	wsm := &WebSocketSecurityManager{
		config:          config,
		jwtManager:      jwtManager,
		connections:     make(map[string]*ConnectionInfo),
		ipConnections:   make(map[string][]string),
		userConnections: make(map[int64][]string),
		stats:           &WebSocketStats{LastUpdate: time.Now()},
	}

	// Configure WebSocket upgrader
	wsm.upgrader = websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  config.ConnectionTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin:       wsm.checkOrigin,
		Subprotocols:      config.SubProtocols,
	}

	// Start monitoring goroutines
	wsm.startHeartbeat()
	wsm.startCleanup()

	return wsm
}

// UpgradeConnection upgrades an HTTP connection to WebSocket with security checks
func (wsm *WebSocketSecurityManager) UpgradeConnection(w http.ResponseWriter, r *http.Request) (*ConnectionInfo, error) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	// Validate origin
	if wsm.config.OriginValidation {
		if !wsm.isValidOrigin(r.Header.Get("Origin")) {
			wsm.stats.OriginViolations++
			return nil, fmt.Errorf("origin not allowed")
		}
	}

	// Check connection limits
	clientIP := getClientIPFromRequest(r)
	if err := wsm.checkConnectionLimits(clientIP, 0); err != nil {
		return nil, fmt.Errorf("connection limit exceeded: %w", err)
	}

	// Authenticate if required
	var claims *EnhancedClaims
	if wsm.config.RequireAuth {
		token := extractTokenFromRequest(r)
		if token == "" {
			wsm.stats.AuthFailures++
			return nil, fmt.Errorf("authentication required")
		}

		var err error
		claims, err = wsm.jwtManager.ValidateToken(token, clientIP, r.Header.Get("User-Agent"))
		if err != nil {
			wsm.stats.AuthFailures++
			return nil, fmt.Errorf("authentication failed: %w", err)
		}

		// Check user connection limits
		if err := wsm.checkConnectionLimits(clientIP, claims.UserID); err != nil {
			return nil, fmt.Errorf("user connection limit exceeded: %w", err)
		}
	}

	// Upgrade connection
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket upgrade failed: %w", err)
	}

	// Create connection info
	connID := generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:            connID,
		IP:            clientIP,
		UserAgent:     r.Header.Get("User-Agent"),
		Origin:        r.Header.Get("Origin"),
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Connection:    conn,
		Claims:        claims,
		RateLimit:     &ConnectionRateLimit{WindowStart: time.Now()},
	}

	if claims != nil {
		connInfo.UserID = claims.UserID
		connInfo.Username = claims.Username
		connInfo.SecurityLevel = claims.SecurityLevel
		connInfo.SessionID = claims.SessionID
	}

	// Store connection
	wsm.connections[connID] = connInfo
	wsm.ipConnections[clientIP] = append(wsm.ipConnections[clientIP], connID)
	if connInfo.UserID > 0 {
		wsm.userConnections[connInfo.UserID] = append(wsm.userConnections[connInfo.UserID], connID)
	}

	// Update stats
	wsm.stats.TotalConnections++
	wsm.stats.ActiveConnections = len(wsm.connections)

	logrus.WithFields(logrus.Fields{
		"connection_id": connID,
		"ip":           clientIP,
		"user_id":      connInfo.UserID,
		"origin":       connInfo.Origin,
	}).Info("WebSocket connection established")

	return connInfo, nil
}

// HandleMessage processes and validates incoming WebSocket messages
func (wsm *WebSocketSecurityManager) HandleMessage(connInfo *ConnectionInfo, messageType int, data []byte) error {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	// Update activity
	connInfo.LastActivity = time.Now()
	connInfo.MessageCount++
	wsm.stats.TotalMessages++

	// Validate message size
	if wsm.config.MaxMessageSize > 0 && int64(len(data)) > wsm.config.MaxMessageSize {
		wsm.stats.BlockedMessages++
		return fmt.Errorf("message too large: %d bytes, max: %d", len(data), wsm.config.MaxMessageSize)
	}

	// Check rate limiting
	if wsm.config.EnableRateLimit {
		if !wsm.checkMessageRateLimit(connInfo) {
			wsm.stats.RateLimitViolations++
			wsm.stats.BlockedMessages++
			return fmt.Errorf("rate limit exceeded")
		}
	}

	// Validate message format if enabled
	if wsm.config.MessageValidation {
		if err := wsm.validateMessage(data); err != nil {
			wsm.stats.BlockedMessages++
			return fmt.Errorf("message validation failed: %w", err)
		}
	}

	// Log message if enabled
	if wsm.config.EnableLogging && wsm.config.LogAllMessages {
		logrus.WithFields(logrus.Fields{
			"connection_id": connInfo.ID,
			"user_id":       connInfo.UserID,
			"message_type":  messageType,
			"message_size":  len(data),
		}).Debug("WebSocket message received")
	}

	return nil
}

// CloseConnection closes a WebSocket connection
func (wsm *WebSocketSecurityManager) CloseConnection(connID string, reason string) error {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	connInfo, exists := wsm.connections[connID]
	if !exists {
		return fmt.Errorf("connection not found")
	}

	// Close WebSocket connection
	if connInfo.Connection != nil {
		connInfo.Connection.Close()
	}

	// Remove from tracking
	delete(wsm.connections, connID)

	// Remove from IP tracking
	if ipConns, exists := wsm.ipConnections[connInfo.IP]; exists {
		for i, id := range ipConns {
			if id == connID {
				wsm.ipConnections[connInfo.IP] = append(ipConns[:i], ipConns[i+1:]...)
				break
			}
		}
		if len(wsm.ipConnections[connInfo.IP]) == 0 {
			delete(wsm.ipConnections, connInfo.IP)
		}
	}

	// Remove from user tracking
	if connInfo.UserID > 0 {
		if userConns, exists := wsm.userConnections[connInfo.UserID]; exists {
			for i, id := range userConns {
				if id == connID {
					wsm.userConnections[connInfo.UserID] = append(userConns[:i], userConns[i+1:]...)
					break
				}
			}
			if len(wsm.userConnections[connInfo.UserID]) == 0 {
				delete(wsm.userConnections, connInfo.UserID)
			}
		}
	}

	// Update stats
	wsm.stats.ActiveConnections = len(wsm.connections)

	logrus.WithFields(logrus.Fields{
		"connection_id": connID,
		"user_id":       connInfo.UserID,
		"ip":            connInfo.IP,
		"reason":        reason,
		"duration":      time.Since(connInfo.ConnectedAt),
		"message_count": connInfo.MessageCount,
	}).Info("WebSocket connection closed")

	return nil
}

// checkOrigin validates the origin header for WebSocket connections
func (wsm *WebSocketSecurityManager) checkOrigin(r *http.Request) bool {
	if !wsm.config.OriginValidation {
		return true
	}

	origin := r.Header.Get("Origin")
	return wsm.isValidOrigin(origin)
}

// isValidOrigin checks if an origin is allowed
func (wsm *WebSocketSecurityManager) isValidOrigin(origin string) bool {
	if origin == "" && !wsm.config.StrictOriginCheck {
		return true
	}

	for _, allowed := range wsm.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if len(allowed) > 2 && allowed[:2] == "*." {
			domain := allowed[2:]
			if len(origin) > len(domain) && origin[len(origin)-len(domain):] == domain {
				return true
			}
		}
	}

	return false
}

// checkConnectionLimits validates connection limits
func (wsm *WebSocketSecurityManager) checkConnectionLimits(ip string, userID int64) error {
	// Check global connection limit
	if wsm.config.MaxConnections > 0 && len(wsm.connections) >= wsm.config.MaxConnections {
		return fmt.Errorf("global connection limit reached")
	}

	// Check per-IP connection limit
	if wsm.config.MaxConnectionsPerIP > 0 {
		if ipConns, exists := wsm.ipConnections[ip]; exists && len(ipConns) >= wsm.config.MaxConnectionsPerIP {
			return fmt.Errorf("IP connection limit reached")
		}
	}

	// Check per-user connection limit
	if userID > 0 && wsm.config.MaxConnectionsPerUser > 0 {
		if userConns, exists := wsm.userConnections[userID]; exists && len(userConns) >= wsm.config.MaxConnectionsPerUser {
			return fmt.Errorf("user connection limit reached")
		}
	}

	return nil
}

// checkMessageRateLimit validates message rate limits
func (wsm *WebSocketSecurityManager) checkMessageRateLimit(connInfo *ConnectionInfo) bool {
	now := time.Now()
	rateLimit := connInfo.RateLimit

	// Check window expiration
	if now.Sub(rateLimit.WindowStart) >= time.Minute {
		rateLimit.MessageCount = 0
		rateLimit.WindowStart = now
	}

	// Check burst limit
	if wsm.config.BurstLimit > 0 {
		timeSinceLastMessage := now.Sub(rateLimit.LastMessage)
		if timeSinceLastMessage < time.Second && rateLimit.MessageCount >= wsm.config.BurstLimit {
			return false
		}
	}

	// Check per-minute limit
	if wsm.config.MessagesPerMinute > 0 && rateLimit.MessageCount >= wsm.config.MessagesPerMinute {
		return false
	}

	// Allow message
	rateLimit.MessageCount++
	rateLimit.LastMessage = now
	return true
}

// validateMessage validates the structure and content of WebSocket messages
func (wsm *WebSocketSecurityManager) validateMessage(data []byte) error {
	// Try to parse as JSON
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate message type
	if len(wsm.config.AllowedMessageTypes) > 0 {
		typeAllowed := false
		for _, allowedType := range wsm.config.AllowedMessageTypes {
			if msg.Type == allowedType {
				typeAllowed = true
				break
			}
		}
		if !typeAllowed {
			return fmt.Errorf("message type '%s' not allowed", msg.Type)
		}
	}

	// Additional validation based on message type
	switch msg.Type {
	case "command":
		// Validate command messages more strictly
		if msg.Data == nil {
			return fmt.Errorf("command message must have data")
		}
	case "subscribe", "unsubscribe":
		// Validate subscription messages
		if msg.Data == nil {
			return fmt.Errorf("subscription message must have data")
		}
	}

	return nil
}

// startHeartbeat starts the heartbeat mechanism
func (wsm *WebSocketSecurityManager) startHeartbeat() {
	if wsm.config.HeartbeatInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(wsm.config.HeartbeatInterval)
		defer ticker.Stop()

		for range ticker.C {
			wsm.sendHeartbeat()
		}
	}()
}

// sendHeartbeat sends heartbeat messages to all connections
func (wsm *WebSocketSecurityManager) sendHeartbeat() {
	wsm.mutex.RLock()
	connections := make([]*ConnectionInfo, 0, len(wsm.connections))
	for _, conn := range wsm.connections {
		connections = append(connections, conn)
	}
	wsm.mutex.RUnlock()

	heartbeatMsg := WebSocketMessage{
		Type:      "ping",
		Timestamp: time.Now(),
	}

	for _, connInfo := range connections {
		if connInfo.Connection != nil {
			connInfo.Connection.WriteJSON(heartbeatMsg)
		}
	}
}

// startCleanup starts the cleanup mechanism for idle connections
func (wsm *WebSocketSecurityManager) startCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			wsm.cleanupIdleConnections()
		}
	}()
}

// cleanupIdleConnections removes idle connections
func (wsm *WebSocketSecurityManager) cleanupIdleConnections() {
	if wsm.config.IdleTimeout <= 0 {
		return
	}

	wsm.mutex.RLock()
	idleConnections := make([]string, 0)
	now := time.Now()

	for connID, connInfo := range wsm.connections {
		if now.Sub(connInfo.LastActivity) > wsm.config.IdleTimeout {
			idleConnections = append(idleConnections, connID)
		}
	}
	wsm.mutex.RUnlock()

	for _, connID := range idleConnections {
		wsm.CloseConnection(connID, "idle timeout")
	}

	if len(idleConnections) > 0 {
		logrus.WithField("closed_connections", len(idleConnections)).Info("Cleaned up idle WebSocket connections")
	}
}

// GetStats returns WebSocket security statistics
func (wsm *WebSocketSecurityManager) GetStats() map[string]interface{} {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	// Count connections by type
	authenticatedConnections := 0
	for _, conn := range wsm.connections {
		if conn.UserID > 0 {
			authenticatedConnections++
		}
	}

	wsm.stats.LastUpdate = time.Now()

	return map[string]interface{}{
		"connections": map[string]interface{}{
			"total":         wsm.stats.TotalConnections,
			"active":        wsm.stats.ActiveConnections,
			"authenticated": authenticatedConnections,
			"anonymous":     wsm.stats.ActiveConnections - authenticatedConnections,
		},
		"messages": map[string]interface{}{
			"total":                 wsm.stats.TotalMessages,
			"blocked":               wsm.stats.BlockedMessages,
			"rate_limit_violations": wsm.stats.RateLimitViolations,
		},
		"security": map[string]interface{}{
			"auth_failures":     wsm.stats.AuthFailures,
			"origin_violations": wsm.stats.OriginViolations,
		},
		"limits": map[string]interface{}{
			"max_connections":         wsm.config.MaxConnections,
			"max_connections_per_ip":  wsm.config.MaxConnectionsPerIP,
			"max_connections_per_user": wsm.config.MaxConnectionsPerUser,
			"messages_per_minute":     wsm.config.MessagesPerMinute,
		},
		"config": map[string]interface{}{
			"require_auth":      wsm.config.RequireAuth,
			"origin_validation": wsm.config.OriginValidation,
			"message_validation": wsm.config.MessageValidation,
			"rate_limit_enabled": wsm.config.EnableRateLimit,
		},
	}
}

// GetConnectionInfo returns information about a specific connection
func (wsm *WebSocketSecurityManager) GetConnectionInfo(connID string) (*ConnectionInfo, error) {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	connInfo, exists := wsm.connections[connID]
	if !exists {
		return nil, fmt.Errorf("connection not found")
	}

	// Return a copy to avoid concurrent access issues
	infoCopy := *connInfo
	infoCopy.Connection = nil // Don't expose the connection object
	infoCopy.Claims = nil     // Don't expose sensitive claims

	return &infoCopy, nil
}

// BroadcastMessage sends a message to all connected clients
func (wsm *WebSocketSecurityManager) BroadcastMessage(msg WebSocketMessage) error {
	wsm.mutex.RLock()
	connections := make([]*ConnectionInfo, 0, len(wsm.connections))
	for _, conn := range wsm.connections {
		connections = append(connections, conn)
	}
	wsm.mutex.RUnlock()

	successCount := 0
	for _, connInfo := range connections {
		if connInfo.Connection != nil {
			if err := connInfo.Connection.WriteJSON(msg); err == nil {
				successCount++
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"message_type":  msg.Type,
		"total_connections": len(connections),
		"successful_sends": successCount,
	}).Debug("Broadcast message sent")

	return nil
}

// SendMessageToUser sends a message to all connections of a specific user
func (wsm *WebSocketSecurityManager) SendMessageToUser(userID int64, msg WebSocketMessage) error {
	wsm.mutex.RLock()
	userConnIDs, exists := wsm.userConnections[userID]
	if !exists {
		wsm.mutex.RUnlock()
		return fmt.Errorf("user not connected")
	}

	userConns := make([]*ConnectionInfo, 0, len(userConnIDs))
	for _, connID := range userConnIDs {
		if conn, exists := wsm.connections[connID]; exists {
			userConns = append(userConns, conn)
		}
	}
	wsm.mutex.RUnlock()

	successCount := 0
	for _, connInfo := range userConns {
		if connInfo.Connection != nil {
			if err := connInfo.Connection.WriteJSON(msg); err == nil {
				successCount++
			}
		}
	}

	return nil
}

// Helper functions

// getClientIPFromRequest extracts the client IP from HTTP request
func getClientIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	return r.RemoteAddr
}

// extractTokenFromRequest extracts JWT token from HTTP request
func extractTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Check query parameter as fallback
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	return ""
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	// Use the existing secure random string generation
	id, _ := generateSecureID()
	return "ws_" + id[:16]
}

