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
	// Setup cleanup to restore original script before modifying
	t.Cleanup(func() {
		DefaultScriptRegistry.Register("create_issue", createIssueScriptSource)
	})

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

	// Extract just the create_issue job section
	// Find the job definition
	createIssueJobStart := strings.Index(lockStr, "  create_issue:")
	if createIssueJobStart == -1 {
		t.Fatal("Could not find create_issue job in lock file")
	}

	// Find the next top-level job (starts with "  " and ends with ":")
	// We need to find the next line that starts with exactly 2 spaces followed by a non-space
	remainingContent := lockStr[createIssueJobStart+15:] // Skip past "  create_issue:"
	nextJobStart := -1
	lines := strings.Split(remainingContent, "\n")
	currentPos := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && len(line) > 2 && line[2] != ' ' {
			// This is a top-level job
			nextJobStart = currentPos
			break
		}
		currentPos += len(line) + 1 // +1 for the newline
		if i >= len(lines)-1 {
			break
		}
	}

	var createIssueJobSection string
	if nextJobStart == -1 {
		createIssueJobSection = lockStr[createIssueJobStart:]
	} else {
		createIssueJobSection = lockStr[createIssueJobStart : createIssueJobStart+15+nextJobStart]
	}

	t.Logf("create_issue job section (%d bytes):\n%s", len(createIssueJobSection), createIssueJobSection)

	// Verify it uses custom action reference instead of actions/github-script
	if !strings.Contains(createIssueJobSection, "uses: ./actions/create-issue") {
		t.Error("Expected custom action reference './actions/create-issue' not found in create_issue job")
	}

	// Verify it does NOT contain actions/github-script in the create_issue job
	if strings.Contains(createIssueJobSection, "actions/github-script@") {
		t.Error("create_issue job should not contain 'actions/github-script@' when using dev action mode")
	}

	// Verify it has the token input instead of github-token with script
	if strings.Contains(createIssueJobSection, "github-token:") {
		t.Error("Dev action mode should use 'token:' input, not 'github-token:'")
	}

	if !strings.Contains(createIssueJobSection, "token:") {
		t.Error("Expected 'token:' input not found for custom action in create_issue job")
	}
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
	// Setup cleanup to restore original script before modifying
	t.Cleanup(func() {
		DefaultScriptRegistry.Register("create_issue", createIssueScriptSource)
	})
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
