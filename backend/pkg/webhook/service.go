package webhook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service manages webhook delivery with queuing, retries, and rate limiting
type Service struct {
	config     *config.WebhookConfig
	webhook    Webhook
	queue      *WebhookQueue
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex

	// Metrics
	metrics *WebhookMetrics
}

// NewService creates a new webhook service
func NewService(config *config.WebhookConfig, logger *logrus.Logger) (*Service, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Validate configuration
	if !config.Enabled {
		return &Service{
			config:  config,
			logger:  logger,
			metrics: &WebhookMetrics{StatusCodes: make(map[int]int64)},
		}, nil
	}

	// Create webhook implementation
	webhookConfig := WebhookConfig{
		URL:             config.URL,
		Secret:          config.Secret,
		SignatureHeader: config.SignatureHeader,
		VerifySSL:       config.VerifySSL,
		Timeout:         time.Duration(config.Timeout) * time.Second,
		RetryAttempts:   config.RetryAttempts,
		RetryDelay:      time.Duration(config.RetryDelay) * time.Second,
	}

	webhook := NewHTTPWebhook(webhookConfig, logger)

	// Validate webhook configuration
	if err := webhook.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid webhook configuration: %w", err)
	}

	// Create rate limiter config
	var rateLimitConfig *RateLimitConfig
	if config.RateLimit > 0 {
		rateLimitConfig = &RateLimitConfig{
			RequestsPerMinute: config.RateLimit,
			Burst:             config.RateLimitBurst,
			Window:            time.Duration(config.RateLimitWindow) * time.Second,
		}
	}

	// Create queue
	queue := NewWebhookQueue(config.QueueSize, rateLimitConfig, logger)

	service := &Service{
		config:  config,
		webhook: webhook,
		queue:   queue,
		logger:  logger,
		metrics: &WebhookMetrics{StatusCodes: make(map[int]int64)},
	}

	return service, nil
}

// Start starts the webhook service workers
func (s *Service) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Webhook service is disabled")
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("webhook service is already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start worker goroutines
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.isRunning = true
	s.logger.WithField("worker_count", s.config.WorkerCount).Info("Webhook service started")

	return nil
}

// Stop stops the webhook service
func (s *Service) Stop() error {
	if !s.config.Enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("webhook service is not running")
	}

	s.logger.Info("Stopping webhook service...")

	// Cancel context to stop workers
	s.cancel()

	// Wait for workers to finish
	s.wg.Wait()

	s.isRunning = false
	s.logger.Info("Webhook service stopped")

	return nil
}

// SendWebhook sends a webhook immediately
func (s *Service) SendWebhook(ctx context.Context, event WebhookEvent, data map[string]interface{}) error {
	if !s.config.Enabled {
		s.logger.Debug("Webhook service disabled, skipping send")
		return nil
	}

	payload := &Payload{
		ID:        uuid.New().String(),
		Event:     string(event),
		Timestamp: time.Now(),
		Source:    "docker-auto",
		Data:      data,
		Priority:  PriorityNormal,
	}

	startTime := time.Now()
	err := s.webhook.Send(ctx, payload)
	duration := time.Since(startTime)

	s.updateMetrics(err == nil, duration, 0) // 0 for immediate send (no status code)

	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}

	return nil
}

// QueueWebhook queues a webhook for delivery
func (s *Service) QueueWebhook(event WebhookEvent, data map[string]interface{}, priority Priority) error {
	if !s.config.Enabled {
		s.logger.Debug("Webhook service disabled, skipping queue")
		return nil
	}

	payload := &Payload{
		ID:        uuid.New().String(),
		Event:     string(event),
		Timestamp: time.Now(),
		Source:    "docker-auto",
		Data:      data,
		Priority:  priority,
	}

	queuedWebhook := &QueuedWebhook{
		ID:          uuid.New().String(),
		Payload:     payload,
		URL:         s.config.URL,
		ScheduledAt: time.Now(),
		Attempts:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.queue.Enqueue(queuedWebhook); err != nil {
		return fmt.Errorf("failed to queue webhook: %w", err)
	}

	s.metrics.mu.Lock()
	s.metrics.TotalQueued++
	s.metrics.CurrentQueueSize = int64(s.queue.Size())
	s.metrics.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"webhook_id": queuedWebhook.ID,
		"payload_id": payload.ID,
		"event":      event,
		"priority":   priority,
	}).Debug("Webhook queued for delivery")

	return nil
}

