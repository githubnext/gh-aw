package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestCompileWorkflow(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with basic frontmatter
	testContent := `---
timeout_minutes: 10
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues, create_issue]
  Bash:
    allowed: ["echo", "ls"]
---

# Test Workflow

This is a test workflow for compilation.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		inputFile   string
		expectError bool
	}{
		{
			name:        "empty input file",
			inputFile:   "",
			expectError: true, // Should error with empty file
		},
		{
			name:        "nonexistent file",
			inputFile:   "/nonexistent/file.md",
			expectError: true, // Should error with nonexistent file
		},
		{
			name:        "valid workflow file",
			inputFile:   testFile,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.CompileWorkflow(tt.inputFile)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}

			// If compilation succeeded, check that lock file was created
			if !tt.expectError && err == nil {
				lockFile := strings.TrimSuffix(tt.inputFile, ".md") + ".lock.yml"
				if _, statErr := os.Stat(lockFile); os.IsNotExist(statErr) {
					t.Errorf("Expected lock file %s to be created", lockFile)
				}
			}
		})
	}
}

func TestEmptyMarkdownContentError(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "empty-markdown-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name             string
		content          string
		expectError      bool
		expectedErrorMsg string
		description      string
	}{
		{
			name: "frontmatter_only_no_content",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---`,
			expectError:      true,
			expectedErrorMsg: "no markdown content found",
			description:      "Should error when workflow has only frontmatter with no markdown content",
		},
		{
			name: "frontmatter_with_empty_lines",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---


`,
			expectError:      true,
			expectedErrorMsg: "no markdown content found",
			description:      "Should error when workflow has only frontmatter followed by empty lines",
		},
		{
			name: "frontmatter_with_whitespace_only",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---
   	   
`,
			expectError:      true,
			expectedErrorMsg: "no markdown content found",
			description:      "Should error when workflow has only frontmatter followed by whitespace (spaces and tabs)",
		},
		{
			name:             "frontmatter_with_just_newlines",
			content:          "---\non:\n  issues:\n    types: [opened]\npermissions:\n  issues: write\ntools:\n  github:\n    allowed: [add_issue_comment]\nengine: claude\n---\n\n\n\n",
			expectError:      true,
			expectedErrorMsg: "no markdown content found",
			description:      "Should error when workflow has only frontmatter followed by just newlines",
		},
		{
			name: "valid_workflow_with_content",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---

# Test Workflow

This is a valid workflow with actual markdown content.
`,
			expectError:      false,
			expectedErrorMsg: "",
			description:      "Should succeed when workflow has frontmatter and valid markdown content",
		},
		{
			name: "workflow_with_minimal_content",
			content: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
tools:
  github:
    allowed: [add_issue_comment]
engine: claude
---

Brief content`,
			expectError:      false,
			expectedErrorMsg: "",
			description:      "Should succeed when workflow has frontmatter and minimal but valid markdown content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			err := compiler.CompileWorkflow(testFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: Expected error but compilation succeeded", tt.description)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrorMsg) {
					t.Errorf("%s: Expected error containing '%s', got: %s", tt.description, tt.expectedErrorMsg, err.Error())
				}
				// Verify error contains file:line:column format for better IDE integration
				expectedPrefix := fmt.Sprintf("%s:1:1:", testFile)
				if !strings.Contains(err.Error(), expectedPrefix) {
					t.Errorf("%s: Error should contain '%s' for IDE integration, got: %s", tt.description, expectedPrefix, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("%s: Unexpected error: %v", tt.description, err)
					return
				}
				// Verify lock file was created
				lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
				if _, statErr := os.Stat(lockFile); os.IsNotExist(statErr) {
					t.Errorf("%s: Expected lock file %s to be created", tt.description, lockFile)
				}
			}
		})
	}
}

func TestWorkflowDataStructure(t *testing.T) {
	// Test the WorkflowData structure
	data := &WorkflowData{
		Name:            "Test Workflow",
		MarkdownContent: "# Test Content",
		AllowedTools:    "Bash,github",
	}

	if data.Name != "Test Workflow" {
		t.Errorf("Expected Name 'Test Workflow', got '%s'", data.Name)
	}

	if data.MarkdownContent != "# Test Content" {
		t.Errorf("Expected MarkdownContent '# Test Content', got '%s'", data.MarkdownContent)
	}

	if data.AllowedTools != "Bash,github" {
		t.Errorf("Expected AllowedTools 'Bash,github', got '%s'", data.AllowedTools)
	}
}

func TestInvalidJSONInMCPConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "invalid-json-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with invalid JSON in MCP config
	testContent := `---
on: push
tools:
  badApi:
    mcp: '{"type": "stdio", "command": "test", invalid json'
    allowed: ["*"]
---

# Test Invalid JSON MCP Configuration

This workflow tests error handling for invalid JSON in MCP configuration.
`

	testFile := filepath.Join(tmpDir, "test-invalid-json.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// This should fail with a JSON parsing error
	err = compiler.CompileWorkflow(testFile)
	if err == nil {
		t.Error("Expected error for invalid JSON in MCP configuration, got nil")
		return
	}

	// Check that the error message contains expected text
	expectedErrorSubstrings := []string{
		"invalid MCP configuration",
		"badApi",
		"invalid JSON",
	}

	errorMsg := err.Error()
	for _, expectedSubstring := range expectedErrorSubstrings {
		if !strings.Contains(errorMsg, expectedSubstring) {
			t.Errorf("Expected error message to contain '%s', but got: %s", expectedSubstring, errorMsg)
		}
	}
}

func TestComputeAllowedTools(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name     string
		tools    map[string]any
		expected string
	}{
		{
			name:     "empty tools",
			tools:    map[string]any{},
			expected: "",
		},
		{
			name: "bash with specific commands in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls"},
					},
				},
			},
			expected: "Bash(echo),Bash(ls)",
		},
		{
			name: "bash with nil value (all commands allowed)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil,
					},
				},
			},
			expected: "Bash",
		},
		{
			name: "regular tools in claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read":  nil,
						"Write": nil,
					},
				},
			},
			expected: "Read,Write",
		},
		{
			name: "mcp tools",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues", "create_issue"},
				},
			},
			expected: "mcp__github__create_issue,mcp__github__list_issues",
		},
		{
			name: "mixed claude and mcp tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"LS":   nil,
						"Read": nil,
						"Edit": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Edit,LS,Read,mcp__github__list_issues",
		},
		{
			name: "custom mcp servers with new format",
			tools: map[string]any{
				"custom_server": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"tool1", "tool2"},
				},
			},
			expected: "mcp__custom_server__tool1,mcp__custom_server__tool2",
		},
		{
			name: "mcp server with wildcard access",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"*"},
				},
			},
			expected: "mcp__notion",
		},
		{
			name: "mixed mcp servers - one with wildcard, one with specific tools",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"*"},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues", "create_issue"},
				},
			},
			expected: "mcp__github__create_issue,mcp__github__list_issues,mcp__notion",
		},
		{
			name: "bash with :* wildcard (should ignore other bash tools)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{":*"},
					},
				},
			},
			expected: "Bash",
		},
		{
			name: "bash with :* wildcard mixed with other commands (should ignore other commands)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls", ":*", "cat"},
					},
				},
			},
			expected: "Bash",
		},
		{
			name: "bash with :* wildcard and other tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{":*"},
						"Read": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Bash,Read,mcp__github__list_issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.computeAllowedTools(tt.tools, nil)

			// Since map iteration order is not guaranteed, we need to check if
			// the expected tools are present (for simple cases)
			if tt.expected == "" && result != "" {
				t.Errorf("Expected empty result, got '%s'", result)
			} else if tt.expected != "" && result == "" {
				t.Errorf("Expected non-empty result, got empty")
			} else if tt.expected == "Bash" && result != "Bash" {
				t.Errorf("Expected 'Bash', got '%s'", result)
			}
			// For more complex cases, we'd need more sophisticated comparison
		})
	}
}

func TestOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-on-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		frontmatter string
		expectedOn  string
	}{
		{
			name: "default on section",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues]
---`,
			expectedOn: "schedule:",
		},
		{
			name: "custom on workflow_dispatch",
			frontmatter: `---
on:
  workflow_dispatch:
tools:
  github:
    allowed: [list_issues]
---`,
			expectedOn: `on:
  workflow_dispatch: null`,
		},
		{
			name: "custom on with push",
			frontmatter: `---
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---`,
			expectedOn: `on:
  pull_request:
    branches:
    - main
  push:
    branches:
    - main`,
		},
		{
			name: "custom on with multiple events",
			frontmatter: `---
on:
  workflow_dispatch:
  issues:
    types: [opened, closed]  
  schedule:
    - cron: "0 8 * * *"
tools:
  github:
    allowed: [list_issues]
---`,
			expectedOn: `on:
  issues:
    types:
    - opened
    - closed
  schedule:
  - cron: 0 8 * * *
  workflow_dispatch: null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that the expected on section is present
			if !strings.Contains(lockContent, tt.expectedOn) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedOn, lockContent)
			}
		})
	}
}

func TestCommandSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-command-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name            string
		frontmatter     string
		filename        string
		expectedOn      string
		expectedIf      string
		expectedCommand string
	}{
		{
			name: "command trigger",
			frontmatter: `---
on:
  command:
    name: test-bot
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "test-bot.md",
			expectedOn:      "on:\n  issues:\n    types: [opened, edited, reopened]\n  issue_comment:\n    types: [created, edited]\n  pull_request:\n    types: [opened, edited, reopened]",
			expectedIf:      "if: ((contains(github.event.issue.body, '/test-bot')) || (contains(github.event.comment.body, '/test-bot'))) || (contains(github.event.pull_request.body, '/test-bot'))",
			expectedCommand: "test-bot",
		},
		{
			name: "new format command trigger",
			frontmatter: `---
on:
  command:
    name: new-bot
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "test-new-format.md",
			expectedOn:      "on:\n  issues:\n    types: [opened, edited, reopened]\n  issue_comment:\n    types: [created, edited]\n  pull_request:\n    types: [opened, edited, reopened]",
			expectedIf:      "if: ((contains(github.event.issue.body, '/new-bot')) || (contains(github.event.comment.body, '/new-bot'))) || (contains(github.event.pull_request.body, '/new-bot'))",
			expectedCommand: "new-bot",
		},
		{
			name: "new format command trigger no name defaults to filename",
			frontmatter: `---
on:
  command: {}
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "default-name-bot.md",
			expectedOn:      "on:\n  issues:\n    types: [opened, edited, reopened]\n  issue_comment:\n    types: [created, edited]\n  pull_request:\n    types: [opened, edited, reopened]",
			expectedIf:      "if: ((contains(github.event.issue.body, '/default-name-bot')) || (contains(github.event.comment.body, '/default-name-bot'))) || (contains(github.event.pull_request.body, '/default-name-bot'))",
			expectedCommand: "default-name-bot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Command Workflow

This is a test workflow for command triggering.
`

			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that the expected on section is present
			if !strings.Contains(lockContent, tt.expectedOn) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedOn, lockContent)
			}

			// Check that the expected if condition is present
			if !strings.Contains(lockContent, tt.expectedIf) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedIf, lockContent)
			}

			// The command is validated during compilation and should be present in the if condition
		})
	}
}

