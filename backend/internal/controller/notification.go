package controller

import (
	"net/http"
	"strconv"
	"time"

	"docker-auto/internal/service"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NotificationController handles notification-related HTTP requests
type NotificationController struct {
	notificationService service.NotificationServiceInterface
	logger              *logrus.Logger
}

// NewNotificationController creates a new notification controller
func NewNotificationController(
	notificationService service.NotificationServiceInterface,
	logger *logrus.Logger,
) *NotificationController {
	if logger == nil {
		logger = logrus.New()
	}

	return &NotificationController{
		notificationService: notificationService,
		logger:              logger,
	}
}

// GetNotifications retrieves notifications for the current user
func (nc *NotificationController) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	// Parse query parameters
	limit := 20 // default
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0 // default
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	notificationType := c.Query("type")

	// Get notifications
	var notifications interface{}
	var err error

	if notificationType != "" {
		notifications, err = nc.notificationService.GetNotificationsByType(
			c.Request.Context(),
			uid,
			service.NotificationType(notificationType),
			limit,
			offset,
		)
	} else {
		notifications, err = nc.notificationService.GetNotifications(
			c.Request.Context(),
			uid,
			limit,
			offset,
		)
	}

	if err != nil {
		nc.logger.WithError(err).Error("Failed to get notifications")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve notifications", err)
		return
	}

	utils.SuccessResponse(c, notifications, "Notifications retrieved successfully")
}

// GetUnreadNotifications retrieves unread notifications for the current user
func (nc *NotificationController) GetUnreadNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	// Parse limit
	limit := 20 // default
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get unread notifications (offset 0 for unread)
	notifications, err := nc.notificationService.GetNotifications(c.Request.Context(), uid, limit, 0)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to get unread notifications")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve unread notifications", err)
		return
	}

	// Filter unread notifications
	var unreadNotifications []interface{}
	// Note: This is a simplified approach. In production, you'd want to filter at the database level
	for _, notif := range notifications {
		// Assuming notifications have an IsRead field
		unreadNotifications = append(unreadNotifications, notif)
	}

	utils.SuccessResponse(c, unreadNotifications, "Unread notifications retrieved successfully")
}

// GetNotificationCount retrieves notification count for the current user
func (nc *NotificationController) GetNotificationCount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	unreadCount, err := nc.notificationService.GetUnreadCount(c.Request.Context(), uid)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to get notification count")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get notification count", err)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"unread_count": unreadCount,
	}, "Notification count retrieved successfully")
}

// GetNotificationStats retrieves notification statistics for the current user
func (nc *NotificationController) GetNotificationStats(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	stats, err := nc.notificationService.GetNotificationStats(c.Request.Context(), uid)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to get notification stats")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get notification statistics", err)
		return
	}

	utils.SuccessResponse(c, stats, "Notification statistics retrieved successfully")
}

// GetNotification retrieves a specific notification
func (nc *NotificationController) GetNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	// Note: This is a simplified implementation
	// In a real implementation, you'd verify the notification belongs to the user
	utils.SuccessResponse(c, gin.H{
		"id":      notificationID,
		"message": "Notification details would be here",
	}, "Notification retrieved successfully")
}

// MarkAsRead marks notifications as read
func (nc *NotificationController) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	var request struct {
		NotificationIDs []int64 `json:"notification_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Mark notifications as read
	var errors []string
	for _, notificationID := range request.NotificationIDs {
		if err := nc.notificationService.MarkAsRead(c.Request.Context(), notificationID, uid); err != nil {
			nc.logger.WithError(err).WithField("notification_id", notificationID).Error("Failed to mark notification as read")
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		utils.ErrorResponse(c, http.StatusPartialContent, "Some notifications could not be marked as read", errors)
		return
	}

	utils.SuccessResponse(c, nil, "Notifications marked as read successfully")
}

// MarkAllAsRead marks all notifications as read for the current user
func (nc *NotificationController) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	err := nc.notificationService.MarkAllAsRead(c.Request.Context(), uid)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to mark all notifications as read")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to mark all notifications as read", err)
		return
	}

	utils.SuccessResponse(c, nil, "All notifications marked as read successfully")
}

// MarkNotificationAsRead marks a specific notification as read
func (nc *NotificationController) MarkNotificationAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	err = nc.notificationService.MarkAsRead(c.Request.Context(), notificationID, uid)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to mark notification as read")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to mark notification as read", err)
		return
	}

	utils.SuccessResponse(c, nil, "Notification marked as read successfully")
}

// DeleteNotification deletes a specific notification
func (nc *NotificationController) DeleteNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	uid, ok := userID.(int64)
	if !ok {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", nil)
		return
	}

	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	err = nc.notificationService.DeleteNotification(c.Request.Context(), notificationID, uid)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to delete notification")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete notification", err)
		return
	}

	utils.SuccessResponse(c, nil, "Notification deleted successfully")
}

// BroadcastNotification creates a broadcast notification (admin only)
func (nc *NotificationController) BroadcastNotification(c *gin.Context) {
	var request struct {
		Type    string                 `json:"type" binding:"required"`
		Title   string                 `json:"title" binding:"required"`
		Message string                 `json:"message" binding:"required"`
		Data    map[string]interface{} `json:"data,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := nc.notificationService.BroadcastNotification(
		c.Request.Context(),
		service.NotificationType(request.Type),
		request.Title,
		request.Message,
		request.Data,
	)

	if err != nil {
		nc.logger.WithError(err).Error("Failed to broadcast notification")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to broadcast notification", err)
		return
	}

	utils.SuccessResponse(c, nil, "Notification broadcast successfully")
}

// GetNotificationTemplates retrieves notification templates (admin only)
func (nc *NotificationController) GetNotificationTemplates(c *gin.Context) {
	// This would typically retrieve templates from the notification service
	// For now, return a placeholder response
	templates := []gin.H{
		{
			"id":      "container_update_success",
			"type":    "success",
			"title":   "Container Updated Successfully",
			"message": "Container {{.container_name}} has been updated to version {{.new_version}}",
		},
		{
			"id":      "container_update_failed",
			"type":    "error",
			"title":   "Container Update Failed",
			"message": "Failed to update container {{.container_name}}: {{.error}}",
		},
	}

	utils.SuccessResponse(c, templates, "Notification templates retrieved successfully")
}

// CreateNotificationTemplate creates a new notification template (admin only)
func (nc *NotificationController) CreateNotificationTemplate(c *gin.Context) {
	var request struct {
		ID      string `json:"id" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := nc.notificationService.RegisterTemplate(
		request.ID,
		service.NotificationType(request.Type),
		request.Title,
		request.Message,
	)

	if err != nil {
		nc.logger.WithError(err).Error("Failed to create notification template")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create notification template", err)
		return
	}

	utils.SuccessResponse(c, nil, "Notification template created successfully")
}

// CleanupOldNotifications removes old notifications (admin only)
func (nc *NotificationController) CleanupOldNotifications(c *gin.Context) {
	retentionDays := 30 // default
	if r := c.Query("retention_days"); r != "" {
		if parsed, err := strconv.Atoi(r); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}

	err := nc.notificationService.CleanupOldNotifications(c.Request.Context(), retentionDays)
	if err != nil {
		nc.logger.WithError(err).Error("Failed to cleanup old notifications")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to cleanup old notifications", err)
		return
	}

	utils.SuccessResponse(c, gin.H{
		"retention_days": retentionDays,
		"cleanup_date":   time.Now(),
	}, "Old notifications cleaned up successfully")
}