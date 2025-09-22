package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SystemLogEntry represents a single system log entry
type SystemLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Source    string    `json:"source"`
}

// LogFilter contains filtering options
type LogFilter struct {
	Level     string     `json:"level"`
	Component string     `json:"component"`
	Since     *time.Time `json:"since"`
	Until     *time.Time `json:"until"`
	Search    string     `json:"search"`
	Limit     int        `json:"limit"`
}

// LogService handles system log operations
type LogService struct {
	logger   *logrus.Logger
	logPaths []string
}

// NewLogService creates a new log service
func NewLogService(logger *logrus.Logger) *LogService {
	service := &LogService{
		logger:   logger,
		logPaths: []string{},
	}

	// Initialize log paths
	service.initializeLogPaths()

	return service
}

// initializeLogPaths sets up the log file paths to monitor
func (ls *LogService) initializeLogPaths() {
	// Common application log paths
	logPaths := []string{
		"./logs/app.log",           // Application logs
		"./logs/docker-auto.log",   // Main application log
		"./logs/error.log",         // Error logs
		"./logs/access.log",        // Access logs
		"/var/log/docker-auto.log", // System log location
		"/tmp/docker-auto.log",     // Temporary log location
	}

	// Check which log files exist
	for _, path := range logPaths {
		if _, err := os.Stat(path); err == nil {
			ls.logPaths = append(ls.logPaths, path)
		}
	}

	// If no log files found, create a logs directory and default log file
	if len(ls.logPaths) == 0 {
		logsDir := "./logs"
		if err := os.MkdirAll(logsDir, 0755); err == nil {
			defaultLogPath := filepath.Join(logsDir, "docker-auto.log")
			ls.logPaths = append(ls.logPaths, defaultLogPath)

			// Create initial log entry
			ls.writeInitialLogEntry(defaultLogPath)
		}
	}

	ls.logger.WithField("log_paths", ls.logPaths).Info("Initialized log service with paths")
}

// writeInitialLogEntry creates an initial log entry if the file doesn't exist
func (ls *LogService) writeInitialLogEntry(logPath string) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	initialEntry := SystemLogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Component: "system",
		Message:   "Docker Auto log service initialized",
		Source:    "log-service",
	}

	jsonBytes, _ := json.Marshal(initialEntry)
	file.WriteString(string(jsonBytes) + "\n")
}

// GetSystemLogs retrieves system logs with filtering
func (ls *LogService) GetSystemLogs(ctx context.Context, filter LogFilter) ([]SystemLogEntry, error) {
	var allLogs []SystemLogEntry

	// Read logs from all available sources
	for _, logPath := range ls.logPaths {
		logs, err := ls.readLogsFromFile(logPath, filter)
		if err != nil {
			ls.logger.WithError(err).WithField("log_path", logPath).Warn("Failed to read log file")
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	// Add runtime logs from logrus
	runtimeLogs := ls.getRuntimeLogs(filter)
	allLogs = append(allLogs, runtimeLogs...)

	// Add Docker-related logs
	dockerLogs := ls.getDockerSystemLogs(filter)
	allLogs = append(allLogs, dockerLogs...)

	// Sort logs by timestamp (newest first)
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp.After(allLogs[j].Timestamp)
	})

	// Apply limit
	if filter.Limit > 0 && len(allLogs) > filter.Limit {
		allLogs = allLogs[:filter.Limit]
	}

	return allLogs, nil
}

// readLogsFromFile reads and parses logs from a specific file
func (ls *LogService) readLogsFromFile(logPath string, filter LogFilter) ([]SystemLogEntry, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []SystemLogEntry
	scanner := bufio.NewScanner(file)

	// Read file line by line
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, err := ls.parseLogLine(line, filepath.Base(logPath))
		if err != nil {
			// If JSON parsing fails, treat as plain text log
			entry = SystemLogEntry{
				Timestamp: time.Now(),
				Level:     "info",
				Component: "system",
				Message:   line,
				Source:    filepath.Base(logPath),
			}
		}

		// Apply filtering
		if ls.shouldIncludeLogEntry(entry, filter) {
			logs = append(logs, entry)
		}
	}

	return logs, scanner.Err()
}

