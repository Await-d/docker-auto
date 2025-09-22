package docker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// NotificationManager handles real-time notifications for Docker operations
type NotificationManager struct {
	connections     map[string]*ClientConnection
	channels        map[string]map[string]*ClientConnection // channel -> connectionID -> connection
	mutex           sync.RWMutex
	logger          *logrus.Logger
	progressTracker *ProgressTracker
}

// ClientConnection represents a WebSocket client connection
type ClientConnection struct {
	ID          string
	UserID      int64
	Conn        *websocket.Conn
	Channels    []string
	SendChannel chan []byte
	Done        chan bool
	LastPing    time.Time
}

// ProgressTracker tracks progress for various operations
type ProgressTracker struct {
	operations map[string]*OperationProgress
	mutex      sync.RWMutex
}

// OperationProgress represents progress for a single operation
type OperationProgress struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	Progress      float64                `json:"progress"`
	Message       string                 `json:"message"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Error         string                 `json:"error,omitempty"`
	Details       map[string]interface{} `json:"details"`
	Steps         []ProgressStep         `json:"steps"`
	UserID        int64                  `json:"user_id"`
	ContainerID   string                 `json:"container_id,omitempty"`
	ImageName     string                 `json:"image_name,omitempty"`
	LastUpdate    time.Time              `json:"last_update"`
}

// ProgressStep represents a single step in an operation
type ProgressStep struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Progress    float64                `json:"progress"`
	Message     string                 `json:"message"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details"`
}

// NotificationMessage represents a notification message
type NotificationMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	UserID    int64       `json:"user_id,omitempty"`
}

// Operation types
const (
	OperationImagePull      = "image_pull"
	OperationImageRemove    = "image_remove"
	OperationImageScan      = "image_scan"
	OperationContainerUpdate = "container_update"
	OperationContainerCreate = "container_create"
	OperationContainerStart  = "container_start"
	OperationContainerStop   = "container_stop"
	OperationContainerRemove = "container_remove"
)

// Notification types
const (
	NotificationTypeProgress     = "progress"
	NotificationTypeStatus       = "status"
	NotificationTypeError        = "error"
	NotificationTypeSuccess      = "success"
	NotificationTypeAlert        = "alert"
	NotificationTypeHeartbeat    = "heartbeat"
)

// Channels
const (
	ChannelOperations = "operations"
	ChannelAlerts     = "alerts"
	ChannelSystem     = "system"
	ChannelUser       = "user"
)

// NewNotificationManager creates a new notification manager
func NewNotificationManager(logger *logrus.Logger) *NotificationManager {
	if logger == nil {
		logger = logrus.New()
	}

	nm := &NotificationManager{
		connections:     make(map[string]*ClientConnection),
		channels:        make(map[string]map[string]*ClientConnection),
		logger:          logger,
		progressTracker: NewProgressTracker(),
	}

	// Initialize channels
	nm.channels[ChannelOperations] = make(map[string]*ClientConnection)
	nm.channels[ChannelAlerts] = make(map[string]*ClientConnection)
	nm.channels[ChannelSystem] = make(map[string]*ClientConnection)
	nm.channels[ChannelUser] = make(map[string]*ClientConnection)

	// Start cleanup routine
	go nm.cleanupRoutine()

	return nm
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		operations: make(map[string]*OperationProgress),
	}
}

// AddConnection adds a new WebSocket connection
func (nm *NotificationManager) AddConnection(connID string, userID int64, conn *websocket.Conn, channels []string) *ClientConnection {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	client := &ClientConnection{
		ID:          connID,
		UserID:      userID,
		Conn:        conn,
		Channels:    channels,
		SendChannel: make(chan []byte, 256),
		Done:        make(chan bool),
		LastPing:    time.Now(),
	}

	nm.connections[connID] = client

	// Subscribe to channels
	for _, channel := range channels {
		if nm.channels[channel] == nil {
			nm.channels[channel] = make(map[string]*ClientConnection)
		}
		nm.channels[channel][connID] = client
	}

	// Start connection handler
	go nm.handleConnection(client)

	nm.logger.WithFields(logrus.Fields{
		"connection_id": connID,
		"user_id":       userID,
		"channels":      channels,
	}).Info("WebSocket connection added")

	return client
}

// RemoveConnection removes a WebSocket connection
func (nm *NotificationManager) RemoveConnection(connID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	client, exists := nm.connections[connID]
	if !exists {
		return
	}

	// Remove from channels
	for _, channel := range client.Channels {
		if channelMap, exists := nm.channels[channel]; exists {
			delete(channelMap, connID)
		}
	}

	// Close connection
	close(client.Done)
	client.Conn.Close()
	delete(nm.connections, connID)

	nm.logger.WithField("connection_id", connID).Info("WebSocket connection removed")
}

