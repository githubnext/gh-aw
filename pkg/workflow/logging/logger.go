package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with convenience methods for compiler logging
type Logger struct {
	*slog.Logger
	verbose  bool
	category string
}

// NewLogger creates a new Logger instance
// If verbose is true, sets level to Debug, otherwise Info
func NewLogger(verbose bool) *Logger {
	return NewLoggerWithCategory(verbose, "")
}

// NewLoggerWithCategory creates a new Logger instance with a category
// Category is used for filtering logs via environment variables
// Set GH_AW_LOG_FILTER to a comma-separated list of categories to enable (e.g., "compiler,parser")
// Set GH_AW_LOG_FILTER to "all" to enable all categories
func NewLoggerWithCategory(verbose bool, category string) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	// Check if this category should be enabled based on environment variable
	enabled := isCategoryEnabled(category)
	if !enabled && category != "" {
		// If category is disabled, set level to a very high value to suppress all logs
		level = slog.Level(1000)
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	if category != "" {
		// Add category as a field to all log messages
		logger = logger.With("category", category)
	}

	return &Logger{
		Logger:   logger,
		verbose:  verbose,
		category: category,
	}
}

// NewLoggerWithWriter creates a new Logger with custom output writer
// Useful for testing and capturing log output
func NewLoggerWithWriter(verbose bool, writer io.Writer) *Logger {
	return NewLoggerWithWriterAndCategory(verbose, writer, "")
}

// NewLoggerWithWriterAndCategory creates a new Logger with custom output writer and category
// Useful for testing and capturing log output
func NewLoggerWithWriterAndCategory(verbose bool, writer io.Writer, category string) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	// Check if this category should be enabled based on environment variable
	enabled := isCategoryEnabled(category)
	if !enabled && category != "" {
		// If category is disabled, set level to a very high value to suppress all logs
		level = slog.Level(1000)
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	if category != "" {
		// Add category as a field to all log messages
		logger = logger.With("category", category)
	}

	return &Logger{
		Logger:   logger,
		verbose:  verbose,
		category: category,
	}
}

// isCategoryEnabled checks if a category is enabled based on GH_AW_LOG_FILTER environment variable
// Returns true if:
// - category is empty (default logger)
// - GH_AW_LOG_FILTER is not set (all categories enabled by default)
// - GH_AW_LOG_FILTER contains "all"
// - GH_AW_LOG_FILTER contains the category name
func isCategoryEnabled(category string) bool {
	if category == "" {
		return true // Default logger is always enabled
	}

	filter := os.Getenv("GH_AW_LOG_FILTER")
	if filter == "" {
		return true // If no filter is set, all categories are enabled
	}

	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "all" {
		return true
	}

	// Check if category is in the comma-separated list
	categories := strings.Split(filter, ",")
	categoryLower := strings.ToLower(strings.TrimSpace(category))
	for _, cat := range categories {
		if strings.TrimSpace(cat) == categoryLower {
			return true
		}
	}

	return false
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
}

// GetCategory returns the logger's category
func (l *Logger) GetCategory() string {
	return l.category
}

// Infof logs an info message with format string
func (l *Logger) Infof(format string, args ...any) {
	l.Logger.Info(format, args...)
}

// Debugf logs a debug message with format string
func (l *Logger) Debugf(format string, args ...any) {
	l.Logger.Debug(format, args...)
}

// Warnf logs a warning message with format string
func (l *Logger) Warnf(format string, args ...any) {
	l.Logger.Warn(format, args...)
}

// Errorf logs an error message with format string
func (l *Logger) Errorf(format string, args ...any) {
	l.Logger.Error(format, args...)
}

// InfoWithFields logs an info message with structured fields
func (l *Logger) InfoWithFields(msg string, fields ...any) {
	l.Logger.Info(msg, fields...)
}

// DebugWithFields logs a debug message with structured fields
func (l *Logger) DebugWithFields(msg string, fields ...any) {
	l.Logger.Debug(msg, fields...)
}

// WarnWithFields logs a warning message with structured fields
func (l *Logger) WarnWithFields(msg string, fields ...any) {
	l.Logger.Warn(msg, fields...)
}

// ErrorWithFields logs an error message with structured fields
func (l *Logger) ErrorWithFields(msg string, fields ...any) {
	l.Logger.Error(msg, fields...)
}
