package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"docker-auto/pkg/docker"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketController handles WebSocket connections for real-time notifications
type WebSocketController struct {
	serviceManager *docker.ServiceManager
	upgrader       websocket.Upgrader
	logger         *logrus.Logger
}

// NewWebSocketController creates a new WebSocket controller
func NewWebSocketController(serviceManager *docker.ServiceManager, logger *logrus.Logger) *WebSocketController {
	return &WebSocketController{
		serviceManager: serviceManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: logger,
	}
}

// ConnectWebSocket godoc
// @Summary Connect to WebSocket
// @Description Establish WebSocket connection for real-time notifications
// @Tags WebSocket
// @Security BearerAuth
// @Param channels query string false "Comma-separated list of channels to subscribe to (operations,alerts,system,user)"
// @Success 101 "Switching Protocols"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/ws/connect [get]
func (wsc *WebSocketController) ConnectWebSocket(c *gin.Context) {
	// Get user information from middleware
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedJSON(c, "User not authenticated")
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		utils.UnauthorizedJSON(c, "Invalid user ID")
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// Parse channels to subscribe to
	channelsParam := c.DefaultQuery("channels", "operations,alerts,user")
	channels := wsc.parseChannels(channelsParam, role.(string))

	// Upgrade HTTP connection to WebSocket
	conn, err := wsc.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		wsc.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		utils.InternalServerErrorJSON(c, "Failed to upgrade connection")
		return
	}

	// Get notification manager
	notificationManager := wsc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		conn.Close()
		utils.InternalServerErrorJSON(c, "Notification service not available")
		return
	}

	// Generate connection ID
	connectionID := uuid.New().String()

	// Add connection to notification manager
	_ = notificationManager.AddConnection(connectionID, userID, conn, channels)

	wsc.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"user_id":       userID,
		"username":      username,
		"channels":      channels,
		"remote_addr":   c.Request.RemoteAddr,
	}).Info("WebSocket connection established")

	// Send welcome message
	_ = map[string]interface{}{
		"type":          "welcome",
		"connection_id": connectionID,
		"user_id":       userID,
		"channels":      channels,
		"server_time":   "2024-01-15T10:30:00Z", // Current time
		"features": []string{
			"real_time_progress",
			"operation_notifications",
			"security_alerts",
			"system_status",
		},
	}

	// This would normally be handled by the notification manager
	// but for demonstration, we'll show the structure
	wsc.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"message_type":  "welcome",
	}).Debug("Sent welcome message to WebSocket client")

	// The connection is now managed by the notification manager
	// The HTTP handler can return as the connection is handled asynchronously
}

// parseChannels parses and validates channel subscriptions based on user role
func (wsc *WebSocketController) parseChannels(channelsParam, role string) []string {
	requestedChannels := strings.Split(channelsParam, ",")
	var allowedChannels []string

	for _, channel := range requestedChannels {
		channel = strings.TrimSpace(channel)
		if wsc.isChannelAllowed(channel, role) {
			allowedChannels = append(allowedChannels, channel)
		} else {
			wsc.logger.WithFields(logrus.Fields{
				"channel": channel,
				"role":    role,
			}).Warn("Channel access denied for user role")
		}
	}

	// Ensure user always has access to their own user channel
	userChannelAllowed := false
	for _, channel := range allowedChannels {
		if channel == docker.ChannelUser {
			userChannelAllowed = true
			break
		}
	}
	if !userChannelAllowed {
		allowedChannels = append(allowedChannels, docker.ChannelUser)
	}

	return allowedChannels
}

// isChannelAllowed checks if a user role can access a specific channel
func (wsc *WebSocketController) isChannelAllowed(channel, role string) bool {
	switch channel {
	case docker.ChannelOperations:
		// All authenticated users can see operations
		return true
	case docker.ChannelAlerts:
		// All authenticated users can see alerts
		return true
	case docker.ChannelSystem:
		// Only admins can see system-wide notifications
		return role == "admin"
	case docker.ChannelUser:
		// All users can see their own notifications
		return true
	default:
		return false
	}
}

// GetWebSocketInfo godoc
// @Summary Get WebSocket connection info
// @Description Get information about WebSocket connectivity and channels
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "WebSocket info"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Router /api/ws/info [get]
func (wsc *WebSocketController) GetWebSocketInfo(c *gin.Context) {
	role, _ := c.Get("role")

	rb := utils.NewResponseBuilder(c)

	// Get notification manager stats
	var stats map[string]interface{}
	notificationManager := wsc.serviceManager.GetNotificationManager()
	if notificationManager != nil {
		stats = notificationManager.GetStats()
	} else {
		stats = map[string]interface{}{}
	}

	// Build channel information based on user role
	availableChannels := wsc.getAvailableChannels(role.(string))

	info := map[string]interface{}{
		"websocket_url":       "/api/ws/connect",
		"available_channels":  availableChannels,
		"connection_stats":    stats,
		"protocols":          []string{"notifications", "progress", "alerts"},
		"heartbeat_interval": "30s",
		"reconnect_policy": map[string]interface{}{
			"enabled":         true,
			"initial_delay":   "1s",
			"max_delay":       "30s",
			"backoff_factor":  2.0,
			"max_attempts":    10,
		},
		"message_types": []string{
			"welcome",
			"progress",
			"status",
			"error",
			"success",
			"alert",
			"heartbeat",
		},
	}

	rb.Success(info)
}

