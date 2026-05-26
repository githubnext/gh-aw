//go:build !integration

package cli

import (
	"context"
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
	ts := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)
	eventsContent := strings.Join([]string{
		`{"type":"user.message","id":"id1","timestamp":"` + ts + `","data":{}}`,
		`{"type":"tool.execution_start","id":"id2","timestamp":"` + ts + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github"}}`,
		`{"type":"tool.execution_complete","id":"id3","timestamp":"` + ts + `","data":{"toolCallId":"c1","toolName":"search","mcpServerName":"github","success":true}}`,
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
	// Should succeed: it reads from the pre-populated local directory.
	// We provide a non-zero runID that matches the directory created above.
	if err := ReplayWorkflowRun(context.Background(), 9999, opts); err != nil {
		t.Errorf("ReplayWorkflowRun returned unexpected error: %v", err)
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
