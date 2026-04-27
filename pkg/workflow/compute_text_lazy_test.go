//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestComputeTextLazyInsertion(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "compute-text-lazy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .git directory to simulate a git repository
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Test case 1: Workflow that uses task.outputs.text
	workflowWithText := `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Workflow With Text Output

This workflow uses the text output: "${{ steps.sanitized.outputs.text }}"

Please analyze this issue and provide a helpful response.`

	workflowWithTextPath := filepath.Join(tempDir, "with-text.md")
	if err := os.WriteFile(workflowWithTextPath, []byte(workflowWithText), 0644); err != nil {
		t.Fatalf("Failed to write workflow with text: %v", err)
	}

	// Test case 2: Workflow that does NOT use task.outputs.text
	workflowWithoutText := `---
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Workflow Without Text Output

This workflow does NOT use the text output.

Create a report based on repository analysis.`

	workflowWithoutTextPath := filepath.Join(tempDir, "without-text.md")
	if err := os.WriteFile(workflowWithoutTextPath, []byte(workflowWithoutText), 0644); err != nil {
		t.Fatalf("Failed to write workflow without text: %v", err)
	}

	compiler := NewCompiler()

	// Test workflow WITH text usage
	t.Run("workflow_with_text_usage", func(t *testing.T) {
		err := compiler.CompileWorkflow(workflowWithTextPath)
		if err != nil {
			t.Fatalf("Failed to compile workflow with text: %v", err)
		}

		// Check that compute-text action was NOT created (JavaScript is now inlined)
		actionPath := filepath.Join(tempDir, ".github", "actions", "compute-text", "action.yml")
		if _, err := os.Stat(actionPath); !os.IsNotExist(err) {
			t.Error("Expected compute-text action NOT to be created (JavaScript should be inlined)")
		}

		// Check that the compiled YAML contains inlined sanitized step
		lockPath := stringutil.MarkdownToLockFile(workflowWithTextPath)
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		lockStr := string(lockContent)
		if !strings.Contains(lockStr, "id: sanitized") {
			t.Error("Expected compiled workflow to contain sanitized step")
		}
		if !strings.Contains(lockStr, "text: ${{ steps.sanitized.outputs.text }}") {
			t.Error("Expected compiled workflow to contain text output referencing sanitized step")
		}
		// Check that JavaScript is inlined instead of using shared action
		if !strings.Contains(lockStr, "uses: actions/github-script@373c709c69115d41ff229c7e5df9f8788daa9553") {
			t.Error("Expected sanitized step to use inlined JavaScript")
		}
		// Check that it does NOT use the old shared action path
		if strings.Contains(lockStr, "uses: ./.github/actions/compute-text") {
			t.Error("Expected sanitized step NOT to use shared compute-text action")
		}
		if strings.Contains(lockStr, "uses: ./.github/actions/sanitized") {
			t.Error("Expected sanitized step NOT to use shared sanitized action")
		}
	})

	// Clean up for next test
	os.RemoveAll(filepath.Join(tempDir, ".github"))

	// Test workflow WITHOUT text usage
	t.Run("workflow_without_text_usage", func(t *testing.T) {
		err := compiler.CompileWorkflow(workflowWithoutTextPath)
		if err != nil {
			t.Fatalf("Failed to compile workflow without text: %v", err)
		}

		// Check that the action was NOT created
		actionPath := filepath.Join(tempDir, ".github", "actions", "compute-text", "action.yml")
		if _, err := os.Stat(actionPath); !os.IsNotExist(err) {
			t.Error("Expected compute-text action NOT to be created for workflow that doesn't use text output")
		}

		// Check that the compiled YAML does NOT contain sanitized step
		lockPath := stringutil.MarkdownToLockFile(workflowWithoutTextPath)
		lockContent, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("Failed to read compiled workflow: %v", err)
		}

		lockStr := string(lockContent)
		if strings.Contains(lockStr, "id: sanitized") {
			t.Error("Expected compiled workflow NOT to contain sanitized step")
		}
		if strings.Contains(lockStr, "text: ${{ steps.sanitized.outputs.text }}") {
			t.Error("Expected compiled workflow NOT to contain text output")
		}
	})
}

