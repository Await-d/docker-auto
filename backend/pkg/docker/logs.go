package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

// LogLevel represents different log levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarn    LogLevel = "warn"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
	LogLevelPanic   LogLevel = "panic"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"` // stdout or stderr
	Raw       string    `json:"raw"`    // original log line
}

// LogFilter represents filters for log queries
type LogFilter struct {
	Since     *time.Time  `json:"since,omitempty"`
	Until     *time.Time  `json:"until,omitempty"`
	Level     *LogLevel   `json:"level,omitempty"`
	Contains  string      `json:"contains,omitempty"`
	Regex     string      `json:"regex,omitempty"`
	Source    string      `json:"source,omitempty"` // stdout, stderr, or both
	Tail      int         `json:"tail,omitempty"`   // number of lines from end
	Follow    bool        `json:"follow,omitempty"`
}

// LogOptions represents options for log operations
type LogOptions struct {
	ShowStdout    bool
	ShowStderr    bool
	ShowTimestamp bool
	Follow        bool
	Tail          string
	Since         string
	Until         string
	Details       bool
}

// Container log operations

// GetContainerLogs gets logs from a Docker container
func (d *DockerClient) GetContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	reader, err := d.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}

	return reader, nil
}

// GetContainerLogsAsString gets container logs as a string
func (d *DockerClient) GetContainerLogsAsString(ctx context.Context, containerID string, options types.ContainerLogsOptions) (string, error) {
	reader, err := d.GetContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	logs, err := ParseDockerLogs(reader)
	if err != nil {
		return "", err
	}

	return strings.Join(logs, "\n"), nil
}

// GetContainerLogsAsEntries gets container logs as structured log entries
func (d *DockerClient) GetContainerLogsAsEntries(ctx context.Context, containerID string, options types.ContainerLogsOptions) ([]LogEntry, error) {
	reader, err := d.GetContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return d.ParseLogEntries(reader, options.Timestamps)
}

// StreamContainerLogs streams container logs in real-time
func (d *DockerClient) StreamContainerLogs(ctx context.Context, containerID string, options types.ContainerLogsOptions) (<-chan string, <-chan error) {
	logChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(logChan)
		defer close(errChan)

		reader, err := d.GetContainerLogs(ctx, containerID, options)
		if err != nil {
			errChan <- err
			return
		}
		defer reader.Close()

		// Create separate readers for stdout and stderr if both are enabled
		if options.ShowStdout && options.ShowStderr {
			// Use stdcopy to demultiplex
			_, err = stdcopy.StdCopy(
				&writeToChannel{ch: logChan, prefix: "stdout"},
				&writeToChannel{ch: logChan, prefix: "stderr"},
				reader,
			)
			if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("failed to parse log stream: %w", err)
				return
			}
		} else {
			// Simple case - read line by line
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				case logChan <- scanner.Text():
				}
			}
			if err := scanner.Err(); err != nil {
				errChan <- fmt.Errorf("error reading logs: %w", err)
			}
		}
	}()

	return logChan, errChan
}

// StreamContainerLogsAsEntries streams container logs as structured entries
func (d *DockerClient) StreamContainerLogsAsEntries(ctx context.Context, containerID string, options types.ContainerLogsOptions) (<-chan LogEntry, <-chan error) {
	entryChan := make(chan LogEntry, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(entryChan)
		defer close(errChan)

		logChan, logErrChan := d.StreamContainerLogs(ctx, containerID, options)

		for {
			select {
			case <-ctx.Done():
				return
			case logLine, ok := <-logChan:
				if !ok {
					return
				}
				entry := d.parseLogLine(logLine, options.Timestamps)
				entryChan <- entry
			case err, ok := <-logErrChan:
				if !ok {
					return
				}
				errChan <- err
				return
			}
		}
	}()

	return entryChan, errChan
}

// GetLogsSince gets logs since a specific time
func (d *DockerClient) GetLogsSince(ctx context.Context, containerID string, since time.Time) ([]string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since.Format(time.RFC3339),
		Timestamps: true,
	}

	reader, err := d.GetContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ParseDockerLogs(reader)
}

// GetLogsWithFilter gets logs with advanced filtering
func (d *DockerClient) GetLogsWithFilter(ctx context.Context, containerID string, filter LogFilter) ([]LogEntry, error) {
	// Build container logs options
	options := types.ContainerLogsOptions{
		ShowStdout: filter.Source == "" || filter.Source == "stdout" || filter.Source == "both",
		ShowStderr: filter.Source == "" || filter.Source == "stderr" || filter.Source == "both",
		Timestamps: true,
		Follow:     filter.Follow,
	}

	if filter.Since != nil {
		options.Since = filter.Since.Format(time.RFC3339)
	}

	if filter.Until != nil {
		options.Until = filter.Until.Format(time.RFC3339)
	}

	if filter.Tail > 0 {
		options.Tail = fmt.Sprintf("%d", filter.Tail)
	}

	// Get logs
	entries, err := d.GetContainerLogsAsEntries(ctx, containerID, options)
	if err != nil {
		return nil, err
	}

	// Apply additional filters
	return d.applyLogFilters(entries, filter)
}

