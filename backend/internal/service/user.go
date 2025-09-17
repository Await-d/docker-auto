package service

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/utils"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// UserService manages user authentication and operations
type UserService struct {
	userRepo     repository.UserRepository
	sessionRepo  repository.UserSessionRepository
	activityRepo repository.ActivityLogRepository
	config       *config.Config
	cache        *CacheService
	jwtManager   *utils.JWTManager
}

// NewUserService creates a new user service instance
func NewUserService(
	userRepo repository.UserRepository,
	sessionRepo repository.UserSessionRepository,
	activityRepo repository.ActivityLogRepository,
	config *config.Config,
	cache *CacheService,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		activityRepo: activityRepo,
		config:       config,
		cache:        cache,
		jwtManager:   utils.NewJWTManager(config),
	}
}

// Authentication related methods

// Login authenticates a user and returns login response with tokens
func (s *UserService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("login request cannot be nil")
	}

	// Validate input
	if err := s.validateLoginRequest(req); err != nil {
		return nil, fmt.Errorf("invalid login request: %w", err)
	}

	// Get user by username or email
	user, err := s.validateLogin(req.Username, req.Password)
	if err != nil {
		// Log failed login attempt
		s.logUserActivity(0, "login_failed", fmt.Sprintf("Failed login attempt for username: %s", req.Username), nil)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		s.logUserActivity(user.ID, "login_blocked", "Login blocked for inactive user", nil)
		return nil, fmt.Errorf("user account is inactive")
	}

	// Generate token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(user)
	if err != nil {
		s.logUserActivity(user.ID, "token_generation_failed", "Failed to generate tokens", nil)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create user session
	if err := s.createUserSession(user.ID, tokenPair.RefreshToken); err != nil {
		logrus.WithError(err).WithField("user_id", user.ID).Warn("Failed to create user session")
		// Don't fail login if session creation fails
	}

	// Update last login time
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		logrus.WithError(err).WithField("user_id", user.ID).Warn("Failed to update last login time")
	}

	// Log successful login
	s.logUserActivity(user.ID, "login_success", "User logged in successfully", map[string]interface{}{
		"remember": req.Remember,
	})

	return &LoginResponse{
		User:         s.userToResponse(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// Logout invalidates user session and tokens
func (s *UserService) Logout(ctx context.Context, userID int64, sessionID string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	// Revoke session if provided
	if sessionID != "" {
		if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
			logrus.WithError(err).WithField("session_id", sessionID).Warn("Failed to delete session")
		}
	}

	// Clear user cache
	s.invalidateUserCache(userID)

	// Log logout
	s.logUserActivity(userID, "logout", "User logged out", map[string]interface{}{
		"session_id": sessionID,
	})

	return nil
}

// RefreshToken generates new access token using refresh token
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}

	// Validate refresh token
	refreshClaims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if session exists and is valid
	session, err := s.sessionRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.IsExpired() {
		// Clean up expired session
		s.sessionRepo.Delete(ctx, session.ID)
		return nil, fmt.Errorf("refresh token has expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, refreshClaims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if user is still active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	// Log token refresh
	s.logUserActivity(user.ID, "token_refresh", "Access token refreshed", nil)

	return &TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Keep the same refresh token
		ExpiresIn:    int64(time.Duration(s.config.JWT.ExpireHours) * time.Hour / time.Second),
	}, nil
}

// ValidateToken validates JWT token and returns user
func (s *UserService) ValidateToken(ctx context.Context, token string) (*model.User, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Validate and parse token
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check cache first
	if cachedUser := s.getCachedUser(claims.UserID); cachedUser != nil {
		return cachedUser, nil
	}

	// Get user from database
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Verify user is still active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Cache user for future requests
	s.cacheUser(user)

	return user, nil
}

// User management methods

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, req *RegisterRequest) (*model.User, error) {
	if req == nil {
		return nil, fmt.Errorf("register request cannot be nil")
	}

	// Validate input
	if err := s.validateRegisterRequest(req); err != nil {
		return nil, fmt.Errorf("invalid register request: %w", err)
	}

	// Check if user already exists
	exists, err := s.userRepo.Exists(ctx, req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with username or email already exists")
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Set default role if not provided
	role := model.UserRoleViewer
	if req.Role != "" {
		role = model.UserRole(req.Role)
	}

	// Create user
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         role,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log user registration
	s.logUserActivity(user.ID, "user_registered", "New user registered", map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})

	return user, nil
}