func TestDetectTextOutputUsage(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name          string
		content       string
		expectedUsage bool
	}{
		{
			name:          "with_text_usage",
			content:       "Analyze this: \"${{ steps.sanitized.outputs.text }}\"",
			expectedUsage: true,
		},
		{
			name:          "without_text_usage",
			content:       "Create a report based on repository analysis.",
			expectedUsage: false,
		},
		{
			name:          "with_other_github_expressions",
			content:       "Repository: ${{ github.repository }} but no text output",
			expectedUsage: false,
		},
		{
			name:          "with_partial_match",
			content:       "Something about task.outputs but not the full expression",
			expectedUsage: false,
		},
		{
			name:          "with_multiple_usages",
			content:       "First: \"${{ steps.sanitized.outputs.text }}\" and second: \"${{ steps.sanitized.outputs.text }}\"",
			expectedUsage: true,
		},
		{
			name:          "with_title_usage",
			content:       "Title: \"${{ steps.sanitized.outputs.title }}\"",
			expectedUsage: true,
		},
		{
			name:          "with_body_usage",
			content:       "Body: \"${{ steps.sanitized.outputs.body }}\"",
			expectedUsage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.detectTextOutputUsage(tt.content)
			if result != tt.expectedUsage {
				t.Errorf("detectTextOutputUsage() = %v, expected %v", result, tt.expectedUsage)
			}
		})
	}
}

func TestHasContentContext(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedContext bool
	}{
		{
			name: "issues_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "pull_request_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types": []string{"opened"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "pull_request_target_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_target": map[string]any{
						"types": []string{"opened"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "issue_comment_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issue_comment": map[string]any{
						"types": []string{"created"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "pull_request_review_comment_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_review_comment": map[string]any{
						"types": []string{"created"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "pull_request_review_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request_review": map[string]any{
						"types": []string{"submitted"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "discussion_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"discussion": map[string]any{
						"types": []string{"created"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "discussion_comment_event",
			frontmatter: map[string]any{
				"on": map[string]any{
					"discussion_comment": map[string]any{
						"types": []string{"created"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "schedule_event_no_context",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []map[string]string{
						{"cron": "0 0 * * *"},
					},
				},
			},
			expectedContext: false,
		},
		{
			name: "push_event_no_context",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			expectedContext: false,
		},
		{
			name: "workflow_dispatch_no_context",
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
			},
			expectedContext: false,
		},
		{
			name: "multiple_events_with_context",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"opened"},
					},
					"workflow_dispatch": map[string]any{},
				},
			},
			expectedContext: true,
		},
		{
			name: "multiple_events_no_context",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
					"workflow_dispatch": map[string]any{},
				},
			},
			expectedContext: false,
		},
		{
			name:            "no_on_field",
			frontmatter:     map[string]any{},
			expectedContext: false,
		},
		{
			name: "slash_command_trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"slash_command": map[string]any{
						"name":   "test",
						"events": []string{"issues", "issue_comment"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "labeled_on_issues",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues": map[string]any{
						"types": []string{"labeled", "unlabeled"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "labeled_on_pull_request",
			frontmatter: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types": []string{"opened", "labeled"},
					},
				},
			},
			expectedContext: true,
		},
		{
			name: "labeled_on_discussion",
			frontmatter: map[string]any{
				"on": map[string]any{
					"discussion": map[string]any{
						"types": []string{"labeled"},
					},
				},
			},
			expectedContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.hasContentContext(tt.frontmatter)
			if result != tt.expectedContext {
				t.Errorf("hasContentContext() = %v, expected %v", result, tt.expectedContext)
			}
		})
	}
}

