package repository

import (
	"context"
	"fmt"
	"time"

	"docker-auto/internal/model"

	"gorm.io/gorm"
)

// containerRepository implements ContainerRepository interface
type containerRepository struct {
	db *gorm.DB
}

// NewContainerRepository creates a new container repository
func NewContainerRepository(db *gorm.DB) ContainerRepository {
	return &containerRepository{db: db}
}

// Create creates a new container
func (r *containerRepository) Create(ctx context.Context, container *model.Container) error {
	if container == nil {
		return fmt.Errorf("container cannot be nil")
	}

	// Validate required fields
	if container.Name == "" {
		return fmt.Errorf("container name is required")
	}
	if container.Image == "" {
		return fmt.Errorf("container image is required")
	}

	// Check for existing container with same name
	exists, err := r.Exists(ctx, container.Name)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return fmt.Errorf("container with name '%s' already exists", container.Name)
	}

	if err := r.db.WithContext(ctx).Create(container).Error; err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	return nil
}

// GetByID retrieves a container by ID
func (r *containerRepository) GetByID(ctx context.Context, id int64) (*model.Container, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid container ID: %d", id)
	}

	var container model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Preload("UpdateHistories").
		First(&container, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("container with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get container by ID: %w", err)
	}

	return &container, nil
}

// GetByName retrieves a container by name
func (r *containerRepository) GetByName(ctx context.Context, name string) (*model.Container, error) {
	if name == "" {
		return nil, fmt.Errorf("container name cannot be empty")
	}

	var container model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("name = ?", name).
		First(&container).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("container with name '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get container by name: %w", err)
	}

	return &container, nil
}

// GetByContainerID retrieves a container by Docker container ID
func (r *containerRepository) GetByContainerID(ctx context.Context, containerID string) (*model.Container, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	var container model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("container_id = ?", containerID).
		First(&container).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("container with container ID '%s' not found", containerID)
		}
		return nil, fmt.Errorf("failed to get container by container ID: %w", err)
	}

	return &container, nil
}

// Update updates an existing container
func (r *containerRepository) Update(ctx context.Context, container *model.Container) error {
	if container == nil {
		return fmt.Errorf("container cannot be nil")
	}
	if container.ID <= 0 {
		return fmt.Errorf("invalid container ID: %d", container.ID)
	}

	// Check if container exists
	var existingContainer model.Container
	if err := r.db.WithContext(ctx).First(&existingContainer, container.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("container with ID %d not found", container.ID)
		}
		return fmt.Errorf("failed to check container existence: %w", err)
	}

	// Check for duplicate name if changed
	if container.Name != existingContainer.Name {
		exists, err := r.Exists(ctx, container.Name)
		if err != nil {
			return fmt.Errorf("failed to check name uniqueness: %w", err)
		}
		if exists {
			return fmt.Errorf("container with name '%s' already exists", container.Name)
		}
	}

	// Update timestamp manually
	container.UpdatedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(container).Error; err != nil {
		return fmt.Errorf("failed to update container: %w", err)
	}

	return nil
}

// Delete deletes a container by ID
func (r *containerRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid container ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.Container{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete container: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("container with ID %d not found", id)
	}

	return nil
}

// List retrieves containers with filtering and pagination
func (r *containerRepository) List(ctx context.Context, filter *model.ContainerFilter) ([]*model.Container, int64, error) {
	var containers []*model.Container
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Container{})

	// Apply filters
	if filter != nil {
		if filter.CreatedBy != nil {
			query = query.Where("created_by = ?", *filter.CreatedBy)
		}
		if filter.Name != "" {
			query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
		}
		if filter.Image != "" {
			query = query.Where("image ILIKE ?", "%"+filter.Image+"%")
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.UpdatePolicy != "" {
			query = query.Where("update_policy = ?", filter.UpdatePolicy)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count containers: %w", err)
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
	query = query.Preload("CreatedByUser").Preload("UpdateHistories")

	if err := query.Find(&containers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, total, nil
}

// GetByStatus retrieves containers by status
func (r *containerRepository) GetByStatus(ctx context.Context, status model.ContainerStatus) ([]*model.Container, error) {
	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("status = ?", status).
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get containers by status: %w", err)
	}

	return containers, nil
}

// GetByUpdatePolicy retrieves containers by update policy
func (r *containerRepository) GetByUpdatePolicy(ctx context.Context, policy model.UpdatePolicy) ([]*model.Container, error) {
	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("update_policy = ?", policy).
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get containers by update policy: %w", err)
	}

	return containers, nil
}

// GetByCreatedBy retrieves containers created by a specific user
func (r *containerRepository) GetByCreatedBy(ctx context.Context, createdBy int64) ([]*model.Container, error) {
	if createdBy <= 0 {
		return nil, fmt.Errorf("invalid created by user ID: %d", createdBy)
	}

	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("created_by = ?", createdBy).
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get containers by created by: %w", err)
	}

	return containers, nil
}

