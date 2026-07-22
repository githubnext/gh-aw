//go:build !integration

package console

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

// captureStderr redirects Print* output to an in-memory buffer by swapping the
// package-level stderr variable. Tests using this helper must not call
// t.Parallel() because the variable is not concurrency-safe.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	old := stderr
	stderr = &buf
	defer func() { stderr = old }()
	fn()
	return buf.String()
}

// TestPrintErrorNewline verifies that PrintError does not emit a spurious blank
// line: FormatError already terminates with \n, so Fprint (not Fprintln) must
// be used.
func TestPrintErrorNewline(t *testing.T) {
	ce := CompilerError{Type: "error", Message: "something went wrong"}
	got := captureStderr(t, func() { PrintError(ce) })

	want := FormatError(ce)
	if got != want {
		t.Fatalf("PrintError output mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

func TestPrintSuccessMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintSuccessMessage("ok") })
	want := FormatSuccessMessageStderr("ok") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintInfoMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintInfoMessage("info") })
	want := FormatInfoMessageStderr("info") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintTableHeaderStderr(t *testing.T) {
	got := captureStderr(t, func() { PrintTableHeaderStderr("Name") })
	want := FormatTableHeaderStderr("Name") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintWarningMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintWarningMessage("warn") })
	want := FormatWarningMessageStderr("warn") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintErrorMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintErrorMessage("boom") })
	want := FormatErrorMessage("boom") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintErrorTextStderr(t *testing.T) {
	got := captureStderr(t, func() { PrintErrorTextStderr("error text") })
	want := FormatErrorTextStderr("error text") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintCommandMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintCommandMessage("gh aw status") })
	want := FormatCommandMessageStderr("gh aw status") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintProgressMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintProgressMessage("loading") })
	want := FormatProgressMessageStderr("loading") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintPromptMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintPromptMessage("continue?") })
	want := FormatPromptMessageStderr("continue?") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintVerboseMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintVerboseMessage("debug info") })
	want := FormatVerboseMessageStderr("debug info") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintListItem(t *testing.T) {
	got := captureStderr(t, func() { PrintListItem("item one") })
	want := FormatListItemStderr("item one") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintSectionHeader(t *testing.T) {
	got := captureStderr(t, func() { PrintSectionHeader("Section") })
	want := FormatSectionHeaderStderr("Section") + "\n"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestPrintErrorChain(t *testing.T) {
	err := errors.New("inner")
	wrapped := fmt.Errorf("outer: %w", err)

	got := captureStderr(t, func() { PrintErrorChain(wrapped) })

	want := FormatErrorChain(wrapped) + "\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
