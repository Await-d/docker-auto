package model

import (
	"time"
	"gorm.io/gorm"
)

// AuditLog represents security audit events in the system
type AuditLog struct {
	ID          int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	EventID     string     `json:"event_id" gorm:"uniqueIndex;not null;size:64"`
	EventType   string     `json:"event_type" gorm:"not null;size:50;index:idx_audit_logs_event_type"`
	Severity    string     `json:"severity" gorm:"not null;size:20;index:idx_audit_logs_severity"`
	Source      string     `json:"source" gorm:"not null;size:100;index:idx_audit_logs_source"`
	UserID      *int64     `json:"user_id,omitempty" gorm:"index:idx_audit_logs_user_id"`
	ContainerID *string    `json:"container_id,omitempty" gorm:"size:64;index:idx_audit_logs_container_id"`
	Action      string     `json:"action" gorm:"not null;size:100"`
	Resource    string     `json:"resource" gorm:"size:255"`
	Details     string     `json:"details" gorm:"type:jsonb"`
	Result      string     `json:"result" gorm:"size:20"`
	RemoteIP    string     `json:"remote_ip" gorm:"size:45"`
	UserAgent   string     `json:"user_agent" gorm:"size:500"`
	SessionID   string     `json:"session_id" gorm:"size:64"`
	Metadata    string     `json:"metadata" gorm:"type:jsonb"`
	Timestamp   time.Time  `json:"timestamp" gorm:"not null;index:idx_audit_logs_timestamp"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`

	// Relationships
	User        *User      `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

// TableName returns the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate hook for GORM
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.Timestamp.IsZero() {
		al.Timestamp = time.Now()
	}
	return nil
}

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventTypeLogin           AuditEventType = "login"
	AuditEventTypeLogout          AuditEventType = "logout"
	AuditEventTypeContainerCreate AuditEventType = "container_create"
	AuditEventTypeContainerUpdate AuditEventType = "container_update"
	AuditEventTypeContainerDelete AuditEventType = "container_delete"
	AuditEventTypeContainerStart  AuditEventType = "container_start"
	AuditEventTypeContainerStop   AuditEventType = "container_stop"
	AuditEventTypeContainerRestart AuditEventType = "container_restart"
	AuditEventTypeImagePull       AuditEventType = "image_pull"
	AuditEventTypeImageScan       AuditEventType = "image_scan"
	AuditEventTypeSecurityScan    AuditEventType = "security_scan"
	AuditEventTypeComplianceCheck AuditEventType = "compliance_check"
	AuditEventTypeRiskAssessment  AuditEventType = "risk_assessment"
	AuditEventTypeConfigChange    AuditEventType = "config_change"
	AuditEventTypeSystemAccess    AuditEventType = "system_access"
	AuditEventTypeDataExport      AuditEventType = "data_export"
	AuditEventTypePrivilegeEscalation AuditEventType = "privilege_escalation"
)

// SecuritySeverity represents the severity level of security events
type SecuritySeverity string

const (
	SecuritySeverityLow      SecuritySeverity = "low"
	SecuritySeverityMedium   SecuritySeverity = "medium"
	SecuritySeverityHigh     SecuritySeverity = "high"
	SecuritySeverityCritical SecuritySeverity = "critical"
)

// AuditResult represents the result of an audited action
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
	AuditResultBlocked AuditResult = "blocked"
	AuditResultWarning AuditResult = "warning"
)

// SecurityAlert represents security alerts generated from audit events
type SecurityAlert struct {
	ID          int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	AlertID     string            `json:"alert_id" gorm:"uniqueIndex;not null;size:64"`
	AlertType   string            `json:"alert_type" gorm:"not null;size:50;index:idx_security_alerts_type"`
	Severity    SecuritySeverity  `json:"severity" gorm:"not null;index:idx_security_alerts_severity"`
	Title       string            `json:"title" gorm:"not null;size:255"`
	Description string            `json:"description" gorm:"type:text"`
	Source      string            `json:"source" gorm:"not null;size:100"`
	ContainerID *string           `json:"container_id,omitempty" gorm:"size:64;index:idx_security_alerts_container_id"`
	UserID      *int64            `json:"user_id,omitempty" gorm:"index:idx_security_alerts_user_id"`
	Status      AlertStatus       `json:"status" gorm:"not null;default:'open';index:idx_security_alerts_status"`
	Metadata    string            `json:"metadata" gorm:"type:jsonb"`
	CreatedAt   time.Time         `json:"created_at" gorm:"autoCreateTime;index:idx_security_alerts_created_at"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`

	// Relationships
	User        *User             `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

// TableName returns the table name for SecurityAlert
func (SecurityAlert) TableName() string {
	return "security_alerts"
}

// AlertStatus represents the status of a security alert
type AlertStatus string

const (
	AlertStatusOpen       AlertStatus = "open"
	AlertStatusInProgress AlertStatus = "in_progress"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusIgnored    AlertStatus = "ignored"
	AlertStatusFalsePositive AlertStatus = "false_positive"
)

// SecurityAlertType represents the type of security alert
type SecurityAlertType string

const (
	SecurityAlertTypeVulnerability     SecurityAlertType = "vulnerability"
	SecurityAlertTypeCompliance        SecurityAlertType = "compliance"
	SecurityAlertTypeRiskAssessment    SecurityAlertType = "risk_assessment"
	SecurityAlertTypeAnomaly           SecurityAlertType = "anomaly"
	SecurityAlertTypePrivilegeEscalation SecurityAlertType = "privilege_escalation"
	SecurityAlertTypeUnauthorizedAccess SecurityAlertType = "unauthorized_access"
	SecurityAlertTypeSuspiciousActivity SecurityAlertType = "suspicious_activity"
	SecurityAlertTypeConfigViolation   SecurityAlertType = "config_violation"
)