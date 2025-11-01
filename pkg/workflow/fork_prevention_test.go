package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForkPreventionForPullRequestTriggers tests that all pull_request triggers
// automatically prevent execution from forked repositories
func TestForkPreventionForPullRequestTriggers(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         string
		markdown            string
		shouldHaveForkCheck bool
		description         string
	}{
		{
			name: "pull_request trigger gets fork prevention",
			frontmatter: `---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the pull request.",
			shouldHaveForkCheck: true,
			description:         "pull_request trigger should automatically add fork prevention",
		},
		{
			name: "pull_request with types gets fork prevention",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited, synchronize]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the pull request.",
			shouldHaveForkCheck: true,
			description:         "pull_request with types should add fork prevention",
		},
		{
			name: "pull_request_target trigger gets fork prevention",
			frontmatter: `---
on:
  pull_request_target:
    types: [opened]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the pull request.",
			shouldHaveForkCheck: true,
			description:         "pull_request_target should add fork prevention",
		},
		{
			name: "pull_request_review gets fork prevention",
			frontmatter: `---
on:
  pull_request_review:
    types: [submitted]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the review.",
			shouldHaveForkCheck: true,
			description:         "pull_request_review should add fork prevention",
		},
		{
			name: "pull_request_review_comment gets fork prevention",
			frontmatter: `---
on:
  pull_request_review_comment:
    types: [created]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the comment.",
			shouldHaveForkCheck: true,
			description:         "pull_request_review_comment should add fork prevention",
		},
		{
			name: "issues trigger does not get fork prevention",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
tools:
  github:
    allowed: [get_issue]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the issue.",
			shouldHaveForkCheck: false,
			description:         "issues trigger should NOT add fork prevention",
		},
		{
			name: "push trigger does not get fork prevention",
			frontmatter: `---
on:
  push:
    branches: [main]
permissions:
  contents: read
tools:
  github:
    allowed: []
---`,
			markdown:            "# Test Workflow\n\nRun on push.",
			shouldHaveForkCheck: false,
			description:         "push trigger should NOT add fork prevention",
		},
		{
			name: "multiple triggers with pull_request gets fork prevention",
			frontmatter: `---
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
permissions:
  contents: read
tools:
  github:
    allowed: [get_issue, get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze issues and PRs.",
			shouldHaveForkCheck: true,
			description:         "mixed triggers with pull_request should add fork prevention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "fork-prevention-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create workflow file
			workflowContent := tt.frontmatter + "\n" + tt.markdown
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			workflowData, err := compiler.ParseWorkflowFile(workflowPath)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Generate YAML
			yamlContent, err := compiler.generateYAML(workflowData, workflowPath)
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check for fork prevention condition in the agent job
			// The fork prevention should be in the agent job's if condition
			// Expected pattern: github.event.pull_request.head.repo.full_name == github.repository
			hasForkCheck := strings.Contains(yamlContent, "github.event.pull_request.head.repo.full_name == github.repository")

			if hasForkCheck != tt.shouldHaveForkCheck {
				t.Errorf("%s: hasForkCheck = %v, want %v", tt.description, hasForkCheck, tt.shouldHaveForkCheck)
				// Print relevant parts of the YAML for debugging
				if tt.shouldHaveForkCheck {
					// Find the agent job section
					agentPos := strings.Index(yamlContent, "  agent:")
					if agentPos != -1 {
						// Print 50 lines after agent job
						lines := strings.Split(yamlContent[agentPos:], "\n")
						if len(lines) > 50 {
							lines = lines[:50]
						}
						t.Logf("Agent job section:\n%s", strings.Join(lines, "\n"))
					}
				}
			}
		})
	}
}

// TestForkPreventionWithExistingCondition tests that fork prevention
// is properly combined with existing if conditions
func TestForkPreventionWithExistingCondition(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      string
		markdown         string
		expectedPatterns []string
		description      string
	}{
		{
			name: "pull_request with existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
if: github.actor != 'dependabot[bot]'
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown: "# Test Workflow\n\nAnalyze the PR.",
			expectedPatterns: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"github.actor != 'dependabot[bot]'",
			},
			description: "should combine fork check with existing condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "fork-prevention-condition-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create workflow file
			workflowContent := tt.frontmatter + "\n" + tt.markdown
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			workflowData, err := compiler.ParseWorkflowFile(workflowPath)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Generate YAML
			yamlContent, err := compiler.generateYAML(workflowData, workflowPath)
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check that all expected patterns are present
			for _, pattern := range tt.expectedPatterns {
				if !strings.Contains(yamlContent, pattern) {
					t.Errorf("%s: expected pattern not found: %s", tt.description, pattern)
					// Print agent job for debugging
					agentPos := strings.Index(yamlContent, "  agent:")
					if agentPos != -1 {
						lines := strings.Split(yamlContent[agentPos:], "\n")
						if len(lines) > 50 {
							lines = lines[:50]
						}
						t.Logf("Agent job section:\n%s", strings.Join(lines, "\n"))
					}
				}
			}
		})
	}
}

// TestForkPreventionWithFrontmatterField tests that the forks: true/false field
// in frontmatter controls automatic fork prevention
func TestForkPreventionWithFrontmatterField(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         string
		markdown            string
		shouldHaveForkCheck bool
		description         string
	}{
		{
			name: "pull_request without forks field (default: false)",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the PR.",
			shouldHaveForkCheck: true,
			description:         "default behavior should add fork prevention",
		},
		{
			name: "pull_request with forks: false",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
    forks: false
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the PR.",
			shouldHaveForkCheck: true,
			description:         "forks: false should add fork prevention",
		},
		{
			name: "pull_request with forks: true",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
    forks: true
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the PR.",
			shouldHaveForkCheck: false,
			description:         "forks: true should disable fork prevention",
		},
		{
			name: "pull_request_target with forks: true",
			frontmatter: `---
on:
  pull_request_target:
    types: [opened]
    forks: true
permissions:
  contents: read
tools:
  github:
    allowed: [get_pull_request]
---`,
			markdown:            "# Test Workflow\n\nAnalyze the PR.",
			shouldHaveForkCheck: false,
			description:         "pull_request_target with forks: true should disable fork prevention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test files
			tmpDir, err := os.MkdirTemp("", "forks-frontmatter-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create workflow file
			workflowContent := tt.frontmatter + "\n" + tt.markdown
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			workflowData, err := compiler.ParseWorkflowFile(workflowPath)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Generate YAML
			yamlContent, err := compiler.generateYAML(workflowData, workflowPath)
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check for fork prevention condition
			hasForkCheck := strings.Contains(yamlContent, "github.event.pull_request.head.repo.full_name == github.repository")

			if hasForkCheck != tt.shouldHaveForkCheck {
				t.Errorf("%s: hasForkCheck = %v, want %v", tt.description, hasForkCheck, tt.shouldHaveForkCheck)
				// Print agent job for debugging
				agentPos := strings.Index(yamlContent, "  agent:")
				if agentPos != -1 {
					lines := strings.Split(yamlContent[agentPos:], "\n")
					if len(lines) > 20 {
						lines = lines[:20]
					}
					t.Logf("Agent job section:\n%s", strings.Join(lines, "\n"))
				}
			}
		})
	}
}