// Log parsing utilities

// ParseDockerLogs parses Docker log stream into individual lines
func ParseDockerLogs(logStream io.ReadCloser) ([]string, error) {
	var logs []string

	// Try to use stdcopy to separate stdout/stderr streams
	var stdout, stderr strings.Builder
	_, err := stdcopy.StdCopy(&stdout, &stderr, logStream)
	if err != nil {
		// If stdcopy fails, fall back to reading as plain text
		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			logs = append(logs, scanner.Text())
		}
		return logs, scanner.Err()
	}

	// Combine stdout and stderr
	stdoutLines := strings.Split(stdout.String(), "\n")
	stderrLines := strings.Split(stderr.String(), "\n")

	// Remove empty last line if present
	if len(stdoutLines) > 0 && stdoutLines[len(stdoutLines)-1] == "" {
		stdoutLines = stdoutLines[:len(stdoutLines)-1]
	}
	if len(stderrLines) > 0 && stderrLines[len(stderrLines)-1] == "" {
		stderrLines = stderrLines[:len(stderrLines)-1]
	}

	logs = append(logs, stdoutLines...)
	logs = append(logs, stderrLines...)

	return logs, nil
}

// ParseLogEntries parses Docker logs into structured log entries
func (d *DockerClient) ParseLogEntries(logStream io.ReadCloser, hasTimestamps bool) ([]LogEntry, error) {
	var entries []LogEntry

	// Use a custom writer to capture both stdout and stderr with source info
	parser := &logEntryParser{
		entries:       &entries,
		hasTimestamps: hasTimestamps,
	}

	_, err := stdcopy.StdCopy(
		&sourceWriter{parser: parser, source: "stdout"},
		&sourceWriter{parser: parser, source: "stderr"},
		logStream,
	)

	if err != nil {
		// Fallback to simple parsing
		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			entry := d.parseLogLine(scanner.Text(), hasTimestamps)
			entries = append(entries, entry)
		}
		if scanErr := scanner.Err(); scanErr != nil {
			return entries, scanErr
		}
	}

	return entries, nil
}

// parseLogLine parses a single log line into a LogEntry
func (d *DockerClient) parseLogLine(line string, hasTimestamp bool) LogEntry {
	entry := LogEntry{
		Raw:       line,
		Source:    "stdout", // default
		Timestamp: time.Now(),
		Level:     LogLevelInfo, // default
	}

	message := line

	// Parse timestamp if present
	if hasTimestamp && len(line) > 30 {
		timestampStr := line[:30]
		if timestamp, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
			entry.Timestamp = timestamp
			message = strings.TrimSpace(line[30:])
		}
	}

	// Try to detect log level from message
	entry.Level = d.detectLogLevel(message)
	entry.Message = message

	return entry
}

// detectLogLevel tries to detect log level from log message
func (d *DockerClient) detectLogLevel(message string) LogLevel {
	messageLower := strings.ToLower(message)

	levelPatterns := map[LogLevel][]string{
		LogLevelDebug:   {"debug", "dbg", "[debug]", "[dbg]"},
		LogLevelInfo:    {"info", "information", "[info]", "[inf]"},
		LogLevelWarn:    {"warn", "warning", "[warn]", "[warning]"},
		LogLevelError:   {"error", "err", "[error]", "[err]", "exception", "failed", "failure"},
		LogLevelFatal:   {"fatal", "[fatal]", "panic", "[panic]"},
	}

	for level, patterns := range levelPatterns {
		for _, pattern := range patterns {
			if strings.Contains(messageLower, pattern) {
				return level
			}
		}
	}

	return LogLevelInfo // default
}

// Filter utilities

