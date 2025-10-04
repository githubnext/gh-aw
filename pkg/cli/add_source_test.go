package cli

import (
	"strings"
	"testing"
)

// TestBuildSourceString tests the buildSourceString function
func TestBuildSourceString(t *testing.T) {
	tests := []struct {
		name     string
		workflow *WorkflowSpec
		expected string
	}{
		{
			name: "full_spec_with_version",
			workflow: &WorkflowSpec{
				Repo:         "githubnext/agentics",
				WorkflowPath: "workflows/ci-doctor.md",
				Version:      "v1.0.0",
			},
			expected: "githubnext/agentics/workflows/ci-doctor.md@v1.0.0",
		},
		{
			name: "spec_without_version",
			workflow: &WorkflowSpec{
				Repo:         "githubnext/agentics",
				WorkflowPath: "workflows/ci-doctor.md",
				Version:      "",
			},
			expected: "githubnext/agentics/workflows/ci-doctor.md",
		},
		{
			name: "spec_with_branch",
			workflow: &WorkflowSpec{
				Repo:         "githubnext/agentics",
				WorkflowPath: "workflows/daily-plan.md",
				Version:      "main",
			},
			expected: "githubnext/agentics/workflows/daily-plan.md@main",
		},
		{
			name: "empty_repo",
			workflow: &WorkflowSpec{
				Repo:         "",
				WorkflowPath: "workflows/test.md",
				Version:      "v1.0.0",
			},
			expected: "",
		},
		{
			name: "empty_workflow_path",
			workflow: &WorkflowSpec{
				Repo:         "githubnext/agentics",
				WorkflowPath: "",
				Version:      "v1.0.0",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSourceString(tt.workflow)
			if result != tt.expected {
				t.Errorf("buildSourceString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAddSourceToWorkflow tests the addSourceToWorkflow function
func TestAddSourceToWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		source      string
		expectError bool
		checkSource bool
	}{
		{
			name: "add_source_to_workflow_with_frontmatter",
			content: `---
on: push
permissions:
  contents: read
engine: claude
---

# Test Workflow

This is a test workflow.`,
			source:      "githubnext/agentics/workflows/ci-doctor.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
		{
			name: "add_source_to_workflow_without_frontmatter",
			content: `# Test Workflow

This is a test workflow without frontmatter.`,
			source:      "githubnext/agentics/workflows/test.md@main",
			expectError: false,
			checkSource: true,
		},
		{
			name: "add_source_to_existing_workflow_with_fields",
			content: `---
description: "Test workflow description"
on: push
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---

# Test Workflow

This is a test workflow.`,
			source:      "githubnext/agentics/workflows/complex.md@v1.0.0",
			expectError: false,
			checkSource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := addSourceToWorkflow(tt.content, tt.source, false)

			if tt.expectError && err == nil {
				t.Errorf("addSourceToWorkflow() expected error, got nil")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("addSourceToWorkflow() error = %v", err)
				return
			}

			if !tt.expectError && tt.checkSource {
				// Verify that the source field is present in the result
				if !strings.Contains(result, "source:") {
					t.Errorf("addSourceToWorkflow() result does not contain 'source:' field")
				}
				if !strings.Contains(result, tt.source) {
					t.Errorf("addSourceToWorkflow() result does not contain source value '%s'", tt.source)
				}

				// Verify that frontmatter delimiters are present
				if !strings.Contains(result, "---") {
					t.Errorf("addSourceToWorkflow() result does not contain frontmatter delimiters")
				}

				// Verify that markdown content is preserved
				if strings.Contains(tt.content, "# Test Workflow") && !strings.Contains(result, "# Test Workflow") {
					t.Errorf("addSourceToWorkflow() result does not preserve markdown content")
				}
			}
		})
	}
}
