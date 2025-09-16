package repository

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/model"

	"gorm.io/gorm"
)

// imageVersionRepository implements ImageVersionRepository interface
type imageVersionRepository struct {
	db *gorm.DB
}

// NewImageVersionRepository creates a new image version repository
func NewImageVersionRepository(db *gorm.DB) ImageVersionRepository {
	return &imageVersionRepository{db: db}
}

// Create creates a new image version
func (r *imageVersionRepository) Create(ctx context.Context, version *model.ImageVersion) error {
	if version == nil {
		return fmt.Errorf("image version cannot be nil")
	}

	// Validate required fields
	if version.ImageName == "" {
		return fmt.Errorf("image name is required")
	}
	if version.Tag == "" {
		version.Tag = "latest"
	}
	if version.Digest == "" {
		return fmt.Errorf("image digest is required")
	}

	if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
		return fmt.Errorf("failed to create image version: %w", err)
	}

	return nil
}

// GetByID retrieves an image version by ID
func (r *imageVersionRepository) GetByID(ctx context.Context, id int64) (*model.ImageVersion, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid image version ID: %d", id)
	}

	var version model.ImageVersion
	err := r.db.WithContext(ctx).First(&version, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image version with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get image version by ID: %w", err)
	}

	return &version, nil
}

// Update updates an existing image version
func (r *imageVersionRepository) Update(ctx context.Context, version *model.ImageVersion) error {
	if version == nil {
		return fmt.Errorf("image version cannot be nil")
	}
	if version.ID <= 0 {
		return fmt.Errorf("invalid image version ID: %d", version.ID)
	}

	// Check if version exists
	var existingVersion model.ImageVersion
	if err := r.db.WithContext(ctx).First(&existingVersion, version.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("image version with ID %d not found", version.ID)
		}
		return fmt.Errorf("failed to check image version existence: %w", err)
	}

	// Update checked time
	version.CheckedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(version).Error; err != nil {
		return fmt.Errorf("failed to update image version: %w", err)
	}

	return nil
}

// Delete deletes an image version by ID
func (r *imageVersionRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid image version ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.ImageVersion{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete image version: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("image version with ID %d not found", id)
	}

	return nil
}

// List retrieves image versions with filtering and pagination
func (r *imageVersionRepository) List(ctx context.Context, filter *model.ImageVersionFilter) ([]*model.ImageVersion, int64, error) {
	var versions []*model.ImageVersion
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ImageVersion{})

	// Apply filters
	if filter != nil {
		if filter.ImageName != "" {
			query = query.Where("image_name ILIKE ?", "%"+filter.ImageName+"%")
		}
		if filter.Tag != "" {
			query = query.Where("tag = ?", filter.Tag)
		}
		if filter.RegistryURL != "" {
			query = query.Where("registry_url = ?", filter.RegistryURL)
		}
		if filter.Architecture != "" {
			query = query.Where("architecture = ?", filter.Architecture)
		}
		if filter.OS != "" {
			query = query.Where("os = ?", filter.OS)
		}
		if filter.IsLatest != nil {
			query = query.Where("is_latest = ?", *filter.IsLatest)
		}
		if filter.CheckedAfter != nil {
			query = query.Where("checked_at >= ?", *filter.CheckedAfter)
		}
		if filter.CheckedBefore != nil {
			query = query.Where("checked_at <= ?", *filter.CheckedBefore)
		}
		if filter.PublishedAfter != nil {
			query = query.Where("published_at >= ?", *filter.PublishedAfter)
		}
		if filter.PublishedBefore != nil {
			query = query.Where("published_at <= ?", *filter.PublishedBefore)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count image versions: %w", err)
	}

	// Apply ordering
	orderBy := "checked_at DESC"
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

	if err := query.Find(&versions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list image versions: %w", err)
	}

	return versions, total, nil
}

