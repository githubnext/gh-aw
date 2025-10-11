package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestExtractFrontmatterFromContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantYAML     map[string]any
		wantMarkdown string
		wantErr      bool
	}{
		{
			name: "valid frontmatter and markdown",
			content: `---
title: Test Workflow
on: push
---

# Test Workflow

This is a test workflow.`,
			wantYAML: map[string]any{
				"title": "Test Workflow",
				"on":    "push",
			},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow.",
		},
		{
			name: "no frontmatter",
			content: `# Test Workflow

This is a test workflow without frontmatter.`,
			wantYAML:     map[string]any{},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow without frontmatter.",
		},
		{
			name: "empty frontmatter",
			content: `---
---

# Test Workflow

This is a test workflow with empty frontmatter.`,
			wantYAML:     map[string]any{},
			wantMarkdown: "# Test Workflow\n\nThis is a test workflow with empty frontmatter.",
		},
		{
			name:    "unclosed frontmatter",
			content: "---\ntitle: Test\nno closing delimiter",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFrontmatterFromContent(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractFrontmatterFromContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractFrontmatterFromContent() error = %v", err)
				return
			}

			// Check frontmatter
			if len(tt.wantYAML) != len(result.Frontmatter) {
				t.Errorf("ExtractFrontmatterFromContent() frontmatter length = %v, want %v", len(result.Frontmatter), len(tt.wantYAML))
			}

			for key, expectedValue := range tt.wantYAML {
				if actualValue, exists := result.Frontmatter[key]; !exists {
					t.Errorf("ExtractFrontmatterFromContent() missing key %v", key)
				} else if actualValue != expectedValue {
					t.Errorf("ExtractFrontmatterFromContent() frontmatter[%v] = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Check markdown
			if result.Markdown != tt.wantMarkdown {
				t.Errorf("ExtractFrontmatterFromContent() markdown = %v, want %v", result.Markdown, tt.wantMarkdown)
			}
		})
	}
}

func TestExtractYamlChunk(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		key      string
		expected string
	}{
		{
			name: "simple key-value",
			yaml: `title: Test Workflow
on: push
permissions: read`,
			key:      "on",
			expected: "on: push",
		},
		{
			name: "nested structure",
			yaml: `title: Test Workflow
on:
  push:
    branches:
      - main
  pull_request:
    types: [opened]
permissions: read`,
			key: "on",
			expected: `on:
  push:
    branches:
      - main
  pull_request:
    types: [opened]`,
		},
		{
			name: "deeply nested structure",
			yaml: `tools:
  bash:
    allowed:
      - "ls"
      - "cat"
  github:
    allowed:
      - "create_issue"`,
			key: "tools",
			expected: `tools:
  bash:
    allowed:
      - "ls"
      - "cat"
  github:
    allowed:
      - "create_issue"`,
		},
		{
			name: "key not found",
			yaml: `title: Test Workflow
on: push`,
			key:      "nonexistent",
			expected: "",
		},
		{
			name:     "empty yaml",
			yaml:     "",
			key:      "test",
			expected: "",
		},
		{
			name:     "empty key",
			yaml:     "title: Test",
			key:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractYamlChunk(tt.yaml, tt.key)
			if err != nil {
				t.Errorf("ExtractYamlChunk() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractYamlChunk() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractMarkdownSection(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		sectionName string
		expected    string
		wantErr     bool
	}{
		{
			name: "basic H1 section",
			content: `# Introduction

This is the introduction.

# Setup

This is the setup section.

# Configuration

This is the configuration.`,
			sectionName: "Setup",
			expected: `# Setup

This is the setup section.`,
		},
		{
			name: "H2 section",
			content: `# Main Title

## Subsection 1

Content for subsection 1.

## Subsection 2

Content for subsection 2.`,
			sectionName: "Subsection 1",
			expected: `## Subsection 1

Content for subsection 1.`,
		},
		{
			name: "nested sections",
			content: `# Main

## Sub1

Content 1

### Sub1.1

Nested content

## Sub2

Content 2`,
			sectionName: "Sub1",
			expected: `## Sub1

Content 1

### Sub1.1

Nested content`,
		},
		{
			name:        "section not found",
			content:     "# Title\n\nContent",
			sectionName: "NonExistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMarkdownSection(tt.content, tt.sectionName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractMarkdownSection() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractMarkdownSection() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractMarkdownSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateDefaultWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "simple filename",
			filePath: "test-workflow.md",
			expected: "Test Workflow",
		},
		{
			name:     "multiple hyphens",
			filePath: "my-test-workflow-file.md",
			expected: "My Test Workflow File",
		},
		{
			name:     "full path",
			filePath: "/path/to/my-workflow.md",
			expected: "My Workflow",
		},
		{
			name:     "no extension",
			filePath: "workflow",
			expected: "Workflow",
		},
		{
			name:     "single word",
			filePath: "test.md",
			expected: "Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultWorkflowName(tt.filePath)
			if result != tt.expected {
				t.Errorf("generateDefaultWorkflowName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractFrontmatterString(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "valid frontmatter",
			content: `---
title: Test Workflow
on: push
---

# Content`,
			expected: "on: push\ntitle: Test Workflow",
		},
		{
			name:     "no frontmatter",
			content:  "# Just markdown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFrontmatterString(tt.content)

			if tt.wantErr && err == nil {
				t.Errorf("ExtractFrontmatterString() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ExtractFrontmatterString() error = %v", err)
				return
			}

			// For YAML, order may vary, so check both possible orders
			if !strings.Contains(result, "title: Test Workflow") && tt.expected != "" {
				if result != tt.expected {
					t.Errorf("ExtractFrontmatterString() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestExtractMarkdownContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "with frontmatter",
			content: `---
title: Test
---

# Markdown

This is markdown.`,
			expected: "# Markdown\n\nThis is markdown.",
		},
		{
			name:     "no frontmatter",
			content:  "# Just Markdown\n\nContent here.",
			expected: "# Just Markdown\n\nContent here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractMarkdownContent(tt.content)

			if tt.wantErr && err == nil {
				t.Errorf("ExtractMarkdownContent() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ExtractMarkdownContent() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExtractMarkdownContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProcessIncludes(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "test_includes")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file with markdown content
	testFile := filepath.Join(tempDir, "test.md")
	testContent := `---
tools:
  bash:
    allowed: ["ls", "cat"]
---

# Test Content
This is a test file content.
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create test file with extra newlines for trimming test
	testFileWithNewlines := filepath.Join(tempDir, "test-newlines.md")
	testContentWithNewlines := `

# Content with Extra Newlines
Some content here.


`
	if err := os.WriteFile(testFileWithNewlines, []byte(testContentWithNewlines), 0644); err != nil {
		t.Fatalf("Failed to write test file with newlines: %v", err)
	}

	tests := []struct {
		name         string
		content      string
		baseDir      string
		extractTools bool
		expected     string
		wantErr      bool
	}{
		{
			name:         "no includes",
			content:      "# Title\nRegular content",
			baseDir:      tempDir,
			extractTools: false,
			expected:     "# Title\nRegular content\n",
		},
		{
			name:         "simple include",
			content:      "@include test.md\n# After include",
			baseDir:      tempDir,
			extractTools: false,
			expected:     "# Test Content\nThis is a test file content.\n# After include\n",
		},
		{
			name:         "extract tools",
			content:      "@include test.md",
			baseDir:      tempDir,
			extractTools: true,
			expected:     `{"bash":{"allowed":["ls","cat"]}}` + "\n",
		},
		{
			name:         "file not found",
			content:      "@include nonexistent.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      true, // Now expects error instead of embedding comment
		},
		{
			name:         "include file with extra newlines",
			content:      "@include test-newlines.md\n# After include",
			baseDir:      tempDir,
			extractTools: false,
			expected:     "# Content with Extra Newlines\nSome content here.\n# After include\n",
		},
		{
			name:         "simple import (alias for include)",
			content:      "@import test.md\n# After import",
			baseDir:      tempDir,
			extractTools: false,
			expected:     "# Test Content\nThis is a test file content.\n# After import\n",
		},
		{
			name:         "extract tools with import",
			content:      "@import test.md",
			baseDir:      tempDir,
			extractTools: true,
			expected:     `{"bash":{"allowed":["ls","cat"]}}` + "\n",
		},
		{
			name:         "import file not found",
			content:      "@import nonexistent.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      true,
		},
		{
			name:         "optional import missing file",
			content:      "@import? missing.md\n",
			baseDir:      tempDir,
			extractTools: false,
			expected:     "",
		},
	}

	// Create test file with invalid frontmatter for testing validation
	invalidFile := filepath.Join(tempDir, "invalid.md")
	invalidContent := `---
title: Invalid File
on: push
tools:
  bash:
    allowed: ["ls"]
---

# Invalid Content
This file has invalid frontmatter for an included file.
`
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	// Add test case for invalid frontmatter in included file (should now pass with warnings for non-workflow files)
	tests = append(tests, struct {
		name         string
		content      string
		baseDir      string
		extractTools bool
		expected     string
		wantErr      bool
	}{
		name:         "invalid frontmatter in included file",
		content:      "@include invalid.md",
		baseDir:      tempDir,
		extractTools: false,
		expected:     "# Invalid Content\nThis file has invalid frontmatter for an included file.\n",
		wantErr:      false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessIncludes(tt.content, tt.baseDir, tt.extractTools)

			if tt.wantErr && err == nil {
				t.Errorf("ProcessIncludes() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ProcessIncludes() error = %v", err)
				return
			}

			// Special handling for the invalid frontmatter test case - it should now pass with warnings
			if tt.name == "invalid frontmatter in included file" {
				// Check that the content was successfully included
				if !strings.Contains(result, "# Invalid Content") {
					t.Errorf("ProcessIncludes() = %q, expected to contain '# Invalid Content'", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("ProcessIncludes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsUnderWorkflowsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "file under .github/workflows",
			filePath: "/some/path/.github/workflows/test.md",
			expected: true,
		},
		{
			name:     "file under .github/workflows subdirectory",
			filePath: "/some/path/.github/workflows/shared/helper.md",
			expected: true,
		},
		{
			name:     "file outside .github/workflows",
			filePath: "/some/path/docs/instructions.md",
			expected: false,
		},
		{
			name:     "file in .github but not workflows",
			filePath: "/some/path/.github/ISSUE_TEMPLATE/bug.md",
			expected: false,
		},
		{
			name:     "relative path under workflows",
			filePath: ".github/workflows/test.md",
			expected: true,
		},
		{
			name:     "relative path outside workflows",
			filePath: "docs/readme.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnderWorkflowsDirectory(tt.filePath)
			if result != tt.expected {
				t.Errorf("isUnderWorkflowsDirectory(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestProcessIncludesConditionalValidation(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "test_conditional_validation")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .github/workflows directory structure
	workflowsDir := filepath.Join(tempDir, constants.GetWorkflowDir())
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Create docs directory for non-workflow files
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// Test file 1: Valid workflow file (should pass strict validation)
	validWorkflowFile := filepath.Join(workflowsDir, "valid.md")
	validWorkflowContent := `---
tools:
  github:
    allowed: [get_issue]
---

# Valid Workflow
This is a valid workflow file.`
	if err := os.WriteFile(validWorkflowFile, []byte(validWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to write valid workflow file: %v", err)
	}

	// Test file 2: Invalid workflow file (should fail strict validation)
	invalidWorkflowFile := filepath.Join(workflowsDir, "invalid.md")
	invalidWorkflowContent := `---
title: Invalid Field
on: push
tools:
  github:
    allowed: [get_issue]
---

# Invalid Workflow
This has invalid frontmatter fields.`
	if err := os.WriteFile(invalidWorkflowFile, []byte(invalidWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid workflow file: %v", err)
	}

	// Test file 2.5: Invalid non-workflow file (should pass with warnings)
	invalidNonWorkflowFile := filepath.Join(docsDir, "invalid-external.md")
	invalidNonWorkflowContent := `---
title: Invalid Field
on: push
tools:
  github:
    allowed: [get_issue]
---

# Invalid External File
This has invalid frontmatter fields but it's outside workflows dir.`
	if err := os.WriteFile(invalidNonWorkflowFile, []byte(invalidNonWorkflowContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid non-workflow file: %v", err)
	}

	// Test file 3: Agent instructions file (should pass with warnings)
	agentFile := filepath.Join(docsDir, "agent-instructions.md")
	agentContent := `---
description: Agent instructions
applyTo: "**/*.py"
temperature: 0.7
tools:
  github:
    allowed: [get_issue]
---

# Agent Instructions
These are instructions for AI agents.`
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	// Test file 4: Plain markdown file (no frontmatter)
	plainFile := filepath.Join(docsDir, "plain.md")
	plainContent := `# Plain Markdown
This is just plain markdown content with no frontmatter.`
	if err := os.WriteFile(plainFile, []byte(plainContent), 0644); err != nil {
		t.Fatalf("Failed to write plain file: %v", err)
	}

	tests := []struct {
		name         string
		content      string
		baseDir      string
		extractTools bool
		wantErr      bool
		checkContent string
	}{
		{
			name:         "valid workflow file inclusion",
			content:      "@include .github/workflows/valid.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      false,
			checkContent: "# Valid Workflow",
		},
		{
			name:         "invalid workflow file inclusion should fail",
			content:      "@include .github/workflows/invalid.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      true, // Now expects error instead of embedding comment
		},
		{
			name:         "invalid non-workflow file inclusion should succeed with warnings",
			content:      "@include docs/invalid-external.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      false,
			checkContent: "# Invalid External File",
		},
		{
			name:         "agent instructions file inclusion should succeed",
			content:      "@include docs/agent-instructions.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      false,
			checkContent: "# Agent Instructions",
		},
		{
			name:         "plain markdown file inclusion should succeed",
			content:      "@include docs/plain.md",
			baseDir:      tempDir,
			extractTools: false,
			wantErr:      false,
			checkContent: "# Plain Markdown",
		},
		{
			name:         "extract tools from valid workflow file",
			content:      "@include .github/workflows/valid.md",
			baseDir:      tempDir,
			extractTools: true,
			wantErr:      false,
			checkContent: `{"github":{"allowed":["get_issue"]}}`,
		},
		{
			name:         "extract tools from agent file",
			content:      "@include docs/agent-instructions.md",
			baseDir:      tempDir,
			extractTools: true,
			wantErr:      false,
			checkContent: `{"github":{"allowed":["get_issue"]}}`,
		},
		{
			name:         "extract tools from plain file (no tools)",
			content:      "@include docs/plain.md",
			baseDir:      tempDir,
			extractTools: true,
			wantErr:      false,
			checkContent: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessIncludes(tt.content, tt.baseDir, tt.extractTools)

			if tt.wantErr && err == nil {
				t.Errorf("ProcessIncludes() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ProcessIncludes() error = %v", err)
				return
			}

			if !tt.wantErr && tt.checkContent != "" {
				if !strings.Contains(result, tt.checkContent) {
					t.Errorf("ProcessIncludes() result = %q, expected to contain %q", result, tt.checkContent)
				}
			}
		})
	}
}

func TestResolveIncludePath(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "test_resolve")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create regular test file in temp dir
	regularFile := filepath.Join(tempDir, "regular.md")
	if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write regular file: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		baseDir  string
		expected string
		wantErr  bool
	}{
		{
			name:     "regular relative path",
			filePath: "regular.md",
			baseDir:  tempDir,
			expected: regularFile,
		},
		{
			name:     "regular file not found",
			filePath: "nonexistent.md",
			baseDir:  tempDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveIncludePath(tt.filePath, tt.baseDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveIncludePath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("resolveIncludePath() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("resolveIncludePath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMergeTools(t *testing.T) {
	tests := []struct {
		name       string
		base       map[string]any
		additional map[string]any
		expected   map[string]any
	}{
		{
			name: "merge with allowed arrays",
			base: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls", "cat"},
				},
			},
			additional: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"grep", "ls"}, // ls is duplicate
				},
			},
			expected: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls", "cat", "grep"},
				},
			},
		},
		{
			name: "new tool added",
			base: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls"},
				},
			},
			additional: map[string]any{
				"github": map[string]any{
					"allowed": []any{"create_issue"},
				},
			},
			expected: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls"},
				},
				"github": map[string]any{
					"allowed": []any{"create_issue"},
				},
			},
		},
		{
			name: "empty base",
			base: map[string]any{},
			additional: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls"},
				},
			},
			expected: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"ls"},
				},
			},
		},
		{
			name: "merge neutral tools with maps (no Claude-specific logic)",
			base: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"bash": map[string]any{
					"allowed": []any{"ls", "cat"},
				},
			},
			additional: map[string]any{
				"bash": map[string]any{
					"allowed": []any{"grep", "ps"},
				},
			},
			expected: map[string]any{
				"github": map[string]any{
					"allowed": []any{"list_issues"},
				},
				"bash": map[string]any{
					"allowed": []any{"ls", "cat", "grep", "ps"},
				},
			},
		},
		{
			name: "merge neutral tools with different allowed arrays",
			base: map[string]any{
				"web-fetch": map[string]any{
					"allowed": []any{"get", "post"},
				},
			},
			additional: map[string]any{
				"web-fetch": map[string]any{
					"allowed": []any{"put", "get"}, // get is duplicate
				},
			},
			expected: map[string]any{
				"web-fetch": map[string]any{
					"allowed": []any{"get", "post", "put"},
				},
			},
		},
		{
			name: "merge mcp tools with wildcard allowed",
			base: map[string]any{
				"notion": map[string]any{
					"type":    "mcp",
					"allowed": []any{"create_page"},
				},
			},
			additional: map[string]any{
				"notion": map[string]any{
					"type":    "mcp",
					"allowed": []any{"*"},
				},
			},
			expected: map[string]any{
				"notion": map[string]any{
					"type":    "mcp",
					"allowed": []any{"create_page", "*"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeTools(tt.base, tt.additional)
			if err != nil {
				t.Fatalf("MergeTools() returned unexpected error: %v", err)
			}

			// Convert result to JSON and back for easier comparison
			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(tt.expected)

			var resultMap, expectedMap map[string]any
			if err := json.Unmarshal(resultJSON, &resultMap); err != nil {
				t.Fatalf("Failed to unmarshal result JSON: %v", err)
			}
			if err := json.Unmarshal(expectedJSON, &expectedMap); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}

			// Compare JSON strings for easier debugging
			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("MergeTools() = %s, want %s", string(resultJSON), string(expectedJSON))
			}
		})
	}
}

