package email

import (
	"context"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/pkg/email/providers"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service manages email sending with multiple providers, templates, and queuing
type Service struct {
	config           *config.EmailConfig
	provider         providers.EmailProvider
	templateManager  *TemplateManager
	queue            *Queue
	logger           *logrus.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	isRunning        bool
	mu               sync.RWMutex

	// Metrics
	metrics *ServiceMetrics
}

// ServiceMetrics tracks email service metrics
type ServiceMetrics struct {
	TotalSent       int64     `json:"total_sent"`
	TotalFailed     int64     `json:"total_failed"`
	TotalQueued     int64     `json:"total_queued"`
	CurrentQueueSize int64    `json:"current_queue_size"`
	LastSentAt      *time.Time `json:"last_sent_at,omitempty"`
	LastFailureAt   *time.Time `json:"last_failure_at,omitempty"`
	AverageDeliveryTime time.Duration `json:"average_delivery_time"`
	SuccessRate     float64   `json:"success_rate"`
	mu              sync.RWMutex
}

// NewService creates a new email service
func NewService(config *config.EmailConfig, logger *logrus.Logger) (*Service, error) {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Validate configuration
	if !config.Enabled {
		return &Service{
			config:  config,
			logger:  logger,
			metrics: &ServiceMetrics{},
		}, nil
	}

	// Create email provider
	provider, err := createProvider(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create email provider: %w", err)
	}

	// Create template manager
	templateManager := NewTemplateManager(config.TemplateDir, config.DefaultLocale, logger)
	if err := templateManager.LoadTemplates(); err != nil {
		logger.WithError(err).Warn("Failed to load email templates, using defaults")
	}

	// Create queue
	queue := NewQueue(config.QueueSize, logger)

	service := &Service{
		config:          config,
		provider:        provider,
		templateManager: templateManager,
		queue:           queue,
		logger:          logger,
		metrics:         &ServiceMetrics{},
	}

	return service, nil
}

// Start starts the email service workers
func (s *Service) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Email service is disabled")
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("email service is already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start worker goroutines
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	s.isRunning = true
	s.logger.WithField("worker_count", s.config.WorkerCount).Info("Email service started")

	return nil
}

// Stop stops the email service
func (s *Service) Stop() error {
	if !s.config.Enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("email service is not running")
	}

	s.logger.Info("Stopping email service...")

	// Cancel context to stop workers
	s.cancel()

	// Wait for workers to finish
	s.wg.Wait()

	s.isRunning = false
	s.logger.Info("Email service stopped")

	return nil
}

// SendEmail sends an email immediately
func (s *Service) SendEmail(ctx context.Context, message *providers.Message) error {
	if !s.config.Enabled {
		s.logger.Debug("Email service disabled, skipping send")
		return nil
	}

	startTime := time.Now()

	err := s.provider.Send(ctx, message)
	if err != nil {
		s.updateMetrics(false, time.Since(startTime))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.updateMetrics(true, time.Since(startTime))
	return nil
}

// QueueEmail queues an email for sending
func (s *Service) QueueEmail(message *providers.Message) error {
	if !s.config.Enabled {
		s.logger.Debug("Email service disabled, skipping queue")
		return nil
	}

	queuedMessage := &QueuedMessage{
		ID:          uuid.New().String(),
		Message:     message,
		ScheduledAt: time.Now(),
		Attempts:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.queue.Enqueue(queuedMessage); err != nil {
		return fmt.Errorf("failed to queue email: %w", err)
	}

	s.metrics.mu.Lock()
	s.metrics.TotalQueued++
	s.metrics.CurrentQueueSize = int64(s.queue.Size())
	s.metrics.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"message_id": queuedMessage.ID,
		"to":         message.To,
		"subject":    message.Subject,
	}).Debug("Email queued for sending")

	return nil
}

// SendTemplatedEmail sends an email using a template
func (s *Service) SendTemplatedEmail(ctx context.Context, templateName string, data *TemplateData, locale string) error {
	if !s.config.Enabled {
		s.logger.Debug("Email service disabled, skipping templated send")
		return nil
	}

	// Validate template data
	if err := data.Validate(); err != nil {
		return fmt.Errorf("invalid template data: %w", err)
	}

	// Render template
	rendered, err := s.templateManager.RenderTemplate(templateName, data, locale)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Create message
	message := &Message{
		To:       []string{data.RecipientEmail},
		From:     data.SenderEmail,
		Subject:  rendered.Subject,
		TextBody: rendered.TextBody,
		HTMLBody: rendered.HTMLBody,
	}

	return s.SendEmail(ctx, message)
}

// QueueTemplatedEmail queues a templated email for sending
func (s *Service) QueueTemplatedEmail(templateName string, data *TemplateData, locale string) error {
	if !s.config.Enabled {
		s.logger.Debug("Email service disabled, skipping templated queue")
		return nil
	}

	// Validate template data
	if err := data.Validate(); err != nil {
		return fmt.Errorf("invalid template data: %w", err)
	}

	// Render template
	rendered, err := s.templateManager.RenderTemplate(templateName, data, locale)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Create message
	message := &Message{
		To:       []string{data.RecipientEmail},
		From:     data.SenderEmail,
		Subject:  rendered.Subject,
		TextBody: rendered.TextBody,
		HTMLBody: rendered.HTMLBody,
	}

	return s.QueueEmail(message)
}

