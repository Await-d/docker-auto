package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"docker-auto/internal/model"
	"docker-auto/pkg/docker"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// Batch operations

// BulkStartContainers starts multiple containers
func (s *ContainerService) BulkStartContainers(ctx context.Context, userID int64, containerIDs []int64) ([]*OperationResult, error) {
	results := make([]*OperationResult, len(containerIDs))

	for i, containerID := range containerIDs {
		result := &OperationResult{
			ContainerID: containerID,
		}

		// Get container info
		container, err := s.containerRepo.GetByID(ctx, containerID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to get container: %v", err)
			results[i] = result
			continue
		}

		result.Name = container.Name

		// Check permissions
		if err := s.checkContainerPermission(container, userID); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Permission denied: %v", err)
			results[i] = result
			continue
		}

		// Start container
		if err := s.StartContainer(ctx, userID, containerID); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to start: %v", err)
		} else {
			result.Success = true
			result.Message = "Container started successfully"
		}

		results[i] = result
	}

	// Log bulk operation
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	s.logUserActivity(userID, "bulk_start_containers", fmt.Sprintf("Bulk start operation: %d/%d successful", successCount, len(containerIDs)), map[string]interface{}{
		"container_ids":   containerIDs,
		"success_count":   successCount,
		"total_count":     len(containerIDs),
	})

	return results, nil
}

// BulkStopContainers stops multiple containers
func (s *ContainerService) BulkStopContainers(ctx context.Context, userID int64, containerIDs []int64) ([]*OperationResult, error) {
	results := make([]*OperationResult, len(containerIDs))

	for i, containerID := range containerIDs {
		result := &OperationResult{
			ContainerID: containerID,
		}

		// Get container info
		container, err := s.containerRepo.GetByID(ctx, containerID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to get container: %v", err)
			results[i] = result
			continue
		}

		result.Name = container.Name

		// Check permissions
		if err := s.checkContainerPermission(container, userID); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Permission denied: %v", err)
			results[i] = result
			continue
		}

		// Stop container
		if err := s.StopContainer(ctx, userID, containerID); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to stop: %v", err)
		} else {
			result.Success = true
			result.Message = "Container stopped successfully"
		}

		results[i] = result
	}

	// Log bulk operation
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	s.logUserActivity(userID, "bulk_stop_containers", fmt.Sprintf("Bulk stop operation: %d/%d successful", successCount, len(containerIDs)), map[string]interface{}{
		"container_ids":   containerIDs,
		"success_count":   successCount,
		"total_count":     len(containerIDs),
	})

	return results, nil
}

// BulkUpdateContainers performs bulk updates on multiple containers
func (s *ContainerService) BulkUpdateContainers(ctx context.Context, userID int64, req *BulkUpdateRequest) ([]*OperationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("bulk update request cannot be nil")
	}

	results := make([]*OperationResult, len(req.ContainerIDs))

	for i, containerID := range req.ContainerIDs {
		result := &OperationResult{
			ContainerID: containerID,
		}

		// Get container info
		container, err := s.containerRepo.GetByID(ctx, containerID)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to get container: %v", err)
			results[i] = result
			continue
		}

		result.Name = container.Name

		// Check permissions
		if err := s.checkContainerPermission(container, userID); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Permission denied: %v", err)
			results[i] = result
			continue
		}

		// Perform action based on request
		var actionErr error
		switch req.Action {
		case "start":
			actionErr = s.StartContainer(ctx, userID, containerID)
		case "stop":
			actionErr = s.StopContainer(ctx, userID, containerID)
		case "restart":
			actionErr = s.RestartContainer(ctx, userID, containerID)
		case "update":
			if req.UpdateImage != nil {
				_, actionErr = s.UpdateContainerImage(ctx, userID, containerID, req.UpdateImage)
			} else if req.Config != nil {
				updateReq := &UpdateContainerRequest{
					Config: req.Config,
				}
				actionErr = s.UpdateContainer(ctx, userID, containerID, updateReq)
			}
		default:
			actionErr = fmt.Errorf("unknown action: %s", req.Action)
		}

		if actionErr != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to %s: %v", req.Action, actionErr)
		} else {
			result.Success = true
			result.Message = fmt.Sprintf("Container %s successful", req.Action)
		}

		results[i] = result
	}

	// Log bulk operation
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	s.logUserActivity(userID, fmt.Sprintf("bulk_%s_containers", req.Action), fmt.Sprintf("Bulk %s operation: %d/%d successful", req.Action, successCount, len(req.ContainerIDs)), map[string]interface{}{
		"container_ids":   req.ContainerIDs,
		"action":          req.Action,
		"success_count":   successCount,
		"total_count":     len(req.ContainerIDs),
	})

	return results, nil
}

// Import and export operations