// GetByImageName retrieves all versions for a specific image
func (r *imageVersionRepository) GetByImageName(ctx context.Context, imageName string) ([]*model.ImageVersion, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	var versions []*model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ?", imageName).
		Order("checked_at DESC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get image versions by name: %w", err)
	}

	return versions, nil
}

// GetByImageAndTag retrieves a specific image version by name and tag
func (r *imageVersionRepository) GetByImageAndTag(ctx context.Context, imageName, tag string) (*model.ImageVersion, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}
	if tag == "" {
		tag = "latest"
	}

	var version model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ? AND tag = ?", imageName, tag).
		First(&version).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image version for %s:%s not found", imageName, tag)
		}
		return nil, fmt.Errorf("failed to get image version by name and tag: %w", err)
	}

	return &version, nil
}

// GetLatest retrieves the latest version for a specific image
func (r *imageVersionRepository) GetLatest(ctx context.Context, imageName string) (*model.ImageVersion, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	var version model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ? AND is_latest = ?", imageName, true).
		First(&version).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// If no explicit latest version, get the most recently checked one
			err = r.db.WithContext(ctx).
				Where("image_name = ?", imageName).
				Order("checked_at DESC").
				First(&version).Error

			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil, fmt.Errorf("no versions found for image %s", imageName)
				}
				return nil, fmt.Errorf("failed to get latest image version: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get latest image version: %w", err)
		}
	}

	return &version, nil
}

// UpsertVersion creates or updates an image version
func (r *imageVersionRepository) UpsertVersion(ctx context.Context, version *model.ImageVersion) error {
	if version == nil {
		return fmt.Errorf("image version cannot be nil")
	}

	// Validate required fields
	if version.ImageName == "" {
		return fmt.Errorf("image name is required")
	}
	if version.Tag == "" {
		version.Tag = "latest"
	}
	if version.Digest == "" {
		return fmt.Errorf("image digest is required")
	}

	// Try to find existing version
	var existingVersion model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ? AND tag = ? AND registry_url = ?",
			version.ImageName, version.Tag, version.RegistryURL).
		First(&existingVersion).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing version: %w", err)
	}

	version.CheckedAt = time.Now().UTC()

	if err == gorm.ErrRecordNotFound {
		// Create new version
		if err := r.db.WithContext(ctx).Create(version).Error; err != nil {
			return fmt.Errorf("failed to create image version: %w", err)
		}
	} else {
		// Update existing version
		version.ID = existingVersion.ID
		if err := r.db.WithContext(ctx).Save(version).Error; err != nil {
			return fmt.Errorf("failed to update image version: %w", err)
		}
	}

	return nil
}

// GetVersionHistory retrieves version history for an image with limit
func (r *imageVersionRepository) GetVersionHistory(ctx context.Context, imageName string, limit int) ([]*model.ImageVersion, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	var versions []*model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ?", imageName).
		Order("checked_at DESC").
		Limit(limit).
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}

	return versions, nil
}

// DeleteOldVersions deletes old versions for an image, keeping the specified count of recent ones
func (r *imageVersionRepository) DeleteOldVersions(ctx context.Context, imageName string, keepCount int) error {
	if imageName == "" {
		return fmt.Errorf("image name cannot be empty")
	}
	if keepCount <= 0 {
		keepCount = 5
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get IDs of versions to keep (most recent ones and latest marked versions)
		var keepIDs []int

		// Keep the most recent versions
		var recentVersions []model.ImageVersion
		if err := tx.Model(&model.ImageVersion{}).
			Select("id").
			Where("image_name = ?", imageName).
			Order("checked_at DESC").
			Limit(keepCount).
			Find(&recentVersions).Error; err != nil {
			return fmt.Errorf("failed to get recent versions: %w", err)
		}

		for _, v := range recentVersions {
			keepIDs = append(keepIDs, v.ID)
		}

		// Also keep any version marked as latest
		var latestVersions []model.ImageVersion
		if err := tx.Model(&model.ImageVersion{}).
			Select("id").
			Where("image_name = ? AND is_latest = ?", imageName, true).
			Find(&latestVersions).Error; err != nil {
			return fmt.Errorf("failed to get latest versions: %w", err)
		}

		for _, v := range latestVersions {
			keepIDs = append(keepIDs, v.ID)
		}

		// Delete versions not in the keep list
		if len(keepIDs) > 0 {
			result := tx.Where("image_name = ? AND id NOT IN ?", imageName, keepIDs).
				Delete(&model.ImageVersion{})
			if result.Error != nil {
				return fmt.Errorf("failed to delete old versions: %w", result.Error)
			}
		}

		return nil
	})
}