// SendContainerEvent sends a container-related webhook
func (s *Service) SendContainerEvent(event WebhookEvent, containerID, containerName, image string, additionalData map[string]interface{}) error {
	data := map[string]interface{}{
		"container_id":   containerID,
		"container_name": containerName,
		"image":          image,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	// Merge additional data
	for key, value := range additionalData {
		data[key] = value
	}

	return s.QueueWebhook(event, data, PriorityNormal)
}

// SendSystemAlert sends a system alert webhook
func (s *Service) SendSystemAlert(alertType, message string, severity string, additionalData map[string]interface{}) error {
	data := map[string]interface{}{
		"alert_type": alertType,
		"message":    message,
		"severity":   severity,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	// Merge additional data
	for key, value := range additionalData {
		data[key] = value
	}

	// Determine priority based on severity
	priority := PriorityNormal
	switch severity {
	case "critical":
		priority = PriorityUrgent
	case "error":
		priority = PriorityHigh
	case "warning":
		priority = PriorityNormal
	default:
		priority = PriorityLow
	}

	return s.QueueWebhook(EventSystemAlert, data, priority)
}

// worker processes webhooks from the queue
func (s *Service) worker(workerID int) {
	defer s.wg.Done()

	logger := s.logger.WithField("worker_id", workerID)
	logger.Debug("Webhook worker started")

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("Webhook worker stopping")
			return
		default:
			// Try to dequeue a webhook
			queuedWebhook, err := s.queue.Dequeue(s.ctx)
			if err != nil {
				if err == ErrWebhookQueueEmpty {
					// No webhooks, wait a bit
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if err == context.Canceled {
					return
				}
				logger.WithError(err).Error("Failed to dequeue webhook")
				continue
			}

			// Process the webhook
			s.processQueuedWebhook(logger, queuedWebhook)
		}
	}
}

// processQueuedWebhook processes a single queued webhook
func (s *Service) processQueuedWebhook(logger *logrus.Entry, queuedWebhook *QueuedWebhook) {
	startTime := time.Now()

	logger = logger.WithFields(logrus.Fields{
		"webhook_id": queuedWebhook.ID,
		"payload_id": queuedWebhook.Payload.ID,
		"event":      queuedWebhook.Payload.Event,
		"attempts":   queuedWebhook.Attempts,
		"url":        queuedWebhook.URL,
	})

	logger.Debug("Processing queued webhook")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, s.webhook.(*HTTPWebhook).config.Timeout)
	defer cancel()

	// Increment attempt count
	queuedWebhook.Attempts++
	queuedWebhook.UpdatedAt = time.Now()
	queuedWebhook.Payload.RetryCount = queuedWebhook.Attempts - 1

	// Get delivery info (includes actual sending)
	result, err := s.webhook.(*HTTPWebhook).GetDeliveryInfo(ctx, queuedWebhook.Payload)
	duration := time.Since(startTime)

	if err != nil {
		logger.WithError(err).Warn("Failed to get delivery info for queued webhook")

		// Update last error
		queuedWebhook.LastError = err.Error()

		// Check if we should retry
		if queuedWebhook.Attempts < s.config.RetryAttempts {
			// Schedule retry with exponential backoff
			retryDelay := time.Duration(s.config.RetryDelay) * time.Second
			backoffMultiplier := time.Duration(queuedWebhook.Attempts)
			queuedWebhook.ScheduledAt = time.Now().Add(retryDelay * backoffMultiplier)

			if err := s.queue.Enqueue(queuedWebhook); err != nil {
				logger.WithError(err).Error("Failed to requeue webhook for retry")
			} else {
				logger.WithField("retry_at", queuedWebhook.ScheduledAt).Info("Webhook requeued for retry")
			}
		} else {
			logger.Error("Webhook failed after maximum retry attempts")
		}

		s.updateMetrics(false, duration, 0)
		return
	}

	// Process result
	if result.Success {
		logger.WithFields(logrus.Fields{
			"status_code": result.StatusCode,
			"duration_ms": duration.Milliseconds(),
		}).Info("Queued webhook delivered successfully")
	} else {
		logger.WithFields(logrus.Fields{
			"status_code": result.StatusCode,
			"error":       result.Error,
			"duration_ms": duration.Milliseconds(),
		}).Warn("Queued webhook delivery failed")

		// Update last error
		queuedWebhook.LastError = result.Error

		// Check if we should retry based on status code
		shouldRetry := s.shouldRetryStatusCode(result.StatusCode)
		if shouldRetry && queuedWebhook.Attempts < s.config.RetryAttempts {
			// Schedule retry
			retryDelay := time.Duration(s.config.RetryDelay) * time.Second
			backoffMultiplier := time.Duration(queuedWebhook.Attempts)
			queuedWebhook.ScheduledAt = time.Now().Add(retryDelay * backoffMultiplier)

			if err := s.queue.Enqueue(queuedWebhook); err != nil {
				logger.WithError(err).Error("Failed to requeue webhook for retry")
			} else {
				logger.WithField("retry_at", queuedWebhook.ScheduledAt).Info("Webhook requeued for retry")
			}
		} else {
			logger.Error("Webhook failed and will not be retried")
		}
	}

	s.updateMetrics(result.Success, duration, result.StatusCode)

	// Update queue size metric
	s.metrics.mu.Lock()
	s.metrics.CurrentQueueSize = int64(s.queue.Size())
	s.metrics.mu.Unlock()
}

