//go:build js || wasm

package logger

import (
	"io"
	"os"
)

func newColorProfileWriter(w io.Writer, _ []string) io.Writer {
	return w
}

func stderrWriter() io.Writer {
	return os.Stderr
}
