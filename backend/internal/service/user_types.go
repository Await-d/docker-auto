package service

import (
	"time"

	"docker-auto/internal/model"
)

// Authentication related request types
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role,omitempty"`
}

// Authentication response types
type LoginResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int64         `json:"expires_in"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// User management request types
type UpdateProfileRequest struct {
	Username  *string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin operator viewer"`
}

type UpdateUserRequest struct {
	Username  *string `json:"username,omitempty"`
	Email     *string `json:"email,omitempty"`
	Role      *string `json:"role,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// Response types
type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Filter types
type UserFilter struct {
	Search    string `json:"search,omitempty"`
	Role      string `json:"role,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ActivityFilter struct {
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Action    string     `json:"action,omitempty"`
	Page      int        `json:"page,omitempty"`
	Limit     int        `json:"limit,omitempty"`
}

// Permission definitions
const (
	// Container related permissions
	PermissionContainerRead   = "container:read"
	PermissionContainerCreate = "container:create"
	PermissionContainerUpdate = "container:update"
	PermissionContainerDelete = "container:delete"
	PermissionContainerStart  = "container:start"
	PermissionContainerStop   = "container:stop"

	// Image related permissions
	PermissionImageRead   = "image:read"
	PermissionImageCheck  = "image:check"
	PermissionImageUpdate = "image:update"

	// Update related permissions
	PermissionUpdateRead     = "update:read"
	PermissionUpdateCreate   = "update:create"
	PermissionUpdateRollback = "update:rollback"

	// System related permissions
	PermissionSystemRead   = "system:read"
	PermissionSystemConfig = "system:config"
	PermissionSystemLogs   = "system:logs"

	// User related permissions
	PermissionUserRead   = "user:read"
	PermissionUserCreate = "user:create"
	PermissionUserUpdate = "user:update"
	PermissionUserDelete = "user:delete"
)

// Role permissions mapping
var RolePermissions = map[string][]string{
	"admin": {
		// Administrator has all permissions
		PermissionContainerRead, PermissionContainerCreate, PermissionContainerUpdate, PermissionContainerDelete,
		PermissionContainerStart, PermissionContainerStop,
		PermissionImageRead, PermissionImageCheck, PermissionImageUpdate,
		PermissionUpdateRead, PermissionUpdateCreate, PermissionUpdateRollback,
		PermissionSystemRead, PermissionSystemConfig, PermissionSystemLogs,
		PermissionUserRead, PermissionUserCreate, PermissionUserUpdate, PermissionUserDelete,
	},
	"operator": {
		// Operator permissions
		PermissionContainerRead, PermissionContainerCreate, PermissionContainerUpdate,
		PermissionContainerStart, PermissionContainerStop,
		PermissionImageRead, PermissionImageCheck, PermissionImageUpdate,
		PermissionUpdateRead, PermissionUpdateCreate, PermissionUpdateRollback,
		PermissionSystemRead, PermissionSystemLogs,
	},
	"viewer": {
		// Viewer permissions (read-only)
		PermissionContainerRead,
		PermissionImageRead,
		PermissionUpdateRead,
		PermissionSystemRead,
	},
}

// SessionInfo represents user session information
type SessionInfo struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers      int64 `json:"total_users"`
	ActiveUsers     int64 `json:"active_users"`
	InactiveUsers   int64 `json:"inactive_users"`
	AdminUsers      int64 `json:"admin_users"`
	OperatorUsers   int64 `json:"operator_users"`
	ViewerUsers     int64 `json:"viewer_users"`
	OnlineUsers     int64 `json:"online_users"`
	RecentLogins    int64 `json:"recent_logins"`
	FailedLogins    int64 `json:"failed_logins"`
	ActiveSessions  int64 `json:"active_sessions"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}
	return ve.Errors[0].Message
}

// PasswordPolicy represents password policy configuration
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSpecial   bool `json:"require_special"`
	MaxAge           int  `json:"max_age_days"`
	PreventReuse     int  `json:"prevent_reuse_count"`
}

// DefaultPasswordPolicy returns the default password policy
func DefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:        6,
		RequireUppercase: false,
		RequireLowercase: false,
		RequireNumbers:   false,
		RequireSpecial:   false,
		MaxAge:           90,
		PreventReuse:     3,
	}
}

// SecuritySettings represents security-related settings
type SecuritySettings struct {
	PasswordPolicy           *PasswordPolicy `json:"password_policy"`
	SessionTimeoutMinutes    int             `json:"session_timeout_minutes"`
	MaxFailedAttempts        int             `json:"max_failed_attempts"`
	LockoutDurationMinutes   int             `json:"lockout_duration_minutes"`
	RequireMFA               bool            `json:"require_mfa"`
	AllowConcurrentSessions  bool            `json:"allow_concurrent_sessions"`
	MaxConcurrentSessions    int             `json:"max_concurrent_sessions"`
	ForcePasswordChange      bool            `json:"force_password_change"`
	AllowPasswordRecovery    bool            `json:"allow_password_recovery"`
	RequireEmailVerification bool            `json:"require_email_verification"`
}

// DefaultSecuritySettings returns default security settings
func DefaultSecuritySettings() *SecuritySettings {
	return &SecuritySettings{
		PasswordPolicy:           DefaultPasswordPolicy(),
		SessionTimeoutMinutes:    1440, // 24 hours
		MaxFailedAttempts:        5,
		LockoutDurationMinutes:   30,
		RequireMFA:               false,
		AllowConcurrentSessions:  true,
		MaxConcurrentSessions:    5,
		ForcePasswordChange:      false,
		AllowPasswordRecovery:    true,
		RequireEmailVerification: false,
	}
}