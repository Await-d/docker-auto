package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/repository"
	"docker-auto/pkg/docker"

	"github.com/sirupsen/logrus"
)

// ContainerMonitoringService manages real-time container monitoring and data processing
type ContainerMonitoringService struct {
	containerRepo     repository.ContainerRepository
	metricsRepo       repository.MonitoringMetricsRepository
	monitor           *docker.ContainerMonitor
	cache             *CacheService
	config            *config.Config

	// Real-time monitoring state
	activeMonitoring  map[string]*MonitoringSession
	mu                sync.RWMutex

	// WebSocket connections for real-time updates
	subscribers       map[string][]chan *ContainerMetricsUpdate
	subscribersMu     sync.RWMutex

	// Background workers
	dataProcessor     *MetricsDataProcessor
	alertManager      *AlertManager

	logger            *logrus.Entry
}

// MonitoringSession represents an active container monitoring session
type MonitoringSession struct {
	ContainerID    string                    `json:"container_id"`
	ContainerName  string                    `json:"container_name"`
	UserID         int64                     `json:"user_id"`
	StartTime      time.Time                 `json:"start_time"`
	LastUpdate     time.Time                 `json:"last_update"`
	MetricsCount   int64                     `json:"metrics_count"`
	IsActive       bool                      `json:"is_active"`
	Config         *MonitoringSessionConfig  `json:"config"`

	// Internal state
	cancel         context.CancelFunc
	metricsChan    chan *ContainerMetricsUpdate
	errorChan      chan error
	mu             sync.RWMutex
}

// MonitoringSessionConfig configures individual monitoring sessions
type MonitoringSessionConfig struct {
	UpdateInterval    time.Duration `json:"update_interval"`
	AlertThresholds   *AlertThresholds `json:"alert_thresholds"`
	EnableAlerts      bool          `json:"enable_alerts"`
	EnableDataLogging bool          `json:"enable_data_logging"`
	RetentionPeriod   time.Duration `json:"retention_period"`
}

// AlertThresholds defines alerting thresholds for container metrics
type AlertThresholds struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryPercent    float64 `json:"memory_percent"`
	DiskUsagePercent float64 `json:"disk_usage_percent"`
	NetworkErrorRate float64 `json:"network_error_rate"`
}

// ContainerMetricsUpdate represents a real-time metrics update
type ContainerMetricsUpdate struct {
	ContainerID   string              `json:"container_id"`
	ContainerName string              `json:"container_name"`
	Metrics       *ContainerMetrics   `json:"metrics"`
	Timestamp     time.Time           `json:"timestamp"`
	UserID        int64               `json:"user_id"`
	Alerts        []*MetricsAlert     `json:"alerts,omitempty"`
}

// MetricsAlert represents a monitoring alert
type MetricsAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Threshold   float64   `json:"threshold"`
	ActualValue float64   `json:"actual_value"`
	Timestamp   time.Time `json:"timestamp"`
}

// MetricsDataProcessor processes and aggregates metrics data
type MetricsDataProcessor struct {
	metricsRepo repository.MonitoringMetricsRepository
	config      *config.Config
	logger      *logrus.Entry
}

// AlertManager manages monitoring alerts
type AlertManager struct {
	thresholds map[string]*AlertThresholds
	config     *config.Config
	logger     *logrus.Entry
	mu         sync.RWMutex
}

