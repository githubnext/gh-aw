//go:build !integration

package console

import (
	"bytes"
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

func TestPrintCommandMessage(t *testing.T) {
	got := captureStderr(t, func() { PrintCommandMessage("gh aw status") })
	want := FormatCommandMessageStderr("gh aw status") + "\n"
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
