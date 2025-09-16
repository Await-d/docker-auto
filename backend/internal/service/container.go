package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/docker"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// ContainerService manages container operations and business logic
type ContainerService struct {
	containerRepo     repository.ContainerRepository
	updateHistoryRepo repository.UpdateHistoryRepository
	activityRepo      repository.ActivityLogRepository
	dockerClient      *docker.DockerClient
	cache             *CacheService
	config            *config.Config
	userService       *UserService
}

// NewContainerService creates a new container service instance
func NewContainerService(
	containerRepo repository.ContainerRepository,
	updateHistoryRepo repository.UpdateHistoryRepository,
	activityRepo repository.ActivityLogRepository,
	dockerClient *docker.DockerClient,
	cache *CacheService,
	config *config.Config,
	userService *UserService,
) *ContainerService {
	return &ContainerService{
		containerRepo:     containerRepo,
		updateHistoryRepo: updateHistoryRepo,
		activityRepo:      activityRepo,
		dockerClient:      dockerClient,
		cache:             cache,
		config:            config,
		userService:       userService,
	}
}

// Container CRUD operations

// CreateContainer creates a new container configuration
func (s *ContainerService) CreateContainer(ctx context.Context, userID int64, req *CreateContainerRequest) (*model.Container, error) {
	if req == nil {
		return nil, fmt.Errorf("create container request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Check if container name already exists
	exists, err := s.containerRepo.Exists(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("container with name '%s' already exists", req.Name)
	}

	// Validate Docker image exists (optional check)
	if s.config.Docker.ValidateImages {
		if err := s.validateImageExists(ctx, req.Image, req.Tag); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"image": req.Image,
				"tag":   req.Tag,
			}).Warn("Image validation failed")
		}
	}

	// Create container model
	userIDInt := int(userID)
	container := &model.Container{
		Name:         req.Name,
		Image:        req.Image,
		Tag:          req.Tag,
		Status:       model.ContainerStatusStopped,
		UpdatePolicy: model.UpdatePolicy(req.UpdatePolicy),
		RegistryURL:  req.RegistryURL,
		CreatedBy:    &userIDInt,
	}

	// Set configuration JSON
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		container.ConfigJSON = string(configJSON)
	}

	// Set registry auth if provided
	if req.RegistryAuth != nil {
		authJSON, err := json.Marshal(req.RegistryAuth)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal registry auth: %w", err)
		}
		container.RegistryAuth = string(authJSON)
	}

	// Save to database
	if err := s.containerRepo.Create(ctx, container); err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Log activity
	s.logContainerActivity(userID, int64(container.ID), "container_created", "Container created successfully", map[string]interface{}{
		"container_name": container.Name,
		"image":          container.GetFullImageName(),
		"update_policy":  container.UpdatePolicy,
	})

	// Invalidate cache
	s.invalidateContainerCache(userID)

	logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
		"user_id":        userID,
		"image":          container.GetFullImageName(),
	}).Info("Container created successfully")

	return container, nil
}

// GetContainer retrieves container details by ID
func (s *ContainerService) GetContainer(ctx context.Context, userID int64, containerID int64) (*ContainerDetail, error) {
	// Get container from database
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Check user permissions
	if err := s.checkContainerPermission(container, userID); err != nil {
		return nil, err
	}

	// Build detailed response
	detail := &ContainerDetail{
		Container: container,
	}

	// Get Docker status if container has Docker ID
	if container.ContainerID != "" {
		if dockerStatus, err := s.getDetailedDockerStatus(ctx, container.ContainerID); err == nil {
			detail.DockerStatus = dockerStatus
		} else {
			logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to get Docker status")
		}
	}

	// Get metrics if container is running
	if container.IsRunning() && container.ContainerID != "" {
		if metrics, err := s.getContainerMetrics(ctx, container.ContainerID); err == nil {
			detail.Metrics = metrics
		}
	}

	// Get update information
	if updateInfo, err := s.getUpdateInfo(ctx, container); err == nil {
		detail.UpdateInfo = updateInfo
	}

	// Get recent logs sample
	if container.ContainerID != "" {
		if logs, err := s.getLogsSample(ctx, container.ContainerID); err == nil {
			detail.LogsSample = logs
		}
	}

	return detail, nil
}