func TestCommandWithOtherEvents(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-command-merge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name             string
		frontmatter      string
		filename         string
		expectedOn       string
		expectedIf       string
		expectedCommand  string
		shouldError      bool
		expectedErrorMsg string
	}{
		{
			name: "command with workflow_dispatch",
			frontmatter: `---
on:
  command:
    name: test-bot
  workflow_dispatch:
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "command-with-dispatch.md",
			expectedOn:      "\"on\":\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  workflow_dispatch: null",
			expectedIf:      "if: ((github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment') && (((contains(github.event.issue.body, '/test-bot')) || (contains(github.event.comment.body, '/test-bot'))) || (contains(github.event.pull_request.body, '/test-bot')))) || (!(github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment'))",
			expectedCommand: "test-bot",
			shouldError:     false,
		},
		{
			name: "command with schedule",
			frontmatter: `---
on:
  command:
    name: schedule-bot
  schedule:
    - cron: "0 9 * * 1"
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "command-with-schedule.md",
			expectedOn:      "\"on\":\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  schedule:\n  - cron: 0 9 * * 1",
			expectedIf:      "if: ((github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment') && (((contains(github.event.issue.body, '/schedule-bot')) || (contains(github.event.comment.body, '/schedule-bot'))) || (contains(github.event.pull_request.body, '/schedule-bot')))) || (!(github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment'))",
			expectedCommand: "schedule-bot",
			shouldError:     false,
		},
		{
			name: "command with multiple compatible events",
			frontmatter: `---
on:
  command:
    name: multi-bot
  workflow_dispatch:
  push:
    branches: [main]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "command-with-multiple.md",
			expectedOn:      "\"on\":\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  push:\n    branches:\n    - main\n  workflow_dispatch: null",
			expectedIf:      "if: ((github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment') && (((contains(github.event.issue.body, '/multi-bot')) || (contains(github.event.comment.body, '/multi-bot'))) || (contains(github.event.pull_request.body, '/multi-bot')))) || (!(github.event_name == 'issues' || github.event_name == 'issue_comment' || github.event_name == 'pull_request' || github.event_name == 'pull_request_review_comment'))",
			expectedCommand: "multi-bot",
			shouldError:     false,
		},
		{
			name: "command with conflicting issues event - should error",
			frontmatter: `---
on:
  command:
    name: conflict-bot
  issues:
    types: [closed]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:         "command-with-issues.md",
			shouldError:      true,
			expectedErrorMsg: "cannot use 'command' with 'issues' in the same workflow",
		},
		{
			name: "command with conflicting issue_comment event - should error",
			frontmatter: `---
on:
  command:
    name: conflict-bot
  issue_comment:
    types: [deleted]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:         "command-with-issue-comment.md",
			shouldError:      true,
			expectedErrorMsg: "cannot use 'command' with 'issue_comment'",
		},
		{
			name: "command with conflicting pull_request event - should error",
			frontmatter: `---
on:
  command:
    name: conflict-bot
  pull_request:
    types: [closed]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:         "command-with-pull-request.md",
			shouldError:      true,
			expectedErrorMsg: "cannot use 'command' with 'pull_request'",
		},
		{
			name: "command with conflicting pull_request_review_comment event - should error",
			frontmatter: `---
on:
  command:
    name: conflict-bot
  pull_request_review_comment:
    types: [created]
tools:
  github:
    allowed: [list_issues]
---`,
			filename:         "command-with-pull-request-review-comment.md",
			shouldError:      true,
			expectedErrorMsg: "cannot use 'command' with 'pull_request_review_comment'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Command with Other Events Workflow

This is a test workflow for command merging with other events.
`

			testFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)

			if tt.shouldError {
				if err == nil {
					t.Fatalf("Expected error but compilation succeeded")
				}
				if !strings.Contains(err.Error(), tt.expectedErrorMsg) {
					t.Errorf("Expected error message to contain '%s' but got '%s'", tt.expectedErrorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that the expected on section is present
			if !strings.Contains(lockContent, tt.expectedOn) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedOn, lockContent)
			}

			// Check that the expected if condition is present
			if !strings.Contains(lockContent, tt.expectedIf) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedIf, lockContent)
			}

			// The alias is validated during compilation and should be correctly applied
		})
	}
}

func TestRunsOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-runs-on-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		frontmatter    string
		expectedRunsOn string
	}{
		{
			name: "default runs-on",
			frontmatter: `---
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: "runs-on: ubuntu-latest",
		},
		{
			name: "custom runs-on",
			frontmatter: `---
runs-on: windows-latest
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: "runs-on: windows-latest",
		},
		{
			name: "custom runs-on with array",
			frontmatter: `---
runs-on: [self-hosted, linux, x64]
tools:
  github:
    allowed: [list_issues]
---`,
			expectedRunsOn: `runs-on:
                - self-hosted
				- linux
				- x64`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that the expected runs-on value is present
			if !strings.Contains(lockContent, "    "+tt.expectedRunsOn) {
				// For array format, check differently
				if strings.Contains(tt.expectedRunsOn, "\n") {
					// For multiline YAML, just check that it contains the main components
					if !strings.Contains(lockContent, "runs-on:") || !strings.Contains(lockContent, "- self-hosted") {
						t.Errorf("Expected lock file to contain runs-on with array format but it didn't.\nContent:\n%s", lockContent)
					}
				} else {
					t.Errorf("Expected lock file to contain '    %s' but it didn't.\nContent:\n%s", tt.expectedRunsOn, lockContent)
				}
			}
		})
	}
}

func TestApplyDefaultGitHubMCPTools_DefaultClaudeTools(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                     string
		inputTools               map[string]any
		expectedClaudeTools      []string
		expectedTopLevelTools    []string
		shouldNotHaveClaudeTools []string
		hasGitHubTool            bool
	}{
		{
			name: "adds default claude tools when github tool present",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "adds default github and claude tools when no github tool",
			inputTools: map[string]any{
				"other": map[string]any{
					"allowed": []any{"some_action"},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"other", "github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "preserves existing claude tools when github tool present (new format)",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Task": map[string]any{
							"custom": "config",
						},
						"Read": map[string]any{
							"timeout": 30,
						},
					},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "adds only missing claude tools when some already exist (new format)",
			inputTools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Task": nil,
						"Grep": nil,
					},
				},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
		{
			name: "handles empty github tool configuration",
			inputTools: map[string]any{
				"github": map[string]any{},
			},
			expectedClaudeTools:   []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"},
			expectedTopLevelTools: []string{"github", "claude"},
			hasGitHubTool:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of input tools to avoid modifying the test data
			tools := make(map[string]any)
			for k, v := range tt.inputTools {
				tools[k] = v
			}

			result := compiler.applyDefaultGitHubMCPAndClaudeTools(tools, nil)

			// Check that all expected top-level tools are present
			for _, expectedTool := range tt.expectedTopLevelTools {
				if _, exists := result[expectedTool]; !exists {
					t.Errorf("Expected top-level tool '%s' to be present in result", expectedTool)
				}
			}

			// Check claude section if we expect claude tools
			if len(tt.expectedClaudeTools) > 0 {
				claudeSection, hasClaudeSection := result["claude"]
				if !hasClaudeSection {
					t.Error("Expected 'claude' section to exist")
					return
				}

				claudeConfig, ok := claudeSection.(map[string]any)
				if !ok {
					t.Error("Expected 'claude' section to be a map")
					return
				}

				// Check that the allowed section exists (new format)
				allowedSection, hasAllowed := claudeConfig["allowed"]
				if !hasAllowed {
					t.Error("Expected 'claude.allowed' section to exist")
					return
				}

				claudeTools, ok := allowedSection.(map[string]any)
				if !ok {
					t.Error("Expected 'claude.allowed' section to be a map")
					return
				}

				// Check that all expected Claude tools are present in the claude.allowed section
				for _, expectedTool := range tt.expectedClaudeTools {
					if _, exists := claudeTools[expectedTool]; !exists {
						t.Errorf("Expected Claude tool '%s' to be present in claude.allowed section", expectedTool)
					}
				}
			}

			// Check that tools that should not be present are indeed absent
			if len(tt.shouldNotHaveClaudeTools) > 0 {
				// Check top-level first
				for _, shouldNotHaveTool := range tt.shouldNotHaveClaudeTools {
					if _, exists := result[shouldNotHaveTool]; exists {
						t.Errorf("Expected tool '%s' to NOT be present at top level", shouldNotHaveTool)
					}
				}

				// Also check claude section doesn't exist or doesn't have these tools
				if claudeSection, hasClaudeSection := result["claude"]; hasClaudeSection {
					if claudeTools, ok := claudeSection.(map[string]any); ok {
						for _, shouldNotHaveTool := range tt.shouldNotHaveClaudeTools {
							if _, exists := claudeTools[shouldNotHaveTool]; exists {
								t.Errorf("Expected tool '%s' to NOT be present in claude section", shouldNotHaveTool)
							}
						}
					}
				}
			}

			// Verify github tool presence matches expectation
			_, hasGitHub := result["github"]
			if hasGitHub != tt.hasGitHubTool {
				t.Errorf("Expected github tool presence to be %v, got %v", tt.hasGitHubTool, hasGitHub)
			}

			// Verify that existing tool configurations are preserved
			if tt.name == "preserves existing claude tools when github tool present" {
				claudeSection := result["claude"].(map[string]any)

				if taskTool, ok := claudeSection["Task"].(map[string]any); ok {
					if custom, exists := taskTool["custom"]; !exists || custom != "config" {
						t.Errorf("Expected Task tool to preserve custom config, got %v", taskTool)
					}
				} else {
					t.Errorf("Expected Task tool to be a map[string]any with preserved config")
				}

				if readTool, ok := claudeSection["Read"].(map[string]any); ok {
					if timeout, exists := readTool["timeout"]; !exists || timeout != 30 {
						t.Errorf("Expected Read tool to preserve timeout config, got %v", readTool)
					}
				} else {
					t.Errorf("Expected Read tool to be a map[string]any with preserved config")
				}
			}
		})
	}
}

func TestDefaultClaudeToolsList(t *testing.T) {
	// Test that ensures the default Claude tools list contains the expected tools
	// This test will need to be updated if the default tools list changes
	expectedDefaultTools := []string{
		"Task",
		"Glob",
		"Grep",
		"ExitPlanMode",
		"TodoWrite",
		"LS",
		"Read",
		"NotebookRead",
	}

	compiler := NewCompiler(false, "", "test")

	// Create a minimal tools map with github tool to trigger the default Claude tools logic
	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"list_issues"},
		},
	}

	result := compiler.applyDefaultGitHubMCPAndClaudeTools(tools, nil)

	// Verify the claude section was created
	claudeSection, hasClaudeSection := result["claude"]
	if !hasClaudeSection {
		t.Error("Expected 'claude' section to be created")
		return
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude' section to be a map")
		return
	}

	// Check that the allowed section exists (new format)
	allowedSection, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Error("Expected 'claude.allowed' section to exist")
		return
	}

	claudeTools, ok := allowedSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude.allowed' section to be a map")
		return
	}

	// Verify all expected default Claude tools are added to the claude.allowed section
	for _, expectedTool := range expectedDefaultTools {
		if _, exists := claudeTools[expectedTool]; !exists {
			t.Errorf("Expected default Claude tool '%s' to be added, but it was not found", expectedTool)
		}
	}

	// Verify the count matches (github tool + claude section)
	expectedTopLevelCount := 2 // github tool + claude section
	if len(result) != expectedTopLevelCount {
		t.Errorf("Expected %d top-level tools in result (github + claude section), got %d: %v",
			expectedTopLevelCount, len(result), getToolNames(result))
	}

	// Verify the claude section has the right number of tools
	if len(claudeTools) != len(expectedDefaultTools) {
		t.Errorf("Expected %d tools in claude section, got %d: %v",
			len(expectedDefaultTools), len(claudeTools), getToolNames(claudeTools))
	}
}

func TestDefaultClaudeToolsIntegrationWithComputeAllowedTools(t *testing.T) {
	// Test that default Claude tools are properly included in the allowed tools computation
	compiler := NewCompiler(false, "", "test")

	tools := map[string]any{
		"github": map[string]any{
			"allowed": []any{"list_issues", "create_issue"},
		},
	}

	// Apply default tools first
	toolsWithDefaults := compiler.applyDefaultGitHubMCPAndClaudeTools(tools, nil)

	// Verify that the claude section was created with default tools (new format)
	claudeSection, hasClaudeSection := toolsWithDefaults["claude"]
	if !hasClaudeSection {
		t.Error("Expected 'claude' section to be created")
	}

	claudeConfig, ok := claudeSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude' section to be a map")
	}

	// Check that the allowed section exists
	allowedSection, hasAllowed := claudeConfig["allowed"]
	if !hasAllowed {
		t.Error("Expected 'claude' section to have 'allowed' subsection")
	}

	claudeTools, ok := allowedSection.(map[string]any)
	if !ok {
		t.Error("Expected 'claude.allowed' section to be a map")
	}

	// Verify default tools are present
	expectedClaudeTools := []string{"Task", "Glob", "Grep", "LS", "Read", "NotebookRead"}
	for _, expectedTool := range expectedClaudeTools {
		if _, exists := claudeTools[expectedTool]; !exists {
			t.Errorf("Expected claude.allowed section to contain '%s'", expectedTool)
		}
	}

	// Compute allowed tools
	allowedTools := compiler.computeAllowedTools(toolsWithDefaults, nil)

	// Verify that default Claude tools appear in the allowed tools string
	for _, expectedTool := range expectedClaudeTools {
		if !strings.Contains(allowedTools, expectedTool) {
			t.Errorf("Expected allowed tools to contain '%s', but got: %s", expectedTool, allowedTools)
		}
	}

	// Verify github MCP tools are also present
	if !strings.Contains(allowedTools, "mcp__github__list_issues") {
		t.Errorf("Expected allowed tools to contain 'mcp__github__list_issues', but got: %s", allowedTools)
	}
	if !strings.Contains(allowedTools, "mcp__github__create_issue") {
		t.Errorf("Expected allowed tools to contain 'mcp__github__create_issue', but got: %s", allowedTools)
	}
}

// Helper function to get tool names from a tools map for better error messages
func getToolNames(tools map[string]any) []string {
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	return names
}

func TestComputeAllowedToolsWithCustomMCP(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name     string
		tools    map[string]any
		expected []string // expected tools to be present
	}{
		{
			name: "custom mcp servers with new format",
			tools: map[string]any{
				"custom_server": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"tool1", "tool2"},
				},
				"another_server": map[string]any{
					"mcp": map[string]any{
						"type": "stdio",
					},
					"allowed": []any{"tool3"},
				},
			},
			expected: []string{"mcp__custom_server__tool1", "mcp__custom_server__tool2", "mcp__another_server__tool3"},
		},
		{
			name: "mixed tools with custom mcp",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"custom_server": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"custom_tool"},
				},
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
			},
			expected: []string{"Read", "mcp__github__list_issues", "mcp__custom_server__custom_tool"},
		},
		{
			name: "custom mcp with invalid config",
			tools: map[string]any{
				"server_no_allowed": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"command": "some-command",
				},
				"server_with_allowed": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"tool1"},
				},
			},
			expected: []string{"mcp__server_with_allowed__tool1"},
		},
		{
			name: "custom mcp with wildcard access",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"*"},
				},
			},
			expected: []string{"mcp__notion"},
		},
		{
			name: "mixed mcp servers with wildcard and specific tools",
			tools: map[string]any{
				"notion": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"*"},
				},
				"custom_server": map[string]any{
					"mcp":     map[string]any{"type": "stdio"},
					"allowed": []any{"tool1", "tool2"},
				},
			},
			expected: []string{"mcp__notion", "mcp__custom_server__tool1", "mcp__custom_server__tool2"},
		},
		{
			name: "mcp config as JSON string",
			tools: map[string]any{
				"trelloApi": map[string]any{
					"mcp":     `{"type": "stdio", "command": "python", "args": ["-m", "trello_mcp"]}`,
					"allowed": []any{"create_card", "list_boards"},
				},
			},
			expected: []string{"mcp__trelloApi__create_card", "mcp__trelloApi__list_boards"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.computeAllowedTools(tt.tools, nil)

			// Check that all expected tools are present
			for _, expectedTool := range tt.expected {
				if !strings.Contains(result, expectedTool) {
					t.Errorf("Expected tool '%s' not found in result: %s", expectedTool, result)
				}
			}
		})
	}
}

func TestGenerateCustomMCPCodexWorkflowConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name       string
		toolConfig map[string]any
		expected   []string // expected strings in output
		wantErr    bool
	}{
		{
			name: "valid stdio mcp server",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type":    "stdio",
					"command": "custom-mcp-server",
					"args":    []any{"--option", "value"},
					"env": map[string]any{
						"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
					},
				},
			},
			expected: []string{
				"[mcp_servers.custom_server]",
				"command = \"custom-mcp-server\"",
				"--option",
				"\"CUSTOM_TOKEN\" = \"${CUSTOM_TOKEN}\"",
			},
			wantErr: false,
		},
		{
			name: "server with http type should be ignored for codex",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type":    "http",
					"command": "should-be-ignored",
				},
			},
			expected: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := engine.renderCodexMCPConfig(&yaml, "custom_server", tt.toolConfig)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateCustomMCPCodexWorkflowConfigForTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := yaml.String()
				for _, expected := range tt.expected {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
					}
				}
			}
		})
	}
}

func TestGenerateCustomMCPClaudeWorkflowConfig(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name       string
		toolConfig map[string]any
		isLast     bool
		expected   []string // expected strings in output
		wantErr    bool
	}{
		{
			name: "valid stdio mcp server",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type":    "stdio",
					"command": "custom-mcp-server",
					"args":    []any{"--option", "value"},
					"env": map[string]any{
						"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
					},
				},
			},
			isLast: true,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"custom-mcp-server\"",
				"\"--option\"",
				"\"CUSTOM_TOKEN\": \"${CUSTOM_TOKEN}\"",
				"              }",
			},
			wantErr: false,
		},
		{
			name: "not last server",
			toolConfig: map[string]any{
				"mcp": map[string]any{
					"type":    "stdio",
					"command": "valid-server",
				},
			},
			isLast: false,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"valid-server\"",
				"              },", // should have comma since not last
			},
			wantErr: false,
		},
		{
			name: "mcp config as JSON string",
			toolConfig: map[string]any{
				"mcp": `{"type": "stdio", "command": "python", "args": ["-m", "trello_mcp"]}`,
			},
			isLast: true,
			expected: []string{
				"\"custom_server\": {",
				"\"command\": \"python\"",
				"\"-m\"",
				"\"trello_mcp\"",
				"              }",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := engine.renderClaudeMCPConfig(&yaml, "custom_server", tt.toolConfig, tt.isLast)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateCustomMCPCodexWorkflowConfigForTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := yaml.String()
				for _, expected := range tt.expected {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain '%s', but got: %s", expected, output)
					}
				}
			}
		})
	}
}

func TestComputeAllowedToolsWithClaudeSection(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name     string
		tools    map[string]any
		expected string
	}{
		{
			name: "claude section with tools (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Edit":      nil,
						"MultiEdit": nil,
						"Write":     nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Edit,MultiEdit,Write,mcp__github__list_issues",
		},
		{
			name: "claude section with bash tools (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls"},
						"Edit": nil,
					},
				},
			},
			expected: "Bash(echo),Bash(ls),Edit",
		},
		{
			name: "mixed top-level and claude section (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Edit":  nil,
						"Write": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expected: "Edit,Write,mcp__github__list_issues",
		},
		{
			name: "claude section with bash all commands (new format)",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil,
					},
				},
			},
			expected: "Bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.computeAllowedTools(tt.tools, nil)

			// Split both expected and result into slices and check each tool is present
			expectedTools := strings.Split(tt.expected, ",")
			if tt.expected == "" {
				expectedTools = []string{}
			}

			resultTools := strings.Split(result, ",")
			if result == "" {
				resultTools = []string{}
			}

			// Check that all expected tools are present
			for _, expected := range expectedTools {
				found := false
				for _, actual := range resultTools {
					if expected == actual {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tool '%s' not found in result: %s", expected, result)
				}
			}

			// Check that no unexpected tools are present
			for _, actual := range resultTools {
				if actual == "" {
					continue // Skip empty strings
				}
				found := false
				for _, expected := range expectedTools {
					if expected == actual {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected tool '%s' found in result: %s", actual, result)
				}
			}
		})
	}
}

func TestGenerateAllowedToolsComment(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name            string
		allowedToolsStr string
		indent          string
		expected        string
	}{
		{
			name:            "empty allowed tools",
			allowedToolsStr: "",
			indent:          "  ",
			expected:        "",
		},
		{
			name:            "single tool",
			allowedToolsStr: "Bash",
			indent:          "  ",
			expected:        "  # Allowed tools (sorted):\n  # - Bash\n",
		},
		{
			name:            "multiple tools",
			allowedToolsStr: "Bash,Edit,Read",
			indent:          "    ",
			expected:        "    # Allowed tools (sorted):\n    # - Bash\n    # - Edit\n    # - Read\n",
		},
		{
			name:            "tools with special characters",
			allowedToolsStr: "Bash(echo),mcp__github__get_issue,Write",
			indent:          "      ",
			expected:        "      # Allowed tools (sorted):\n      # - Bash(echo)\n      # - mcp__github__get_issue\n      # - Write\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.generateAllowedToolsComment(tt.allowedToolsStr, tt.indent)
			if result != tt.expected {
				t.Errorf("Expected comment:\n%q\nBut got:\n%q", tt.expected, result)
			}
		})
	}
}

func TestMergeAllowedListsFromMultipleIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "multiple-includes-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first include file with Bash tools (new format)
	include1Content := `---
tools:
  claude:
    allowed:
      Bash: ["ls", "cat", "echo"]
---

# Include 1
First include file with bash tools.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with Bash tools (new format)
	include2Content := `---
tools:
  claude:
    allowed:
      Bash: ["grep", "find", "ls"] # ls is duplicate
---

# Include 2
Second include file with bash tools.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both files (new format)
	mainContent := fmt.Sprintf(`---
tools:
  claude:
    allowed:
      Bash: ["pwd"] # Additional command in main file
---

# Test Workflow for Multiple Includes

@include %s

Some content here.

@include %s

More content.
`, filepath.Base(include1File), filepath.Base(include2File))

	// Test now with simplified structure - no includes, just main file
	// Create a simple workflow file with claude.Bash tools (no includes) (new format)
	simpleContent := `---
tools:
  claude:
    allowed:
      Bash: ["pwd", "ls", "cat"]
---

# Simple Test Workflow

This is a simple test workflow with Bash tools.
`

	simpleFile := filepath.Join(tmpDir, "simple-workflow.md")
	if err := os.WriteFile(simpleFile, []byte(simpleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the simple workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(simpleFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling simple workflow: %v", err)
	}

	// Read the generated lock file for simple workflow
	simpleLockFile := strings.TrimSuffix(simpleFile, ".md") + ".lock.yml"
	simpleContent2, err := os.ReadFile(simpleLockFile)
	if err != nil {
		t.Fatalf("Failed to read simple lock file: %v", err)
	}

	simpleLockContent := string(simpleContent2)
	// t.Logf("Simple workflow lock file content: %s", simpleLockContent)

	// Check if simple case works first
	expectedSimpleCommands := []string{"pwd", "ls", "cat"}
	for _, cmd := range expectedSimpleCommands {
		expectedTool := fmt.Sprintf("Bash(%s)", cmd)
		if !strings.Contains(simpleLockContent, expectedTool) {
			t.Errorf("Expected simple lock file to contain '%s' but it didn't.", expectedTool)
		}
	}

	// Now proceed with the original test
	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	err = compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that all bash commands from all includes are present in allowed_tools
	expectedCommands := []string{"pwd", "ls", "cat", "echo", "grep", "find"}

	// The allowed_tools should contain Bash(command) for each command
	for _, cmd := range expectedCommands {
		expectedTool := fmt.Sprintf("Bash(%s)", cmd)
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected lock file to contain '%s' but it didn't.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that 'ls' appears only once in the allowed_tools line (no duplicates in functionality)
	// We need to check specifically in the allowed_tools line, not in comments
	allowedToolsLinePattern := `allowed_tools: "([^"]+)"`
	re := regexp.MustCompile(allowedToolsLinePattern)
	matches := re.FindStringSubmatch(lockContent)
	if len(matches) < 2 {
		t.Errorf("Could not find allowed_tools line in lock file")
	} else {
		allowedToolsValue := matches[1]
		bashLsCount := strings.Count(allowedToolsValue, "Bash(ls)")
		if bashLsCount != 1 {
			t.Errorf("Expected 'Bash(ls)' to appear exactly once in allowed_tools value, but found %d occurrences in: %s", bashLsCount, allowedToolsValue)
		}
	}
}

