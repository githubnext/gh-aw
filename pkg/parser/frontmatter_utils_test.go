//go:build !integration

package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

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
			expected: false, // Files in subdirectories are not top-level workflow files
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

func TestIsCustomAgentFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "file under .github/agents with .md extension",
			filePath: "/some/path/.github/agents/test-agent.md",
			expected: true,
		},
		{
			name:     "file under .github/agents with .agent.md extension",
			filePath: "/some/path/.github/agents/feature-flag-remover.agent.md",
			expected: true,
		},
		{
			name:     "file under .github/agents subdirectory",
			filePath: "/some/path/.github/agents/subdir/helper.md",
			expected: true, // Still an agent file even in subdirectory
		},
		{
			name:     "file outside .github/agents",
			filePath: "/some/path/docs/instructions.md",
			expected: false,
		},
		{
			name:     "file in .github/workflows",
			filePath: "/some/path/.github/workflows/test.md",
			expected: false,
		},
		{
			name:     "file in .github but not agents",
			filePath: "/some/path/.github/ISSUE_TEMPLATE/bug.md",
			expected: false,
		},
		{
			name:     "relative path under agents",
			filePath: ".github/agents/test-agent.md",
			expected: true,
		},
		{
			name:     "file under agents but not markdown",
			filePath: ".github/agents/config.json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCustomAgentFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("isCustomAgentFile(%q) = %v, want %v", tt.filePath, result, tt.expected)
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
			result, err := ResolveIncludePath(tt.filePath, tt.baseDir, nil)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveIncludePath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveIncludePath() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ResolveIncludePath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

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

// Benchmark StripANSI function for performance

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
			path: "/tmp/gh-aw/gh-aw/file.md",
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

func TestProcessImportsFromFrontmatter(t *testing.T) {
	// Create temp directory for test files
	tempDir := testutil.TempDir(t, "test-*")

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

// TestProcessIncludedFileWithNameAndDescription verifies that name and description fields
// do not generate warnings when processing included files outside .github/workflows/