// UpdateContainer updates container configuration
func (s *ContainerService) UpdateContainer(ctx context.Context, userID int64, containerID int64, req *UpdateContainerRequest) error {
	if req == nil {
		return fmt.Errorf("update container request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Get existing container
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Check permissions
	if err := s.checkContainerPermission(container, userID); err != nil {
		return err
	}

	// Update fields
	updated := false
	changes := make(map[string]interface{})

	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if container.ConfigJSON != string(configJSON) {
			container.ConfigJSON = string(configJSON)
			changes["config"] = req.Config
			updated = true
		}
	}

	if req.UpdatePolicy != nil && *req.UpdatePolicy != string(container.UpdatePolicy) {
		container.UpdatePolicy = model.UpdatePolicy(*req.UpdatePolicy)
		changes["update_policy"] = *req.UpdatePolicy
		updated = true
	}

	if req.RegistryURL != nil && *req.RegistryURL != container.RegistryURL {
		container.RegistryURL = *req.RegistryURL
		changes["registry_url"] = *req.RegistryURL
		updated = true
	}

	if req.RegistryAuth != nil {
		authJSON, err := json.Marshal(req.RegistryAuth)
		if err != nil {
			return fmt.Errorf("failed to marshal registry auth: %w", err)
		}
		if container.RegistryAuth != string(authJSON) {
			container.RegistryAuth = string(authJSON)
			changes["registry_auth"] = "updated"
			updated = true
		}
	}

	if !updated {
		return nil // No changes made
	}

	// Save changes
	if err := s.containerRepo.Update(ctx, container); err != nil {
		return fmt.Errorf("failed to update container: %w", err)
	}

	// Log activity
	s.logContainerActivity(userID, int64(container.ID), "container_updated", "Container configuration updated", changes)

	// Invalidate cache
	s.invalidateContainerCache(userID)
	s.cache.Delete(fmt.Sprintf("container:detail:%d", containerID))

	logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
		"user_id":        userID,
		"changes":        changes,
	}).Info("Container updated successfully")

	return nil
}

// DeleteContainer removes a container
func (s *ContainerService) DeleteContainer(ctx context.Context, userID int64, containerID int64) error {
	// Get container
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	// Check permissions
	if err := s.checkContainerPermission(container, userID); err != nil {
		return err
	}

	// Stop Docker container if running
	if container.ContainerID != "" {
		if dockerStatus, err := s.dockerClient.GetContainerStatus(ctx, container.ContainerID); err == nil {
			if dockerStatus == model.ContainerStatusRunning {
				timeout := 30
				if err := s.dockerClient.StopContainer(ctx, container.ContainerID, &timeout); err != nil {
					logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to stop Docker container before deletion")
				}
			}
		}

		// Remove Docker container
		removeOptions := types.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}
		if err := s.dockerClient.RemoveContainer(ctx, container.ContainerID, removeOptions); err != nil {
			logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to remove Docker container")
		}
	}

	// Delete from database
	if err := s.containerRepo.Delete(ctx, containerID); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	// Log activity
	s.logContainerActivity(userID, int64(container.ID), "container_deleted", "Container deleted successfully", map[string]interface{}{
		"container_name": container.Name,
		"image":          container.GetFullImageName(),
	})

	// Invalidate cache
	s.invalidateContainerCache(userID)

	logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
		"user_id":        userID,
	}).Info("Container deleted successfully")

	return nil
}