// UpdateStatus updates the status of a container
func (r *containerRepository) UpdateStatus(ctx context.Context, id int64, status model.ContainerStatus) error {
	if id <= 0 {
		return fmt.Errorf("invalid container ID: %d", id)
	}

	result := r.db.WithContext(ctx).
		Model(&model.Container{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update container status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("container with ID %d not found", id)
	}

	return nil
}

// UpdateContainerID updates the Docker container ID
func (r *containerRepository) UpdateContainerID(ctx context.Context, id int64, containerID string) error {
	if id <= 0 {
		return fmt.Errorf("invalid container ID: %d", id)
	}

	result := r.db.WithContext(ctx).
		Model(&model.Container{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"container_id": containerID,
			"updated_at":   time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update container ID: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("container with ID %d not found", id)
	}

	return nil
}

// GetAutoUpdateContainers retrieves containers with auto update policy
func (r *containerRepository) GetAutoUpdateContainers(ctx context.Context) ([]*model.Container, error) {
	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("update_policy = ?", model.UpdatePolicyAuto).
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get auto update containers: %w", err)
	}

	return containers, nil
}

// UpdateStatusBatch updates the status of multiple containers
func (r *containerRepository) UpdateStatusBatch(ctx context.Context, ids []int64, status model.ContainerStatus) error {
	if len(ids) == 0 {
		return fmt.Errorf("container IDs slice cannot be empty")
	}

	// Validate IDs
	for i, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid container ID at index %d: %d", i, id)
		}
	}

	result := r.db.WithContext(ctx).
		Model(&model.Container{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update container status batch: %w", result.Error)
	}

	return nil
}

// GetByIDs retrieves multiple containers by their IDs
func (r *containerRepository) GetByIDs(ctx context.Context, ids []int64) ([]*model.Container, error) {
	if len(ids) == 0 {
		return []*model.Container{}, nil
	}

	// Validate IDs
	for i, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("invalid container ID at index %d: %d", i, id)
		}
	}

	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Preload("UpdateHistories").
		Where("id IN ?", ids).
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get containers by IDs: %w", err)
	}

	return containers, nil
}

// SearchByImage searches containers by image name pattern
func (r *containerRepository) SearchByImage(ctx context.Context, image string) ([]*model.Container, error) {
	if image == "" {
		return nil, fmt.Errorf("image pattern cannot be empty")
	}

	var containers []*model.Container
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("image ILIKE ?", "%"+image+"%").
		Find(&containers).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search containers by image: %w", err)
	}

	return containers, nil
}

// Exists checks if a container with given name exists
func (r *containerRepository) Exists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("container name cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Container{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check container existence: %w", err)
	}

	return count > 0, nil
}

// registryCredentialsRepository implements RegistryCredentialsRepository interface
type registryCredentialsRepository struct {
	db *gorm.DB
}

// NewRegistryCredentialsRepository creates a new registry credentials repository
func NewRegistryCredentialsRepository(db *gorm.DB) RegistryCredentialsRepository {
	return &registryCredentialsRepository{db: db}
}

// Create creates new registry credentials
func (r *registryCredentialsRepository) Create(ctx context.Context, credentials *model.RegistryCredentials) error {
	if credentials == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	// Validate required fields
	if credentials.Name == "" {
		return fmt.Errorf("credentials name is required")
	}
	if credentials.RegistryURL == "" {
		return fmt.Errorf("registry URL is required")
	}

	// Check for existing credentials with same name
	exists, err := r.Exists(ctx, credentials.Name)
	if err != nil {
		return fmt.Errorf("failed to check credentials existence: %w", err)
	}
	if exists {
		return fmt.Errorf("credentials with name '%s' already exist", credentials.Name)
	}

	if err := r.db.WithContext(ctx).Create(credentials).Error; err != nil {
		return fmt.Errorf("failed to create registry credentials: %w", err)
	}

	return nil
}

// GetByID retrieves registry credentials by ID
func (r *registryCredentialsRepository) GetByID(ctx context.Context, id int64) (*model.RegistryCredentials, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid credentials ID: %d", id)
	}

	var credentials model.RegistryCredentials
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		First(&credentials, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("credentials with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get credentials by ID: %w", err)
	}

	return &credentials, nil
}

// GetByName retrieves registry credentials by name
func (r *registryCredentialsRepository) GetByName(ctx context.Context, name string) (*model.RegistryCredentials, error) {
	if name == "" {
		return nil, fmt.Errorf("credentials name cannot be empty")
	}

	var credentials model.RegistryCredentials
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("name = ?", name).
		First(&credentials).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("credentials with name '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to get credentials by name: %w", err)
	}

	return &credentials, nil
}

