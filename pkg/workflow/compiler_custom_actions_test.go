package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
)

// TestActionModeValidation tests the ActionMode type validation
func TestActionModeValidation(t *testing.T) {
	tests := []struct {
		mode  ActionMode
		valid bool
	}{
		// Removed ActionModeInline as it no longer exists
		{ActionModeDev, true},
		{ActionModeRelease, true},
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
		// Removed ActionModeInline as it no longer exists
		{ActionModeDev, "dev"},
		{ActionModeRelease, "release"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("ActionMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompilerActionModeDefault tests that the compiler defaults to dev mode
func TestCompilerActionModeDefault(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")
	if compiler.GetActionMode() != ActionModeDev {
		t.Errorf("Default action mode should be dev, got %s", compiler.GetActionMode())
	}
}

// TestCompilerSetActionMode tests setting the action mode
func TestCompilerSetActionMode(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	compiler.SetActionMode(ActionModeRelease)
	if compiler.GetActionMode() != ActionModeRelease {
		t.Errorf("Expected action mode release, got %s", compiler.GetActionMode())
	}

	compiler.SetActionMode(ActionModeDev)
	if compiler.GetActionMode() != ActionModeDev {
		t.Errorf("Expected action mode dev, got %s", compiler.GetActionMode())
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
	// Save original state first
	origSource := DefaultScriptRegistry.GetSource("create_issue")
	origActionPath := DefaultScriptRegistry.GetActionPath("create_issue")

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

	// Restore after test
	defer func() {
		if origSource != "" {
			if origActionPath != "" {
				DefaultScriptRegistry.RegisterWithAction("create_issue", origSource, RuntimeModeGitHubScript, origActionPath)
			} else {
				DefaultScriptRegistry.RegisterWithMode("create_issue", origSource, RuntimeModeGitHubScript)
			}
		}
	}()

	// Compile with dev action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify safe_outputs job exists (consolidated mode)
	safeOutputsJobStart := strings.Index(lockStr, "safe_outputs:")
	if safeOutputsJobStart == -1 {
		t.Fatal("safe_outputs job not found in lock file")
	}

	// Verify handler manager step is present (create_issue is now handled by handler manager)
	if !strings.Contains(lockStr, "id: process_safe_outputs") {
		t.Error("Expected process_safe_outputs step in compiled workflow (create-issue is now handled by handler manager)")
	}
	// Verify handler config contains create_issue
	if !strings.Contains(lockStr, "create_issue") {
		t.Error("Expected create_issue in handler config")
	}

	// Verify the workflow compiles successfully with custom action mode
	if !strings.Contains(lockStr, "actions/github-script") {
		t.Error("Expected github-script action in compiled workflow")
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

Test workflow with dev mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile with dev mode (default)
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
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
	// Save original state first
	origSource := DefaultScriptRegistry.GetSource("create_issue")
	origActionPath := DefaultScriptRegistry.GetActionPath("create_issue")

	testScript := `console.log('test');`
	DefaultScriptRegistry.RegisterWithMode("create_issue", testScript, RuntimeModeGitHubScript)

	// Restore after test
	defer func() {
		if origSource != "" {
			if origActionPath != "" {
				DefaultScriptRegistry.RegisterWithAction("create_issue", origSource, RuntimeModeGitHubScript, origActionPath)
			} else {
				DefaultScriptRegistry.RegisterWithMode("create_issue", origSource, RuntimeModeGitHubScript)
			}
		}
	}()

	// Compile with dev action mode
	compiler := NewCompiler(false, "", "1.0.0")
	compiler.SetActionMode(ActionModeDev)
	compiler.SetNoEmit(false)

	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
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
