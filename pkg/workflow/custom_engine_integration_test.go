//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomEngineWorkflowCompilation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "custom-engine-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name             string
		content          string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "custom engine with simple steps",
			content: `---
on: push
permissions:
  contents: read
  issues: write
engine:
  id: custom
  steps:
    - name: Setup Node.js
      uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
      with:
        node-version: '18'
    - name: Run tests
      run: |
        echo "Running tests..."
        npm test
---

# Custom Engine Test Workflow

This workflow uses the custom engine to execute defined steps.`,
			shouldContain: []string{
				"- name: Setup Node.js",
				"uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020",
				"node-version: \"18\"",
				"- name: Run tests",
				"echo \"Running tests...\"",
				"npm test",
				"- name: Ensure log file exists",
				"Custom steps execution completed",
			},
			shouldNotContain: []string{
				"claude",
				"codex",
				"ANTHROPIC_API_KEY",
				"OPENAI_API_KEY",
			},
		},
		{
			name: "custom engine with single step",
			content: `---
on: pull_request
engine:
  id: custom
  steps:
    - name: Hello World
      run: echo "Hello from custom engine!"
---

# Single Step Custom Workflow

Simple custom workflow with one step.`,
			shouldContain: []string{
				"- name: Hello World",
				"echo \"Hello from custom engine!\"",
				"- name: Ensure log file exists",
			},
			shouldNotContain: []string{
				"ANTHROPIC_API_KEY",
				"OPENAI_API_KEY",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-custom-workflow.md")
			if err := os.WriteFile(testFile, []byte(test.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")
			compiler.SetSkipValidation(true) // Skip validation for test simplicity

			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated .lock.yml file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			contentStr := string(content)

			// Check that expected strings are present
			for _, expected := range test.shouldContain {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Expected generated workflow to contain '%s', but it was missing", expected)
				}
			}

			// Check that unwanted strings are not present
			for _, unwanted := range test.shouldNotContain {
				if strings.Contains(contentStr, unwanted) {
					t.Errorf("Expected generated workflow to NOT contain '%s', but it was present", unwanted)
				}
			}

			// Verify that the custom steps are properly formatted YAML
			if !strings.Contains(contentStr, "name: Setup Node.js") || !strings.Contains(contentStr, "uses: actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020") {
				// This is expected for the first test only
				if test.name == "custom engine with simple steps" {
					t.Error("Custom engine steps were not properly formatted in the generated workflow")
				}
			}
		})
	}
}

func TestCustomEngineWithoutSteps(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "custom-engine-no-steps-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `---
on: push
engine:
  id: custom
---

# Custom Engine Without Steps

This workflow uses the custom engine but doesn't define any steps.`

	testFile := filepath.Join(tmpDir, "test-custom-no-steps.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)

	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content_bytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	contentStr := string(content_bytes)

	// Should still contain the log file creation step
	if !strings.Contains(contentStr, "Custom steps execution completed") {
		t.Error("Expected workflow to contain log file creation even without custom steps")
	}
}
