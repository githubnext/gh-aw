package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestNoopStepInConclusionJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-in-conclusion-test")

	// Create a test markdown file with noop safe output
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 5
---

# Test Noop in Conclusion

Test that noop step is generated inside the conclusion job.
`

	testFile := filepath.Join(tmpDir, "test-noop.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that there is NO separate noop job
	if strings.Contains(compiled, "\n  noop:") {
		t.Error("There should NOT be a separate noop job")
	}

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Verify that "Process No-Op Messages" step is in the conclusion job
	conclusionSection := extractJobSection(compiled, "conclusion")
	if !strings.Contains(conclusionSection, "Process No-Op Messages") {
		t.Error("Conclusion job should contain 'Process No-Op Messages' step")
	}

	// Verify that conclusion job has noop_message output
	if !strings.Contains(conclusionSection, "noop_message:") {
		t.Error("Conclusion job should have 'noop_message' output")
	}

	// Verify that conclusion job does NOT depend on noop job
	if strings.Contains(conclusionSection, "- noop") {
		t.Error("Conclusion job should NOT depend on 'noop' job")
	}

	// Verify that conclusion job depends on agent job
	if !strings.Contains(conclusionSection, "- agent") {
		t.Error("Conclusion job should depend on 'agent' job")
	}
}
