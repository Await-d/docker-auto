package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/registry"
	"docker-auto/pkg/utils"

	"github.com/sirupsen/logrus"
)

// RegistryService manages registry operations
type RegistryService struct {
	registryRepo repository.RegistryCredentialsRepository
	activityRepo repository.ActivityLogRepository
	imageService *ImageService
	encryptionKey string
	logger       *logrus.Logger
}

// NewRegistryService creates a new registry service
func NewRegistryService(
	registryRepo repository.RegistryCredentialsRepository,
	activityRepo repository.ActivityLogRepository,
	imageService *ImageService,
	encryptionKey string,
	logger *logrus.Logger,
) *RegistryService {
	return &RegistryService{
		registryRepo: registryRepo,
		activityRepo: activityRepo,
		imageService: imageService,
		encryptionKey: encryptionKey,
		logger:       logger,
	}
}

// RegistryResponse represents registry response data
type RegistryResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RegistryCreateRequest represents registry creation request
type RegistryCreateRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	URL         string `json:"url" binding:"required,url,max=255"`
	Type        string `json:"type" binding:"required"`
	Description string `json:"description,omitempty" binding:"max=500"`
	Username    string `json:"username,omitempty" binding:"max=100"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

// RegistryUpdateRequest represents registry update request
type RegistryUpdateRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	URL         string `json:"url" binding:"omitempty,url,max=255"`
	Type        string `json:"type" binding:"omitempty"`
	Description string `json:"description,omitempty" binding:"max=500"`
	Username    string `json:"username,omitempty" binding:"max=100"`
	Password    string `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
	IsDefault   *bool  `json:"is_default,omitempty"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// RegistryFilter represents registry query filters
type RegistryFilter struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	IsActive *bool  `json:"is_active,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// TestConnectionResult represents registry connection test result
type TestConnectionResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	ResponseTime string   `json:"response_time"`
	Capabilities []string `json:"capabilities,omitempty"`
	Info         interface{} `json:"info,omitempty"`
}

// RegistryStatistics represents registry usage statistics
type RegistryStatistics struct {
	TotalContainers    int64     `json:"total_containers"`
	ActiveContainers   int64     `json:"active_containers"`
	TotalPulls         int64     `json:"total_pulls"`
	RecentPulls        int64     `json:"recent_pulls"`
	LastPullTime       time.Time `json:"last_pull_time,omitempty"`
	AverageResponseTime string   `json:"average_response_time"`
	SuccessRate        float64   `json:"success_rate"`
}

// ListRegistries returns a list of registries based on filter
func (s *RegistryService) ListRegistries(ctx context.Context, userID int64, filter *RegistryFilter) ([]*RegistryResponse, int64, error) {
	// Convert filter to repository filter
	repoFilter := &model.RegistryCredentialsFilter{
		Name:     filter.Name,
		IsActive: filter.IsActive,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	}

	registries, total, err := s.registryRepo.List(ctx, repoFilter)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to list registries")
		return nil, 0, fmt.Errorf("failed to list registries: %w", err)
	}

	// Convert to response format
	responses := make([]*RegistryResponse, len(registries))
	for i, reg := range registries {
		responses[i] = s.convertToResponse(reg)
	}

	// Log activity
	s.logActivity(ctx, userID, "list_registries", fmt.Sprintf("Listed %d registries", len(responses)), nil)

	return responses, total, nil
}

