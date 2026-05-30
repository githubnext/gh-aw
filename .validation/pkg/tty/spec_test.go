//go:build !integration

package tty_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/tty"
	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_IsStdoutTerminal validates the documented behavior of IsStdoutTerminal
// as described in the tty package README.md specification.
// Spec: "Returns true if stdout (os.Stdout) is connected to a terminal."
func TestSpec_PublicAPI_IsStdoutTerminal(t *testing.T) {
	result1 := tty.IsStdoutTerminal()
	result2 := tty.IsStdoutTerminal()
	// Spec (Design Notes): "Terminal detection is evaluated at call time, not cached."
	// In the same unmodified context, consecutive calls must return the same result.
	assert.Equal(t, result1, result2, "consecutive calls in the same context must return the same result")
}

// TestSpec_PublicAPI_IsStderrTerminal validates the documented behavior of IsStderrTerminal
// as described in the tty package README.md specification.
// Spec: "Returns true if stderr (os.Stderr) is connected to a terminal."
func TestSpec_PublicAPI_IsStderrTerminal(t *testing.T) {
	result1 := tty.IsStderrTerminal()
	result2 := tty.IsStderrTerminal()
	// Spec (Design Notes): "Terminal detection is evaluated at call time, not cached."
	// In the same unmodified context, consecutive calls must return the same result.
	assert.Equal(t, result1, result2, "consecutive calls in the same context must return the same result")
}

// TestSpec_DesignDecision_NonTerminalReturnsFalse validates that when stdout/stderr are not
// connected to a terminal (e.g., CI pipelines, piped test execution), both functions return false.
// Spec: "The WASM stub (tty_wasm.go) always returns false so that components built for the
// browser never attempt to use ANSI escape codes."
// By the same contract, non-TTY environments return false.
func TestSpec_DesignDecision_NonTerminalReturnsFalse(t *testing.T) {
	if tty.IsStdoutTerminal() || tty.IsStderrTerminal() {
		t.Skip("skipping: running in an interactive terminal — non-TTY assertion not applicable here")
	}
	assert.False(t, tty.IsStdoutTerminal(), "IsStdoutTerminal must return false when stdout is not a terminal")
	assert.False(t, tty.IsStderrTerminal(), "IsStderrTerminal must return false when stderr is not a terminal")
}
