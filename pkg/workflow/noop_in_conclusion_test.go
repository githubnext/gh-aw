//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
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

	compiler := NewCompiler()

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

func TestMissingToolStepInConclusionJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "missing-tool-in-conclusion-test")

	// Create a test markdown file with missing-tool safe output
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  missing-tool:
    max: 10
---

# Test Missing Tool in Conclusion

Test that missing_tool step is generated inside the conclusion job.
`

	testFile := filepath.Join(tmpDir, "test-missing-tool.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-missing-tool.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that there is NO separate missing_tool job
	if strings.Contains(compiled, "\n  missing_tool:") {
		t.Error("There should NOT be a separate missing_tool job")
	}

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Verify that "Record missing tool" step is in the conclusion job
	conclusionSection := extractJobSection(compiled, "conclusion")
	if !strings.Contains(conclusionSection, "Record missing tool") {
		t.Error("Conclusion job should contain 'Record missing tool' step")
	}

	// Verify that conclusion job has missing_tool outputs
	if !strings.Contains(conclusionSection, "tools_reported:") {
		t.Error("Conclusion job should have 'tools_reported' output")
	}
	if !strings.Contains(conclusionSection, "total_count:") {
		t.Error("Conclusion job should have 'total_count' output")
	}

	// Verify that conclusion job does NOT depend on missing_tool job
	if strings.Contains(conclusionSection, "- missing_tool") {
		t.Error("Conclusion job should NOT depend on 'missing_tool' job")
	}

	// Verify that conclusion job depends on agent job
	if !strings.Contains(conclusionSection, "- agent") {
		t.Error("Conclusion job should depend on 'agent' job")
	}
}

func TestBothNoopAndMissingToolInConclusionJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "both-in-conclusion-test")

	// Create a test markdown file with both noop and missing-tool safe outputs
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
  missing-tool:
    max: 10
---

# Test Both Noop and Missing Tool in Conclusion

Test that both noop and missing_tool steps are generated inside the conclusion job.
`

	testFile := filepath.Join(tmpDir, "test-both.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-both.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that there are NO separate jobs
	if strings.Contains(compiled, "\n  noop:") {
		t.Error("There should NOT be a separate noop job")
	}
	if strings.Contains(compiled, "\n  missing_tool:") {
		t.Error("There should NOT be a separate missing_tool job")
	}

	// Verify that conclusion job exists and contains both steps
	conclusionSection := extractJobSection(compiled, "conclusion")
	if !strings.Contains(conclusionSection, "Process No-Op Messages") {
		t.Error("Conclusion job should contain 'Process No-Op Messages' step")
	}
	if !strings.Contains(conclusionSection, "Record missing tool") {
		t.Error("Conclusion job should contain 'Record missing tool' step")
	}

	// Verify that conclusion job has all outputs
	if !strings.Contains(conclusionSection, "noop_message:") {
		t.Error("Conclusion job should have 'noop_message' output")
	}
	if !strings.Contains(conclusionSection, "tools_reported:") {
		t.Error("Conclusion job should have 'tools_reported' output")
	}
	if !strings.Contains(conclusionSection, "total_count:") {
		t.Error("Conclusion job should have 'total_count' output")
	}
}

func TestReportIncompleteStepInConclusionJob(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "report-incomplete-in-conclusion-test")

	// Create a test markdown file with report-incomplete safe output
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  report-incomplete:
    max: 5
---

# Test Report Incomplete in Conclusion

Test that report_incomplete step is generated inside the conclusion job.
`

	testFile := filepath.Join(tmpDir, "test-report-incomplete.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-report-incomplete.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that there is NO separate report_incomplete job
	if strings.Contains(compiled, "\n  report_incomplete:") {
		t.Error("There should NOT be a separate report_incomplete job")
	}

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Verify that "Record incomplete" step is in the conclusion job
	conclusionSection := extractJobSection(compiled, "conclusion")
	if !strings.Contains(conclusionSection, "Record incomplete") {
		t.Error("Conclusion job should contain 'Record incomplete' step")
	}

	// Verify that conclusion job has report_incomplete output
	if !strings.Contains(conclusionSection, "incomplete_count:") {
		t.Error("Conclusion job should have 'incomplete_count' output")
	}

	// Verify that conclusion job does NOT depend on report_incomplete job
	if strings.Contains(conclusionSection, "- report_incomplete") {
		t.Error("Conclusion job should NOT depend on 'report_incomplete' job")
	}

	// Verify that conclusion job depends on agent job
	if !strings.Contains(conclusionSection, "- agent") {
		t.Error("Conclusion job should depend on 'agent' job")
	}
}
