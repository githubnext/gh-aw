//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestCompileWorkflows_ContextCancelledAtStart verifies that CompileWorkflows
// returns context.Canceled when the context is already cancelled on entry.
// This exercises the early-exit guard at the top of CompileWorkflows.
func TestCompileWorkflows_ContextCancelledAtStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := CompileWorkflows(ctx, CompileConfig{
		MarkdownFiles: []string{"any.md"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestCompileWorkflows_ContextCancelledDuringSpecificFiles verifies that the
// per-file compilation loop in compileSpecificFiles stops processing additional
// files once the context is cancelled between iterations.
func TestCompileWorkflows_ContextCancelledDuringSpecificFiles(t *testing.T) {
	tempDir := testutil.TempDir(t, "compile-ctx-cancel-*")
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := initTestGitRepo(tempDir); err != nil {
		t.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Create several dummy .md files.  The compiler will fail to parse them,
	// but that is fine – we just want to observe that the loop is exited early.
	const numFiles = 5
	var files []string
	for i := 0; i < numFiles; i++ {
		name := filepath.Join(workflowsDir, "workflow-"+string(rune('a'+i))+".md")
		if err := os.WriteFile(name, []byte("# dummy"), 0644); err != nil {
			t.Fatal(err)
		}
		files = append(files, name)
	}

	// Cancel the context before calling CompileWorkflows so that the loop
	// exits on the very first ctx.Done() check.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := CompileWorkflows(ctx, CompileConfig{
		MarkdownFiles: files,
		NoEmit:        true,
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestCompileWorkflows_ContextCancelledDuringDirectory verifies that the
// directory-wide compilation loop stops processing additional files once the
// context is cancelled between iterations.
func TestCompileWorkflows_ContextCancelledDuringDirectory(t *testing.T) {
	tempDir := testutil.TempDir(t, "compile-ctx-dir-*")
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := initTestGitRepo(tempDir); err != nil {
		t.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Write a workflow file with the required frontmatter marker so it passes
	// the filterMarkdownFilesWithFrontmatter check.
	frontmatter := "---\nname: test\n---\n# Workflow\nSome content\n"
	wfFile := filepath.Join(workflowsDir, "workflow-a.md")
	if err := os.WriteFile(wfFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	// Cancel the context before calling so the loop exits immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := CompileWorkflows(ctx, CompileConfig{
		NoEmit:      true,
		WorkflowDir: ".github/workflows",
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestWatchAndCompileWorkflows_ContextCancellation verifies that the watch loop
// exits cleanly when the supplied context is cancelled.
func TestWatchAndCompileWorkflows_ContextCancellation(t *testing.T) {
	tempDir := testutil.TempDir(t, "watch-ctx-cancel-*")
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := initTestGitRepo(tempDir); err != nil {
		t.Fatal(err)
	}

	// Create a dummy markdown file to satisfy the watcher's existence check.
	testFile := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- watchAndCompileWorkflows(ctx, testFile, &workflow.Compiler{}, false)
	}()

	// Give the watcher a moment to start, then cancel the context.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		// nil is expected: context cancellation should cause a clean exit.
		if err != nil {
			t.Errorf("watchAndCompileWorkflows returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("watchAndCompileWorkflows did not exit within 2s after context cancellation")
	}
}
