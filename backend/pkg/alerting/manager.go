package alerting

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// AlertManager manages alerts and notifications
type AlertManager struct {
	mu                  sync.RWMutex
	config              AlertingConfig
	rules               map[string]*AlertRule
	alerts              map[string]*Alert
	groups              map[string]*AlertGroup
	channels            map[string]AlertChannel
	suppressionRules    map[string]*SuppressionRule
	escalationPolicies  map[string]*EscalationPolicy
	activeNotifications map[string]*AlertNotification
	evaluationTicker    *time.Ticker
	stopChan           chan bool
	metrics            *AlertMetrics
	storage            AlertStorage
	router             *AlertRouter
}

// AlertStorage interface for persisting alerts
type AlertStorage interface {
	Save(alert Alert) error
	Get(id string) (*Alert, error)
	List(filters map[string]interface{}) ([]Alert, error)
	Delete(id string) error
	UpdateStatus(id string, status AlertStatus) error
}

// AlertRouter handles alert routing and grouping
type AlertRouter struct {
	routes []AlertRoute
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertingConfig) *AlertManager {
	am := &AlertManager{
		config:              config,
		rules:               make(map[string]*AlertRule),
		alerts:              make(map[string]*Alert),
		groups:              make(map[string]*AlertGroup),
		channels:            make(map[string]AlertChannel),
		suppressionRules:    make(map[string]*SuppressionRule),
		escalationPolicies:  make(map[string]*EscalationPolicy),
		activeNotifications: make(map[string]*AlertNotification),
		stopChan:           make(chan bool),
		metrics: &AlertMetrics{
			AlertsByStatus:     make(map[string]uint64),
			AlertsBySeverity:   make(map[string]uint64),
			AlertsByComponent:  make(map[string]uint64),
		},
		router: &AlertRouter{routes: config.Routes},
	}

	if config.Enabled {
		am.evaluationTicker = time.NewTicker(config.EvaluationInterval)
		go am.evaluationLoop()
		go am.cleanupLoop()
	}

	return am
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule AlertRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[rule.ID]; exists {
		return fmt.Errorf("rule with ID %s already exists", rule.ID)
	}

	am.rules[rule.ID] = &rule
	return nil
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(ruleID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[ruleID]; !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	delete(am.rules, ruleID)
	return nil
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(channel AlertChannel) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.channels[channel.Name()] = channel
	return nil
}

// RemoveChannel removes an alert channel
func (am *AlertManager) RemoveChannel(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.channels[name]; !exists {
		return fmt.Errorf("channel with name %s not found", name)
	}

	delete(am.channels, name)
	return nil
}

// CreateAlert creates a new alert
func (am *AlertManager) CreateAlert(alert Alert) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate ID if not provided
	if alert.ID == "" {
		alert.ID = am.generateAlertID()
	}

	// Set timestamps
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}
	alert.UpdatedAt = time.Now()
	alert.FirstSeen = alert.Timestamp
	alert.LastSeen = alert.Timestamp
	alert.Count = 1
	alert.Status = AlertStatusActive

	// Check for existing alerts with same fingerprint
	fingerprint := am.generateFingerprint(alert)
	for _, existingAlert := range am.alerts {
		if am.generateFingerprint(*existingAlert) == fingerprint {
			// Update existing alert
			existingAlert.Count++
			existingAlert.LastSeen = alert.Timestamp
			existingAlert.UpdatedAt = time.Now()
			existingAlert.Value = alert.Value

			// Save to storage if available
			if am.storage != nil {
				am.storage.Save(*existingAlert)
			}

			return nil
		}
	}

	// Store new alert
	am.alerts[alert.ID] = &alert

	// Update metrics
	am.updateMetrics(&alert, "created")

	// Save to storage if available
	if am.storage != nil {
		am.storage.Save(alert)
	}

	// Route alert for notifications
	go am.routeAlert(alert)

	return nil
}

// ResolveAlert resolves an active alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert with ID %s not found", alertID)
	}

	if alert.Status == AlertStatusResolved {
		return nil // Already resolved
	}

	// Update alert status
	alert.Status = AlertStatusResolved
	resolvedAt := time.Now()
	alert.ResolvedAt = &resolvedAt
	alert.UpdatedAt = resolvedAt

	// Update metrics
	am.updateMetrics(alert, "resolved")

	// Save to storage if available
	if am.storage != nil {
		am.storage.UpdateStatus(alertID, AlertStatusResolved)
	}

	// Send resolution notification
	go am.sendResolutionNotification(*alert)

	return nil
}

