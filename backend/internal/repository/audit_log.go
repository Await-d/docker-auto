package repository

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/model"

	"gorm.io/gorm"
)

// AuditLogRepository defines the interface for audit log operations
type AuditLogRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, auditLog *model.AuditLog) error
	GetByID(ctx context.Context, id int64) (*model.AuditLog, error)
	GetByEventID(ctx context.Context, eventID string) (*model.AuditLog, error)
	Update(ctx context.Context, auditLog *model.AuditLog) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	List(ctx context.Context, filter *AuditLogFilter) ([]*model.AuditLog, error)
	Count(ctx context.Context, filter *AuditLogFilter) (int64, error)
	GetByUserID(ctx context.Context, userID int64, filter *AuditLogFilter) ([]*model.AuditLog, error)
	GetByContainerID(ctx context.Context, containerID string, filter *AuditLogFilter) ([]*model.AuditLog, error)
	GetByEventType(ctx context.Context, eventType string, filter *AuditLogFilter) ([]*model.AuditLog, error)
	GetBySeverity(ctx context.Context, severity string, filter *AuditLogFilter) ([]*model.AuditLog, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filter *AuditLogFilter) ([]*model.AuditLog, error)

	// Batch operations
	CreateBatch(ctx context.Context, auditLogs []*model.AuditLog) error
	DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error)

	// Analytics operations
	GetEventTypeStats(ctx context.Context, timeRange *TimeRange) (map[string]int64, error)
	GetSeverityStats(ctx context.Context, timeRange *TimeRange) (map[string]int64, error)
	GetUserActivityStats(ctx context.Context, userID int64, timeRange *TimeRange) (*UserActivityStats, error)
	GetContainerActivityStats(ctx context.Context, containerID string, timeRange *TimeRange) (*ContainerActivityStats, error)
}

// AuditLogFilter represents filtering options for audit log queries
type AuditLogFilter struct {
	EventTypes   []string    `json:"event_types,omitempty"`
	Severities   []string    `json:"severities,omitempty"`
	Sources      []string    `json:"sources,omitempty"`
	UserID       *int64      `json:"user_id,omitempty"`
	ContainerID  *string     `json:"container_id,omitempty"`
	Actions      []string    `json:"actions,omitempty"`
	Results      []string    `json:"results,omitempty"`
	StartTime    *time.Time  `json:"start_time,omitempty"`
	EndTime      *time.Time  `json:"end_time,omitempty"`
	RemoteIP     *string     `json:"remote_ip,omitempty"`
	SearchTerm   *string     `json:"search_term,omitempty"`
	Limit        int         `json:"limit"`
	Offset       int         `json:"offset"`
	SortBy       string      `json:"sort_by"`
	SortOrder    string      `json:"sort_order"`
}

// TimeRange represents a time range for analytics queries
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// UserActivityStats represents user activity statistics
type UserActivityStats struct {
	UserID         int64             `json:"user_id"`
	TotalEvents    int64             `json:"total_events"`
	EventsByType   map[string]int64  `json:"events_by_type"`
	EventsBySeverity map[string]int64 `json:"events_by_severity"`
	LastActivity   time.Time         `json:"last_activity"`
	FirstActivity  time.Time         `json:"first_activity"`
}

// ContainerActivityStats represents container activity statistics
type ContainerActivityStats struct {
	ContainerID    string            `json:"container_id"`
	TotalEvents    int64             `json:"total_events"`
	EventsByType   map[string]int64  `json:"events_by_type"`
	EventsBySeverity map[string]int64 `json:"events_by_severity"`
	LastActivity   time.Time         `json:"last_activity"`
	FirstActivity  time.Time         `json:"first_activity"`
}

// auditLogRepository implements AuditLogRepository interface
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{
		db: db,
	}
}

// Create creates a new audit log entry
func (r *auditLogRepository) Create(ctx context.Context, auditLog *model.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(auditLog).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// GetByID retrieves an audit log by ID
func (r *auditLogRepository) GetByID(ctx context.Context, id int64) (*model.AuditLog, error) {
	var auditLog model.AuditLog
	if err := r.db.WithContext(ctx).Preload("User").First(&auditLog, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get audit log by ID: %w", err)
	}
	return &auditLog, nil
}

// GetByEventID retrieves an audit log by event ID
func (r *auditLogRepository) GetByEventID(ctx context.Context, eventID string) (*model.AuditLog, error) {
	var auditLog model.AuditLog
	if err := r.db.WithContext(ctx).Preload("User").Where("event_id = ?", eventID).First(&auditLog).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get audit log by event ID: %w", err)
	}
	return &auditLog, nil
}

