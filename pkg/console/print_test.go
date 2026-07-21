//go:build !integration

package console

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stderr writer: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stderr output: %v", err)
	}
	return string(out)
}

func TestPrintMessageHelpers(t *testing.T) {
	stderr := captureStderr(t, func() {
		_, _ = PrintSuccessMessage("ok")
		_, _ = PrintInfoMessage("info")
		_, _ = PrintWarningMessage("warn")
		_, _ = PrintErrorMessage("boom")
		_, _ = PrintCommandMessage("gh aw status")
		_, _ = PrintSectionHeader("Header")
	})

	expected := []string{
		FormatSuccessMessage("ok") + "\n",
		FormatInfoMessage("info") + "\n",
		FormatWarningMessage("warn") + "\n",
		FormatErrorMessage("boom") + "\n",
		FormatCommandMessage("gh aw status") + "\n",
		FormatSectionHeader("Header") + "\n",
	}
	for _, want := range expected {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected stderr to contain line %q, got %q", want, stderr)
		}
	}
}

func TestPrintErrorChain(t *testing.T) {
	err := errors.New("inner")
	wrapped := fmt.Errorf("outer: %w", err)

	stderr := captureStderr(t, func() {
		_, _ = PrintErrorChain(wrapped)
	})

	want := FormatErrorChain(wrapped) + "\n"
	if stderr != want {
		t.Fatalf("expected %q, got %q", want, stderr)
	}
}
