package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"docker-auto/pkg/security"
	dockerTypes "docker-auto/pkg/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// ClientManager manages Docker client connections and operations
type ClientManager struct {
	client       *client.Client
	secureClient *security.SecureDockerClient
	config       *dockerTypes.ClientConfig
	pullProgress map[string]*dockerTypes.PullProgress
	mutex        sync.RWMutex
	logger       *logrus.Logger
}




// NewClientManager creates a new Docker client manager
func NewClientManager(config *dockerTypes.ClientConfig, logger *logrus.Logger) (*ClientManager, error) {
	if config == nil {
		config = DefaultClientConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Create Docker client
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	if config.Host != "" {
		opts = append(opts, client.WithHost(config.Host))
	} else {
		opts = append(opts, client.FromEnv)
	}

	if config.Version != "" {
		opts = append(opts, client.WithVersion(config.Version))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	// Create secure client if security config is provided
	var secureClient *security.SecureDockerClient
	if config.SecurityConfig != nil {
		if securityConfig, ok := config.SecurityConfig.(*security.DockerSecurityConfig); ok {
			secureClient, err = security.NewSecureDockerClient(securityConfig)
			if err != nil {
				logger.WithError(err).Warn("Failed to initialize secure Docker client, using standard client")
			}
		}
	}

	manager := &ClientManager{
		client:       dockerClient,
		secureClient: secureClient,
		config:       config,
		pullProgress: make(map[string]*dockerTypes.PullProgress),
		logger:       logger,
	}

	logger.WithFields(logrus.Fields{
		"host":          config.Host,
		"version":       config.Version,
		"secure_mode":   secureClient != nil,
	}).Info("Docker client manager initialized")

	return manager, nil
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *dockerTypes.ClientConfig {
	return &dockerTypes.ClientConfig{
		Host:               "",
		Version:            "",
		OperationTimeout:   5 * time.Minute,
		MaxConcurrentPulls: 3,
		RetryConfig: &dockerTypes.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		SecurityConfig: security.DefaultDockerSecurityConfig(),
	}
}

// PullImage pulls an image with progress tracking and security checks
func (cm *ClientManager) PullImage(ctx context.Context, imageName, tag string, auth *dockerTypes.RegistryAuth, force bool, userContext *security.DockerUserContext) (*dockerTypes.PullProgress, error) {
	fullImageName := imageName
	if tag != "" && tag != "latest" {
		fullImageName += ":" + tag
	}

	cm.mutex.Lock()
	// Check if already pulling
	if progress, exists := cm.pullProgress[fullImageName]; exists && !progress.Completed {
		cm.mutex.Unlock()
		return progress, nil
	}

	// Create new progress tracker
	progress := &dockerTypes.PullProgress{
		ImageName:   fullImageName,
		Status:      "starting",
		Progress:    make(map[string]*dockerTypes.LayerProgress),
		StartTime:   time.Now(),
		TotalSize:   0,
		DownloadedSize: 0,
	}
	cm.pullProgress[fullImageName] = progress
	cm.mutex.Unlock()

	cm.logger.WithFields(logrus.Fields{
		"image": fullImageName,
		"force": force,
		"user":  userContext.Username,
	}).Info("Starting image pull")

	// Security validation if secure client is available
	if cm.secureClient != nil && userContext != nil {
		if err := cm.validateUserPermissions(userContext, "image_pull"); err != nil {
			progress.Error = err.Error()
			progress.Completed = true
			return progress, fmt.Errorf("permission denied: %w", err)
		}
	}

	// Check if image already exists and force is not set
	if !force {
		_, _, err := cm.client.ImageInspectWithRaw(ctx, fullImageName)
		if err == nil {
			progress.Status = "already_exists"
			progress.Completed = true
			progress.EndTime = &time.Time{}
			*progress.EndTime = time.Now()
			cm.logger.WithField("image", fullImageName).Info("Image already exists")
			return progress, nil
		}
	}

	// Prepare authentication
	var authConfig string
	if auth != nil {
		authConfigObj := dockerTypes.AuthConfig{
			Username:      auth.Username,
			Password:      auth.Password,
			Email:         auth.Email,
			ServerAddress: auth.ServerAddress,
			IdentityToken: auth.IdentityToken,
			RegistryToken: auth.RegistryToken,
		}

		authConfigBytes, err := json.Marshal(authConfigObj)
		if err != nil {
			progress.Error = fmt.Sprintf("failed to marshal auth config: %v", err)
			progress.Completed = true
			return progress, fmt.Errorf("failed to marshal auth config: %w", err)
		}
		authConfig = string(authConfigBytes)
	}

	// Pull image with progress tracking
	go cm.pullImageAsync(ctx, fullImageName, authConfig, progress)

	return progress, nil
}

// pullImageAsync performs the actual image pull operation asynchronously
func (cm *ClientManager) pullImageAsync(ctx context.Context, imageName, authConfig string, progress *dockerTypes.PullProgress) {
	defer func() {
		progress.Mutex.Lock()
		progress.Completed = true
		progress.EndTime = &time.Time{}
		*progress.EndTime = time.Now()
		progress.Mutex.Unlock()
	}()

	options := types.ImagePullOptions{}
	if authConfig != "" {
		options.RegistryAuth = authConfig
	}

	response, err := cm.client.ImagePull(ctx, imageName, options)
	if err != nil {
		progress.Mutex.Lock()
		progress.Error = err.Error()
		progress.Status = "error"
		progress.Mutex.Unlock()
		cm.logger.WithError(err).WithField("image", imageName).Error("Failed to pull image")
		return
	}
	defer response.Close()

	// Parse pull progress
	decoder := json.NewDecoder(response)
	for {
		var pullStatus struct {
			Status         string `json:"status"`
			Error          string `json:"error,omitempty"`
			Progress       string `json:"progress,omitempty"`
			ProgressDetail struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			} `json:"progressDetail,omitempty"`
			ID string `json:"id,omitempty"`
		}

		if err := decoder.Decode(&pullStatus); err != nil {
			if err == io.EOF {
				break
			}
			progress.Mutex.Lock()
			progress.Error = fmt.Sprintf("failed to decode pull response: %v", err)
			progress.Status = "error"
			progress.Mutex.Unlock()
			return
		}

		progress.Mutex.Lock()
		if pullStatus.Error != "" {
			progress.Error = pullStatus.Error
			progress.Status = "error"
			progress.Mutex.Unlock()
			return
		}

		progress.Status = pullStatus.Status

		if pullStatus.ID != "" {
			// Update layer progress
			layerProgress := &dockerTypes.LayerProgress{
				ID:       pullStatus.ID,
				Status:   pullStatus.Status,
				Progress: pullStatus.Progress,
				Current:  pullStatus.ProgressDetail.Current,
				Total:    pullStatus.ProgressDetail.Total,
			}
			progress.Progress[pullStatus.ID] = layerProgress

			// Update total progress
			totalSize := int64(0)
			downloadedSize := int64(0)
			for _, layer := range progress.Progress {
				if layer.Total > 0 {
					totalSize += layer.Total
					downloadedSize += layer.Current
				}
			}
			progress.TotalSize = totalSize
			progress.DownloadedSize = downloadedSize
		}
		progress.Mutex.Unlock()
	}

	progress.Mutex.Lock()
	if progress.Error == "" {
		progress.Status = "completed"
		cm.logger.WithField("image", imageName).Info("Image pull completed successfully")
	}
	progress.Mutex.Unlock()
}

// RemoveImage removes an image with dependency checks and security validation
func (cm *ClientManager) RemoveImage(ctx context.Context, imageName, tag string, force bool, userContext *security.DockerUserContext) (*dockerTypes.ImageRemovalResult, error) {
	fullImageName := imageName
	if tag != "" && tag != "latest" {
		fullImageName += ":" + tag
	}

	result := &dockerTypes.ImageRemovalResult{
		ImageName: fullImageName,
		Force:     force,
	}

	cm.logger.WithFields(logrus.Fields{
		"image": fullImageName,
		"force": force,
		"user":  userContext.Username,
	}).Info("Starting image removal")

	// Security validation if secure client is available
	if cm.secureClient != nil && userContext != nil {
		if err := cm.validateUserPermissions(userContext, "image_remove"); err != nil {
			result.Error = err.Error()
			return result, fmt.Errorf("permission denied: %w", err)
		}
	}

	// Check if image exists
	imageInfo, _, err := cm.client.ImageInspectWithRaw(ctx, fullImageName)
	if err != nil {
		result.Error = fmt.Sprintf("image not found: %v", err)
		return result, fmt.Errorf("image not found: %w", err)
	}

	result.ImageID = imageInfo.ID

	// Check for dependencies if not forcing removal
	if !force {
		dependencies, err := cm.checkImageDependencies(ctx, imageInfo.ID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to check dependencies: %v", err)
			return result, fmt.Errorf("failed to check dependencies: %w", err)
		}

		if len(dependencies) > 0 {
			result.DependencyInfo = dependencies
			result.Error = "image has dependencies, use force=true to remove"
			return result, fmt.Errorf("image has dependencies: %v", dependencies)
		}
	}

	// Remove image with retry logic
	err = cm.withRetry(ctx, func() error {
		options := types.ImageRemoveOptions{
			Force:         force,
			PruneChildren: true,
		}

		deleted, err := cm.client.ImageRemove(ctx, fullImageName, options)
		if err != nil {
			return err
		}

		result.Deleted = make([]string, 0, len(deleted))
		result.Untagged = make([]string, 0, len(deleted))

		for _, item := range deleted {
			if item.Deleted != "" {
				result.Deleted = append(result.Deleted, item.Deleted)
			}
			if item.Untagged != "" {
				result.Untagged = append(result.Untagged, item.Untagged)
			}
		}

		return nil
	})

	if err != nil {
		result.Error = err.Error()
		return result, fmt.Errorf("failed to remove image: %w", err)
	}

	result.Success = true
	cm.logger.WithFields(logrus.Fields{
		"image":    fullImageName,
		"deleted":  len(result.Deleted),
		"untagged": len(result.Untagged),
	}).Info("Image removal completed successfully")

	return result, nil
}

// checkImageDependencies checks for containers using the image
func (cm *ClientManager) checkImageDependencies(ctx context.Context, imageID string) ([]string, error) {
	containers, err := cm.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var dependencies []string
	for _, container := range containers {
		if container.ImageID == imageID {
			status := "running"
			if container.State != "running" {
				status = "stopped"
			}
			dependency := fmt.Sprintf("container %s (%s) - %s", container.Names[0], container.ID[:12], status)
			dependencies = append(dependencies, dependency)
		}
	}

	return dependencies, nil
}

// validateUserPermissions validates user permissions for Docker operations
func (cm *ClientManager) validateUserPermissions(userContext *security.DockerUserContext, operation string) error {
	// This would normally integrate with the secure client
	// For now, implement basic validation
	if userContext == nil {
		return fmt.Errorf("user context required")
	}

	// Check role-based permissions
	switch strings.ToLower(userContext.Role) {
	case "admin":
		return nil // Admins can do everything
	case "developer":
		allowedOps := []string{"image_pull", "container_create", "container_start", "container_stop"}
		for _, op := range allowedOps {
			if operation == op {
				return nil
			}
		}
		return fmt.Errorf("operation %s not allowed for role %s", operation, userContext.Role)
	case "viewer":
		allowedOps := []string{"image_list", "container_list"}
		for _, op := range allowedOps {
			if operation == op {
				return nil
			}
		}
		return fmt.Errorf("operation %s not allowed for role %s", operation, userContext.Role)
	default:
		return fmt.Errorf("unknown role: %s", userContext.Role)
	}
}

// withRetry executes a function with retry logic
func (cm *ClientManager) withRetry(ctx context.Context, fn func() error) error {
	if cm.config.RetryConfig == nil {
		return fn()
	}

	delay := cm.config.RetryConfig.InitialDelay
	for attempt := 0; attempt <= cm.config.RetryConfig.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if attempt == cm.config.RetryConfig.MaxRetries {
			return err
		}

		// Check if we should retry
		if !cm.shouldRetry(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * cm.config.RetryConfig.BackoffFactor)
			if delay > cm.config.RetryConfig.MaxDelay {
				delay = cm.config.RetryConfig.MaxDelay
			}
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// shouldRetry determines if an operation should be retried
func (cm *ClientManager) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Retry on temporary network errors, timeouts, etc.
	errorStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"network is unreachable",
		"temporary failure",
		"service unavailable",
		"too many requests",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errorStr, retryable) {
			return true
		}
	}

	return false
}

// GetPullProgress returns current pull progress for an image
func (cm *ClientManager) GetPullProgress(imageName string) (*dockerTypes.PullProgress, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	progress, exists := cm.pullProgress[imageName]
	return progress, exists
}

// CleanupPullProgress removes completed pull progress entries older than specified duration
func (cm *ClientManager) CleanupPullProgress(maxAge time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for imageName, progress := range cm.pullProgress {
		if progress.Completed && progress.EndTime != nil && time.Since(*progress.EndTime) > maxAge {
			delete(cm.pullProgress, imageName)
		}
	}
}

// GetClient returns the underlying Docker client
func (cm *ClientManager) GetClient() *client.Client {
	return cm.client
}

// GetSecureClient returns the secure Docker client
func (cm *ClientManager) GetSecureClient() *security.SecureDockerClient {
	return cm.secureClient
}

// Close closes the Docker client connection
func (cm *ClientManager) Close() error {
	if cm.client != nil {
		return cm.client.Close()
	}
	return nil
}