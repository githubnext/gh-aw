package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWorkflowInputs(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedCount int
		expectedReq   map[string]bool // map of input name to required status
	}{
		{
			name: "workflow with required and optional inputs",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: true
        type: string
      debug_mode:
        description: 'Enable debug mode'
        required: false
        type: boolean
---

# Test Workflow`,
			expectedCount: 2,
			expectedReq: map[string]bool{
				"issue_url":  true,
				"debug_mode": false,
			},
		},
		{
			name: "workflow with no inputs",
			content: `---
on:
  workflow_dispatch:
---

# Test Workflow`,
			expectedCount: 0,
		},
		{
			name: "workflow without workflow_dispatch",
			content: `---
on:
  issues:
    types: [opened]
---

# Test Workflow`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Extract inputs
			inputs, err := getWorkflowInputs(tmpFile)
			if err != nil {
				t.Fatalf("getWorkflowInputs() error = %v", err)
			}

			// Check count
			if len(inputs) != tt.expectedCount {
				t.Errorf("Expected %d inputs, got %d", tt.expectedCount, len(inputs))
			}

			// Check required status
			for name, expectedReq := range tt.expectedReq {
				input, exists := inputs[name]
				if !exists {
					t.Errorf("Expected input '%s' not found", name)
					continue
				}
				if input.Required != expectedReq {
					t.Errorf("Input '%s': expected required=%v, got %v", name, expectedReq, input.Required)
				}
			}
		})
	}
}

func TestValidateWorkflowInputs(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		providedInputs []string
		expectError    bool
		errorContains  []string // strings that should be in the error message
	}{
		{
			name: "all required inputs provided",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: true
        type: string
---

# Test Workflow`,
			providedInputs: []string{"issue_url=https://github.com/owner/repo/issues/123"},
			expectError:    false,
		},
		{
			name: "missing required input",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: true
        type: string
---

# Test Workflow`,
			providedInputs: []string{},
			expectError:    true,
			errorContains:  []string{"Missing required input(s)", "issue_url"},
		},
		{
			name: "typo in input name",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: true
        type: string
---

# Test Workflow`,
			providedInputs: []string{"issue_ur=https://github.com/owner/repo/issues/123"},
			expectError:    true,
			errorContains:  []string{"Invalid input name", "issue_ur", "issue_url"},
		},
		{
			name: "multiple errors: missing required and typo",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: true
        type: string
      debug_mode:
        description: 'Debug mode'
        required: true
        type: boolean
---

# Test Workflow`,
			providedInputs: []string{"debugmode=true"},
			expectError:    true,
			errorContains:  []string{"Missing required input(s)", "issue_url", "Invalid input name", "debugmode"},
		},
		{
			name: "no inputs defined",
			content: `---
on:
  workflow_dispatch:
---

# Test Workflow`,
			providedInputs: []string{"any_input=value"},
			expectError:    false, // No inputs defined, so no validation
		},
		{
			name: "optional input not provided - should not error",
			content: `---
on:
  workflow_dispatch:
    inputs:
      issue_url:
        description: 'Issue URL'
        required: false
        type: string
---

# Test Workflow`,
			providedInputs: []string{},
			expectError:    false,
		},
		{
			name: "unknown input with no close matches",
			content: `---
on:
  workflow_dispatch:
    inputs:
      config_file:
        description: 'Config file path'
        required: false
        type: string
---

# Test Workflow`,
			providedInputs: []string{"xyz=value"},
			expectError:    true,
			errorContains:  []string{"Invalid input name", "xyz", "not a valid input name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Validate inputs
			err := validateWorkflowInputs(tmpFile, tt.providedInputs)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					errStr := err.Error()
					for _, expected := range tt.errorContains {
						if !strings.Contains(errStr, expected) {
							t.Errorf("Expected error to contain '%s', but got: %s", expected, errStr)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
