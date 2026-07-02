//go:build js || wasm

package console

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
