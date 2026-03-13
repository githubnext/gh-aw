//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

// TestActionModeValidation tests the ActionMode type validation
func TestActionModeValidation(t *testing.T) {
	tests := []struct {
		mode  ActionMode
		valid bool
	}{
		{ActionModeDev, true},
		{ActionModeRelease, true},
		{ActionModeScript, true},
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
		{ActionModeDev, "dev"},
		{ActionModeRelease, "release"},
		{ActionModeScript, "script"},
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
	compiler := NewCompilerWithVersion("1.0.0")
	if compiler.GetActionMode() != ActionModeDev {
		t.Errorf("Default action mode should be dev, got %s", compiler.GetActionMode())
	}
}

// TestCompilerSetActionMode tests setting the action mode
func TestCompilerSetActionMode(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	compiler.SetActionMode(ActionModeRelease)
	if compiler.GetActionMode() != ActionModeRelease {
		t.Errorf("Expected action mode release, got %s", compiler.GetActionMode())
	}

	compiler.SetActionMode(ActionModeDev)
	if compiler.GetActionMode() != ActionModeDev {
		t.Errorf("Expected action mode dev, got %s", compiler.GetActionMode())
	}

	compiler.SetActionMode(ActionModeScript)
	if compiler.GetActionMode() != ActionModeScript {
		t.Errorf("Expected action mode script, got %s", compiler.GetActionMode())
	}
}

// TestActionModeIsScript tests the IsScript() method
func TestActionModeIsScript(t *testing.T) {
	tests := []struct {
		mode     ActionMode
		isScript bool
	}{
		{ActionModeDev, false},
		{ActionModeRelease, false},
		{ActionModeScript, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			if got := tt.mode.IsScript(); got != tt.isScript {
				t.Errorf("ActionMode(%q).IsScript() = %v, want %v", tt.mode, got, tt.isScript)
			}
		})
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
	compiler := NewCompilerWithVersion("1.0.0")
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

// TestScriptActionModeCompilation tests workflow compilation with script mode
func TestScriptActionModeCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test workflow file with action-mode: script feature flag
	workflowContent := `---
name: Test Script Mode
on: workflow_dispatch
features:
  action-mode: "script"
permissions:
  contents: read
---

Test workflow with script mode.
`

	workflowPath := tempDir + "/test-workflow.md"
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile with script mode (will be overridden by feature flag)
	compiler := NewCompilerWithVersion("1.0.0")
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

	// Verify script mode behavior:
	// 1. Checkout should use repository: github/gh-aw
	if !strings.Contains(lockStr, "repository: github/gh-aw") {
		t.Error("Expected 'repository: github/gh-aw' in checkout step for script mode")
	}

	// 2. Checkout should target path: /tmp/gh-aw/actions-source
	if !strings.Contains(lockStr, "path: /tmp/gh-aw/actions-source") {
		t.Error("Expected 'path: /tmp/gh-aw/actions-source' in checkout step for script mode")
	}

	// 3. Checkout should use shallow clone (fetch-depth: 1)
	if !strings.Contains(lockStr, "fetch-depth: 1") {
		t.Error("Expected 'fetch-depth: 1' in checkout step for script mode (shallow checkout)")
	}

	// 4. Setup step should run bash script instead of using "uses:"
	if !strings.Contains(lockStr, "bash /tmp/gh-aw/actions-source/actions/setup/setup.sh") {
		t.Error("Expected setup script to run bash directly in script mode")
	}

	// 5. Setup step should have INPUT_DESTINATION environment variable
	if !strings.Contains(lockStr, "INPUT_DESTINATION: /opt/gh-aw/actions") {
		t.Error("Expected INPUT_DESTINATION environment variable in setup step for script mode")
	}

	// 6. Should not use "uses:" for setup action in script mode
	setupActionPattern := "uses: ./actions/setup"
	if strings.Contains(lockStr, setupActionPattern) {
		t.Error("Expected script mode to NOT use 'uses: ./actions/setup' but instead run bash script directly")
	}

	// 7. Checkout should include ref: for the version
	if !strings.Contains(lockStr, "ref: 1.0.0") {
		t.Error("Expected 'ref: 1.0.0' in checkout step for script mode when version is set")
	}
}

// TestVersionToGitRef tests the versionToGitRef helper function used to derive
// a clean git ref from `git describe` output for use in actions/checkout ref: fields.
func TestVersionToGitRef(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "empty version returns empty ref",
			version: "",
			want:    "",
		},
		{
			name:    "dev version returns empty ref",
			version: "dev",
			want:    "",
		},
		{
			name:    "plain short SHA used as-is",
			version: "e284d1e",
			want:    "e284d1e",
		},
		{
			name:    "short SHA with -dirty suffix stripped",
			version: "e284d1e-dirty",
			want:    "e284d1e",
		},
		{
			name:    "simple version tag used as-is",
			version: "v1.2.3",
			want:    "v1.2.3",
		},
		{
			name:    "version tag with -dirty stripped",
			version: "v1.2.3-dirty",
			want:    "v1.2.3",
		},
		{
			name:    "git describe output with N commits extracts SHA",
			version: "v0.57.2-60-ge284d1e",
			want:    "e284d1e",
		},
		{
			name:    "git describe output with -dirty extracts SHA",
			version: "v0.57.2-60-ge284d1e-dirty",
			want:    "e284d1e",
		},
		{
			name:    "numeric version tag used as-is",
			version: "1.0.0",
			want:    "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionToGitRef(tt.version); got != tt.want {
				t.Errorf("versionToGitRef(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

// TestCheckoutActionsFolderDevModeHasRepository verifies that the Checkout actions folder
// step in dev mode includes repository: github/gh-aw so that cross-repo callers (e.g.
// event-driven relays) can find the actions/ directory instead of defaulting to the
// caller's repo which has no actions/ directory.
func TestCheckoutActionsFolderDevModeHasRepository(t *testing.T) {
	compiler := NewCompilerWithVersion("dev")
	compiler.SetActionMode(ActionModeDev)

	lines := compiler.generateCheckoutActionsFolder(nil)
	combined := strings.Join(lines, "")

	if !strings.Contains(combined, "repository: github/gh-aw") {
		t.Error("Dev mode Checkout actions folder should include 'repository: github/gh-aw' (fix for #20658)")
	}

	// When version is "dev", no ref: should be emitted
	if strings.Contains(combined, "ref:") {
		t.Error("Dev mode with 'dev' version should not include ref: field")
	}
}

// TestCheckoutActionsFolderDevModeWithVersionHasRef verifies that when a real version
// SHA is set, the Checkout actions folder step in dev mode includes both
// repository: and ref: fields.
func TestCheckoutActionsFolderDevModeWithVersionHasRef(t *testing.T) {
	compiler := NewCompilerWithVersion("e284d1e")
	compiler.SetActionMode(ActionModeDev)

	lines := compiler.generateCheckoutActionsFolder(nil)
	combined := strings.Join(lines, "")

	if !strings.Contains(combined, "repository: github/gh-aw") {
		t.Error("Dev mode Checkout actions folder should include 'repository: github/gh-aw'")
	}
	if !strings.Contains(combined, "ref: e284d1e") {
		t.Error("Dev mode Checkout actions folder should include 'ref: e284d1e' when version is set")
	}
}