// ImportContainerFromDocker imports an existing Docker container
func (s *ContainerService) ImportContainerFromDocker(ctx context.Context, userID int64, dockerContainerID string) (*model.Container, error) {
	// Get Docker container info
	dockerContainer, err := s.dockerClient.GetContainer(ctx, dockerContainerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect Docker container: %w", err)
	}

	// Extract container configuration
	name := strings.TrimPrefix(dockerContainer.Name, "/")
	imageParts := strings.Split(dockerContainer.Config.Image, ":")
	image := imageParts[0]
	tag := "latest"
	if len(imageParts) > 1 {
		tag = imageParts[1]
	}

	// Check if container already exists
	exists, err := s.containerRepo.Exists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("container with name '%s' already exists", name)
	}

	// Create container configuration
	config := map[string]interface{}{
		"image":     dockerContainer.Config.Image,
		"env":       dockerContainer.Config.Env,
		"labels":    dockerContainer.Config.Labels,
		"cmd":       dockerContainer.Config.Cmd,
		"entrypoint": dockerContainer.Config.Entrypoint,
		"working_dir": dockerContainer.Config.WorkingDir,
		"exposed_ports": dockerContainer.Config.ExposedPorts,
		"volumes":     dockerContainer.Config.Volumes,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create container model
	container := &model.Container{
		Name:         name,
		Image:        image,
		Tag:          tag,
		ContainerID:  dockerContainerID,
		ConfigJSON:   string(configJSON),
		UpdatePolicy: model.UpdatePolicyManual,
		CreatedBy:    func() *int { u := int(userID); return &u }(),
	}

	// Set status based on Docker state
	switch dockerContainer.State.Status {
	case "running":
		container.Status = model.ContainerStatusRunning
	case "exited":
		container.Status = model.ContainerStatusExited
	case "paused":
		container.Status = model.ContainerStatusPaused
	case "restarting":
		container.Status = model.ContainerStatusRestarting
	default:
		container.Status = model.ContainerStatusStopped
	}

	// Save to database
	if err := s.containerRepo.Create(ctx, container); err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Log activity
	s.logContainerActivity(userID, int64(container.ID), "container_imported", "Container imported from Docker", map[string]interface{}{
		"docker_container_id": dockerContainerID,
		"container_name":      container.Name,
		"image":               container.GetFullImageName(),
	})

	// Invalidate cache
	s.invalidateContainerCache(userID)

	logrus.WithFields(logrus.Fields{
		"container_id":        container.ID,
		"container_name":      container.Name,
		"docker_container_id": dockerContainerID,
		"user_id":             userID,
	}).Info("Container imported successfully")

	return container, nil
}

// ExportContainerConfig exports container configuration
func (s *ContainerService) ExportContainerConfig(ctx context.Context, userID int64, containerID int64) (*ContainerExport, error) {
	// Get container
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	// Check permissions
	if err := s.checkContainerPermission(container, userID); err != nil {
		return nil, err
	}

	// Parse configuration
	var config map[string]interface{}
	if container.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(container.ConfigJSON), &config); err != nil {
			logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to parse container config")
			config = make(map[string]interface{})
		}
	}

	// Create export object
	export := &ContainerExport{
		Name:         container.Name,
		Image:        container.Image,
		Tag:          container.Tag,
		Config:       config,
		UpdatePolicy: string(container.UpdatePolicy),
		RegistryURL:  container.RegistryURL,
		ExportedAt:   time.Now(),
		Version:      "1.0",
	}

	// Extract specific configuration sections
	if labels, ok := config["labels"].(map[string]interface{}); ok {
		export.Labels = make(map[string]string)
		for k, v := range labels {
			if str, ok := v.(string); ok {
				export.Labels[k] = str
			}
		}
	}

	if env, ok := config["env"].([]interface{}); ok {
		export.Environment = make(map[string]string)
		for _, e := range env {
			if envStr, ok := e.(string); ok {
				parts := strings.SplitN(envStr, "=", 2)
				if len(parts) == 2 {
					export.Environment[parts[0]] = parts[1]
				}
			}
		}
	}

	// Add port mappings if available
	if ports, ok := config["ports"].([]interface{}); ok {
		for _, p := range ports {
			if portMap, ok := p.(map[string]interface{}); ok {
				mapping := PortMapping{}
				if containerPort, ok := portMap["container_port"].(float64); ok {
					mapping.ContainerPort = int(containerPort)
				}
				if hostPort, ok := portMap["host_port"].(float64); ok {
					mapping.HostPort = int(hostPort)
				}
				if protocol, ok := portMap["protocol"].(string); ok {
					mapping.Protocol = protocol
				}
				if hostIP, ok := portMap["host_ip"].(string); ok {
					mapping.HostIP = hostIP
				}
				export.Ports = append(export.Ports, mapping)
			}
		}
	}

	// Add volume mappings if available
	if volumes, ok := config["volumes"].([]interface{}); ok {
		for _, v := range volumes {
			if volumeMap, ok := v.(map[string]interface{}); ok {
				mapping := VolumeMapping{}
				if source, ok := volumeMap["source"].(string); ok {
					mapping.Source = source
				}
				if target, ok := volumeMap["target"].(string); ok {
					mapping.Target = target
				}
				if volType, ok := volumeMap["type"].(string); ok {
					mapping.Type = volType
				}
				if readOnly, ok := volumeMap["read_only"].(bool); ok {
					mapping.ReadOnly = readOnly
				}
				export.Volumes = append(export.Volumes, mapping)
			}
		}
	}

	// Log activity
	s.logContainerActivity(userID, containerID, "container_exported", "Container configuration exported", nil)

	return export, nil
}