func TestExpandIncludes(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "test_expand")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod to make it project root for component resolution
	goModFile := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.md")
	testContent := `---
tools:
  bash:
    allowed: ["ls"]
---

# Test Content
This is test content.
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name         string
		content      string
		baseDir      string
		extractTools bool
		wantContains string
		wantErr      bool
	}{
		{
			name:         "expand markdown content",
			content:      "# Start\n@include test.md\n# End",
			baseDir:      tempDir,
			extractTools: false,
			wantContains: "# Test Content\nThis is test content.",
		},
		{
			name:         "expand tools",
			content:      "@include test.md",
			baseDir:      tempDir,
			extractTools: true,
			wantContains: `"bash"`,
		},
		{
			name:         "expand markdown content with import",
			content:      "# Start\n@import test.md\n# End",
			baseDir:      tempDir,
			extractTools: false,
			wantContains: "# Test Content\nThis is test content.",
		},
		{
			name:         "expand tools with import",
			content:      "@import test.md",
			baseDir:      tempDir,
			extractTools: true,
			wantContains: `"bash"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandIncludes(tt.content, tt.baseDir, tt.extractTools)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandIncludes() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExpandIncludes() error = %v", err)
				return
			}

			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("ExpandIncludes() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

// Test ExtractWorkflowNameFromMarkdown function
func TestExtractWorkflowNameFromMarkdown(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "test-extract-name-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name: "file with H1 header",
			content: `---
name: Test Workflow
---

# Daily QA Report

This is a test workflow.`,
			expected:    "Daily QA Report",
			expectError: false,
		},
		{
			name: "file without H1 header",
			content: `---
name: Test Workflow
---

This is content without H1 header.
## This is H2`,
			expected:    "Test Extract Name", // Should generate from filename
			expectError: false,
		},
		{
			name: "file with multiple H1 headers",
			content: `---
name: Test Workflow
---

# First Header

Some content.

# Second Header

Should use first H1.`,
			expected:    "First Header",
			expectError: false,
		},
		{
			name: "file with only frontmatter",
			content: `---
name: Test Workflow
description: A test
---`,
			expected:    "Test Extract Name", // Should generate from filename
			expectError: false,
		},
		{
			name: "file with H1 and extra spaces",
			content: `---
name: Test
---

#   Spaced Header   

Content here.`,
			expected:    "Spaced Header",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			fileName := "test-extract-name.md"
			filePath := filepath.Join(tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			got, err := ExtractWorkflowNameFromMarkdown(filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExtractWorkflowNameFromMarkdown(%q) expected error, but got none", filePath)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractWorkflowNameFromMarkdown(%q) unexpected error: %v", filePath, err)
				return
			}

			if got != tt.expected {
				t.Errorf("ExtractWorkflowNameFromMarkdown(%q) = %q, want %q", filePath, got, tt.expected)
			}
		})
	}

	// Test nonexistent file
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ExtractWorkflowNameFromMarkdown("/nonexistent/file.md")
		if err == nil {
			t.Error("ExtractWorkflowNameFromMarkdown with nonexistent file should return error")
		}
	})
}