// RefreshImageCache updates the cache timestamp for an image
func (r *imageVersionRepository) RefreshImageCache(ctx context.Context, imageName string) error {
	if imageName == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	result := r.db.WithContext(ctx).
		Model(&model.ImageVersion{}).
		Where("image_name = ?", imageName).
		Update("checked_at", time.Now().UTC())

	if result.Error != nil {
		return fmt.Errorf("failed to refresh image cache: %w", result.Error)
	}

	return nil
}

// GetCachedVersions retrieves cached versions for an image that are not stale
func (r *imageVersionRepository) GetCachedVersions(ctx context.Context, imageName string) ([]*model.ImageVersion, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	// Consider versions stale if older than 6 hours
	staleThreshold := time.Now().UTC().Add(-6 * time.Hour)

	var versions []*model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name = ? AND checked_at > ?", imageName, staleThreshold).
		Order("checked_at DESC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get cached versions: %w", err)
	}

	return versions, nil
}

// SearchByName searches image versions by name pattern
func (r *imageVersionRepository) SearchByName(ctx context.Context, namePattern string) ([]*model.ImageVersion, error) {
	if namePattern == "" {
		return nil, fmt.Errorf("name pattern cannot be empty")
	}

	var versions []*model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("image_name ILIKE ?", "%"+namePattern+"%").
		Order("image_name, checked_at DESC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search image versions by name: %w", err)
	}

	return versions, nil
}

// GetOutdatedImages retrieves images that haven't been checked recently
func (r *imageVersionRepository) GetOutdatedImages(ctx context.Context) ([]*model.ImageVersion, error) {
	// Consider images outdated if not checked in the last 24 hours
	outdatedThreshold := time.Now().UTC().Add(-24 * time.Hour)

	var versions []*model.ImageVersion
	err := r.db.WithContext(ctx).
		Where("checked_at < ?", outdatedThreshold).
		Order("checked_at ASC").
		Find(&versions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get outdated images: %w", err)
	}

	return versions, nil
}

// CountOlderThan counts image versions older than the specified date
func (r *imageVersionRepository) CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.ImageVersion{}).
		Where("checked_at < ?", cutoffDate).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count old image versions: %w", err)
	}

	return count, nil
}

