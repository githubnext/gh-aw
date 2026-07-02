//go:build !integration && !js && !wasm

package console

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestColorProfileWriterStripsANSIWithNoColor(t *testing.T) {
	var buf bytes.Buffer
	w := newColorProfileWriter(&buf, []string{"NO_COLOR=1", "TERM=xterm-256color"})

	if _, err := fmt.Fprint(w, "\x1b[38;2;255;0;0mhello\x1b[0m"); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected ANSI-free output with NO_COLOR, got %q", buf.String())
	}
}
