//go:build !integration

package console

import (
	"errors"
	"fmt"
	"io"
	"os"
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

// TestPrintErrorNewline verifies that PrintError does not emit a spurious blank
// line: FormatError already terminates with \n, so Fprint (not Fprintln) must
// be used.
func TestPrintErrorNewline(t *testing.T) {
	ce := CompilerError{Type: "error", Message: "something went wrong"}
	stderr := captureStderr(t, func() {
		_, _ = PrintError(ce)
	})

	want := FormatError(ce)
	if stderr != want {
		t.Fatalf("PrintError output mismatch\nwant: %q\ngot:  %q", want, stderr)
	}
}

func TestPrintSuccessMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintSuccessMessage("ok") })
	want := FormatSuccessMessageStderr("ok") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintInfoMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintInfoMessage("info") })
	want := FormatInfoMessageStderr("info") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintTableHeaderStderr(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintTableHeaderStderr("Name") })
	want := FormatTableHeaderStderr("Name") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintWarningMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintWarningMessage("warn") })
	want := FormatWarningMessageStderr("warn") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintErrorMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintErrorMessage("boom") })
	want := FormatErrorMessage("boom") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintErrorTextStderr(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintErrorTextStderr("error text") })
	want := FormatErrorTextStderr("error text") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintCommandMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintCommandMessage("gh aw status") })
	want := FormatCommandMessageStderr("gh aw status") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintProgressMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintProgressMessage("loading") })
	want := FormatProgressMessageStderr("loading") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintPromptMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintPromptMessage("continue?") })
	want := FormatPromptMessageStderr("continue?") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintVerboseMessage(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintVerboseMessage("debug info") })
	want := FormatVerboseMessageStderr("debug info") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintListItem(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintListItem("item one") })
	want := FormatListItemStderr("item one") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
	}
}

func TestPrintSectionHeader(t *testing.T) {
	stderr := captureStderr(t, func() { _, _ = PrintSectionHeader("Section") })
	want := FormatSectionHeaderStderr("Section") + "\n"
	if stderr != want {
		t.Fatalf("want %q, got %q", want, stderr)
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
