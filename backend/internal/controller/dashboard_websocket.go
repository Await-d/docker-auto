package controller

import (
	"fmt"
	"net/http"
	"time"

	"docker-auto/pkg/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// DashboardWebSocketController handles Dashboard WebSocket connections
type DashboardWebSocketController struct {
	wsManager *dashboard.DashboardWebSocketManager
	upgrader  websocket.Upgrader
	logger    *logrus.Logger
}

// NewDashboardWebSocketController creates a new dashboard WebSocket controller
func NewDashboardWebSocketController(
	wsManager *dashboard.DashboardWebSocketManager,
	logger *logrus.Logger,
) *DashboardWebSocketController {
	upgrader := websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
	}

	return &DashboardWebSocketController{
		wsManager: wsManager,
		upgrader:  upgrader,
		logger:    logger,
	}
}

// HandleDashboardWebSocket handles WebSocket connections for dashboard data
func (dwc *DashboardWebSocketController) HandleDashboardWebSocket(c *gin.Context) {
	conn, err := dwc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		dwc.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to upgrade WebSocket connection",
			"details": err.Error(),
		})
		return
	}

	// Extract user information from context (assuming it's set by auth middleware)
	userID := dwc.getUserIDFromContext(c)
	clientID := fmt.Sprintf("dashboard_%s_%d", userID, time.Now().UnixNano())

	// Create client
	client := &dashboard.DashboardClient{
		ID:            clientID,
		Conn:          conn,
		Send:          make(chan *dashboard.ClientMessage, 256),
		Subscriptions: make(map[string]bool),
		LastSeen:      time.Now(),
		UserID:        userID,
		RemoteAddr:    c.Request.RemoteAddr,
	}

	// Register client with WebSocket manager
	dwc.wsManager.RegisterClient(client)

	// Start client goroutines
	go dwc.handleClientWrites(client)
	go dwc.handleClientReads(client)
}

// handleClientReads handles reading messages from WebSocket client
func (dwc *DashboardWebSocketController) handleClientReads(client *dashboard.DashboardClient) {
	defer func() {
		dwc.wsManager.UnregisterClient(client)
		client.Conn.Close()
	}()

	// Set read deadline and pong handler
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, messageData, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				dwc.logger.WithError(err).WithField("clientId", client.ID).Error("WebSocket read error")
			}
			break
		}

		// Handle client message
		if err := dwc.wsManager.HandleClientMessage(client, messageData); err != nil {
			dwc.logger.WithError(err).WithField("clientId", client.ID).Error("Failed to handle client message")

			// Send error message to client
			errorMsg := &dashboard.ClientMessage{
				Type:      "error",
				Data:      map[string]interface{}{"error": err.Error()},
				Timestamp: time.Now(),
			}

			select {
			case client.Send <- errorMsg:
			default:
				// Client send channel is full, close connection
				return
			}
		}
	}
}

// handleClientWrites handles writing messages to WebSocket client
func (dwc *DashboardWebSocketController) handleClientWrites(client *dashboard.DashboardClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				// Channel closed, close WebSocket connection
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send JSON message
			if err := client.Conn.WriteJSON(message); err != nil {
				dwc.logger.WithError(err).WithField("clientId", client.ID).Error("WebSocket write error")
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				dwc.logger.WithError(err).WithField("clientId", client.ID).Error("Failed to send ping")
				return
			}
		}
	}
}

// GetWebSocketStats handles GET /api/dashboard/ws/stats
func (dwc *DashboardWebSocketController) GetWebSocketStats(c *gin.Context) {
	stats := dwc.wsManager.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// BroadcastAlert handles POST /api/dashboard/ws/broadcast-alert
func (dwc *DashboardWebSocketController) BroadcastAlert(c *gin.Context) {
	var request struct {
		Type     string `json:"type" binding:"required"`
		Message  string `json:"message" binding:"required"`
		Severity string `json:"severity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate severity
	validSeverities := map[string]bool{
		"info":     true,
		"warning":  true,
		"error":    true,
		"critical": true,
	}

	if !validSeverities[request.Severity] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid severity. Must be one of: info, warning, error, critical",
		})
		return
	}

	// Broadcast alert
	dwc.wsManager.BroadcastAlert(request.Type, request.Message, request.Severity)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert broadcasted successfully",
		"data": gin.H{
			"type":      request.Type,
			"message":   request.Message,
			"severity":  request.Severity,
			"timestamp": time.Now(),
		},
	})
}

// Helper functions

func (dwc *DashboardWebSocketController) getUserIDFromContext(c *gin.Context) string {
	// Try to get user ID from JWT claims or session
	if userID, exists := c.Get("userID"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}

	// Try to get from query parameter (for testing)
	if userID := c.Query("userId"); userID != "" {
		return userID
	}

	// Default to IP address if no user info available
	return c.ClientIP()
}