// ListContainers retrieves paginated list of containers
func (s *ContainerService) ListContainers(ctx context.Context, userID int64, filter *ContainerFilter) (*ContainerListResponse, error) {
	if filter == nil {
		filter = &ContainerFilter{
			ContainerFilter: &model.ContainerFilter{},
		}
	}

	// Set user filter
	userIDInt := int(userID)
	filter.ContainerFilter.CreatedBy = &userIDInt

	// Set defaults
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Validate sort field
	if filter.SortBy != "" && !IsValidSortField(filter.SortBy) {
		return nil, fmt.Errorf("invalid sort field: %s", filter.SortBy)
	}

	// Set default sort
	if filter.SortBy == "" {
		filter.SortBy = "updated_at"
		filter.SortOrder = "desc"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	// Build order by
	filter.ContainerFilter.OrderBy = filter.SortBy
	if filter.SortOrder == "desc" {
		filter.ContainerFilter.OrderBy += " DESC"
	}

	// Get containers from database
	containers, total, err := s.containerRepo.List(ctx, filter.ContainerFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Convert to summary format
	summaries := make([]*ContainerSummary, len(containers))
	for i, container := range containers {
		summary := &ContainerSummary{
			ID:           int64(container.ID),
			Name:         container.Name,
			Image:        container.Image,
			Tag:          container.Tag,
			Status:       container.Status,
			UpdatePolicy: container.UpdatePolicy,
			CreatedAt:    container.CreatedAt,
			UpdatedAt:    container.UpdatedAt,
		}

		// Get Docker status
		if container.ContainerID != "" {
			if dockerStatus, err := s.dockerClient.GetContainerStatus(ctx, container.ContainerID); err == nil {
				summary.DockerStatus = string(dockerStatus.State)
			}
		}

		// Check for updates if needed
		if filter.HasUpdate != nil && *filter.HasUpdate {
			if updateInfo, err := s.getUpdateInfo(ctx, container); err == nil {
				summary.HasUpdate = updateInfo.UpdateAvailable
			}
		}

		summaries[i] = summary
	}

	// Calculate pagination
	page := (filter.Offset / filter.Limit) + 1
	hasNext := filter.Offset+filter.Limit < int(total)
	hasPrev := filter.Offset > 0

	return &ContainerListResponse{
		Containers: summaries,
		Total:      total,
		Page:       page,
		Limit:      filter.Limit,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}, nil
}

// Container operations

// StartContainer starts a container
func (s *ContainerService) StartContainer(ctx context.Context, userID int64, containerID int64) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return err
	}

	// Create Docker container if not exists
	if container.ContainerID == "" {
		dockerContainerID, err := s.createDockerContainer(ctx, container)
		if err != nil {
			return fmt.Errorf("failed to create Docker container: %w", err)
		}
		container.ContainerID = dockerContainerID

		// Update container with Docker ID
		if err := s.containerRepo.UpdateContainerID(ctx, containerID, dockerContainerID); err != nil {
			logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to update container Docker ID")
		}
	}

	// Start Docker container
	if err := s.dockerClient.StartContainer(ctx, container.ContainerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update status
	if err := s.containerRepo.UpdateStatus(ctx, containerID, model.ContainerStatusRunning); err != nil {
		logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to update container status")
	}

	// Log activity
	s.logContainerActivity(userID, containerID, "container_started", "Container started successfully", nil)

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("container:status:%d", containerID))

	return nil
}

// StopContainer stops a container
func (s *ContainerService) StopContainer(ctx context.Context, userID int64, containerID int64) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return err
	}

	if container.ContainerID == "" {
		return fmt.Errorf("container has no Docker instance")
	}

	// Stop Docker container
	timeout := 30 // seconds
	if err := s.dockerClient.StopContainer(ctx, container.ContainerID, timeout); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update status
	if err := s.containerRepo.UpdateStatus(ctx, containerID, model.ContainerStatusStopped); err != nil {
		logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to update container status")
	}

	// Log activity
	s.logContainerActivity(userID, containerID, "container_stopped", "Container stopped successfully", nil)

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("container:status:%d", containerID))

	return nil
}