// DeleteOlderThan deletes image versions older than the specified date
func (r *imageVersionRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("checked_at < ?", cutoffDate).
		Delete(&model.ImageVersion{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old image versions: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// updateHistoryRepository implements UpdateHistoryRepository interface
type updateHistoryRepository struct {
	db *gorm.DB
}

// NewUpdateHistoryRepository creates a new update history repository
func NewUpdateHistoryRepository(db *gorm.DB) UpdateHistoryRepository {
	return &updateHistoryRepository{db: db}
}

// Create creates a new update history entry
func (r *updateHistoryRepository) Create(ctx context.Context, history *model.UpdateHistory) error {
	if history == nil {
		return fmt.Errorf("update history cannot be nil")
	}

	// Validate required fields
	if history.ContainerID <= 0 {
		return fmt.Errorf("container ID is required")
	}

	if err := r.db.WithContext(ctx).Create(history).Error; err != nil {
		return fmt.Errorf("failed to create update history: %w", err)
	}

	return nil
}

// GetByID retrieves an update history entry by ID
func (r *updateHistoryRepository) GetByID(ctx context.Context, id int64) (*model.UpdateHistory, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid update history ID: %d", id)
	}

	var history model.UpdateHistory
	err := r.db.WithContext(ctx).
		Preload("Container").
		Preload("CreatedByUser").
		First(&history, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("update history with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get update history by ID: %w", err)
	}

	return &history, nil
}

// Delete deletes an update history entry by ID
func (r *updateHistoryRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid update history ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.UpdateHistory{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete update history: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("update history with ID %d not found", id)
	}

	return nil
}

// List retrieves update history entries with filtering and pagination
func (r *updateHistoryRepository) List(ctx context.Context, filter *model.UpdateHistoryFilter) ([]*model.UpdateHistory, int64, error) {
	var histories []*model.UpdateHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&model.UpdateHistory{})

	// Apply filters
	if filter != nil {
		if filter.ContainerID != nil {
			query = query.Where("container_id = ?", *filter.ContainerID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.CreatedBy != nil {
			query = query.Where("created_by = ?", *filter.CreatedBy)
		}
		if filter.StartedAfter != nil {
			query = query.Where("started_at >= ?", *filter.StartedAfter)
		}
		if filter.StartedBefore != nil {
			query = query.Where("started_at <= ?", *filter.StartedBefore)
		}
		if filter.CompletedAfter != nil {
			query = query.Where("completed_at >= ?", *filter.CompletedAfter)
		}
		if filter.CompletedBefore != nil {
			query = query.Where("completed_at <= ?", *filter.CompletedBefore)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count update histories: %w", err)
	}

	// Apply ordering
	orderBy := "started_at DESC"
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
	query = query.Preload("Container").Preload("CreatedByUser")

	if err := query.Find(&histories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list update histories: %w", err)
	}

	return histories, total, nil
}

// GetByContainerID retrieves update history for a specific container
func (r *updateHistoryRepository) GetByContainerID(ctx context.Context, containerID int64, limit, offset int) ([]*model.UpdateHistory, int64, error) {
	if containerID <= 0 {
		return nil, 0, fmt.Errorf("invalid container ID: %d", containerID)
	}

	var histories []*model.UpdateHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&model.UpdateHistory{}).
		Where("container_id = ?", containerID)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count update histories for container: %w", err)
	}

	// Apply pagination and ordering
	query = query.Order("started_at DESC").
		Preload("Container").
		Preload("CreatedByUser")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&histories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get update histories by container ID: %w", err)
	}

	return histories, total, nil
}

// GetByStatus retrieves update history entries by status
func (r *updateHistoryRepository) GetByStatus(ctx context.Context, status model.UpdateStatus) ([]*model.UpdateHistory, error) {
	var histories []*model.UpdateHistory
	err := r.db.WithContext(ctx).
		Preload("Container").
		Preload("CreatedByUser").
		Where("status = ?", status).
		Order("started_at DESC").
		Find(&histories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get update histories by status: %w", err)
	}

	return histories, nil
}

// GetRecent retrieves recent update history entries
func (r *updateHistoryRepository) GetRecent(ctx context.Context, limit int) ([]*model.UpdateHistory, error) {
	if limit <= 0 {
		limit = 10
	}

	var histories []*model.UpdateHistory
	err := r.db.WithContext(ctx).
		Preload("Container").
		Preload("CreatedByUser").
		Order("started_at DESC").
		Limit(limit).
		Find(&histories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent update histories: %w", err)
	}

	return histories, nil
}

// GetUpdateStats retrieves update statistics for a container
func (r *updateHistoryRepository) GetUpdateStats(ctx context.Context, containerID int64) (*model.UpdateStats, error) {
	if containerID <= 0 {
		return nil, fmt.Errorf("invalid container ID: %d", containerID)
	}

	var stats model.UpdateStats

	// Get total count
	if err := r.db.WithContext(ctx).
		Model(&model.UpdateHistory{}).
		Where("container_id = ?", containerID).
		Count(&stats.TotalUpdates).Error; err != nil {
		return nil, fmt.Errorf("failed to get total updates count: %w", err)
	}

	// Get successful count
	if err := r.db.WithContext(ctx).
		Model(&model.UpdateHistory{}).
		Where("container_id = ? AND status = ?", containerID, model.UpdateStatusCompleted).
		Count(&stats.SuccessfulUpdates).Error; err != nil {
		return nil, fmt.Errorf("failed to get successful updates count: %w", err)
	}

	// Get failed count
	if err := r.db.WithContext(ctx).
		Model(&model.UpdateHistory{}).
		Where("container_id = ? AND status = ?", containerID, model.UpdateStatusFailed).
		Count(&stats.FailedUpdates).Error; err != nil {
		return nil, fmt.Errorf("failed to get failed updates count: %w", err)
	}

	// Get last update
	var lastUpdate model.UpdateHistory
	if err := r.db.WithContext(ctx).
		Where("container_id = ?", containerID).
		Order("started_at DESC").
		First(&lastUpdate).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to get last update: %w", err)
		}
	} else {
		stats.LastUpdateAt = &lastUpdate.StartedAt
	}

	// Calculate success rate
	if stats.TotalUpdates > 0 {
		stats.SuccessRate = float64(stats.SuccessfulUpdates) / float64(stats.TotalUpdates) * 100
	}

	return &stats, nil
}

