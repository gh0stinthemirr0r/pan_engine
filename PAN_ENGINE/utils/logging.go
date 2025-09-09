package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

// InitLogging sets up the logging configuration
func InitLogging() error {
	// Create logs directory if it doesn't exist
	logsDir := "Logging"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logsDir, fmt.Sprintf("app_%s.log", timestamp))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Create loggers with different prefixes
	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(file, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Also write to stdout for development
	InfoLogger.SetOutput(os.Stdout)
	ErrorLogger.SetOutput(os.Stdout)
	DebugLogger.SetOutput(os.Stdout)

	InfoLogger.Printf("Logging initialized - Log file: %s", logFile)
	return nil
}

// EnsureDirectories creates necessary directories if they don't exist
func EnsureDirectories() error {
	dirs := []string{
		"Reports",
		"Logging",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
		if InfoLogger != nil {
			InfoLogger.Printf("Ensured directory exists: %s", dir)
		}
	}

	return nil
}
