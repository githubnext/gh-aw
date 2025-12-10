package workflow

import (
	"os"
	"strings"
	"testing"
)

// TestActionModeValidation tests the ActionMode type validation
func TestActionModeValidation(t *testing.T) {
	tests := []struct {
		mode  ActionMode
		valid bool
	}{
		{ActionModeInline, true},
		{ActionModeDev, true},
		{ActionMode("invalid"), false},
		{ActionMode(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.IsValid(); got != tt.valid {
				t.Errorf("ActionMode(%q).IsValid() = %v, want %v", tt.mode, got, tt.valid)
			}
		})
	}
}

// TestActionModeString tests the String() method
func TestActionModeString(t *testing.T) {
	tests := []struct {
		mode ActionMode
		want string
	}{
		{ActionModeInline, "inline"},
		{ActionModeDev, "dev"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("ActionMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompilerActionModeDefault tests that the compiler defaults to inline mode
func TestCompilerActionModeDefault(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")
	if compiler.GetActionMode() != ActionModeInline {
		t.Errorf("Default action mode should be inline, got %s", compiler.GetActionMode())
	}
}

// TestCompilerSetActionMode tests setting the action mode
func TestCompilerSetActionMode(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	compiler.SetActionMode(ActionModeDev)
	if compiler.GetActionMode() != ActionModeDev {
		t.Errorf("Expected action mode dev, got %s", compiler.GetActionMode())
	}

	compiler.SetActionMode(ActionModeInline)
	if compiler.GetActionMode() != ActionModeInline {
		t.Errorf("Expected action mode inline, got %s", compiler.GetActionMode())
	}
}

// TestScriptRegistryWithAction tests registering scripts with action paths
func TestScriptRegistryWithAction(t *testing.T) {
	registry := NewScriptRegistry()

	testScript := `console.log('test');`
	actionPath := "./actions/test-action"

	registry.RegisterWithAction("test_script", testScript, RuntimeModeGitHubScript, actionPath)

	if !registry.Has("test_script") {
		t.Error("Script should be registered")
	}

	if got := registry.GetActionPath("test_script"); got != actionPath {
		t.Errorf("Expected action path %q, got %q", actionPath, got)
	}

	if got := registry.GetSource("test_script"); got != testScript {
		t.Errorf("Expected source %q, got %q", testScript, got)
	}
}

// TestScriptRegistryActionPathEmpty tests that scripts without action paths return empty string
func TestScriptRegistryActionPathEmpty(t *testing.T) {
	registry := NewScriptRegistry()

	testScript := `console.log('test');`
	registry.Register("test_script", testScript)

	if got := registry.GetActionPath("test_script"); got != "" {
		t.Errorf("Expected empty action path, got %q", got)
	}
}

// TestCustomActionModeCompilation tests workflow compilation with custom action mode
func TestCustomActionModeCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
name: Test Custom Actions
on: issues
safe-outputs:
  create-issue:
    max: 1
---

Test workflow with safe-outputs.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Register a test script with an action path
	testScript := `
const { core } = require('@actions/core');
core.info('Creating issue');
`
	DefaultScriptRegistry.RegisterWithAction(
		"create_issue",
		testScript,
		RuntimeModeGitHubScript,
		"./actions/create-issue",
	)

	// Compile with dev action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify it uses custom action reference instead of actions/github-script
	if !strings.Contains(lockStr, "uses: ./actions/create-issue") {
		t.Error("Expected custom action reference './actions/create-issue' not found in lock file")
	}

	// Extract the create_issue job section to verify it doesn't use actions/github-script
	// The test should check the safe output job (create_issue), not the entire workflow
	// (conclusion job may still use actions/github-script for failure tracking)
	createIssueJobStart := strings.Index(lockStr, "  create_issue:")
	if createIssueJobStart == -1 {
		t.Fatal("Could not find create_issue job in lock file")
	}
	
	// Find the next job (starts with "  " at the beginning of a line, not "    ")
	remainingContent := lockStr[createIssueJobStart+17:] // Skip past "  create_issue:\n"
	nextJobIdx := -1
	lines := strings.Split(remainingContent, "\n")
	for i, line := range lines {
		// A new job starts with exactly 2 spaces at the start and is not indented further
		if len(line) >= 2 && line[0:2] == "  " && len(line) > 2 && line[2] != ' ' {
			// Calculate position in remaining content
			nextJobIdx = 0
			for j := 0; j < i; j++ {
				nextJobIdx += len(lines[j]) + 1 // +1 for newline
			}
			break
		}
	}
	
	var createIssueJobSection string
	if nextJobIdx == -1 {
		createIssueJobSection = lockStr[createIssueJobStart:]
	} else {
		createIssueJobSection = lockStr[createIssueJobStart : createIssueJobStart+17+nextJobIdx]
	}

	// Verify create_issue job does NOT contain actions/github-script
	if strings.Contains(createIssueJobSection, "actions/github-script@") {
		t.Error("create_issue job should not contain 'actions/github-script@' when using dev action mode")
	}

	// Verify create_issue job has the token input instead of github-token with script
	if strings.Contains(createIssueJobSection, "github-token:") {
		t.Error("create_issue job in dev action mode should use 'token:' input, not 'github-token:'")
	}

	if !strings.Contains(createIssueJobSection, "token:") {
		t.Error("Expected 'token:' input not found for custom action in create_issue job")
	}

	// Clean up: reset the registry to avoid affecting other tests
	DefaultScriptRegistry.RegisterWithMode("create_issue", testScript, RuntimeModeGitHubScript)
}

// TestInlineActionModeCompilation tests workflow compilation with inline mode (default)
func TestInlineActionModeCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
name: Test Inline Actions
on: issues
safe-outputs:
  create-issue:
    max: 1
---

Test workflow with inline mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile with inline mode (default)
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeInline)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify it uses actions/github-script
	if !strings.Contains(lockStr, "actions/github-script@") {
		t.Error("Expected 'actions/github-script@' not found in lock file for inline mode")
	}

	// Verify it has github-token parameter
	if !strings.Contains(lockStr, "github-token:") {
		t.Error("Expected 'github-token:' parameter not found for inline mode")
	}

	// Verify it has script: parameter
	if !strings.Contains(lockStr, "script: |") {
		t.Error("Expected 'script: |' parameter not found for inline mode")
	}
}

// TestCustomActionModeFallback tests that compilation falls back to inline mode
// when action path is not registered
func TestCustomActionModeFallback(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
name: Test Fallback
on: issues
safe-outputs:
  create-issue:
    max: 1
---

Test fallback to inline mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Ensure create_issue is registered without an action path
	testScript := `console.log('test');`
	DefaultScriptRegistry.RegisterWithMode("create_issue", testScript, RuntimeModeGitHubScript)

	// Compile with dev action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify it falls back to actions/github-script when action path is not found
	if !strings.Contains(lockStr, "actions/github-script@") {
		t.Error("Expected fallback to 'actions/github-script@' when action path not found")
	}
}
