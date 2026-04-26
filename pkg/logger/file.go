package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EnableFileLogging enables file logging to .gollum/logs/<sessionID>.log.
// It creates the logs directory, cleans up old logs if needed, and opens the log file.
func (s *service) EnableFileLogging(gollumDir string, sessionID uuid.UUID) error {
	// Create logs directory
	logsDir := filepath.Join(gollumDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Clean up old log files if configured
	maxFiles := s.configService.GetLoggingConfig().MaxSessionLogFiles
	if maxFiles > 0 {
		if err := s.cleanupOldLogs(logsDir, maxFiles); err != nil {
			// Log but don't fail - cleanup is best effort
			s.Warnf("failed to cleanup old logs: %v", err)
		}
	}

	// Create log file path
	logFilePath := filepath.Join(logsDir, sessionID.String()+".log")

	// Open file for writing
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	s.logFile = file
	s.logFilePath = logFilePath

	// Create a WriteSyncer with automatic locking and immediate syncing
	// Using zapcore.AddSync with Lock ensures thread-safe writes
	// We'll sync after each write for maximum data safety
	fileWriteSyncer := zapcore.Lock(zapcore.AddSync(file))

	// Create a separate file-only logger with increased sampling for better performance
	// but with immediate sync behavior
	// Use JSON encoder for structured logging
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(encoder, fileWriteSyncer, s.atomicLevel)

	// Create the file logger with options for better caller information
	s.fileLogger = zap.New(fileCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Note: We do NOT add fileCore to the main logger's Tee anymore.
	// The fileLogger is called explicitly in the Info/Error/Debug/Warn methods,
	// which prevents double-logging and gives us better control.

	s.Infof("Session logging enabled: %s", logFilePath)

	// Force immediate sync to ensure the first log is written to disk
	if err := s.fileLogger.Sync(); err != nil {
		s.Warnf("failed to sync file logger after enabling: %v", err)
	}
	if err := file.Sync(); err != nil {
		s.Warnf("failed to sync log file after enabling: %v", err)
	}

	return nil
}

// CloseFileLogging closes the current log file if open.
func (s *service) CloseFileLogging() error {
	if s.logFile == nil {
		return nil
	}

	// Sync and close the file
	if err := s.logFile.Sync(); err != nil {
		s.Warnf("failed to sync log file: %v", err)
	}

	if err := s.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	s.logFile = nil
	s.logFilePath = ""
	s.fileLogger = nil // Clear the file logger
	return nil
}

// Flush forces an immediate sync of the log file to disk.
// This ensures all buffered log entries are written to the file system.
func (s *service) Flush() error {
	if s.fileLogger == nil || s.logFile == nil {
		return nil // No file logging enabled
	}

	// Sync both the file logger and the underlying file
	if err := s.fileLogger.Sync(); err != nil {
		return fmt.Errorf("failed to sync file logger: %w", err)
	}
	if err := s.logFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync log file: %w", err)
	}

	return nil
}

// cleanupOldLogs removes the oldest log files when the count exceeds maxFiles.
func (s *service) cleanupOldLogs(logsDir string, maxFiles int) error {
	// Read all log files
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	// Filter for .log files and get their info
	type logFileInfo struct {
		name    string
		modTime time.Time
	}
	var logFiles []logFileInfo

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			info, err := entry.Info()
			if err != nil {
				continue // Skip files we can't read
			}
			logFiles = append(logFiles, logFileInfo{
				name:    entry.Name(),
				modTime: info.ModTime(),
			})
		}
	}

	// Check if cleanup is needed
	if len(logFiles) < maxFiles {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].modTime.Before(logFiles[j].modTime)
	})

	// Delete oldest files until we're under the limit
	filesToDelete := len(logFiles) - maxFiles + 1 // +1 because we're about to create a new one
	for i := 0; i < filesToDelete && i < len(logFiles); i++ {
		filePath := filepath.Join(logsDir, logFiles[i].name)
		if err := os.Remove(filePath); err != nil {
			s.Warnf("failed to remove old log file %s: %v", filePath, err)
		} else {
			s.Debugf("removed old log file: %s", filePath)
		}
	}

	return nil
}
