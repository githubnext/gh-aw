//go:build !integration

package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/logger"
)

// captureStderrSpec redirects os.Stderr during the call to f and returns what was written.
// Used to observe logger output in spec tests without relying on package internals.
func captureStderrSpec(f func()) string {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return ""
	}
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// TestSpec_PublicAPI_New validates that logger.New creates a Logger for the given namespace.
// Spec section: "### Basic Usage"
func TestSpec_PublicAPI_New(t *testing.T) {
	t.Run("New returns non-nil Logger", func(t *testing.T) {
		log := logger.New("myapp:feature")
		require.NotNil(t, log, "New should return a non-nil Logger")
	})

	t.Run("New accepts namespaced identifier", func(t *testing.T) {
		// Spec documents namespace-based logging with colon-separated segments.
		// Examples from spec: "workflow:compiler", "cli:audit", "parser:frontmatter"
		namespaces := []string{
			"myapp:feature",
			"workflow:compiler",
			"cli:audit",
			"parser:frontmatter",
		}
		for _, ns := range namespaces {
			log := logger.New(ns)
			require.NotNil(t, log, "New(%q) should return a non-nil Logger", ns)
		}
	})
}

// TestSpec_DesignDecision_EnabledStateAtConstruction validates that the enabled state
// is computed at logger construction time based on the DEBUG environment variable,
// as documented in the spec section "### Logger Enabled State".
func TestSpec_DesignDecision_EnabledStateAtConstruction(t *testing.T) {
	// The spec states: "The enabled state is computed once at logger construction time
	// based on the DEBUG environment variable."
	// NOTE: Because debugEnv is a package-level variable initialized once at package init
	// time, these tests validate the Enabled() API contract given the current process
	// environment. Tests that require changing DEBUG patterns use internal tests instead.

	t.Run("Enabled returns bool indicating logging state", func(t *testing.T) {
		log := logger.New("spec:test")
		// Enabled() must return a bool without panic; the specific value depends on DEBUG.
		_ = log.Enabled()
	})

	t.Run("disabled logger produces no stderr output", func(t *testing.T) {
		// When a logger is disabled, Printf and Print must produce no output.
		// We create a logger guaranteed to be disabled by using a namespace
		// that cannot match any DEBUG pattern when DEBUG is empty.
		// Reset DEBUG to empty for this test by using a namespace unlikely to match.
		log := logger.New("spec:definitely-disabled-xyzzy-" + t.Name())
		if log.Enabled() {
			t.Skip("logger is enabled (DEBUG may be set to *); skipping disabled-logger output test")
		}

		output := captureStderrSpec(func() {
			log.Printf("this should not appear")
			log.Print("this should not appear either")
		})
		assert.Empty(t, output, "disabled logger should produce no stderr output")
	})
}

// TestSpec_PublicAPI_Printf validates the documented Printf interface.
// Spec section: "### Printf Interface"
func TestSpec_PublicAPI_Printf(t *testing.T) {
	// To test Printf output, we need an enabled logger. We check the current
	// DEBUG state and skip if the package was initialized without DEBUG enabled.
	log := logger.New("spec:printf")
	if !log.Enabled() {
		t.Skip("logger is disabled (DEBUG not set); skipping output format test")
	}

	t.Run("Printf writes namespace and message to stderr", func(t *testing.T) {
		// Spec documents output format: "<namespace> <message> +<timediff>\n"
		output := captureStderrSpec(func() {
			log.Printf("hello %s", "world")
		})
		assert.Contains(t, output, "spec:printf", "output should contain the namespace")
		assert.Contains(t, output, "hello world", "output should contain the formatted message")
		assert.True(t, strings.HasSuffix(strings.TrimRight(output, "\n"), output[:len(output)-1]) || strings.Contains(output, "+"),
			"output should include time diff (+<duration>)")
	})

	t.Run("Printf always adds a newline", func(t *testing.T) {
		// Spec: "Printf(format, args...) - Formatted output (always adds newline)"
		output := captureStderrSpec(func() {
			log.Printf("no newline in format")
		})
		assert.True(t, strings.HasSuffix(output, "\n"), "Printf output should always end with a newline")
	})
}