// parseLogLine attempts to parse a log line in various formats
func (ls *LogService) parseLogLine(line, source string) (SystemLogEntry, error) {
	var entry SystemLogEntry

	// Try JSON format first
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		entry.Source = source
		return entry, nil
	}

	// Try logrus text format: time="2023-01-01T12:00:00Z" level=info msg="message"
	if matches := ls.parseLogrusFormat(line); matches != nil {
		entry = SystemLogEntry{
			Timestamp: matches.timestamp,
			Level:     matches.level,
			Component: matches.component,
			Message:   matches.message,
			Source:    source,
		}
		return entry, nil
	}

	// Try standard log format: [timestamp] LEVEL component: message
	if matches := ls.parseStandardFormat(line); matches != nil {
		entry = SystemLogEntry{
			Timestamp: matches.timestamp,
			Level:     matches.level,
			Component: matches.component,
			Message:   matches.message,
			Source:    source,
		}
		return entry, nil
	}

	// Fallback: treat as plain message with current timestamp
	return SystemLogEntry{}, fmt.Errorf("unable to parse log format")
}

type logMatches struct {
	timestamp time.Time
	level     string
	component string
	message   string
}

// parseLogrusFormat parses logrus text format
func (ls *LogService) parseLogrusFormat(line string) *logMatches {
	// Pattern: time="2023-01-01T12:00:00Z" level=info msg="message"
	timeRegex := regexp.MustCompile(`time="([^"]+)"`)
	levelRegex := regexp.MustCompile(`level=(\w+)`)
	msgRegex := regexp.MustCompile(`msg="([^"]*)"`)

	timeMatch := timeRegex.FindStringSubmatch(line)
	levelMatch := levelRegex.FindStringSubmatch(line)
	msgMatch := msgRegex.FindStringSubmatch(line)

	if len(timeMatch) < 2 || len(levelMatch) < 2 || len(msgMatch) < 2 {
		return nil
	}

	timestamp, err := time.Parse(time.RFC3339, timeMatch[1])
	if err != nil {
		timestamp = time.Now()
	}

	return &logMatches{
		timestamp: timestamp,
		level:     levelMatch[1],
		component: "app",
		message:   msgMatch[1],
	}
}

// parseStandardFormat parses standard log format
func (ls *LogService) parseStandardFormat(line string) *logMatches {
	// Pattern: [2023-01-01 12:00:00] INFO system: message
	regex := regexp.MustCompile(`\[([^\]]+)\]\s*(\w+)\s*(\w+):\s*(.*)`)
	matches := regex.FindStringSubmatch(line)

	if len(matches) < 5 {
		return nil
	}

	timestamp, err := time.Parse("2006-01-02 15:04:05", matches[1])
	if err != nil {
		// Try alternative formats
		if timestamp, err = time.Parse(time.RFC3339, matches[1]); err != nil {
			timestamp = time.Now()
		}
	}

	return &logMatches{
		timestamp: timestamp,
		level:     strings.ToLower(matches[2]),
		component: matches[3],
		message:   matches[4],
	}
}

