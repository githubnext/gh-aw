//go:build !integration

package colorwriter_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/colorwriter"
)

// TestSpec_PublicAPI_New validates the documented behavior of New from the
// README.md specification.
func TestSpec_PublicAPI_New(t *testing.T) {
	t.Run("returns a usable writer wrapping the provided writer", func(t *testing.T) {
		var buf bytes.Buffer
		w := colorwriter.New(&buf, []string{"NO_COLOR=1"})
		require.NotNil(t, w, "New should return a non-nil io.Writer")

		n, err := io.WriteString(w, "spec output")
		require.NoError(t, err, "writer returned by New should accept writes")
		assert.Equal(t, len("spec output"), n, "writer should report bytes written")
		assert.Contains(t, buf.String(), "spec output", "writes should reach the underlying writer")
	})

	t.Run("accepts environment slices such as os.Environ", func(t *testing.T) {
		var buf bytes.Buffer
		w := colorwriter.New(&buf, os.Environ())
		require.NotNil(t, w, "New should accept os.Environ() as documented")

		_, err := io.WriteString(w, "env aware output")
		require.NoError(t, err, "writer returned by New should remain usable with os.Environ input")
		assert.Contains(t, buf.String(), "env aware output", "wrapped writer should forward output")
	})
}

// TestSpec_PublicAPI_Stderr validates the documented behavior of Stderr from the
// README.md specification.
func TestSpec_PublicAPI_Stderr(t *testing.T) {
	w := colorwriter.Stderr()
	require.NotNil(t, w, "Stderr should return a non-nil io.Writer")
	assert.Implements(t, (*io.Writer)(nil), w, "Stderr should return an io.Writer as documented")
}

// TestSpec_PublicAPI_Stdout validates the documented behavior of Stdout from the
// README.md specification.
func TestSpec_PublicAPI_Stdout(t *testing.T) {
	w := colorwriter.Stdout()
	require.NotNil(t, w, "Stdout should return a non-nil io.Writer")
	assert.Implements(t, (*io.Writer)(nil), w, "Stdout should return an io.Writer as documented")
}

// TestSpec_PublicAPI_Degrade validates the documented behavior of Degrade from the
// README.md specification.
func TestSpec_PublicAPI_Degrade(t *testing.T) {
	const ansiRed = "\x1b[31mhello\x1b[0m"

	tests := []struct {
		name    string
		environ []string
		want    string
	}{
		{
			name:    "strips ansi when NO_COLOR is set",
			environ: []string{"NO_COLOR=1", "TERM=xterm-256color"},
			want:    "hello",
		},
		{
			name:    "strips ansi for dumb terminals",
			environ: []string{"TERM=dumb"},
			want:    "hello",
		},
		{
			name:    "preserves ansi for forced color profiles",
			environ: []string{"TERM=xterm-256color", "CLICOLOR_FORCE=1"},
			want:    ansiRed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorwriter.Degrade(ansiRed, tt.environ)
			if tt.want == ansiRed {
				assert.Contains(t, got, "hello")
				assert.Contains(t, got, "\x1b[31m")
				assert.Contains(t, got, "\x1b[m")
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