// handleConnection handles a WebSocket connection
func (nm *NotificationManager) handleConnection(client *ClientConnection) {
	defer nm.RemoveConnection(client.ID)

	// Set up ping/pong handlers
	client.Conn.SetPongHandler(func(string) error {
		client.LastPing = time.Now()
		return nil
	})

	// Start ping routine
	go nm.pingRoutine(client)

	// Handle incoming messages
	go nm.readMessages(client)

	// Handle outgoing messages
	for {
		select {
		case message := <-client.SendChannel:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				nm.logger.WithError(err).WithField("connection_id", client.ID).Error("Failed to send message")
				return
			}
		case <-client.Done:
			return
		}
	}
}

// readMessages reads messages from WebSocket connection
func (nm *NotificationManager) readMessages(client *ClientConnection) {
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			nm.logger.WithError(err).WithField("connection_id", client.ID).Debug("Connection read error")
			return
		}

		// Handle incoming message (subscription changes, etc.)
		nm.handleIncomingMessage(client, message)
	}
}

// pingRoutine sends periodic ping messages
func (nm *NotificationManager) pingRoutine(client *ClientConnection) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				nm.logger.WithError(err).WithField("connection_id", client.ID).Debug("Failed to send ping")
				return
			}
		case <-client.Done:
			return
		}
	}
}

// handleIncomingMessage handles incoming messages from clients
func (nm *NotificationManager) handleIncomingMessage(client *ClientConnection, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		nm.logger.WithError(err).Warn("Invalid message format")
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		if channels, ok := msg["channels"].([]interface{}); ok {
			nm.subscribeToChannels(client, channels)
		}
	case "unsubscribe":
		if channels, ok := msg["channels"].([]interface{}); ok {
			nm.unsubscribeFromChannels(client, channels)
		}
	case "heartbeat":
		nm.sendHeartbeat(client)
	}
}

// subscribeToChannels subscribes client to additional channels
func (nm *NotificationManager) subscribeToChannels(client *ClientConnection, channels []interface{}) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	for _, ch := range channels {
		if channel, ok := ch.(string); ok {
			if nm.channels[channel] == nil {
				nm.channels[channel] = make(map[string]*ClientConnection)
			}
			nm.channels[channel][client.ID] = client

			// Add to client's channel list if not already present
			found := false
			for _, existingChannel := range client.Channels {
				if existingChannel == channel {
					found = true
					break
				}
			}
			if !found {
				client.Channels = append(client.Channels, channel)
			}
		}
	}
}

// unsubscribeFromChannels unsubscribes client from channels
func (nm *NotificationManager) unsubscribeFromChannels(client *ClientConnection, channels []interface{}) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	for _, ch := range channels {
		if channel, ok := ch.(string); ok {
			if channelMap, exists := nm.channels[channel]; exists {
				delete(channelMap, client.ID)
			}

			// Remove from client's channel list
			newChannels := make([]string, 0, len(client.Channels))
			for _, existingChannel := range client.Channels {
				if existingChannel != channel {
					newChannels = append(newChannels, existingChannel)
				}
			}
			client.Channels = newChannels
		}
	}
}

// sendHeartbeat sends heartbeat response
func (nm *NotificationManager) sendHeartbeat(client *ClientConnection) {
	message := NotificationMessage{
		Type:      NotificationTypeHeartbeat,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": "alive",
		},
	}

	nm.sendToClient(client, message)
}

// StartOperation starts tracking a new operation
func (nm *NotificationManager) StartOperation(operationID, operationType string, userID int64, details map[string]interface{}) *OperationProgress {
	nm.progressTracker.mutex.Lock()
	defer nm.progressTracker.mutex.Unlock()

	operation := &OperationProgress{
		ID:         operationID,
		Type:       operationType,
		Status:     "started",
		Progress:   0.0,
		Message:    fmt.Sprintf("%s operation started", operationType),
		StartTime:  time.Now(),
		Details:    details,
		Steps:      []ProgressStep{},
		UserID:     userID,
		LastUpdate: time.Now(),
	}

	if details != nil {
		if containerID, ok := details["container_id"].(string); ok {
			operation.ContainerID = containerID
		}
		if imageName, ok := details["image_name"].(string); ok {
			operation.ImageName = imageName
		}
	}

	nm.progressTracker.operations[operationID] = operation

	// Notify operation start
	nm.NotifyOperationProgress(operation)

	nm.logger.WithFields(logrus.Fields{
		"operation_id":   operationID,
		"operation_type": operationType,
		"user_id":        userID,
	}).Info("Operation started")

	return operation
}

