package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// DashboardWebSocketManager manages real-time WebSocket connections for dashboard data
type DashboardWebSocketManager struct {
	clients     map[*DashboardClient]bool
	register    chan *DashboardClient
	unregister  chan *DashboardClient
	broadcast   chan *BroadcastMessage
	aggregator  *DashboardAggregator
	logger      *logrus.Logger
	mutex       sync.RWMutex
	subscribers map[string]map[*DashboardClient]bool // topic -> clients
}

// DashboardClient represents a WebSocket client connection
type DashboardClient struct {
	ID           string
	Conn         *websocket.Conn
	Send         chan *ClientMessage
	Subscriptions map[string]bool // subscribed topics
	LastSeen     time.Time
	UserID       string
	RemoteAddr   string
}

// ClientMessage represents a message to be sent to a client
type ClientMessage struct {
	Type      string      `json:"type"`
	Topic     string      `json:"topic,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	MessageID string      `json:"messageId,omitempty"`
}

// BroadcastMessage represents a message to be broadcast to clients
type BroadcastMessage struct {
	Topic     string
	Data      interface{}
	ClientIDs []string // if empty, broadcast to all clients subscribed to topic
}

// SubscriptionMessage represents a subscription request from client
type SubscriptionMessage struct {
	Action string   `json:"action"` // subscribe, unsubscribe
	Topics []string `json:"topics"`
}

// DashboardTopics defines available WebSocket topics
const (
	TopicOverview       = "dashboard:overview"
	TopicContainerStats = "dashboard:containers"
	TopicResourceMetrics = "dashboard:resources"
	TopicSecurityStatus = "dashboard:security"
	TopicUpdateActivity = "dashboard:updates"
	TopicHealthMetrics  = "dashboard:health"
	TopicSystemAlerts   = "dashboard:alerts"
)

// NewDashboardWebSocketManager creates a new dashboard WebSocket manager
func NewDashboardWebSocketManager(aggregator *DashboardAggregator, logger *logrus.Logger) *DashboardWebSocketManager {
	return &DashboardWebSocketManager{
		clients:     make(map[*DashboardClient]bool),
		register:    make(chan *DashboardClient),
		unregister:  make(chan *DashboardClient),
		broadcast:   make(chan *BroadcastMessage, 256),
		aggregator:  aggregator,
		logger:      logger,
		subscribers: make(map[string]map[*DashboardClient]bool),
	}
}

// Start starts the WebSocket manager
func (dwm *DashboardWebSocketManager) Start(ctx context.Context) {
	go dwm.run(ctx)
	go dwm.startDataBroadcast(ctx)
}

// run handles client registration, unregistration, and message broadcasting
func (dwm *DashboardWebSocketManager) run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			dwm.logger.Info("Dashboard WebSocket manager shutting down")
			return

		case client := <-dwm.register:
			dwm.mutex.Lock()
			dwm.clients[client] = true
			dwm.mutex.Unlock()

			dwm.logger.WithFields(logrus.Fields{
				"clientId":   client.ID,
				"userID":     client.UserID,
				"remoteAddr": client.RemoteAddr,
			}).Info("Dashboard client connected")

			// Send welcome message
			welcomeMsg := &ClientMessage{
				Type:      "welcome",
				Data:      map[string]interface{}{"clientId": client.ID, "connectedAt": time.Now()},
				Timestamp: time.Now(),
			}
			select {
			case client.Send <- welcomeMsg:
			default:
				close(client.Send)
				delete(dwm.clients, client)
			}

		case client := <-dwm.unregister:
			dwm.mutex.Lock()
			if _, ok := dwm.clients[client]; ok {
				delete(dwm.clients, client)
				close(client.Send)

				// Remove from all subscriptions
				for topic, subscribers := range dwm.subscribers {
					if subscribers[client] {
						delete(subscribers, client)
						if len(subscribers) == 0 {
							delete(dwm.subscribers, topic)
						}
					}
				}
			}
			dwm.mutex.Unlock()

			dwm.logger.WithFields(logrus.Fields{
				"clientId":   client.ID,
				"userID":     client.UserID,
			}).Info("Dashboard client disconnected")

		case message := <-dwm.broadcast:
			dwm.broadcastToSubscribers(message)

		case <-ticker.C:
			// Clean up inactive clients
			dwm.cleanupInactiveClients()
		}
	}
}

// startDataBroadcast starts periodic broadcasting of dashboard data
func (dwm *DashboardWebSocketManager) startDataBroadcast(ctx context.Context) {
	// Different intervals for different data types
	overviewTicker := time.NewTicker(30 * time.Second)
	resourcesTicker := time.NewTicker(5 * time.Second)
	containersTicker := time.NewTicker(10 * time.Second)
	securityTicker := time.NewTicker(5 * time.Minute)
	updatesTicker := time.NewTicker(1 * time.Minute)
	healthTicker := time.NewTicker(15 * time.Second)

	defer func() {
		overviewTicker.Stop()
		resourcesTicker.Stop()
		containersTicker.Stop()
		securityTicker.Stop()
		updatesTicker.Stop()
		healthTicker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case <-overviewTicker.C:
			dwm.broadcastSystemOverview(ctx)

		case <-resourcesTicker.C:
			dwm.broadcastResourceMetrics(ctx)

		case <-containersTicker.C:
			dwm.broadcastContainerStats(ctx)

		case <-securityTicker.C:
			dwm.broadcastSecurityStatus(ctx)

		case <-updatesTicker.C:
			dwm.broadcastUpdateActivity(ctx)

		case <-healthTicker.C:
			dwm.broadcastHealthMetrics(ctx)
		}
	}
}

// broadcastSystemOverview broadcasts system overview data
func (dwm *DashboardWebSocketManager) broadcastSystemOverview(ctx context.Context) {
	if !dwm.hasSubscribers(TopicOverview) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get system overview for broadcast")
		return
	}

	message := &BroadcastMessage{
		Topic: TopicOverview,
		Data:  overview,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping system overview message")
	}
}

// broadcastResourceMetrics broadcasts resource metrics
func (dwm *DashboardWebSocketManager) broadcastResourceMetrics(ctx context.Context) {
	if !dwm.hasSubscribers(TopicResourceMetrics) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get resource metrics for broadcast")
		return
	}

	resourceData := map[string]interface{}{
		"cpu":         overview.ResourceUsage.CPU,
		"memory":      overview.ResourceUsage.Memory,
		"disk":        overview.ResourceUsage.Disk,
		"network":     overview.ResourceUsage.Network,
		"lastUpdated": time.Now(),
	}

	message := &BroadcastMessage{
		Topic: TopicResourceMetrics,
		Data:  resourceData,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping resource metrics message")
	}
}

// broadcastContainerStats broadcasts container statistics
func (dwm *DashboardWebSocketManager) broadcastContainerStats(ctx context.Context) {
	if !dwm.hasSubscribers(TopicContainerStats) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get container stats for broadcast")
		return
	}

	containerData := map[string]interface{}{
		"stats":       overview.ContainerStats,
		"lastUpdated": time.Now(),
	}

	message := &BroadcastMessage{
		Topic: TopicContainerStats,
		Data:  containerData,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping container stats message")
	}
}

// broadcastSecurityStatus broadcasts security status
func (dwm *DashboardWebSocketManager) broadcastSecurityStatus(ctx context.Context) {
	if !dwm.hasSubscribers(TopicSecurityStatus) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get security status for broadcast")
		return
	}

	securityData := map[string]interface{}{
		"status":      overview.SecurityStatus,
		"lastUpdated": time.Now(),
	}

	message := &BroadcastMessage{
		Topic: TopicSecurityStatus,
		Data:  securityData,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping security status message")
	}
}

// broadcastUpdateActivity broadcasts update activity
func (dwm *DashboardWebSocketManager) broadcastUpdateActivity(ctx context.Context) {
	if !dwm.hasSubscribers(TopicUpdateActivity) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get update activity for broadcast")
		return
	}

	updateData := map[string]interface{}{
		"activity":    overview.UpdateActivity,
		"lastUpdated": time.Now(),
	}

	message := &BroadcastMessage{
		Topic: TopicUpdateActivity,
		Data:  updateData,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping update activity message")
	}
}

// broadcastHealthMetrics broadcasts health metrics
func (dwm *DashboardWebSocketManager) broadcastHealthMetrics(ctx context.Context) {
	if !dwm.hasSubscribers(TopicHealthMetrics) {
		return
	}

	overview, err := dwm.aggregator.GetSystemOverview(ctx)
	if err != nil {
		dwm.logger.WithError(err).Error("Failed to get health metrics for broadcast")
		return
	}

	healthData := map[string]interface{}{
		"health":      overview.SystemHealth,
		"lastUpdated": time.Now(),
	}

	message := &BroadcastMessage{
		Topic: TopicHealthMetrics,
		Data:  healthData,
	}

	select {
	case dwm.broadcast <- message:
	default:
		dwm.logger.Warn("Broadcast channel full, dropping health metrics message")
	}
}

// broadcastToSubscribers broadcasts a message to all subscribers of a topic
func (dwm *DashboardWebSocketManager) broadcastToSubscribers(message *BroadcastMessage) {
	dwm.mutex.RLock()
	subscribers, exists := dwm.subscribers[message.Topic]
	if !exists {
		dwm.mutex.RUnlock()
		return
	}

	// Create a copy of subscribers to avoid holding the lock
	clientList := make([]*DashboardClient, 0, len(subscribers))
	for client := range subscribers {
		clientList = append(clientList, client)
	}
	dwm.mutex.RUnlock()

	clientMsg := &ClientMessage{
		Type:      "data",
		Topic:     message.Topic,
		Data:      message.Data,
		Timestamp: time.Now(),
		MessageID: fmt.Sprintf("%s_%d", message.Topic, time.Now().UnixNano()),
	}

	for _, client := range clientList {
		// Filter by client IDs if specified
		if len(message.ClientIDs) > 0 {
			found := false
			for _, clientID := range message.ClientIDs {
				if client.ID == clientID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		select {
		case client.Send <- clientMsg:
		default:
			// Client's send channel is full, remove client
			dwm.unregisterClient(client)
		}
	}
}

// RegisterClient registers a new WebSocket client
func (dwm *DashboardWebSocketManager) RegisterClient(client *DashboardClient) {
	client.LastSeen = time.Now()
	dwm.register <- client
}

// UnregisterClient unregisters a WebSocket client
func (dwm *DashboardWebSocketManager) UnregisterClient(client *DashboardClient) {
	dwm.unregister <- client
}

// HandleClientMessage handles incoming messages from clients
func (dwm *DashboardWebSocketManager) HandleClientMessage(client *DashboardClient, messageData []byte) error {
	client.LastSeen = time.Now()

	var msg map[string]interface{}
	if err := json.Unmarshal(messageData, &msg); err != nil {
		return fmt.Errorf("invalid message format: %w", err)
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid message type")
	}

	switch msgType {
	case "subscribe":
		return dwm.handleSubscription(client, msg)
	case "unsubscribe":
		return dwm.handleUnsubscription(client, msg)
	case "ping":
		return dwm.handlePing(client, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}

// handleSubscription handles topic subscription requests
func (dwm *DashboardWebSocketManager) handleSubscription(client *DashboardClient, msg map[string]interface{}) error {
	topics, ok := msg["topics"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid topics format")
	}

	dwm.mutex.Lock()
	defer dwm.mutex.Unlock()

	for _, topicInterface := range topics {
		topic, ok := topicInterface.(string)
		if !ok {
			continue
		}

		// Validate topic
		if !dwm.isValidTopic(topic) {
			dwm.logger.WithFields(logrus.Fields{
				"clientId": client.ID,
				"topic":    topic,
			}).Warn("Invalid subscription topic")
			continue
		}

		// Add to client subscriptions
		if client.Subscriptions == nil {
			client.Subscriptions = make(map[string]bool)
		}
		client.Subscriptions[topic] = true

		// Add to topic subscribers
		if dwm.subscribers[topic] == nil {
			dwm.subscribers[topic] = make(map[*DashboardClient]bool)
		}
		dwm.subscribers[topic][client] = true

		dwm.logger.WithFields(logrus.Fields{
			"clientId": client.ID,
			"topic":    topic,
		}).Debug("Client subscribed to topic")
	}

	// Send confirmation
	confirmMsg := &ClientMessage{
		Type:      "subscription_confirmed",
		Data:      map[string]interface{}{"topics": topics},
		Timestamp: time.Now(),
	}

	select {
	case client.Send <- confirmMsg:
	default:
		return fmt.Errorf("failed to send confirmation")
	}

	return nil
}

// handleUnsubscription handles topic unsubscription requests
func (dwm *DashboardWebSocketManager) handleUnsubscription(client *DashboardClient, msg map[string]interface{}) error {
	topics, ok := msg["topics"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid topics format")
	}

	dwm.mutex.Lock()
	defer dwm.mutex.Unlock()

	for _, topicInterface := range topics {
		topic, ok := topicInterface.(string)
		if !ok {
			continue
		}

		// Remove from client subscriptions
		if client.Subscriptions != nil {
			delete(client.Subscriptions, topic)
		}

		// Remove from topic subscribers
		if dwm.subscribers[topic] != nil {
			delete(dwm.subscribers[topic], client)
			if len(dwm.subscribers[topic]) == 0 {
				delete(dwm.subscribers, topic)
			}
		}

		dwm.logger.WithFields(logrus.Fields{
			"clientId": client.ID,
			"topic":    topic,
		}).Debug("Client unsubscribed from topic")
	}

	return nil
}

// handlePing handles ping messages from clients
func (dwm *DashboardWebSocketManager) handlePing(client *DashboardClient, msg map[string]interface{}) error {
	pongMsg := &ClientMessage{
		Type:      "pong",
		Data:      map[string]interface{}{"timestamp": time.Now()},
		Timestamp: time.Now(),
	}

	select {
	case client.Send <- pongMsg:
	default:
		return fmt.Errorf("failed to send pong")
	}

	return nil
}

// BroadcastAlert broadcasts an alert to all connected clients
func (dwm *DashboardWebSocketManager) BroadcastAlert(alertType, message string, severity string) {
	alertData := map[string]interface{}{
		"type":      alertType,
		"message":   message,
		"severity":  severity,
		"timestamp": time.Now(),
	}

	broadcastMsg := &BroadcastMessage{
		Topic: TopicSystemAlerts,
		Data:  alertData,
	}

	select {
	case dwm.broadcast <- broadcastMsg:
	default:
		dwm.logger.Warn("Failed to broadcast alert: channel full")
	}
}

// Helper functions

func (dwm *DashboardWebSocketManager) hasSubscribers(topic string) bool {
	dwm.mutex.RLock()
	defer dwm.mutex.RUnlock()
	subscribers, exists := dwm.subscribers[topic]
	return exists && len(subscribers) > 0
}

func (dwm *DashboardWebSocketManager) isValidTopic(topic string) bool {
	validTopics := []string{
		TopicOverview,
		TopicContainerStats,
		TopicResourceMetrics,
		TopicSecurityStatus,
		TopicUpdateActivity,
		TopicHealthMetrics,
		TopicSystemAlerts,
	}

	for _, validTopic := range validTopics {
		if topic == validTopic {
			return true
		}
	}

	return false
}

func (dwm *DashboardWebSocketManager) unregisterClient(client *DashboardClient) {
	select {
	case dwm.unregister <- client:
	default:
		dwm.logger.Warn("Failed to unregister client: channel full")
	}
}

func (dwm *DashboardWebSocketManager) cleanupInactiveClients() {
	dwm.mutex.RLock()
	inactiveClients := make([]*DashboardClient, 0)
	timeout := 5 * time.Minute

	for client := range dwm.clients {
		if time.Since(client.LastSeen) > timeout {
			inactiveClients = append(inactiveClients, client)
		}
	}
	dwm.mutex.RUnlock()

	for _, client := range inactiveClients {
		dwm.logger.WithField("clientId", client.ID).Info("Removing inactive client")
		dwm.unregisterClient(client)
	}
}

// GetStats returns WebSocket manager statistics
func (dwm *DashboardWebSocketManager) GetStats() map[string]interface{} {
	dwm.mutex.RLock()
	defer dwm.mutex.RUnlock()

	topicStats := make(map[string]int)
	for topic, subscribers := range dwm.subscribers {
		topicStats[topic] = len(subscribers)
	}

	return map[string]interface{}{
		"totalClients":     len(dwm.clients),
		"totalTopics":      len(dwm.subscribers),
		"topicStats":       topicStats,
		"lastUpdated":      time.Now(),
	}
}