package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestResolveWorkflowPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-resolver-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(constants.GetWorkflowDir())
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create test workflow files
	testWorkflow := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(testWorkflow, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "workflow name without extension",
			input:    "test-workflow",
			expected: testWorkflow,
		},
		{
			name:     "workflow name with extension",
			input:    "test-workflow.md",
			expected: testWorkflow,
		},
		{
			name:        "nonexistent workflow",
			input:       "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveWorkflowPath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ResolveWorkflowPath(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWorkflowFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "add .md extension",
			input:    "workflow",
			expected: "workflow.md",
		},
		{
			name:     "already has .md extension",
			input:    "workflow.md",
			expected: "workflow.md",
		},
		{
			name:     "empty string",
			input:    "",
			expected: ".md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWorkflowFile(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWorkflowFile(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