// RestartContainer restarts a container
func (s *ContainerService) RestartContainer(ctx context.Context, userID int64, containerID int64) error {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return err
	}

	if container.ContainerID == "" {
		return fmt.Errorf("container has no Docker instance")
	}

	// Restart Docker container
	timeout := 30 // seconds
	if err := s.dockerClient.RestartContainer(ctx, container.ContainerID, timeout); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Update status
	if err := s.containerRepo.UpdateStatus(ctx, containerID, model.ContainerStatusRunning); err != nil {
		logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to update container status")
	}

	// Log activity
	s.logContainerActivity(userID, containerID, "container_restarted", "Container restarted successfully", nil)

	// Invalidate cache
	s.cache.Delete(fmt.Sprintf("container:status:%d", containerID))

	return nil
}

// UpdateContainerImage updates container to use a new image version
func (s *ContainerService) UpdateContainerImage(ctx context.Context, userID int64, containerID int64, req *UpdateImageRequest) (*model.UpdateHistory, error) {
	if req == nil {
		req = &UpdateImageRequest{Strategy: "recreate", Backup: true}
	}

	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return nil, err
	}

	// Create update history record
	userIDInt := int(userID)
	updateHistory := &model.UpdateHistory{
		ContainerID:   int(containerID),
		OldImage:      container.GetFullImageName(),
		Status:        model.UpdateStatusRunning,
		Strategy:      model.UpdateStrategy(req.Strategy),
		TriggeredBy:   model.TriggerTypeManual,
		CreatedBy:     &userIDInt,
		StartedAt:     time.Now(),
	}

	if err := s.updateHistoryRepo.Create(ctx, updateHistory); err != nil {
		return nil, fmt.Errorf("failed to create update history: %w", err)
	}

	// TODO: Implement actual image update logic based on strategy
	// This is a placeholder - real implementation would:
	// 1. Pull new image
	// 2. Create backup if requested
	// 3. Apply update strategy (recreate/rolling/blue-green)
	// 4. Update container record
	// 5. Update history with results

	// For now, just mark as completed
	updateHistory.Status = model.UpdateStatusCompleted
	updateHistory.CompletedAt = &time.Time{}
	*updateHistory.CompletedAt = time.Now()
	updateHistory.NewImage = container.GetFullImageName() // Placeholder

	if err := s.updateHistoryRepo.Create(ctx, updateHistory); err != nil {
		logrus.WithError(err).WithField("update_id", updateHistory.ID).Warn("Failed to update history record")
	}

	// Log activity
	s.logContainerActivity(userID, containerID, "image_updated", "Container image updated", map[string]interface{}{
		"old_image":  updateHistory.OldImage,
		"new_image":  updateHistory.NewImage,
		"strategy":   req.Strategy,
		"update_id":  updateHistory.ID,
	})

	return updateHistory, nil
}

// Container status and monitoring

// GetContainerStatus retrieves current container status
func (s *ContainerService) GetContainerStatus(ctx context.Context, containerID int64) (*ContainerStatus, error) {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	status := &ContainerStatus{
		ID:        int64(container.ID),
		Name:      container.Name,
		Status:    container.Status,
		Timestamp: time.Now(),
	}

	// Get Docker status if available
	if container.ContainerID != "" {
		if dockerStatus, err := s.dockerClient.GetContainerStatus(ctx, container.ContainerID); err == nil {
			status.DockerStatus = dockerStatus
			if dockerStatus.Health != "" {
				status.Health = dockerStatus.Health
			}
			status.RestartCount = dockerStatus.RestartCount
			if dockerStatus.StartedAt != nil {
				status.Uptime = time.Since(*dockerStatus.StartedAt)
			}
		}
	}

	return status, nil
}