func TestMergeCustomMCPFromMultipleIncludes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "custom-mcp-includes-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first include file with custom MCP server
	include1Content := `---
tools:
  notionApi:
    mcp:
      type: stdio
      command: docker
      args: [
        "run",
        "--rm",
        "-i",
        "-e", "NOTION_TOKEN",
        "mcp/notion"
      ]
      env:
        NOTION_TOKEN: "{{ secrets.NOTION_TOKEN }}"
    allowed: ["create_page", "search_pages"]
  claude:
    allowed:
      Read:
      Write:
---

# Include 1
First include file with custom MCP server.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with different custom MCP server
	include2Content := `---
tools:
  trelloApi:
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "trello_mcp"]
      env:
        TRELLO_TOKEN: "{{ secrets.TRELLO_TOKEN }}"
    allowed: ["create_card", "list_boards"]
  claude:
    allowed:
      Grep:
      Glob:
---

# Include 2
Second include file with different custom MCP server.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create third include file with overlapping custom MCP server (same name, compatible config)
	include3Content := `---
tools:
  notionApi:
    mcp:
      type: stdio
      command: docker  # Same command as include1
      args: [
        "run",
        "--rm",
        "-i",
        "-e", "NOTION_TOKEN",
        "mcp/notion"
      ]
      env:
        NOTION_TOKEN: "{{ secrets.NOTION_TOKEN }}"  # Same env as include1
    allowed: ["list_databases", "query_database"]  # Different allowed tools - should be merged
  customTool:
    mcp:
      type: stdio
      command: "custom-tool"
    allowed: ["tool1", "tool2"]
---

# Include 3
Third include file with compatible MCP server configuration.
`
	include3File := filepath.Join(tmpDir, "include3.md")
	if err := os.WriteFile(include3File, []byte(include3Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes all files and has its own custom MCP
	mainContent := fmt.Sprintf(`---
tools:
  mainCustomApi:
    mcp:
      type: stdio
      command: "main-custom-server"
    allowed: ["main_tool1", "main_tool2"]
  github:
    allowed: ["list_issues", "create_issue"]
  claude:
    allowed:
      LS:
      Task:
---

# Test Workflow for Custom MCP Merging

@include %s

Some content here.

@include %s

More content.

@include %s

Final content.
`, filepath.Base(include1File), filepath.Base(include2File), filepath.Base(include3File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that all custom MCP tools from all includes are present in allowed_tools
	expectedCustomMCPTools := []string{
		// From include1 notionApi (merged with include3)
		"mcp__notionApi__create_page",
		"mcp__notionApi__search_pages",
		// From include2 trelloApi
		"mcp__trelloApi__create_card",
		"mcp__trelloApi__list_boards",
		// From include3 notionApi (merged with include1)
		"mcp__notionApi__list_databases",
		"mcp__notionApi__query_database",
		// From include3 customTool
		"mcp__customTool__tool1",
		"mcp__customTool__tool2",
		// From main file
		"mcp__mainCustomApi__main_tool1",
		"mcp__mainCustomApi__main_tool2",
		// Standard github MCP tools
		"mcp__github__list_issues",
		"mcp__github__create_issue",
	}

	// Check that all expected custom MCP tools are present
	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Since tools are merged rather than overridden, both sets of tools should be present
	// This tests that the merging behavior works correctly for same-named MCP servers

	// Check that Claude tools from all includes are merged
	expectedClaudeTools := []string{
		"Read", "Write", // from include1
		"Grep", "Glob", // from include2
		"LS", "Task", // from main file
	}
	for _, expectedTool := range expectedClaudeTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected Claude tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that custom MCP configurations are properly generated in the setup
	// The configuration should merge settings from all includes for the same tool name
	// Check for notionApi configuration (should contain docker command from both includes)
	if !strings.Contains(lockContent, `"command": "docker"`) {
		t.Errorf("Expected notionApi configuration from includes (docker) not found in lock file")
	}
	// The args should be the same from both includes
	if !strings.Contains(lockContent, `"NOTION_TOKEN": "{{ secrets.NOTION_TOKEN }}"`) {
		t.Errorf("Expected notionApi env configuration not found in lock file")
	}

	// Check for trelloApi configuration (from include2)
	if !strings.Contains(lockContent, `"command": "python"`) {
		t.Errorf("Expected trelloApi configuration (python) not found in lock file")
	}
	if !strings.Contains(lockContent, `"TRELLO_TOKEN": "{{ secrets.TRELLO_TOKEN }}"`) {
		t.Errorf("Expected trelloApi env configuration not found in lock file")
	}

	// Check for mainCustomApi configuration
	if !strings.Contains(lockContent, `"command": "main-custom-server"`) {
		t.Errorf("Expected mainCustomApi configuration not found in lock file")
	}
}

func TestCustomMCPOnlyInIncludes(t *testing.T) {
	// Test case where custom MCPs are only defined in includes, not in main file
	tmpDir, err := os.MkdirTemp("", "custom-mcp-includes-only-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create include file with custom MCP server
	includeContent := `---
tools:
  customApi:
    mcp:
      type: stdio
      command: "custom-server"
      args: ["--config", "/path/to/config"]
      env:
        API_KEY: "{{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data", "delete_data"]
---

# Include with Custom MCP
Include file with custom MCP server only.
`
	includeFile := filepath.Join(tmpDir, "include.md")
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file with only standard tools
	mainContent := fmt.Sprintf(`---
tools:
  github:
    allowed: ["list_issues"]
  claude:
    allowed:
      Read:
      Write:
---

# Test Workflow with Custom MCP Only in Include

@include %s

Content using custom API from include.
`, filepath.Base(includeFile))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that custom MCP tools from include are present
	expectedCustomMCPTools := []string{
		"mcp__customApi__get_data",
		"mcp__customApi__post_data",
		"mcp__customApi__delete_data",
	}

	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' from include not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that custom MCP configuration is properly generated
	if !strings.Contains(lockContent, `"customApi": {`) {
		t.Errorf("Expected customApi MCP server configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"command": "custom-server"`) {
		t.Errorf("Expected customApi command configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"--config"`) {
		t.Errorf("Expected customApi args configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"API_KEY": "{{ secrets.API_KEY }}"`) {
		t.Errorf("Expected customApi env configuration not found in lock file")
	}
}

func TestCustomMCPMergingConflictDetection(t *testing.T) {
	// Test that conflicting MCP configurations result in errors
	tmpDir, err := os.MkdirTemp("", "custom-mcp-conflict-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first include file with custom MCP server
	include1Content := `---
tools:
  apiServer:
    mcp:
      type: stdio
    command: "server-v1"
    args: ["--port", "8080"]
    env:
      API_KEY: "{{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data"]
---

# Include 1
First include file with apiServer MCP.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with CONFLICTING custom MCP server (same name, different command)
	include2Content := `---
tools:
  apiServer:
    mcp:
      type: stdio
    command: "server-v2"  # Different command - should cause conflict
    args: ["--port", "9090"]  # Different args - should cause conflict
    env:
      API_KEY: "{{ secrets.API_KEY }}"  # Same env - should be OK
    allowed: ["delete_data", "update_data"]  # Different allowed - should be merged
---

# Include 2
Second include file with conflicting apiServer MCP.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both conflicting files
	mainContent := fmt.Sprintf(`---
tools:
  github:
    allowed: ["list_issues"]
---

# Test Workflow with Conflicting MCPs

@include %s

@include %s

This should fail due to conflicting MCP configurations.
`, filepath.Base(include1File), filepath.Base(include2File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - this should produce an error due to conflicting configurations
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(mainFile)

	// We expect this to fail due to conflicting MCP configurations
	if err == nil {
		t.Errorf("Expected compilation to fail due to conflicting MCP configurations, but it succeeded")
	} else {
		// Check that the error message mentions the conflict
		errorStr := err.Error()
		if !strings.Contains(errorStr, "conflict") && !strings.Contains(errorStr, "apiServer") {
			t.Errorf("Expected error to mention MCP conflict for 'apiServer', but got: %v", err)
		}
	}
}

func TestCustomMCPMergingAllowedArrays(t *testing.T) {
	// Test that 'allowed' arrays are properly merged when MCPs have the same name but compatible configs
	tmpDir, err := os.MkdirTemp("", "custom-mcp-merge-allowed-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first include file with custom MCP server
	include1Content := `---
tools:
  apiServer:
    mcp:
      type: stdio
      command: "shared-server"
      args: ["--config", "/shared/config"]
      env:
        API_KEY: "{{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data"]
---

# Include 1
First include file with apiServer MCP.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with COMPATIBLE custom MCP server (same config, different allowed)
	include2Content := `---
tools:
  apiServer:
    mcp:
      type: stdio
      command: "shared-server"  # Same command - should be OK
      args: ["--config", "/shared/config"]  # Same args - should be OK
      env:
        API_KEY: "{{ secrets.API_KEY }}"  # Same env - should be OK
    allowed: ["delete_data", "update_data", "get_data"]  # Different allowed with overlap - should be merged
---

# Include 2
Second include file with compatible apiServer MCP.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both compatible files
	mainContent := fmt.Sprintf(`---
tools:
  github:
    allowed: ["list_issues"]
---

# Test Workflow with Compatible MCPs

@include %s

@include %s

This should succeed and merge the allowed arrays.
`, filepath.Base(include1File), filepath.Base(include2File))

	mainFile := filepath.Join(tmpDir, "main-workflow.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow - this should succeed
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(mainFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with compatible MCPs: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(mainFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Check that all allowed tools from both includes are present (merged)
	expectedMergedTools := []string{
		"mcp__apiServer__get_data",    // from both includes
		"mcp__apiServer__post_data",   // from include1
		"mcp__apiServer__delete_data", // from include2
		"mcp__apiServer__update_data", // from include2
	}

	for _, expectedTool := range expectedMergedTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected merged MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that get_data appears only once in the allowed_tools line (no duplicates)
	// We need to check specifically in the allowed_tools line, not in comments
	allowedToolsLinePattern := `allowed_tools: "([^"]+)"`
	re := regexp.MustCompile(allowedToolsLinePattern)
	matches := re.FindStringSubmatch(lockContent)
	if len(matches) < 2 {
		t.Errorf("Could not find allowed_tools line in lock file")
	} else {
		allowedToolsValue := matches[1]
		allowedToolsMatch := strings.Count(allowedToolsValue, "mcp__apiServer__get_data")
		if allowedToolsMatch != 1 {
			t.Errorf("Expected 'mcp__apiServer__get_data' to appear exactly once in allowed_tools value, but found %d occurrences", allowedToolsMatch)
		}
	}

	// Check that the MCP server configuration is present
	if !strings.Contains(lockContent, `"apiServer": {`) {
		t.Errorf("Expected apiServer MCP configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"command": "shared-server"`) {
		t.Errorf("Expected shared apiServer command not found in lock file")
	}
}

func TestWorkflowNameWithColon(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with a header containing a colon
	testContent := `---
timeout_minutes: 10
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
---

# Playground: Everything Echo Test

This is a test workflow with a colon in the header.
`

	testFile := filepath.Join(tmpDir, "test-colon-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Test compilation
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify the workflow name is properly quoted
	lockContentStr := string(lockContent)
	if !strings.Contains(lockContentStr, `name: "Playground: Everything Echo Test"`) {
		t.Errorf("Expected quoted workflow name 'name: \"Playground: Everything Echo Test\"' not found in lock file. Content:\n%s", lockContentStr)
	}

	// Verify it doesn't contain the unquoted version which would be invalid YAML
	if strings.Contains(lockContentStr, "name: Playground: Everything Echo Test\n") {
		t.Errorf("Found unquoted workflow name which would be invalid YAML. Content:\n%s", lockContentStr)
	}
}

func TestExtractTopLevelYAMLSection_NestedEnvIssue(t *testing.T) {
	// This test verifies the fix for the nested env issue where
	// tools.mcps.*.env was being confused with top-level env
	compiler := NewCompiler(false, "", "test")

	// Create frontmatter with nested env under tools.notionApi.env
	// but NO top-level env section
	frontmatter := map[string]any{
		"on": map[string]any{
			"workflow_dispatch": nil,
		},
		"timeout_minutes": 15,
		"permissions": map[string]any{
			"contents": "read",
			"models":   "read",
		},
		"tools": map[string]any{
			"notionApi": map[string]any{
				"mcp":     map[string]any{"type": "stdio"},
				"command": "docker",
				"args": []any{
					"run",
					"--rm",
					"-i",
					"-e", "NOTION_TOKEN",
					"mcp/notion",
				},
				"env": map[string]any{
					"NOTION_TOKEN": "{{ secrets.NOTION_TOKEN }}",
				},
			},
			"github": map[string]any{
				"allowed": []any{},
			},
			"claude": map[string]any{
				"allowed": map[string]any{
					"Read":  nil,
					"Write": nil,
					"Grep":  nil,
					"Glob":  nil,
				},
			},
		},
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "top-level on section should be found",
			key:      "on",
			expected: "on:\n  workflow_dispatch: null",
		},
		{
			name:     "top-level timeout_minutes should be found",
			key:      "timeout_minutes",
			expected: "timeout_minutes: 15",
		},
		{
			name:     "top-level permissions should be found",
			key:      "permissions",
			expected: "permissions:\n  contents: read\n  models: read",
		},
		{
			name:     "nested env should NOT be found as top-level env",
			key:      "env",
			expected: "", // Should be empty since there's no top-level env
		},
		{
			name:     "top-level tools should be found",
			key:      "tools",
			expected: "tools:", // Should start with tools:
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractTopLevelYAMLSection(frontmatter, tt.key)

			if tt.expected == "" {
				if result != "" {
					t.Errorf("Expected empty result for key '%s', but got: %s", tt.key, result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result for key '%s' to contain '%s', but got: %s", tt.key, tt.expected, result)
				}
			}
		})
	}
}

func TestCompileWorkflowWithNestedEnv_NoOrphanedEnv(t *testing.T) {
	// This test verifies that workflows with nested env sections (like tools.*.env)
	// don't create orphaned env blocks in the generated YAML
	tmpDir, err := os.MkdirTemp("", "nested-env-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a workflow with nested env (similar to the original bug report)
	testContent := `---
on:
  workflow_dispatch:

timeout_minutes: 15

permissions:
  contents: read
  models: read

tools:
  notionApi:
    mcp:
      type: stdio
      command: docker
      args: [
        "run",
        "--rm",
        "-i",
        "-e", "NOTION_TOKEN",
        "mcp/notion"
      ]
      env:
        NOTION_TOKEN: "{{ secrets.NOTION_TOKEN }}"
  github:
    allowed: []
  claude:
    allowed:
      Read:
      Write:
      Grep:
      Glob:
---

# Test Workflow

This is a test workflow with nested env.
`

	testFile := filepath.Join(tmpDir, "test-nested-env.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Verify the generated YAML is valid by parsing it
	var yamlData map[string]any
	err = yaml.Unmarshal(content, &yamlData)
	if err != nil {
		t.Fatalf("Generated YAML is invalid: %v\nContent:\n%s", err, lockContent)
	}

	// Verify there's no orphaned env block at the top level
	// Look for the specific pattern that was causing the issue
	orphanedEnvPattern := `            env:
                NOTION_TOKEN:`
	if strings.Contains(lockContent, orphanedEnvPattern) {
		t.Errorf("Found orphaned env block in generated YAML:\n%s", lockContent)
	}

	// Verify the env section is properly placed in the MCP config
	if !strings.Contains(lockContent, `"NOTION_TOKEN": "{{ secrets.NOTION_TOKEN }}"`) {
		t.Errorf("Expected MCP env configuration not found in generated YAML:\n%s", lockContent)
	}

	// Verify the workflow has the expected basic structure
	expectedSections := []string{
		"name:",
		"on:",
		"  workflow_dispatch: null",
		"permissions:",
		"  contents: read",
		"  models: read",
		"jobs:",
		"  test-workflow:",
		"    runs-on: ubuntu-latest",
	}

	for _, section := range expectedSections {
		if !strings.Contains(lockContent, section) {
			t.Errorf("Expected section '%s' not found in generated YAML:\n%s", section, lockContent)
		}
	}
}

func TestGeneratedDisclaimerInLockFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a simple test workflow
	testContent := `---
name: Test Workflow
on:
  schedule:
    - cron: "0 9 * * 1"
engine: claude
claude:
  allowed:
    Bash: ["echo 'hello'"]
---

# Test Workflow

This is a test workflow.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "v1.0.0")
	err := compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Verify the disclaimer is present
	expectedDisclaimer := []string{
		"# This file was automatically generated by gh-aw. DO NOT EDIT.",
		"# To update this file, edit the corresponding .md file and run:",
		"#   gh aw compile",
	}

	for _, line := range expectedDisclaimer {
		if !strings.Contains(lockContent, line) {
			t.Errorf("Expected disclaimer line '%s' not found in generated YAML:\n%s", line, lockContent)
		}
	}

	// Verify the disclaimer appears at the beginning of the file
	lines := strings.Split(lockContent, "\n")
	if len(lines) < 3 {
		t.Fatalf("Generated file too short, expected at least 3 lines")
	}

	// Check that the first 3 lines are comment lines (disclaimer)
	for i := 0; i < 3; i++ {
		if !strings.HasPrefix(lines[i], "#") {
			t.Errorf("Line %d should be a comment (disclaimer), but got: %s", i+1, lines[i])
		}
	}

	// Check that line 4 is empty (separator after disclaimer)
	if lines[3] != "" {
		t.Errorf("Line 4 should be empty (separator), but got: %s", lines[3])
	}

	// Check that line 5 starts the actual workflow content
	if !strings.HasPrefix(lines[4], "name:") {
		t.Errorf("Line 5 should start with 'name:', but got: %s", lines[4])
	}
}

func TestValidateWorkflowSchema(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false) // Enable validation for testing

	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal workflow",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3`,
			wantErr: false,
		},
		{
			name: "invalid workflow - missing jobs",
			yaml: `name: "Test Workflow"
on: push`,
			wantErr: true,
			errMsg:  "missing property 'jobs'",
		},
		{
			name: "invalid workflow - invalid YAML",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test: [invalid yaml structure`,
			wantErr: true,
			errMsg:  "failed to parse generated YAML",
		},
		{
			name: "invalid workflow - invalid job structure",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test:
    invalid-property: value`,
			wantErr: true,
			errMsg:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.validateWorkflowSchema(tt.yaml)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateWorkflowSchema() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateWorkflowSchema() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateWorkflowSchema() unexpected error = %v", err)
				}
			}
		})
	}
}
func TestValidationCanBeSkipped(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test via CompileWorkflow - should succeed because validation is skipped by default
	tmpDir, err := os.MkdirTemp("", "validation-skip-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Workflow
on: push
---
# Test workflow`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler.customOutput = tmpDir

	// This should succeed because validation is skipped by default
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Errorf("CompileWorkflow() should succeed when validation is skipped, but got error: %v", err)
	}
}

func TestGenerateJobName(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		workflowName string
		expected     string
	}{
		{
			name:         "simple name",
			workflowName: "Test Workflow",
			expected:     "test-workflow",
		},
		{
			name:         "name with special characters",
			workflowName: "The Linter Maniac",
			expected:     "the-linter-maniac",
		},
		{
			name:         "name with colon",
			workflowName: "Playground: Everything Echo Test",
			expected:     "playground-everything-echo-test",
		},
		{
			name:         "name with parentheses",
			workflowName: "Daily Plan (Automatic)",
			expected:     "daily-plan-automatic",
		},
		{
			name:         "name with slashes",
			workflowName: "CI/CD Pipeline",
			expected:     "ci-cd-pipeline",
		},
		{
			name:         "name with quotes",
			workflowName: "Test \"Production\" System",
			expected:     "test-production-system",
		},
		{
			name:         "name with multiple spaces",
			workflowName: "Multiple   Spaces   Test",
			expected:     "multiple-spaces-test",
		},
		{
			name:         "single word",
			workflowName: "Build",
			expected:     "build",
		},
		{
			name:         "empty string",
			workflowName: "",
			expected:     "workflow-",
		},
		{
			name:         "starts with number",
			workflowName: "2024 Release",
			expected:     "workflow-2024-release",
		},
		{
			name:         "name with @ symbol",
			workflowName: "@mergefest - Merge Parent Branch Changes",
			expected:     "mergefest-merge-parent-branch-changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.generateJobName(tt.workflowName)
			if result != tt.expected {
				t.Errorf("generateJobName(%q) = %q, expected %q", tt.workflowName, result, tt.expected)
			}
		})
	}
}

func TestNetworkPermissionsDefaultBehavior(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tmpDir := t.TempDir()

	t.Run("no network field defaults to full access", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
---

# Test Workflow

This is a test workflow without network permissions.
`
		testFile := filepath.Join(tmpDir, "no-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "no-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should contain network hook setup (defaults to whitelist)
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup when no network field specified (defaults to whitelist)")
		}
	})

	t.Run("network: defaults should enforce whitelist restrictions", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
network: defaults
---

# Test Workflow

This is a test workflow with explicit defaults network permissions.
`
		testFile := filepath.Join(tmpDir, "defaults-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "defaults-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should contain network hook setup (defaults mode uses whitelist)
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup for network: defaults (uses whitelist)")
		}
	})

	t.Run("network: {} should enforce deny-all", func(t *testing.T) {
		testContent := `---
on: push
engine: claude
network: {}
---

# Test Workflow

This is a test workflow with empty network permissions (deny all).
`
		testFile := filepath.Join(tmpDir, "deny-all-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "deny-all-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should contain network hook setup (deny-all enforcement)
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup for network: {}")
		}
		// Should have empty ALLOWED_DOMAINS array for deny-all
		if !strings.Contains(string(lockContent), "ALLOWED_DOMAINS = []") {
			t.Error("Should have empty ALLOWED_DOMAINS array for deny-all policy")
		}
	})

	t.Run("network with allowed domains should enforce restrictions", func(t *testing.T) {
		testContent := `---
on: push
engine:
  id: claude
network:
  allowed: ["example.com", "api.github.com"]
---

# Test Workflow

This is a test workflow with explicit network permissions.
`
		testFile := filepath.Join(tmpDir, "allowed-domains-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "allowed-domains-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should contain network hook setup with specified domains
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup with explicit network permissions")
		}
		if !strings.Contains(string(lockContent), `"example.com"`) {
			t.Error("Should contain example.com in allowed domains")
		}
		if !strings.Contains(string(lockContent), `"api.github.com"`) {
			t.Error("Should contain api.github.com in allowed domains")
		}
	})

	t.Run("network permissions with non-claude engine should be ignored", func(t *testing.T) {
		testContent := `---
on: push
engine: codex
network:
  allowed: ["example.com"]
---

# Test Workflow

This is a test workflow with network permissions and codex engine.
`
		testFile := filepath.Join(tmpDir, "codex-network-workflow.md")
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Compile the workflow
		err := compiler.CompileWorkflow(testFile)
		if err != nil {
			t.Fatalf("Unexpected compilation error: %v", err)
		}

		// Read the compiled output
		lockFile := filepath.Join(tmpDir, "codex-network-workflow.lock.yml")
		lockContent, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}

		// Should not contain claude-specific network hook setup
		if strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should not contain network hook setup for non-claude engines")
		}
	})
}

func TestMCPImageField(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "mcp-container-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		frontmatter    string
		expectedInLock []string // Strings that should appear in the lock file
		notExpected    []string // Strings that should NOT appear in the lock file
		expectError    bool
		errorContains  string
	}{
		{
			name: "simple container field",
			frontmatter: `---
tools:
  notionApi:
    mcp:
      type: stdio
      container: mcp/notion
    allowed: ["create_page", "search"]
---`,
			expectedInLock: []string{
				`"command": "docker"`,
				`"run"`,
				`"--rm"`,
				`"-i"`,
				`"mcp/notion"`,
			},
			notExpected: []string{
				`"container"`, // container field should be removed after transformation
			},
			expectError: false,
		},
		{
			name: "container with environment variables",
			frontmatter: `---
tools:
  notionApi:
    mcp:
      type: stdio
      container: mcp/notion:v1.2.3
      env:
        NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
        API_URL: "https://api.notion.com"
    allowed: ["create_page"]
---`,
			expectedInLock: []string{
				`"command": "docker"`,
				`"-e"`,
				`"API_URL"`,
				`"-e"`,
				`"NOTION_TOKEN"`,
				`"mcp/notion:v1.2.3"`,
				`"NOTION_TOKEN": "${{ secrets.NOTION_TOKEN }}"`,
				`"API_URL": "https://api.notion.com"`,
			},
			expectError: false,
		},
		{
			name: "container with both container and command should fail",
			frontmatter: `---
tools:
  badApi:
    mcp:
      type: stdio
      container: mcp/bad
      command: docker
    allowed: ["test"]
---`,
			expectError:   true,
			errorContains: "cannot specify both 'container' and 'command'",
		},
		{
			name: "container with http type should fail",
			frontmatter: `---
tools:
  badApi:
    mcp:
      type: http
      container: mcp/bad
      url: "http://contoso.com"
    allowed: ["test"]
---`,
			expectError:   true,
			errorContains: "with type 'http' cannot use 'container' field",
		},
		{
			name: "container field as JSON string",
			frontmatter: `---
tools:
  trelloApi:
    mcp: '{"type": "stdio", "container": "trello/mcp", "env": {"TRELLO_KEY": "key123"}}'
    allowed: ["create_card"]
---`,
			expectedInLock: []string{
				`"command": "docker"`,
				`"-e"`,
				`"TRELLO_KEY"`,
				`"trello/mcp"`,
			},
			expectError: false,
		},
		{
			name: "multiple MCP servers with container fields",
			frontmatter: `---
tools:
  notionApi:
    mcp:
      type: stdio
      container: mcp/notion
    allowed: ["create_page"]
  trelloApi:
    mcp:
      type: stdio
      container: mcp/trello:latest
      env:
        TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["list_boards"]
---`,
			expectedInLock: []string{
				`"notionApi": {`,
				`"trelloApi": {`,
				`"mcp/notion"`,
				`"mcp/trello:latest"`,
				`"TRELLO_TOKEN"`,
			},
			expectError: false,
		},
	}

	compiler := NewCompiler(false, "", "test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow for container field.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that expected strings are present
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockContent, expected) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", expected, lockContent)
				}
			}

			// Check that unexpected strings are NOT present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(lockContent, notExpected) {
					t.Errorf("Lock file should NOT contain '%s' but it did.\nContent:\n%s", notExpected, lockContent)
				}
			}
		})
	}
}

