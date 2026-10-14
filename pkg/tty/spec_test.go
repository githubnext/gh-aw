//go:build !integration

package tty_test

import (
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/tty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_IsStdinTerminal validates the documented behavior of IsStdinTerminal
// as described by the tty package README.md specification.
// Spec: "Returns true if stdin (os.Stdin) is connected to a terminal."
func TestSpec_PublicAPI_IsStdinTerminal(t *testing.T) {
	testNonTerminalStream(t, &os.Stdin, tty.IsStdinTerminal)
}

// TestSpec_PublicAPI_IsStdoutTerminal validates the documented behavior of IsStdoutTerminal.
func TestSpec_PublicAPI_IsStdoutTerminal(t *testing.T) {
	testNonTerminalStream(t, &os.Stdout, tty.IsStdoutTerminal)
}

// TestSpec_PublicAPI_IsStderrTerminal validates the documented behavior of IsStderrTerminal.
func TestSpec_PublicAPI_IsStderrTerminal(t *testing.T) {
	testNonTerminalStream(t, &os.Stderr, tty.IsStderrTerminal)
}

func testNonTerminalStream(t *testing.T, stream **os.File, detect func() bool) {
	t.Helper()
	original := *stream
	reader, writer, err := os.Pipe()
	require.NoError(t, err, "creating a non-terminal pipe should succeed")
	t.Cleanup(func() {
		*stream = original
		require.NoError(t, reader.Close(), "closing the pipe reader should succeed")
		require.NoError(t, writer.Close(), "closing the pipe writer should succeed")
	})

	*stream = reader
	assert.False(t, detect(), "a standard stream redirected to a pipe must not be reported as a terminal")
}
