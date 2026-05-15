package observability

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"
)

// Logger handles structured JSON logging
type Logger struct {
	enabled bool
	logger  *log.Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewLogger creates a new structured logger
func NewLogger(enabled bool) *Logger {
	return &Logger{
		enabled: enabled,
		logger:  log.New(os.Stdout, "", 0),
	}
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.log("INFO", message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.log("ERROR", message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.log("WARN", message, fields)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.log("DEBUG", message, fields)
}

// log is the internal logging method
func (l *Logger) log(level, message string, fields map[string]interface{}) {
	if !l.enabled {
		// If logging is disabled, only log errors to stderr
		if level == "ERROR" {
			l.logger.Printf("[ERROR] %s", message)
		}
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		l.logger.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonBytes))
}

// ShutdownHandler manages graceful shutdown
type ShutdownHandler struct {
	spooler         SpoolerInterface
	shutdownTimeout time.Duration
	stopSignal      chan os.Signal
}

// SpoolerInterface defines the spooler methods needed for shutdown
type SpoolerInterface interface {
	Drain(timeout time.Duration) bool
	Stop(timeout time.Duration)
}

// NewShutdownHandler creates a new shutdown handler
func NewShutdownHandler(spooler SpoolerInterface, timeout time.Duration) *ShutdownHandler {
	return &ShutdownHandler{
		spooler:         spooler,
		shutdownTimeout: timeout,
		stopSignal:      make(chan os.Signal, 1),
	}
}

// WaitForSignal blocks until a shutdown signal is received
func (h *ShutdownHandler) WaitForSignal(signals ...os.Signal) {
	signal.Notify(h.stopSignal, signals...)
	<-h.stopSignal
}

// Shutdown initiates graceful shutdown
func (h *ShutdownHandler) Shutdown() {
	log.Println("Shutdown signal received. Initiating graceful shutdown...")

	// Drain the spooler
	log.Printf("Waiting for spooler to drain (timeout: %v)...", h.shutdownTimeout)
	drained := h.spooler.Drain(h.shutdownTimeout)

	if !drained {
		log.Printf("WARNING: Spooler did not drain completely within timeout")
		log.Printf("Some print jobs may not have completed")
	} else {
		log.Printf("Spooler drained successfully")
	}

	// Stop the spooler
	h.spooler.Stop(h.shutdownTimeout)

	log.Println("Graceful shutdown complete. Exiting...")
}