// AcknowledgeAlert acknowledges an alert
func (am *AlertManager) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert with ID %s not found", alertID)
	}

	// Update alert status
	alert.Status = AlertStatusAcknowledged
	ackedAt := time.Now()
	alert.AckedAt = &ackedAt
	alert.AckedBy = acknowledgedBy
	alert.UpdatedAt = ackedAt

	// Update metrics
	am.updateMetrics(alert, "acknowledged")

	// Save to storage if available
	if am.storage != nil {
		am.storage.UpdateStatus(alertID, AlertStatusAcknowledged)
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (am *AlertManager) GetAlert(alertID string) (*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert with ID %s not found", alertID)
	}

	// Create a copy to avoid external modification
	alertCopy := *alert
	return &alertCopy, nil
}

// ListAlerts lists all alerts with optional filters
func (am *AlertManager) ListAlerts(filters map[string]interface{}) ([]Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var result []Alert
	for _, alert := range am.alerts {
		if am.matchesFilters(*alert, filters) {
			result = append(result, *alert)
		}
	}

	return result, nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var activeAlerts []Alert
	for _, alert := range am.alerts {
		if alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, *alert)
		}
	}

	return activeAlerts
}

// GetMetrics returns current alerting metrics
func (am *AlertManager) GetMetrics() AlertMetrics {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Create a copy to avoid concurrent modification
	metrics := AlertMetrics{
		TotalAlerts:         am.metrics.TotalAlerts,
		ActiveAlerts:        am.metrics.ActiveAlerts,
		ResolvedAlerts:      am.metrics.ResolvedAlerts,
		NotificationsSent:   am.metrics.NotificationsSent,
		NotificationsFailed: am.metrics.NotificationsFailed,
		EvaluationTime:      am.metrics.EvaluationTime,
		LastEvaluation:      am.metrics.LastEvaluation,
		AlertsByStatus:      make(map[string]uint64),
		AlertsBySeverity:    make(map[string]uint64),
		AlertsByComponent:   make(map[string]uint64),
	}

	// Copy maps
	for k, v := range am.metrics.AlertsByStatus {
		metrics.AlertsByStatus[k] = v
	}
	for k, v := range am.metrics.AlertsBySeverity {
		metrics.AlertsBySeverity[k] = v
	}
	for k, v := range am.metrics.AlertsByComponent {
		metrics.AlertsByComponent[k] = v
	}

	return metrics
}

// AddSuppressionRule adds a suppression rule
func (am *AlertManager) AddSuppressionRule(rule SuppressionRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.suppressionRules[rule.ID] = &rule
	return nil
}

// RemoveSuppressionRule removes a suppression rule
func (am *AlertManager) RemoveSuppressionRule(ruleID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.suppressionRules, ruleID)
	return nil
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	if am.evaluationTicker != nil {
		am.evaluationTicker.Stop()
	}
	close(am.stopChan)
}

// evaluationLoop runs the periodic rule evaluation
func (am *AlertManager) evaluationLoop() {
	for {
		select {
		case <-am.evaluationTicker.C:
			am.evaluateRules()
		case <-am.stopChan:
			return
		}
	}
}

// evaluateRules evaluates all active alert rules
func (am *AlertManager) evaluateRules() {
	start := time.Now()

	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	for _, rule := range rules {
		go am.evaluateRule(*rule)
	}

	am.mu.Lock()
	am.metrics.EvaluationTime = time.Since(start)
	am.metrics.LastEvaluation = time.Now()
	am.mu.Unlock()
}

// evaluateRule evaluates a single alert rule
func (am *AlertManager) evaluateRule(rule AlertRule) {
	// This is a simplified rule evaluation
	// In a real implementation, you would evaluate the rule expression
	// against metrics or other data sources

	// For demonstration, we'll create a placeholder evaluation
	// In practice, this would query metrics and evaluate the condition
	value := 0.0 // This would come from metrics evaluation

	shouldAlert := false
	switch rule.Condition {
	case AlertConditionGreaterThan:
		shouldAlert = value > rule.Threshold
	case AlertConditionLessThan:
		shouldAlert = value < rule.Threshold
	// Add other conditions as needed
	}

	if shouldAlert {
		alert := Alert{
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    rule.Severity,
			Source:      "rule_evaluation",
			Labels:      rule.Labels,
			Annotations: rule.Annotations,
			Value:       value,
			Threshold:   rule.Threshold,
		}

		am.CreateAlert(alert)
	}
}

