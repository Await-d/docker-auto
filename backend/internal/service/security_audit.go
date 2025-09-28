package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/security"

	"github.com/sirupsen/logrus"
)

// SecurityAuditService provides comprehensive security audit capabilities
type SecurityAuditService struct {
	// auditRepo       repository.AuditLogRepository  // TODO: Implement AuditLogRepository
	containerRepo   repository.ContainerRepository
	userRepo        repository.UserRepository
	dockerClient    *docker.DockerClient
	// securityScanner *security.SecurityScanner  // TODO: Implement security.SecurityScanner
	complianceChecker *ComplianceChecker
	riskAnalyzer    *RiskAnalyzer
	config          *config.Config
	logger          *logrus.Entry

	// Real-time monitoring
	auditQueue      chan *AuditEvent
	alertChannels   map[string]chan *SecurityAlert
	mu              sync.RWMutex

	// Background workers
	scanScheduler   *SecurityScanScheduler
	alertManager    *SecurityAlertManager
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	EventType    AuditEventType         `json:"event_type"`
	Severity     SecuritySeverity       `json:"severity"`
	Source       string                 `json:"source"`
	UserID       *int64                 `json:"user_id,omitempty"`
	ContainerID  *string                `json:"container_id,omitempty"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	Details      map[string]interface{} `json:"details"`
	Result       AuditResult            `json:"result"`
	RemoteIP     string                 `json:"remote_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	AlertType    SecurityAlertType      `json:"alert_type"`
	Severity     SecuritySeverity       `json:"severity"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Source       string                 `json:"source"`
	ContainerID  *string                `json:"container_id,omitempty"`
	UserID       *int64                 `json:"user_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	Status       AlertStatus            `json:"status"`
	Acknowledged bool                   `json:"acknowledged"`
	AcknowledgedBy *int64               `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time           `json:"acknowledged_at,omitempty"`
}

// ComplianceCheck represents a compliance check result
type ComplianceCheck struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Standard      ComplianceStandard     `json:"standard"`
	Category      string                 `json:"category"`
	Severity      SecuritySeverity       `json:"severity"`
	Status        ComplianceStatus       `json:"status"`
	Details       string                 `json:"details"`
	Remediation   string                 `json:"remediation"`
	LastChecked   time.Time              `json:"last_checked"`
	ContainerID   *string                `json:"container_id,omitempty"`
	Evidence      map[string]interface{} `json:"evidence"`
}

// RiskAssessment represents a security risk assessment
type RiskAssessment struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	ContainerID      string                 `json:"container_id"`
	ContainerName    string                 `json:"container_name"`
	OverallRiskScore float64                `json:"overall_risk_score"`
	RiskLevel        RiskLevel              `json:"risk_level"`
	Factors          []RiskFactor           `json:"factors"`
	Recommendations  []SecurityRecommendation `json:"recommendations"`
	LastUpdated      time.Time              `json:"last_updated"`
}

// Enums and constants

type AuditEventType string

const (
	EventTypeAuthentication    AuditEventType = "authentication"
	EventTypeAuthorization     AuditEventType = "authorization"
	EventTypeContainerOperation AuditEventType = "container_operation"
	EventTypeImageOperation    AuditEventType = "image_operation"
	EventTypeSystemAccess      AuditEventType = "system_access"
	EventTypeSecurityViolation AuditEventType = "security_violation"
	EventTypeDataAccess        AuditEventType = "data_access"
	EventTypeConfigurationChange AuditEventType = "configuration_change"
)

type SecuritySeverity string

const (
	SeverityLow      SecuritySeverity = "low"
	SeverityMedium   SecuritySeverity = "medium"
	SeverityHigh     SecuritySeverity = "high"
	SeverityCritical SecuritySeverity = "critical"
)

type AuditResult string

const (
	ResultSuccess    AuditResult = "success"
	ResultFailure    AuditResult = "failure"
	ResultSuspicious AuditResult = "suspicious"
	ResultDenied     AuditResult = "denied"
)

type SecurityAlertType string

const (
	AlertTypeAnomaly           SecurityAlertType = "anomaly"
	AlertTypeVulnerability     SecurityAlertType = "vulnerability"
	AlertTypeCompliance        SecurityAlertType = "compliance"
	AlertTypeUnauthorizedAccess SecurityAlertType = "unauthorized_access"
	AlertTypeSuspiciousActivity SecurityAlertType = "suspicious_activity"
	AlertTypeResourceAbuse     SecurityAlertType = "resource_abuse"
)

type AlertStatus string

const (
	AlertStatusOpen       AlertStatus = "open"
	AlertStatusInProgress AlertStatus = "in_progress"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusFalsePositive AlertStatus = "false_positive"
)

type ComplianceStandard string

const (
	StandardCIS         ComplianceStandard = "cis"
	StandardNIST        ComplianceStandard = "nist"
	StandardSOC2        ComplianceStandard = "soc2"
	StandardISO27001    ComplianceStandard = "iso27001"
	StandardPCIDSS      ComplianceStandard = "pci_dss"
	StandardHIPAA       ComplianceStandard = "hipaa"
)

type ComplianceStatus string

const (
	ComplianceStatusPass    ComplianceStatus = "pass"
	ComplianceStatusFail    ComplianceStatus = "fail"
	ComplianceStatusWarning ComplianceStatus = "warning"
	ComplianceStatusSkipped ComplianceStatus = "skipped"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// Supporting structures

type RiskFactor struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Category    string  `json:"category"`
}

type SecurityRecommendation struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Priority    SecuritySeverity `json:"priority"`
	Actions     []string         `json:"actions"`
	Resources   []string         `json:"resources"`
}

type ComplianceChecker struct {
	checks  map[ComplianceStandard][]*ComplianceRule
	logger  *logrus.Entry
}

type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Standard    ComplianceStandard     `json:"standard"`
	Category    string                 `json:"category"`
	Severity    SecuritySeverity       `json:"severity"`
	CheckFunc   func(context.Context, *docker.ContainerInfo) *ComplianceCheck `json:"-"`
}

type RiskAnalyzer struct {
	riskModels map[string]*RiskModel
	logger     *logrus.Entry
}

type RiskModel struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Factors     []RiskFactorDefinition `json:"factors"`
	Weights     map[string]float64     `json:"weights"`
	Thresholds  map[RiskLevel]float64  `json:"thresholds"`
}

type RiskFactorDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Weight      float64                `json:"weight"`
	Evaluator   func(context.Context, *docker.ContainerInfo) float64 `json:"-"`
}

type SecurityScanScheduler struct {
	scanQueue   chan *ScanRequest
	config      *config.Config
	logger      *logrus.Entry
}

type ScanRequest struct {
	ContainerID string    `json:"container_id"`
	ScanType    ScanType  `json:"scan_type"`
	Priority    int       `json:"priority"`
	Scheduled   time.Time `json:"scheduled"`
}

type ScanType string

const (
	ScanTypeVulnerability ScanType = "vulnerability"
	ScanTypeCompliance    ScanType = "compliance"
	ScanTypeRisk          ScanType = "risk"
	ScanTypeFull          ScanType = "full"
)

type SecurityAlertManager struct {
	alertChannels map[string]chan *SecurityAlert
	config        *config.Config
	logger        *logrus.Entry
	mu            sync.RWMutex
}

// NewSecurityAuditService creates a new security audit service
func NewSecurityAuditService(
	// auditRepo repository.AuditLogRepository,  // TODO: Implement AuditLogRepository
	containerRepo repository.ContainerRepository,
	userRepo repository.UserRepository,
	dockerClient *docker.DockerClient,
	config *config.Config,
) *SecurityAuditService {
	logger := logrus.WithField("component", "security_audit")

	// Initialize security scanner
	securityScanner := security.NewSecurityScanner(dockerClient, &security.ScannerConfig{
		VulnerabilityDBPath: config.Security.VulnerabilityDBPath,
		ScanTimeout:         5 * time.Minute,
		MaxConcurrentScans:  3,
		EnabledScanners:     []string{"trivy", "clair", "grype"},
	}, logger.Logger)

	// Initialize compliance checker
	complianceChecker := NewComplianceChecker(logger)

	// Initialize risk analyzer
	riskAnalyzer := NewRiskAnalyzer(logger)

	// Initialize scan scheduler
	scanScheduler := &SecurityScanScheduler{
		scanQueue: make(chan *ScanRequest, 1000),
		config:    config,
		logger:    logger.WithField("subcomponent", "scan_scheduler"),
	}

	// Initialize alert manager
	alertManager := &SecurityAlertManager{
		alertChannels: make(map[string]chan *SecurityAlert),
		config:        config,
		logger:        logger.WithField("subcomponent", "alert_manager"),
	}

	service := &SecurityAuditService{
		// auditRepo:         auditRepo,  // TODO: Implement AuditLogRepository
		containerRepo:     containerRepo,
		userRepo:          userRepo,
		dockerClient:      dockerClient,
		// securityScanner:   securityScanner,  // TODO: Implement security.SecurityScanner
		complianceChecker: complianceChecker,
		riskAnalyzer:      riskAnalyzer,
		config:            config,
		logger:            logger,
		auditQueue:        make(chan *AuditEvent, 10000),
		alertChannels:     make(map[string]chan *SecurityAlert),
		scanScheduler:     scanScheduler,
		alertManager:      alertManager,
	}

	// Start background workers
	go service.auditEventProcessor()
	go service.securityScanWorker()
	go service.complianceMonitorWorker()
	go service.riskAssessmentWorker()

	logger.Info("Security audit service initialized")
	return service
}

// LogAuditEvent logs a security audit event
func (s *SecurityAuditService) LogAuditEvent(ctx context.Context, event *AuditEvent) error {
	// Enrich event with additional context
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), generateRandomID())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Add to audit queue for asynchronous processing
	select {
	case s.auditQueue <- event:
		// Successfully queued
	default:
		// Queue full, log immediately (synchronous fallback)
		return s.persistAuditEvent(event)
	}

	// Check for security violations and generate alerts
	s.analyzeEventForViolations(event)

	return nil
}

// PerformComplianceCheck performs a compliance check on a container
func (s *SecurityAuditService) PerformComplianceCheck(ctx context.Context, containerID string, standards []ComplianceStandard) ([]*ComplianceCheck, error) {
	container, err := s.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	var allChecks []*ComplianceCheck

	for _, standard := range standards {
		checks, err := s.complianceChecker.RunChecks(ctx, standard, container)
		if err != nil {
			s.logger.WithError(err).WithField("standard", standard).Warn("Compliance check failed")
			continue
		}
		allChecks = append(allChecks, checks...)
	}

	// Generate alerts for failed compliance checks
	for _, check := range allChecks {
		if check.Status == ComplianceStatusFail && check.Severity == SeverityCritical {
			alert := &SecurityAlert{
				ID:          fmt.Sprintf("compliance_%s_%d", check.ID, time.Now().UnixNano()),
				Timestamp:   time.Now(),
				AlertType:   AlertTypeCompliance,
				Severity:    check.Severity,
				Title:       fmt.Sprintf("Compliance Violation: %s", check.Name),
				Description: check.Details,
				Source:      "compliance_checker",
				ContainerID: &containerID,
				Metadata: map[string]interface{}{
					"standard":     check.Standard,
					"check_id":     check.ID,
					"remediation":  check.Remediation,
				},
				Status: AlertStatusOpen,
			}
			s.GenerateAlert(alert)
		}
	}

	return allChecks, nil
}

// PerformRiskAssessment performs a security risk assessment on a container
func (s *SecurityAuditService) PerformRiskAssessment(ctx context.Context, containerID string) (*RiskAssessment, error) {
	container, err := s.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	assessment := s.riskAnalyzer.AssessRisk(ctx, container)

	// Generate alerts for high-risk containers
	if assessment.RiskLevel == RiskLevelHigh || assessment.RiskLevel == RiskLevelCritical {
		alert := &SecurityAlert{
			ID:          fmt.Sprintf("risk_%s_%d", containerID, time.Now().UnixNano()),
			Timestamp:   time.Now(),
			AlertType:   AlertTypeAnomaly,
			Severity:    SecuritySeverity(assessment.RiskLevel),
			Title:       fmt.Sprintf("High Risk Container Detected: %s", container.Name),
			Description: fmt.Sprintf("Container has a risk score of %.2f", assessment.OverallRiskScore),
			Source:      "risk_analyzer",
			ContainerID: &containerID,
			Metadata: map[string]interface{}{
				"risk_score":      assessment.OverallRiskScore,
				"risk_level":      assessment.RiskLevel,
				"factors_count":   len(assessment.Factors),
				"recommendations": len(assessment.Recommendations),
			},
			Status: AlertStatusOpen,
		}
		s.GenerateAlert(alert)
	}

	return assessment, nil
}

// GenerateAlert generates a security alert
func (s *SecurityAuditService) GenerateAlert(alert *SecurityAlert) {
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert_%d_%d", time.Now().UnixNano(), generateRandomID())
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Send to all registered alert channels
	s.mu.RLock()
	for _, channel := range s.alertChannels {
		select {
		case channel <- alert:
			// Alert sent
		default:
			// Channel full, skip
		}
	}
	s.mu.RUnlock()

	// Persist alert
	s.persistAlert(alert)

	s.logger.WithFields(logrus.Fields{
		"alert_id":   alert.ID,
		"alert_type": alert.AlertType,
		"severity":   alert.Severity,
		"title":      alert.Title,
	}).Info("Security alert generated")
}

// SubscribeToAlerts subscribes to security alerts
func (s *SecurityAuditService) SubscribeToAlerts(subscriberID string) <-chan *SecurityAlert {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel := make(chan *SecurityAlert, 100)
	s.alertChannels[subscriberID] = channel
	return channel
}

// UnsubscribeFromAlerts unsubscribes from security alerts
func (s *SecurityAuditService) UnsubscribeFromAlerts(subscriberID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if channel, exists := s.alertChannels[subscriberID]; exists {
		close(channel)
		delete(s.alertChannels, subscriberID)
	}
}

// GetAuditEvents retrieves audit events with filtering
func (s *SecurityAuditService) GetAuditEvents(ctx context.Context, filter *AuditEventFilter) ([]*AuditEvent, int64, error) {
	// This would query the audit repository with filters
	// For now, return mock data
	return []*AuditEvent{}, 0, nil
}

// GetSecurityAlerts retrieves security alerts with filtering
func (s *SecurityAuditService) GetSecurityAlerts(ctx context.Context, filter *SecurityAlertFilter) ([]*SecurityAlert, int64, error) {
	// This would query alerts from storage
	// For now, return mock data
	return []*SecurityAlert{}, 0, nil
}

// AcknowledgeAlert acknowledges a security alert
func (s *SecurityAuditService) AcknowledgeAlert(ctx context.Context, alertID string, userID int64) error {
	// This would update the alert status in storage
	s.logger.WithFields(logrus.Fields{
		"alert_id": alertID,
		"user_id":  userID,
	}).Info("Security alert acknowledged")
	return nil
}

// Background workers

// auditEventProcessor processes audit events from the queue
func (s *SecurityAuditService) auditEventProcessor() {
	for event := range s.auditQueue {
		if err := s.persistAuditEvent(event); err != nil {
			s.logger.WithError(err).WithField("event_id", event.ID).Error("Failed to persist audit event")
		}
	}
}

// securityScanWorker processes security scan requests
func (s *SecurityAuditService) securityScanWorker() {
	for req := range s.scanScheduler.scanQueue {
		s.processScanRequest(req)
	}
}

// complianceMonitorWorker performs periodic compliance monitoring
func (s *SecurityAuditService) complianceMonitorWorker() {
	ticker := time.NewTicker(24 * time.Hour) // Daily compliance checks
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performScheduledComplianceChecks()
		}
	}
}

// riskAssessmentWorker performs periodic risk assessments
func (s *SecurityAuditService) riskAssessmentWorker() {
	ticker := time.NewTicker(12 * time.Hour) // Twice daily risk assessments
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performScheduledRiskAssessments()
		}
	}
}

// Helper methods

// persistAuditEvent persists an audit event to storage
func (s *SecurityAuditService) persistAuditEvent(event *AuditEvent) error {
	// Convert to model and save to repository
	auditLog := &model.AuditLog{
		EventID:     event.ID,
		EventType:   string(event.EventType),
		Severity:    string(event.Severity),
		Source:      event.Source,
		UserID:      event.UserID,
		Action:      event.Action,
		Resource:    event.Resource,
		Details:     marshalToJSON(event.Details),
		Result:      string(event.Result),
		RemoteIP:    event.RemoteIP,
		UserAgent:   event.UserAgent,
		SessionID:   event.SessionID,
		Timestamp:   event.Timestamp,
	}

	return s.auditRepo.Create(context.Background(), auditLog)
}

// persistAlert persists a security alert to storage
func (s *SecurityAuditService) persistAlert(alert *SecurityAlert) {
	// This would save the alert to a database
	s.logger.WithField("alert_id", alert.ID).Debug("Alert persisted")
}

// analyzeEventForViolations analyzes an audit event for security violations
func (s *SecurityAuditService) analyzeEventForViolations(event *AuditEvent) {
	// Check for suspicious patterns
	if s.isSuspiciousActivity(event) {
		alert := &SecurityAlert{
			ID:          fmt.Sprintf("violation_%s_%d", event.ID, time.Now().UnixNano()),
			Timestamp:   time.Now(),
			AlertType:   AlertTypeSuspiciousActivity,
			Severity:    event.Severity,
			Title:       "Suspicious Activity Detected",
			Description: fmt.Sprintf("Suspicious %s activity detected", event.EventType),
			Source:      "violation_analyzer",
			UserID:      event.UserID,
			ContainerID: event.ContainerID,
			Metadata: map[string]interface{}{
				"original_event_id": event.ID,
				"event_type":        event.EventType,
				"action":            event.Action,
			},
			Status: AlertStatusOpen,
		}
		s.GenerateAlert(alert)
	}
}

// isSuspiciousActivity determines if an event represents suspicious activity
func (s *SecurityAuditService) isSuspiciousActivity(event *AuditEvent) bool {
	// Simple heuristics - could be enhanced with ML
	switch event.EventType {
	case EventTypeAuthentication:
		return event.Result == ResultFailure
	case EventTypeContainerOperation:
		return event.Result == ResultDenied
	case EventTypeSystemAccess:
		return event.Severity == SeverityCritical
	}
	return false
}

// processScanRequest processes a security scan request
func (s *SecurityAuditService) processScanRequest(req *ScanRequest) {
	ctx := context.Background()

	switch req.ScanType {
	case ScanTypeVulnerability:
		s.performVulnerabilityScan(ctx, req.ContainerID)
	case ScanTypeCompliance:
		s.performComplianceScan(ctx, req.ContainerID)
	case ScanTypeRisk:
		s.performRiskScan(ctx, req.ContainerID)
	case ScanTypeFull:
		s.performFullScan(ctx, req.ContainerID)
	}
}

// performVulnerabilityScan performs a vulnerability scan
func (s *SecurityAuditService) performVulnerabilityScan(ctx context.Context, containerID string) {
	result, err := s.securityScanner.ScanContainer(ctx, containerID)
	if err != nil {
		s.logger.WithError(err).WithField("container_id", containerID).Error("Vulnerability scan failed")
		return
	}

	// Generate alerts for critical vulnerabilities
	for _, vuln := range result.Vulnerabilities {
		if vuln.Severity == "CRITICAL" {
			alert := &SecurityAlert{
				ID:          fmt.Sprintf("vuln_%s_%s", containerID, vuln.CVE),
				Timestamp:   time.Now(),
				AlertType:   AlertTypeVulnerability,
				Severity:    SeverityCritical,
				Title:       fmt.Sprintf("Critical Vulnerability: %s", vuln.CVE),
				Description: vuln.Description,
				Source:      "vulnerability_scanner",
				ContainerID: &containerID,
				Metadata: map[string]interface{}{
					"cve":         vuln.CVE,
					"package":     vuln.Package,
					"fixed_in":    vuln.FixedIn,
					"cvss_score":  vuln.CVSSScore,
				},
				Status: AlertStatusOpen,
			}
			s.GenerateAlert(alert)
		}
	}
}

// performComplianceScan performs a compliance scan
func (s *SecurityAuditService) performComplianceScan(ctx context.Context, containerID string) {
	standards := []ComplianceStandard{StandardCIS, StandardNIST}
	_, err := s.PerformComplianceCheck(ctx, containerID, standards)
	if err != nil {
		s.logger.WithError(err).WithField("container_id", containerID).Error("Compliance scan failed")
	}
}

// performRiskScan performs a risk assessment scan
func (s *SecurityAuditService) performRiskScan(ctx context.Context, containerID string) {
	_, err := s.PerformRiskAssessment(ctx, containerID)
	if err != nil {
		s.logger.WithError(err).WithField("container_id", containerID).Error("Risk assessment failed")
	}
}

// performFullScan performs all types of scans
func (s *SecurityAuditService) performFullScan(ctx context.Context, containerID string) {
	s.performVulnerabilityScan(ctx, containerID)
	s.performComplianceScan(ctx, containerID)
	s.performRiskScan(ctx, containerID)
}

// performScheduledComplianceChecks performs scheduled compliance checks
func (s *SecurityAuditService) performScheduledComplianceChecks() {
	ctx := context.Background()
	containers, err := s.containerRepo.GetAll(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get containers for compliance check")
		return
	}

	for _, container := range containers {
		s.scanScheduler.scanQueue <- &ScanRequest{
			ContainerID: container.ContainerID,
			ScanType:    ScanTypeCompliance,
			Priority:    1,
			Scheduled:   time.Now(),
		}
	}
}

// performScheduledRiskAssessments performs scheduled risk assessments
func (s *SecurityAuditService) performScheduledRiskAssessments() {
	ctx := context.Background()
	containers, err := s.containerRepo.GetAll(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get containers for risk assessment")
		return
	}

	for _, container := range containers {
		s.scanScheduler.scanQueue <- &ScanRequest{
			ContainerID: container.ContainerID,
			ScanType:    ScanTypeRisk,
			Priority:    2,
			Scheduled:   time.Now(),
		}
	}
}

// Utility functions

// marshalToJSON marshals a map to JSON string
func marshalToJSON(data map[string]interface{}) string {
	if data == nil {
		return "{}"
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

// generateRandomID generates a random ID for events/alerts
func generateRandomID() int64 {
	return time.Now().UnixNano() % 1000000
}

// Filter structures

type AuditEventFilter struct {
	EventTypes  []AuditEventType `json:"event_types,omitempty"`
	Severities  []SecuritySeverity `json:"severities,omitempty"`
	UserID      *int64           `json:"user_id,omitempty"`
	ContainerID *string          `json:"container_id,omitempty"`
	StartTime   *time.Time       `json:"start_time,omitempty"`
	EndTime     *time.Time       `json:"end_time,omitempty"`
	Limit       int              `json:"limit"`
	Offset      int              `json:"offset"`
}

type SecurityAlertFilter struct {
	AlertTypes  []SecurityAlertType `json:"alert_types,omitempty"`
	Severities  []SecuritySeverity  `json:"severities,omitempty"`
	Status      []AlertStatus       `json:"status,omitempty"`
	UserID      *int64              `json:"user_id,omitempty"`
	ContainerID *string             `json:"container_id,omitempty"`
	StartTime   *time.Time          `json:"start_time,omitempty"`
	EndTime     *time.Time          `json:"end_time,omitempty"`
	Limit       int                 `json:"limit"`
	Offset      int                 `json:"offset"`
}

// Close gracefully shuts down the security audit service
func (s *SecurityAuditService) Close() error {
	// Close audit queue
	close(s.auditQueue)

	// Close all alert channels
	s.mu.Lock()
	for _, channel := range s.alertChannels {
		close(channel)
	}
	s.alertChannels = make(map[string]chan *SecurityAlert)
	s.mu.Unlock()

	// Close scan queue
	close(s.scanScheduler.scanQueue)

	s.logger.Info("Security audit service shut down")
	return nil
}