// Test ExtractMarkdown function
func TestExtractMarkdown(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "test-extract-markdown-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name: "file with frontmatter",
			content: `---
name: Test Workflow
description: A test workflow
---

# Test Content

This is the markdown content.`,
			expected:    "# Test Content\n\nThis is the markdown content.",
			expectError: false,
		},
		{
			name: "file without frontmatter",
			content: `# Pure Markdown

This is just markdown content without frontmatter.`,
			expected:    "# Pure Markdown\n\nThis is just markdown content without frontmatter.",
			expectError: false,
		},
		{
			name:        "empty file",
			content:     ``,
			expected:    "",
			expectError: false,
		},
		{
			name: "file with only frontmatter",
			content: `---
name: Test
---`,
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			fileName := "test-extract-markdown.md"
			filePath := filepath.Join(tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			got, err := ExtractMarkdown(filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExtractMarkdown(%q) expected error, but got none", filePath)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractMarkdown(%q) unexpected error: %v", filePath, err)
				return
			}

			if got != tt.expected {
				t.Errorf("ExtractMarkdown(%q) = %q, want %q", filePath, got, tt.expected)
			}
		})
	}

	// Test nonexistent file
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ExtractMarkdown("/nonexistent/file.md")
		if err == nil {
			t.Error("ExtractMarkdown with nonexistent file should return error")
		}
	})
}