// routeAlert routes an alert to appropriate channels
func (am *AlertManager) routeAlert(alert Alert) {
	// Check suppression rules
	if am.isAlertSuppressed(alert) {
		return
	}

	// Find matching routes
	receivers := am.router.findReceivers(alert)

	// Group alerts if configured
	group := am.findOrCreateGroup(alert)
	if group != nil {
		am.addAlertToGroup(alert, group)

		// Check if group should be notified
		if am.shouldNotifyGroup(*group) {
			am.notifyGroup(*group, receivers)
		}
	} else {
		// Send individual alert
		am.sendNotifications(alert, receivers)
	}
}

// isAlertSuppressed checks if an alert is suppressed
func (am *AlertManager) isAlertSuppressed(alert Alert) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	now := time.Now()
	for _, rule := range am.suppressionRules {
		if !rule.Enabled {
			continue
		}

		// Check time window
		if rule.StartTime != nil && now.Before(*rule.StartTime) {
			continue
		}
		if rule.EndTime != nil && now.After(*rule.EndTime) {
			continue
		}
		if rule.Duration != nil {
			// Check if rule is still valid based on creation time + duration
			if now.After(rule.CreatedAt.Add(*rule.Duration)) {
				continue
			}
		}

		// Check matchers
		if am.matchesFilters(alert, am.filtersToMap(rule.Matchers)) {
			return true
		}
	}

	return false
}

// sendNotifications sends notifications for an alert
func (am *AlertManager) sendNotifications(alert Alert, receivers []string) {
	for _, receiverName := range receivers {
		// Find channels for receiver
		for _, channelName := range am.getChannelsForReceiver(receiverName) {
			go am.sendNotification(alert, channelName)
		}
	}
}

// sendNotification sends a single notification
func (am *AlertManager) sendNotification(alert Alert, channelName string) {
	am.mu.RLock()
	channel, exists := am.channels[channelName]
	am.mu.RUnlock()

	if !exists {
		return
	}

	notification := AlertNotification{
		ID:        am.generateNotificationID(),
		AlertID:   alert.ID,
		Channel:   channelName,
		Subject:   fmt.Sprintf("[%s] %s", alert.Severity, alert.Name),
		Message:   am.formatAlertMessage(alert, channel),
		Timestamp: time.Now(),
		Status:    "pending",
	}

	am.mu.Lock()
	am.activeNotifications[notification.ID] = &notification
	am.mu.Unlock()

	err := channel.Send(alert)

	am.mu.Lock()
	if err != nil {
		notification.Status = "failed"
		notification.Error = err.Error()
		am.metrics.NotificationsFailed++
	} else {
		notification.Status = "sent"
		sentAt := time.Now()
		notification.SentAt = &sentAt
		am.metrics.NotificationsSent++
	}
	am.mu.Unlock()
}

// Helper methods

// generateAlertID generates a unique alert ID
func (am *AlertManager) generateAlertID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// generateNotificationID generates a unique notification ID
func (am *AlertManager) generateNotificationID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return "notif_" + hex.EncodeToString(bytes)
}

// generateFingerprint generates a fingerprint for alert deduplication
func (am *AlertManager) generateFingerprint(alert Alert) string {
	fingerprint := alert.Name + "|" + alert.Component
	for k, v := range alert.Labels {
		fingerprint += "|" + k + "=" + v
	}
	return fingerprint
}

// updateMetrics updates internal metrics
func (am *AlertManager) updateMetrics(alert *Alert, action string) {
	switch action {
	case "created":
		am.metrics.TotalAlerts++
		am.metrics.ActiveAlerts++
		am.metrics.AlertsByStatus[string(alert.Status)]++
		am.metrics.AlertsBySeverity[string(alert.Severity)]++
		if alert.Component != "" {
			am.metrics.AlertsByComponent[alert.Component]++
		}
	case "resolved":
		am.metrics.ActiveAlerts--
		am.metrics.ResolvedAlerts++
		am.metrics.AlertsByStatus[string(AlertStatusActive)]--
		am.metrics.AlertsByStatus[string(AlertStatusResolved)]++
	case "acknowledged":
		am.metrics.AlertsByStatus[string(AlertStatusActive)]--
		am.metrics.AlertsByStatus[string(AlertStatusAcknowledged)]++
	}
}

