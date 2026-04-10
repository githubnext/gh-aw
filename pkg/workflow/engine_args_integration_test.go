//go:build integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineArgsIntegration(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test workflow with engine args
	workflowContent := `---
on: workflow_dispatch
engine:
  id: copilot
  args: ["--add-dir", "/"]
---

# Test Workflow

This is a test workflow to verify engine args injection.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(content)

	// Check that the compiled YAML contains the custom args
	if !strings.Contains(result, "--add-dir /") {
		t.Errorf("Expected compiled YAML to contain '--add-dir /', got:\n%s", result)
	}

	// Verify args come before --prompt
	addDirIdx := strings.Index(result, "--add-dir /")
	promptIdx := strings.Index(result, "--prompt")
	if addDirIdx == -1 || promptIdx == -1 {
		t.Fatal("Could not find both --add-dir and --prompt in compiled YAML")
	}
	if addDirIdx > promptIdx {
		t.Error("Expected --add-dir to come before --prompt in compiled YAML")
	}
}

func TestEngineArgsIntegrationMultipleArgs(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test workflow with multiple engine args
	workflowContent := `---
on: workflow_dispatch
engine:
  id: copilot
  args: ["--add-dir", "/workspace", "--verbose"]
---

# Test Workflow with Multiple Args

This is a test workflow to verify multiple engine args injection.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(content)

	// Check that the compiled YAML contains all custom args
	if !strings.Contains(result, "--add-dir /workspace") {
		t.Errorf("Expected compiled YAML to contain '--add-dir /workspace'")
	}
	if !strings.Contains(result, "--verbose") {
		t.Errorf("Expected compiled YAML to contain '--verbose'")
	}

	// Verify args come before --prompt
	verboseIdx := strings.Index(result, "--verbose")
	promptIdx := strings.Index(result, "--prompt")
	if verboseIdx == -1 || promptIdx == -1 {
		t.Fatal("Could not find both --verbose and --prompt in compiled YAML")
	}
	if verboseIdx > promptIdx {
		t.Error("Expected --verbose to come before --prompt in compiled YAML")
	}
}

func TestEngineArgsIntegrationNoArgs(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test workflow without engine args
	workflowContent := `---
on: workflow_dispatch
engine:
  id: copilot
---

# Test Workflow without Args

This is a test workflow without engine args.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(content)

	// Should still have the --prompt flag
	if !strings.Contains(result, "--prompt") {
		t.Errorf("Expected compiled YAML to contain '--prompt'")
	}

	// Verify the workflow compiles successfully
	if result == "" {
		t.Error("Expected non-empty compiled YAML")
	}
}

func TestEngineArgsIntegrationClaude(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test workflow with claude engine args
	workflowContent := `---
on: workflow_dispatch
engine:
  id: claude
  args: ["--custom-flag", "value"]
---

# Test Workflow with Claude Args

This is a test workflow to verify claude engine args injection.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(content)

	// Check that the compiled YAML contains the custom args
	if !strings.Contains(result, "--custom-flag") {
		t.Errorf("Expected compiled YAML to contain '--custom-flag'")
	}
	if !strings.Contains(result, "value") {
		t.Errorf("Expected compiled YAML to contain 'value'")
	}
}

func TestEngineArgsIntegrationCodex(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test workflow with codex engine args
	workflowContent := `---
on: workflow_dispatch
engine:
  id: codex
  args: ["--custom-flag", "value"]
---

# Test Workflow with Codex Args

This is a test workflow to verify codex engine args injection.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	result := string(content)

	// Check that the compiled YAML contains the custom args before INSTRUCTION
	if !strings.Contains(result, "--custom-flag value") {
		t.Errorf("Expected compiled YAML to contain '--custom-flag value'")
	}

	// Verify args come before "$INSTRUCTION"
	customFlagIdx := strings.Index(result, "--custom-flag value")
	instructionIdx := strings.Index(result, "\"$INSTRUCTION\"")
	if customFlagIdx == -1 || instructionIdx == -1 {
		t.Fatal("Could not find both --custom-flag and $INSTRUCTION in compiled YAML")
	}
	if customFlagIdx > instructionIdx {
		t.Error("Expected --custom-flag to come before $INSTRUCTION in compiled YAML")
	}
}

func TestEngineBareModeCopilotIntegration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	workflowContent := `---
on: workflow_dispatch
engine:
  id: copilot
  bare: true
---

# Test Bare Mode Copilot
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test-workflow.lock.yml"))
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	result := string(content)

	if !strings.Contains(result, "--no-custom-instructions") {
		t.Errorf("Expected --no-custom-instructions in compiled output when bare=true, got:\n%s", result)
	}
}

func TestEngineBareModeClaudeIntegration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	workflowContent := `---
on: workflow_dispatch
engine:
  id: claude
  bare: true
---

# Test Bare Mode Claude
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "test-workflow.lock.yml"))
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	result := string(content)

	if !strings.Contains(result, "--bare") {
		t.Errorf("Expected --bare in compiled output when bare=true, got:\n%s", result)
	}
}

func TestBareMode_UnsupportedEngineWarningIntegration(t *testing.T) {
	tests := []struct {
		name          string
		engineID      string
		bannedOutput  []string
		workflowTitle string
	}{
		{
			name:          "codex emits warning, no --no-system-prompt in output",
			engineID:      "codex",
			bannedOutput:  []string{"--no-system-prompt"},
			workflowTitle: "Test Bare Mode Codex",
		},
		{
			name:          "gemini emits warning, no GEMINI_SYSTEM_MD=/dev/null in output",
			engineID:      "gemini",
			bannedOutput:  []string{"GEMINI_SYSTEM_MD=/dev/null"},
			workflowTitle: "Test Bare Mode Gemini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")

			workflowContent := fmt.Sprintf(`---
on: workflow_dispatch
engine:
  id: %s
  bare: true
---

# %s
`, tt.engineID, tt.workflowTitle)

			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644),
				"should write workflow file")

			compiler := NewCompiler()
			require.NoError(t, compiler.CompileWorkflow(workflowPath),
				"should compile without error (bare mode unsupported is a warning, not an error)")

			// A warning should have been counted
			assert.Greater(t, compiler.GetWarningCount(), 0,
				"should emit a warning when bare mode is specified for unsupported engine")

			content, err := os.ReadFile(filepath.Join(tmpDir, "test-workflow.lock.yml"))
			require.NoError(t, err, "should read lock file")
			result := string(content)

			for _, banned := range tt.bannedOutput {
				assert.NotContains(t, result, banned,
					"compiled output should not contain %q for unsupported bare mode engine", banned)
			}
		})
	}
}
