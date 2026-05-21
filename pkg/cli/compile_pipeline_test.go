//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
)

// writeCopilotWorkflowFile writes a minimal copilot workflow markdown to dir
// and returns the absolute path.
func writeCopilotWorkflowFile(t *testing.T, dir, name string) string {
	t.Helper()
	content := "---\nname: " + name + "\ndescription: test\non:\n  workflow_dispatch:\npermissions:\n  contents: read\nengine: copilot\ntimeout-minutes: 10\n---\n\n# " + name + "\n"
	path := filepath.Join(dir, name+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow file: %v", err)
	}
	return path
}

// TestCompileResolvedFilesInParallel_EmptyInput verifies that an empty file
// list returns nil without panicking.
func TestCompileResolvedFilesInParallel_EmptyInput(t *testing.T) {
	compiler := workflow.NewCompiler()
	compiler.WarmUp()

	results := compileResolvedFilesInParallel(
		context.Background(), compiler, nil, 4,
		compileWorkflowFileOptions{noEmit: true},
	)
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}

// TestCompileResolvedFilesInParallel_SequentialVsParallel verifies that the
// sequential path (workers=1) and the parallel path (workers>1) produce the
// same number of results for the same input.
func TestCompileResolvedFilesInParallel_SequentialVsParallel(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeCopilotWorkflowFile(t, dir, "workflow-a"),
		writeCopilotWorkflowFile(t, dir, "workflow-b"),
		writeCopilotWorkflowFile(t, dir, "workflow-c"),
	}

	ctx := context.Background()
	opts := compileWorkflowFileOptions{noEmit: true}

	// Sequential
	c1 := workflow.NewCompiler()
	c1.WarmUp()
	seqResults := compileResolvedFilesInParallel(ctx, c1, files, 1, opts)

	// Parallel (2 workers)
	c2 := workflow.NewCompiler()
	c2.WarmUp()
	parResults := compileResolvedFilesInParallel(ctx, c2, files, 2, opts)

	if len(seqResults) != len(files) {
		t.Fatalf("sequential: expected %d results, got %d", len(files), len(seqResults))
	}
	if len(parResults) != len(files) {
		t.Fatalf("parallel: expected %d results, got %d", len(files), len(parResults))
	}

	// Each result should correspond to its input file (results are ordered)
	for i, file := range files {
		seqName := filepath.Base(file)
		if seqResults[i].validationResult.Workflow != seqName && seqResults[i].validationResult.Workflow != "" {
			t.Errorf("sequential result[%d]: expected workflow %q, got %q", i, seqName, seqResults[i].validationResult.Workflow)
		}
		if parResults[i].validationResult.Workflow != seqName && parResults[i].validationResult.Workflow != "" {
			t.Errorf("parallel result[%d]: expected workflow %q, got %q", i, seqName, parResults[i].validationResult.Workflow)
		}
	}
}

// TestCompileResolvedFilesInParallel_ResultsOrdered verifies that results are
// returned in the same order as the input slice, regardless of completion order.
func TestCompileResolvedFilesInParallel_ResultsOrdered(t *testing.T) {
	dir := t.TempDir()
	n := 5
	files := make([]string, n)
	for i := range files {
		files[i] = writeCopilotWorkflowFile(t, dir, filepath.Join("wf"+string(rune('a'+i))))
	}

	compiler := workflow.NewCompiler()
	compiler.WarmUp()

	results := compileResolvedFilesInParallel(
		context.Background(), compiler, files, 4,
		compileWorkflowFileOptions{noEmit: true},
	)

	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}

	// Each slot must correspond to the input file at the same index
	for i, file := range files {
		// If compilation produced a validation result with a workflow name, it
		// should match the input file (or be empty for failed compilations).
		wf := results[i].validationResult.Workflow
		base := filepath.Base(file)
		if wf != "" && wf != base {
			t.Errorf("result[%d] workflow=%q but input file was %q", i, wf, base)
		}
	}
}
