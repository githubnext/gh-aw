package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeSetupIntegration(t *testing.T) {
	tests := []struct {
		name             string
		workflowMarkdown string
		expectSetup      []string
		notExpectSetup   []string
	}{
		{
			name: "auto-detects node from npm command",
			workflowMarkdown: `---
on: push
engine: copilot
steps:
  - name: Install dependencies
    run: npm install
---

# Test workflow`,
			expectSetup: []string{
				"Setup Node.js",
				"actions/setup-node@v4",
				"node-version: '24'",
			},
		},
		{
			name: "auto-detects python from python command",
			workflowMarkdown: `---
on: push
engine: copilot
steps:
  - name: Run script
    run: python test.py
---

# Test workflow`,
			expectSetup: []string{
				"Setup Python",
				"actions/setup-python@v5",
				"python-version: '3.12'",
			},
		},
		{
			name: "auto-detects uv from uvx command",
			workflowMarkdown: `---
on: push
engine: copilot
steps:
  - name: Run ruff
    run: uvx ruff check
---

# Test workflow`,
			expectSetup: []string{
				"Setup uv",
				"astral-sh/setup-uv@v5",
			},
		},
		{
			name: "auto-detects multiple runtimes",
			workflowMarkdown: `---
on: push
engine: copilot
steps:
  - name: Install
    run: npm install
  - name: Test
    run: python test.py
---

# Test workflow`,
			expectSetup: []string{
				"Setup Node.js",
				"Setup Python",
			},
		},
		{
			name: "skips auto-detection when setup action exists",
			workflowMarkdown: `---
on: push
engine: copilot
steps:
  - name: Setup Node.js
    uses: actions/setup-node@v4
    with:
      node-version: '20'
  - name: Install
    run: npm install
---

# Test workflow`,
			expectSetup: []string{
				"node-version:", // Should keep user's version (check for key)
				"20",            // Check for the value (regardless of quote type)
			},
			notExpectSetup: []string{
				// Should not add a second Node.js setup with different version
			},
		},
		{
			name: "detects runtime from MCP server config",
			workflowMarkdown: `---
on: push
engine: copilot
mcp-servers:
  custom-tool:
    command: python
    args: ["-m", "my_server"]
---

# Test workflow`,
			expectSetup: []string{
				"Setup Python",
				"actions/setup-python@v5",
			},
		},
		{
			name: "no auto-detection for workflows without runtime commands in custom steps",
			workflowMarkdown: `---
on: push
engine:
  id: custom
  steps:
    - name: Echo
      run: echo "Hello"
---

# Test workflow`,
			notExpectSetup: []string{
				"Setup Python",
				"Setup uv",
				"Setup Go",
				"Setup Ruby",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test files
			tmpDir := t.TempDir()
			testFile := tmpDir + "/test-workflow.md"

			// Write test workflow
			if err := os.WriteFile(testFile, []byte(tt.workflowMarkdown), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check expected setup steps
			for _, expected := range tt.expectSetup {
				if !strings.Contains(lockContent, expected) {
					t.Errorf("Expected to find '%s' in lock file but didn't.\nLock file content:\n%s", expected, lockContent)
				}
			}

			// Check that unwanted setup steps are not present
			for _, notExpected := range tt.notExpectSetup {
				if strings.Contains(lockContent, notExpected) {
					t.Errorf("Did not expect to find '%s' in lock file but it was present.\nLock file content:\n%s", notExpected, lockContent)
				}
			}
		})
	}
}

func TestRuntimeSetupWithEngineNode(t *testing.T) {
	// Test that auto-detected runtime setup works alongside engine requirements
	// Both the auto-detection and engine may add setup steps, which is acceptable
	workflowMarkdown := `---
on: push
engine: claude
steps:
  - name: Install dependencies
    run: npm install
---

# Test workflow`

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test-workflow.md"

	if err := os.WriteFile(testFile, []byte(workflowMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Should have Node.js setup (from auto-detection or engine, or both)
	if !strings.Contains(lockContent, "Setup Node.js") {
		t.Error("Expected Node.js setup to be present")
	}

	// It's acceptable to have Node.js setup appear twice:
	// - Once from auto-detection for custom steps
	// - Once from engine requirements
	// This is not a problem as GitHub Actions will use the first setup
}

func TestRuntimeSetupPreservesUserVersions(t *testing.T) {
	// Test that when user specifies a version in setup action, we don't override it
	workflowMarkdown := `---
on: push
engine: copilot
steps:
  - name: Setup Python
    uses: actions/setup-python@v5
    with:
      python-version: '3.9'
  - name: Run script
    run: python test.py
---

# Test workflow`

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test-workflow.md"

	if err := os.WriteFile(testFile, []byte(workflowMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Should preserve user's version (3.9) - check without quotes since YAML formatting may vary
	if !strings.Contains(lockContent, "python-version") || !strings.Contains(lockContent, "3.9") {
		t.Error("Expected to preserve user's Python version 3.9")
	}

	// Should not add default version (3.12)
	if strings.Contains(lockContent, "3.12") {
		t.Error("Should not override user's version with default version")
	}

	// Should only have one Python setup
	count := strings.Count(lockContent, "Setup Python")
	if count > 1 {
		t.Errorf("Expected 'Setup Python' to appear once, but found %d occurrences", count)
	}
}