// GetRegistry returns a specific registry by ID
func (s *RegistryService) GetRegistry(ctx context.Context, userID int64, registryID int64) (*RegistryResponse, error) {
	registry, err := s.registryRepo.GetByID(ctx, registryID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to get registry")
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	if registry == nil {
		return nil, fmt.Errorf("registry not found")
	}

	return s.convertToResponse(registry), nil
}

// CreateRegistry creates a new registry
func (s *RegistryService) CreateRegistry(ctx context.Context, userID int64, req *RegistryCreateRequest) (*RegistryResponse, error) {
	// Check if name already exists
	exists, err := s.registryRepo.Exists(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check registry existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("registry with name '%s' already exists", req.Name)
	}

	// Create registry model
	createdByInt := int(userID)
	registry := &model.RegistryCredentials{
		Name:        req.Name,
		RegistryURL: req.URL,
		Username:    req.Username,
		AuthType:    s.determineAuthType(req),
		IsDefault:   req.IsDefault,
		IsActive:    true,
		CreatedBy:   &createdByInt,
	}

	// Encrypt sensitive data
	if req.Password != "" {
		encrypted, err := utils.EncryptSensitiveData(req.Password, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		registry.PasswordEncrypted = encrypted
	}

	if req.Token != "" {
		encrypted, err := utils.EncryptSensitiveData(req.Token, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		registry.TokenEncrypted = encrypted
	}

	// Create metadata
	metadata := map[string]interface{}{
		"type":        req.Type,
		"description": req.Description,
		"created_by":  userID,
	}
	metadataJSON, _ := json.Marshal(metadata)
	registry.Metadata = string(metadataJSON)

	// Handle default registry logic
	if req.IsDefault {
		// Clear existing default
		if err := s.clearDefaultRegistry(ctx); err != nil {
			s.logger.WithError(err).Warn("Failed to clear existing default registry")
		}
	}

	// Save to database
	if err := s.registryRepo.Create(ctx, registry); err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"name":    req.Name,
			"url":     req.URL,
			"error":   err.Error(),
		}).Error("Failed to create registry")
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "create_registry", fmt.Sprintf("Created registry: %s", req.Name), map[string]interface{}{
		"registry_id": registry.ID,
		"name":        req.Name,
		"url":         req.URL,
		"type":        req.Type,
	})

	return s.convertToResponse(registry), nil
}

// UpdateRegistry updates an existing registry
func (s *RegistryService) UpdateRegistry(ctx context.Context, userID int64, registryID int64, req *RegistryUpdateRequest) (*RegistryResponse, error) {
	// Get existing registry
	registry, err := s.registryRepo.GetByID(ctx, registryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	if registry == nil {
		return nil, fmt.Errorf("registry not found")
	}

	// Update fields
	if req.Name != "" {
		registry.Name = req.Name
	}
	if req.URL != "" {
		registry.RegistryURL = req.URL
	}
	if req.Username != "" {
		registry.Username = req.Username
	}
	if req.IsDefault != nil {
		registry.IsDefault = *req.IsDefault
		if *req.IsDefault {
			// Clear other defaults
			if err := s.clearDefaultRegistry(ctx); err != nil {
				s.logger.WithError(err).Warn("Failed to clear existing default registry")
			}
		}
	}
	if req.IsActive != nil {
		registry.IsActive = *req.IsActive
	}

	// Handle password/token updates
	if req.Password != "" {
		encrypted, err := utils.EncryptSensitiveData(req.Password, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		registry.PasswordEncrypted = encrypted
	}
	if req.Token != "" {
		encrypted, err := utils.EncryptSensitiveData(req.Token, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt token: %w", err)
		}
		registry.TokenEncrypted = encrypted
	}

	// Update metadata
	var metadata map[string]interface{}
	if registry.Metadata != "" {
		json.Unmarshal([]byte(registry.Metadata), &metadata)
	} else {
		metadata = make(map[string]interface{})
	}

	if req.Type != "" {
		metadata["type"] = req.Type
	}
	if req.Description != "" {
		metadata["description"] = req.Description
	}
	metadata["updated_by"] = userID

	metadataJSON, _ := json.Marshal(metadata)
	registry.Metadata = string(metadataJSON)

	// Save changes
	if err := s.registryRepo.Update(ctx, registry); err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to update registry")
		return nil, fmt.Errorf("failed to update registry: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "update_registry", fmt.Sprintf("Updated registry: %s", registry.Name), map[string]interface{}{
		"registry_id": registryID,
		"name":        registry.Name,
	})

	return s.convertToResponse(registry), nil
}

// DeleteRegistry deletes a registry
func (s *RegistryService) DeleteRegistry(ctx context.Context, userID int64, registryID int64) error {
	// Get registry first to log the name
	registry, err := s.registryRepo.GetByID(ctx, registryID)
	if err != nil {
		return fmt.Errorf("failed to get registry: %w", err)
	}
	if registry == nil {
		return fmt.Errorf("registry not found")
	}

	// Delete from database
	if err := s.registryRepo.Delete(ctx, registryID); err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
		}).Error("Failed to delete registry")
		return fmt.Errorf("failed to delete registry: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "delete_registry", fmt.Sprintf("Deleted registry: %s", registry.Name), map[string]interface{}{
		"registry_id": registryID,
		"name":        registry.Name,
	})

	return nil
}

