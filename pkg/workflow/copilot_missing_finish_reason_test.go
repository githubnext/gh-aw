package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestCopilotMissingFinishReasonRecovery tests that the Copilot engine includes
// a step to check for the "missing finish_reason" error and treats it as success
func TestCopilotMissingFinishReasonRecovery(t *testing.T) {
	tmpDir := testutil.TempDir(t, "copilot-missing-finish-reason-test")

	// Create a simple workflow with Copilot engine
	frontmatter := `---
name: test-missing-finish-reason
engine: copilot
on: workflow_dispatch
---`

	testContent := frontmatter + "\n\n# Test Workflow\n\nTest workflow content."
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "dev")
	compiler.SetActionMode(ActionModeDev)

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify that the agentic_execution step has continue-on-error: true
	if !strings.Contains(lockStr, "id: agentic_execution") {
		t.Error("Expected 'id: agentic_execution' in compiled workflow")
	}

	// Find the agentic_execution step and check for continue-on-error
	agenticExecutionIndex := strings.Index(lockStr, "id: agentic_execution")
	if agenticExecutionIndex == -1 {
		t.Fatal("Could not find agentic_execution step")
	}

	// Look for continue-on-error within the next 500 characters
	executionSection := lockStr[agenticExecutionIndex : agenticExecutionIndex+500]
	if !strings.Contains(executionSection, "continue-on-error: true") {
		t.Error("Expected 'continue-on-error: true' in agentic_execution step")
	}

	// Test 2: Verify that there is a "Check for recoverable errors" step
	if !strings.Contains(lockStr, "Check for recoverable errors") {
		t.Error("Expected 'Check for recoverable errors' step in compiled workflow")
	}

	// Test 3: Verify the step checks the execution outcome
	if !strings.Contains(lockStr, "if: steps.agentic_execution.outcome == 'failure'") {
		t.Error("Expected error check step to have condition 'if: steps.agentic_execution.outcome == 'failure''")
	}

	// Test 4: Verify the step greps for the specific error message
	if !strings.Contains(lockStr, "missing finish_reason for choice 0") {
		t.Error("Expected grep for 'missing finish_reason for choice 0' in error check step")
	}

	// Test 5: Verify the step treats the error as recoverable
	if !strings.Contains(lockStr, "Treating execution as successful") {
		t.Error("Expected message 'Treating execution as successful' in error check step")
	}

	// Test 6: Verify the step fails for other errors
	if !strings.Contains(lockStr, "Execution failed with non-recoverable error") {
		t.Error("Expected message 'Execution failed with non-recoverable error' in error check step")
	}
}

// TestCopilotMissingFinishReasonStepOrder tests that the error check step
// is placed immediately after the agentic_execution step
func TestCopilotMissingFinishReasonStepOrder(t *testing.T) {
	tmpDir := testutil.TempDir(t, "copilot-step-order-test")

	// Create a simple workflow with Copilot engine
	frontmatter := `---
name: test-step-order
engine: copilot
on: workflow_dispatch
---`

	testContent := frontmatter + "\n\n# Test Workflow\n\nTest workflow content."
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "dev")
	compiler.SetActionMode(ActionModeDev)

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Find the position of agentic_execution step
	agenticExecutionIndex := strings.Index(lockStr, "id: agentic_execution")
	if agenticExecutionIndex == -1 {
		t.Fatal("Could not find agentic_execution step")
	}

	// Find the position of the error check step
	errorCheckIndex := strings.Index(lockStr, "Check for recoverable errors")
	if errorCheckIndex == -1 {
		t.Fatal("Could not find 'Check for recoverable errors' step")
	}

	// Verify the error check step comes after the agentic_execution step
	if errorCheckIndex <= agenticExecutionIndex {
		t.Error("Expected 'Check for recoverable errors' step to come after 'agentic_execution' step")
	}

	// Find what comes between the two steps
	betweenSteps := lockStr[agenticExecutionIndex:errorCheckIndex]

	// The error check should be the next step (only env vars and command should be between them)
	// Count the number of "- name:" occurrences between the steps
	stepsBetween := strings.Count(betweenSteps, "- name:")

	// There should be exactly 1 "- name:" (the agentic_execution itself)
	// and then the next "- name:" should be the error check
	if stepsBetween > 1 {
		t.Errorf("Expected error check step to be immediately after agentic_execution, but found %d steps in between", stepsBetween-1)
	}
}

// TestCopilotMissingFinishReasonLogFile tests that the error check uses the correct log file path
func TestCopilotMissingFinishReasonLogFile(t *testing.T) {
	tmpDir := testutil.TempDir(t, "copilot-log-file-test")

	// Create a simple workflow with Copilot engine
	frontmatter := `---
name: test-log-file
engine: copilot
on: workflow_dispatch
---`

	testContent := frontmatter + "\n\n# Test Workflow\n\nTest workflow content."
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "dev")
	compiler.SetActionMode(ActionModeDev)

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Find the error check step
	errorCheckIndex := strings.Index(lockStr, "Check for recoverable errors")
	if errorCheckIndex == -1 {
		t.Fatal("Could not find 'Check for recoverable errors' step")
	}

	// Extract the section containing the error check
	errorCheckSection := lockStr[errorCheckIndex:]
	if nextStepIndex := strings.Index(errorCheckSection, "\n      - name:"); nextStepIndex != -1 {
		errorCheckSection = errorCheckSection[:nextStepIndex]
	}

	// Verify the log file path is correct (/tmp/gh-aw/agent-stdio.log)
	if !strings.Contains(errorCheckSection, "/tmp/gh-aw/agent-stdio.log") {
		t.Error("Expected error check step to use '/tmp/gh-aw/agent-stdio.log' as log file path")
	}
}