// NewContainerMonitoringService creates a new container monitoring service
func NewContainerMonitoringService(
	containerRepo repository.ContainerRepository,
	metricsRepo repository.MonitoringMetricsRepository,
	monitor *docker.ContainerMonitor,
	cache *CacheService,
	config *config.Config,
) *ContainerMonitoringService {

	logger := logrus.WithField("component", "container_monitoring_service")

	// Create data processor
	dataProcessor := &MetricsDataProcessor{
		metricsRepo: metricsRepo,
		config:      config,
		logger:      logger.WithField("subcomponent", "data_processor"),
	}

	// Create alert manager
	alertManager := &AlertManager{
		thresholds: make(map[string]*AlertThresholds),
		config:     config,
		logger:     logger.WithField("subcomponent", "alert_manager"),
	}

	service := &ContainerMonitoringService{
		containerRepo:    containerRepo,
		metricsRepo:      metricsRepo,
		monitor:          monitor,
		cache:           cache,
		config:          config,
		activeMonitoring: make(map[string]*MonitoringSession),
		subscribers:     make(map[string][]chan *ContainerMetricsUpdate),
		dataProcessor:   dataProcessor,
		alertManager:    alertManager,
		logger:          logger,
	}

	// Start background workers
	go service.metricsCollectionWorker()
	go service.dataProcessingWorker()
	go service.alertProcessingWorker()

	logger.Info("Container monitoring service initialized")
	return service
}

// StartMonitoring starts monitoring for a specific container
func (s *ContainerMonitoringService) StartMonitoring(ctx context.Context, userID int64, containerID int64, config *MonitoringSessionConfig) error {
	// Get container information
	container, err := s.containerRepo.GetByID(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}

	if container.ContainerID == "" {
		return fmt.Errorf("container has no Docker instance")
	}

	// Check if already monitoring
	s.mu.RLock()
	if session, exists := s.activeMonitoring[container.ContainerID]; exists {
		if session.IsActive {
			s.mu.RUnlock()
			return fmt.Errorf("container is already being monitored")
		}
	}
	s.mu.RUnlock()

	// Set default config if not provided
	if config == nil {
		config = &MonitoringSessionConfig{
			UpdateInterval:    time.Duration(s.config.Monitoring.UpdateInterval) * time.Second,
			EnableAlerts:      true,
			EnableDataLogging: true,
			RetentionPeriod:   24 * time.Hour,
			AlertThresholds: &AlertThresholds{
				CPUPercent:       80.0,
				MemoryPercent:    85.0,
				DiskUsagePercent: 90.0,
				NetworkErrorRate: 5.0,
			},
		}
	}

	// Create monitoring context
	monitorCtx, cancel := context.WithCancel(ctx)

	// Create monitoring session
	session := &MonitoringSession{
		ContainerID:   container.ContainerID,
		ContainerName: container.Name,
		UserID:        userID,
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		IsActive:      true,
		Config:        config,
		cancel:        cancel,
		metricsChan:   make(chan *ContainerMetricsUpdate, 100),
		errorChan:     make(chan error, 10),
	}

	// Store session
	s.mu.Lock()
	s.activeMonitoring[container.ContainerID] = session
	s.mu.Unlock()

	// Start Docker-level monitoring if not already started
	if s.monitor != nil && !s.monitor.IsMonitoring(container.ContainerID) {
		if err := s.monitor.StartMonitoring(monitorCtx, container.ContainerID); err != nil {
			s.mu.Lock()
			delete(s.activeMonitoring, container.ContainerID)
			s.mu.Unlock()
			cancel()
			return fmt.Errorf("failed to start Docker monitoring: %w", err)
		}
	}

	// Start session processing
	go s.processMonitoringSession(monitorCtx, session)

	// Set alert thresholds
	if config.EnableAlerts && config.AlertThresholds != nil {
		s.alertManager.SetThresholds(container.ContainerID, config.AlertThresholds)
	}

	s.logger.WithFields(logrus.Fields{
		"container_id":   container.ContainerID,
		"container_name": container.Name,
		"user_id":        userID,
	}).Info("Started container monitoring")

	return nil
}

