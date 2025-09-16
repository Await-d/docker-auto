package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthCheck interface defines the contract for health checks
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthResult
	Dependencies() []string
	Config() HealthCheckConfig
}

// HealthChecker manages and executes health checks
type HealthChecker struct {
	mu                sync.RWMutex
	checks            map[string]HealthCheck
	results           map[string]HealthResult
	history           map[string][]HealthHistoryItem
	config            HealthConfig
	failureCounts     map[string]int
	successCounts     map[string]int
	lastChecked       map[string]time.Time
	recoveryActions   map[string][]RecoveryAction
	alertHandlers     []HealthAlertHandler
	startTime         time.Time
	stopped           bool
	stopChan          chan bool
	metrics           map[string]*HealthMetrics
}

// HealthAlertHandler interface for handling health alerts
type HealthAlertHandler interface {
	HandleAlert(alert HealthAlert) error
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(config HealthConfig) *HealthChecker {
	hc := &HealthChecker{
		checks:          make(map[string]HealthCheck),
		results:         make(map[string]HealthResult),
		history:         make(map[string][]HealthHistoryItem),
		config:          config,
		failureCounts:   make(map[string]int),
		successCounts:   make(map[string]int),
		lastChecked:     make(map[string]time.Time),
		recoveryActions: make(map[string][]RecoveryAction),
		startTime:       time.Now(),
		stopChan:        make(chan bool),
		metrics:         make(map[string]*HealthMetrics),
	}

	if config.Enabled {
		go hc.runHealthChecks()
	}

	return hc
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(check HealthCheck) error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	name := check.Name()
	if _, exists := hc.checks[name]; exists {
		return fmt.Errorf("health check %s already registered", name)
	}

	hc.checks[name] = check
	hc.results[name] = HealthResult{
		Status:    HealthStatusUnknown,
		Message:   "Not yet checked",
		Timestamp: time.Now(),
	}
	hc.history[name] = make([]HealthHistoryItem, 0)
	hc.metrics[name] = &HealthMetrics{
		CheckName:     name,
		CurrentStatus: HealthStatusUnknown,
	}

	return nil
}

// UnregisterCheck removes a health check
func (hc *HealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	delete(hc.results, name)
	delete(hc.history, name)
	delete(hc.failureCounts, name)
	delete(hc.successCounts, name)
	delete(hc.lastChecked, name)
	delete(hc.recoveryActions, name)
	delete(hc.metrics, name)
}

// AddRecoveryAction adds a recovery action for a specific check
func (hc *HealthChecker) AddRecoveryAction(checkName string, action RecoveryAction) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.recoveryActions[checkName] == nil {
		hc.recoveryActions[checkName] = make([]RecoveryAction, 0)
	}
	hc.recoveryActions[checkName] = append(hc.recoveryActions[checkName], action)
}

// AddAlertHandler adds an alert handler
func (hc *HealthChecker) AddAlertHandler(handler HealthAlertHandler) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.alertHandlers = append(hc.alertHandlers, handler)
}

// CheckHealth performs a single health check
func (hc *HealthChecker) CheckHealth(name string) (HealthResult, error) {
	hc.mu.RLock()
	check, exists := hc.checks[name]
	hc.mu.RUnlock()

	if !exists {
		return HealthResult{}, fmt.Errorf("health check %s not found", name)
	}

	return hc.executeCheck(check)
}

// CheckAllHealth performs all registered health checks
func (hc *HealthChecker) CheckAllHealth() map[string]HealthResult {
	hc.mu.RLock()
	checks := make([]HealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	results := make(map[string]HealthResult)

	// Execute checks in parallel
	var wg sync.WaitGroup
	resultsChan := make(chan struct {
		name   string
		result HealthResult
	}, len(checks))

	for _, check := range checks {
		wg.Add(1)
		go func(c HealthCheck) {
			defer wg.Done()
			result := hc.executeCheck(c)
			resultsChan <- struct {
				name   string
				result HealthResult
			}{c.Name(), result}
		}(check)
	}

	wg.Wait()
	close(resultsChan)

	for result := range resultsChan {
		results[result.name] = result.result
	}

	return results
}

// GetAggregateHealth returns the overall health status
func (hc *HealthChecker) GetAggregateHealth() AggregateHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	aggregate := AggregateHealth{
		Checks:      make(map[string]HealthResult),
		Dependencies: make(map[string]HealthResult),
		Components:  make(map[string]HealthResult),
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Uptime:      time.Since(hc.startTime),
		Environment: "production",
	}

	// Copy current results
	for name, result := range hc.results {
		aggregate.Checks[name] = result
	}

	// Determine overall status
	aggregate.Status = hc.calculateAggregateStatus()
	aggregate.Message = hc.generateAggregateMessage(aggregate.Status)

	return aggregate
}