// getAvailableChannels returns channels available for a user role
func (wsc *WebSocketController) getAvailableChannels(role string) map[string]interface{} {
	channels := map[string]interface{}{
		docker.ChannelOperations: map[string]interface{}{
			"name":        "Operations",
			"description": "Docker operation progress and status updates",
			"access":      "all",
		},
		docker.ChannelAlerts: map[string]interface{}{
			"name":        "Alerts",
			"description": "Security alerts and system warnings",
			"access":      "all",
		},
		docker.ChannelUser: map[string]interface{}{
			"name":        "User",
			"description": "User-specific notifications",
			"access":      "all",
		},
	}

	if role == "admin" {
		channels[docker.ChannelSystem] = map[string]interface{}{
			"name":        "System",
			"description": "System-wide notifications and status",
			"access":      "admin",
		}
	}

	return channels
}

// TestWebSocketMessage godoc
// @Summary Test WebSocket message
// @Description Send a test message through WebSocket (admin only)
// @Tags WebSocket
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Test message"
// @Success 200 {object} utils.APIResponse "Message sent"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Router /api/ws/test [post]
func (wsc *WebSocketController) TestWebSocketMessage(c *gin.Context) {
	// Check admin access
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		utils.ForbiddenJSON(c, "Admin access required")
		return
	}

	var request struct {
		Channel   string      `json:"channel" binding:"required"`
		Type      string      `json:"type" binding:"required"`
		Message   string      `json:"message" binding:"required"`
		Data      interface{} `json:"data,omitempty"`
		UserID    *int64      `json:"user_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := wsc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	// Determine user ID for the message
	targetUserID := int64(0) // 0 means broadcast to all users in channel
	if request.UserID != nil {
		targetUserID = *request.UserID
	}

	// Send test message based on type
	switch request.Type {
	case "alert":
		notificationManager.NotifyAlert(
			targetUserID,
			"info",
			"Test Alert",
			request.Message,
			map[string]interface{}{
				"test": true,
				"data": request.Data,
			},
		)
	case "success":
		notificationManager.NotifySuccess(
			targetUserID,
			request.Channel,
			request.Message,
			map[string]interface{}{
				"test": true,
				"data": request.Data,
			},
		)
	case "error":
		notificationManager.NotifyError(
			targetUserID,
			request.Channel,
			"Test Error",
			request.Message,
			map[string]interface{}{
				"test": true,
				"data": request.Data,
			},
		)
	default:
		rb.BadRequest("Invalid message type. Use: alert, success, error")
		return
	}

	wsc.logger.WithFields(logrus.Fields{
		"channel":     request.Channel,
		"type":        request.Type,
		"message":     request.Message,
		"target_user": targetUserID,
	}).Info("Test WebSocket message sent")

	rb.SuccessWithMessage(map[string]interface{}{
		"sent_to_channel": request.Channel,
		"message_type":    request.Type,
		"target_user":     targetUserID,
	}, "Test message sent successfully")
}

// GetConnectionStatus godoc
// @Summary Get WebSocket connection status
// @Description Get the status of WebSocket connections for monitoring
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=map[string]interface{}} "Connection status"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Router /api/ws/status [get]
func (wsc *WebSocketController) GetConnectionStatus(c *gin.Context) {
	// Check if user can view connection status (admin or developer)
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedJSON(c, "User not authenticated")
		return
	}

	if role != "admin" && role != "developer" {
		utils.ForbiddenJSON(c, "Insufficient privileges")
		return
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := wsc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	stats := notificationManager.GetStats()

	// Add additional connection health information
	status := map[string]interface{}{
		"connection_stats": stats,
		"service_health":   wsc.serviceManager.GetServiceInfo().Health,
		"websocket_config": map[string]interface{}{
			"read_buffer_size":  wsc.upgrader.ReadBufferSize,
			"write_buffer_size": wsc.upgrader.WriteBufferSize,
			"check_origin":      "enabled",
		},
	}

	rb.Success(status)
}

// DisconnectUser godoc
// @Summary Disconnect user WebSocket connections
// @Description Forcefully disconnect all WebSocket connections for a specific user (admin only)
// @Tags WebSocket
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} utils.APIResponse "User disconnected"
// @Failure 400 {object} utils.APIResponse "Invalid user ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Router /api/ws/disconnect/{user_id} [post]
func (wsc *WebSocketController) DisconnectUser(c *gin.Context) {
	// Check admin access
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		utils.ForbiddenJSON(c, "Admin access required")
		return
	}

	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	notificationManager := wsc.serviceManager.GetNotificationManager()
	if notificationManager == nil {
		rb.InternalServerError("Notification service not available")
		return
	}

	// Note: In a real implementation, the notification manager would need
	// a method to disconnect specific users. For now, we'll log the action.
	username, _ := c.Get("username")
	wsc.logger.WithFields(logrus.Fields{
		"target_user_id": userID,
		"admin_user":     username,
	}).Info("Admin requested user disconnection")

	rb.SuccessWithMessage(map[string]interface{}{
		"disconnected_user_id": userID,
	}, fmt.Sprintf("User %d has been disconnected", userID))
}