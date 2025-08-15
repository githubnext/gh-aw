package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConcurrencyRules(t *testing.T) {
	// Test the new concurrency rules for pull_request and alias workflows
	tmpDir, err := os.MkdirTemp("", "concurrency-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                string
		frontmatter         string
		filename            string
		expectedConcurrency string
		shouldHaveCancel    bool
		description         string
	}{
		{
			name: "PR workflow should have dynamic concurrency with cancel",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
  github:
    allowed: [list_issues]
---`,
			filename: "pr-workflow.md",
			expectedConcurrency: `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.ref }}"
  cancel-in-progress: true`,
			shouldHaveCancel: true,
			description:      "PR workflows should use dynamic concurrency with ref and cancellation",
		},
		{
			name: "alias workflow should have dynamic concurrency without cancel",
			frontmatter: `---
on:
  alias:
    name: test-bot
tools:
  github:
    allowed: [list_issues]
---`,
			filename: "alias-workflow.md",
			expectedConcurrency: `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}"`,
			shouldHaveCancel: false,
			description:      "Alias workflows should use dynamic concurrency with ref but without cancellation",
		},
		{
			name: "regular workflow should use static concurrency without cancel",
			frontmatter: `---
on:
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
    allowed: [list_issues]
---`,
			filename: "regular-workflow.md",
			expectedConcurrency: `concurrency:
  group: "gh-aw-${{ github.workflow }}"`,
			shouldHaveCancel: false,
			description:      "Regular workflows should use static concurrency without cancellation",
		},
		{
			name: "push workflow should use static concurrency without cancel",
			frontmatter: `---
on:
  push:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---`,
			filename: "push-workflow.md",
			expectedConcurrency: `concurrency:
  group: "gh-aw-${{ github.workflow }}"`,
			shouldHaveCancel: false,
			description:      "Push workflows should use static concurrency without cancellation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Concurrency Workflow

This is a test workflow for concurrency behavior.
`

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
			t.Logf("  On: %s", workflowData.On)
			t.Logf("  Concurrency: %s", workflowData.Concurrency)

			// Check that the concurrency field matches expected pattern
			if !strings.Contains(workflowData.Concurrency, "gh-aw-${{ github.workflow }}") {
				t.Errorf("Expected concurrency to use gh-aw-${{ github.workflow }}, got: %s", workflowData.Concurrency)
			}

			// Check for cancel-in-progress based on workflow type
			hasCancel := strings.Contains(workflowData.Concurrency, "cancel-in-progress: true")
			if tt.shouldHaveCancel && !hasCancel {
				t.Errorf("Expected cancel-in-progress: true for %s workflow, but not found in: %s", tt.name, workflowData.Concurrency)
			} else if !tt.shouldHaveCancel && hasCancel {
				t.Errorf("Did not expect cancel-in-progress: true for %s workflow, but found in: %s", tt.name, workflowData.Concurrency)
			}

			// For PR workflows, check for ref inclusion; for alias workflows, check for issue/PR numbers
			isPRWorkflow := strings.Contains(tt.name, "PR workflow")
			isAliasWorkflow := strings.Contains(tt.name, "alias workflow")

			if isPRWorkflow {
				if !strings.Contains(workflowData.Concurrency, "github.ref") {
					t.Errorf("Expected concurrency to include github.ref for %s workflow, got: %s", tt.name, workflowData.Concurrency)
				}
			} else if isAliasWorkflow {
				if !strings.Contains(workflowData.Concurrency, "github.event.issue.number || github.event.pull_request.number") {
					t.Errorf("Expected concurrency to include issue/PR numbers for %s workflow, got: %s", tt.name, workflowData.Concurrency)
				}
			} else {
				if strings.Contains(workflowData.Concurrency, "github.ref") {
					t.Errorf("Did not expect concurrency to include github.ref for %s workflow, got: %s", tt.name, workflowData.Concurrency)
				}
			}
		})
	}
}

func TestGenerateConcurrencyConfig(t *testing.T) {
	tests := []struct {
		name           string
		workflowData   *WorkflowData
		isAliasTrigger bool
		expected       string
		description    string
	}{
		{
			name: "PR workflow should have dynamic concurrency with cancel",
			workflowData: &WorkflowData{
				On: `on:
  pull_request:
    types: [opened, synchronize]`,
				Concurrency: "", // Empty, should be generated
			},
			isAliasTrigger: false,
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.ref }}"
  cancel-in-progress: true`,
			description: "PR workflows should use dynamic concurrency with ref and cancellation",
		},
		{
			name: "Alias workflow should have dynamic concurrency without cancel",
			workflowData: &WorkflowData{
				On: `on:
  issues:
    types: [opened, edited, reopened]`,
				Concurrency: "", // Empty, should be generated
			},
			isAliasTrigger: true,
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number || github.event.pull_request.number }}"`,
			description: "Alias workflows should use dynamic concurrency with ref but without cancellation",
		},
		{
			name: "Regular workflow should use static concurrency without cancel",
			workflowData: &WorkflowData{
				On: `on:
  schedule:
    - cron: "0 9 * * 1"`,
				Concurrency: "", // Empty, should be generated
			},
			isAliasTrigger: false,
			expected: `concurrency:
  group: "gh-aw-${{ github.workflow }}"`,
			description: "Regular workflows should use static concurrency without cancellation",
		},
		{
			name: "Existing concurrency should not be overridden",
			workflowData: &WorkflowData{
				On: `on:
  pull_request:
    types: [opened, synchronize]`,
				Concurrency: `concurrency:
  group: "custom-group"`,
			},
			isAliasTrigger: false,
			expected: `concurrency:
  group: "custom-group"`,
			description: "Existing concurrency configuration should be preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateConcurrencyConfig(tt.workflowData, tt.isAliasTrigger)

			if result != tt.expected {
				t.Errorf("GenerateConcurrencyConfig() failed for %s\nExpected:\n%s\nGot:\n%s", tt.description, tt.expected, result)
			}
		})
	}
}

func TestIsPullRequestWorkflow(t *testing.T) {
	tests := []struct {
		name     string
		on       string
		expected bool
	}{
		{
			name: "Pull request workflow should be identified",
			on: `on:
  pull_request:
    types: [opened, synchronize]`,
			expected: true,
		},
		{
			name: "Pull request review comment workflow should be identified",
			on: `on:
  pull_request_review_comment:
    types: [created]`,
			expected: true,
		},
		{
			name: "Schedule workflow should not be identified as PR workflow",
			on: `on:
  schedule:
    - cron: "0 9 * * 1"`,
			expected: false,
		},
		{
			name: "Issues workflow should not be identified as PR workflow",
			on: `on:
  issues:
    types: [opened, edited]`,
			expected: false,
		},
		{
			name: "Mixed workflow with PR should be identified",
			on: `on:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, synchronize]`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPullRequestWorkflow(tt.on)
			if result != tt.expected {
				t.Errorf("isPullRequestWorkflow() for %s = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}