// UpdateOperation updates an operation's progress
func (nm *NotificationManager) UpdateOperation(operationID string, progress float64, message string, details map[string]interface{}) {
	nm.progressTracker.mutex.Lock()
	operation, exists := nm.progressTracker.operations[operationID]
	if !exists {
		nm.progressTracker.mutex.Unlock()
		return
	}

	operation.Progress = progress
	operation.Message = message
	operation.LastUpdate = time.Now()

	if details != nil {
		for key, value := range details {
			operation.Details[key] = value
		}
	}
	nm.progressTracker.mutex.Unlock()

	// Notify progress update
	nm.NotifyOperationProgress(operation)
}

// AddOperationStep adds a step to an operation
func (nm *NotificationManager) AddOperationStep(operationID, stepName, message string) {
	nm.progressTracker.mutex.Lock()
	operation, exists := nm.progressTracker.operations[operationID]
	if !exists {
		nm.progressTracker.mutex.Unlock()
		return
	}

	step := ProgressStep{
		Name:      stepName,
		Status:    "running",
		Progress:  0.0,
		Message:   message,
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	operation.Steps = append(operation.Steps, step)
	operation.LastUpdate = time.Now()
	nm.progressTracker.mutex.Unlock()

	// Notify step added
	nm.NotifyOperationProgress(operation)
}

// CompleteOperationStep completes a step in an operation
func (nm *NotificationManager) CompleteOperationStep(operationID, stepName string, success bool, errorMsg string) {
	nm.progressTracker.mutex.Lock()
	operation, exists := nm.progressTracker.operations[operationID]
	if !exists {
		nm.progressTracker.mutex.Unlock()
		return
	}

	// Find and update the step
	for i := range operation.Steps {
		if operation.Steps[i].Name == stepName {
			step := &operation.Steps[i]
			endTime := time.Now()
			step.EndTime = &endTime
			step.Duration = endTime.Sub(step.StartTime)
			step.Progress = 100.0

			if success {
				step.Status = "completed"
			} else {
				step.Status = "failed"
				step.Error = errorMsg
			}
			break
		}
	}

	operation.LastUpdate = time.Now()
	nm.progressTracker.mutex.Unlock()

	// Notify step completion
	nm.NotifyOperationProgress(operation)
}

// CompleteOperation completes an operation
func (nm *NotificationManager) CompleteOperation(operationID string, success bool, errorMsg string) {
	nm.progressTracker.mutex.Lock()
	operation, exists := nm.progressTracker.operations[operationID]
	if !exists {
		nm.progressTracker.mutex.Unlock()
		return
	}

	endTime := time.Now()
	operation.EndTime = &endTime
	operation.Duration = endTime.Sub(operation.StartTime)
	operation.LastUpdate = time.Now()

	if success {
		operation.Status = "completed"
		operation.Progress = 100.0
		operation.Message = fmt.Sprintf("%s operation completed successfully", operation.Type)
	} else {
		operation.Status = "failed"
		operation.Error = errorMsg
		operation.Message = fmt.Sprintf("%s operation failed: %s", operation.Type, errorMsg)
	}
	nm.progressTracker.mutex.Unlock()

	// Notify operation completion
	nm.NotifyOperationProgress(operation)

	// Send success/error notification
	if success {
		nm.NotifySuccess(operation.UserID, ChannelOperations, fmt.Sprintf("%s completed successfully", operation.Type), map[string]interface{}{
			"operation_id": operationID,
			"type":         operation.Type,
			"duration":     operation.Duration,
		})
	} else {
		nm.NotifyError(operation.UserID, ChannelOperations, fmt.Sprintf("%s failed", operation.Type), errorMsg, map[string]interface{}{
			"operation_id": operationID,
			"type":         operation.Type,
		})
	}

	nm.logger.WithFields(logrus.Fields{
		"operation_id":   operationID,
		"operation_type": operation.Type,
		"success":        success,
		"duration":       operation.Duration,
	}).Info("Operation completed")
}

// NotifyOperationProgress sends operation progress notification
func (nm *NotificationManager) NotifyOperationProgress(operation *OperationProgress) {
	message := NotificationMessage{
		Type:      NotificationTypeProgress,
		Channel:   ChannelOperations,
		Timestamp: time.Now(),
		Data:      operation,
		UserID:    operation.UserID,
	}

	nm.BroadcastToChannel(ChannelOperations, message, operation.UserID)
}

// NotifySuccess sends success notification
func (nm *NotificationManager) NotifySuccess(userID int64, channel, title string, data map[string]interface{}) {
	message := NotificationMessage{
		Type:      NotificationTypeSuccess,
		Channel:   channel,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"title": title,
			"data":  data,
		},
		UserID: userID,
	}

	nm.BroadcastToChannel(channel, message, userID)
}