// Update updates existing registry credentials
func (r *registryCredentialsRepository) Update(ctx context.Context, credentials *model.RegistryCredentials) error {
	if credentials == nil {
		return fmt.Errorf("credentials cannot be nil")
	}
	if credentials.ID <= 0 {
		return fmt.Errorf("invalid credentials ID: %d", credentials.ID)
	}

	// Check if credentials exist
	var existingCredentials model.RegistryCredentials
	if err := r.db.WithContext(ctx).First(&existingCredentials, credentials.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("credentials with ID %d not found", credentials.ID)
		}
		return fmt.Errorf("failed to check credentials existence: %w", err)
	}

	// Check for duplicate name if changed
	if credentials.Name != existingCredentials.Name {
		exists, err := r.Exists(ctx, credentials.Name)
		if err != nil {
			return fmt.Errorf("failed to check name uniqueness: %w", err)
		}
		if exists {
			return fmt.Errorf("credentials with name '%s' already exist", credentials.Name)
		}
	}

	// Update timestamp manually
	credentials.UpdatedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(credentials).Error; err != nil {
		return fmt.Errorf("failed to update registry credentials: %w", err)
	}

	return nil
}

// Delete deletes registry credentials by ID
func (r *registryCredentialsRepository) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid credentials ID: %d", id)
	}

	result := r.db.WithContext(ctx).Delete(&model.RegistryCredentials{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete registry credentials: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("credentials with ID %d not found", id)
	}

	return nil
}

// List retrieves registry credentials with filtering and pagination
func (r *registryCredentialsRepository) List(ctx context.Context, filter *model.RegistryCredentialsFilter) ([]*model.RegistryCredentials, int64, error) {
	var credentials []*model.RegistryCredentials
	var total int64

	query := r.db.WithContext(ctx).Model(&model.RegistryCredentials{})

	// Apply filters
	if filter != nil {
		if filter.RegistryURL != "" {
			query = query.Where("registry_url ILIKE ?", "%"+filter.RegistryURL+"%")
		}
		if filter.AuthType != "" {
			query = query.Where("auth_type = ?", filter.AuthType)
		}
		if filter.IsDefault != nil {
			query = query.Where("is_default = ?", *filter.IsDefault)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.CreatedBy != nil {
			query = query.Where("created_by = ?", *filter.CreatedBy)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count registry credentials: %w", err)
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
	query = query.Preload("CreatedByUser")

	if err := query.Find(&credentials).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list registry credentials: %w", err)
	}

	return credentials, total, nil
}

// GetByRegistryURL retrieves credentials by registry URL
func (r *registryCredentialsRepository) GetByRegistryURL(ctx context.Context, registryURL string) ([]*model.RegistryCredentials, error) {
	if registryURL == "" {
		return nil, fmt.Errorf("registry URL cannot be empty")
	}

	var credentials []*model.RegistryCredentials
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("registry_url = ? AND is_active = ?", registryURL, true).
		Find(&credentials).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get credentials by registry URL: %w", err)
	}

	return credentials, nil
}

// GetDefault retrieves the default registry credentials
func (r *registryCredentialsRepository) GetDefault(ctx context.Context) (*model.RegistryCredentials, error) {
	var credentials model.RegistryCredentials
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("is_default = ? AND is_active = ?", true, true).
		First(&credentials).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default registry credentials found")
		}
		return nil, fmt.Errorf("failed to get default credentials: %w", err)
	}

	return &credentials, nil
}

// GetActive retrieves all active registry credentials
func (r *registryCredentialsRepository) GetActive(ctx context.Context) ([]*model.RegistryCredentials, error) {
	var credentials []*model.RegistryCredentials
	err := r.db.WithContext(ctx).
		Preload("CreatedByUser").
		Where("is_active = ?", true).
		Find(&credentials).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get active credentials: %w", err)
	}

	return credentials, nil
}

// SetDefault sets a credentials entry as default (unsets others)
func (r *registryCredentialsRepository) SetDefault(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid credentials ID: %d", id)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, unset all other defaults
		if err := tx.Model(&model.RegistryCredentials{}).
			Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}

		// Then set the specified one as default
		result := tx.Model(&model.RegistryCredentials{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"is_default": true,
				"updated_at": time.Now().UTC(),
			})

		if result.Error != nil {
			return fmt.Errorf("failed to set default credentials: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("credentials with ID %d not found", id)
		}

		return nil
	})
}

// SetActive sets the active status of registry credentials
func (r *registryCredentialsRepository) SetActive(ctx context.Context, id int64, isActive bool) error {
	if id <= 0 {
		return fmt.Errorf("invalid credentials ID: %d", id)
	}

	result := r.db.WithContext(ctx).
		Model(&model.RegistryCredentials{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  isActive,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update credentials status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("credentials with ID %d not found", id)
	}

	return nil
}

// Exists checks if registry credentials with given name exist
func (r *registryCredentialsRepository) Exists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("credentials name cannot be empty")
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.RegistryCredentials{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check credentials existence: %w", err)
	}

	return count > 0, nil
}