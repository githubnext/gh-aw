package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestInjectSourceField(t *testing.T) {
	tests := []struct {
		name                 string
		content              string
		sourceInfo           *WorkflowSourceInfo
		originalWorkflowPath string
		expectedSource       string
		expectError          bool
	}{
		{
			name: "inject source field for package workflow",
			content: `---
on: push
engine: claude
---

# Test Workflow

This is a test workflow.`,
			sourceInfo: &WorkflowSourceInfo{
				IsPackage:   true,
				PackagePath: "/home/user/.aw/packages/testorg/testrepo",
				SourcePath:  "/home/user/.aw/packages/testorg/testrepo/test-workflow.md",
			},
			originalWorkflowPath: "test-workflow.md",
			expectedSource:       "testorg/testrepo main test-workflow.md",
		},
		{
			name: "no injection for non-package workflow",
			content: `---
on: push
engine: claude
---

# Test Workflow

This is a test workflow.`,
			sourceInfo: &WorkflowSourceInfo{
				IsPackage:  false,
				SourcePath: "/path/to/local/workflow.md",
			},
			originalWorkflowPath: "workflow.md",
			expectedSource:       "", // No source field should be added
		},
		{
			name: "inject source field with existing frontmatter",
			content: `---
on: push
engine: claude
permissions:
  contents: read
---

# Test Workflow

This is a test with existing frontmatter.`,
			sourceInfo: &WorkflowSourceInfo{
				IsPackage:   true,
				PackagePath: "/home/user/.aw/packages/github/awesome-workflows",
				SourcePath:  "/home/user/.aw/packages/github/awesome-workflows/ci-doctor.md",
			},
			originalWorkflowPath: "ci-doctor.md",
			expectedSource:       "github/awesome-workflows main ci-doctor.md",
		},
		{
			name: "inject source field with no frontmatter",
			content: `# Simple Workflow

This workflow has no frontmatter.`,
			sourceInfo: &WorkflowSourceInfo{
				IsPackage:   true,
				PackagePath: "/tmp/.aw/packages/myorg/myrepo",
				SourcePath:  "/tmp/.aw/packages/myorg/myrepo/simple.md",
			},
			originalWorkflowPath: "simple.md",
			expectedSource:       "myorg/myrepo main simple.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary metadata file if this is a package workflow
			if tt.sourceInfo.IsPackage {
				// Create temporary directory structure
				tmpDir := t.TempDir()
				packagePath := filepath.Join(tmpDir, ".aw", "packages", "testorg", "testrepo")
				if strings.Contains(tt.sourceInfo.PackagePath, "github/awesome-workflows") {
					packagePath = filepath.Join(tmpDir, ".aw", "packages", "github", "awesome-workflows")
				} else if strings.Contains(tt.sourceInfo.PackagePath, "myorg/myrepo") {
					packagePath = filepath.Join(tmpDir, ".aw", "packages", "myorg", "myrepo")
				}

				err := os.MkdirAll(packagePath, 0755)
				if err != nil {
					t.Fatalf("Failed to create package directory: %v", err)
				}

				// Create metadata file
				metadataFile := filepath.Join(packagePath, ".aw-metadata")
				metadataContent := "commit_sha=abc123def456\nother_field=value\n"
				err = os.WriteFile(metadataFile, []byte(metadataContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create metadata file: %v", err)
				}

				// Update the source info with the temporary path
				tt.sourceInfo.PackagePath = packagePath
				tt.sourceInfo.SourcePath = filepath.Join(packagePath, filepath.Base(tt.originalWorkflowPath))

				// Update expected source with actual commit SHA
				if tt.expectedSource != "" {
					parts := strings.Split(tt.expectedSource, " ")
					if len(parts) >= 3 {
						parts[1] = "abc123def456" // Replace "main" with actual commit SHA
						tt.expectedSource = strings.Join(parts, " ")
					}
				}
			}

			result, err := injectSourceField(tt.content, tt.sourceInfo, tt.originalWorkflowPath, false)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Parse the result to check if source field was added correctly
			parsed, err := parser.ExtractFrontmatterFromContent(result)
			if err != nil {
				t.Fatalf("Failed to parse result frontmatter: %v", err)
			}

			if tt.expectedSource == "" {
				// Should not have source field
				if source, exists := parsed.Frontmatter["source"]; exists {
					t.Errorf("Expected no source field, but got: %v", source)
				}
			} else {
				// Should have source field with expected value
				source, exists := parsed.Frontmatter["source"]
				if !exists {
					t.Errorf("Expected source field to exist, but it doesn't")
				} else if source != tt.expectedSource {
					t.Errorf("Expected source field '%s', but got '%s'", tt.expectedSource, source)
				}
			}

			// Verify that other frontmatter fields are preserved
			if tt.sourceInfo.IsPackage && strings.Contains(tt.content, "on: push") {
				if on, exists := parsed.Frontmatter["on"]; !exists || on != "push" {
					t.Errorf("Expected 'on' field to be preserved, got %v", on)
				}
			}

			// Verify that markdown content is preserved
			if !strings.Contains(result, "Test Workflow") && !strings.Contains(result, "Simple Workflow") {
				t.Errorf("Markdown content was not preserved in result")
			}
		})
	}
}

func TestInjectSourceFieldErrorHandling(t *testing.T) {
	// Test case with invalid frontmatter
	content := `---
invalid yaml: [
---

# Test`

	sourceInfo := &WorkflowSourceInfo{
		IsPackage:   true,
		PackagePath: "/tmp/.aw/packages/org/repo",
		SourcePath:  "/tmp/.aw/packages/org/repo/test.md",
	}

	result, err := injectSourceField(content, sourceInfo, "test.md", false)

	// Should handle error gracefully and return original content
	if err == nil {
		t.Errorf("Expected error due to invalid YAML, but got none")
	}

	// Should return original content when there's an error
	if result != content {
		t.Errorf("Expected original content to be returned on error")
	}
}

func TestReadCommitSHAFromMetadata(t *testing.T) {
	tests := []struct {
		name           string
		metadataContent string
		expectedSHA    string
	}{
		{
			name:           "valid metadata with commit_sha",
			metadataContent: "commit_sha=abc123def456\nother_field=value\n",
			expectedSHA:    "abc123def456",
		},
		{
			name:           "metadata without commit_sha",
			metadataContent: "other_field=value\nyet_another=test\n",
			expectedSHA:    "",
		},
		{
			name:           "empty metadata",
			metadataContent: "",
			expectedSHA:    "",
		},
		{
			name:           "metadata with spaces",
			metadataContent: "  commit_sha=def789ghi012  \nother=value\n",
			expectedSHA:    "def789ghi012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and metadata file
			tmpDir := t.TempDir()
			metadataFile := filepath.Join(tmpDir, ".aw-metadata")

			err := os.WriteFile(metadataFile, []byte(tt.metadataContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create metadata file: %v", err)
			}

			result := readCommitSHAFromMetadata(tmpDir)

			if result != tt.expectedSHA {
				t.Errorf("Expected SHA '%s', but got '%s'", tt.expectedSHA, result)
			}
		})
	}

	// Test case with missing metadata file
	t.Run("missing metadata file", func(t *testing.T) {
		tmpDir := t.TempDir()
		result := readCommitSHAFromMetadata(tmpDir)

		if result != "" {
			t.Errorf("Expected empty string for missing metadata file, but got '%s'", result)
		}
	})
}