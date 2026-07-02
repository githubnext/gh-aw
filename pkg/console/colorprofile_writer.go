//go:build !js && !wasm

package console

import (
	"io"
	"os"

	"github.com/charmbracelet/colorprofile"
)

func newColorProfileWriter(w io.Writer, environ []string) io.Writer {
	return colorprofile.NewWriter(w, environ)
}

func stderrWriter() io.Writer {
	return newColorProfileWriter(os.Stderr, os.Environ())
}