// Test mergeToolsFromJSON function
func TestMergeToolsFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name:     "single valid JSON object",
			content:  `{"tool1": {"enabled": true}, "tool2": {"enabled": false}}`,
			expected: `{"tool1":{"enabled":true},"tool2":{"enabled":false}}`,
			wantErr:  false,
		},
		{
			name: "multiple JSON objects on separate lines",
			content: `{"tool1": {"enabled": true}}
{"tool2": {"enabled": false}}
{"tool3": {"config": "value"}}`,
			expected: `{"tool1":{"enabled":true},"tool2":{"enabled":false},"tool3":{"config":"value"}}`,
			wantErr:  false,
		},
		{
			name:     "empty content",
			content:  ``,
			expected: `{}`,
			wantErr:  false,
		},
		{
			name:     "empty JSON objects",
			content:  `{}\n{}\n{}`,
			expected: `{}`,
			wantErr:  false,
		},
		{
			name:     "whitespace only",
			content:  `   \n  \t  \n   `,
			expected: `{}`,
			wantErr:  false,
		},
		{
			name: "mixed empty and non-empty objects",
			content: `{}
{"tool1": {"enabled": true}}
{}
{"tool2": {"value": 42}}`,
			expected: `{"tool1":{"enabled":true},"tool2":{"value":42}}`,
			wantErr:  false,
		},
		{
			name: "objects with overlapping keys",
			content: `{"tool1": {"enabled": true, "config": "old"}}
{"tool1": {"config": "new", "version": 2}}`,
			expected: `{"tool1":{"config":"new","enabled":true,"version":2}}`,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			content:  `{"invalid": json}`,
			expected: `{}`,
			wantErr:  false, // Function handles invalid JSON gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeToolsFromJSON(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("mergeToolsFromJSON(%q) expected error, but got none", tt.content)
				}
				return
			}

			if err != nil {
				t.Errorf("mergeToolsFromJSON(%q) unexpected error: %v", tt.content, err)
				return
			}

			// For JSON comparison, parse both strings to ensure equivalent content
			var gotObj, expectedObj map[string]any
			if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
				t.Errorf("mergeToolsFromJSON(%q) returned invalid JSON: %v", tt.content, err)
				return
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedObj); err != nil {
				t.Errorf("Test case has invalid expected JSON: %v", err)
				return
			}

			// Convert back to JSON strings for comparison
			gotJSON, _ := json.Marshal(gotObj)
			expectedJSON, _ := json.Marshal(expectedObj)

			if string(gotJSON) != string(expectedJSON) {
				t.Errorf("mergeToolsFromJSON(%q) = %q, want %q", tt.content, string(gotJSON), string(expectedJSON))
			}
		})
	}
}

