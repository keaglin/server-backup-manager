package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger is a simple logger that writes to both stdout and a log file
type Logger struct {
	stdLogger  *log.Logger
	fileLogger *log.Logger
	logFile    *os.File
}

// NewLogger creates a new logger that writes to both stdout and a log file
func NewLogger(logDir string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("backup_%s.log", timestamp))

	logFile, err := os.Create(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Create loggers
	stdLogger := log.New(os.Stdout, "", log.LstdFlags)
	fileLogger := log.New(logFile, "", log.LstdFlags)

	return &Logger{
		stdLogger:  stdLogger,
		fileLogger: fileLogger,
		logFile:    logFile,
	}, nil
}

// Info logs an informational message
func (l *Logger) Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.stdLogger.Printf("[INFO] %s", msg)
	l.fileLogger.Printf("[INFO] %s", msg)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.stdLogger.Printf("[ERROR] %s", msg)
	l.fileLogger.Printf("[ERROR] %s", msg)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
