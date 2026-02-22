//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestExtractWorkflowNameFromFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "test-*")

	tests := []struct {
		name        string
		content     string
		filename    string
		expected    string
		expectError bool
	}{
		{
			name: "file with H1 header",
			content: `---
title: Test Workflow
---

# Daily Test Coverage Improvement

This is a test workflow.`,
			filename:    "test-workflow.md",
			expected:    "Daily Test Coverage Improvement",
			expectError: false,
		},
		{
			name: "file with H1 header with extra spaces",
			content: `# Weekly Research   

This is a research workflow.`,
			filename:    "weekly-research.md",
			expected:    "Weekly Research",
			expectError: false,
		},
		{
			name: "file without H1 header - generates from filename",
			content: `This is content without H1 header.

## Some H2 header

Content here.`,
			filename:    "daily-dependency-updates.md",
			expected:    "Daily Dependency Updates",
			expectError: false,
		},
		{
			name:        "file with complex filename",
			content:     `No headers here.`,
			filename:    "complex-workflow-name-test.md",
			expected:    "Complex Workflow Name Test",
			expectError: false,
		},
		{
			name:        "file with single word filename",
			content:     `No headers.`,
			filename:    "workflow.md",
			expected:    "Workflow",
			expectError: false,
		},
		{
			name:        "empty file - generates from filename",
			content:     "",
			filename:    "empty-workflow.md",
			expected:    "Empty Workflow",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test the function
			result, err := extractWorkflowNameFromFile(filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestExtractWorkflowNameFromFile_NonExistentFile(t *testing.T) {
	_, err := extractWorkflowNameFromFile("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestIsGitRepo(t *testing.T) {
	// Test in current directory (should be a git repo based on project setup)
	result := isGitRepo()

	// Since we're running in a git repository, this should return true
	if !result {
		t.Error("Expected isGitRepo() to return true in git repository")
	}
}

// TestFindGitRoot is already tested in gitroot_test.go, skipping duplicate

func TestExtractWorkflowNameFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple workflow file",
			path:     ".github/workflows/daily-test.lock.yml",
			expected: "daily-test",
		},
		{
			name:     "workflow file without lock suffix",
			path:     ".github/workflows/weekly-research.yml",
			expected: "weekly-research",
		},
		{
			name:     "nested path",
			path:     "/home/user/project/.github/workflows/complex-workflow-name.lock.yml",
			expected: "complex-workflow-name",
		},
		{
			name:     "file without extension",
			path:     ".github/workflows/workflow",
			expected: "workflow",
		},
		{
			name:     "single file name",
			path:     "test.yml",
			expected: "test",
		},
		{
			name:     "file with multiple dots",
			path:     "test.lock.yml",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkflowNameFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFindIncludesInContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "no includes",
			content:  "This is regular content without includes.",
			expected: []string{},
		},
		{
			name: "single include",
			content: `This is content with include:
@include shared/tools.md
More content here.`,
			expected: []string{"shared/tools.md"},
		},
		{
			name: "multiple includes",
			content: `Content with multiple includes:
@include shared/tools.md
Some content between.
@include shared/config.md
More content.
@include another/file.md`,
			expected: []string{"shared/tools.md", "shared/config.md", "another/file.md"},
		},
		{
			name: "includes with different whitespace",
			content: `Content:
@include shared/tools.md
@include  shared/config.md  
@include	shared/tabs.md`,
			expected: []string{"shared/tools.md", "shared/config.md", "shared/tabs.md"},
		},
		{
			name: "includes with section references",
			content: `Content:
@include shared/tools.md#Tools
@include shared/config.md#Configuration`,
			expected: []string{"shared/tools.md", "shared/config.md"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findIncludesInContent(tt.content, "", false)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d includes, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, expected := range tt.expected {
				if i >= len(result) || result[i] != expected {
					t.Errorf("Expected include %d to be %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkExtractWorkflowNameFromFile(b *testing.B) {
	// Create temporary test file
	tmpDir := b.TempDir()
	content := `---
title: Test Workflow
---

# Daily Test Coverage Improvement

This is a test workflow with some content.`

	filePath := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	for b.Loop() {
		_, _ = extractWorkflowNameFromFile(filePath)
	}
}

func BenchmarkFindIncludesInContent(b *testing.B) {
	content := `This is content with includes:
@include shared/tools.md
Some content between includes.
@include shared/config.md
More content here.
@include another/file.md
Final content.`

	for b.Loop() {
		_, _ = findIncludesInContent(content, "", false)
	}
}

func TestIsRunnable(t *testing.T) {
	tests := []struct {
		name         string
		mdContent    string
		lockContent  string
		expected     bool
		expectError  bool
		errorMessage string
	}{
		{
			name: "workflow with schedule trigger",
			mdContent: `---
on:
  schedule:
    - cron: "0 9 * * *"
---
# Test Workflow
This workflow runs on schedule.`,
			lockContent: `name: "Test Workflow"
on:
  schedule:
    - cron: "0 9 * * *"
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "workflow with workflow_dispatch trigger",
			mdContent: `---
on:
  workflow_dispatch:
---
# Manual Workflow
This workflow can be triggered manually.`,
			lockContent: `name: "Manual Workflow"
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "workflow with both schedule and workflow_dispatch",
			mdContent: `---
on:
  schedule:
    - cron: "0 9 * * 1"  
  workflow_dispatch:
  push:
    branches: [main]
---
# Mixed Triggers Workflow`,
			lockContent: `name: "Mixed Triggers Workflow"
on:
  schedule:
    - cron: "0 9 * * 1"
  workflow_dispatch:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "workflow with only push trigger (not runnable)",
			mdContent: `---
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
---
# CI Workflow
This is not runnable via schedule or manual dispatch.`,
			lockContent: `name: "CI Workflow"
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    false,
			expectError: false,
		},
		{
			name: "workflow with no 'on' section (defaults to runnable)",
			mdContent: `---
name: Default Workflow
---
# Default Workflow
No on section means it defaults to runnable.`,
			lockContent: `name: "Default Workflow"
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    false,
			expectError: false,
		},
		{
			name: "workflow with cron trigger (alternative schedule format)",
			mdContent: `---
on:
  cron: "0 */6 * * *"
---
# Cron Workflow
Uses cron format directly.`,
			lockContent: `name: "Cron Workflow"
on:
  schedule:
    - cron: "0 */6 * * *"
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "case insensitive schedule detection",
			mdContent: `---
on:
  SCHEDULE:
    - cron: "0 12 * * 0"
---
# Case Test Workflow`,
			lockContent: `name: "Case Test Workflow"
on:
  schedule:
    - cron: "0 12 * * 0"
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "case insensitive workflow_dispatch detection",
			mdContent: `---
on:
  WORKFLOW_DISPATCH:
---
# Case Test Manual Workflow`,
			lockContent: `name: "Case Test Manual Workflow"
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "complex on section with schedule buried in text",
			mdContent: `---
on:
  push:
    branches: [main]
  schedule:
    - cron: "0 0 * * 0"  # Weekly
  issues:
    types: [opened]
---
# Complex Workflow`,
			lockContent: `name: "Complex Workflow"
on:
  push:
    branches: [main]
  schedule:
    - cron: "0 0 * * 0"
  workflow_dispatch:
  issues:
    types: [opened]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
		{
			name: "empty on section (not runnable)",
			mdContent: `---
on: {}
---
# Empty On Section`,
			lockContent: `name: "Empty On Section"
on: {}
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    false,
			expectError: false,
		},
		{
			name: "malformed frontmatter",
			mdContent: `---
invalid yaml structure {
on:
  schedule
---
# Malformed YAML`,
			lockContent:  `invalid yaml`,
			expected:     false,
			expectError:  true,
			errorMessage: "failed to parse lock file YAML",
		},
		{
			name: "no frontmatter at all (defaults to runnable)",
			mdContent: `# Simple Markdown
This file has no frontmatter.
Just plain markdown content.`,
			lockContent: `name: "Simple Markdown"
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"
`,
			expected:    true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test files
			tmpDir := testutil.TempDir(t, "test-*")
			mdPath := filepath.Join(tmpDir, "test-workflow.md")
			lockPath := filepath.Join(tmpDir, "test-workflow.lock.yml")

			err := os.WriteFile(mdPath, []byte(tt.mdContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create markdown file: %v", err)
			}

			err = os.WriteFile(lockPath, []byte(tt.lockContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create lock file: %v", err)
			}

			// Test the function
			result, err := IsRunnable(mdPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestIsRunnable_FileErrors(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		expectErr bool
	}{
		{
			name:      "nonexistent file",
			filePath:  "/nonexistent/path/workflow.md",
			expectErr: true,
		},
		{
			name:      "empty file path",
			filePath:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsRunnable(tt.filePath)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				// Result should be false when there's an error
				if result {
					t.Errorf("Expected false result on error, got true")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
