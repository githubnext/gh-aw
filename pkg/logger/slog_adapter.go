package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// SlogHandler implements slog.Handler by wrapping a gh-aw Logger
// This allows integration with libraries that expect slog.Logger
type SlogHandler struct {
	logger *Logger
}

// NewSlogHandler creates a new slog.Handler that wraps a gh-aw Logger
func NewSlogHandler(logger *Logger) *SlogHandler {
	return &SlogHandler{logger: logger}
}

// Enabled reports whether the handler handles records at the given level.
// We enable all levels when our logger is enabled.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.Enabled()
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
func (h *SlogHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.logger.Enabled() {
		return nil
	}

	// Format the message with attributes
	msg := r.Message
	if r.NumAttrs() > 0 {
		attrs := make([]any, 0, r.NumAttrs()*2)
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a.Key, a.Value)
			return true
		})
		// Format attributes as key=value pairs
		for i := 0; i < len(attrs); i += 2 {
			// Safely handle non-string keys (defensive programming - slog always provides string keys)
			key, ok := attrs[i].(string)
			if !ok {
				key = fmt.Sprint(attrs[i])
			}
			msg += " " + key + "=" + formatSlogValue(attrs[i+1])
		}
	}

	// Log with appropriate level prefix
	levelPrefix := ""
	switch r.Level {
	case slog.LevelDebug:
		levelPrefix = "[DEBUG] "
	case slog.LevelInfo:
		levelPrefix = "[INFO] "
	case slog.LevelWarn:
		levelPrefix = "[WARN] "
	case slog.LevelError:
		levelPrefix = "[ERROR] "
	}

	h.logger.Print(levelPrefix + msg)
	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// This implementation does not persist attributes.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// This implementation does not persist groups.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	return h
}

// formatSlogValue formats an slog.Value for display
func formatSlogValue(v any) string {
	if val, ok := v.(slog.Value); ok {
		return val.String()
	}
	return slog.AnyValue(v).String()
}

// NewSlogLogger creates a new slog.Logger that uses gh-aw's logger package
// This allows integration with libraries that expect slog.Logger
func NewSlogLogger(namespace string) *slog.Logger {
	logger := New(namespace)
	handler := NewSlogHandler(logger)
	return slog.New(handler)
}

// NewSlogLoggerWithHandler creates a new slog.Logger using an existing Logger instance
func NewSlogLoggerWithHandler(logger *Logger) *slog.Logger {
	handler := NewSlogHandler(logger)
	return slog.New(handler)
}

// Discard returns a slog.Logger that discards all output
func Discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