// matchesFilters checks if an alert matches the given filters
func (am *AlertManager) matchesFilters(alert Alert, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "status":
			if string(alert.Status) != value.(string) {
				return false
			}
		case "severity":
			if string(alert.Severity) != value.(string) {
				return false
			}
		case "component":
			if alert.Component != value.(string) {
				return false
			}
		case "source":
			if alert.Source != value.(string) {
				return false
			}
		default:
			// Check labels
			if labelValue, exists := alert.Labels[key]; !exists || labelValue != value.(string) {
				return false
			}
		}
	}
	return true
}

// filtersToMap converts AlertFilter slice to map
func (am *AlertManager) filtersToMap(filters []AlertFilter) map[string]interface{} {
	result := make(map[string]interface{})
	for _, filter := range filters {
		result[filter.Field] = filter.Value
	}
	return result
}

// findReceivers finds receivers for an alert based on routing rules
func (ar *AlertRouter) findReceivers(alert Alert) []string {
	var receivers []string

	for _, route := range ar.routes {
		if ar.matchesRoute(alert, route) {
			receivers = append(receivers, route.Receiver)
			if !route.Continue {
				break
			}
		}
	}

	if len(receivers) == 0 {
		receivers = append(receivers, "default")
	}

	return receivers
}

// matchesRoute checks if an alert matches a route
func (ar *AlertRouter) matchesRoute(alert Alert, route AlertRoute) bool {
	// Check exact matches
	for key, value := range route.Match {
		if labelValue, exists := alert.Labels[key]; !exists || labelValue != value {
			return false
		}
	}

	// Check regex matches
	for key, pattern := range route.MatchRE {
		labelValue, exists := alert.Labels[key]
		if !exists {
			return false
		}

		matched, err := regexp.MatchString(pattern, labelValue)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// getChannelsForReceiver gets channel names for a receiver
func (am *AlertManager) getChannelsForReceiver(receiverName string) []string {
	// This is a simplified implementation
	// In practice, you would look up the receiver configuration
	// and return the associated channels

	am.mu.RLock()
	defer am.mu.RUnlock()

	var channels []string
	for channelName := range am.channels {
		channels = append(channels, channelName)
	}

	return channels
}

// formatAlertMessage formats an alert message for a channel
func (am *AlertManager) formatAlertMessage(alert Alert, channel AlertChannel) string {
	// This is a basic implementation
	// In practice, you would use templates and format based on channel type
	return fmt.Sprintf("Alert: %s\nSeverity: %s\nDescription: %s\nValue: %.2f\nTime: %s",
		alert.Name, alert.Severity, alert.Description, alert.Value, alert.Timestamp.Format(time.RFC3339))
}

// Placeholder methods for grouping (simplified implementation)
func (am *AlertManager) findOrCreateGroup(alert Alert) *AlertGroup {
	// Simplified grouping logic
	return nil
}

func (am *AlertManager) addAlertToGroup(alert Alert, group *AlertGroup) {
	// Add alert to group
}

func (am *AlertManager) shouldNotifyGroup(group AlertGroup) bool {
	// Check if group should be notified
	return true
}

func (am *AlertManager) notifyGroup(group AlertGroup, receivers []string) {
	// Notify about the group
}

func (am *AlertManager) sendResolutionNotification(alert Alert) {
	// Send resolution notification
}

// cleanupLoop periodically cleans up old alerts and notifications
func (am *AlertManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			am.cleanup()
		case <-am.stopChan:
			return
		}
	}
}

// cleanup removes old resolved alerts and notifications
func (am *AlertManager) cleanup() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour) // Keep alerts for 24 hours

	// Clean up resolved alerts
	for id, alert := range am.alerts {
		if alert.Status == AlertStatusResolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, id)
		}
	}

	// Clean up old notifications
	for id, notification := range am.activeNotifications {
		if notification.Timestamp.Before(cutoff) {
			delete(am.activeNotifications, id)
		}
	}
}