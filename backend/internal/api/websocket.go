package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/pkg/events"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID           string
	UserID       *string
	Conn         *websocket.Conn
	Send         chan []byte
	Publisher    events.Publisher
	Subscription *events.Subscription
	LastPing     time.Time
	mu           sync.RWMutex
	closed       bool
	rateLimiter  *RateLimiter
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
	publisher   events.Publisher
	logger      *logrus.Logger
	config      *config.Config
	jwtManager  *utils.JWTManager
}

// ClientMessage represents a message from client to server
type ClientMessage struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	MessageID string      `json:"message_id,omitempty"`
}

// ServerMessage represents a message from server to client
type ServerMessage struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	MessageID string      `json:"message_id,omitempty"`
}

// RateLimiter implements simple token bucket rate limiting
type RateLimiter struct {
	tokens    int
	maxTokens int
	refill    time.Duration
	lastFill  time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillDuration time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		refill:    refillDuration,
		lastFill:  time.Now(),
	}
}

// Allow checks if an action is allowed under rate limiting
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastFill)

	// Refill tokens based on elapsed time
	if elapsed >= rl.refill {
		tokensToAdd := int(elapsed / rl.refill)
		rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
		rl.lastFill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(publisher events.Publisher, logger *logrus.Logger, cfg *config.Config) *WebSocketManager {
	if logger == nil {
		logger = logrus.New()
	}

	return &WebSocketManager{
		connections: make(map[string]*WebSocketConnection),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking based on configuration
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		publisher:  publisher,
		logger:     logger,
		config:     cfg,
		jwtManager: utils.NewJWTManager(cfg),
	}
}

// HandleWebSocket handles WebSocket connection requests
func (wm *WebSocketManager) HandleWebSocket(c *gin.Context) {
	// Extract JWT token from query parameter or header
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if token != "" && len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
		return
	}

	// Validate JWT token
	claims, err := wm.jwtManager.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := wm.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		wm.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Create WebSocket connection
	userIDStr := fmt.Sprintf("%d", claims.UserID)
	wsConn := &WebSocketConnection{
		ID:          uuid.New().String(),
		UserID:      &userIDStr,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Publisher:   wm.publisher,
		LastPing:    time.Now(),
		rateLimiter: NewRateLimiter(100, time.Minute), // 100 messages per minute
	}

	// Register connection
	wm.mu.Lock()
	wm.connections[wsConn.ID] = wsConn
	wm.mu.Unlock()

	wm.logger.WithFields(logrus.Fields{
		"connection_id": wsConn.ID,
		"user_id":       claims.UserID,
		"remote_addr":   c.Request.RemoteAddr,
	}).Info("WebSocket connection established")

	// Start goroutines for handling the connection
	go wsConn.writePump(wm)
	go wsConn.readPump(wm)
}

// readPump handles reading messages from the WebSocket connection
func (wsc *WebSocketConnection) readPump(wm *WebSocketManager) {
	defer func() {
		wsc.close(wm)
	}()

	// Set read deadline and pong handler
	wsc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsc.Conn.SetPongHandler(func(string) error {
		wsc.mu.Lock()
		wsc.LastPing = time.Now()
		wsc.mu.Unlock()
		wsc.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := wsc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wm.logger.WithError(err).Error("WebSocket connection error")
			}
			break
		}

		// Rate limiting check
		if !wsc.rateLimiter.Allow() {
			wsc.sendError("Rate limit exceeded")
			continue
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			wsc.sendError("Invalid message format")
			continue
		}

		// Handle message based on type
		wsc.handleClientMessage(wm, &clientMsg)
	}
}