// StopMonitoring stops monitoring for a specific container
func (s *ContainerMonitoringService) StopMonitoring(containerID string) error {
	s.mu.Lock()
	session, exists := s.activeMonitoring[containerID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("container is not being monitored")
	}

	// Mark as inactive
	session.mu.Lock()
	session.IsActive = false
	session.mu.Unlock()

	// Cancel monitoring context
	session.cancel()

	// Clean up channels
	close(session.metricsChan)
	close(session.errorChan)

	// Remove from active monitoring
	delete(s.activeMonitoring, containerID)
	s.mu.Unlock()

	// Stop Docker-level monitoring
	if s.monitor != nil && s.monitor.IsMonitoring(containerID) {
		if err := s.monitor.StopMonitoring(containerID); err != nil {
			s.logger.WithError(err).WithField("container_id", containerID).Warn("Failed to stop Docker monitoring")
		}
	}

	// Remove alert thresholds
	s.alertManager.RemoveThresholds(containerID)

	// Close any subscribers
	s.subscribersMu.Lock()
	if subscribers, exists := s.subscribers[containerID]; exists {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(s.subscribers, containerID)
	}
	s.subscribersMu.Unlock()

	s.logger.WithField("container_id", containerID).Info("Stopped container monitoring")
	return nil
}

// GetContainerMetrics retrieves current metrics for a container
func (s *ContainerMonitoringService) GetContainerMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	// Try to get from monitor cache first
	if s.monitor != nil {
		if metrics, err := s.monitor.GetContainerMetrics(ctx, containerID); err == nil {
			return s.convertDockerMetrics(metrics), nil
		}
	}

	return nil, fmt.Errorf("metrics not available for container %s", containerID)
}

// GetAllMonitoredContainers returns metrics for all monitored containers
func (s *ContainerMonitoringService) GetAllMonitoredContainers(ctx context.Context) (map[string]*ContainerMetrics, error) {
	s.mu.RLock()
	containerIDs := make([]string, 0, len(s.activeMonitoring))
	for containerID, session := range s.activeMonitoring {
		if session.IsActive {
			containerIDs = append(containerIDs, containerID)
		}
	}
	s.mu.RUnlock()

	result := make(map[string]*ContainerMetrics)
	for _, containerID := range containerIDs {
		if metrics, err := s.GetContainerMetrics(ctx, containerID); err == nil {
			result[containerID] = metrics
		}
	}

	return result, nil
}

// SubscribeToMetrics subscribes to real-time metrics updates for a container
func (s *ContainerMonitoringService) SubscribeToMetrics(containerID string) <-chan *ContainerMetricsUpdate {
	updateChan := make(chan *ContainerMetricsUpdate, 100)

	s.subscribersMu.Lock()
	if _, exists := s.subscribers[containerID]; !exists {
		s.subscribers[containerID] = make([]chan *ContainerMetricsUpdate, 0)
	}
	s.subscribers[containerID] = append(s.subscribers[containerID], updateChan)
	s.subscribersMu.Unlock()

	return updateChan
}

// UnsubscribeFromMetrics unsubscribes from metrics updates
func (s *ContainerMonitoringService) UnsubscribeFromMetrics(containerID string, updateChan <-chan *ContainerMetricsUpdate) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	if subscribers, exists := s.subscribers[containerID]; exists {
		// Find and remove the channel
		for i, ch := range subscribers {
			if ch == updateChan {
				// Remove from slice
				subscribers[i] = subscribers[len(subscribers)-1]
				subscribers = subscribers[:len(subscribers)-1]
				s.subscribers[containerID] = subscribers
				break
			}
		}

		// Note: Cannot close read-only channel, this should be handled by the publisher
	}
}

// GetMonitoringStatus returns the status of all monitoring sessions
func (s *ContainerMonitoringService) GetMonitoringStatus() map[string]*MonitoringSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]*MonitoringSession)
	for containerID, session := range s.activeMonitoring {
		// Create a safe copy
		status[containerID] = &MonitoringSession{
			ContainerID:   session.ContainerID,
			ContainerName: session.ContainerName,
			UserID:        session.UserID,
			StartTime:     session.StartTime,
			LastUpdate:    session.LastUpdate,
			MetricsCount:  session.MetricsCount,
			IsActive:      session.IsActive,
			Config:        session.Config,
		}
	}

	return status
}

