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
  bash: ["echo", "ls"]
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
				// The error should contain the filename (relative or absolute) with :line:column:
				expectedPattern := fmt.Sprintf("%s:1:1:", tt.name+".md")
				if !strings.Contains(err.Error(), expectedPattern) {
					t.Errorf("%s: Error should contain '%s' for IDE integration, got: %s", tt.description, expectedPattern, err.Error())
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
	}

	if data.Name != "Test Workflow" {
		t.Errorf("Expected Name 'Test Workflow', got '%s'", data.Name)
	}

	if data.MarkdownContent != "# Test Content" {
		t.Errorf("Expected MarkdownContent '# Test Content', got '%s'", data.MarkdownContent)
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
			expectedOn: `"on":
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
			expectedOn: `"on":
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
			expectedOn: `"on":
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
			expectedOn:      "pull_request_review_comment:\n    types:\n    - created\n    - edited",
			expectedIf:      "contains(github.event.issue.body, '/test-bot')",
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
			expectedOn:      "pull_request_review_comment:\n    types:\n    - created\n    - edited",
			expectedIf:      "contains(github.event.issue.body, '/new-bot')",
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
			expectedOn:      "pull_request_review_comment:\n    types:\n    - created\n    - edited",
			expectedIf:      "contains(github.event.issue.body, '/default-name-bot')",
			expectedCommand: "default-name-bot",
		},
		{
			name: "string format command trigger",
			frontmatter: `---
on:
  command: "customname"
tools:
  github:
    allowed: [list_issues]
---`,
			filename:        "test-string-format.md",
			expectedOn:      "pull_request_review_comment:\n    types:\n    - created\n    - edited",
			expectedIf:      "contains(github.event.issue.body, '/customname')",
			expectedCommand: "customname",
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

			// Check that the expected if condition is present (normalize for multiline comparison)
			normalizedLockContent := NormalizeExpressionForComparison(lockContent)
			normalizedExpectedIf := NormalizeExpressionForComparison(tt.expectedIf)
			if !strings.Contains(normalizedLockContent, normalizedExpectedIf) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nExpected (normalized): %s\nActual (normalized): %s\nOriginal Content:\n%s",
					tt.expectedIf, normalizedExpectedIf, normalizedLockContent, lockContent)
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
			expectedOn:      "\"on\":\n  discussion:\n    types:\n    - created\n    - edited\n  discussion_comment:\n    types:\n    - created\n    - edited\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  workflow_dispatch: null",
			expectedIf:      "github.event_name == 'issues'",
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
			expectedOn:      "\"on\":\n  discussion:\n    types:\n    - created\n    - edited\n  discussion_comment:\n    types:\n    - created\n    - edited\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  schedule:\n  - cron: 0 9 * * 1",
			expectedIf:      "github.event_name == 'issues'",
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
			expectedOn:      "\"on\":\n  discussion:\n    types:\n    - created\n    - edited\n  discussion_comment:\n    types:\n    - created\n    - edited\n  issue_comment:\n    types:\n    - created\n    - edited\n  issues:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request:\n    types:\n    - opened\n    - edited\n    - reopened\n  pull_request_review_comment:\n    types:\n    - created\n    - edited\n  push:\n    branches:\n    - main\n  workflow_dispatch: null",
			expectedIf:      "github.event_name == 'issues'",
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

			// Check that the expected if condition is present (normalize for multiline comparison)
			normalizedLockContent := NormalizeExpressionForComparison(lockContent)
			normalizedExpectedIf := NormalizeExpressionForComparison(tt.expectedIf)
			if !strings.Contains(normalizedLockContent, normalizedExpectedIf) {
				t.Errorf("Expected lock file to contain '%s' but it didn't.\nExpected (normalized): %s\nActual (normalized): %s\nOriginal Content:\n%s",
					tt.expectedIf, normalizedExpectedIf, normalizedLockContent, lockContent)
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
				"type":    "stdio",
				"command": "custom-mcp-server",
				"args":    []any{"--option", "value"},
				"env": map[string]any{
					"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
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
				"type": "http",
				"url":  "https://example.com/api",
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
				"type":    "stdio",
				"command": "custom-mcp-server",
				"args":    []any{"--option", "value"},
				"env": map[string]any{
					"CUSTOM_TOKEN": "${CUSTOM_TOKEN}",
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
				"type":    "stdio",
				"command": "valid-server",
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
			name: "mcp config with direct fields",
			toolConfig: map[string]any{
				"type":    "stdio",
				"command": "python",
				"args":    []any{"-m", "trello_mcp"},
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
  bash: ["ls", "cat", "echo"]
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
  bash: ["grep", "find", "ls"] # ls is duplicate
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
engine: claude
tools:
  bash: ["pwd"] # Additional command in main file
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
engine: claude
tools:
  bash: ["pwd", "ls", "cat"]
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

	// Verify that 'ls' appears only once in the allowed-tools line (no duplicates in functionality)
	// We need to check specifically in the --allowed-tools line in CLI args, not in comments
	allowedToolsLinePattern := `--allowed-tools ([^\n]+)`
	re := regexp.MustCompile(allowedToolsLinePattern)
	matches := re.FindStringSubmatch(lockContent)
	if len(matches) < 2 {
		t.Errorf("Could not find --allowed-tools line in lock file")
	} else {
		allowedToolsValue := matches[1]
		bashLsCount := strings.Count(allowedToolsValue, "Bash(ls)")
		if bashLsCount != 1 {
			t.Errorf("Expected 'Bash(ls)' to appear exactly once in allowed-tools value, but found %d occurrences in: %s", bashLsCount, allowedToolsValue)
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
mcp-servers:
  notionApi:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["create_page", "search_pages"]
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
mcp-servers:
  trelloApi:
    command: "python"
    args: ["-m", "trello_mcp"]
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["create_card", "list_boards"]
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
mcp-servers:
  notionApi:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
    allowed: ["list_databases", "query_database"]  # Different allowed tools - should be merged
  customTool:
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
mcp-servers:
  mainCustomApi:
    command: "main-custom-server"
    allowed: ["main_tool1", "main_tool2"]
tools:
  github:
    allowed: ["list_issues", "create_issue"]
  edit:
---

# Test Workflow for Custom MCP Merging

{{#import %s}}

Some content here.

{{#import %s}}

More content.

{{#import %s}}

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
		// From include3 notionApi (last one wins, overrides include1)
		"notionApi(list_databases)",
		"notionApi(query_database)",
		// From include2 trelloApi
		"trelloApi(create_card)",
		"trelloApi(list_boards)",
		// From include3 customTool
		"customTool(tool1)",
		"customTool(tool2)",
		// From main file
		"mainCustomApi(main_tool1)",
		"mainCustomApi(main_tool2)",
		// Standard github MCP tools
		"github(list_issues)",
		"github(create_issue)",
	}

	// Check that all expected custom MCP tools are present
	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that the notionApi tools from both include1 and include3 are present
	// This shows that MCP servers with the same name get their 'allowed' arrays merged
	expectedAllTools := []string{
		"notionApi(create_page)",    // from include1
		"notionApi(search_pages)",   // from include1
		"notionApi(list_databases)", // from include3
		"notionApi(query_database)", // from include3
	}
	for _, expectedTool := range expectedAllTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected merged tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that Claude tools from all includes are present
	expectedClaudeTools := []string{
		"Read", "Write", // from includes
		"LS", "Task", // always present
	}
	for _, expectedTool := range expectedClaudeTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected Claude tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Verify that custom MCP configurations are properly generated in the setup
	// The configuration should use the last import for the same tool name (include3 for notionApi)
	// Check for notionApi configuration (should contain container reference from include3)
	if !strings.Contains(lockContent, `"notionApi"`) {
		t.Errorf("Expected notionApi configuration from includes not found in lock file")
	}
	// The env should be present
	if !strings.Contains(lockContent, `NOTION_TOKEN`) {
		t.Errorf("Expected notionApi env configuration not found in lock file")
	}

	// Check for trelloApi configuration (from include2)
	if !strings.Contains(lockContent, `"trelloApi"`) {
		t.Errorf("Expected trelloApi configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `TRELLO_TOKEN`) {
		t.Errorf("Expected trelloApi env configuration not found in lock file")
	}

	// Check for mainCustomApi configuration
	if !strings.Contains(lockContent, `"mainCustomApi"`) {
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
mcp-servers:
  customApi:
    command: "custom-server"
    args: ["--config", "/path/to/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
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
  edit:
---

# Test Workflow with Custom MCP Only in Include

{{#import %s}}

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
		"customApi(get_data)",
		"customApi(post_data)",
		"customApi(delete_data)",
	}

	for _, expectedTool := range expectedCustomMCPTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected custom MCP tool '%s' from include not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that custom MCP configuration is properly generated
	if !strings.Contains(lockContent, `"customApi"`) {
		t.Errorf("Expected customApi MCP server configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `"--config"`) {
		t.Errorf("Expected customApi args configuration not found in lock file")
	}
	if !strings.Contains(lockContent, `API_KEY`) {
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

func TestCustomMCPMergingFromMultipleIncludes(t *testing.T) {
	// Test that tools from imports with the same MCP server name get merged
	tmpDir, err := os.MkdirTemp("", "custom-mcp-merge-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create first include file with custom MCP server
	include1Content := `---
mcp-servers:
  apiServer:
    command: "shared-server"
    args: ["--config", "/shared/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["get_data", "post_data"]
---

# Include 1
First include file with apiServer MCP.
`
	include1File := filepath.Join(tmpDir, "include1.md")
	if err := os.WriteFile(include1File, []byte(include1Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create second include file with same MCP server but different allowed list
	include2Content := `---
mcp-servers:
  apiServer:
    command: "shared-server"
    args: ["--config", "/shared/config"]
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed: ["delete_data", "update_data"]  # Different allowed - should merge with include1
---

# Include 2
Second include file with apiServer MCP that merges with include1.
`
	include2File := filepath.Join(tmpDir, "include2.md")
	if err := os.WriteFile(include2File, []byte(include2Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create main workflow file that includes both files
	mainContent := fmt.Sprintf(`---
tools:
  github:
    allowed: ["list_issues"]
---

# Test Workflow with Merged Allowed Arrays

{{#import %s}}

{{#import %s}}

This should merge the allowed lists from both imports.
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

	// Check that tools from both imports are present (merged, not overridden)
	expectedMergedTools := []string{
		"apiServer(get_data)",
		"apiServer(post_data)",
		"apiServer(delete_data)",
		"apiServer(update_data)",
	}

	for _, expectedTool := range expectedMergedTools {
		if !strings.Contains(lockContent, expectedTool) {
			t.Errorf("Expected merged MCP tool '%s' not found in lock file.\nLock file content:\n%s", expectedTool, lockContent)
		}
	}

	// Check that the MCP server configuration is present
	if !strings.Contains(lockContent, `"apiServer"`) {
		t.Errorf("Expected apiServer MCP configuration not found in lock file")
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
				"type":    "stdio",
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
			expected: "\"on\":\n  workflow_dispatch: null",
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
	// This test verifies that workflows with nested env sections (like mcp-servers.*.env)
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

mcp-servers:
  notionApi:
    container: "mcp/notion"
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
tools:
  github:
    allowed: []
  edit:
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
	if !strings.Contains(lockContent, `NOTION_TOKEN`) {
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
		"  agent:",
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
tools:
  bash: ["echo 'hello'"]
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
		"# For more information: https://github.com/githubnext/gh-aw/blob/main/.github/instructions/github-agentic-workflows.instructions.md",
	}

	for _, line := range expectedDisclaimer {
		if !strings.Contains(lockContent, line) {
			t.Errorf("Expected disclaimer line '%s' not found in generated YAML:\n%s", line, lockContent)
		}
	}

	// Verify the disclaimer appears at the beginning of the file
	lines := strings.Split(lockContent, "\n")
	if len(lines) < 4 {
		t.Fatalf("Generated file too short, expected at least 4 lines")
	}

	// Check that the first 4 lines are comment lines (disclaimer)
	for i := 0; i < 4; i++ {
		if !strings.HasPrefix(lines[i], "#") {
			t.Errorf("Line %d should be a comment (disclaimer), but got: %s", i+1, lines[i])
		}
	}

	// Verify that the Mermaid graph is present in the header (as comments)
	if !strings.Contains(lockContent, "# Job Dependency Graph:") {
		t.Error("Expected 'Job Dependency Graph:' comment not found in generated YAML")
	}
	if !strings.Contains(lockContent, "# ```mermaid") {
		t.Error("Expected Mermaid code block not found in generated YAML")
	}
	if !strings.Contains(lockContent, "# graph LR") {
		t.Error("Expected Mermaid graph directive not found in generated YAML")
	}

	// Find the first non-comment, non-empty line - this should be the workflow name
	var firstContentLine string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			firstContentLine = trimmed
			break
		}
	}

	// Check that the first content line starts the actual workflow with name:
	if !strings.HasPrefix(firstContentLine, "name:") {
		t.Errorf("First non-comment line should start with 'name:', but got: %s", firstContentLine)
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
        uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8`,
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
			err := compiler.validateGitHubActionsSchema(tt.yaml)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateGitHubActionsSchema() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateGitHubActionsSchema() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateGitHubActionsSchema() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestBasicYAMLValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid YAML",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test:
    runs-on: ubuntu-latest`,
			wantErr: false,
		},
		{
			name: "invalid YAML syntax",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test: [invalid yaml structure`,
			wantErr: true,
			errMsg:  "sequence end token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic YAML validation (this is now always performed)
			var yamlTest any
			err := yaml.Unmarshal([]byte(tt.yaml), &yamlTest)

			if tt.wantErr {
				if err == nil {
					t.Errorf("YAML validation expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("YAML validation error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("YAML validation unexpected error = %v", err)
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

		// Should contain network hook setup (defaults to allow-list)
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup when no network field specified (defaults to allow-list)")
		}
	})

	t.Run("network: defaults should enforce allow-list restrictions", func(t *testing.T) {
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

		// Should contain network hook setup (defaults mode uses allow-list)
		if !strings.Contains(string(lockContent), "Generate Network Permissions Hook") {
			t.Error("Should contain network hook setup for network: defaults (uses allow-list)")
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
mcp-servers:
  notionApi:
    container: mcp/notion
    allowed: ["create_page", "search"]
---`,
			expectedInLock: []string{
				`"notionApi"`,
				`"mcp/notion"`,
			},
			expectError: false,
		},
		{
			name: "container with environment variables",
			frontmatter: `---
mcp-servers:
  notionApi:
    container: mcp/notion:v1.2.3
    env:
      NOTION_TOKEN: "${{ secrets.NOTION_TOKEN }}"
      API_URL: "https://api.notion.com"
    allowed: ["create_page"]
---`,
			expectedInLock: []string{
				`"mcp/notion:v1.2.3"`,
				`NOTION_TOKEN`,
				`API_URL`,
			},
			expectError: false,
		},
		{
			name: "multiple MCP servers with container fields",
			frontmatter: `---
mcp-servers:
  notionApi:
    container: mcp/notion
    allowed: ["create_page"]
  trelloApi:
    container: mcp/trello:latest
    env:
      TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
    allowed: ["list_boards"]
---`,
			expectedInLock: []string{
				`"notionApi"`,
				`"trelloApi"`,
				`"mcp/notion"`,
				`"mcp/trello:latest"`,
				`TRELLO_TOKEN`,
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
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction-specific content in generated YAML
	expectedStrings := []string{
		"GH_AW_REACTION: eyes",
		"uses: actions/github-script@v8",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify three jobs are created (check_membership, activation, main) - reaction step is now in activation job
	jobCount := strings.Count(yamlContent, "runs-on: ubuntu-latest")
	if jobCount != 3 {
		t.Errorf("Expected 3 jobs (check_membership, activation, main), found %d", jobCount)
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
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
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is empty (not defaulted)
	if workflowData.AIReaction != "" {
		t.Errorf("Expected AIReaction to be empty, got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it does NOT contain reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check that reaction-specific content is NOT in generated YAML
	unexpectedStrings := []string{
		"GH_AW_REACTION:",
		"Add eyes reaction to the triggering item",
	}

	for _, unexpected := range unexpectedStrings {
		if strings.Contains(yamlContent, unexpected) {
			t.Errorf("Generated YAML should NOT contain: %s", unexpected)
		}
	}

	// Verify three jobs are created (check_membership, activation, main) - no separate add_reaction job
	jobCount := strings.Count(yamlContent, "runs-on: ubuntu-latest")
	if jobCount != 3 {
		t.Errorf("Expected 3 jobs (check_membership, activation, main), found %d", jobCount)
	}
}

// TestAIReactionWithCommentEditFunctionality tests that the enhanced reaction script includes comment creation
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

# AI Reaction with Comment Creation Test

Test workflow with reaction and comment creation.
`

	testFile := filepath.Join(tmpDir, "test-reaction-edit.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains the enhanced reaction script
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for enhanced reaction functionality in generated YAML
	expectedStrings := []string{
		"GH_AW_REACTION: eyes",
		"uses: actions/github-script@v8",
		"addCommentWithWorkflowLink", // This should be in the new script
		"runUrl =",                   // This should be in the new script for workflow run URL
		"Comment endpoint",           // This should be logged in the new script
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify that the script includes comment creation logic for all workflows (not just command workflows)
	if !strings.Contains(yamlContent, "shouldCreateComment") {
		t.Error("Generated YAML should contain shouldCreateComment logic")
	}

	// Verify the script handles different event types appropriately
	if !strings.Contains(yamlContent, "issue_comment") {
		t.Error("Generated YAML should reference issue_comment event handling")
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
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
	workflowData, err := compiler.ParseWorkflowFile(testFile)
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
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for both environment variables in the generated YAML
	expectedEnvVars := []string{
		"GH_AW_REACTION: eyes",
		"GH_AW_COMMAND: test-bot",
	}

	for _, expected := range expectedEnvVars {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected environment variable: %s", expected)
		}
	}

	// Verify the script contains comment creation logic (now always enabled, not just for command workflows)
	if !strings.Contains(yamlContent, "shouldCreateComment = true") {
		t.Error("Generated YAML should contain comment creation logic")
	}
}

// TestCommandTriggerDefaultReaction tests that command triggers automatically enable "eyes" reaction
func TestCommandTriggerDefaultReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "command-default-reaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with command but NO explicit reaction
	testContent := `---
on:
  command:
    name: auto-bot
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
---

# Command Bot with Auto Reaction

Test command workflow that should automatically get "eyes" reaction.
`

	testFile := filepath.Join(tmpDir, "test-auto-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command is parsed correctly
	if workflowData.Command != "auto-bot" {
		t.Errorf("Expected Command to be 'auto-bot', got '%s'", workflowData.Command)
	}

	// Verify reaction is automatically set to "eyes"
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be auto-set to 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains reaction
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction environment variable in the generated YAML
	if !strings.Contains(yamlContent, "GH_AW_REACTION: eyes") {
		t.Error("Generated YAML should contain default 'eyes' reaction for command workflow")
	}

	// Check for command environment variable
	if !strings.Contains(yamlContent, "GH_AW_COMMAND: auto-bot") {
		t.Error("Generated YAML should contain command environment variable")
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
	}
}

// TestCommandTriggerCustomReaction tests that command triggers allow custom reaction override
func TestCommandTriggerCustomReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "command-custom-reaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with command and custom reaction
	testContent := `---
on:
  command:
    name: custom-bot
  reaction: rocket
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
---

# Command Bot with Custom Reaction

Test command workflow with custom reaction override.
`

	testFile := filepath.Join(tmpDir, "test-custom-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command is parsed correctly
	if workflowData.Command != "custom-bot" {
		t.Errorf("Expected Command to be 'custom-bot', got '%s'", workflowData.Command)
	}

	// Verify custom reaction overrides the default
	if workflowData.AIReaction != "rocket" {
		t.Errorf("Expected AIReaction to be 'rocket', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains custom reaction
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for custom reaction in the generated YAML
	if !strings.Contains(yamlContent, "GH_AW_REACTION: rocket") {
		t.Error("Generated YAML should contain custom 'rocket' reaction")
	}

	// Verify it doesn't contain default "eyes"
	if strings.Contains(yamlContent, "GH_AW_REACTION: eyes") {
		t.Error("Generated YAML should not contain default 'eyes' when custom reaction is specified")
	}
}

// TestInvalidReactionValue tests that invalid reaction values are rejected
func TestInvalidReactionValue(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "invalid-reaction-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test invalid reaction value
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: invalid_emoji
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [get_issue]
---

# Invalid Reaction Test

Test workflow with invalid reaction value.
`

	testFile := filepath.Join(tmpDir, "test-invalid-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow - should fail with validation error
	_, err = compiler.ParseWorkflowFile(testFile)
	if err == nil {
		t.Fatal("Expected error for invalid reaction value, but got none")
	}

	// Verify error message mentions the invalid value and valid options
	// The error can come from either schema validation or custom validation
	errMsg := err.Error()
	hasInvalidValue := strings.Contains(errMsg, "invalid_emoji") || strings.Contains(errMsg, "reaction")
	hasValidOptions := strings.Contains(errMsg, "must be one of") || strings.Contains(errMsg, "+1") || strings.Contains(errMsg, "eyes")

	if !hasInvalidValue {
		t.Errorf("Error message should mention the invalid reaction value, got: %v", err)
	}
	if !hasValidOptions {
		t.Errorf("Error message should mention valid reaction options, got: %v", err)
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
			expectedIf:   "(github.event_name != 'pull_request') || (github.event.pull_request.draft == false)",
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
			expectedIf:   "(github.event_name != 'pull_request') || (github.event.pull_request.draft == true)",
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
			expectedIf:   "(github.actor != 'dependabot[bot]') && ((github.event_name != 'pull_request') || (github.event.pull_request.draft == false))",
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
			expectedIf:   "(github.actor != 'dependabot[bot]') && ((github.event_name != 'pull_request') || (github.event.pull_request.draft == true))",
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
				// Check that the expected if condition is present (normalize for multiline comparison)
				normalizedLockContent := NormalizeExpressionForComparison(lockContent)
				normalizedExpectedIf := NormalizeExpressionForComparison(tt.expectedIf)
				if !strings.Contains(normalizedLockContent, normalizedExpectedIf) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nExpected (normalized): %s\nActual (normalized): %s\nOriginal Content:\n%s",
						tt.expectedIf, normalizedExpectedIf, normalizedLockContent, lockContent)
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
			expectedMessagePart: "Unknown property: message",
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
			// The error should contain the filename (relative or absolute) with :line:column:
			expectedPattern := fmt.Sprintf("%s.md:%d:%d:", tt.name, tt.expectedErrorLine, tt.expectedErrorColumn)
			if !strings.Contains(errorStr, expectedPattern) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, expectedPattern, errorStr)
			}

			// Verify error contains "error:" type indicator
			if !strings.Contains(errorStr, "error:") {
				t.Errorf("%s: error should contain 'error:' type indicator, got: %s", tt.description, errorStr)
			}

			// Verify error contains the expected YAML error message part
			if !strings.Contains(errorStr, tt.expectedMessagePart) {
				t.Errorf("%s: error should contain '%s', got: %s", tt.description, tt.expectedMessagePart, errorStr)
			}

			// For YAML parsing errors, verify error contains context lines
			if strings.Contains(errorStr, "frontmatter parsing failed") {
				// Verify error contains context lines (should show surrounding code)
				if !strings.Contains(errorStr, "|") {
					t.Errorf("%s: error should contain context lines with '|' markers, got: %s", tt.description, errorStr)
				}
			}
		})
	}
}

// TestCommentOutProcessedFieldsInOnSection tests the commentOutProcessedFieldsInOnSection function directly
func TestCommentOutProcessedFieldsInOnSection(t *testing.T) {
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
			result := compiler.commentOutProcessedFieldsInOnSection(tt.input)

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
				"uses: actions/cache@v4",
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
				"uses: actions/cache@v4",
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
				"uses: actions/cache@v4",
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
	aiStepIndex := strings.Index(lockContent, "- name: Execute Claude Code CLI")
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
	aiStepIndex := strings.Index(lockContent, "- name: Execute Claude Code CLI")
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
	var workflow map[string]any
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify that jobs section exists
	jobs, exists := workflow["jobs"]
	if !exists {
		t.Fatal("Jobs section not found in parsed workflow")
	}

	jobsMap, ok := jobs.(map[string]any)
	if !ok {
		t.Fatal("Jobs section is not a map")
	}

	// Find the main job (should be the one with the workflow name converted to kebab-case)
	var mainJob map[string]any
	for jobName, job := range jobsMap {
		if jobName == "agent" { // The workflow name "Test Workflow" becomes "test-workflow"
			if jobMap, ok := job.(map[string]any); ok {
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
	var workflow map[string]any
	if err := yaml.Unmarshal(lockContent, &workflow); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify that jobs section exists
	jobs, exists := workflow["jobs"]
	if !exists {
		t.Fatal("Jobs section not found in parsed workflow")
	}

	jobsMap, ok := jobs.(map[string]any)
	if !ok {
		t.Fatal("Jobs section is not a map")
	}

	// Find the main job (should be the one with the workflow name converted to kebab-case)
	var mainJob map[string]any
	for jobName, job := range jobsMap {
		if jobName == "agent" { // The workflow name "Test Workflow" becomes "test-workflow"
			if jobMap, ok := job.(map[string]any); ok {
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
	permissionsMap, ok := permissions.(map[string]any)
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
    uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
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
     uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
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
			var yamlData map[string]any
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

				var yamlData map[string]any
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

func TestAccessLogUploadConditional(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		tools       map[string]any
		mcpServers  map[string]any
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
			name: "mcp server with container but no network permissions - no access log steps",
			mcpServers: map[string]any{
				"simple": map[string]any{
					"container": "simple/tool",
					"allowed":   []any{"test"},
				},
			},
			expectSteps: false,
		},
		{
			name: "mcp server with container and network permissions - access log steps generated",
			mcpServers: map[string]any{
				"fetch": map[string]any{
					"container": "mcp/fetch",
					"network": map[string]any{
						"allowed": []any{"example.com"},
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

			// Combine tools and mcpServers for testing
			testTools := tt.tools
			if testTools == nil {
				testTools = make(map[string]any)
			}
			if tt.mcpServers != nil {
				// Add mcp servers to tools map for the test
				for name, config := range tt.mcpServers {
					testTools[name] = config
				}
			}

			// Test generateExtractAccessLogs
			compiler.generateExtractAccessLogs(&yaml, testTools)
			extractContent := yaml.String()

			// Test generateUploadAccessLogs
			yaml.Reset()
			compiler.generateUploadAccessLogs(&yaml, testTools)
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

// TestPullRequestForksArrayFilter tests the pull_request forks: []string filter functionality with glob support
func TestPullRequestForksArrayFilter(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "forks-array-filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name               string
		frontmatter        string
		expectedConditions []string // Expected substrings in the generated condition
		shouldHaveIf       bool     // Whether an if condition should be present
	}{
		{
			name: "pull_request with forks array (exact matches)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks:
      - "githubnext/test-repo"
      - "octocat/hello-world"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"github.event.pull_request.head.repo.full_name == 'githubnext/test-repo'",
				"github.event.pull_request.head.repo.full_name == 'octocat/hello-world'",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks array (glob patterns)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks:
      - "githubnext/*"
      - "octocat/*"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"startsWith(github.event.pull_request.head.repo.full_name, 'githubnext/')",
				"startsWith(github.event.pull_request.head.repo.full_name, 'octocat/')",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks array (mixed exact and glob)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks:
      - "githubnext/test-repo"
      - "octocat/*"
      - "microsoft/vscode"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"github.event.pull_request.head.repo.full_name == 'githubnext/test-repo'",
				"startsWith(github.event.pull_request.head.repo.full_name, 'octocat/')",
				"github.event.pull_request.head.repo.full_name == 'microsoft/vscode'",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with empty forks array",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks: []

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks array and existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks:
      - "trusted-org/*"

if: github.actor != 'dependabot[bot]'

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.actor != 'dependabot[bot]'",
				"startsWith(github.event.pull_request.head.repo.full_name, 'trusted-org/')",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks single string (exact match)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks: "githubnext/test-repo"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"github.event.pull_request.head.repo.full_name == 'githubnext/test-repo'",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks single string (glob pattern)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks: "githubnext/*"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{
				"github.event.pull_request.head.repo.full_name == github.repository",
				"startsWith(github.event.pull_request.head.repo.full_name, 'githubnext/')",
			},
			shouldHaveIf: true,
		},
		{
			name: "pull_request with forks wildcard string (allow all forks)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks: "*"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{},
			shouldHaveIf:       false, // No fork filtering should be applied
		},
		{
			name: "pull_request with forks array containing wildcard",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    forks:
      - "*"
      - "githubnext/test-repo"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedConditions: []string{},
			shouldHaveIf:       false, // No fork filtering should be applied due to "*"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Forks Array Filter Workflow

This is a test workflow for forks array filtering with glob support.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := testFile[:len(testFile)-3] + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContent := string(content)

			if tt.shouldHaveIf {
				// Check that each expected condition is present
				for _, expectedCondition := range tt.expectedConditions {
					if !strings.Contains(lockContent, expectedCondition) {
						t.Errorf("Expected lock file to contain '%s' but it didn't.\nContent:\n%s", expectedCondition, lockContent)
					}
				}
			} else {
				// Check that no fork-related if condition is present in the main job
				for _, condition := range tt.expectedConditions {
					if strings.Contains(lockContent, condition) {
						t.Errorf("Expected no fork filter condition but found '%s' in lock file.\nContent:\n%s", condition, lockContent)
					}
				}
			}
		})
	}
}

// TestForksArrayFieldCommentingInOnSection specifically tests that the forks array field is commented out in the on section
func TestForksArrayFieldCommentingInOnSection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "forks-array-commenting-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		frontmatter  string
		expectedYAML string // Expected YAML structure with commented forks
		description  string
	}{
		{
			name: "pull_request with forks array and types",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
    paths: ["src/**"]
    forks:
      - "org/repo"
      - "trusted/*"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedYAML: `  pull_request:
    # forks: # Fork filtering applied via job conditions
    # - org/repo # Fork filtering applied via job conditions
    # - trusted/* # Fork filtering applied via job conditions
    paths:
    - src/**
    types:
    - opened`,
			description: "Should comment out entire forks array but keep paths and types",
		},
		{
			name: "pull_request with only forks array",
			frontmatter: `---
on:
  pull_request:
    forks:
      - "specific/repo"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedYAML: `  pull_request:
    # forks: # Fork filtering applied via job conditions
    # - specific/repo # Fork filtering applied via job conditions`,
			description: "Should comment out forks array even when it's the only field",
		},
		{
			name: "pull_request with forks single string",
			frontmatter: `---
on:
  pull_request:
    forks: "specific/repo"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedYAML: `  pull_request:
    # forks: specific/repo # Fork filtering applied via job conditions`,
			description: "Should comment out forks single string",
		},
		{
			name: "pull_request with forks wildcard string",
			frontmatter: `---
on:
  pull_request:
    forks: "*"

permissions:
  contents: read
  issues: write

tools:
  github:
    allowed: [get_issue]
---`,
			expectedYAML: `  pull_request:
    # forks: "*" # Fork filtering applied via job conditions`,
			description: "Should comment out forks wildcard string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Forks Array Field Commenting Workflow

This workflow tests that forks array fields are properly commented out in the on section.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := testFile[:len(testFile)-3] + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}
			lockContent := string(content)

			// Check that the expected YAML structure is present
			if !strings.Contains(lockContent, tt.expectedYAML) {
				t.Errorf("Expected YAML structure not found in lock file.\nExpected:\n%s\nActual content:\n%s", tt.expectedYAML, lockContent)
			}

			// For test cases with forks field, ensure specific checks
			if strings.Contains(tt.frontmatter, "forks:") {
				// Check that the forks field is commented out
				if !strings.Contains(lockContent, "# forks:") {
					t.Errorf("Expected commented forks field but not found in lock file.\nContent:\n%s", lockContent)
				}

				// Check that the comment includes the explanation
				if !strings.Contains(lockContent, "# Fork filtering applied via job conditions") {
					t.Errorf("Expected forks comment to include explanation but not found in lock file.\nContent:\n%s", lockContent)
				}

				// Parse the generated YAML to ensure the forks field is not active in the parsed structure
				var workflow map[string]any
				if err := yaml.Unmarshal(content, &workflow); err != nil {
					t.Fatalf("Failed to parse generated YAML: %v", err)
				}

				if onSection, exists := workflow["on"]; exists {
					if onMap, ok := onSection.(map[string]any); ok {
						if prSection, hasPR := onMap["pull_request"]; hasPR {
							if prMap, isPRMap := prSection.(map[string]any); isPRMap {
								// The forks field should NOT be present in the parsed YAML (since it's commented)
								if _, hasForks := prMap["forks"]; hasForks {
									t.Errorf("Forks field found in parsed YAML pull_request section (should be commented): %v", prMap)
								}
							}
						}
					}
				}
			}

			// Ensure that active forks field is never present in the compiled YAML
			if strings.Contains(lockContent, "forks:") && !strings.Contains(lockContent, "# forks:") {
				t.Errorf("Active (non-commented) forks field found in compiled workflow content:\n%s", lockContent)
			}
		})
	}
}

func TestExtractSafeOutputsMaximumPatchSize(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    int
	}{
		{
			name:        "default value when safe-outputs not specified",
			frontmatter: map[string]any{},
			expected:    1024,
		},
		{
			name: "default value when max-patch-size not specified in safe-outputs",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-pull-request": nil,
				},
			},
			expected: 1024,
		},
		{
			name: "custom value as int",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      512,
					"create-pull-request": nil,
				},
			},
			expected: 512,
		},
		{
			name: "custom value as float64",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      2048.0,
					"create-pull-request": nil,
				},
			},
			expected: 2048,
		},
		{
			name: "custom value as uint64 (from YAML parsing)",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      uint64(4096),
					"create-pull-request": nil,
				},
			},
			expected: 4096,
		},
		{
			name: "invalid negative value falls back to default",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      -5,
					"create-pull-request": nil,
				},
			},
			expected: 1024,
		},
		{
			name: "invalid zero value falls back to default",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      0,
					"create-pull-request": nil,
				},
			},
			expected: 1024,
		},
		{
			name: "invalid string value falls back to default",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"max-patch-size":      "invalid",
					"create-pull-request": nil,
				},
			},
			expected: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)
			var result int
			if config != nil {
				result = config.MaximumPatchSize
			} else {
				result = 1024 // Default when no safe outputs
			}
			if result != tt.expected {
				t.Errorf("extractSafeOutputsConfig().MaximumPatchSize = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestDescriptionFieldRendering tests that the description field from frontmatter
// is correctly rendered as comments in the generated lock file
func TestDescriptionFieldRendering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "description-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                string
		frontmatter         string
		expectedDescription string
		description         string
	}{
		{
			name: "single_line_description",
			frontmatter: `---
description: "This is a simple workflow description"
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---`,
			expectedDescription: "# This is a simple workflow description",
			description:         "Should render single-line description as comment",
		},
		{
			name: "multiline_description",
			frontmatter: `---
description: |
  This is a multi-line workflow description.
  It explains what the workflow does in detail.
  Each line should be rendered as a separate comment.
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---`,
			expectedDescription: "# This is a multi-line workflow description.\n# It explains what the workflow does in detail.\n# Each line should be rendered as a separate comment.",
			description:         "Should render multi-line description with each line as comment",
		},
		{
			name: "no_description",
			frontmatter: `---
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: claude
tools:
  github:
    allowed: [list_commits]
---`,
			expectedDescription: "",
			description:         "Should not render any description comments when no description is provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

This is a test workflow to verify description field rendering.
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
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			lockContent := string(content)

			if tt.expectedDescription == "" {
				// Verify no description comments are present
				// The standard header should end and workflow content should start immediately
				lines := strings.Split(lockContent, "\n")
				inHeader := true
				for i, line := range lines {
					if inHeader && strings.HasPrefix(line, "# For more information:") {
						// This is the last line of the standard header
						// Next non-empty line should be workflow content (name: ...)
						for j := i + 1; j < len(lines); j++ {
							if strings.TrimSpace(lines[j]) != "" {
								if strings.HasPrefix(lines[j], "#") && !strings.HasPrefix(lines[j], "# ") {
									// Found a comment that's not a space-prefixed description comment
									break
								}
								if strings.HasPrefix(lines[j], "# ") {
									t.Errorf("Found unexpected description comment when none expected: %s", lines[j])
								}
								break
							}
						}
						break
					}
				}
			} else {
				// Verify description comments are present
				if !strings.Contains(lockContent, tt.expectedDescription) {
					t.Errorf("Expected description comments not found in generated YAML:\nExpected: %s\nGenerated content:\n%s", tt.expectedDescription, lockContent)
				}

				// Verify description comes after standard header and before workflow content
				headerEndPattern := "# For more information: https://github.com/githubnext/gh-aw/blob/main/.github/instructions/github-agentic-workflows.instructions.md"
				workflowStartPattern := `name: "`

				headerPos := strings.Index(lockContent, headerEndPattern)
				descPos := strings.Index(lockContent, tt.expectedDescription)
				workflowPos := strings.Index(lockContent, workflowStartPattern)

				if headerPos == -1 {
					t.Error("Standard header not found in generated YAML")
				}
				if descPos == -1 {
					t.Error("Description comments not found in generated YAML")
				}
				if workflowPos == -1 {
					t.Error("Workflow content not found in generated YAML")
				}

				if headerPos >= descPos {
					t.Error("Description should come after standard header")
				}
				if descPos >= workflowPos {
					t.Error("Description should come before workflow content")
				}
			}

			// Clean up generated lock file
			os.Remove(lockFile)
		})
	}
}