// GetCurrentUser retrieves current user information
func (s *UserService) GetCurrentUser(ctx context.Context, userID int64) (*model.User, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Check cache first
	if cachedUser := s.getCachedUser(userID); cachedUser != nil {
		return cachedUser, nil
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Cache user
	s.cacheUser(user)

	return user, nil
}

// UpdateProfile updates user profile information
func (s *UserService) UpdateProfile(ctx context.Context, userID int64, req *UpdateProfileRequest) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	if req == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	// Get current user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update fields if provided
	updated := false
	changes := make(map[string]interface{})

	if req.Username != nil && *req.Username != user.Username {
		// Check if new username is available
		if exists, _ := s.userRepo.Exists(ctx, *req.Username, ""); exists {
			return fmt.Errorf("username already exists")
		}
		user.Username = *req.Username
		changes["username"] = *req.Username
		updated = true
	}

	if req.Email != nil && *req.Email != user.Email {
		// Check if new email is available
		if exists, _ := s.userRepo.Exists(ctx, "", *req.Email); exists {
			return fmt.Errorf("email already exists")
		}
		user.Email = *req.Email
		changes["email"] = *req.Email
		updated = true
	}

	if req.AvatarURL != nil && *req.AvatarURL != user.AvatarURL {
		user.AvatarURL = *req.AvatarURL
		changes["avatar_url"] = *req.AvatarURL
		updated = true
	}

	if !updated {
		return nil // No changes to update
	}

	// Save changes
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log profile update
	s.logUserActivity(userID, "profile_updated", "User profile updated", changes)

	return nil
}

// ChangePassword changes user password
func (s *UserService) ChangePassword(ctx context.Context, userID int64, req *ChangePasswordRequest) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	if req == nil {
		return fmt.Errorf("change password request cannot be nil")
	}

	// Validate input
	if err := s.validateChangePasswordRequest(req); err != nil {
		return fmt.Errorf("invalid change password request: %w", err)
	}

	// Get current user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		s.logUserActivity(userID, "password_change_failed", "Invalid old password provided", nil)
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all user sessions (force re-login)
	if err := s.sessionRepo.DeleteUserSessions(ctx, userID); err != nil {
		logrus.WithError(err).WithField("user_id", userID).Warn("Failed to revoke user sessions")
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log password change
	s.logUserActivity(userID, "password_changed", "User password changed", nil)

	return nil
}

// DeactivateUser deactivates a user account
func (s *UserService) DeactivateUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	// Set user as inactive
	if err := s.userRepo.SetUserStatus(ctx, userID, false); err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Revoke all user sessions
	if err := s.sessionRepo.DeleteUserSessions(ctx, userID); err != nil {
		logrus.WithError(err).WithField("user_id", userID).Warn("Failed to revoke user sessions")
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log user deactivation
	s.logUserActivity(userID, "user_deactivated", "User account deactivated", nil)

	return nil
}

// User query and management methods

// ListUsers retrieves users with filtering and pagination
func (s *UserService) ListUsers(ctx context.Context, filter *model.UserFilter) ([]*model.User, int64, error) {
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	return s.userRepo.GetByID(ctx, userID)
}

// CreateUser creates a new user (admin operation)
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*model.User, error) {
	if req == nil {
		return nil, fmt.Errorf("create user request cannot be nil")
	}

	// Validate input
	if err := s.validateCreateUserRequest(req); err != nil {
		return nil, fmt.Errorf("invalid create user request: %w", err)
	}

	// Check if user already exists
	exists, err := s.userRepo.Exists(ctx, req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user with username or email already exists")
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         model.UserRole(req.Role),
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log user creation
	s.logUserActivity(user.ID, "user_created", "User created by admin", map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})

	return user, nil
}

// UpdateUser updates user information (admin operation)
func (s *UserService) UpdateUser(ctx context.Context, userID int64, req *UpdateUserRequest) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	if req == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	// Get current user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update fields if provided
	updated := false
	changes := make(map[string]interface{})

	if req.Username != nil && *req.Username != user.Username {
		if exists, _ := s.userRepo.Exists(ctx, *req.Username, ""); exists {
			return fmt.Errorf("username already exists")
		}
		user.Username = *req.Username
		changes["username"] = *req.Username
		updated = true
	}

	if req.Email != nil && *req.Email != user.Email {
		if exists, _ := s.userRepo.Exists(ctx, "", *req.Email); exists {
			return fmt.Errorf("email already exists")
		}
		user.Email = *req.Email
		changes["email"] = *req.Email
		updated = true
	}

	if req.Role != nil && model.UserRole(*req.Role) != user.Role {
		user.Role = model.UserRole(*req.Role)
		changes["role"] = *req.Role
		updated = true
	}

	if req.IsActive != nil && *req.IsActive != user.IsActive {
		user.IsActive = *req.IsActive
		changes["is_active"] = *req.IsActive
		updated = true

		// If user is being deactivated, revoke all sessions
		if !*req.IsActive {
			s.sessionRepo.DeleteUserSessions(ctx, userID)
		}
	}

	if req.AvatarURL != nil && *req.AvatarURL != user.AvatarURL {
		user.AvatarURL = *req.AvatarURL
		changes["avatar_url"] = *req.AvatarURL
		updated = true
	}

	if !updated {
		return nil // No changes to update
	}

	// Save changes
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log user update
	s.logUserActivity(userID, "user_updated", "User updated by admin", changes)

	return nil
}