// GetSystemMetrics returns monitoring system performance metrics
func (s *ContainerMonitoringService) GetSystemMetrics() *docker.MonitoringMetrics {
	if s.monitor != nil {
		return s.monitor.GetMonitoringMetrics()
	}
	return &docker.MonitoringMetrics{}
}

// processMonitoringSession processes metrics for a specific container session
func (s *ContainerMonitoringService) processMonitoringSession(ctx context.Context, session *MonitoringSession) {
	ticker := time.NewTicker(session.Config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get metrics from monitor
			if s.monitor != nil {
				dockerMetrics, err := s.monitor.GetContainerMetrics(ctx, session.ContainerID)
				if err != nil {
					session.errorChan <- fmt.Errorf("failed to get metrics: %w", err)
					continue
				}

				// Convert and process metrics
				metrics := s.convertDockerMetrics(dockerMetrics)

				// Check for alerts
				var alerts []*MetricsAlert
				if session.Config.EnableAlerts {
					alerts = s.alertManager.CheckThresholds(session.ContainerID, metrics)
				}

				// Create metrics update
				update := &ContainerMetricsUpdate{
					ContainerID:   session.ContainerID,
					ContainerName: session.ContainerName,
					Metrics:       metrics,
					Timestamp:     time.Now(),
					UserID:        session.UserID,
					Alerts:        alerts,
				}

				// Update session statistics
				session.mu.Lock()
				session.LastUpdate = time.Now()
				session.MetricsCount++
				session.mu.Unlock()

				// Send to session channel
				select {
				case session.metricsChan <- update:
				default:
					// Channel full, skip this update
					s.logger.WithField("container_id", session.ContainerID).Warn("Metrics channel full, skipping update")
				}

				// Broadcast to subscribers
				s.broadcastMetricsUpdate(session.ContainerID, update)
			}
		}
	}
}

// broadcastMetricsUpdate sends metrics update to all subscribers
func (s *ContainerMonitoringService) broadcastMetricsUpdate(containerID string, update *ContainerMetricsUpdate) {
	s.subscribersMu.RLock()
	subscribers, exists := s.subscribers[containerID]
	if !exists || len(subscribers) == 0 {
		s.subscribersMu.RUnlock()
		return
	}
	s.subscribersMu.RUnlock()

	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}

// convertDockerMetrics converts docker.ContainerMetrics to service.ContainerMetrics
func (s *ContainerMonitoringService) convertDockerMetrics(dockerMetrics *docker.ContainerMetrics) *ContainerMetrics {
	metrics := &ContainerMetrics{
		CPUPercent:    dockerMetrics.CPU.CPUPercent,
		MemoryUsage:   int64(dockerMetrics.Memory.Usage),
		MemoryLimit:   int64(dockerMetrics.Memory.Limit),
		MemoryPercent: dockerMetrics.Memory.MemoryPercent,
		PIDs:          int(dockerMetrics.PIDs.Current),
		Timestamp:     dockerMetrics.Timestamp,
	}

	// Convert network I/O metrics
	metrics.NetworkIO = &NetworkIOMetrics{
		RxBytes:   int64(dockerMetrics.Network.RxBytes),
		TxBytes:   int64(dockerMetrics.Network.TxBytes),
		RxPackets: int64(dockerMetrics.Network.RxPackets),
		TxPackets: int64(dockerMetrics.Network.TxPackets),
	}

	// Convert block I/O metrics
	metrics.BlockIO = &BlockIOMetrics{
		ReadBytes:  int64(dockerMetrics.BlockIO.ReadBytes),
		WriteBytes: int64(dockerMetrics.BlockIO.WriteBytes),
		ReadOps:    int64(dockerMetrics.BlockIO.ReadOperations),
		WriteOps:   int64(dockerMetrics.BlockIO.WriteOperations),
	}

	return metrics
}

