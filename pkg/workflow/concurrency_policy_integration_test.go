package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConcurrencyPolicyIntegration(t *testing.T) {
	// Test the concurrency policy system with real workflow files using code-based rules
	tmpDir, err := os.MkdirTemp("", "concurrency-policy-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		frontmatter    string
		filename       string
		expectedGroup  string
		expectedCancel *bool
		description    string
	}{
		{
			name: "basic push workflow",
			frontmatter: `---
name: basic-push-test
on:
  push:
    branches: [main]
---`,
			filename:       "basic-push.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}",
			expectedCancel: nil,
			description:    "Should use basic group for push workflows",
		},
		{
			name: "pull request workflow",
			frontmatter: `---
name: pr-test
on:
  pull_request:
    types: [opened, synchronize]
---`,
			filename:       "pr.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}-${{ github.ref }}",
			expectedCancel: boolPtr(true),
			description:    "Should use ref-based group with cancellation for PR workflows",
		},
		{
			name: "issue workflow",
			frontmatter: `---
name: issue-test
on:
  issues:
    types: [opened]
---`,
			filename:       "issue.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}",
			expectedCancel: boolPtr(true),
			description:    "Should use issue number with cancellation for issue workflows",
		},
		{
			name: "schedule workflow",
			frontmatter: `---
name: schedule-test
on:
  schedule:
    - cron: "0 9 * * *"
---`,
			filename:       "schedule.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}",
			expectedCancel: nil,
			description:    "Should use basic group for scheduled workflows",
		},
		{
			name: "workflow_dispatch workflow",
			frontmatter: `---
name: manual-test
on:
  workflow_dispatch:
---`,
			filename:       "manual.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}",
			expectedCancel: nil,
			description:    "Should use basic group for manual workflows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + "\n\nThis is a test workflow for concurrency policy."

			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Parse the workflow to get its data
			workflowData, err := compiler.parseWorkflowFile(testFile)
			if err != nil {
				t.Errorf("Failed to parse workflow: %v", err)
				return
			}

			t.Logf("Workflow: %s", tt.description)
			t.Logf("  Expected Group: %s", tt.expectedGroup)
			t.Logf("  Generated Concurrency: %s", workflowData.Concurrency)

			// Check that the concurrency field contains the expected group
			if !strings.Contains(workflowData.Concurrency, tt.expectedGroup) {
				t.Errorf("Expected concurrency to contain group '%s', got: %s", tt.expectedGroup, workflowData.Concurrency)
			}

			// Check for cancel-in-progress behavior
			hasCancel := strings.Contains(workflowData.Concurrency, "cancel-in-progress: true")
			if tt.expectedCancel != nil {
				if *tt.expectedCancel && !hasCancel {
					t.Errorf("Expected cancel-in-progress: true, but not found in: %s", workflowData.Concurrency)
				} else if !*tt.expectedCancel && hasCancel {
					t.Errorf("Did not expect cancel-in-progress: true, but found in: %s", workflowData.Concurrency)
				}
			}

			// Ensure it's valid YAML format
			if !strings.HasPrefix(workflowData.Concurrency, "concurrency:") {
				t.Errorf("Generated concurrency should start with 'concurrency:', got: %s", workflowData.Concurrency)
			}
		})
	}
}

func TestConcurrencyUserOverrideRespected(t *testing.T) {
	// Test that user-provided concurrency is not overridden by the policy system
	tmpDir, err := os.MkdirTemp("", "concurrency-user-override-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	frontmatter := `---
name: user-override-test
on:
  push:
    branches: [main]
concurrency: |
  concurrency:
    group: user-defined-group
    cancel-in-progress: true
---`

	testContent := frontmatter + "\n\nThis is a test for user override."
	testFile := filepath.Join(tmpDir, "user-override.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Errorf("Failed to parse workflow: %v", err)
		return
	}

	// Should use the user-defined concurrency, not the auto-generated policy
	if !strings.Contains(workflowData.Concurrency, "user-defined-group") {
		t.Errorf("Expected user-defined concurrency to be preserved, got: %s", workflowData.Concurrency)
	}

	// Should not contain auto-generated content
	if strings.Contains(workflowData.Concurrency, "gh-aw-${{ github.workflow }}") {
		t.Errorf("User-defined concurrency should not be overridden by auto-generated policy, got: %s", workflowData.Concurrency)
	}
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}