func TestTransformImageToDockerCommand(t *testing.T) {
	tests := []struct {
		name      string
		mcpConfig map[string]any
		expected  map[string]any
		wantErr   bool
		errMsg    string
	}{
		{
			name: "simple container transformation",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "mcp/notion",
			},
			expected: map[string]any{
				"type":    "stdio",
				"command": "docker",
				"args":    []any{"run", "--rm", "-i", "mcp/notion"},
			},
			wantErr: false,
		},
		{
			name: "container with environment variables",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "custom/mcp:v2",
				"env": map[string]any{
					"TOKEN":   "secret",
					"API_URL": "https://api.contoso.com",
				},
			},
			expected: map[string]any{
				"type":    "stdio",
				"command": "docker",
				"args":    []any{"run", "--rm", "-i", "-e", "API_URL", "-e", "TOKEN", "custom/mcp:v2"},
				"env": map[string]any{
					"TOKEN":   "secret",
					"API_URL": "https://api.contoso.com",
				},
			},
			wantErr: false,
		},
		{
			name: "container with command conflict",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "mcp/test",
				"command":   "docker",
			},
			wantErr: true,
			errMsg:  "cannot specify both 'container' and 'command'",
		},
		{
			name: "no container field",
			mcpConfig: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "mcp_server"},
			},
			expected: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "mcp_server"},
			},
			wantErr: false,
		},
		{
			name: "invalid container type",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": 123, // Not a string
			},
			wantErr: true,
			errMsg:  "'container' must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the input to avoid modifying test data
			mcpConfig := make(map[string]any)
			for k, v := range tt.mcpConfig {
				mcpConfig[k] = v
			}

			err := transformContainerToDockerCommand(mcpConfig, "test")

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check that the transformation is correct
			if tt.expected != nil {
				// Check command
				if expCmd, hasCmd := tt.expected["command"]; hasCmd {
					if actCmd, ok := mcpConfig["command"]; !ok || actCmd != expCmd {
						t.Errorf("Expected command '%v', got '%v'", expCmd, actCmd)
					}
				}

				// Check args
				if expArgs, hasArgs := tt.expected["args"]; hasArgs {
					if actArgs, ok := mcpConfig["args"]; !ok {
						t.Errorf("Expected args %v, but args not found", expArgs)
					} else {
						// Compare args arrays
						expArgsSlice := expArgs.([]any)
						actArgsSlice, ok := actArgs.([]any)
						if !ok {
							t.Errorf("Args is not a slice")
						} else if len(expArgsSlice) != len(actArgsSlice) {
							t.Errorf("Expected %d args, got %d", len(expArgsSlice), len(actArgsSlice))
						} else {
							for i, expArg := range expArgsSlice {
								if actArgsSlice[i] != expArg {
									t.Errorf("Arg[%d]: expected '%v', got '%v'", i, expArg, actArgsSlice[i])
								}
							}
						}
					}
				}

				// Check that container field is removed
				if _, hasContainer := mcpConfig["container"]; hasContainer {
					t.Errorf("Container field should be removed after transformation")
				}

				// Check env is preserved
				if expEnv, hasEnv := tt.expected["env"]; hasEnv {
					if actEnv, ok := mcpConfig["env"]; !ok {
						t.Errorf("Expected env to be preserved")
					} else {
						expEnvMap := expEnv.(map[string]any)
						actEnvMap := actEnv.(map[string]any)
						for k, v := range expEnvMap {
							if actEnvMap[k] != v {
								t.Errorf("Env[%s]: expected '%v', got '%v'", k, v, actEnvMap[k])
							}
						}
					}
				}
			}
		})
	}
}