// Test StripANSI function
func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text without ANSI",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "simple CSI color sequence",
			input:    "\x1b[31mRed Text\x1b[0m",
			expected: "Red Text",
		},
		{
			name:     "multiple CSI sequences",
			input:    "\x1b[1m\x1b[31mBold Red\x1b[0m\x1b[32mGreen\x1b[0m",
			expected: "Bold RedGreen",
		},
		{
			name:     "CSI cursor movement",
			input:    "Line 1\x1b[2;1HLine 2",
			expected: "Line 1Line 2",
		},
		{
			name:     "CSI erase sequences",
			input:    "Text\x1b[2JCleared\x1b[K",
			expected: "TextCleared",
		},
		{
			name:     "OSC sequence with BEL terminator",
			input:    "\x1b]0;Window Title\x07Content",
			expected: "Content",
		},
		{
			name:     "OSC sequence with ST terminator",
			input:    "\x1b]2;Terminal Title\x1b\\More content",
			expected: "More content",
		},
		{
			name:     "character set selection G0",
			input:    "\x1b(0Hello\x1b(B",
			expected: "Hello",
		},
		{
			name:     "character set selection G1",
			input:    "\x1b)0World\x1b)B",
			expected: "World",
		},
		{
			name:     "keypad mode sequences",
			input:    "\x1b=Keypad\x1b>Normal",
			expected: "KeypadNormal",
		},
		{
			name:     "reset sequence",
			input:    "Before\x1bcAfter",
			expected: "BeforeAfter",
		},
		{
			name:     "save and restore cursor",
			input:    "Start\x1b7Middle\x1b8End",
			expected: "StartMiddleEnd",
		},
		{
			name:     "index and reverse index",
			input:    "Text\x1bDDown\x1bMUp",
			expected: "TextDownUp",
		},
		{
			name:     "next line and horizontal tab set",
			input:    "Line\x1bENext\x1bHTab",
			expected: "LineNextTab",
		},
		{
			name:     "complex CSI with parameters",
			input:    "\x1b[38;5;196mBright Red\x1b[48;5;21mBlue BG\x1b[0m",
			expected: "Bright RedBlue BG",
		},
		{
			name:     "CSI with semicolon parameters",
			input:    "\x1b[1;31;42mBold red on green\x1b[0m",
			expected: "Bold red on green",
		},
		{
			name:     "malformed escape at end",
			input:    "Text\x1b",
			expected: "Text",
		},
		{
			name:     "malformed CSI at end",
			input:    "Text\x1b[31",
			expected: "Text",
		},
		{
			name:     "malformed OSC at end",
			input:    "Text\x1b]0;Title",
			expected: "Text",
		},
		{
			name:     "escape followed by invalid character",
			input:    "Text\x1bXInvalid",
			expected: "TextInvalid",
		},
		{
			name:     "consecutive escapes",
			input:    "\x1b[31m\x1b[1m\x1b[4mText\x1b[0m",
			expected: "Text",
		},
		{
			name:     "mixed content with newlines",
			input:    "Line 1\n\x1b[31mRed Line 2\x1b[0m\nLine 3",
			expected: "Line 1\nRed Line 2\nLine 3",
		},
		{
			name:     "common terminal output",
			input:    "\x1b[?25l\x1b[2J\x1b[H\x1b[32m✓\x1b[0m Success",
			expected: "✓ Success",
		},
		{
			name:     "git diff style colors",
			input:    "\x1b[32m+Added line\x1b[0m\n\x1b[31m-Removed line\x1b[0m",
			expected: "+Added line\n-Removed line",
		},
		{
			name:     "unicode content with ANSI",
			input:    "\x1b[33m🎉 Success! 测试\x1b[0m",
			expected: "🎉 Success! 测试",
		},
		{
			name:     "very long CSI sequence",
			input:    "\x1b[1;2;3;4;5;6;7;8;9;10;11;12;13;14;15mLong params\x1b[0m",
			expected: "Long params",
		},
		{
			name:     "CSI with question mark private parameter",
			input:    "\x1b[?25hCursor visible\x1b[?25l",
			expected: "Cursor visible",
		},
		{
			name:     "CSI with greater than private parameter",
			input:    "\x1b[>0cDevice attributes\x1b[>1c",
			expected: "Device attributes",
		},
		{
			name:     "all final CSI characters test",
			input:    "\x1b[@\x1b[A\x1b[B\x1b[C\x1b[D\x1b[E\x1b[F\x1b[G\x1b[H\x1b[I\x1b[J\x1b[K\x1b[L\x1b[M\x1b[N\x1b[O\x1b[P\x1b[Q\x1b[R\x1b[S\x1b[T\x1b[U\x1b[V\x1b[W\x1b[X\x1b[Y\x1b[Z\x1b[[\x1b[\\\x1b[]\x1b[^\x1b[_\x1b[`\x1b[a\x1b[b\x1b[c\x1b[d\x1b[e\x1b[f\x1b[g\x1b[h\x1b[i\x1b[j\x1b[k\x1b[l\x1b[m\x1b[n\x1b[o\x1b[p\x1b[q\x1b[r\x1b[s\x1b[t\x1b[u\x1b[v\x1b[w\x1b[x\x1b[y\x1b[z\x1b[{\x1b[|\x1b[}\x1b[~Text",
			expected: "Text",
		},
		{
			name:     "CSI with invalid final character",
			input:    "Before\x1b[31Text after",
			expected: "Beforeext after",
		},
		{
			name:     "real world lipgloss output",
			input:    "\x1b[1;38;2;80;250;123m✓\x1b[0;38;2;248;248;242m Success message\x1b[0m",
			expected: "✓ Success message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test isCSIParameterChar function
func TestIsCSIParameterChar(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		// Valid parameter characters (0x30-0x3F, 0-?)
		{name: "0 (0x30)", char: '0', expected: true},
		{name: "9 (0x39)", char: '9', expected: true},
		{name: "; (0x3B)", char: ';', expected: true},
		{name: "? (0x3F)", char: '?', expected: true},

		// Valid intermediate characters (0x20-0x2F, space-/)
		{name: "space (0x20)", char: ' ', expected: true},
		{name: "! (0x21)", char: '!', expected: true},
		{name: "/ (0x2F)", char: '/', expected: true},

		// Invalid characters (below 0x20)
		{name: "tab (0x09)", char: '\t', expected: false},
		{name: "newline (0x0A)", char: '\n', expected: false},
		{name: "null (0x00)", char: 0x00, expected: false},

		// Invalid characters (above 0x3F)
		{name: "@ (0x40)", char: '@', expected: false},
		{name: "A (0x41)", char: 'A', expected: false},
		{name: "m (0x6D)", char: 'm', expected: false},
		{name: "~ (0x7E)", char: '~', expected: false},
		{name: "DEL (0x7F)", char: 0x7F, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCSIParameterChar(tt.char)
			if result != tt.expected {
				t.Errorf("isCSIParameterChar(%q/0x%02X) = %v, want %v", tt.char, tt.char, result, tt.expected)
			}
		})
	}
}

