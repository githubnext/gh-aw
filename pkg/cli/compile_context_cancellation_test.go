//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
// per-file compilation loop in compileSpecificFiles stops when the context is
// cancelled mid-loop (i.e. after CompileWorkflows has already started).
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
	// but each failure still advances through the loop so the per-iteration
	// ctx.Done() guard is reached on subsequent iterations.
	const numFiles = 5
	var files []string
	for i := range numFiles {
		name := filepath.Join(workflowsDir, "workflow-"+string(rune('a'+i))+".md")
		if err := os.WriteFile(name, []byte("# dummy"), 0644); err != nil {
			t.Fatal(err)
		}
		files = append(files, name)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run CompileWorkflows concurrently and cancel the context shortly after
	// it starts — after the goroutine has been scheduled but while it is still
	// iterating through the file list.  The per-iteration guard must catch the
	// cancellation and return context.Canceled.
	done := make(chan error, 1)
	go func() {
		_, err := CompileWorkflows(ctx, CompileConfig{
			MarkdownFiles: files,
			NoEmit:        true,
		})
		done <- err
	}()

	// Yield to let the goroutine start, then cancel.
	runtime.Gosched()
	time.Sleep(time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("CompileWorkflows did not exit within 5s after context cancellation")
	}
}

// TestCompileWorkflows_ContextCancelledDuringDirectory verifies that the
// directory-wide compilation loop stops when the context is cancelled while
// CompileWorkflows is already running.
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := CompileWorkflows(ctx, CompileConfig{
			NoEmit:      true,
			WorkflowDir: ".github/workflows",
		})
		done <- err
	}()

	// Yield to let the goroutine start, then cancel.
	runtime.Gosched()
	time.Sleep(time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("CompileWorkflows did not exit within 5s after context cancellation")
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

	// Create a minimal valid workflow file so compilation reaches the watch
	// select loop rather than exiting early on a parse error.
	testFile := filepath.Join(workflowsDir, "test.md")
	validFrontmatter := "---\nname: test\non:\n  push:\n    branches: [main]\n---\n# Test Workflow\n"
	if err := os.WriteFile(testFile, []byte(validFrontmatter), 0644); err != nil {
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
		done <- watchAndCompileWorkflows(ctx, testFile, workflow.NewCompiler(), false)
	}()

	// Give the watcher time to enter the select loop, then cancel the context.
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