// GetSuccessRate calculates the success rate for a container's updates
func (r *updateHistoryRepository) GetSuccessRate(ctx context.Context, containerID int64) (float64, error) {
	if containerID <= 0 {
		return 0, fmt.Errorf("invalid container ID: %d", containerID)
	}

	var total, successful int64

	// Get total count
	if err := r.db.WithContext(ctx).
		Model(&model.UpdateHistory{}).
		Where("container_id = ?", containerID).
		Count(&total).Error; err != nil {
		return 0, fmt.Errorf("failed to get total updates count: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	// Get successful count
	if err := r.db.WithContext(ctx).
		Model(&model.UpdateHistory{}).
		Where("container_id = ? AND status = ?", containerID, model.UpdateStatusCompleted).
		Count(&successful).Error; err != nil {
		return 0, fmt.Errorf("failed to get successful updates count: %w", err)
	}

	return float64(successful) / float64(total) * 100, nil
}

// DeleteOldHistory deletes update history entries older than specified retention days
func (r *updateHistoryRepository) DeleteOldHistory(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be positive")
	}

	cutoffDate := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result := r.db.WithContext(ctx).
		Where("started_at < ?", cutoffDate).
		Delete(&model.UpdateHistory{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old update history: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// CreateBatch creates multiple update history entries in a single transaction
func (r *updateHistoryRepository) CreateBatch(ctx context.Context, histories []*model.UpdateHistory) error {
	if len(histories) == 0 {
		return fmt.Errorf("histories slice cannot be empty")
	}

	// Validate all histories before creating
	for i, history := range histories {
		if history == nil {
			return fmt.Errorf("update history at index %d cannot be nil", i)
		}
		if history.ContainerID <= 0 {
			return fmt.Errorf("container ID is required for history at index %d", i)
		}
	}

	if err := r.db.WithContext(ctx).CreateInBatches(histories, 100).Error; err != nil {
		return fmt.Errorf("failed to create update histories batch: %w", err)
	}

	return nil
}

// CountOlderThan counts update histories older than the specified date
func (r *updateHistoryRepository) CountOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.UpdateHistory{}).
		Where("created_at < ?", cutoffDate).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count old update histories: %w", err)
	}

	return count, nil
}

// DeleteOlderThan deletes update histories older than the specified date
func (r *updateHistoryRepository) DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&model.UpdateHistory{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old update histories: %w", result.Error)
	}

	return result.RowsAffected, nil
}