// Update updates an audit log entry
func (r *auditLogRepository) Update(ctx context.Context, auditLog *model.AuditLog) error {
	if err := r.db.WithContext(ctx).Save(auditLog).Error; err != nil {
		return fmt.Errorf("failed to update audit log: %w", err)
	}
	return nil
}

// Delete deletes an audit log entry
func (r *auditLogRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&model.AuditLog{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete audit log: %w", err)
	}
	return nil
}

// List retrieves audit logs with filtering
func (r *auditLogRepository) List(ctx context.Context, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	query := r.db.WithContext(ctx).Preload("User")
	query = r.applyFilter(query, filter)

	var auditLogs []*model.AuditLog
	if err := query.Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return auditLogs, nil
}

// Count counts audit logs with filtering
func (r *auditLogRepository) Count(ctx context.Context, filter *AuditLogFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditLog{})
	query = r.applyFilter(query, filter)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}

// GetByUserID retrieves audit logs for a specific user
func (r *auditLogRepository) GetByUserID(ctx context.Context, userID int64, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	filter.UserID = &userID
	return r.List(ctx, filter)
}

// GetByContainerID retrieves audit logs for a specific container
func (r *auditLogRepository) GetByContainerID(ctx context.Context, containerID string, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	filter.ContainerID = &containerID
	return r.List(ctx, filter)
}

// GetByEventType retrieves audit logs for a specific event type
func (r *auditLogRepository) GetByEventType(ctx context.Context, eventType string, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	filter.EventTypes = []string{eventType}
	return r.List(ctx, filter)
}

// GetBySeverity retrieves audit logs for a specific severity
func (r *auditLogRepository) GetBySeverity(ctx context.Context, severity string, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	filter.Severities = []string{severity}
	return r.List(ctx, filter)
}

// GetByTimeRange retrieves audit logs within a time range
func (r *auditLogRepository) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, filter *AuditLogFilter) ([]*model.AuditLog, error) {
	if filter == nil {
		filter = &AuditLogFilter{}
	}
	filter.StartTime = &startTime
	filter.EndTime = &endTime
	return r.List(ctx, filter)
}

// CreateBatch creates multiple audit log entries in a batch
func (r *auditLogRepository) CreateBatch(ctx context.Context, auditLogs []*model.AuditLog) error {
	if len(auditLogs) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(auditLogs, 1000).Error; err != nil {
		return fmt.Errorf("failed to create audit logs batch: %w", err)
	}
	return nil
}

// DeleteOldLogs deletes audit logs older than the specified time
func (r *auditLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("timestamp < ?", olderThan).Delete(&model.AuditLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// GetEventTypeStats returns statistics grouped by event type
func (r *auditLogRepository) GetEventTypeStats(ctx context.Context, timeRange *TimeRange) (map[string]int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("event_type, COUNT(*) as count").
		Group("event_type")

	if timeRange != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", timeRange.StartTime, timeRange.EndTime)
	}

	var results []struct {
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get event type stats: %w", err)
	}

	stats := make(map[string]int64)
	for _, result := range results {
		stats[result.EventType] = result.Count
	}

	return stats, nil
}

// GetSeverityStats returns statistics grouped by severity
func (r *auditLogRepository) GetSeverityStats(ctx context.Context, timeRange *TimeRange) (map[string]int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).
		Select("severity, COUNT(*) as count").
		Group("severity")

	if timeRange != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", timeRange.StartTime, timeRange.EndTime)
	}

	var results []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get severity stats: %w", err)
	}

	stats := make(map[string]int64)
	for _, result := range results {
		stats[result.Severity] = result.Count
	}

	return stats, nil
}

// GetUserActivityStats returns activity statistics for a user
func (r *auditLogRepository) GetUserActivityStats(ctx context.Context, userID int64, timeRange *TimeRange) (*UserActivityStats, error) {
	// Get total events
	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("user_id = ?", userID)
	if timeRange != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", timeRange.StartTime, timeRange.EndTime)
	}

	var totalEvents int64
	if err := query.Count(&totalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count user events: %w", err)
	}

	// Get events by type
	var eventTypeResults []struct {
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}

	eventTypeQuery := query.Select("event_type, COUNT(*) as count").Group("event_type")
	if err := eventTypeQuery.Scan(&eventTypeResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get user event type stats: %w", err)
	}

	eventsByType := make(map[string]int64)
	for _, result := range eventTypeResults {
		eventsByType[result.EventType] = result.Count
	}

	// Get events by severity
	var severityResults []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	severityQuery := query.Select("severity, COUNT(*) as count").Group("severity")
	if err := severityQuery.Scan(&severityResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get user severity stats: %w", err)
	}

	eventsBySeverity := make(map[string]int64)
	for _, result := range severityResults {
		eventsBySeverity[result.Severity] = result.Count
	}

	// Get first and last activity
	var firstActivity, lastActivity time.Time

	if err := query.Select("MIN(timestamp)").Scan(&firstActivity).Error; err != nil {
		return nil, fmt.Errorf("failed to get first activity: %w", err)
	}

	if err := query.Select("MAX(timestamp)").Scan(&lastActivity).Error; err != nil {
		return nil, fmt.Errorf("failed to get last activity: %w", err)
	}

	return &UserActivityStats{
		UserID:           userID,
		TotalEvents:      totalEvents,
		EventsByType:     eventsByType,
		EventsBySeverity: eventsBySeverity,
		FirstActivity:    firstActivity,
		LastActivity:     lastActivity,
	}, nil
}

