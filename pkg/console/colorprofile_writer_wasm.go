//go:build js || wasm

package console

import (
	"io"

	"github.com/github/gh-aw/pkg/colorwriter"
)

func newColorProfileWriter(w io.Writer, environ []string) io.Writer {
	return colorwriter.New(w, environ)
}

func stderrWriter() io.Writer {
	return colorwriter.Stderr()
}