func TestComputeTextStepIncludesAllowedDomainsEnv(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "compute-text-allowed-domains-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	workflowContent := `---
on:
  issues:
    types: [opened]
  bots: ["dependabot[bot]"]
engine: copilot
strict: false
network:
  allowed:
    - cnn.com
safe-outputs:
  create-issue:
  allowed-domains:
    - bbc.com
---

# Test Workflow

Use the incoming text: "${{ steps.sanitized.outputs.text }}"
`

	workflowPath := filepath.Join(tempDir, "compute-text-allowed-domains.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	lockStr := string(lockContent)
	lines := strings.Split(lockStr, "\n")
	sanitizedIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "id: sanitized" {
			sanitizedIndex = i
			break
		}
	}
	if sanitizedIndex == -1 {
		t.Fatal("Expected compiled workflow to contain sanitized step")
	}

	withIndex := -1
	for i := sanitizedIndex + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "with:" {
			withIndex = i
			break
		}
	}
	if withIndex == -1 {
		t.Fatal("Expected sanitized step to contain a with section")
	}

	sanitizedLines := lines[sanitizedIndex+1 : withIndex]
	envIndex := -1
	envIndent := 0
	for i, line := range sanitizedLines {
		if strings.TrimSpace(line) == "env:" {
			envIndex = i
			envIndent = len(line) - len(strings.TrimLeft(line, " "))
			break
		}
	}
	if envIndex == -1 {
		t.Fatalf("Expected sanitized step to contain an env block, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}

	envLines := make([]string, 0, 2)
	for i := envIndex + 1; i < len(sanitizedLines); i++ {
		line := sanitizedLines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent <= envIndent {
			break
		}
		envLines = append(envLines, strings.TrimSpace(line))
	}
	if len(envLines) == 0 {
		t.Fatalf("Expected sanitized step env block to contain variables, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
	envBlock := strings.Join(envLines, "\n")

	if !strings.Contains(envBlock, "GH_AW_ALLOWED_BOTS:") {
		t.Errorf("Expected sanitized step env to contain GH_AW_ALLOWED_BOTS, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
	if !strings.Contains(envBlock, "dependabot[bot]") {
		t.Errorf("Expected sanitized step GH_AW_ALLOWED_BOTS value to include dependabot[bot], got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
	if !strings.Contains(envBlock, "GH_AW_ALLOWED_DOMAINS:") {
		t.Errorf("Expected sanitized step env to contain GH_AW_ALLOWED_DOMAINS, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
	if !strings.Contains(envBlock, "cnn.com") {
		t.Errorf("Expected sanitized step GH_AW_ALLOWED_DOMAINS to include network.allowed domain cnn.com, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
	if !strings.Contains(envBlock, "bbc.com") {
		t.Errorf("Expected sanitized step GH_AW_ALLOWED_DOMAINS to include safe-outputs.allowed-domains domain bbc.com, got:\n%s", strings.Join(sanitizedLines, "\n"))
	}
}

func TestComputeTextContextBasedInsertion(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "compute-text-context-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .git directory to simulate a git repository
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	tests := []struct {
		name               string
		workflow           string
		expectedSanitized  bool
		expectedTextOutput bool
	}{
		{
			name: "issue_trigger_without_explicit_usage",
			workflow: `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Issue Workflow

Analyze the issue and provide a response.

This workflow does NOT explicitly use text output but should get sanitized step.`,
			expectedSanitized:  true,
			expectedTextOutput: true,
		},
		{
			name: "pr_trigger_without_explicit_usage",
			workflow: `---
on:
  pull_request:
    types: [opened]
permissions:
  pull-requests: read
strict: false
tools:
  github:
    toolsets: [pull_requests]
---

# Test PR Workflow

Review the pull request.

This workflow does NOT explicitly use text output but should get sanitized step.`,
			expectedSanitized:  true,
			expectedTextOutput: true,
		},
		{
			name: "discussion_trigger_without_explicit_usage",
			workflow: `---
on:
  discussion:
    types: [created]
permissions:
  discussions: read
strict: false
tools:
  github:
    toolsets: [discussions]
---

# Test Discussion Workflow

Respond to the discussion.

This workflow does NOT explicitly use text output but should get sanitized step.`,
			expectedSanitized:  true,
			expectedTextOutput: true,
		},
		{
			name: "issue_comment_trigger_without_explicit_usage",
			workflow: `---
on:
  issue_comment:
    types: [created]
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Comment Workflow

Respond to the comment.

This workflow does NOT explicitly use text output but should get sanitized step.`,
			expectedSanitized:  true,
			expectedTextOutput: true,
		},
		{
			name: "schedule_trigger_without_explicit_usage",
			workflow: `---
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Schedule Workflow

Create a report.

This workflow does NOT use text output and has no content context, so NO sanitized step.`,
			expectedSanitized:  false,
			expectedTextOutput: false,
		},
		{
			name: "issue_trigger_with_explicit_usage",
			workflow: `---
on:
  issues:
    types: [opened]
permissions:
  issues: read
strict: false
tools:
  github:
    toolsets: [issues]
---

# Test Issue Workflow With Explicit Usage

Analyze this: "${{ steps.sanitized.outputs.text }}"

This workflow explicitly uses text output AND has content context.`,
			expectedSanitized:  true,
			expectedTextOutput: true,
		},
	}

	compiler := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowPath := filepath.Join(tempDir, tt.name+".md")
			if err := os.WriteFile(workflowPath, []byte(tt.workflow), 0644); err != nil {
				t.Fatalf("Failed to write workflow: %v", err)
			}

			err := compiler.CompileWorkflow(workflowPath)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Check the compiled YAML
			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read compiled workflow: %v", err)
			}

			lockStr := string(lockContent)

			// Check for sanitized step
			hasSanitizedStep := strings.Contains(lockStr, "id: sanitized")
			if hasSanitizedStep != tt.expectedSanitized {
				t.Errorf("Expected sanitized step: %v, got: %v\nWorkflow:\n%s",
					tt.expectedSanitized, hasSanitizedStep, lockStr)
			}

			// Check for text output
			hasTextOutput := strings.Contains(lockStr, "text: ${{ steps.sanitized.outputs.text }}")
			if hasTextOutput != tt.expectedTextOutput {
				t.Errorf("Expected text output: %v, got: %v", tt.expectedTextOutput, hasTextOutput)
			}

			// Cleanup for next test
			os.RemoveAll(filepath.Join(tempDir, ".github"))
		})
	}
}
