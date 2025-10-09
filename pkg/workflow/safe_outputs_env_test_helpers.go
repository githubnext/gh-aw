package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertEnvVarsInSteps checks that all expected environment variables are present in the job steps.
// This is a helper function to reduce duplication in safe outputs env tests.
func assertEnvVarsInSteps(t *testing.T, steps []string, expectedEnvVars []string) {
	t.Helper()
	stepsStr := strings.Join(steps, "")
	for _, expectedEnvVar := range expectedEnvVars {
		if !strings.Contains(stepsStr, expectedEnvVar) {
			t.Errorf("Expected env var %q not found in job YAML", expectedEnvVar)
		}
	}
}

// parseWorkflowFromContent creates a temporary workflow file, writes the content,
// and parses it using the compiler. Returns the parsed WorkflowData.
// This is a helper function to reduce duplication in integration tests.
//
//nolint:unused // Used in integration tests with build tags
func parseWorkflowFromContent(t *testing.T, workflowContent, filename string) *WorkflowData {
	t.Helper()

	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, filename)

	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	compiler := NewCompiler(false, "", "test")
	workflowData, err := compiler.ParseWorkflowFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	return workflowData
}
