// Package logging provides structured logging for mycelicmemory.
//
// This package wraps Go's log/slog package to provide consistent,
// structured logging across all mycelicmemory components.
//
// Usage:
//
//	import "github.com/MycelicMemory/mycelicmemory/internal/logging"
//
//	// Initialize once at startup
//	logging.Init(logging.Config{
//	    Level:  "info",
//	    Format: "json",
//	    Output: "stderr",
//	})
//
//	// Get a logger for a component
//	log := logging.GetLogger("mcp")
//
//	// Log with context
//	log.Info("request received", "method", "tools/call", "tool", "store_memory")
//	log.Error("operation failed", "error", err, "memory_id", id)
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Config holds logging configuration
type Config struct {
	// Level is the minimum log level: debug, info, warn, error
	Level string
	// Format is the output format: console, json
	Format string
	// Output is the output destination: stderr, stdout, or a file path
	Output string
}

var (
	defaultLogger *slog.Logger
	loggerMu      sync.RWMutex
	initialized   bool
)

func init() {
	// Initialize with default console logger
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Init initializes the global logger with the given configuration.
// This should be called once at application startup.
func Init(cfg Config) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	var output io.Writer
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		output = os.Stdout
	case "", "stderr":
		output = os.Stderr
	default:
		// Try to open as file, fall back to stderr
		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			output = os.Stderr
		} else {
			output = f
		}
	}

	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level: level,
		// Add source location for debug level
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	initialized = true
}

// parseLevel converts a string level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger returns a logger for the specified component.
// The component name is added as an attribute to all log entries.
func GetLogger(component string) *Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return &Logger{
		slog:      defaultLogger.With("component", component),
		component: component,
	}
}

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	slog      *slog.Logger
	component string
}

// With returns a new Logger with the given attributes added
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog:      l.slog.With(args...),
		component: l.component,
	}
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// LogRequest logs an incoming request
func (l *Logger) LogRequest(method string, args ...any) {
	allArgs := append([]any{"method", method}, args...)
	l.slog.Info("request", allArgs...)
}

// LogResponse logs an outgoing response
func (l *Logger) LogResponse(method string, duration float64, args ...any) {
	allArgs := append([]any{"method", method, "duration_ms", duration}, args...)
	l.slog.Info("response", allArgs...)
}

// LogError logs an error with context
func (l *Logger) LogError(operation string, err error, args ...any) {
	allArgs := append([]any{"operation", operation, "error", err.Error()}, args...)
	l.slog.Error("operation_failed", allArgs...)
}

// LogOperation logs a successful operation
func (l *Logger) LogOperation(operation string, args ...any) {
	allArgs := append([]any{"operation", operation}, args...)
	l.slog.Info("operation_success", allArgs...)
}

// Convenience functions for package-level logging

// Debug logs at debug level using the default logger
func Debug(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	defaultLogger.Debug(msg, args...)
}

// Info logs at info level using the default logger
func Info(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	defaultLogger.Info(msg, args...)
}

// Warn logs at warn level using the default logger
func Warn(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	defaultLogger.Warn(msg, args...)
}

// Error logs at error level using the default logger
func Error(msg string, args ...any) {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	defaultLogger.Error(msg, args...)
}