// Helper and utility methods

// checkContainerPermission checks if user has permission to access container
func (s *ContainerService) checkContainerPermission(container *model.Container, userID int64) error {
	// For now, only allow access to containers created by the user
	// In a more complex system, you might have role-based permissions
	if container.CreatedBy == nil || int64(*container.CreatedBy) != userID {
		return fmt.Errorf("access denied: container belongs to different user")
	}
	return nil
}

// logContainerActivity logs container-related activities
func (s *ContainerService) logContainerActivity(userID int64, containerID int64, action, description string, metadata map[string]interface{}) {
	if s.activityRepo == nil {
		return
	}

	metadataJSON := "{}"
	if metadata != nil {
		if jsonBytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	activity := &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: "container",
		ResourceID:   func() *int { id := int(containerID); return &id }(),
		Description:  description,
		Metadata:     metadataJSON,
		IPAddress:    "", // Would be set from request context
		UserAgent:    "", // Would be set from request context
	}

	if err := s.activityRepo.Create(context.Background(), activity); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id":      userID,
			"container_id": containerID,
			"action":       action,
		}).Warn("Failed to log container activity")
	}
}

// logUserActivity logs user activities
func (s *ContainerService) logUserActivity(userID int64, action, description string, metadata map[string]interface{}) {
	if s.activityRepo == nil {
		return
	}

	metadataJSON := "{}"
	if metadata != nil {
		if jsonBytes, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(jsonBytes)
		}
	}

	activity := &model.ActivityLog{
		UserID:      &userID,
		Action:      action,
		Description: description,
		Metadata:    metadataJSON,
	}

	if err := s.activityRepo.Create(context.Background(), activity); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"action":  action,
		}).Warn("Failed to log user activity")
	}
}

// invalidateContainerCache invalidates container-related cache entries
func (s *ContainerService) invalidateContainerCache(userID int64) {
	if s.cache == nil {
		return
	}

	// Invalidate container list cache for user
	s.cache.Delete(fmt.Sprintf("container:list:%d", userID))

	// Could also invalidate other related caches
	s.cache.Delete("containers:stats")
	s.cache.Delete("containers:summary")
}

// validateImageExists checks if the Docker image exists (optional validation)
func (s *ContainerService) validateImageExists(ctx context.Context, image, tag string) error {
	fullImage := image
	if tag != "" && tag != "latest" {
		fullImage = fmt.Sprintf("%s:%s", image, tag)
	}

	// Try to inspect the image
	_, err := s.dockerClient.InspectImage(ctx, fullImage)
	if err != nil {
		// Try to pull the image
		if pullErr := s.dockerClient.PullImageAndWait(ctx, fullImage, types.ImagePullOptions{}); pullErr != nil {
			return fmt.Errorf("image not found and failed to pull: %w", pullErr)
		}
	}

	return nil
}

// createDockerContainer creates a Docker container from the container model
func (s *ContainerService) createDockerContainer(ctx context.Context, container *model.Container) (string, error) {
	// Parse container configuration
	var config map[string]interface{}
	if container.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(container.ConfigJSON), &config); err != nil {
			return "", fmt.Errorf("failed to parse container config: %w", err)
		}
	}

	// Build Docker create options
	createConfig := &docker.ContainerCreateConfig{
		Name:  container.Name,
		Image: container.Image,
		Tag:   container.Tag,
	}

	// Set environment variables
	if env, ok := config["env"].([]interface{}); ok {
		for _, e := range env {
			if envStr, ok := e.(string); ok {
				createConfig.Env = append(createConfig.Env, envStr)
			}
		}
	}

	// Set labels
	if labels, ok := config["labels"].(map[string]interface{}); ok {
		createConfig.Labels = make(map[string]string)
		for k, v := range labels {
			if str, ok := v.(string); ok {
				createConfig.Labels[k] = str
			}
		}
	}

	// Add our own labels
	if createConfig.Labels == nil {
		createConfig.Labels = make(map[string]string)
	}
	createConfig.Labels["docker-auto.container-id"] = strconv.Itoa(container.ID)
	createConfig.Labels["docker-auto.managed"] = "true"

	// Create the container
	containerID, err := s.dockerClient.CreateContainer(ctx, createConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create Docker container: %w", err)
	}

	return containerID.ID, nil
}

