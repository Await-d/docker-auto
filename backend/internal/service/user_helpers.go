package service

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"docker-auto/internal/model"
	"docker-auto/pkg/utils"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// Private helper methods for UserService

// Authentication helpers

// validateLogin validates user credentials
func (s *UserService) validateLogin(username, password string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}

	// Try to find user by username first, then by email
	var user *model.User
	var err error

	// Check if username looks like an email
	if strings.Contains(username, "@") {
		user, err = s.userRepo.GetByEmail(nil, username)
	} else {
		user, err = s.userRepo.GetByUsername(nil, username)
	}

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return user, nil
}

// generateTokens generates access and refresh tokens for user
func (s *UserService) generateTokens(user *model.User) (*TokenResponse, error) {
	tokenPair, err := s.jwtManager.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// createUserSession creates a new user session with refresh token
func (s *UserService) createUserSession(userID int64, refreshToken string) error {
	session := &model.UserSession{
		ID:           uuid.New().String(),
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(s.config.JWT.RefreshDays) * 24 * time.Hour),
		CreatedAt:    time.Now().UTC(),
	}

	return s.sessionRepo.Create(nil, session)
}

// invalidateUserSessions removes all sessions for a user
func (s *UserService) invalidateUserSessions(userID int64) error {
	return s.sessionRepo.DeleteUserSessions(nil, userID)
}

// Password management

// hashPassword hashes a password using bcrypt
func (s *UserService) hashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// validatePassword validates password against policy
func (s *UserService) validatePassword(password string) error {
	policy := DefaultPasswordPolicy()

	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}

	if policy.RequireUppercase && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}

	if policy.RequireLowercase && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}

	if policy.RequireNumbers && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one number")
	}

	if policy.RequireSpecial && !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// Permission management helpers

// hasPermission checks if a role has a specific permission
func (s *UserService) hasPermission(userRole string, permission string) bool {
	permissions, exists := RolePermissions[userRole]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// hasAnyPermission checks if a role has any of the specified permissions
func (s *UserService) hasAnyPermission(userRole string, permissions []string) bool {
	for _, permission := range permissions {
		if s.hasPermission(userRole, permission) {
			return true
		}
	}
	return false
}

// getUserPermissions returns all permissions for a user role
func (s *UserService) getUserPermissions(userRole string) []string {
	permissions, exists := RolePermissions[userRole]
	if !exists {
		return []string{}
	}

	// Return a copy to prevent modification
	result := make([]string, len(permissions))
	copy(result, permissions)
	return result
}

// hasPermissionInList checks if a permission exists in a list of permissions
func (s *UserService) hasPermissionInList(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Data conversion helpers

// userToResponse converts a user model to response format
func (s *UserService) userToResponse(user *model.User) *UserResponse {
	if user == nil {
		return nil
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// sessionToResponse converts a session model to response format
func (s *UserService) sessionToResponse(session *model.UserSession) *SessionInfo {
	if session == nil {
		return nil
	}

	return &SessionInfo{
		ID:        session.ID,
		UserID:    session.UserID,
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	}
}

// Validation helpers

// validateLoginRequest validates login request data
func (s *UserService) validateLoginRequest(req *LoginRequest) error {
	var errors []ValidationError

	if req.Username == "" {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username is required",
		})
	}

	if req.Password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password is required",
		})
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateRegisterRequest validates registration request data
func (s *UserService) validateRegisterRequest(req *RegisterRequest) error {
	var errors []ValidationError

	// Validate username
	if req.Username == "" {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username is required",
		})
	} else if len(req.Username) < 3 || len(req.Username) > 50 {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username must be between 3 and 50 characters",
			Value:   req.Username,
		})
	} else if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(req.Username) {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username can only contain letters, numbers, underscores, and hyphens",
			Value:   req.Username,
		})
	}

	// Validate email
	if req.Email == "" {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Email is required",
		})
	} else if _, err := mail.ParseAddress(req.Email); err != nil {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Invalid email format",
			Value:   req.Email,
		})
	}

	// Validate password
	if req.Password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password is required",
		})
	} else if err := s.validatePassword(req.Password); err != nil {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: err.Error(),
		})
	}

	// Validate role if provided
	if req.Role != "" {
		validRoles := model.GetValidRoles()
		roleValid := false
		for _, role := range validRoles {
			if string(role) == req.Role {
				roleValid = true
				break
			}
		}
		if !roleValid {
			errors = append(errors, ValidationError{
				Field:   "role",
				Message: "Invalid role",
				Value:   req.Role,
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateCreateUserRequest validates create user request data
func (s *UserService) validateCreateUserRequest(req *CreateUserRequest) error {
	var errors []ValidationError

	// Validate username
	if req.Username == "" {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username is required",
		})
	} else if len(req.Username) < 3 || len(req.Username) > 50 {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Username must be between 3 and 50 characters",
			Value:   req.Username,
		})
	}

	// Validate email
	if req.Email == "" {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Email is required",
		})
	} else if _, err := mail.ParseAddress(req.Email); err != nil {
		errors = append(errors, ValidationError{
			Field:   "email",
			Message: "Invalid email format",
			Value:   req.Email,
		})
	}

	// Validate password
	if req.Password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Password is required",
		})
	} else if err := s.validatePassword(req.Password); err != nil {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: err.Error(),
		})
	}

	// Validate role
	if req.Role == "" {
		errors = append(errors, ValidationError{
			Field:   "role",
			Message: "Role is required",
		})
	} else {
		validRoles := model.GetValidRoles()
		roleValid := false
		for _, role := range validRoles {
			if string(role) == req.Role {
				roleValid = true
				break
			}
		}
		if !roleValid {
			errors = append(errors, ValidationError{
				Field:   "role",
				Message: "Invalid role",
				Value:   req.Role,
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// validateChangePasswordRequest validates change password request
func (s *UserService) validateChangePasswordRequest(req *ChangePasswordRequest) error {
	var errors []ValidationError

	if req.OldPassword == "" {
		errors = append(errors, ValidationError{
			Field:   "old_password",
			Message: "Old password is required",
		})
	}

	if req.NewPassword == "" {
		errors = append(errors, ValidationError{
			Field:   "new_password",
			Message: "New password is required",
		})
	} else if err := s.validatePassword(req.NewPassword); err != nil {
		errors = append(errors, ValidationError{
			Field:   "new_password",
			Message: err.Error(),
		})
	}

	if req.OldPassword == req.NewPassword {
		errors = append(errors, ValidationError{
			Field:   "new_password",
			Message: "New password must be different from old password",
		})
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// sanitizeUserInput sanitizes user input to prevent injection attacks
func (s *UserService) sanitizeUserInput(input string) string {
	// Remove potential dangerous characters
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "&", "&amp;")

	return input
}

// Activity logging helpers

// logUserActivity logs user activity with metadata
func (s *UserService) logUserActivity(userID int64, action, description string, metadata map[string]interface{}) error {
	// Convert metadata to JSON
	var metadataJSON string
	if metadata != nil {
		if bytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(bytes)
		}
	}

	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	activity := &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "user",
		Description:  description,
		Metadata:     metadataJSON,
		CreatedAt:    time.Now().UTC(),
	}

	if userID > 0 {
		activity.UserID = &userID
	}

	if err := s.activityRepo.Create(nil, activity); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"action":  action,
		}).Warn("Failed to log user activity")
		return err
	}

	return nil
}