// shouldRetryStatusCode determines if a status code warrants a retry
func (s *Service) shouldRetryStatusCode(statusCode int) bool {
	// Retry on server errors (5xx) and some client errors
	if statusCode >= 500 {
		return true
	}

	// Retry on specific client errors
	switch statusCode {
	case 408, 429: // Request Timeout, Too Many Requests
		return true
	default:
		return false
	}
}

// updateMetrics updates service metrics
func (s *Service) updateMetrics(success bool, duration time.Duration, statusCode int) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	now := time.Now()

	if success {
		s.metrics.TotalSent++
		s.metrics.LastSentAt = &now
	} else {
		s.metrics.TotalFailed++
		s.metrics.LastFailureAt = &now
	}

	// Update status code distribution
	if statusCode > 0 {
		s.metrics.StatusCodes[statusCode]++
	}

	// Update average response time (simple moving average)
	if s.metrics.AverageResponseTime == 0 {
		s.metrics.AverageResponseTime = duration
	} else {
		s.metrics.AverageResponseTime = (s.metrics.AverageResponseTime + duration) / 2
	}

	// Update success rate
	total := s.metrics.TotalSent + s.metrics.TotalFailed
	if total > 0 {
		s.metrics.SuccessRate = float64(s.metrics.TotalSent) / float64(total) * 100
	}

	// TODO: Update percentiles (would require storing response times)
}

// GetMetrics returns current service metrics
func (s *Service) GetMetrics() *WebhookMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	// Return a copy
	metrics := *s.metrics

	// Deep copy status codes map
	metrics.StatusCodes = make(map[int]int64)
	for k, v := range s.metrics.StatusCodes {
		metrics.StatusCodes[k] = v
	}

	return &metrics
}

// GetQueueStats returns current queue statistics
func (s *Service) GetQueueStats() *WebhookQueueStats {
	if s.queue == nil {
		return nil
	}
	return s.queue.GetStats()
}

// IsEnabled returns whether the webhook service is enabled
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// TestConnection tests the webhook endpoint
func (s *Service) TestConnection(ctx context.Context) error {
	if !s.config.Enabled {
		return fmt.Errorf("webhook service is disabled")
	}

	return s.webhook.(*HTTPWebhook).TestConnection(ctx)
}

// Add mutex for metrics
func init() {
	// Initialize metrics mutex in the WebhookMetrics struct
}

// Add mutex to WebhookMetrics
type WebhookMetricsWithMutex struct {
	WebhookMetrics
	mu sync.RWMutex
}