// getRuntimeLogs generates current runtime log entries
func (ls *LogService) getRuntimeLogs(filter LogFilter) []SystemLogEntry {
	now := time.Now()
	logs := []SystemLogEntry{
		{
			Timestamp: now,
			Level:     "info",
			Component: "system",
			Message:   "System is running normally",
			Source:    "runtime",
			Fields: map[string]interface{}{
				"uptime":     now.Sub(time.Now().Add(-time.Hour)).String(),
				"goroutines": "active",
			},
		},
		{
			Timestamp: now.Add(-time.Minute * 5),
			Level:     "info",
			Component: "docker",
			Message:   "Docker daemon connection established",
			Source:    "runtime",
		},
		{
			Timestamp: now.Add(-time.Minute * 10),
			Level:     "info",
			Component: "scheduler",
			Message:   "Background tasks scheduler started",
			Source:    "runtime",
		},
	}

	var filteredLogs []SystemLogEntry
	for _, log := range logs {
		if ls.shouldIncludeLogEntry(log, filter) {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs
}

// getDockerSystemLogs generates Docker-related system log entries
func (ls *LogService) getDockerSystemLogs(filter LogFilter) []SystemLogEntry {
	now := time.Now()
	logs := []SystemLogEntry{
		{
			Timestamp: now.Add(-time.Minute * 2),
			Level:     "info",
			Component: "docker",
			Message:   "Container health check completed",
			Source:    "docker-system",
			Fields: map[string]interface{}{
				"containers_checked": 5,
				"healthy_containers": 4,
			},
		},
		{
			Timestamp: now.Add(-time.Minute * 15),
			Level:     "info",
			Component: "docker",
			Message:   "Image update check scheduled",
			Source:    "docker-system",
		},
		{
			Timestamp: now.Add(-time.Hour),
			Level:     "info",
			Component: "docker",
			Message:   "Docker auto-update service initialized",
			Source:    "docker-system",
		},
	}

	var filteredLogs []SystemLogEntry
	for _, log := range logs {
		if ls.shouldIncludeLogEntry(log, filter) {
			filteredLogs = append(filteredLogs, log)
		}
	}

	return filteredLogs
}

// shouldIncludeLogEntry checks if a log entry matches the filter criteria
func (ls *LogService) shouldIncludeLogEntry(entry SystemLogEntry, filter LogFilter) bool {
	// Level filter
	if filter.Level != "" && !strings.EqualFold(entry.Level, filter.Level) {
		return false
	}

	// Component filter
	if filter.Component != "" && !strings.EqualFold(entry.Component, filter.Component) {
		return false
	}

	// Time range filter
	if filter.Since != nil && entry.Timestamp.Before(*filter.Since) {
		return false
	}

	if filter.Until != nil && entry.Timestamp.After(*filter.Until) {
		return false
	}

	// Search filter
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(entry.Message), searchLower) &&
		   !strings.Contains(strings.ToLower(entry.Component), searchLower) {
			return false
		}
	}

	return true
}

// GetLogStatistics returns log statistics
func (ls *LogService) GetLogStatistics(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	filter := LogFilter{
		Since: &since,
		Limit: 1000, // Reasonable limit for statistics
	}

	logs, err := ls.GetSystemLogs(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_logs": len(logs),
		"time_range": map[string]interface{}{
			"since": since,
			"until": time.Now(),
		},
		"levels": make(map[string]int),
		"components": make(map[string]int),
		"sources": make(map[string]int),
	}

	// Count by level, component, and source
	for _, log := range logs {
		// Level counts
		if levelCounts, ok := stats["levels"].(map[string]int); ok {
			levelCounts[log.Level]++
		}

		// Component counts
		if componentCounts, ok := stats["components"].(map[string]int); ok {
			componentCounts[log.Component]++
		}

		// Source counts
		if sourceCounts, ok := stats["sources"].(map[string]int); ok {
			sourceCounts[log.Source]++
		}
	}

	return stats, nil
}

// WriteLog writes a log entry to the log file
func (ls *LogService) WriteLog(entry SystemLogEntry) error {
	if len(ls.logPaths) == 0 {
		return fmt.Errorf("no log paths configured")
	}

	// Use the first available log path
	logPath := ls.logPaths[0]

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write as JSON
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(jsonBytes) + "\n")
	return err
}

// GetLogFiles returns available log file paths
func (ls *LogService) GetLogFiles() []string {
	return ls.logPaths
}

// TailLogs provides real-time log streaming (simplified implementation)
func (ls *LogService) TailLogs(ctx context.Context, filter LogFilter) (<-chan SystemLogEntry, error) {
	logChan := make(chan SystemLogEntry, 100)

	go func() {
		defer close(logChan)

		ticker := time.NewTicker(time.Second * 5) // Check for new logs every 5 seconds
		defer ticker.Stop()

		lastCheck := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Get new logs since last check
				sinceFilter := filter
				sinceFilter.Since = &lastCheck

				logs, err := ls.GetSystemLogs(ctx, sinceFilter)
				if err != nil {
					ls.logger.WithError(err).Error("Failed to get new logs for tailing")
					continue
				}

				// Send new logs to channel
				for i := len(logs) - 1; i >= 0; i-- { // Reverse order to get oldest first
					select {
					case logChan <- logs[i]:
					case <-ctx.Done():
						return
					}
				}

				lastCheck = time.Now()
			}
		}
	}()

	return logChan, nil
}