// worker processes emails from the queue
func (s *Service) worker(workerID int) {
	defer s.wg.Done()

	logger := s.logger.WithField("worker_id", workerID)
	logger.Debug("Email worker started")

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("Email worker stopping")
			return
		default:
			// Try to dequeue a message
			queuedMessage, err := s.queue.Dequeue(s.ctx)
			if err != nil {
				if err == ErrQueueEmpty {
					// No messages, wait a bit
					time.Sleep(100 * time.Millisecond)
					continue
				}
				logger.WithError(err).Error("Failed to dequeue message")
				continue
			}

			// Process the message
			s.processQueuedMessage(logger, queuedMessage)
		}
	}
}

// processQueuedMessage processes a single queued message
func (s *Service) processQueuedMessage(logger *logrus.Entry, queuedMessage *providers.QueuedMessage) {
	startTime := time.Now()

	logger = logger.WithFields(logrus.Fields{
		"message_id": queuedMessage.ID,
		"to":         queuedMessage.Message.To,
		"subject":    queuedMessage.Message.Subject,
		"attempts":   queuedMessage.Attempts,
	})

	logger.Debug("Processing queued email")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Increment attempt count
	queuedMessage.Attempts++
	queuedMessage.UpdatedAt = time.Now()

	// Try to send the email
	err := s.provider.Send(ctx, queuedMessage.Message)
	if err != nil {
		logger.WithError(err).Warn("Failed to send queued email")

		// Update last error
		queuedMessage.LastError = err.Error()

		// Check if we should retry
		if queuedMessage.Attempts < s.config.RetryAttempts {
			// Schedule retry
			retryDelay := time.Duration(s.config.RetryDelay) * time.Second
			queuedMessage.ScheduledAt = time.Now().Add(retryDelay)

			if err := s.queue.Enqueue(queuedMessage); err != nil {
				logger.WithError(err).Error("Failed to requeue message for retry")
			} else {
				logger.WithField("retry_at", queuedMessage.ScheduledAt).Info("Email requeued for retry")
			}
		} else {
			logger.Error("Email failed after maximum retry attempts")
		}

		s.updateMetrics(false, time.Since(startTime))
		return
	}

	logger.Info("Queued email sent successfully")
	s.updateMetrics(true, time.Since(startTime))

	// Update queue size metric
	s.metrics.mu.Lock()
	s.metrics.CurrentQueueSize = int64(s.queue.Size())
	s.metrics.mu.Unlock()
}

// updateMetrics updates service metrics
func (s *Service) updateMetrics(success bool, duration time.Duration) {
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

	// Update average delivery time (simple moving average)
	if s.metrics.AverageDeliveryTime == 0 {
		s.metrics.AverageDeliveryTime = duration
	} else {
		s.metrics.AverageDeliveryTime = (s.metrics.AverageDeliveryTime + duration) / 2
	}

	// Update success rate
	total := s.metrics.TotalSent + s.metrics.TotalFailed
	if total > 0 {
		s.metrics.SuccessRate = float64(s.metrics.TotalSent) / float64(total) * 100
	}
}

// GetMetrics returns current service metrics
func (s *Service) GetMetrics() *ServiceMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	// Return a copy
	metrics := *s.metrics
	return &metrics
}

// IsEnabled returns whether the email service is enabled
func (s *Service) IsEnabled() bool {
	return s.config.Enabled
}

// GetQueueSize returns the current queue size
func (s *Service) GetQueueSize() int {
	if s.queue == nil {
		return 0
	}
	return s.queue.Size()
}

// TestConnection tests the email provider connection
func (s *Service) TestConnection(ctx context.Context) error {
	if !s.config.Enabled {
		return fmt.Errorf("email service is disabled")
	}

	// Test provider-specific connection
	if tester, ok := s.provider.(interface{ TestConnection(context.Context) error }); ok {
		return tester.TestConnection(ctx)
	}

	// Fallback: try to send a test email to a non-existent address
	testMessage := &Message{
		To:       []string{"test@example.invalid"},
		From:     s.config.From,
		Subject:  "Connection Test",
		TextBody: "This is a connection test.",
	}

	err := s.provider.Send(ctx, testMessage)
	// We expect this to fail due to invalid email, but it validates provider setup
	if err != nil && (err.Error() == "invalid email address" || err.Error() == "recipient not found") {
		return nil
	}

	return err
}

// createProvider creates an email provider based on configuration
func createProvider(config *config.EmailConfig, logger *logrus.Logger) (EmailProvider, error) {
	switch config.Provider {
	case "smtp":
		smtpConfig := providers.SMTPConfig{
			Host:            config.SMTPHost,
			Port:            config.SMTPPort,
			Username:        config.Username,
			Password:        config.Password,
			From:            config.From,
			UseTLS:          config.SMTPPort == 465,
			UseStartTLS:     config.SMTPPort == 587,
			ConnectTimeout:  30 * time.Second,
			SendTimeout:     60 * time.Second,
		}
		return providers.NewSMTPProvider(smtpConfig, logger), nil

	case "ses":
		sesConfig := providers.SESConfig{
			Region:    config.AWSRegion,
			AccessKey: config.AWSAccessKey,
			SecretKey: config.AWSSecretKey,
			From:      config.From,
		}
		return providers.NewSESProvider(sesConfig, logger)

	case "sendgrid":
		sendgridConfig := providers.SendGridConfig{
			APIKey:   config.SendGridAPIKey,
			From:     config.From,
			FromName: "Docker Auto",
		}
		return providers.NewSendGridProvider(sendgridConfig, logger), nil

	default:
		return nil, fmt.Errorf("unsupported email provider: %s", config.Provider)
	}
}