// TestOnSectionWithQuotes tests that the "on" keyword IS quoted in the generated YAML
// to prevent YAML parsers from interpreting it as a boolean value
func TestOnSectionWithQuotes(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "on-quotes-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		frontmatter string
		description string
	}{
		{
			name: "on section with reaction",
			frontmatter: `---
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [get_issue]
---`,
			description: "Test that 'on' IS quoted when reaction is present",
		},
		{
			name: "on section with stop-after",
			frontmatter: `---
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  stop-after: +1h
permissions:
  contents: read
tools:
  github:
    allowed: [list_commits]
---`,
			description: "Test that 'on' IS quoted when stop-after is present",
		},
		{
			name: "on section with both reaction and stop-after",
			frontmatter: `---
on:
  workflow_dispatch:
  issues:
    types: [opened]
  reaction: rocket
  stop-after: +30m
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [get_issue]
---`,
			description: "Test that 'on' IS quoted when both reaction and stop-after are present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Workflow

` + tt.description

			testFile := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")

			// Parse the workflow
			workflowData, err := compiler.ParseWorkflowFile(testFile)
			if err != nil {
				t.Fatalf("Failed to parse workflow: %v", err)
			}

			// Generate YAML
			yamlContent, err := compiler.generateYAML(workflowData, testFile)
			if err != nil {
				t.Fatalf("Failed to generate YAML: %v", err)
			}

			// Check that "on": IS present (quoted form)
			if !strings.Contains(yamlContent, `"on":`) {
				t.Errorf("Generated YAML does not contain quoted 'on' keyword:\n%s", yamlContent)
			}

			// Additional verification: parse the generated YAML to ensure it's valid
			var workflow map[string]any
			if err := yaml.Unmarshal([]byte(yamlContent), &workflow); err != nil {
				t.Fatalf("Failed to parse generated YAML: %v", err)
			}

			// Verify the on section exists and is valid
			if _, hasOn := workflow["on"]; !hasOn {
				t.Error("Generated workflow missing 'on' section")
			}
		})
	}
}

// extractJobSection extracts a specific job section from the YAML content
func extractJobSection(yamlContent, jobName string) string {
	lines := strings.Split(yamlContent, "\n")
	var jobLines []string
	inJob := false
	jobPrefix := "  " + jobName + ":"

	for i, line := range lines {
		if strings.HasPrefix(line, jobPrefix) {
			inJob = true
			jobLines = append(jobLines, line)
			continue
		}

		if inJob {
			// If we hit another job at the same level (starts with "  " and ends with ":"), stop
			if strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "    ") {
				break
			}
			// If we hit the end of jobs section, stop
			if strings.HasPrefix(line, "jobs:") && i > 0 {
				break
			}
			jobLines = append(jobLines, line)
		}
	}

	return strings.Join(jobLines, "\n")
}

func TestPostStepsIndentationFix(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "post-steps-indentation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with various post-steps configurations
	testContent := `---
on: push
permissions:
  contents: read
tools:
  github:
    allowed: [list_issues]
post-steps:
  - name: First Post Step
    run: echo "first"
  - name: Second Post Step
    uses: actions/upload-artifact@v4
    with:
      name: test-artifact
      path: test-file.txt
      retention-days: 7
  - name: Third Post Step
    if: success()
    run: |
      echo "multiline"
      echo "script"
engine: claude
---

# Test Post Steps Indentation

Test post-steps indentation fix.
`

	testFile := filepath.Join(tmpDir, "test-post-steps-indentation.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-post-steps-indentation.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify all post-steps are present
	if !strings.Contains(lockContent, "- name: First Post Step") {
		t.Error("Expected post-step 'First Post Step' to be in generated workflow")
	}
	if !strings.Contains(lockContent, "- name: Second Post Step") {
		t.Error("Expected post-step 'Second Post Step' to be in generated workflow")
	}
	// Note: "Third Post Step" has an 'if' condition, so it appears as "name: Third Post Step" not "- name:"
	if !strings.Contains(lockContent, "name: Third Post Step") {
		t.Error("Expected post-step 'Third Post Step' to be in generated workflow")
	}

	// Verify indentation is correct (6 spaces for list items, 8 for properties)
	lines := strings.Split(lockContent, "\n")
	for i, line := range lines {
		if strings.Contains(line, "- name: First Post Step") {
			// Check that this line has exactly 6 leading spaces
			if !strings.HasPrefix(line, "      - name: First Post Step") {
				t.Errorf("Line %d: Expected 6 spaces before '- name: First Post Step', got: %q", i+1, line)
			}
			// Check the next line (run:) has 8 spaces
			if i+1 < len(lines) {
				nextLine := lines[i+1]
				if strings.Contains(nextLine, "run:") && !strings.HasPrefix(nextLine, "        run:") {
					t.Errorf("Line %d: Expected 8 spaces before 'run:', got: %q", i+2, nextLine)
				}
			}
		}
		if strings.Contains(line, "- name: Second Post Step") {
			// Check that this line has exactly 6 leading spaces
			if !strings.HasPrefix(line, "      - name: Second Post Step") {
				t.Errorf("Line %d: Expected 6 spaces before '- name: Second Post Step', got: %q", i+1, line)
			}
			// Check subsequent lines have correct indentation
			if i+1 < len(lines) && strings.Contains(lines[i+1], "uses:") {
				if !strings.HasPrefix(lines[i+1], "        uses:") {
					t.Errorf("Line %d: Expected 8 spaces before 'uses:', got: %q", i+2, lines[i+1])
				}
			}
			if i+2 < len(lines) && strings.Contains(lines[i+2], "with:") {
				if !strings.HasPrefix(lines[i+2], "        with:") {
					t.Errorf("Line %d: Expected 8 spaces before 'with:', got: %q", i+3, lines[i+2])
				}
			}
		}
	}

	t.Log("Post-steps indentation verified successfully")
}

func TestPromptUploadArtifact(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "prompt-upload-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with basic frontmatter
	testContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
---

# Test Prompt Upload

This workflow should generate a step to upload the prompt as an artifact.
`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockYAML := string(lockContent)

	// Verify that the prompt upload step is present
	if !strings.Contains(lockYAML, "- name: Upload prompt") {
		t.Error("Expected 'Upload prompt' step to be in generated workflow")
	}

	// Verify the upload step uses the correct action
	if !strings.Contains(lockYAML, "uses: actions/upload-artifact@v4") {
		t.Error("Expected 'actions/upload-artifact@v4' action to be used")
	}

	// Verify the upload step has the correct artifact name
	if !strings.Contains(lockYAML, "name: prompt.txt") {
		t.Error("Expected artifact name to be 'prompt.txt'")
	}

	// Verify the upload step has the correct path
	if !strings.Contains(lockYAML, "path: /tmp/gh-aw/aw-prompts/prompt.txt") {
		t.Error("Expected artifact path to be '/tmp/gh-aw/aw-prompts/prompt.txt'")
	}

	// Verify the upload step has the if-no-files-found configuration
	if !strings.Contains(lockYAML, "if-no-files-found: warn") {
		t.Error("Expected 'if-no-files-found: warn' in upload step")
	}

	// Verify the upload step runs always (with if: always())
	uploadStepIndex := strings.Index(lockYAML, "- name: Upload prompt")
	if uploadStepIndex == -1 {
		t.Fatal("Upload prompt step not found")
	}

	// Check for "if: always()" in the section after the upload prompt step name
	afterUploadStep := lockYAML[uploadStepIndex:]
	nextStepIndex := strings.Index(afterUploadStep[20:], "- name:")
	if nextStepIndex == -1 {
		nextStepIndex = len(afterUploadStep) - 20
	}
	uploadStepSection := afterUploadStep[:20+nextStepIndex]

	if !strings.Contains(uploadStepSection, "if: always()") {
		t.Error("Expected 'if: always()' in upload prompt step")
	}

	t.Log("Prompt upload artifact step verified successfully")
}