// Cache management helpers

// cacheUser caches user information
func (s *UserService) cacheUser(user *model.User) {
	if s.cache == nil || user == nil {
		return
	}

	cacheKey := fmt.Sprintf("user:%d", user.ID)
	if err := s.cache.Set(cacheKey, user, time.Hour); err != nil {
		logrus.WithError(err).WithField("user_id", user.ID).Debug("Failed to cache user")
	}
}

// getCachedUser retrieves user from cache
func (s *UserService) getCachedUser(userID int64) *model.User {
	if s.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("user:%d", userID)
	if cached, exists := s.cache.Get(cacheKey); exists {
		if user, ok := cached.(*model.User); ok {
			return user
		}
	}

	return nil
}

// invalidateUserCache removes user from cache
func (s *UserService) invalidateUserCache(userID int64) {
	if s.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf("user:%d", userID)
	s.cache.Delete(cacheKey)

	// Also invalidate permissions cache
	permissionKey := fmt.Sprintf("user_permissions:%d", userID)
	s.cache.Delete(permissionKey)
}

// cachePermissions caches user permissions
func (s *UserService) cachePermissions(userID int64, permissions []string) {
	if s.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf("user_permissions:%d", userID)
	if err := s.cache.Set(cacheKey, permissions, time.Hour); err != nil {
		logrus.WithError(err).WithField("user_id", userID).Debug("Failed to cache user permissions")
	}
}

// getCachedPermissions retrieves user permissions from cache
func (s *UserService) getCachedPermissions(userID int64) []string {
	if s.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("user_permissions:%d", userID)
	if cached, exists := s.cache.Get(cacheKey); exists {
		if permissions, ok := cached.([]string); ok {
			return permissions
		}
	}

	return nil
}

// Utility helpers

// generateSecureRandomString generates a cryptographically secure random string
func (s *UserService) generateSecureRandomString(length int) string {
	return utils.GenerateRandomString(length)
}

// isValidEmail validates email format
func (s *UserService) isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// isValidUsername validates username format
func (s *UserService) isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(username)
}

// isExpiredToken checks if a token is expired
func (s *UserService) isExpiredToken(token string) bool {
	return s.jwtManager.IsTokenExpired(token)
}

// extractUserIDFromToken extracts user ID from JWT token
func (s *UserService) extractUserIDFromToken(token string) (int64, error) {
	return utils.GetUserIDFromToken(token)
}