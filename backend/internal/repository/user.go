package repository

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/model"

	"gorm.io/gorm"
)

// userRepository implements UserRepository interface
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}

	// Validate required fields
	if user.Username == "" {
		return fmt.Errorf("username is required")
	}
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}
	if user.PasswordHash == "" {
		return fmt.Errorf("password hash is required")
	}

	// Check for existing user
	exists, err := r.Exists(ctx, user.Username, user.Email)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return fmt.Errorf("user with username '%s' or email '%s' already exists", user.Username, user.Email)
	}

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", id)
	}

	var user model.User
	err := r.db.WithContext(ctx).
		Preload("Containers").
		First(&user, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with username '%s' not found", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	var user model.User
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with email '%s' not found", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	if user.ID <= 0 {
		return fmt.Errorf("invalid user ID: %d", user.ID)
	}

	// Check if user exists
	var existingUser model.User
	if err := r.db.WithContext(ctx).First(&existingUser, user.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user with ID %d not found", user.ID)
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// Check for duplicate username/email if changed
	if user.Username != existingUser.Username || user.Email != existingUser.Email {
		var count int64
		query := r.db.WithContext(ctx).Model(&model.User{}).
			Where("id != ?", user.ID).
			Where("username = ? OR email = ?", user.Username, user.Email)

		if err := query.Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check for duplicates: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("username or email already exists")
		}
	}

	// Update timestamp manually since GORM might not update it on selective updates
	user.UpdatedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete deletes a user by ID
func (r *userRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid user ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user with ID %d not found", id)
	}

	return nil
}

// List retrieves users with filtering and pagination
func (r *userRepository) List(ctx context.Context, filter *model.UserFilter) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := r.db.WithContext(ctx).Model(&model.User{})

	// Apply filters
	if filter != nil {
		if filter.Username != "" {
			query = query.Where("username ILIKE ?", "%"+filter.Username+"%")
		}
		if filter.Email != "" {
			query = query.Where("email ILIKE ?", "%"+filter.Email+"%")
		}
		if filter.Role != "" {
			query = query.Where("role = ?", filter.Role)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Apply ordering
	orderBy := "created_at DESC"
	if filter != nil && filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	query = query.Order(orderBy)

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	// Preload relationships
	query = query.Preload("Containers")

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// GetActiveUsers gets all active users
func (r *userRepository) GetActiveUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User

	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	return users, nil
}

// Exists checks if a user with given username or email exists
func (r *userRepository) Exists(ctx context.Context, username, email string) (bool, error) {
	if username == "" && email == "" {
		return false, fmt.Errorf("username and email cannot both be empty")
	}

	var count int64
	query := r.db.WithContext(ctx).Model(&model.User{})

	if username != "" && email != "" {
		query = query.Where("username = ? OR email = ?", username, email)
	} else if username != "" {
		query = query.Where("username = ?", username)
	} else {
		query = query.Where("email = ?", email)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return count > 0, nil
}

// UpdateLastLoginAt updates the last login timestamp for a user
func (r *userRepository) UpdateLastLoginAt(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID: %d", userID)
	}

	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": &now,
			"updated_at":    now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update last login time: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user with ID %d not found", userID)
	}

	return nil
}

// SetUserStatus sets the active status of a user
func (r *userRepository) SetUserStatus(ctx context.Context, userID int64, isActive bool) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID: %d", userID)
	}

	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"is_active":  isActive,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update user status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("user with ID %d not found", userID)
	}

	return nil
}

// CreateBatch creates multiple users in a single transaction
func (r *userRepository) CreateBatch(ctx context.Context, users []*model.User) error {
	if len(users) == 0 {
		return fmt.Errorf("users slice cannot be empty")
	}

	// Validate all users before creating
	for i, user := range users {
		if user == nil {
			return fmt.Errorf("user at index %d cannot be nil", i)
		}
		if user.Username == "" {
			return fmt.Errorf("username is required for user at index %d", i)
		}
		if user.Email == "" {
			return fmt.Errorf("email is required for user at index %d", i)
		}
	}

	// Use transaction for batch create
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check for existing users
		var usernames, emails []string
		for _, user := range users {
			usernames = append(usernames, user.Username)
			emails = append(emails, user.Email)
		}

		var existingCount int64
		if err := tx.Model(&model.User{}).
			Where("username IN ? OR email IN ?", usernames, emails).
			Count(&existingCount).Error; err != nil {
			return fmt.Errorf("failed to check existing users: %w", err)
		}

		if existingCount > 0 {
			return fmt.Errorf("one or more users already exist")
		}

		// Create all users
		if err := tx.CreateInBatches(users, 100).Error; err != nil {
			return fmt.Errorf("failed to create users batch: %w", err)
		}

		return nil
	})
}