// Background workers

// metricsCollectionWorker collects metrics from all active sessions
func (s *ContainerMonitoringService) metricsCollectionWorker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.RLock()
			for _, session := range s.activeMonitoring {
				if session.IsActive {
					// Process metrics updates from session channel
					select {
					case update := <-session.metricsChan:
						// Send to data processor
						s.dataProcessor.ProcessMetricsUpdate(update)
					default:
						// No updates available
					}

					// Process errors from session channel
					select {
					case err := <-session.errorChan:
						s.logger.WithError(err).WithField("container_id", session.ContainerID).Warn("Monitoring session error")
					default:
						// No errors
					}
				}
			}
			s.mu.RUnlock()
		}
	}
}

// dataProcessingWorker processes and stores metrics data
func (s *ContainerMonitoringService) dataProcessingWorker() {
	// Implementation would process metrics updates for storage and analysis
	s.logger.Debug("Data processing worker started")
}

// alertProcessingWorker processes alerts
func (s *ContainerMonitoringService) alertProcessingWorker() {
	// Implementation would handle alert processing and notifications
	s.logger.Debug("Alert processing worker started")
}

// ProcessMetricsUpdate processes a metrics update for storage
func (dp *MetricsDataProcessor) ProcessMetricsUpdate(update *ContainerMetricsUpdate) {
	if dp.metricsRepo == nil {
		return
	}

	// Convert to database model and store
	// This would be implemented based on the specific monitoring metrics model
	dp.logger.WithFields(logrus.Fields{
		"container_id": update.ContainerID,
		"timestamp":    update.Timestamp,
	}).Debug("Processing metrics update")
}

// Alert management methods

// SetThresholds sets alert thresholds for a container
func (am *AlertManager) SetThresholds(containerID string, thresholds *AlertThresholds) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.thresholds[containerID] = thresholds
}

// RemoveThresholds removes alert thresholds for a container
func (am *AlertManager) RemoveThresholds(containerID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.thresholds, containerID)
}

// CheckThresholds checks if metrics exceed thresholds and returns alerts
func (am *AlertManager) CheckThresholds(containerID string, metrics *ContainerMetrics) []*MetricsAlert {
	am.mu.RLock()
	thresholds, exists := am.thresholds[containerID]
	am.mu.RUnlock()

	if !exists {
		return nil
	}

	var alerts []*MetricsAlert

	// Check CPU threshold
	if metrics.CPUPercent > thresholds.CPUPercent {
		alerts = append(alerts, &MetricsAlert{
			Type:        "cpu_usage",
			Severity:    "warning",
			Message:     fmt.Sprintf("CPU usage exceeds threshold: %.2f%% > %.2f%%", metrics.CPUPercent, thresholds.CPUPercent),
			Threshold:   thresholds.CPUPercent,
			ActualValue: metrics.CPUPercent,
			Timestamp:   time.Now(),
		})
	}

	// Check memory threshold
	if metrics.MemoryPercent > thresholds.MemoryPercent {
		alerts = append(alerts, &MetricsAlert{
			Type:        "memory_usage",
			Severity:    "warning",
			Message:     fmt.Sprintf("Memory usage exceeds threshold: %.2f%% > %.2f%%", metrics.MemoryPercent, thresholds.MemoryPercent),
			Threshold:   thresholds.MemoryPercent,
			ActualValue: metrics.MemoryPercent,
			Timestamp:   time.Now(),
		})
	}

	return alerts
}

// Close gracefully shuts down the monitoring service
func (s *ContainerMonitoringService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all monitoring sessions
	for containerID := range s.activeMonitoring {
		if err := s.StopMonitoring(containerID); err != nil {
			s.logger.WithError(err).WithField("container_id", containerID).Warn("Failed to stop monitoring session during shutdown")
		}
	}

	s.logger.Info("Container monitoring service shut down")
	return nil
}