// TestAIReactionWorkflow tests the reaction functionality
func TestAIReactionWorkflow(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "reaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with reaction
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
timeout_minutes: 5
---

# AI Reaction Test

Test workflow with reaction.
`

	testFile := filepath.Join(tmpDir, "test-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, "test-workflow.md")
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction-specific content in generated YAML
	expectedStrings := []string{
		"add_reaction:",
		"GITHUB_AW_REACTION: eyes",
		"uses: actions/github-script@v7",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify two jobs are created (add_reaction, main) - missing_tool is not auto-created
	jobCount := strings.Count(yamlContent, "runs-on: ubuntu-latest")
	if jobCount != 2 {
		t.Errorf("Expected 2 jobs (add_reaction, main), found %d", jobCount)
	}
}

// TestAIReactionWorkflowWithoutReaction tests that workflows without explicit reaction do not create reaction actions
func TestAIReactionWorkflowWithoutReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "no-reaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file without explicit reaction (should not create reaction action)
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [get_issue]
timeout_minutes: 5
---

# No Reaction Test

Test workflow without explicit reaction (should not create reaction action).
`

	testFile := filepath.Join(tmpDir, "test-no-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is empty (not defaulted)
	if workflowData.AIReaction != "" {
		t.Errorf("Expected AIReaction to be empty, got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it does NOT contain reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, "test-workflow.md")
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check that reaction-specific content is NOT in generated YAML
	unexpectedStrings := []string{
		"add_reaction:",
		"GITHUB_AW_REACTION:",
		"Add eyes reaction to the triggering item",
	}

	for _, unexpected := range unexpectedStrings {
		if strings.Contains(yamlContent, unexpected) {
			t.Errorf("Generated YAML should NOT contain: %s", unexpected)
		}
	}

	// Verify only one job is created (main) - missing_tool is not auto-created
	jobCount := strings.Count(yamlContent, "runs-on: ubuntu-latest")
	if jobCount != 1 {
		t.Errorf("Expected 1 job (main), found %d", jobCount)
	}
}

// TestAIReactionWithCommentEditFunctionality tests that the enhanced reaction script includes comment editing
func TestAIReactionWithCommentEditFunctionality(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "reaction-edit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with reaction
	testContent := `---
on:
  issue_comment:
    types: [created]
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
---

# AI Reaction with Comment Edit Test

Test workflow with reaction and comment editing.
`

	testFile := filepath.Join(tmpDir, "test-reaction-edit.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains the enhanced reaction script
	yamlContent, err := compiler.generateYAML(workflowData, "test-workflow.md")
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for enhanced reaction functionality in generated YAML
	expectedStrings := []string{
		"add_reaction:",
		"GITHUB_AW_REACTION: eyes",
		"uses: actions/github-script@v7",
		"editCommentWithWorkflowLink", // This should be in the new script
		"runUrl =",                    // This should be in the new script for workflow run URL
		"Comment update endpoint",     // This should be logged in the new script
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify that the script includes comment editing logic but doesn't fail for non-comment events
	if !strings.Contains(yamlContent, "shouldEditComment") {
		t.Error("Generated YAML should contain shouldEditComment logic")
	}

	// Verify the script handles different event types appropriately
	if !strings.Contains(yamlContent, "issue_comment") {
		t.Error("Generated YAML should reference issue_comment event handling")
	}
}

// TestCommandReactionWithCommentEdit tests command workflows with reaction and comment editing
func TestCommandReactionWithCommentEdit(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "command-reaction-edit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with command and reaction
	testContent := `---
on:
  command:
    name: test-bot
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
---

# Command Bot with Reaction Test

Test command workflow with reaction and comment editing.
`

	testFile := filepath.Join(tmpDir, "test-command-bot.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.parseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command and reaction fields are parsed correctly
	if workflowData.Command != "test-bot" {
		t.Errorf("Expected Command to be 'test-bot', got '%s'", workflowData.Command)
	}
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains both alias and reaction environment variables
	yamlContent, err := compiler.generateYAML(workflowData, "test-workflow.md")
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for both environment variables in the generated YAML
	expectedEnvVars := []string{
		"GITHUB_AW_REACTION: eyes",
		"GITHUB_AW_COMMAND: test-bot",
	}

	for _, expected := range expectedEnvVars {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected environment variable: %s", expected)
		}
	}

	// Verify the script contains alias-aware comment editing logic
	if !strings.Contains(yamlContent, "shouldEditComment = alias") {
		t.Error("Generated YAML should contain alias-aware comment editing logic")
	}
}

// TestPullRequestDraftFilter tests the pull_request draft: false filter functionality
func TestPullRequestDraftFilter(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "draft-filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		frontmatter  string
		expectedIf   string // Expected if condition in the generated lock file
		shouldHaveIf bool   // Whether an if condition should be present
	}{
		{
			name: "pull_request with draft: false",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: false

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "if: (github.event_name != 'pull_request') || (github.event.pull_request.draft == false)",
			shouldHaveIf: true,
		},
		{
			name: "pull_request with draft: true (include only drafts)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: true

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "if: (github.event_name != 'pull_request') || (github.event.pull_request.draft == true)",
			shouldHaveIf: true,
		},
		{
			name: "pull_request without draft field (no filter)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldHaveIf: false,
		},
		{
			name: "pull_request with draft: false and existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: false

if: github.actor != 'dependabot[bot]'

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "if: (github.actor != 'dependabot[bot]') && ((github.event_name != 'pull_request') || (github.event.pull_request.draft == false))",
			shouldHaveIf: true,
		},
		{
			name: "pull_request with draft: true and existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: true

if: github.actor != 'dependabot[bot]'

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedIf:   "if: (github.actor != 'dependabot[bot]') && ((github.event_name != 'pull_request') || (github.event.pull_request.draft == true))",
			shouldHaveIf: true,
		},
		{
			name: "non-pull_request trigger (no filter applied)",
			frontmatter: `---
on:
  issues:
    types: [opened]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldHaveIf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Draft Filter Workflow

This is a test workflow for draft filtering.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			if tt.shouldHaveIf {
				// Check that the expected if condition is present
				if !strings.Contains(lockContent, tt.expectedIf) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", tt.expectedIf, lockContent)
				}
			} else {
				// Check that no draft-related if condition is present in the main job
				if strings.Contains(lockContent, "github.event.pull_request.draft == false") {
					t.Errorf("Expected no draft filter condition but found one in lock file.\nContent:\n%s", lockContent)
				}
			}
		})
	}
}

// TestDraftFieldCommentingInOnSection specifically tests that the draft field is commented out in the on section
func TestDraftFieldCommentingInOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "draft-commenting-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                 string
		frontmatter          string
		shouldContainComment bool
		shouldContainPaths   bool
		expectedDraftValue   string
		description          string
	}{
		{
			name: "pull_request with draft: false and paths",
			frontmatter: `---
on:
  pull_request:
    draft: false
    paths:
      - "go.mod"
      - "go.sum"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldContainComment: true,
			shouldContainPaths:   true,
			description:          "Draft field should be commented out while preserving paths",
		},
		{
			name: "pull_request with draft: true and types",
			frontmatter: `---
on:
  pull_request:
    draft: true
    types: [opened, edited]

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldContainComment: true,
			shouldContainPaths:   false,
			description:          "Draft field should be commented out while preserving types",
		},
		{
			name: "pull_request with only draft field",
			frontmatter: `---
on:
  pull_request:
    draft: false

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldContainComment: true,
			shouldContainPaths:   false,
			description:          "Draft field should be commented out even when it's the only field",
		},
		{
			name: "workflow_dispatch with pull_request having draft",
			frontmatter: `---
on:
  workflow_dispatch:
  pull_request:
    draft: false
    paths:
      - "*.go"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			shouldContainComment: true,
			shouldContainPaths:   true,
			description:          "Draft field should be commented out from pull_request in multi-trigger workflows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Draft Commenting Workflow

This workflow tests that draft fields are properly commented out in the on section.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			if tt.shouldContainComment {
				// Check that the draft field is commented out
				if !strings.Contains(lockContent, "# draft:") {
					t.Errorf("Expected commented draft field but not found in lock file.\nContent:\n%s", lockContent)
				}

				// Check that the comment includes the explanation
				if !strings.Contains(lockContent, "Draft filtering applied via job conditions") {
					t.Errorf("Expected draft comment to include explanation but not found in lock file.\nContent:\n%s", lockContent)
				}
			}

			// Parse the YAML to verify structure (ignoring comments)
			var workflow map[string]any
			if err := yaml.Unmarshal(content, &workflow); err != nil {
				t.Fatalf("Failed to parse generated YAML: %v", err)
			}

			// Check the on section
			onSection, hasOn := workflow["on"]
			if !hasOn {
				t.Fatal("Generated workflow missing 'on' section")
			}

			onMap, isOnMap := onSection.(map[string]any)
			if !isOnMap {
				t.Fatal("Generated workflow 'on' section is not a map")
			}

			// Check pull_request section
			prSection, hasPR := onMap["pull_request"]
			if hasPR && prSection != nil {
				if prMap, isPRMap := prSection.(map[string]any); isPRMap {
					// The draft field should NOT be present in the parsed YAML (since it's commented)
					if _, hasDraft := prMap["draft"]; hasDraft {
						t.Errorf("Draft field found in parsed YAML pull_request section (should be commented): %v", prMap)
					}

					// Check if paths are preserved when expected
					if tt.shouldContainPaths {
						if _, hasPaths := prMap["paths"]; !hasPaths {
							t.Errorf("Expected paths to be preserved but not found in pull_request section: %v", prMap)
						}
					}
				}
			}

			// Ensure that active draft field is never present in the compiled YAML
			if strings.Contains(lockContent, "draft: ") && !strings.Contains(lockContent, "# draft: ") {
				t.Errorf("Active (non-commented) draft field found in compiled workflow content:\n%s", lockContent)
			}
		})
	}
}

// TestCompileWorkflowWithInvalidYAML tests that workflows with invalid YAML syntax
// produce properly formatted error messages with file:line:column information
func TestCompileWorkflowWithInvalidYAML(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "invalid-yaml-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name                string
		content             string
		expectedErrorLine   int
		expectedErrorColumn int
		expectedMessagePart string
		description         string
	}{
		{
			name: "unclosed_bracket_in_array",
			content: `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues
engine: claude
---

# Test Workflow

Invalid YAML with unclosed bracket.`,
			expectedErrorLine:   9, // Updated to match new YAML library error reporting
			expectedErrorColumn: 1,
			expectedMessagePart: "',' or ']' must be specified",
			description:         "unclosed bracket in array should be detected",
		},
		{
			name: "invalid_mapping_context",
			content: `---
on: push
permissions:
  contents: read
  issues: write
invalid: yaml: syntax
  more: bad
engine: claude
---

# Test Workflow

Invalid YAML with bad mapping.`,
			expectedErrorLine:   6,
			expectedErrorColumn: 10, // Updated to match new YAML library error reporting
			expectedMessagePart: "mapping value is not allowed in this context",
			description:         "invalid mapping context should be detected",
		},
		{
			name: "bad_indentation",
			content: `---
on: push
permissions:
contents: read
  issues: write
engine: claude
---

# Test Workflow

Invalid YAML with bad indentation.`,
			expectedErrorLine:   4, // Updated to match new YAML library error reporting
			expectedErrorColumn: 11,
			expectedMessagePart: "mapping value is not allowed in this context", // Updated error message
			description:         "bad indentation should be detected",
		},
		{
			name: "unclosed_quote",
			content: `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: ["list_issues]
engine: claude
---

# Test Workflow

Invalid YAML with unclosed quote.`,
			expectedErrorLine:   8,
			expectedErrorColumn: 15, // Updated to match new YAML library error reporting
			expectedMessagePart: "could not find end character of double-quoted text",
			description:         "unclosed quote should be detected",
		},
		{
			name: "duplicate_keys",
			content: `---
on: push
permissions:
  contents: read
permissions:
  issues: write
engine: claude
---

# Test Workflow

Invalid YAML with duplicate keys.`,
			expectedErrorLine:   5, // Line 4 in YAML becomes line 5 in file (adjusted for frontmatter start)
			expectedErrorColumn: 1,
			expectedMessagePart: "mapping key \"permissions\" already defined",
			description:         "duplicate keys should be detected",
		},
		{
			name: "invalid_boolean_value",
			content: `---
on: push
permissions:
  contents: read
  issues: yes_please
engine: claude
---

# Test Workflow

Invalid YAML with non-boolean value for permissions.`,
			expectedErrorLine:   3,                                              // The permissions field is on line 3
			expectedErrorColumn: 13,                                             // After "permissions:"
			expectedMessagePart: "value must be one of 'read', 'write', 'none'", // Schema validation catches this
			description:         "invalid boolean values should trigger schema validation error",
		},
		{
			name: "missing_colon_in_mapping",
			content: `---
on: push
permissions
  contents: read
  issues: write
engine: claude
---

# Test Workflow

Invalid YAML with missing colon.`,
			expectedErrorLine:   3,
			expectedErrorColumn: 1,
			expectedMessagePart: "unexpected key name",
			description:         "missing colon in mapping should be detected",
		},
		{
			name: "invalid_array_syntax_missing_comma",
			content: `---
on: push
tools:
  github:
    allowed: ["list_issues" "create_issue"]
engine: claude
---

# Test Workflow

Invalid YAML with missing comma in array.`,
			expectedErrorLine:   5,
			expectedErrorColumn: 29, // Updated to match new YAML library error reporting
			expectedMessagePart: "',' or ']' must be specified",
			description:         "missing comma in array should be detected",
		},
		{
			name:                "mixed_tabs_and_spaces",
			content:             "---\non: push\npermissions:\n  contents: read\n\tissues: write\nengine: claude\n---\n\n# Test Workflow\n\nInvalid YAML with mixed tabs and spaces.",
			expectedErrorLine:   5,
			expectedErrorColumn: 1,
			expectedMessagePart: "found character '\t' that cannot start any token",
			description:         "mixed tabs and spaces should be detected",
		},
		{
			name: "invalid_number_format",
			content: `---
on: push
timeout_minutes: 05.5
permissions:
  contents: read
engine: claude
---

# Test Workflow

Invalid YAML with invalid number format.`,
			expectedErrorLine:   3,                          // The timeout_minutes field is on line 3
			expectedErrorColumn: 17,                         // After "timeout_minutes: "
			expectedMessagePart: "got number, want integer", // Schema validation catches this
			description:         "invalid number format should trigger schema validation error",
		},
		{
			name: "invalid_nested_structure",
			content: `---
on: push
tools:
  github: {
    allowed: ["list_issues"]
  }
  claude: [
permissions:
  contents: read
engine: claude
---

# Test Workflow

Invalid YAML with malformed nested structure.`,
			expectedErrorLine:   7,
			expectedErrorColumn: 11, // Updated to match new YAML library error reporting
			expectedMessagePart: "sequence end token ']' not found",
			description:         "invalid nested structure should be detected",
		},
		{
			name: "unclosed_flow_mapping",
			content: `---
on: push
permissions: {contents: read, issues: write
engine: claude
---

# Test Workflow

Invalid YAML with unclosed flow mapping.`,
			expectedErrorLine:   4,
			expectedErrorColumn: 1,
			expectedMessagePart: "',' or '}' must be specified",
			description:         "unclosed flow mapping should be detected",
		},
		{
			name: "yaml_error_with_column_information_support",
			content: `---
message: "invalid escape sequence \x in middle"
engine: claude
---

# Test Workflow

YAML error that demonstrates column position handling.`,
			expectedErrorLine:   2, // The message field is on line 2 of the frontmatter (line 3 of file)
			expectedErrorColumn: 1, // Schema validation error
			expectedMessagePart: "additional properties 'message' not allowed",
			description:         "yaml error should be extracted with column information when available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, fmt.Sprintf("%s.md", tt.name))
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Create compiler
			compiler := NewCompiler(false, "", "test")

			// Attempt compilation - should fail with proper error formatting
			err := compiler.CompileWorkflow(testFile)
			if err == nil {
				t.Errorf("%s: expected compilation to fail due to invalid YAML", tt.description)
				return
			}

			errorStr := err.Error()

			// Verify error contains file:line:column: format
			expectedPrefix := fmt.Sprintf("%s:%d:%d:", testFile, tt.expectedErrorLine, tt.expectedErrorColumn)
			if !strings.Contains(errorStr, expectedPrefix) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, expectedPrefix, errorStr)
			}

			// Verify error contains "error:" type indicator
			if !strings.Contains(errorStr, "error:") {
				t.Errorf("%s: error should contain 'error:' type indicator, got: %s", tt.description, errorStr)
			}

			// Verify error contains the expected YAML error message part
			if !strings.Contains(errorStr, tt.expectedMessagePart) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, tt.expectedMessagePart, errorStr)
			}

			// For YAML parsing errors, verify error contains hint and context lines
			if strings.Contains(errorStr, "frontmatter parsing failed") {
				// Verify error contains hint
				if !strings.Contains(errorStr, "hint: check YAML syntax in frontmatter section") {
					t.Errorf("%s: error should contain YAML syntax hint, got: %s", tt.description, errorStr)
				}

				// Verify error contains context lines (should show surrounding code)
				if !strings.Contains(errorStr, "|") {
					t.Errorf("%s: error should contain context lines with '|' markers, got: %s", tt.description, errorStr)
				}
			}
		})
	}
}

// TestCommentOutDraftInOnSection tests the commentOutDraftInOnSection function directly
func TestCommentOutDraftInOnSection(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name: "pull_request with draft and paths",
			input: `on:
    pull_request:
        draft: false
        paths:
            - go.mod
            - go.sum
    workflow_dispatch: null`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
        paths:
            - go.mod
            - go.sum
    workflow_dispatch: null`,
			description: "Should comment out draft but keep paths",
		},
		{
			name: "pull_request with draft and types",
			input: `on:
    pull_request:
        draft: true
        types:
            - opened
            - edited`,
			expected: `on:
    pull_request:
        # draft: true # Draft filtering applied via job conditions
        types:
            - opened
            - edited`,
			description: "Should comment out draft but keep types",
		},
		{
			name: "pull_request with only draft field",
			input: `on:
    pull_request:
        draft: false
    workflow_dispatch: null`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
    workflow_dispatch: null`,
			description: "Should comment out draft even when it's the only field",
		},
		{
			name: "multiple pull_request sections",
			input: `on:
    pull_request:
        draft: false
        paths:
            - "*.go"
    schedule:
        - cron: "0 9 * * 1"`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
        paths:
            - "*.go"
    schedule:
        - cron: "0 9 * * 1"`,
			description: "Should comment out draft in pull_request while leaving other sections unchanged",
		},
		{
			name: "no pull_request section",
			input: `on:
    workflow_dispatch: null
    push:
        branches:
            - main`,
			expected: `on:
    workflow_dispatch: null
    push:
        branches:
            - main`,
			description: "Should leave unchanged when no pull_request section",
		},
		{
			name: "pull_request without draft field",
			input: `on:
    pull_request:
        types:
            - opened`,
			expected: `on:
    pull_request:
        types:
            - opened`,
			description: "Should leave unchanged when no draft field in pull_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.commentOutDraftInOnSection(tt.input)

			if result != tt.expected {
				t.Errorf("%s\nExpected:\n%s\nGot:\n%s", tt.description, tt.expected, result)
			}
		})
	}
}

func TestCacheSupport(t *testing.T) {
	// Test cache support in workflow compilation
	tests := []struct {
		name              string
		frontmatter       string
		expectedInLock    []string
		notExpectedInLock []string
	}{
		{
			name: "single cache configuration",
			frontmatter: `---
name: Test Cache Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: claude
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
tools:
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"# Cache configuration from frontmatter was processed and added to the main job steps",
				"# Cache configuration from frontmatter processed below",
				"- name: Cache",
				"uses: actions/cache@v3",
				"key: node-modules-${{ hashFiles('package-lock.json') }}",
				"path: node_modules",
				"restore-keys: node-modules-",
			},
			notExpectedInLock: []string{
				"cache:",
				"cache.key:",
			},
		},
		{
			name: "multiple cache configurations",
			frontmatter: `---
name: Test Multi Cache Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: claude
cache:
  - key: node-modules-${{ hashFiles('package-lock.json') }}
    path: node_modules
    restore-keys: |
      node-modules-
  - key: build-cache-${{ github.sha }}
    path: 
      - dist
      - .cache
    restore-keys:
      - build-cache-
    fail-on-cache-miss: false
tools:
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"# Cache configuration from frontmatter was processed and added to the main job steps",
				"# Cache configuration from frontmatter processed below",
				"- name: Cache (node-modules-${{ hashFiles('package-lock.json') }})",
				"- name: Cache (build-cache-${{ github.sha }})",
				"uses: actions/cache@v3",
				"key: node-modules-${{ hashFiles('package-lock.json') }}",
				"key: build-cache-${{ github.sha }}",
				"path: node_modules",
				"path: |",
				"dist",
				".cache",
				"fail-on-cache-miss: false",
			},
			notExpectedInLock: []string{
				"cache:",
				"cache.key:",
			},
		},
		{
			name: "cache with all optional parameters",
			frontmatter: `---
name: Test Full Cache Workflow
on: workflow_dispatch
permissions:
  contents: read
engine: claude
cache:
  key: full-cache-${{ github.sha }}
  path: dist
  restore-keys:
    - cache-v1-
    - cache-
  upload-chunk-size: 32000000
  fail-on-cache-miss: true
  lookup-only: false
tools:
  github:
    allowed: [get_repository]
---`,
			expectedInLock: []string{
				"# Cache configuration from frontmatter processed below",
				"- name: Cache",
				"uses: actions/cache@v3",
				"key: full-cache-${{ github.sha }}",
				"path: dist",
				"restore-keys: |",
				"cache-v1-",
				"cache-",
				"upload-chunk-size: 32000000",
				"fail-on-cache-miss: true",
				"lookup-only: false",
			},
			notExpectedInLock: []string{
				"cache:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tmpDir := t.TempDir()

			// Create test workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			testContent := tt.frontmatter + "\n\n# Test Cache Workflow\n\nThis is a test workflow.\n"
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "v1.0.0")
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that expected strings are present
			for _, expected := range tt.expectedInLock {
				if !strings.Contains(lockContent, expected) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", expected, lockContent)
				}
			}

			// Check that unexpected strings are NOT present
			for _, notExpected := range tt.notExpectedInLock {
				if strings.Contains(lockContent, notExpected) {
					t.Errorf("Lock file should NOT contain '%s' but it did.\nContent:\n%s", notExpected, lockContent)
				}
			}
		})
	}
}

func TestPostStepsGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "post-steps-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with both steps and post-steps
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
steps:
  - name: Pre AI Step
    run: echo "This runs before AI"
post-steps:
  - name: Post AI Step
    run: echo "This runs after AI"
  - name: Another Post Step
    uses: actions/upload-artifact@v4
    with:
      name: test-artifact
      path: test-file.txt
engine: claude
---

# Test Post Steps Workflow

This workflow tests the post-steps functionality.
`

	testFile := filepath.Join(tmpDir, "test-post-steps.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with post-steps: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-post-steps.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify pre-steps appear before AI execution
	if !strings.Contains(lockContent, "- name: Pre AI Step") {
		t.Error("Expected pre-step 'Pre AI Step' to be in generated workflow")
	}

	// Verify post-steps appear after AI execution
	if !strings.Contains(lockContent, "- name: Post AI Step") {
		t.Error("Expected post-step 'Post AI Step' to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Another Post Step") {
		t.Error("Expected post-step 'Another Post Step' to be in generated workflow")
	}

	// Verify the order: pre-steps should come before AI execution, post-steps after
	preStepIndex := strings.Index(lockContent, "- name: Pre AI Step")
	aiStepIndex := strings.Index(lockContent, "- name: Execute Claude Code Action")
	postStepIndex := strings.Index(lockContent, "- name: Post AI Step")

	if preStepIndex == -1 || aiStepIndex == -1 || postStepIndex == -1 {
		t.Fatal("Could not find expected steps in generated workflow")
	}

	if preStepIndex >= aiStepIndex {
		t.Error("Pre-step should appear before AI execution step")
	}

	if postStepIndex <= aiStepIndex {
		t.Error("Post-step should appear after AI execution step")
	}

	t.Logf("Step order verified: Pre-step (%d) < AI execution (%d) < Post-step (%d)",
		preStepIndex, aiStepIndex, postStepIndex)
}

func TestPostStepsOnly(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "post-steps-only-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with only post-steps (no pre-steps)
	testContent := `---
on: issues
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
post-steps:
  - name: Only Post Step
    run: echo "This runs after AI only"
engine: claude
---

# Test Post Steps Only Workflow

This workflow tests post-steps without pre-steps.
`

	testFile := filepath.Join(tmpDir, "test-post-steps-only.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with post-steps only: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-post-steps-only.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify post-step appears after AI execution
	if !strings.Contains(lockContent, "- name: Only Post Step") {
		t.Error("Expected post-step 'Only Post Step' to be in generated workflow")
	}

	// Verify default checkout step is used (since no custom steps defined)
	if !strings.Contains(lockContent, "- name: Checkout repository") {
		t.Error("Expected default checkout step when no custom steps defined")
	}

	// Verify the order: AI execution should come before post-steps
	aiStepIndex := strings.Index(lockContent, "- name: Execute Claude Code Action")
	postStepIndex := strings.Index(lockContent, "- name: Only Post Step")

	if aiStepIndex == -1 || postStepIndex == -1 {
		t.Fatal("Could not find expected steps in generated workflow")
	}

	if postStepIndex <= aiStepIndex {
		t.Error("Post-step should appear after AI execution step")
	}
}

func TestDefaultPermissions(t *testing.T) {
	// Test that workflows without permissions in frontmatter get default permissions applied
	tmpDir, err := os.MkdirTemp("", "default-permissions-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITHOUT permissions specified in frontmatter
	testContent := `---
on:
  issues:
    types: [opened]
tools:
  github:
    allowed: [list_issues]
engine: claude
---

# Test Workflow

This workflow should get default permissions applied automatically.
`

	testFile := filepath.Join(tmpDir, "test-default-permissions.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Calculate the lock file path
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

	// Read the generated lock file
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify that default permissions are present in the generated workflow
	expectedDefaultPermissions := []string{
		"read-all",
	}

	for _, expectedPerm := range expectedDefaultPermissions {
		if !strings.Contains(lockContentStr, expectedPerm) {
			t.Errorf("Expected default permission '%s' not found in generated workflow.\nGenerated content:\n%s", expectedPerm, lockContentStr)
		}
	}

	// Verify that permissions section exists
	if !strings.Contains(lockContentStr, "permissions:") {
		t.Error("Expected 'permissions:' section not found in generated workflow")
	}

	// Parse the generated YAML to verify structure
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify that jobs section exists
	jobs, exists := workflow["jobs"]
	if !exists {
		t.Fatal("Jobs section not found in parsed workflow")
	}

	jobsMap, ok := jobs.(map[string]interface{})
	if !ok {
		t.Fatal("Jobs section is not a map")
	}

	// Find the main job (should be the one with the workflow name converted to kebab-case)
	var mainJob map[string]interface{}
	for jobName, job := range jobsMap {
		if jobName == "test-workflow" { // The workflow name "Test Workflow" becomes "test-workflow"
			if jobMap, ok := job.(map[string]interface{}); ok {
				mainJob = jobMap
				break
			}
		}
	}

	if mainJob == nil {
		t.Fatal("Main workflow job not found")
	}

	// Verify permissions section exists in the main job
	permissions, exists := mainJob["permissions"]
	if !exists {
		t.Fatal("Permissions section not found in main job")
	}

	// Verify permissions is a map
	permissionsValue, ok := permissions.(string)
	if !ok {
		t.Fatal("Permissions section is not a string")
	}
	if permissionsValue != "read-all" {
		t.Fatal("Default permissions not read-all")
	}
}

func TestCustomPermissionsOverrideDefaults(t *testing.T) {
	// Test that custom permissions in frontmatter override default permissions
	tmpDir, err := os.MkdirTemp("", "custom-permissions-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test workflow WITH custom permissions specified in frontmatter
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: write
  issues: write
tools:
  github:
    allowed: [list_issues, create_issue]
engine: claude
---

# Test Workflow

This workflow has custom permissions that should override defaults.
`

	testFile := filepath.Join(tmpDir, "test-custom-permissions.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Calculate the lock file path
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

	// Read the generated lock file
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Parse the generated YAML to verify structure
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify that jobs section exists
	jobs, exists := workflow["jobs"]
	if !exists {
		t.Fatal("Jobs section not found in parsed workflow")
	}

	jobsMap, ok := jobs.(map[string]interface{})
	if !ok {
		t.Fatal("Jobs section is not a map")
	}

	// Find the main job (should be the one with the workflow name converted to kebab-case)
	var mainJob map[string]interface{}
	for jobName, job := range jobsMap {
		if jobName == "test-workflow" { // The workflow name "Test Workflow" becomes "test-workflow"
			if jobMap, ok := job.(map[string]interface{}); ok {
				mainJob = jobMap
				break
			}
		}
	}

	if mainJob == nil {
		t.Fatal("Main workflow job not found")
	}

	// Verify permissions section exists in the main job
	permissions, exists := mainJob["permissions"]
	if !exists {
		t.Fatal("Permissions section not found in main job")
	}

	// Verify permissions is a map
	permissionsMap, ok := permissions.(map[string]interface{})
	if !ok {
		t.Fatal("Permissions section is not a map")
	}

	// Verify custom permissions are applied
	expectedCustomPermissions := map[string]string{
		"contents": "write",
		"issues":   "write",
	}

	for key, expectedValue := range expectedCustomPermissions {
		actualValue, exists := permissionsMap[key]
		if !exists {
			t.Errorf("Expected custom permission '%s' not found in permissions map", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Expected permission '%s' to have value '%s', but got '%v'", key, expectedValue, actualValue)
		}
	}

	// Verify that default permissions that are not overridden are NOT present
	// since custom permissions completely replace defaults
	lockContentStr := string(lockContent)
	defaultOnlyPermissions := []string{
		"pull-requests: read",
		"discussions: read",
		"deployments: read",
		"actions: read",
		"checks: read",
		"statuses: read",
	}

	for _, defaultPerm := range defaultOnlyPermissions {
		if strings.Contains(lockContentStr, defaultPerm) {
			t.Errorf("Default permission '%s' should not be present when custom permissions are specified.\nGenerated content:\n%s", defaultPerm, lockContentStr)
		}
	}
}

func TestCustomStepsIndentation(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "steps-indentation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		stepsYAML   string
		description string
	}{
		{
			name: "standard_2_space_indentation",
			stepsYAML: `steps:
  - name: Checkout code
    uses: actions/checkout@v5
  - name: Set up Go
    uses: actions/setup-go@v5
    with:
      go-version-file: go.mod
      cache: true`,
			description: "Standard 2-space indentation should be preserved with 6-space base offset",
		},
		{
			name: "odd_3_space_indentation",
			stepsYAML: `steps:
   - name: Odd indent
     uses: actions/checkout@v5
     with:
       param: value`,
			description: "3-space indentation should be normalized to standard format",
		},
		{
			name: "deep_nesting",
			stepsYAML: `steps:
  - name: Deep nesting
    uses: actions/complex@v1
    with:
      config:
        database:
          host: localhost
          settings:
            timeout: 30`,
			description: "Deep nesting should maintain relative indentation with 6-space base offset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test workflow with the given steps YAML
			testContent := fmt.Sprintf(`---
on: push
permissions:
  contents: read
%s
engine: claude
---

# Test Steps Indentation

%s
`, tt.stepsYAML, tt.description)

			testFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s.md", tt.name))
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			compiler := NewCompiler(false, "", "test")

			// Compile the workflow
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s.lock.yml", tt.name))
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			lockContent := string(content)

			// Verify the YAML is valid by parsing it
			var yamlData map[string]interface{}
			if err := yaml.Unmarshal(content, &yamlData); err != nil {
				t.Errorf("Generated YAML is not valid: %v\nContent:\n%s", err, lockContent)
			}

			// Check that custom steps are present and properly indented
			if !strings.Contains(lockContent, "      - name:") {
				t.Errorf("Expected to find properly indented step items (6 spaces) in generated content")
			}

			// Verify step properties have proper indentation (8+ spaces for uses, with, etc.)
			lines := strings.Split(lockContent, "\n")
			foundCustomSteps := false
			for i, line := range lines {
				// Look for custom step content (not generated workflow infrastructure)
				if strings.Contains(line, "Checkout code") || strings.Contains(line, "Set up Go") ||
					strings.Contains(line, "Odd indent") || strings.Contains(line, "Deep nesting") {
					foundCustomSteps = true
				}

				// Check indentation for lines containing step properties within custom steps section
				if foundCustomSteps && (strings.Contains(line, "uses: actions/") || strings.Contains(line, "with:")) {
					if !strings.HasPrefix(line, "        ") {
						t.Errorf("Step property at line %d should have 8+ spaces indentation: '%s'", i+1, line)
					}
				}
			}

			if !foundCustomSteps {
				t.Error("Expected to find custom steps content in generated workflow")
			}
		})
	}
}

func TestStopAfterCompiledAway(t *testing.T) {
	// Test that stop-after is properly compiled away and doesn't appear in final YAML
	tmpDir, err := os.MkdirTemp("", "stop-after-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name             string
		frontmatter      string
		shouldNotContain []string // Strings that should NOT appear in the lock file
		shouldContain    []string // Strings that should appear in the lock file
		description      string
	}{
		{
			name: "stop-after with workflow_dispatch",
			frontmatter: `---
on:
  workflow_dispatch:
  schedule:
    - cron: "0 2 * * 1-5"
  stop-after: "+48h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +48h",
				"stop-after: \"+48h\"",
			},
			shouldContain: []string{
				"workflow_dispatch: null",
				"- cron: 0 2 * * 1-5",
			},
			description: "stop-after should be compiled away when used with workflow_dispatch and schedule",
		},
		{
			name: "stop-after with command trigger",
			frontmatter: `---
on:
  command:
    name: test-bot
  workflow_dispatch:
  stop-after: "2024-12-31T23:59:59Z"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: 2024-12-31T23:59:59Z",
				"stop-after: \"2024-12-31T23:59:59Z\"",
			},
			shouldContain: []string{
				"workflow_dispatch: null",
				"issue_comment:",
				"issues:",
				"pull_request:",
			},
			description: "stop-after should be compiled away when used with alias triggers",
		},
		{
			name: "stop-after with reaction",
			frontmatter: `---
on:
  issues:
    types: [opened]
  reaction: eyes
  stop-after: "+24h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +24h",
				"stop-after: \"+24h\"",
			},
			shouldContain: []string{
				"issues:",
				"types:",
				"- opened",
			},
			description: "stop-after should be compiled away when used with reaction",
		},
		{
			name: "stop-after only with schedule",
			frontmatter: `---
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+72h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +72h",
				"stop-after: \"+72h\"",
			},
			shouldContain: []string{
				"schedule:",
				"- cron: 0 9 * * 1",
			},
			description: "stop-after should be compiled away when used only with schedule",
		},
		{
			name: "stop-after with both command and reaction",
			frontmatter: `---
on:
  command:
    name: test-bot
  reaction: heart
  workflow_dispatch:
  stop-after: "+36h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +36h",
				"stop-after: \"+36h\"",
			},
			shouldContain: []string{
				"workflow_dispatch: null",
				"issue_comment:",
				"issues:",
				"pull_request:",
			},
			description: "stop-after should be compiled away when used with both alias and reaction",
		},
		{
			name: "stop-after with reaction and schedule",
			frontmatter: `---
on:
  issues:
    types: [opened, edited]
  reaction: rocket
  schedule:
    - cron: "0 8 * * *"
  stop-after: "+12h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +12h",
				"stop-after: \"+12h\"",
			},
			shouldContain: []string{
				"issues:",
				"types:",
				"- opened",
				"- edited",
				"schedule:",
				"- cron: 0 8 * * *",
			},
			description: "stop-after should be compiled away when used with reaction and schedule",
		},
		{
			name: "stop-after with command and schedule",
			frontmatter: `---
on:
  command:
    name: scheduler-bot
  schedule:
    - cron: "0 12 * * *"
  workflow_dispatch:
  stop-after: "+96h"
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			shouldNotContain: []string{
				"stop-after:",
				"stop-after: +96h",
				"stop-after: \"+96h\"",
			},
			shouldContain: []string{
				"workflow_dispatch: null",
				"schedule:",
				"- cron: 0 12 * * *",
				"issue_comment:",
				"issues:",
				"pull_request:",
			},
			description: "stop-after should be compiled away when used with alias and schedule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Stop-After Compilation

This workflow tests that stop-after is properly compiled away.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check that strings that should NOT appear are indeed absent
			for _, shouldNotContain := range tt.shouldNotContain {
				if strings.Contains(lockContent, shouldNotContain) {
					t.Errorf("%s: Lock file should NOT contain '%s' but it did.\nLock file content:\n%s", tt.description, shouldNotContain, lockContent)
				}
			}

			// Check that expected strings are present
			for _, shouldContain := range tt.shouldContain {
				if !strings.Contains(lockContent, shouldContain) {
					t.Errorf("%s: Expected lock file to contain '%s' but it didn't.\nLock file content:\n%s", tt.description, shouldContain, lockContent)
				}
			}

			// Verify the lock file is valid YAML
			var yamlData map[string]any
			if err := yaml.Unmarshal(content, &yamlData); err != nil {
				t.Errorf("%s: Generated YAML is invalid: %v\nContent:\n%s", tt.description, err, lockContent)
			}
		})
	}
}

func TestCustomStepsEdgeCases(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "steps-edge-cases-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		stepsYAML   string
		expectError bool
		description string
	}{
		{
			name:        "no_custom_steps",
			stepsYAML:   `# No steps section defined`,
			expectError: false,
			description: "Should use default checkout step when no custom steps defined",
		},
		{
			name:        "empty_steps",
			stepsYAML:   `steps: []`,
			expectError: false,
			description: "Empty steps array should be handled gracefully",
		},
		{
			name:        "steps_with_only_whitespace",
			stepsYAML:   `# No steps defined`,
			expectError: false,
			description: "No steps section should use default steps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := fmt.Sprintf(`---
on: push
permissions:
  contents: read
%s
engine: claude
---

# Test Edge Cases

%s
`, tt.stepsYAML, tt.description)

			testFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s.md", tt.name))
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			compiler := NewCompiler(false, "", "test")
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}

			if !tt.expectError {
				// Verify lock file was created and is valid YAML
				lockFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s.lock.yml", tt.name))
				content, err := os.ReadFile(lockFile)
				if err != nil {
					t.Fatalf("Failed to read generated lock file: %v", err)
				}

				var yamlData map[string]interface{}
				if err := yaml.Unmarshal(content, &yamlData); err != nil {
					t.Errorf("Generated YAML is not valid: %v", err)
				}

				// For no custom steps, should contain default checkout
				if tt.name == "no_custom_steps" {
					lockContent := string(content)
					if !strings.Contains(lockContent, "- name: Checkout repository") {
						t.Error("Expected default checkout step when no custom steps defined")
					}
				}
			}
		})
	}
}

func TestComputeAllowedToolsWithSafeOutputs(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		tools       map[string]any
		safeOutputs *SafeOutputsConfig
		expected    string
	}{
		{
			name: "SafeOutputs with no tools - should add Write permission",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write",
		},
		{
			name: "SafeOutputs with general Write permission - should not add specific Write",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read":  nil,
						"Write": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write",
		},
		{
			name: "No SafeOutputs - should not add Write permission",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
			},
			safeOutputs: nil,
			expected:    "Read",
		},
		{
			name: "SafeOutputs with multiple output types",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil,
					},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:       &CreateIssuesConfig{Max: 1},
				AddIssueComments:   &AddIssueCommentsConfig{Max: 1},
				CreatePullRequests: &CreatePullRequestsConfig{Max: 1},
			},
			expected: "Bash,Write",
		},
		{
			name: "SafeOutputs with MCP tools",
			tools: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Read": nil,
					},
				},
				"github": map[string]any{
					"allowed": []any{"create_issue", "create_pull_request"},
				},
			},
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{Max: 1},
			},
			expected: "Read,Write,mcp__github__create_issue,mcp__github__create_pull_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.computeAllowedTools(tt.tools, tt.safeOutputs)

			// Split both expected and result into slices and check each tool is present
			expectedTools := strings.Split(tt.expected, ",")
			resultTools := strings.Split(result, ",")

			// Check that all expected tools are present
			for _, expectedTool := range expectedTools {
				if expectedTool == "" {
					continue // Skip empty strings
				}
				found := false
				for _, actualTool := range resultTools {
					if actualTool == expectedTool {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tool '%s' not found in result '%s'", expectedTool, result)
				}
			}

			// Check that no unexpected tools are present
			for _, actual := range resultTools {
				if actual == "" {
					continue // Skip empty strings
				}
				found := false
				for _, expected := range expectedTools {
					if expected == actual {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected tool '%s' found in result '%s'", actual, result)
				}
			}
		})
	}
}

func TestAccessLogUploadConditional(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		tools       map[string]any
		expectSteps bool
	}{
		{
			name: "no tools - no access log steps",
			tools: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
			},
			expectSteps: false,
		},
		{
			name: "tool with container but no network permissions - no access log steps",
			tools: map[string]any{
				"simple": map[string]any{
					"mcp": map[string]any{
						"type":      "stdio",
						"container": "simple/tool",
					},
					"allowed": []any{"test"},
				},
			},
			expectSteps: false,
		},
		{
			name: "tool with container and network permissions - access log steps generated",
			tools: map[string]any{
				"fetch": map[string]any{
					"mcp": map[string]any{
						"type":      "stdio",
						"container": "mcp/fetch",
					},
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []any{"example.com"},
						},
					},
					"allowed": []any{"fetch"},
				},
			},
			expectSteps: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			// Test generateExtractAccessLogs
			compiler.generateExtractAccessLogs(&yaml, tt.tools)
			extractContent := yaml.String()

			// Test generateUploadAccessLogs
			yaml.Reset()
			compiler.generateUploadAccessLogs(&yaml, tt.tools)
			uploadContent := yaml.String()

			hasExtractStep := strings.Contains(extractContent, "name: Extract squid access logs")
			hasUploadStep := strings.Contains(uploadContent, "name: Upload squid access logs")

			if tt.expectSteps {
				if !hasExtractStep {
					t.Errorf("Expected extract step to be generated but it wasn't")
				}
				if !hasUploadStep {
					t.Errorf("Expected upload step to be generated but it wasn't")
				}
			} else {
				if hasExtractStep {
					t.Errorf("Expected no extract step but one was generated")
				}
				if hasUploadStep {
					t.Errorf("Expected no upload step but one was generated")
				}
			}
		})
	}
}