// getContainerMetrics retrieves container performance metrics
func (s *ContainerService) getContainerMetrics(ctx context.Context, dockerContainerID string) (*ContainerMetrics, error) {
	stats, err := s.dockerClient.GetContainerStats(ctx, dockerContainerID)
	if err != nil {
		return nil, err
	}

	metrics := &ContainerMetrics{
		CPUPercent:    calculateCPUPercent(stats),
		MemoryUsage:   int64(stats.MemoryStats.Usage),
		MemoryLimit:   int64(stats.MemoryStats.Limit),
		MemoryPercent: calculateMemoryPercent(stats),
		PIDs:          int(stats.PidsStats.Current),
		Timestamp:     time.Now(),
	}

	// Network I/O metrics
	if len(stats.Networks) > 0 {
		var totalRx, totalTx, totalRxPackets, totalTxPackets uint64
		for _, network := range stats.Networks {
			totalRx += network.RxBytes
			totalTx += network.TxBytes
			totalRxPackets += network.RxPackets
			totalTxPackets += network.TxPackets
		}
		metrics.NetworkIO = &NetworkIOMetrics{
			RxBytes:   int64(totalRx),
			TxBytes:   int64(totalTx),
			RxPackets: int64(totalRxPackets),
			TxPackets: int64(totalTxPackets),
		}
	}

	// Block I/O metrics
	if len(stats.BlkioStats.IoServiceBytesRecursive) > 0 {
		var readBytes, writeBytes, readOps, writeOps uint64
		for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
			if bio.Op == "Read" {
				readBytes += bio.Value
			} else if bio.Op == "Write" {
				writeBytes += bio.Value
			}
		}
		for _, bio := range stats.BlkioStats.IoServicedRecursive {
			if bio.Op == "Read" {
				readOps += bio.Value
			} else if bio.Op == "Write" {
				writeOps += bio.Value
			}
		}
		metrics.BlockIO = &BlockIOMetrics{
			ReadBytes:  int64(readBytes),
			WriteBytes: int64(writeBytes),
			ReadOps:    int64(readOps),
			WriteOps:   int64(writeOps),
		}
	}

	return metrics, nil
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	if stats.PreCPUStats.CPUUsage.TotalUsage == 0 {
		return 0.0
	}

	cpuDelta := stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
	systemDelta := stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage
	onlineCPUs := stats.CPUStats.OnlineCPUs

	if systemDelta > 0 && cpuDelta > 0 {
		return (float64(cpuDelta) / float64(systemDelta)) * float64(onlineCPUs) * 100.0
	}
	return 0.0
}

// calculateMemoryPercent calculates memory usage percentage
func calculateMemoryPercent(stats *types.StatsJSON) float64 {
	if stats.MemoryStats.Limit == 0 {
		return 0.0
	}
	return (float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit)) * 100.0
}

// getUpdateInfo gets update information for a container
func (s *ContainerService) getUpdateInfo(ctx context.Context, container *model.Container) (*UpdateInfo, error) {
	// This is a placeholder - would integrate with image service
	// For now, return basic info
	return &UpdateInfo{
		ContainerID:     int64(container.ID),
		Name:            container.Name,
		CurrentImage:    container.Image,
		CurrentTag:      container.Tag,
		UpdateAvailable: false, // Would be determined by image service
		LastChecked:     time.Now(),
	}, nil
}

// getLogsSample gets a sample of recent container logs
func (s *ContainerService) getLogsSample(ctx context.Context, dockerContainerID string) ([]string, error) {
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "10",
		Timestamps: false,
	}

	logs, err := s.dockerClient.GetContainerLogs(ctx, dockerContainerID, logOptions)
	if err != nil {
		return nil, err
	}
	defer logs.Close()

	// Simple implementation - just return empty sample for compilation
	sample := make([]string, 0)

	return sample, nil
}

// getDetailedDockerStatus gets detailed Docker status for a container
func (s *ContainerService) getDetailedDockerStatus(ctx context.Context, containerID string) (*docker.ContainerStatus, error) {
	// For now, create a basic Docker status from the simplified status
	status, err := s.dockerClient.GetContainerStatus(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Convert model.ContainerStatus to docker.ContainerStatus
	dockerStatus := &docker.ContainerStatus{
		ID:       containerID,
		State:    string(status),
		Status:   string(status),
		Running:  status == model.ContainerStatusRunning,
		Paused:   status == model.ContainerStatusPaused,
		Restarting: status == model.ContainerStatusRestarting,
		Dead:     status == model.ContainerStatusDead,
	}

	return dockerStatus, nil
}