// GetContainerActivityStats returns activity statistics for a container
func (r *auditLogRepository) GetContainerActivityStats(ctx context.Context, containerID string, timeRange *TimeRange) (*ContainerActivityStats, error) {
	// Get total events
	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Where("container_id = ?", containerID)
	if timeRange != nil {
		query = query.Where("timestamp BETWEEN ? AND ?", timeRange.StartTime, timeRange.EndTime)
	}

	var totalEvents int64
	if err := query.Count(&totalEvents).Error; err != nil {
		return nil, fmt.Errorf("failed to count container events: %w", err)
	}

	// Get events by type
	var eventTypeResults []struct {
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}

	eventTypeQuery := query.Select("event_type, COUNT(*) as count").Group("event_type")
	if err := eventTypeQuery.Scan(&eventTypeResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get container event type stats: %w", err)
	}

	eventsByType := make(map[string]int64)
	for _, result := range eventTypeResults {
		eventsByType[result.EventType] = result.Count
	}

	// Get events by severity
	var severityResults []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	severityQuery := query.Select("severity, COUNT(*) as count").Group("severity")
	if err := severityQuery.Scan(&severityResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get container severity stats: %w", err)
	}

	eventsBySeverity := make(map[string]int64)
	for _, result := range severityResults {
		eventsBySeverity[result.Severity] = result.Count
	}

	// Get first and last activity
	var firstActivity, lastActivity time.Time

	if err := query.Select("MIN(timestamp)").Scan(&firstActivity).Error; err != nil {
		return nil, fmt.Errorf("failed to get first activity: %w", err)
	}

	if err := query.Select("MAX(timestamp)").Scan(&lastActivity).Error; err != nil {
		return nil, fmt.Errorf("failed to get last activity: %w", err)
	}

	return &ContainerActivityStats{
		ContainerID:      containerID,
		TotalEvents:      totalEvents,
		EventsByType:     eventsByType,
		EventsBySeverity: eventsBySeverity,
		FirstActivity:    firstActivity,
		LastActivity:     lastActivity,
	}, nil
}

// applyFilter applies filtering conditions to the query
func (r *auditLogRepository) applyFilter(query *gorm.DB, filter *AuditLogFilter) *gorm.DB {
	if filter == nil {
		return query.Order("timestamp DESC").Limit(100) // Default limit
	}

	// Apply filters
	if len(filter.EventTypes) > 0 {
		query = query.Where("event_type IN ?", filter.EventTypes)
	}

	if len(filter.Severities) > 0 {
		query = query.Where("severity IN ?", filter.Severities)
	}

	if len(filter.Sources) > 0 {
		query = query.Where("source IN ?", filter.Sources)
	}

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}

	if filter.ContainerID != nil {
		query = query.Where("container_id = ?", *filter.ContainerID)
	}

	if len(filter.Actions) > 0 {
		query = query.Where("action IN ?", filter.Actions)
	}

	if len(filter.Results) > 0 {
		query = query.Where("result IN ?", filter.Results)
	}

	if filter.StartTime != nil {
		query = query.Where("timestamp >= ?", *filter.StartTime)
	}

	if filter.EndTime != nil {
		query = query.Where("timestamp <= ?", *filter.EndTime)
	}

	if filter.RemoteIP != nil {
		query = query.Where("remote_ip = ?", *filter.RemoteIP)
	}

	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		searchPattern := "%" + *filter.SearchTerm + "%"
		query = query.Where("(action ILIKE ? OR resource ILIKE ? OR details ILIKE ?)",
			searchPattern, searchPattern, searchPattern)
	}

	// Apply sorting
	if filter.SortBy != "" {
		sortOrder := "ASC"
		if filter.SortOrder == "DESC" || filter.SortOrder == "desc" {
			sortOrder = "DESC"
		}
		query = query.Order(filter.SortBy + " " + sortOrder)
	} else {
		query = query.Order("timestamp DESC")
	}

	// Apply pagination
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(100) // Default limit
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	return query
}