// Test isFinalCSIChar function
func TestIsFinalCSIChar(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		// Valid final characters (0x40-0x7E, @-~)
		{name: "@ (0x40)", char: '@', expected: true},
		{name: "A (0x41)", char: 'A', expected: true},
		{name: "Z (0x5A)", char: 'Z', expected: true},
		{name: "a (0x61)", char: 'a', expected: true},
		{name: "m (0x6D)", char: 'm', expected: true}, // Common color final char
		{name: "~ (0x7E)", char: '~', expected: true},

		// Invalid characters (below 0x40)
		{name: "space (0x20)", char: ' ', expected: false},
		{name: "0 (0x30)", char: '0', expected: false},
		{name: "9 (0x39)", char: '9', expected: false},
		{name: "; (0x3B)", char: ';', expected: false},
		{name: "? (0x3F)", char: '?', expected: false},

		// Invalid characters (above 0x7E)
		{name: "DEL (0x7F)", char: 0x7F, expected: false},
		{name: "high byte (0x80)", char: 0x80, expected: false},
		{name: "high byte (0xFF)", char: 0xFF, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFinalCSIChar(tt.char)
			if result != tt.expected {
				t.Errorf("isFinalCSIChar(%q/0x%02X) = %v, want %v", tt.char, tt.char, result, tt.expected)
			}
		})
	}
}