// GetHealthHistory returns historical health data for a check
func (hc *HealthChecker) GetHealthHistory(name string) (HealthCheckHistory, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	history, exists := hc.history[name]
	if !exists {
		return HealthCheckHistory{}, fmt.Errorf("no history found for check %s", name)
	}

	summary := hc.calculateHealthSummary(name, history)

	return HealthCheckHistory{
		CheckName: name,
		Results:   history,
		Summary:   summary,
	}, nil
}

// GetHealthMetrics returns metrics for a specific check
func (hc *HealthChecker) GetHealthMetrics(name string) (*HealthMetrics, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	metrics, exists := hc.metrics[name]
	if !exists {
		return nil, fmt.Errorf("no metrics found for check %s", name)
	}

	// Create a copy to avoid concurrent modification
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// GetAllHealthMetrics returns metrics for all checks
func (hc *HealthChecker) GetAllHealthMetrics() map[string]*HealthMetrics {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	allMetrics := make(map[string]*HealthMetrics)
	for name, metrics := range hc.metrics {
		metricsCopy := *metrics
		allMetrics[name] = &metricsCopy
	}
	return allMetrics
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.stopped {
		hc.stopped = true
		close(hc.stopChan)
	}
}

// executeCheck executes a single health check with timeout and retries
func (hc *HealthChecker) executeCheck(check HealthCheck) HealthResult {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	var result HealthResult
	var lastErr error

	// Retry logic
	for attempt := 0; attempt <= hc.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(hc.config.RetryDelay)
		}

		result = check.Check(ctx)
		result.Timestamp = time.Now()
		result.Duration = time.Since(start)

		if result.Status == HealthStatusHealthy || result.Status == HealthStatusDegraded {
			break
		}

		lastErr = fmt.Errorf("health check failed: %s", result.Message)
	}

	// Store result and update metrics
	hc.storeResult(check.Name(), result)

	// Handle status changes
	hc.handleStatusChange(check.Name(), result)

	return result
}

// storeResult stores the health check result and updates metrics
func (hc *HealthChecker) storeResult(name string, result HealthResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Store current result
	hc.results[name] = result
	hc.lastChecked[name] = result.Timestamp

	// Add to history
	historyItem := HealthHistoryItem{
		Status:    result.Status,
		Message:   result.Message,
		Duration:  result.Duration,
		Timestamp: result.Timestamp,
		Error:     result.Error,
	}

	hc.history[name] = append(hc.history[name], historyItem)

	// Limit history size
	if len(hc.history[name]) > 1000 {
		hc.history[name] = hc.history[name][len(hc.history[name])-1000:]
	}

	// Update failure/success counts
	if result.Status == HealthStatusHealthy || result.Status == HealthStatusDegraded {
		hc.successCounts[name]++
		hc.failureCounts[name] = 0 // Reset consecutive failures
	} else {
		hc.failureCounts[name]++
		hc.successCounts[name] = 0 // Reset consecutive successes
	}

	// Update metrics
	hc.updateMetrics(name, result)
}

// updateMetrics updates the metrics for a health check
func (hc *HealthChecker) updateMetrics(name string, result HealthResult) {
	metrics := hc.metrics[name]
	if metrics == nil {
		return
	}

	metrics.TotalChecks++
	metrics.CurrentStatus = result.Status
	metrics.LastChecked = result.Timestamp

	if result.Status == HealthStatusHealthy || result.Status == HealthStatusDegraded {
		metrics.SuccessfulChecks++
		metrics.ConsecutiveSuccesses++
		metrics.ConsecutiveFailures = 0
	} else {
		metrics.FailedChecks++
		metrics.ConsecutiveFailures++
		metrics.ConsecutiveSuccesses = 0
	}

	if metrics.TotalChecks > 0 {
		metrics.SuccessRate = float64(metrics.SuccessfulChecks) / float64(metrics.TotalChecks) * 100
	}

	// Update duration metrics
	if metrics.TotalChecks == 1 {
		metrics.AverageDuration = result.Duration
		metrics.MinDuration = result.Duration
		metrics.MaxDuration = result.Duration
	} else {
		// Update average duration
		totalDuration := time.Duration(float64(metrics.AverageDuration) * float64(metrics.TotalChecks-1))
		metrics.AverageDuration = (totalDuration + result.Duration) / time.Duration(metrics.TotalChecks)

		// Update min/max duration
		if result.Duration < metrics.MinDuration {
			metrics.MinDuration = result.Duration
		}
		if result.Duration > metrics.MaxDuration {
			metrics.MaxDuration = result.Duration
		}
	}
}

// handleStatusChange handles health status changes and triggers alerts/recovery
func (hc *HealthChecker) handleStatusChange(name string, result HealthResult) {
	hc.mu.RLock()
	failureCount := hc.failureCounts[name]
	successCount := hc.successCounts[name]
	hc.mu.RUnlock()

	// Check if we need to trigger an alert
	if result.Status != HealthStatusHealthy && failureCount >= hc.config.FailureThreshold {
		hc.triggerAlert(name, result)
	}

	// Check if we need to trigger recovery actions
	if hc.config.EnableRecovery && failureCount >= hc.config.FailureThreshold {
		hc.triggerRecoveryActions(name, result)
	}
}

