//go:build !js && !wasm

package colorwriter

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

// New returns an io.Writer that adapts color output based on the provided
// environment variables (e.g. NO_COLOR, COLORTERM, TERM).
func New(w io.Writer, environ []string) io.Writer {
	return colorprofile.NewWriter(w, environ)
}

// Stderr returns a color-profile-aware writer for os.Stderr using the current
// process environment.
func Stderr() io.Writer {
	return New(os.Stderr, os.Environ())
}