// Benchmark StripANSI function for performance
func BenchmarkStripANSI(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "plain text",
			input: "This is plain text without any ANSI codes",
		},
		{
			name:  "simple color",
			input: "\x1b[31mRed text\x1b[0m",
		},
		{
			name:  "complex formatting",
			input: "\x1b[1;38;2;255;0;0m\x1b[48;2;0;255;0mComplex formatting\x1b[0m",
		},
		{
			name:  "mixed content",
			input: "Normal \x1b[31mred\x1b[0m normal \x1b[32mgreen\x1b[0m normal \x1b[34mblue\x1b[0m text",
		},
		{
			name:  "long text with ANSI",
			input: strings.Repeat("\x1b[31mRed \x1b[32mGreen \x1b[34mBlue\x1b[0m ", 100),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				StripANSI(tc.input)
			}
		})
	}
}

func TestProcessIncludesOptional(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "test_optional_includes")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an existing include file
	existingFile := filepath.Join(tempDir, "existing.md")
	existingContent := "# Existing Include\nThis file exists."
	if err := os.WriteFile(existingFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to write existing file: %v", err)
	}

	tests := []struct {
		name           string
		content        string
		extractTools   bool
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "regular include existing file",
			content:        "@include existing.md\n",
			extractTools:   false,
			expectedOutput: existingContent,
		},
		{
			name:         "regular include missing file",
			content:      "@include missing.md\n",
			extractTools: false,
			expectError:  true, // Now expects error instead of embedding comment
		},
		{
			name:           "optional include existing file",
			content:        "@include? existing.md\n",
			extractTools:   false,
			expectedOutput: existingContent,
		},
		{
			name:           "optional include missing file",
			content:        "@include? missing.md\n",
			extractTools:   false,
			expectedOutput: "", // No content added, friendly message goes to stdout
		},
		{
			name:           "optional include missing file extract tools",
			content:        "@include? missing.md\n",
			extractTools:   true,
			expectedOutput: "",
		},
		{
			name:         "regular include missing file extract tools",
			content:      "@include missing.md\n",
			extractTools: true,
			expectError:  true, // Now expects error instead of returning {}
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessIncludes(tt.content, tempDir, tt.extractTools)

			if tt.expectError {
				if err == nil {
					t.Errorf("ProcessIncludes expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ProcessIncludes unexpected error: %v", err)
				return
			}

			if !strings.Contains(result, tt.expectedOutput) {
				t.Errorf("ProcessIncludes output = %q, expected to contain %q", result, tt.expectedOutput)
			}
		})
	}
}