// GetByIDs retrieves multiple users by their IDs
func (r *userRepository) GetByIDs(ctx context.Context, ids []int64) ([]*model.User, error) {
	if len(ids) == 0 {
		return []*model.User{}, nil
	}

	// Validate IDs
	for i, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("invalid user ID at index %d: %d", i, id)
		}
	}

	var users []*model.User
	err := r.db.WithContext(ctx).
		Preload("Containers").
		Where("id IN ?", ids).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w", err)
	}

	return users, nil
}

// userSessionRepository implements UserSessionRepository interface
type userSessionRepository struct {
	db *gorm.DB
}

// NewUserSessionRepository creates a new user session repository
func NewUserSessionRepository(db *gorm.DB) UserSessionRepository {
	return &userSessionRepository{db: db}
}

// Create creates a new user session
func (r *userSessionRepository) Create(ctx context.Context, session *model.UserSession) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if session.UserID <= 0 {
		return fmt.Errorf("invalid user ID: %d", session.UserID)
	}
	if session.RefreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create user session: %w", err)
	}

	return nil
}

// GetByID retrieves a user session by ID
func (r *userSessionRepository) GetByID(ctx context.Context, id string) (*model.UserSession, error) {
	if id == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	var session model.UserSession
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&session, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session with ID '%s' not found", id)
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}

	return &session, nil
}

// GetByRefreshToken retrieves a user session by refresh token
func (r *userSessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*model.UserSession, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token cannot be empty")
	}

	var session model.UserSession
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("refresh_token = ?", refreshToken).
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("session with refresh token not found")
		}
		return nil, fmt.Errorf("failed to get session by refresh token: %w", err)
	}

	return &session, nil
}

// Update updates an existing user session
func (r *userSessionRepository) Update(ctx context.Context, session *model.UserSession) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}
	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("failed to update user session: %w", err)
	}

	return nil
}

// Delete deletes a user session by ID
func (r *userSessionRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	result := r.db.WithContext(ctx).Delete(&model.UserSession{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user session: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("session with ID '%s' not found", id)
	}

	return nil
}

// GetByUserID retrieves all sessions for a user
func (r *userSessionRepository) GetByUserID(ctx context.Context, userID int64) ([]*model.UserSession, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", userID)
	}

	var sessions []*model.UserSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&sessions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user ID: %w", err)
	}

	return sessions, nil
}

// DeleteExpiredSessions deletes all expired sessions
func (r *userSessionRepository) DeleteExpiredSessions(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&model.UserSession{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", result.Error)
	}

	return nil
}

// DeleteUserSessions deletes all sessions for a specific user
func (r *userSessionRepository) DeleteUserSessions(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID: %d", userID)
	}

	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.UserSession{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete user sessions: %w", result.Error)
	}

	return nil
}

// IsValidSession checks if a session with given refresh token is valid
func (r *userSessionRepository) IsValidSession(ctx context.Context, refreshToken string) (bool, error) {
	if refreshToken == "" {
		return false, fmt.Errorf("refresh token cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.UserSession{}).
		Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now().UTC()).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check session validity: %w", err)
	}

	return count > 0, nil
}

// CleanupExpiredSessions removes expired sessions and returns count of deleted sessions
func (r *userSessionRepository) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&model.UserSession{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// activityLogRepository implements ActivityLogRepository interface
type activityLogRepository struct {
	db *gorm.DB
}

// NewActivityLogRepository creates a new activity log repository
func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &activityLogRepository{db: db}
}