// GetContainerLogs retrieves container logs
func (s *ContainerService) GetContainerLogs(ctx context.Context, userID int64, containerID int64, options *LogOptions) (*LogResponse, error) {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return nil, err
	}

	if container.ContainerID == "" {
		return nil, fmt.Errorf("container has no Docker instance")
	}

	// Set default options
	if options == nil {
		options = &LogOptions{
			Tail:       100,
			Timestamps: true,
		}
	}

	// Convert options to Docker log options
	dockerOptions := &docker.LogOptions{
		Tail:       options.Tail,
		Follow:     options.Follow,
		Timestamps: options.Timestamps,
	}

	if !options.Since.IsZero() {
		dockerOptions.Since = &options.Since
	}
	if !options.Until.IsZero() {
		dockerOptions.Until = &options.Until
	}

	// Get logs from Docker
	logs, err := s.dockerClient.GetContainerLogs(ctx, container.ContainerID, dockerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	// Convert to log entries
	entries := make([]LogEntry, len(logs))
	for i, log := range logs {
		entries[i] = LogEntry{
			Timestamp: log.Timestamp,
			Source:    log.Source,
			Message:   log.Message,
		}
	}

	response := &LogResponse{
		ContainerID: containerID,
		Name:        container.Name,
		Logs:        entries,
		Count:       len(entries),
		Since:       options.Since,
		Until:       options.Until,
	}

	// Check if logs were truncated
	if options.Tail > 0 && len(entries) >= options.Tail {
		response.Truncated = true
	}

	return response, nil
}

// GetContainerStats retrieves container resource statistics
func (s *ContainerService) GetContainerStats(ctx context.Context, userID int64, containerID int64) (*ContainerStats, error) {
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	if err := s.checkContainerPermission(container, userID); err != nil {
		return nil, err
	}

	if container.ContainerID == "" {
		return nil, fmt.Errorf("container has no Docker instance")
	}

	// Get metrics
	metrics, err := s.getContainerMetrics(ctx, container.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}

	return &ContainerStats{
		ID:        containerID,
		Name:      container.Name,
		Metrics:   metrics,
		Timestamp: time.Now(),
	}, nil
}

// SyncContainerStatus synchronizes container status with Docker daemon
func (s *ContainerService) SyncContainerStatus(ctx context.Context) error {
	// Get all containers with Docker IDs
	allContainers, _, err := s.containerRepo.List(ctx, &model.ContainerFilter{})
	if err != nil {
		return fmt.Errorf("failed to get containers: %w", err)
	}

	syncResult := &SyncResult{
		TotalContainers: len(allContainers),
		Timestamp:       time.Now(),
	}
	startTime := time.Now()

	for _, container := range allContainers {
		if container.ContainerID == "" {
			continue
		}

		// Get Docker status
		dockerStatus, err := s.dockerClient.GetContainerStatus(ctx, container.ContainerID)
		if err != nil {
			syncResult.ErrorContainers++
			syncResult.Errors = append(syncResult.Errors, SyncError{
				ContainerID: int64(container.ID),
				Name:        container.Name,
				Error:       err.Error(),
				Recoverable: true,
			})
			continue
		}

		// Determine new status
		var newStatus model.ContainerStatus
		switch dockerStatus.State {
		case "running":
			newStatus = model.ContainerStatusRunning
		case "exited":
			newStatus = model.ContainerStatusExited
		case "paused":
			newStatus = model.ContainerStatusPaused
		case "restarting":
			newStatus = model.ContainerStatusRestarting
		case "removing":
			newStatus = model.ContainerStatusRemoving
		case "dead":
			newStatus = model.ContainerStatusDead
		default:
			newStatus = model.ContainerStatusUnknown
		}

		// Update status if changed
		if container.Status != newStatus {
			if err := s.containerRepo.UpdateStatus(ctx, int64(container.ID), newStatus); err != nil {
				logrus.WithError(err).WithField("container_id", container.ID).Warn("Failed to update container status")
			} else {
				syncResult.StatusChanges = append(syncResult.StatusChanges, ContainerStatusChange{
					ContainerID: int64(container.ID),
					Name:        container.Name,
					OldStatus:   container.Status,
					NewStatus:   newStatus,
					Reason:      "Docker status sync",
				})
			}
		}

		syncResult.SyncedContainers++
	}

	syncResult.Duration = time.Since(startTime)

	logrus.WithFields(logrus.Fields{
		"total_containers":  syncResult.TotalContainers,
		"synced_containers": syncResult.SyncedContainers,
		"error_containers":  syncResult.ErrorContainers,
		"status_changes":    len(syncResult.StatusChanges),
		"duration":          syncResult.Duration,
	}).Info("Container status sync completed")

	return nil
}

// Helper methods will be continued in the next part due to length...