// Permission management methods

// CheckPermission checks if user has specific permission
func (s *UserService) CheckPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	if userID <= 0 {
		return false, fmt.Errorf("invalid user ID")
	}
	if permission == "" {
		return false, fmt.Errorf("permission cannot be empty")
	}

	// Check cache first
	if cachedPermissions := s.getCachedPermissions(userID); cachedPermissions != nil {
		return s.hasPermissionInList(cachedPermissions, permission), nil
	}

	// Get user to determine role
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	// Check if user has permission based on role
	hasPermission := s.hasPermission(string(user.Role), permission)

	// Cache user permissions
	permissions := s.getUserPermissions(string(user.Role))
	s.cachePermissions(userID, permissions)

	return hasPermission, nil
}

// GetUserPermissions returns all permissions for a user
func (s *UserService) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Check cache first
	if cachedPermissions := s.getCachedPermissions(userID); cachedPermissions != nil {
		return cachedPermissions, nil
	}

	// Get user to determine role
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get permissions for user role
	permissions := s.getUserPermissions(string(user.Role))

	// Cache permissions
	s.cachePermissions(userID, permissions)

	return permissions, nil
}

// ChangeUserRole changes user role (admin operation)
func (s *UserService) ChangeUserRole(ctx context.Context, userID int64, newRole string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}
	if newRole == "" {
		return fmt.Errorf("role cannot be empty")
	}

	// Validate role
	validRoles := model.GetValidRoles()
	roleValid := false
	for _, role := range validRoles {
		if string(role) == newRole {
			roleValid = true
			break
		}
	}
	if !roleValid {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	// Get current user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	oldRole := string(user.Role)
	if oldRole == newRole {
		return nil // No change needed
	}

	// Update role
	user.Role = model.UserRole(newRole)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log role change
	s.logUserActivity(userID, "role_changed", "User role changed", map[string]interface{}{
		"old_role": oldRole,
		"new_role": newRole,
	})

	return nil
}

// Session management methods

// GetActiveSessions retrieves active sessions for a user
func (s *UserService) GetActiveSessions(ctx context.Context, userID int64) ([]*model.UserSession, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	sessions, err := s.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Filter out expired sessions
	var activeSessions []*model.UserSession
	for _, session := range sessions {
		if !session.IsExpired() {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

// RevokeSession revokes a specific session
func (s *UserService) RevokeSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Get session to log which user
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Log session revocation
	s.logUserActivity(session.UserID, "session_revoked", "User session revoked", map[string]interface{}{
		"session_id": sessionID,
	})

	return nil
}

// RevokeAllSessions revokes all sessions for a user
func (s *UserService) RevokeAllSessions(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	// Delete all user sessions
	if err := s.sessionRepo.DeleteUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// Clear cache
	s.invalidateUserCache(userID)

	// Log session revocation
	s.logUserActivity(userID, "all_sessions_revoked", "All user sessions revoked", nil)

	return nil
}

// Activity logging methods

// LogActivity logs user activity
func (s *UserService) LogActivity(ctx context.Context, activity *model.ActivityLog) error {
	if activity == nil {
		return fmt.Errorf("activity log cannot be nil")
	}

	return s.activityRepo.Create(ctx, activity)
}

// GetUserActivities retrieves user activity logs
func (s *UserService) GetUserActivities(ctx context.Context, userID int64, filter *ActivityFilter) ([]*model.ActivityLog, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	// Convert filter
	modelFilter := &model.ActivityLogFilter{
		UserID:  &userID,
		Limit:   filter.Limit,
		Offset:  filter.Page * filter.Limit,
		OrderBy: "created_at DESC",
	}

	if filter.Action != "" {
		modelFilter.Action = filter.Action
	}

	logs, _, err := s.activityRepo.List(ctx, modelFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activities: %w", err)
	}

	return logs, nil
}