// Create creates a new activity log
func (r *activityLogRepository) Create(ctx context.Context, log *model.ActivityLog) error {
	if log == nil {
		return fmt.Errorf("activity log cannot be nil")
	}

	if log.Action == "" {
		return fmt.Errorf("action is required")
	}
	if log.ResourceType == "" {
		return fmt.Errorf("resource type is required")
	}

	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	return nil
}

// GetByID retrieves an activity log by ID
func (r *activityLogRepository) GetByID(ctx context.Context, id int64) (*model.ActivityLog, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid activity log ID: %d", id)
	}

	var log model.ActivityLog
	err := r.db.WithContext(ctx).
		Preload("User").
		First(&log, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("activity log with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get activity log by ID: %w", err)
	}

	return &log, nil
}

// Delete deletes an activity log by ID
func (r *activityLogRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid activity log ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.ActivityLog{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete activity log: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("activity log with ID %d not found", id)
	}

	return nil
}

// List retrieves activity logs with filtering and pagination
func (r *activityLogRepository) List(ctx context.Context, filter *model.ActivityLogFilter) ([]*model.ActivityLog, int64, error) {
	var logs []*model.ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ActivityLog{})

	// Apply filters
	if filter != nil {
		if filter.UserID != nil {
			query = query.Where("user_id = ?", *filter.UserID)
		}
		if filter.Action != "" {
			query = query.Where("action ILIKE ?", "%"+filter.Action+"%")
		}
		if filter.ResourceType != "" {
			query = query.Where("resource_type = ?", filter.ResourceType)
		}
		if filter.ResourceID != nil {
			query = query.Where("resource_id = ?", *filter.ResourceID)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}

	// Apply ordering
	orderBy := "created_at DESC"
	if filter != nil && filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	query = query.Order(orderBy)

	// Apply pagination
	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	// Preload relationships
	query = query.Preload("User")

	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list activity logs: %w", err)
	}

	return logs, total, nil
}

// GetByUserID retrieves activity logs for a specific user
func (r *activityLogRepository) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*model.ActivityLog, int64, error) {
	if userID <= 0 {
		return nil, 0, fmt.Errorf("invalid user ID: %d", userID)
	}

	var logs []*model.ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ActivityLog{}).
		Where("user_id = ?", userID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs for user: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get activity logs by user ID: %w", err)
	}

	return logs, total, nil
}

// GetByResourceID retrieves activity logs for a specific resource
func (r *activityLogRepository) GetByResourceID(ctx context.Context, resourceType string, resourceID int64) ([]*model.ActivityLog, error) {
	if resourceType == "" {
		return nil, fmt.Errorf("resource type cannot be empty")
	}
	if resourceID <= 0 {
		return nil, fmt.Errorf("invalid resource ID: %d", resourceID)
	}

	var logs []*model.ActivityLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get activity logs by resource ID: %w", err)
	}

	return logs, nil
}

// DeleteOldLogs deletes activity logs older than specified retention days
func (r *activityLogRepository) DeleteOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&model.ActivityLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old activity logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// CreateBatch creates multiple activity logs in a single transaction
func (r *activityLogRepository) CreateBatch(ctx context.Context, logs []*model.ActivityLog) error {
	if len(logs) == 0 {
		return fmt.Errorf("logs slice cannot be empty")
	}

	// Validate all logs before creating
	for i, log := range logs {
		if log == nil {
			return fmt.Errorf("activity log at index %d cannot be nil", i)
		}
		if log.Action == "" {
			return fmt.Errorf("action is required for log at index %d", i)
		}
		if log.ResourceType == "" {
			return fmt.Errorf("resource type is required for log at index %d", i)
		}
	}

	if err := r.db.WithContext(ctx).CreateInBatches(logs, 100).Error; err != nil {
		return fmt.Errorf("failed to create activity logs batch: %w", err)
	}

	return nil
}

// CountOlderThan counts activity logs older than the specified date
func (r *activityLogRepository) CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.ActivityLog{}).
		Where("created_at < ?", cutoffDate).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count old activity logs: %w", err)
	}

	return count, nil
}

// DeleteOlderThan deletes activity logs older than the specified date
func (r *activityLogRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&model.ActivityLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old activity logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}