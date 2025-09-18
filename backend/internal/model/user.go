package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID                 int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	Username           string         `json:"username" gorm:"uniqueIndex;not null;size:50;index:idx_users_username"`
	Email              string         `json:"email" gorm:"uniqueIndex;not null;size:100;index:idx_users_email"`
	PasswordHash       string         `json:"-" gorm:"column:password_hash;not null;size:255"` // Exclude from JSON
	Role               UserRole       `json:"role" gorm:"not null;default:'user';index:idx_users_role"`
	IsActive           bool           `json:"is_active" gorm:"not null;default:true;index:idx_users_is_active"`
	EmailNotifications bool           `json:"email_notifications" gorm:"not null;default:true"`
	AvatarURL          string         `json:"avatar_url,omitempty" gorm:"size:255"`
	LastLoginAt        *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// Relationships
	Containers           []Container               `json:"containers,omitempty" gorm:"foreignKey:CreatedBy"`
	UserSessions         []UserSession             `json:"-" gorm:"foreignKey:UserID"`
	ActivityLogs         []ActivityLog             `json:"-" gorm:"foreignKey:UserID"`
	UpdateHistories      []UpdateHistory           `json:"-" gorm:"foreignKey:CreatedBy"`
	ScheduledTasks       []ScheduledTask           `json:"-" gorm:"foreignKey:CreatedBy"`
	Notifications        []UserNotification        `json:"-" gorm:"foreignKey:UserID"`
	NotificationSettings *UserNotificationSettings `json:"notification_settings,omitempty" gorm:"foreignKey:UserID"`
}

// UserSession represents user refresh token sessions
type UserSession struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID       int64     `json:"user_id" gorm:"not null;index:idx_user_sessions_user_id"`
	RefreshToken string    `json:"-" gorm:"uniqueIndex:idx_user_sessions_refresh_token;not null;size:255"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"not null;index:idx_user_sessions_expires_at"`
	IPAddress    string    `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent    string    `json:"user_agent,omitempty" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// ActivityLog represents system activity logs
type ActivityLog struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       *int64    `json:"user_id,omitempty" gorm:"index:idx_activity_logs_user_id"`
	Action       string    `json:"action" gorm:"not null;size:100;index:idx_activity_logs_action"`
	ResourceType string    `json:"resource_type" gorm:"not null;size:50;index:idx_activity_logs_resource_type"`
	ResourceID   *int      `json:"resource_id,omitempty"`
	ResourceName string    `json:"resource_name,omitempty" gorm:"size:255"`
	Description  string    `json:"description,omitempty" gorm:"type:text"`
	IPAddress    string    `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent    string    `json:"user_agent,omitempty" gorm:"type:text"`
	Metadata     string    `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time `json:"created_at" gorm:"index:idx_activity_logs_created_at,sort:desc"`

	// Relationships
	User *User `json:"-" gorm:"foreignKey:UserID"`
}

// UserRole defines user roles
type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
	UserRoleViewer   UserRole = "viewer"
)

// UserFilter represents filters for querying users
type UserFilter struct {
	Username string   `json:"username,omitempty"`
	Email    string   `json:"email,omitempty"`
	Role     UserRole `json:"role,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	Offset   int      `json:"offset,omitempty"`
	OrderBy  string   `json:"order_by,omitempty"`
}

// ActivityLogFilter represents filters for querying activity logs
type ActivityLogFilter struct {
	UserID       *int64 `json:"user_id,omitempty"`
	Action       string `json:"action,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   *int   `json:"resource_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	OrderBy      string `json:"order_by,omitempty"`
}

// TableName returns the table name for User model
func (User) TableName() string {
	return "users"
}

// TableName returns the table name for UserSession model
func (UserSession) TableName() string {
	return "user_sessions"
}

// TableName returns the table name for ActivityLog model
func (ActivityLog) TableName() string {
	return "activity_logs"
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsUserActive checks if user is active
func (u *User) IsUserActive() bool {
	return u.IsActive
}

// GetValidRoles returns all valid user roles
func GetValidRoles() []UserRole {
	return []UserRole{UserRoleAdmin, UserRoleOperator, UserRoleViewer}
}

// IsExpired checks if user session is expired
func (us *UserSession) IsExpired() bool {
	return time.Now().After(us.ExpiresAt)
}

// BeforeCreate hook for User model
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = UserRoleViewer
	}
	return nil
}