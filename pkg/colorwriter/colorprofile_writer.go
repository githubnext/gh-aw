//go:build !js && !wasm

package colorwriter

import (
	"io"
	"os"
	"strings"

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

// Stdout returns a color-profile-aware writer for os.Stdout using the current
// process environment.
func Stdout() io.Writer {
	return New(os.Stdout, os.Environ())
}

// Degrade returns s with ANSI sequences downgraded (or stripped) according to
// the current process environment (NO_COLOR, COLORTERM, TERM). It is intended
// for use with string-returning format helpers: render the style first, then
// call Degrade so that the caller's output honors the color profile.
func Degrade(s string, environ []string) string {
	var buf strings.Builder
	w := colorprofile.NewWriter(&buf, environ)
	// colorprofile.Writer writes synchronously and does not buffer past Write,
	// and strings.Builder writes cannot fail, so a write error would indicate an
	// unexpected future behavior change; fall back to the original string then.
	if _, err := io.WriteString(w, s); err != nil {
		return s
	}
	return buf.String()
}