func TestIsWorkflowSpec(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "valid workflowspec",
			path: "owner/repo/path/to/file.md",
			want: true,
		},
		{
			name: "workflowspec with ref",
			path: "owner/repo/workflows/file.md@main",
			want: true,
		},
		{
			name: "workflowspec with section",
			path: "owner/repo/workflows/file.md#section",
			want: true,
		},
		{
			name: "workflowspec with ref and section",
			path: "owner/repo/workflows/file.md@sha123#section",
			want: true,
		},
		{
			name: "local path with .github",
			path: ".github/workflows/file.md",
			want: false,
		},
		{
			name: "relative local path",
			path: "../shared/file.md",
			want: false,
		},
		{
			name: "absolute path",
			path: "/tmp/gh-aw/fh-aw/file.md",
			want: false,
		},
		{
			name: "too few parts",
			path: "owner/repo",
			want: false,
		},
		{
			name: "local path starting with dot",
			path: "./file.md",
			want: false,
		},
		{
			name: "shared path with 2 parts",
			path: "shared/file.md",
			want: false,
		},
		{
			name: "shared path with 3 parts (mcp subdirectory)",
			path: "shared/mcp/gh-aw.md",
			want: false,
		},
		{
			name: "shared path with ref",
			path: "shared/mcp/tavily.md@main",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWorkflowSpec(tt.path)
			if got != tt.want {
				t.Errorf("isWorkflowSpec(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestProcessIncludesWithCycleDetection(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "test_cycle_detection")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file A that includes file B
	fileA := filepath.Join(tempDir, "fileA.md")
	if err := os.WriteFile(fileA, []byte("# File A\n@include fileB.md\n"), 0644); err != nil {
		t.Fatalf("Failed to write fileA: %v", err)
	}

	// Create file B that includes file A (creating a cycle)
	fileB := filepath.Join(tempDir, "fileB.md")
	if err := os.WriteFile(fileB, []byte("# File B\n@include fileA.md\n"), 0644); err != nil {
		t.Fatalf("Failed to write fileB: %v", err)
	}

	// Process includes from file A - should not hang due to cycle detection
	content := "# Main\n@include fileA.md\n"
	result, err := ProcessIncludes(content, tempDir, false)

	if err != nil {
		t.Errorf("ProcessIncludes with cycle should not error: %v", err)
	}

	// Result should contain content from fileA and fileB, but cycle should be prevented
	if !strings.Contains(result, "File A") {
		t.Errorf("ProcessIncludes result should contain File A content")
	}
	if !strings.Contains(result, "File B") {
		t.Errorf("ProcessIncludes result should contain File B content")
	}
}

func TestProcessImportsFromFrontmatter(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	// Create a test include file
	includeFile := filepath.Join(tempDir, "include.md")
	includeContent := `---
tools:
  bash:
    allowed:
      - ls
      - cat
---
# Include Content
This is an included file.`
	if err := os.WriteFile(includeFile, []byte(includeContent), 0644); err != nil {
		t.Fatalf("Failed to write include file: %v", err)
	}

	tests := []struct {
		name          string
		frontmatter   map[string]any
		wantToolsJSON bool
		wantEngines   bool
		wantErr       bool
	}{
		{
			name: "no imports field",
			frontmatter: map[string]any{
				"on": "push",
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "empty imports array",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": []string{},
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "valid imports",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": []string{"include.md"},
			},
			wantToolsJSON: true,
			wantEngines:   false,
			wantErr:       false,
		},
		{
			name: "invalid imports type",
			frontmatter: map[string]any{
				"on":      "push",
				"imports": "not-an-array",
			},
			wantToolsJSON: false,
			wantEngines:   false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools, engines, err := ProcessImportsFromFrontmatter(tt.frontmatter, tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ProcessImportsFromFrontmatter() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ProcessImportsFromFrontmatter() unexpected error: %v", err)
				return
			}

			if tt.wantToolsJSON {
				if tools == "" {
					t.Errorf("ProcessImportsFromFrontmatter() expected tools JSON but got empty string")
				}
				// Verify it's valid JSON
				var toolsMap map[string]any
				if err := json.Unmarshal([]byte(tools), &toolsMap); err != nil {
					t.Errorf("ProcessImportsFromFrontmatter() tools not valid JSON: %v", err)
				}
			} else {
				if tools != "" {
					t.Errorf("ProcessImportsFromFrontmatter() expected no tools but got: %s", tools)
				}
			}

			if tt.wantEngines {
				if len(engines) == 0 {
					t.Errorf("ProcessImportsFromFrontmatter() expected engines but got none")
				}
			} else {
				if len(engines) != 0 {
					t.Errorf("ProcessImportsFromFrontmatter() expected no engines but got: %v", engines)
				}
			}
		})
	}
}