// NotifyError sends error notification
func (nm *NotificationManager) NotifyError(userID int64, channel, title, errorMsg string, data map[string]interface{}) {
	message := NotificationMessage{
		Type:      NotificationTypeError,
		Channel:   channel,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"title":   title,
			"error":   errorMsg,
			"data":    data,
		},
		UserID: userID,
	}

	nm.BroadcastToChannel(channel, message, userID)
}

// NotifyAlert sends alert notification
func (nm *NotificationManager) NotifyAlert(userID int64, level, title, message string, data map[string]interface{}) {
	notificationMessage := NotificationMessage{
		Type:      NotificationTypeAlert,
		Channel:   ChannelAlerts,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"level":   level,
			"title":   title,
			"message": message,
			"data":    data,
		},
		UserID: userID,
	}

	nm.BroadcastToChannel(ChannelAlerts, notificationMessage, userID)
}

// BroadcastToChannel broadcasts message to all clients in a channel
func (nm *NotificationManager) BroadcastToChannel(channel string, message NotificationMessage, userID int64) {
	nm.mutex.RLock()
	channelMap, exists := nm.channels[channel]
	if !exists {
		nm.mutex.RUnlock()
		return
	}

	// Send to specific user or all users in channel
	for _, client := range channelMap {
		if userID == 0 || client.UserID == userID {
			nm.sendToClient(client, message)
		}
	}
	nm.mutex.RUnlock()
}

// BroadcastToAll broadcasts message to all connected clients
func (nm *NotificationManager) BroadcastToAll(message NotificationMessage) {
	nm.mutex.RLock()
	defer nm.mutex.RUnlock()

	for _, client := range nm.connections {
		nm.sendToClient(client, message)
	}
}

// sendToClient sends message to a specific client
func (nm *NotificationManager) sendToClient(client *ClientConnection, message NotificationMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		nm.logger.WithError(err).Error("Failed to marshal notification message")
		return
	}

	select {
	case client.SendChannel <- messageBytes:
		// Message sent successfully
	default:
		// Channel full, client may be slow or disconnected
		nm.logger.WithField("connection_id", client.ID).Warn("Client send channel full")
	}
}

// GetOperation returns operation progress
func (nm *NotificationManager) GetOperation(operationID string) (*OperationProgress, bool) {
	nm.progressTracker.mutex.RLock()
	defer nm.progressTracker.mutex.RUnlock()

	operation, exists := nm.progressTracker.operations[operationID]
	return operation, exists
}

// GetUserOperations returns all operations for a user
func (nm *NotificationManager) GetUserOperations(userID int64) []*OperationProgress {
	nm.progressTracker.mutex.RLock()
	defer nm.progressTracker.mutex.RUnlock()

	var operations []*OperationProgress
	for _, operation := range nm.progressTracker.operations {
		if operation.UserID == userID {
			operations = append(operations, operation)
		}
	}

	return operations
}

// GetActiveOperations returns all active operations
func (nm *NotificationManager) GetActiveOperations() []*OperationProgress {
	nm.progressTracker.mutex.RLock()
	defer nm.progressTracker.mutex.RUnlock()

	var operations []*OperationProgress
	for _, operation := range nm.progressTracker.operations {
		if operation.Status == "started" || operation.Status == "running" {
			operations = append(operations, operation)
		}
	}

	return operations
}

// cleanupRoutine performs periodic cleanup
func (nm *NotificationManager) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		nm.cleanupOperations()
		nm.cleanupConnections()
	}
}

// cleanupOperations removes old completed operations
func (nm *NotificationManager) cleanupOperations() {
	nm.progressTracker.mutex.Lock()
	defer nm.progressTracker.mutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, operation := range nm.progressTracker.operations {
		if operation.EndTime != nil && operation.EndTime.Before(cutoff) {
			delete(nm.progressTracker.operations, id)
		}
	}
}

// cleanupConnections removes stale connections
func (nm *NotificationManager) cleanupConnections() {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	staleThreshold := time.Now().Add(-2 * time.Minute)
	for id, client := range nm.connections {
		if client.LastPing.Before(staleThreshold) {
			nm.logger.WithField("connection_id", id).Info("Removing stale connection")
			go nm.RemoveConnection(id)
		}
	}
}

// GetStats returns notification manager statistics
func (nm *NotificationManager) GetStats() map[string]interface{} {
	nm.mutex.RLock()
	nm.progressTracker.mutex.RLock()

	stats := map[string]interface{}{
		"total_connections":     len(nm.connections),
		"total_operations":      len(nm.progressTracker.operations),
		"active_operations":     len(nm.GetActiveOperations()),
		"channels":              make(map[string]int),
	}

	for channel, connections := range nm.channels {
		stats["channels"].(map[string]int)[channel] = len(connections)
	}

	nm.progressTracker.mutex.RUnlock()
	nm.mutex.RUnlock()

	return stats
}