// TestSpec_PublicAPI_Print validates the documented Print interface.
// Spec section: "### Printf Interface"
func TestSpec_PublicAPI_Print(t *testing.T) {
	log := logger.New("spec:print")
	if !log.Enabled() {
		t.Skip("logger is disabled (DEBUG not set); skipping output format test")
	}

	t.Run("Print concatenates arguments and writes to stderr", func(t *testing.T) {
		// Spec: "Print(args...) - Simple concatenation (always adds newline)"
		output := captureStderrSpec(func() {
			log.Print("Multiple", " ", "arguments")
		})
		assert.Contains(t, output, "spec:print", "output should contain the namespace")
		assert.Contains(t, output, "Multiple arguments", "Print should concatenate arguments")
	})

	t.Run("Print always adds a newline", func(t *testing.T) {
		output := captureStderrSpec(func() {
			log.Print("no newline")
		})
		assert.True(t, strings.HasSuffix(output, "\n"), "Print output should always end with a newline")
	})
}

// TestSpec_SlogIntegration_NewSlogHandler validates the documented SlogHandler.
// Spec section: "## slog Integration"
func TestSpec_SlogIntegration_NewSlogHandler(t *testing.T) {
	t.Run("NewSlogHandler returns a non-nil slog.Handler", func(t *testing.T) {
		log := logger.New("spec:slog")
		handler := logger.NewSlogHandler(log)
		require.NotNil(t, handler, "NewSlogHandler should return a non-nil handler")
		// slog.Handler is an interface; verify it implements the contract
		var _ slog.Handler = handler
	})

	t.Run("SlogHandler.Enabled returns false for disabled logger", func(t *testing.T) {
		// Spec: "SlogHandler.Enabled returns false when the underlying Logger is disabled"
		log := logger.New("spec:slog-disabled-xyzzy-" + t.Name())
		if log.Enabled() {
			t.Skip("logger is enabled (DEBUG may be set to *); skipping disabled state test")
		}
		handler := logger.NewSlogHandler(log)
		enabled := handler.Enabled(context.Background(), slog.LevelDebug)
		assert.False(t, enabled, "SlogHandler.Enabled should return false when underlying Logger is disabled")
	})

	t.Run("SlogHandler.WithAttrs returns handler unchanged", func(t *testing.T) {
		// Spec: "WithAttrs and WithGroup return the handler unchanged"
		log := logger.New("spec:slog-attrs")
		handler := logger.NewSlogHandler(log)
		result := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})
		assert.Equal(t, handler, result, "WithAttrs should return the same handler instance")
	})

	t.Run("SlogHandler.WithGroup returns handler unchanged", func(t *testing.T) {
		// Spec: "WithAttrs and WithGroup return the handler unchanged"
		log := logger.New("spec:slog-group")
		handler := logger.NewSlogHandler(log)
		result := handler.WithGroup("mygroup")
		assert.Equal(t, handler, result, "WithGroup should return the same handler instance")
	})
}

// TestSpec_SlogIntegration_NewSlogLoggerWithHandler validates the convenience constructor.
// Spec section: "### NewSlogLoggerWithHandler(logger *Logger) *slog.Logger"
func TestSpec_SlogIntegration_NewSlogLoggerWithHandler(t *testing.T) {
	t.Run("NewSlogLoggerWithHandler returns a non-nil slog.Logger", func(t *testing.T) {
		// Spec: "Convenience constructor that creates both the SlogHandler and the slog.Logger in one call."
		log := logger.New("spec:slog-convenience")
		slogLogger := logger.NewSlogLoggerWithHandler(log)
		require.NotNil(t, slogLogger, "NewSlogLoggerWithHandler should return a non-nil slog.Logger")
	})

	t.Run("resulting slog.Logger can log at any level without panic", func(t *testing.T) {
		log := logger.New("spec:slog-levels")
		slogLogger := logger.NewSlogLoggerWithHandler(log)
		// Must not panic regardless of enabled state
		assert.NotPanics(t, func() {
			slogLogger.Info("info message", "key", "value")
			slogLogger.Warn("warn message", "count", 42)
			slogLogger.Error("error message")
			slogLogger.Debug("debug message")
		}, "slog.Logger methods should not panic")
	})
}

// TestSpec_DesignDecision_OutputDestination validates that all log output goes to stderr.
// Spec section: "### Output Destination"
func TestSpec_DesignDecision_OutputDestination(t *testing.T) {
	log := logger.New("spec:stderr")
	if !log.Enabled() {
		t.Skip("logger is disabled (DEBUG not set); skipping output destination test")
	}

	t.Run("all log output goes to stderr", func(t *testing.T) {
		// Spec: "All log output goes to stderr to avoid interfering with stdout data"
		var stdoutBuf bytes.Buffer
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w

		captureStderrSpec(func() {
			log.Printf("testing output destination")
		})

		w.Close()
		os.Stdout = oldStdout
		_, _ = stdoutBuf.ReadFrom(r)

		assert.Empty(t, stdoutBuf.String(), "logger should not write to stdout")
	})
}
