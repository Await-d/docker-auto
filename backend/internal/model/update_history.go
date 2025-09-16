package model

import (
	"time"

	"gorm.io/gorm"
)

// UpdateHistory represents container update history
type UpdateHistory struct {
	ID              int           `json:"id" gorm:"primaryKey;autoIncrement"`
	ContainerID     int           `json:"container_id" gorm:"not null;index:idx_update_history_container_id"`
	OldImage        string        `json:"old_image,omitempty" gorm:"size:255"`
	NewImage        string        `json:"new_image" gorm:"not null;size:255"`
	OldDigest       string        `json:"old_digest,omitempty" gorm:"size:71"`
	NewDigest       string        `json:"new_digest,omitempty" gorm:"size:71"`
	Status          UpdateStatus  `json:"status" gorm:"not null;index:idx_update_history_status"`
	ErrorMessage    string        `json:"error_message,omitempty" gorm:"type:text"`
	DurationSeconds int           `json:"duration_seconds" gorm:"default:0"`
	TriggeredBy     TriggerType   `json:"triggered_by" gorm:"not null;default:'auto';index:idx_update_history_triggered_by"`
	Strategy        UpdateStrategy `json:"strategy" gorm:"not null;default:'recreate'"`
	BackupCreated   bool          `json:"backup_created" gorm:"not null;default:false"`
	RollbackAvailable bool        `json:"rollback_available" gorm:"not null;default:false"`
	Logs            string        `json:"logs,omitempty" gorm:"type:text"`
	Metadata        string        `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	StartedAt       time.Time     `json:"started_at" gorm:"index:idx_update_history_started_at,sort:desc"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	CreatedBy       *int          `json:"created_by,omitempty"`

	// Relationships
	Container     Container `json:"-" gorm:"foreignKey:ContainerID"`
	CreatedByUser *User     `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`
}

// UpdateStatus defines update status
type UpdateStatus string

const (
	UpdateStatusPending   UpdateStatus = "pending"
	UpdateStatusRunning   UpdateStatus = "running"
	UpdateStatusSuccess   UpdateStatus = "success"
	UpdateStatusFailed    UpdateStatus = "failed"
	UpdateStatusRollback  UpdateStatus = "rollback"
	UpdateStatusCancelled UpdateStatus = "cancelled"
	UpdateStatusCompleted UpdateStatus = "completed" // Alias for success
)

// TriggerType defines how the update was triggered
type TriggerType string

const (
	TriggerTypeAuto     TriggerType = "auto"
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
)

// UpdateStrategy defines update strategies
type UpdateStrategy string

const (
	UpdateStrategyRecreate  UpdateStrategy = "recreate"
	UpdateStrategyRolling   UpdateStrategy = "rolling"
	UpdateStrategyBlueGreen UpdateStrategy = "blue_green"
	UpdateStrategyCanary    UpdateStrategy = "canary"
)

// UpdateHistoryFilter represents filters for querying update history
type UpdateHistoryFilter struct {
	ContainerID *int         `json:"container_id,omitempty"`
	Status      UpdateStatus `json:"status,omitempty"`
	TriggeredBy TriggerType  `json:"triggered_by,omitempty"`
	Strategy    UpdateStrategy `json:"strategy,omitempty"`
	CreatedBy   *int         `json:"created_by,omitempty"`
	StartedAfter *time.Time  `json:"started_after,omitempty"`
	StartedBefore *time.Time `json:"started_before,omitempty"`
	CompletedAfter *time.Time `json:"completed_after,omitempty"`
	CompletedBefore *time.Time `json:"completed_before,omitempty"`
	Limit       int          `json:"limit,omitempty"`
	Offset      int          `json:"offset,omitempty"`
	OrderBy     string       `json:"order_by,omitempty"`
}

// TableName returns the table name for UpdateHistory model
func (UpdateHistory) TableName() string {
	return "update_history"
}

// UpdateStats represents update statistics for a container
type UpdateStats struct {
	TotalUpdates    int64   `json:"total_updates"`
	SuccessfulUpdates int64 `json:"successful_updates"`
	FailedUpdates   int64   `json:"failed_updates"`
	PendingUpdates  int64   `json:"pending_updates"`
	SuccessRate     float64 `json:"success_rate"`
	LastUpdateAt    *time.Time `json:"last_update_at,omitempty"`
	LastSuccessTime *time.Time `json:"last_success_time,omitempty"`
	LastFailureTime *time.Time `json:"last_failure_time,omitempty"`
	AverageUpdateDuration int `json:"average_update_duration"`
}

// IsCompleted checks if update is completed (success or failed)
func (uh *UpdateHistory) IsCompleted() bool {
	return uh.Status == UpdateStatusSuccess || uh.Status == UpdateStatusFailed || uh.Status == UpdateStatusCancelled
}

// IsSuccessful checks if update was successful
func (uh *UpdateHistory) IsSuccessful() bool {
	return uh.Status == UpdateStatusSuccess
}

// GetDuration returns the duration of the update
func (uh *UpdateHistory) GetDuration() time.Duration {
	if uh.CompletedAt != nil {
		return uh.CompletedAt.Sub(uh.StartedAt)
	}
	if uh.IsCompleted() {
		return time.Duration(uh.DurationSeconds) * time.Second
	}
	return time.Since(uh.StartedAt)
}

// GetValidUpdateStatuses returns all valid update statuses
func GetValidUpdateStatuses() []UpdateStatus {
	return []UpdateStatus{
		UpdateStatusPending,
		UpdateStatusRunning,
		UpdateStatusSuccess,
		UpdateStatusFailed,
		UpdateStatusRollback,
		UpdateStatusCancelled,
	}
}

// GetValidTriggerTypes returns all valid trigger types
func GetValidTriggerTypes() []TriggerType {
	return []TriggerType{
		TriggerTypeAuto,
		TriggerTypeManual,
		TriggerTypeSchedule,
		TriggerTypeWebhook,
	}
}

// GetValidUpdateStrategies returns all valid update strategies
func GetValidUpdateStrategies() []UpdateStrategy {
	return []UpdateStrategy{
		UpdateStrategyRecreate,
		UpdateStrategyRolling,
		UpdateStrategyBlueGreen,
		UpdateStrategyCanary,
	}
}

// BeforeCreate hook for UpdateHistory model
func (uh *UpdateHistory) BeforeCreate(tx *gorm.DB) error {
	if uh.TriggeredBy == "" {
		uh.TriggeredBy = TriggerTypeManual
	}
	if uh.Strategy == "" {
		uh.Strategy = UpdateStrategyRecreate
	}
	if uh.Status == "" {
		uh.Status = UpdateStatusPending
	}
	if uh.StartedAt.IsZero() {
		uh.StartedAt = time.Now()
	}
	return nil
}

// BeforeUpdate hook for UpdateHistory model
func (uh *UpdateHistory) BeforeUpdate(tx *gorm.DB) error {
	// Calculate duration when status changes to completed
	if uh.IsCompleted() && uh.CompletedAt == nil {
		now := time.Now()
		uh.CompletedAt = &now
		uh.DurationSeconds = int(now.Sub(uh.StartedAt).Seconds())
	}
	return nil
}