// TestConnection tests connection to a registry
func (s *RegistryService) TestConnection(ctx context.Context, userID int64, registryID int64) (*TestConnectionResult, error) {
	registryData, err := s.registryRepo.GetByID(ctx, registryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	if registryData == nil {
		return nil, fmt.Errorf("registry not found")
	}

	// Create auth config
	authConfig := &registry.AuthConfig{}
	if registryData.Username != "" {
		authConfig.Username = registryData.Username

		// Decrypt password if available
		if registryData.PasswordEncrypted != "" {
			password, err := utils.DecryptSensitiveData(registryData.PasswordEncrypted, s.encryptionKey)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to decrypt registry password")
			} else {
				authConfig.Password = password
			}
		}

		// Decrypt token if available
		if registryData.TokenEncrypted != "" {
			token, err := utils.DecryptSensitiveData(registryData.TokenEncrypted, s.encryptionKey)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to decrypt registry token")
			} else {
				authConfig.Token = token
			}
		}
	}

	// Test connection using ImageService
	start := time.Now()
	err = s.imageService.TestRegistryConnection(ctx, registryData.RegistryURL, authConfig)
	duration := time.Since(start)

	result := &TestConnectionResult{
		Success:      err == nil,
		ResponseTime: duration.String(),
	}

	if err != nil {
		result.Message = fmt.Sprintf("Connection failed: %v", err)
		s.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"error":       err.Error(),
			"duration":    duration,
		}).Warn("Registry connection test failed")
	} else {
		result.Message = "Connection successful"
		result.Capabilities = []string{"push", "pull", "search"}

		// Try to get registry info
		if info, err := s.imageService.GetRegistryInfo(ctx, registryData.RegistryURL); err == nil {
			result.Info = info
		}

		s.logger.WithFields(logrus.Fields{
			"user_id":     userID,
			"registry_id": registryID,
			"duration":    duration,
		}).Info("Registry connection test successful")
	}

	// Log activity
	s.logActivity(ctx, userID, "test_registry_connection", fmt.Sprintf("Tested connection to registry: %s", registryData.Name), map[string]interface{}{
		"registry_id":   registryID,
		"success":       result.Success,
		"response_time": duration.Milliseconds(),
	})

	return result, nil
}

// GetStatistics returns usage statistics for a registry
func (s *RegistryService) GetStatistics(ctx context.Context, userID int64, registryID int64) (*RegistryStatistics, error) {
	registryData, err := s.registryRepo.GetByID(ctx, registryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	if registryData == nil {
		return nil, fmt.Errorf("registry not found")
	}

	// TODO: Implement actual statistics gathering from database
	// For now, return basic statistics
	stats := &RegistryStatistics{
		TotalContainers:     0,
		ActiveContainers:    0,
		TotalPulls:          0,
		RecentPulls:         0,
		AverageResponseTime: "0ms",
		SuccessRate:         100.0,
	}

	return stats, nil
}

// Helper methods

func (s *RegistryService) convertToResponse(reg *model.RegistryCredentials) *RegistryResponse {
	// Parse metadata for type and description
	var metadata map[string]interface{}
	regType := "docker"
	description := ""

	if reg.Metadata != "" {
		if err := json.Unmarshal([]byte(reg.Metadata), &metadata); err == nil {
			if t, ok := metadata["type"].(string); ok {
				regType = t
			}
			if d, ok := metadata["description"].(string); ok {
				description = d
			}
		}
	}

	return &RegistryResponse{
		ID:          reg.ID,
		Name:        reg.Name,
		URL:         reg.RegistryURL,
		Type:        regType,
		Description: description,
		IsDefault:   reg.IsDefault,
		IsActive:    reg.IsActive,
		CreatedAt:   reg.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   reg.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *RegistryService) determineAuthType(req *RegistryCreateRequest) model.RegistryAuthType {
	if req.Token != "" {
		return model.RegistryAuthTypeToken
	}
	return model.RegistryAuthTypeBasic
}

func (s *RegistryService) clearDefaultRegistry(ctx context.Context) error {
	// This would need to be implemented in the repository
	// For now, we'll skip this step
	return nil
}

func (s *RegistryService) logActivity(ctx context.Context, userID int64, action, description string, metadata map[string]interface{}) {
	if s.activityRepo == nil {
		return
	}

	activity := &model.ActivityLog{
		UserID:      &userID,
		Action:      action,
		Description: description,
		IPAddress:   "127.0.0.1", // Should get from context
		UserAgent:   "docker-auto", // Should get from context
	}

	if metadata != nil {
		metadataJSON, _ := json.Marshal(metadata)
		activity.Metadata = string(metadataJSON)
	}

	if err := s.activityRepo.Create(ctx, activity); err != nil {
		s.logger.WithError(err).Warn("Failed to log activity")
	}
}