// writePump handles writing messages to the WebSocket connection
func (wsc *WebSocketConnection) writePump(wm *WebSocketManager) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		wsc.close(wm)
	}()

	for {
		select {
		case message, ok := <-wsc.Send:
			wsc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				wsc.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := wsc.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				wm.logger.WithError(err).Error("Failed to write WebSocket message")
				return
			}

		case <-ticker.C:
			wsc.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsc.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage processes messages from the client
func (wsc *WebSocketConnection) handleClientMessage(wm *WebSocketManager, msg *ClientMessage) {
	switch msg.Type {
	case "subscribe":
		wsc.handleSubscribe(wm, msg)
	case "unsubscribe":
		wsc.handleUnsubscribe(wm, msg)
	case "ping":
		wsc.handlePing(msg)
	case "ack":
		wsc.handleAck(msg)
	default:
		wsc.sendError("Unknown message type: " + msg.Type)
	}
}

// handleSubscribe processes subscription requests
func (wsc *WebSocketConnection) handleSubscribe(wm *WebSocketManager, msg *ClientMessage) {
	if msg.Topic == "" {
		wsc.sendError("Topic is required for subscription")
		return
	}

	// Create event filter based on topic
	filter := wsc.createFilterFromTopic(msg.Topic)

	// Unsubscribe from existing subscription if any
	if wsc.Subscription != nil {
		wm.publisher.Unsubscribe(wsc.Subscription.ID)
	}

	// Create new subscription
	subscription := wm.publisher.SubscribeWithUser(filter, *wsc.UserID)
	wsc.Subscription = subscription

	// Start listening for events
	go wsc.eventListener(wm)

	// Send confirmation
	wsc.sendMessage("subscription_confirmed", msg.Topic, map[string]interface{}{
		"subscription_id": subscription.ID,
		"topic":          msg.Topic,
	}, msg.MessageID)

	wm.logger.WithFields(logrus.Fields{
		"connection_id":   wsc.ID,
		"subscription_id": subscription.ID,
		"topic":           msg.Topic,
	}).Debug("Client subscribed to topic")
}

// handleUnsubscribe processes unsubscription requests
func (wsc *WebSocketConnection) handleUnsubscribe(wm *WebSocketManager, msg *ClientMessage) {
	if wsc.Subscription != nil {
		wm.publisher.Unsubscribe(wsc.Subscription.ID)
		wsc.Subscription = nil

		wsc.sendMessage("unsubscription_confirmed", msg.Topic, nil, msg.MessageID)

		wm.logger.WithFields(logrus.Fields{
			"connection_id": wsc.ID,
			"topic":         msg.Topic,
		}).Debug("Client unsubscribed from topic")
	}
}

// handlePing responds to ping messages
func (wsc *WebSocketConnection) handlePing(msg *ClientMessage) {
	wsc.sendMessage("pong", "", nil, msg.MessageID)
}

// handleAck processes acknowledgment messages
func (wsc *WebSocketConnection) handleAck(msg *ClientMessage) {
	// Log message acknowledgment
	wsc.mu.Lock()
	wsc.LastPing = time.Now()
	wsc.mu.Unlock()
}

// eventListener listens for events and forwards them to the client
func (wsc *WebSocketConnection) eventListener(wm *WebSocketManager) {
	if wsc.Subscription == nil {
		return
	}

	for event := range wsc.Subscription.Channel {
		if wsc.isClosed() {
			break
		}

		eventData := map[string]interface{}{
			"id":            event.ID,
			"type":          event.Type,
			"severity":      event.Severity,
			"source":        event.Source,
			"title":         event.Title,
			"message":       event.Message,
			"data":          event.Data,
			"timestamp":     event.Timestamp.Unix(),
			"tags":          event.Tags,
			"resource_id":   event.ResourceID,
			"resource_type": event.ResourceType,
		}

		wsc.sendMessage("event", string(event.Type), eventData, "")
	}
}

// createFilterFromTopic creates an event filter based on topic
func (wsc *WebSocketConnection) createFilterFromTopic(topic string) events.EventFilter {
	filter := events.EventFilter{}

	switch topic {
	case "container.status":
		filter.Types = []events.EventType{
			events.EventContainerStarted,
			events.EventContainerStopped,
			events.EventContainerUpdated,
			events.EventContainerError,
			events.EventContainerCreated,
			events.EventContainerDeleted,
			events.EventContainerRestarted,
		}
	case "container.logs":
		// This would be handled separately with log streaming
		filter.Types = []events.EventType{events.EventContainerError}
	case "image.update":
		filter.Types = []events.EventType{
			events.EventImageUpdateAvailable,
			events.EventImageUpdateStarted,
			events.EventImageUpdateCompleted,
			events.EventImageUpdateFailed,
		}
	case "system.health":
		filter.Types = []events.EventType{
			events.EventSystemHealthChanged,
			events.EventSystemResourceAlert,
		}
	case "task.progress":
		filter.Types = []events.EventType{
			events.EventTaskStarted,
			events.EventTaskCompleted,
			events.EventTaskFailed,
		}
	case "user.notification":
		filter.UserID = wsc.UserID
		filter.Types = []events.EventType{
			events.EventNotificationCreated,
		}
	case "all":
		// No filter, receive all events
	default:
		// Try to parse as event type
		filter.Types = []events.EventType{events.EventType(topic)}
	}

	return filter
}

// sendMessage sends a message to the client
func (wsc *WebSocketConnection) sendMessage(msgType, topic string, data interface{}, messageID string) {
	if wsc.isClosed() {
		return
	}

	msg := ServerMessage{
		Type:      msgType,
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case wsc.Send <- msgBytes:
	default:
		// Channel is full, drop the message
	}
}

// sendError sends an error message to the client
func (wsc *WebSocketConnection) sendError(errorMsg string) {
	wsc.sendMessage("error", "", map[string]string{"error": errorMsg}, "")
}

// close closes the WebSocket connection
func (wsc *WebSocketConnection) close(wm *WebSocketManager) {
	wsc.mu.Lock()
	if wsc.closed {
		wsc.mu.Unlock()
		return
	}
	wsc.closed = true
	wsc.mu.Unlock()

	// Unsubscribe from events
	if wsc.Subscription != nil {
		wm.publisher.Unsubscribe(wsc.Subscription.ID)
	}

	// Close send channel
	close(wsc.Send)

	// Close WebSocket connection
	wsc.Conn.Close()

	// Remove from manager
	wm.mu.Lock()
	delete(wm.connections, wsc.ID)
	wm.mu.Unlock()

	wm.logger.WithFields(logrus.Fields{
		"connection_id": wsc.ID,
		"user_id":       wsc.UserID,
	}).Info("WebSocket connection closed")
}

// isClosed checks if the connection is closed
func (wsc *WebSocketConnection) isClosed() bool {
	wsc.mu.RLock()
	defer wsc.mu.RUnlock()
	return wsc.closed
}

// IsClosed checks if the connection is closed (public method)
func (wsc *WebSocketConnection) IsClosed() bool {
	return wsc.isClosed()
}

// GetConnections returns all active connections
func (wm *WebSocketManager) GetConnections() []*WebSocketConnection {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	connections := make([]*WebSocketConnection, 0, len(wm.connections))
	for _, conn := range wm.connections {
		if !conn.isClosed() {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetConnectionsByUser returns connections for a specific user
func (wm *WebSocketManager) GetConnectionsByUser(userID string) []*WebSocketConnection {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var connections []*WebSocketConnection
	for _, conn := range wm.connections {
		if !conn.isClosed() && conn.UserID != nil && *conn.UserID == userID {
			connections = append(connections, conn)
		}
	}

	return connections
}

// BroadcastToUser sends a message to all connections of a specific user
func (wm *WebSocketManager) BroadcastToUser(userID string, msgType, topic string, data interface{}) {
	connections := wm.GetConnectionsByUser(userID)
	for _, conn := range connections {
		conn.sendMessage(msgType, topic, data, "")
	}
}

// BroadcastToAll sends a message to all active connections
func (wm *WebSocketManager) BroadcastToAll(msgType, topic string, data interface{}) {
	connections := wm.GetConnections()
	for _, conn := range connections {
		conn.sendMessage(msgType, topic, data, "")
	}
}

// GetStats returns WebSocket manager statistics
func (wm *WebSocketManager) GetStats() map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	activeConnections := 0
	userConnections := make(map[string]int)

	for _, conn := range wm.connections {
		if !conn.isClosed() {
			activeConnections++
			if conn.UserID != nil {
				userConnections[*conn.UserID]++
			}
		}
	}

	return map[string]interface{}{
		"total_connections":  len(wm.connections),
		"active_connections": activeConnections,
		"user_connections":   userConnections,
	}
}

// CleanupInactiveConnections removes inactive connections
func (wm *WebSocketManager) CleanupInactiveConnections() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	var toDelete []string
	for id, conn := range wm.connections {
		if conn.isClosed() {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(wm.connections, id)
	}

	if len(toDelete) > 0 {
		wm.logger.WithField("count", len(toDelete)).Debug("Cleaned up inactive WebSocket connections")
	}
}

// helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}