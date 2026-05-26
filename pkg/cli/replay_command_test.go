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

// ─── NewReplayCommand ─────────────────────────────────────────────────────────

func TestNewReplayCommand_FlagsExist(t *testing.T) {
	cmd := NewReplayCommand()
	if cmd == nil {
		t.Fatal("NewReplayCommand returned nil")
	}

	for _, name := range []string{"output", "repo"} {
		if cmd.Flags().Lookup(name) == nil && cmd.InheritedFlags().Lookup(name) == nil {
			t.Errorf("expected flag --%s to be defined", name)
		}
	}
}

func TestNewReplayCommand_UseAndShort(t *testing.T) {
	cmd := NewReplayCommand()
	if !strings.HasPrefix(cmd.Use, "replay") {
		t.Errorf("Use = %q; want prefix 'replay'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description must not be empty")
	}
}

func TestNewReplayCommand_RequiresExactlyOneArg(t *testing.T) {
	cmd := NewReplayCommand()
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

// ─── ReplayWorkflowRun (local dir with pre-populated JSONL) ──────────────────

// buildReplayRunDir creates a temporary run directory populated with synthetic
// JSONL files so that ReplayWorkflowRun can read them without network access.
func buildReplayRunDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runDir := filepath.Join(dir, "run-9999")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a minimal events.jsonl so the agent timeline source is populated.
	// Include content in user/assistant/reasoning messages to test snippet rendering.
	timestamp := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
	eventsContent := strings.Join([]string{
		`{"type":"user.message","id":"id1","timestamp":"` + timestamp + `","data":{"content":"What files are in the repo?"}}`,
		`{"type":"tool.execution_start","id":"id2","timestamp":"` + timestamp + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github"}}`,
		`{"type":"tool.execution_complete","id":"id3","timestamp":"` + timestamp + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github","success":true}}`,
		`{"type":"assistant.message","id":"id4","timestamp":"` + timestamp + `","data":{"content":"I found the following files in the repo."}}`,
		`{"type":"reasoning","id":"id5","timestamp":"` + timestamp + `","data":{"content":"The user wants a list of files."}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), []byte(eventsContent), 0600); err != nil {
		t.Fatalf("WriteFile events.jsonl: %v", err)
	}

	// Mark the directory as already downloaded so downloadRunArtifacts skips
	// network calls (it returns early when the dir is non-empty and has no cached
	// summary — it just skips the download and lets the caller process what's there).
	return dir
}

func TestReplayWorkflowRun_LocalCache_NoError(t *testing.T) {
	logsDir := buildReplayRunDir(t)

	opts := ReplayOptions{
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

	runErr := ReplayWorkflowRun(context.Background(), 9999, opts)

	// Restore stdout and read captured output.
	w.Close()
	os.Stdout = origStdout
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("reading captured output: %v", readErr)
	}
	output := string(outBytes)

	if runErr != nil {
		t.Errorf("ReplayWorkflowRun returned unexpected error: %v", runErr)
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

func TestReplayWorkflowRun_EmptyDir_WarnsAndReturnsNil(t *testing.T) {
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

	opts := ReplayOptions{
		OutputDir: logsDir,
		Verbose:   false,
	}
	if err := ReplayWorkflowRun(context.Background(), 1111, opts); err != nil {
		t.Errorf("ReplayWorkflowRun returned unexpected error for empty dir: %v", err)
	}
}
