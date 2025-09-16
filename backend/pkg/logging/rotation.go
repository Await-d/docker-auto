package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotatingFileWriter implements log rotation
type RotatingFileWriter struct {
	mu          sync.Mutex
	filename    string
	maxSize     int64 // bytes
	maxBackups  int
	maxAge      int // days
	compress    bool
	currentFile *os.File
	currentSize int64
}

// NewRotatingFileWriter creates a new rotating file writer
func NewRotatingFileWriter(filename string, maxSize int, maxBackups int, maxAge int, compress bool) (*RotatingFileWriter, error) {
	rfw := &RotatingFileWriter{
		filename:   filename,
		maxSize:    int64(maxSize) * 1024 * 1024, // Convert MB to bytes
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open initial file
	if err := rfw.openFile(); err != nil {
		return nil, err
	}

	return rfw, nil
}

// Write implements io.Writer interface
func (rfw *RotatingFileWriter) Write(data []byte) (int, error) {
	rfw.mu.Lock()
	defer rfw.mu.Unlock()

	// Check if rotation is needed
	if rfw.currentSize+int64(len(data)) > rfw.maxSize {
		if err := rfw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rfw.currentFile.Write(data)
	rfw.currentSize += int64(n)
	return n, err
}

// Close closes the current file
func (rfw *RotatingFileWriter) Close() error {
	rfw.mu.Lock()
	defer rfw.mu.Unlock()

	if rfw.currentFile != nil {
		return rfw.currentFile.Close()
	}
	return nil
}

// openFile opens the log file
func (rfw *RotatingFileWriter) openFile() error {
	file, err := os.OpenFile(rfw.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	rfw.currentFile = file
	rfw.currentSize = stat.Size()
	return nil
}

// rotate performs log rotation
func (rfw *RotatingFileWriter) rotate() error {
	// Close current file
	if rfw.currentFile != nil {
		rfw.currentFile.Close()
		rfw.currentFile = nil
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	backupName := fmt.Sprintf("%s.%s", rfw.filename, timestamp)

	// Rename current file to backup
	if err := os.Rename(rfw.filename, backupName); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Compress backup if configured
	if rfw.compress {
		go rfw.compressFile(backupName)
	}

	// Clean up old files
	go rfw.cleanup()

	// Open new file
	return rfw.openFile()
}

// compressFile compresses a log file
func (rfw *RotatingFileWriter) compressFile(filename string) {
	defer func() {
		if r := recover(); r != nil {
			// Log error but don't crash
			fmt.Fprintf(os.Stderr, "Failed to compress log file %s: %v\n", filename, r)
		}
	}()

	// Open source file
	src, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file for compression: %v\n", err)
		return
	}
	defer src.Close()

	// Create compressed file
	compressedName := filename + ".gz"
	dst, err := os.Create(compressedName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create compressed file: %v\n", err)
		return
	}
	defer dst.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()

	// Copy data
	_, err = io.Copy(gzWriter, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compress file: %v\n", err)
		os.Remove(compressedName)
		return
	}

	// Close gzip writer to flush
	gzWriter.Close()
	dst.Close()

	// Remove original file
	os.Remove(filename)
}

// cleanup removes old log files based on maxBackups and maxAge
func (rfw *RotatingFileWriter) cleanup() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Failed to cleanup log files: %v\n", r)
		}
	}()

	dir := filepath.Dir(rfw.filename)
	baseName := filepath.Base(rfw.filename)

	// Get all backup files
	files, err := filepath.Glob(filepath.Join(dir, baseName+".*"))
	if err != nil {
		return
	}

	// Filter and sort backup files
	var backups []os.FileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Skip if it's the current log file
		if file == rfw.filename {
			continue
		}

		backups = append(backups, &fileInfoWithPath{info, file})
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime().After(backups[j].ModTime())
	})

	// Remove files based on maxAge
	if rfw.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -rfw.maxAge)
		for _, backup := range backups {
			if backup.ModTime().Before(cutoff) {
				os.Remove(backup.(*fileInfoWithPath).path)
			}
		}

		// Re-filter backups
		var validBackups []os.FileInfo
		for _, backup := range backups {
			if backup.ModTime().After(cutoff) {
				validBackups = append(validBackups, backup)
			}
		}
		backups = validBackups
	}

	// Remove excess files based on maxBackups
	if rfw.maxBackups > 0 && len(backups) > rfw.maxBackups {
		for _, backup := range backups[rfw.maxBackups:] {
			os.Remove(backup.(*fileInfoWithPath).path)
		}
	}
}

// fileInfoWithPath holds file info with its path
type fileInfoWithPath struct {
	os.FileInfo
	path string
}

// LogRotationConfig represents log rotation configuration
type LogRotationConfig struct {
	Enabled    bool `json:"enabled"`
	MaxSize    int  `json:"max_size"`    // MB
	MaxBackups int  `json:"max_backups"`
	MaxAge     int  `json:"max_age"`     // days
	Compress   bool `json:"compress"`
}

// CreateRotatingWriter creates a rotating writer based on configuration
func CreateRotatingWriter(filename string, config LogRotationConfig) (io.Writer, error) {
	if !config.Enabled {
		// Ensure directory exists
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}

	return NewRotatingFileWriter(
		filename,
		config.MaxSize,
		config.MaxBackups,
		config.MaxAge,
		config.Compress,
	)
}

// isBackupFile checks if a filename is a backup file
func isBackupFile(filename, baseName string) bool {
	if !strings.HasPrefix(filename, baseName+".") {
		return false
	}

	suffix := strings.TrimPrefix(filename, baseName+".")
	if strings.HasSuffix(suffix, ".gz") {
		suffix = strings.TrimSuffix(suffix, ".gz")
	}

	// Check if suffix matches timestamp pattern (YYYY-MM-DD-HH-MM-SS)
	parts := strings.Split(suffix, "-")
	return len(parts) == 6
}