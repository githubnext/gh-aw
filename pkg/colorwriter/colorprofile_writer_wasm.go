//go:build js || wasm

package colorwriter

import (
	"io"
	"os"
)

// New returns w unchanged; color-profile detection is not supported on wasm.
func New(w io.Writer, _ []string) io.Writer {
	return w
}

// Stderr returns os.Stderr directly; color-profile detection is not supported
// on wasm.
func Stderr() io.Writer {
	return os.Stderr
}

// Stdout returns os.Stdout directly; color-profile detection is not supported
// on wasm.
func Stdout() io.Writer {
	return os.Stdout
}

// Degrade returns s unchanged; color-profile detection is not supported on wasm.
func Degrade(s string, _ []string) string {
	return s
}
