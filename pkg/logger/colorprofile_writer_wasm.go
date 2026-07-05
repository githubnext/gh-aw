//go:build js || wasm

package logger

import (
	"io"

	"github.com/github/gh-aw/pkg/colorwriter"
)

func stderrWriter() io.Writer {
	return colorwriter.Stderr()
}
