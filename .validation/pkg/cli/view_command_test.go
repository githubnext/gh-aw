//go:build !integration

package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─── NewViewCommand ───────────────────────────────────────────────────────────

func TestNewViewCommand_FlagsExist(t *testing.T) {
	cmd := NewViewCommand()
	if cmd == nil {
		t.Fatal("NewViewCommand returned nil")
	}

	for _, name := range []string{"output", "repo"} {
		if cmd.Flags().Lookup(name) == nil && cmd.InheritedFlags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestNewViewCommand_UseAndShort(t *testing.T) {
	cmd := NewViewCommand()
	if !strings.HasPrefix(cmd.Use, "view") {
		t.Errorf("Use = %q; want prefix 'view'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description must not be empty")
	}
}

func TestNewViewCommand_RequiresExactlyOneArg(t *testing.T) {
	cmd := NewViewCommand()
	// No args → error
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("expected error for zero arguments")
	}
	// One arg → ok
	if err := cmd.Args(cmd, []string{"1234567890"}); err != nil {
		t.Errorf("expected no error for one argument, got: %v", err)
	}
	// Two args → error
	if err := cmd.Args(cmd, []string{"1", "2"}); err == nil {
		t.Error("expected error for two arguments")
	}
}

// ─── buildRunHTMLURL ──────────────────────────────────────────────────────────

func TestBuildRunHTMLURL_WithOwnerRepo(t *testing.T) {
	got := buildRunHTMLURL("github.com", "myorg", "myrepo", 12345)
	want := "https://github.com/myorg/myrepo/actions/runs/12345"
	if got != want {
		t.Errorf("buildRunHTMLURL = %q; want %q", got, want)
	}
}

func TestBuildRunHTMLURL_MissingOwner(t *testing.T) {
	got := buildRunHTMLURL("github.com", "", "myrepo", 12345)
	if got != "" {
		t.Errorf("buildRunHTMLURL with missing owner = %q; want empty string", got)
	}
}

func TestBuildRunHTMLURL_MissingRepo(t *testing.T) {
	got := buildRunHTMLURL("github.com", "myorg", "", 12345)
	if got != "" {
		t.Errorf("buildRunHTMLURL with missing repo = %q; want empty string", got)
	}
}

func TestBuildRunHTMLURL_DefaultHostname(t *testing.T) {
	got := buildRunHTMLURL("", "myorg", "myrepo", 12345)
	want := "https://github.com/myorg/myrepo/actions/runs/12345"
	if got != want {
		t.Errorf("buildRunHTMLURL with empty hostname = %q; want %q", got, want)
	}
}

func TestBuildRunHTMLURL_GHES(t *testing.T) {
	got := buildRunHTMLURL("github.example.com", "myorg", "myrepo", 12345)
	want := "https://github.example.com/myorg/myrepo/actions/runs/12345"
	if got != want {
		t.Errorf("buildRunHTMLURL GHES = %q; want %q", got, want)
	}
}

// ─── ViewWorkflowRun (local dir with pre-populated JSONL) ────────────────────

// buildViewRunDir creates a temporary run directory populated with synthetic
// JSONL files so that ViewWorkflowRun can read them without network access.
func buildViewRunDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-9999")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a minimal events.jsonl so the agent timeline source is populated.
	// Include content in user/assistant/reasoning messages to test snippet rendering.
	// Use distinct timestamps to guarantee deterministic sort order in BuildUnifiedTimeline.
	base := time.Now().UTC().Add(-time.Minute)
	timestampAt := func(offset time.Duration) string { return base.Add(offset).Format(time.RFC3339Nano) }
	eventsContent := strings.Join([]string{
		`{"type":"user.message","id":"id1","timestamp":"` + timestampAt(0) + `","data":{"content":"What files are in the repo?"}}`,
		`{"type":"tool.execution_start","id":"id2","timestamp":"` + timestampAt(10*time.Millisecond) + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github"}}`,
		`{"type":"tool.execution_complete","id":"id3","timestamp":"` + timestampAt(20*time.Millisecond) + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github","success":true}}`,
		`{"type":"assistant.message","id":"id4","timestamp":"` + timestampAt(30*time.Millisecond) + `","data":{"content":"I found the following files in the repo."}}`,
		`{"type":"reasoning","id":"id5","timestamp":"` + timestampAt(40*time.Millisecond) + `","data":{"content":"The user wants a list of files."}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), []byte(eventsContent), 0600); err != nil {
		t.Fatalf("WriteFile events.jsonl: %v", err)
	}

	// Mark the directory as already downloaded so downloadRunArtifacts skips
	// network calls (it returns early when the dir is non-empty and has no cached
	// summary — it just skips the download and lets the caller process what's there).
	return dir
}

func TestViewWorkflowRun_LocalCache_NoError(t *testing.T) {
	logsDir := buildViewRunDir(t)

	opts := ViewOptions{
		OutputDir: logsDir,
		Verbose:   false,
	}
	// Capture stdout so we can assert on the rendered timeline output.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	runErr := ViewWorkflowRun(context.Background(), 9999, opts)

	// Restore stdout and read captured output.
	w.Close()
	os.Stdout = origStdout
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("reading captured output: %v", readErr)
	}
	output := string(outBytes)

	if runErr != nil {
		t.Errorf("ViewWorkflowRun returned unexpected error: %v", runErr)
	}

	// Verify that renderUnifiedTimelineStream produced streaming output:
	// agent turns as "> Turn N [time]" headers, tool events with icons, no stats or table.
	for _, want := range []string{"> Turn 1", "github/search"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q; got:\n%s", want, output)
		}
	}
	// Verify user/assistant/reasoning message content snippets are rendered.
	for _, want := range []string{
		"What files are in the repo?",
		"I found the following files in the repo.",
		"The user wants a list of files.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing message snippet %q; got:\n%s", want, output)
		}
	}
	// Confirm there are no stats or table headers in the stream output.
	for _, notWant := range []string{"Total Events", "Event Timeline", "Gateway", "Firewall"} {
		if strings.Contains(output, notWant) {
			t.Errorf("stream output should not contain %q; got:\n%s", notWant, output)
		}
	}
}

func TestViewWorkflowRun_WithOwnerRepo_ShowsRunURL(t *testing.T) {
	logsDir := buildViewRunDir(t)

	opts := ViewOptions{
		Owner:     "myorg",
		Repo:      "myrepo",
		Hostname:  "github.com",
		OutputDir: logsDir,
		Verbose:   false,
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	runErr := ViewWorkflowRun(context.Background(), 9999, opts)

	w.Close()
	os.Stdout = origStdout
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("reading captured output: %v", readErr)
	}
	output := string(outBytes)

	if runErr != nil {
		t.Errorf("ViewWorkflowRun returned unexpected error: %v", runErr)
	}

	// The run URL should appear at the end of the output.
	wantURL := "https://github.com/myorg/myrepo/actions/runs/9999"
	if !strings.Contains(output, wantURL) {
		t.Errorf("output missing run URL %q; got:\n%s", wantURL, output)
	}
}

func TestViewWorkflowRun_WithSafeOutputs_ShowsSection(t *testing.T) {
	logsDir := buildViewRunDir(t)
	runDir := filepath.Join(logsDir, "run-9999")

	// Write a minimal safe-output-items.jsonl manifest.
	manifestContent := `{"type":"create_issue","url":"https://github.com/myorg/myrepo/issues/42","number":42,"repo":"myorg/myrepo","timestamp":"2024-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(runDir, "safe-output-items.jsonl"), []byte(manifestContent), 0600); err != nil {
		t.Fatalf("WriteFile safe-output-items.jsonl: %v", err)
	}

	opts := ViewOptions{
		Owner:     "myorg",
		Repo:      "myrepo",
		OutputDir: logsDir,
		Verbose:   false,
	}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	runErr := ViewWorkflowRun(context.Background(), 9999, opts)

	w.Close()
	os.Stdout = origStdout
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("reading captured output: %v", readErr)
	}
	output := string(outBytes)

	if runErr != nil {
		t.Errorf("ViewWorkflowRun returned unexpected error: %v", runErr)
	}

	// The safe outputs section should be present with the item type and URL.
	for _, want := range []string{
		"Safe Outputs",
		"create_issue",
		"https://github.com/myorg/myrepo/issues/42",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing safe output item %q; got:\n%s", want, output)
		}
	}
}

func TestViewWorkflowRun_EmptyDir_WarnsAndReturnsNil(t *testing.T) {
	// A run dir that is non-empty (so downloadRunArtifacts skips the network call)
	// but contains no JSONL files → no events → warning, no error.
	logsDir := t.TempDir()
	runDir := filepath.Join(logsDir, "run-1111")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Place a dummy file so the directory is not empty; downloadRunArtifacts will
	// skip the download when the dir is non-empty (no valid cached summary).
	if err := os.WriteFile(filepath.Join(runDir, "placeholder.txt"), []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile placeholder: %v", err)
	}

	opts := ViewOptions{
		OutputDir: logsDir,
		Verbose:   false,
	}
	if err := ViewWorkflowRun(context.Background(), 1111, opts); err != nil {
		t.Errorf("ViewWorkflowRun returned unexpected error for empty dir: %v", err)
	}
}