// triggerAlert triggers health alerts
func (hc *HealthChecker) triggerAlert(name string, result HealthResult) {
	alert := HealthAlert{
		ID:        fmt.Sprintf("%s-%d", name, time.Now().Unix()),
		CheckName: name,
		Status:    result.Status,
		Message:   result.Message,
		Severity:  hc.getSeverity(result.Status),
		Timestamp: time.Now(),
		Resolved:  false,
		Context: map[string]interface{}{
			"duration":           result.Duration.String(),
			"consecutive_failures": hc.failureCounts[name],
		},
	}

	// Send alert to handlers
	for _, handler := range hc.alertHandlers {
		go func(h HealthAlertHandler) {
			h.HandleAlert(alert)
		}(handler)
	}
}

// triggerRecoveryActions triggers automated recovery actions
func (hc *HealthChecker) triggerRecoveryActions(name string, result HealthResult) {
	hc.mu.RLock()
	actions := hc.recoveryActions[name]
	hc.mu.RUnlock()

	for _, action := range actions {
		if action.Enabled {
			go hc.executeRecoveryAction(action, name, result)
		}
	}
}

// executeRecoveryAction executes a recovery action
func (hc *HealthChecker) executeRecoveryAction(action RecoveryAction, checkName string, result HealthResult) {
	// Implementation would depend on the specific recovery action
	// For now, we'll just log the action
	fmt.Printf("Executing recovery action %s for check %s\n", action.Name, checkName)
}

// runHealthChecks runs the health check loop
func (hc *HealthChecker) runHealthChecks() {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	// Wait for grace period before starting checks
	if hc.config.GracePeriod > 0 {
		time.Sleep(hc.config.GracePeriod)
	}

	for {
		select {
		case <-ticker.C:
			hc.CheckAllHealth()
		case <-hc.stopChan:
			return
		}
	}
}

// calculateAggregateStatus calculates the overall health status
func (hc *HealthChecker) calculateAggregateStatus() HealthStatus {
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0
	totalCount := 0

	for _, result := range hc.results {
		totalCount++
		switch result.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
	}

	if totalCount == 0 {
		return HealthStatusUnknown
	}

	// If any critical checks are unhealthy, the system is unhealthy
	if unhealthyCount > 0 {
		return HealthStatusUnhealthy
	}

	// If any checks are degraded, the system is degraded
	if degradedCount > 0 {
		return HealthStatusDegraded
	}

	// All checks are healthy
	return HealthStatusHealthy
}

// generateAggregateMessage generates a human-readable aggregate health message
func (hc *HealthChecker) generateAggregateMessage(status HealthStatus) string {
	totalChecks := len(hc.results)

	switch status {
	case HealthStatusHealthy:
		return fmt.Sprintf("All %d health checks are passing", totalChecks)
	case HealthStatusDegraded:
		return "Some health checks are degraded"
	case HealthStatusUnhealthy:
		return "One or more health checks are failing"
	default:
		return "Health status unknown"
	}
}

// calculateHealthSummary calculates summary statistics for health history
func (hc *HealthChecker) calculateHealthSummary(name string, history []HealthHistoryItem) HealthSummary {
	summary := HealthSummary{}

	if len(history) == 0 {
		return summary
	}

	summary.TotalChecks = len(history)

	var totalDuration time.Duration
	var lastSuccess, lastFailure *time.Time

	for _, item := range history {
		totalDuration += item.Duration

		if item.Status == HealthStatusHealthy || item.Status == HealthStatusDegraded {
			summary.SuccessfulChecks++
			lastSuccess = &item.Timestamp
		} else {
			summary.FailedChecks++
			lastFailure = &item.Timestamp
		}
	}

	if summary.TotalChecks > 0 {
		summary.SuccessRate = float64(summary.SuccessfulChecks) / float64(summary.TotalChecks) * 100
		summary.AverageDuration = totalDuration / time.Duration(summary.TotalChecks)
	}

	summary.LastSuccess = lastSuccess
	summary.LastFailure = lastFailure

	// Calculate consecutive counts from the most recent results
	summary.ConsecutiveFailures = hc.failureCounts[name]
	summary.ConsecutiveSuccesses = hc.successCounts[name]

	return summary
}

// getSeverity returns the severity level for a health status
func (hc *HealthChecker) getSeverity(status HealthStatus) string {
	switch status {
	case HealthStatusUnhealthy:
		return "critical"
	case HealthStatusDegraded:
		return "warning"
	case HealthStatusHealthy:
		return "info"
	default:
		return "unknown"
	}
}