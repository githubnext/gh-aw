package logging

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with convenience methods for compiler logging
type Logger struct {
	*slog.Logger
	verbose bool
}

// NewLogger creates a new Logger instance
// If verbose is true, sets level to Debug, otherwise Info
func NewLogger(verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger:  slog.New(handler),
		verbose: verbose,
	}
}

// NewLoggerWithWriter creates a new Logger with custom output writer
// Useful for testing and capturing log output
func NewLoggerWithWriter(verbose bool, writer io.Writer) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger:  slog.New(handler),
		verbose: verbose,
	}
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	return l.verbose
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