// FilterLogsByLevel filters logs by level
func FilterLogsByLevel(logs []string, level string) []string {
	var filtered []string
	levelLower := strings.ToLower(level)

	for _, log := range logs {
		logLower := strings.ToLower(log)
		if strings.Contains(logLower, levelLower) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// FilterLogsByPattern filters logs by regex pattern
func FilterLogsByPattern(logs []string, pattern string) ([]string, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var filtered []string
	for _, log := range logs {
		if regex.MatchString(log) {
			filtered = append(filtered, log)
		}
	}

	return filtered, nil
}

// FilterLogsByContains filters logs that contain a specific string
func FilterLogsByContains(logs []string, contains string) []string {
	var filtered []string
	containsLower := strings.ToLower(contains)

	for _, log := range logs {
		if strings.Contains(strings.ToLower(log), containsLower) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// applyLogFilters applies various filters to log entries
func (d *DockerClient) applyLogFilters(entries []LogEntry, filter LogFilter) ([]LogEntry, error) {
	var filtered []LogEntry

	for _, entry := range entries {
		// Apply level filter
		if filter.Level != nil && entry.Level != *filter.Level {
			continue
		}

		// Apply contains filter
		if filter.Contains != "" {
			if !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(filter.Contains)) {
				continue
			}
		}

		// Apply regex filter
		if filter.Regex != "" {
			regex, err := regexp.Compile(filter.Regex)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern: %w", err)
			}
			if !regex.MatchString(entry.Message) {
				continue
			}
		}

		// Apply source filter
		if filter.Source != "" && filter.Source != "both" && entry.Source != filter.Source {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered, nil
}

// Log aggregation and analysis

// GetLogSummary gets a summary of log entries
func (d *DockerClient) GetLogSummary(ctx context.Context, containerID string, since time.Time) (*LogSummary, error) {
	entries, err := d.GetContainerLogsAsEntries(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      since.Format(time.RFC3339),
		Timestamps: true,
	})
	if err != nil {
		return nil, err
	}

	summary := &LogSummary{
		ContainerID: containerID,
		StartTime:   since,
		EndTime:     time.Now(),
		TotalLines:  len(entries),
		LevelCounts: make(map[LogLevel]int),
	}

	for _, entry := range entries {
		summary.LevelCounts[entry.Level]++

		if entry.Level == LogLevelError || entry.Level == LogLevelFatal {
			summary.ErrorCount++
		}
		if entry.Level == LogLevelWarn || entry.Level == LogLevelWarning {
			summary.WarningCount++
		}
	}

	return summary, nil
}

// LogSummary represents a summary of log analysis
type LogSummary struct {
	ContainerID  string             `json:"container_id"`
	StartTime    time.Time          `json:"start_time"`
	EndTime      time.Time          `json:"end_time"`
	TotalLines   int                `json:"total_lines"`
	ErrorCount   int                `json:"error_count"`
	WarningCount int                `json:"warning_count"`
	LevelCounts  map[LogLevel]int   `json:"level_counts"`
}

// Helper types for streaming

// writeToChannel is a helper to write to a channel
type writeToChannel struct {
	ch     chan<- string
	prefix string
}

func (w *writeToChannel) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			if w.prefix != "" {
				line = fmt.Sprintf("[%s] %s", w.prefix, line)
			}
			select {
			case w.ch <- line:
			default:
				// Channel is full, skip this line
			}
		}
	}
	return len(p), nil
}

// logEntryParser helps parse log entries with source information
type logEntryParser struct {
	entries       *[]LogEntry
	hasTimestamps bool
}

// sourceWriter writes to a log entry parser with source information
type sourceWriter struct {
	parser *logEntryParser
	source string
}

func (sw *sourceWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" {
			entry := LogEntry{
				Raw:       line,
				Source:    sw.source,
				Timestamp: time.Now(),
				Level:     LogLevelInfo,
			}

			message := line

			// Parse timestamp if present
			if sw.parser.hasTimestamps && len(line) > 30 {
				timestampStr := line[:30]
				if timestamp, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
					entry.Timestamp = timestamp
					message = strings.TrimSpace(line[30:])
				}
			}

			entry.Message = message
			*sw.parser.entries = append(*sw.parser.entries, entry)
		}
	}
	return len(p), nil
}

// Utility functions

// TailLogs gets the last N lines of logs
func (d *DockerClient) TailLogs(ctx context.Context, containerID string, lines int) ([]string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", lines),
		Timestamps: false,
	}

	reader, err := d.GetContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ParseDockerLogs(reader)
}

// SearchLogs searches for a pattern in container logs
func (d *DockerClient) SearchLogs(ctx context.Context, containerID, pattern string, maxLines int) ([]string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
	}

	if maxLines > 0 {
		options.Tail = fmt.Sprintf("%d", maxLines)
	}

	logs, err := d.GetContainerLogsAsString(ctx, containerID, options)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(logs, "\n")
	return FilterLogsByContains(lines, pattern), nil
}

// ExportLogs exports container logs to a writer
func (d *DockerClient) ExportLogs(ctx context.Context, containerID string, writer io.Writer, options types.ContainerLogsOptions) error {
	reader, err := d.GetContainerLogs(ctx, containerID, options)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(writer, reader)
	return err
}