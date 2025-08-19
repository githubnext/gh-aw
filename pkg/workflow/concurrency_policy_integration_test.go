package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConcurrencyPolicyIntegration(t *testing.T) {
	// Test the new concurrency policy system with real workflow files
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
			name: "workflow with custom concurrency policy",
			frontmatter: `---
name: custom-concurrency-test
on:
  push:
    branches: [main]
concurrency_policy:
  "*":
    group: custom-workflow
    cancel-in-progress: true
  pull_requests:
    group: pr-specific
    node: pull_request.number
    cancel-in-progress: false
---`,
			filename:       "custom-policy.md",
			expectedGroup:  "gh-aw-custom-workflow",
			expectedCancel: boolPtr(true),
			description:    "Should use custom default policy",
		},
		{
			name: "pull request workflow with custom policy",
			frontmatter: `---
name: pr-custom-test
on:
  pull_request:
    types: [opened, synchronize]
concurrency_policy:
  "*":
    group: default-workflow
  pull_requests:
    group: pr-workflow
    node: pull_request.number
    cancel-in-progress: false
---`,
			filename:       "pr-custom.md",
			expectedGroup:  "gh-aw-pr-workflow-${{ github.event.pull_request.number }}",
			expectedCancel: boolPtr(false),
			description:    "Should use PR-specific policy overriding cancel behavior",
		},
		{
			name: "issue workflow with node override",
			frontmatter: `---
name: issue-custom-test
on:
  issues:
    types: [opened]
concurrency_policy:
  issues:
    group: workflow
    node: "github.event.issue.title"
    cancel-in-progress: true
---`,
			filename:       "issue-custom.md",
			expectedGroup:  "gh-aw-${{ github.workflow }}-${{ github.event.issue.title }}",
			expectedCancel: boolPtr(true),
			description:    "Should use custom node expression for issue workflow",
		},
		{
			name: "workflow with backwards compatible id field",
			frontmatter: `---
name: backwards-compat-test
on:
  push:
    branches: [main]
concurrency_policy:
  "*":
    id: legacy-workflow  # using "id" instead of "group"
---`,
			filename:       "backwards-compat.md",
			expectedGroup:  "gh-aw-legacy-workflow",
			expectedCancel: nil,
			description:    "Should support backwards compatible 'id' field",
		},
		{
			name: "schedule workflow with specific policy",
			frontmatter: `---
name: schedule-test
on:
  schedule:
    - cron: "0 9 * * *"
concurrency_policy:
  schedule:
    group: scheduled-tasks
    cancel-in-progress: false
---`,
			filename:       "schedule.md",
			expectedGroup:  "gh-aw-scheduled-tasks",
			expectedCancel: nil,
			description:    "Should use schedule-specific policy",
		},
		{
			name: "workflow with custom trigger policy",
			frontmatter: `---
name: custom-trigger-test
on:
  repository_dispatch:
    types: [custom-event]
concurrency_policy:
  "*":
    group: workflow
  repository_dispatch:
    group: dispatch-workflow
    node: "github.event.client_payload.id"
---`,
			filename:       "custom-trigger.md",
			expectedGroup:  "gh-aw-dispatch-workflow-${{ github.event.client_payload.id }}",
			expectedCancel: nil,
			description:    "Should use custom trigger-specific policy",
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

func TestConcurrencyPolicyMerging(t *testing.T) {
	// Test that user policies correctly override default policies
	tmpDir, err := os.MkdirTemp("", "concurrency-policy-merging-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	// Test case: PR workflow with user policy that overrides the default PR behavior
	frontmatter := `---
name: pr-override-test
on:
  pull_request:
    types: [opened, synchronize]
concurrency_policy:
  pull_requests:
    group: custom-pr-group
    node: "github.sha"
    cancel-in-progress: false
---`

	testContent := frontmatter + "\n\nThis is a test for policy merging."
	testFile := filepath.Join(tmpDir, "pr-override.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Errorf("Failed to parse workflow: %v", err)
		return
	}

	// Verify that the user policy overrode the defaults
	expectedGroup := "gh-aw-custom-pr-group-${{ github.sha }}"
	if !strings.Contains(workflowData.Concurrency, expectedGroup) {
		t.Errorf("Expected custom group '%s', got: %s", expectedGroup, workflowData.Concurrency)
	}

	// Verify that cancellation was overridden to false (should not appear)
	if strings.Contains(workflowData.Concurrency, "cancel-in-progress: true") {
		t.Errorf("Expected cancel-in-progress to be disabled, but found it enabled: %s", workflowData.Concurrency)
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
concurrency: |
  concurrency:
    group: user-defined-group
    cancel-in-progress: true
concurrency_policy:
  "*":
    group: policy-group
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

	// Should use the user-defined concurrency, not the policy
	if !strings.Contains(workflowData.Concurrency, "user-defined-group") {
		t.Errorf("Expected user-defined concurrency to be preserved, got: %s", workflowData.Concurrency)
	}

	// Should not contain policy-generated content
	if strings.Contains(workflowData.Concurrency, "policy-group") {
		t.Errorf("User-defined concurrency should not be overridden by policy, got: %s", workflowData.Concurrency)
	}
}
