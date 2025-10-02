package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

func TestGenerateSourceField(t *testing.T) {
	tests := []struct {
		name           string
		sourceInfo     *WorkflowSourceInfo
		workflowPath   string
		expectedFormat string // Pattern to check (not exact match)
		expectError    bool
	}{
		{
			name: "valid package source",
			sourceInfo: &WorkflowSourceInfo{
				IsPackage:   true,
				PackagePath: filepath.Join(os.TempDir(), ".aw", "packages", "githubnext", "agentics"),
				SourcePath:  filepath.Join(os.TempDir(), ".aw", "packages", "githubnext", "agentics", "researcher.md"),
			},
			workflowPath:   "researcher.md",
			expectedFormat: "githubnext/agentics",
			expectError:    false,
		},
		{
			name: "non-package source",
			sourceInfo: &WorkflowSourceInfo{
				IsPackage: false,
			},
			workflowPath: "test.md",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary package directory if needed
			if tt.sourceInfo.IsPackage {
				err := os.MkdirAll(tt.sourceInfo.PackagePath, 0755)
				if err != nil {
					t.Fatalf("Failed to create temp package dir: %v", err)
				}
				defer os.RemoveAll(filepath.Join(os.TempDir(), ".aw"))

				// Create metadata file
				metadataFile := filepath.Join(tt.sourceInfo.PackagePath, ".aw-metadata")
				metadataContent := "commit_sha=abc123def456\n"
				err = os.WriteFile(metadataFile, []byte(metadataContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create metadata file: %v", err)
				}
			}

			result, err := generateSourceField(tt.sourceInfo, tt.workflowPath, false)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !strings.Contains(result, tt.expectedFormat) {
				t.Errorf("Expected source field to contain '%s', got: %s", tt.expectedFormat, result)
			}

			// Check format: should be "org/repo ref path.md"
			parts := strings.Fields(result)
			if len(parts) != 3 {
				t.Errorf("Expected source field to have 3 parts, got %d: %s", len(parts), result)
			}
		})
	}
}

func TestAddSourceToWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		source      string
		expectError bool
	}{
		{
			name: "add source to workflow with frontmatter",
			content: `---
on: push
---

# Test Workflow

This is a test.`,
			source:      "githubnext/agentics abc123 researcher.md",
			expectError: false,
		},
		{
			name: "add source to workflow without frontmatter",
			content: `# Test Workflow

This is a test.`,
			source:      "githubnext/agentics abc123 researcher.md",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := addSourceToWorkflow(tt.content, tt.source, false)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Parse the result to verify source was added
			frontmatter, err := parser.ExtractFrontmatterFromContent(result)
			if err != nil {
				t.Errorf("Failed to parse result: %v", err)
				return
			}

			source, ok := frontmatter.Frontmatter["source"].(string)
			if !ok {
				t.Errorf("Source field not found in frontmatter")
				return
			}

			if source != tt.source {
				t.Errorf("Expected source '%s', got '%s'", tt.source, source)
			}
		})
	}
}

func TestFindWorkflowsWithSource(t *testing.T) {
	// Create temporary workflows directory
	tempDir := t.TempDir()

	// Create workflows with and without source
	workflows := []struct {
		name       string
		hasSource  bool
		sourceSpec string
	}{
		{name: "with-source.md", hasSource: true, sourceSpec: "githubnext/agentics abc123 test.md"},
		{name: "without-source.md", hasSource: false},
		{name: "another-with-source.md", hasSource: true, sourceSpec: "org/repo def456 workflow.md"},
	}

	for _, wf := range workflows {
		content := "---\non: push\n"
		if wf.hasSource {
			content += "source: " + wf.sourceSpec + "\n"
		}
		content += "---\n\n# Test Workflow"

		filePath := filepath.Join(tempDir, wf.name)
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test workflow: %v", err)
		}
	}

	// Test finding all workflows with source
	result, err := findWorkflowsWithSource(tempDir, nil, false)
	if err != nil {
		t.Fatalf("Failed to find workflows: %v", err)
	}

	expectedCount := 2 // Two workflows have source
	if len(result) != expectedCount {
		t.Errorf("Expected %d workflows with source, got %d", expectedCount, len(result))
	}

	// Test finding specific workflow
	result, err = findWorkflowsWithSource(tempDir, []string{"with-source"}, false)
	if err != nil {
		t.Fatalf("Failed to find workflows: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 workflow, got %d", len(result))
	}

	if len(result) > 0 && result[0].SourceSpec != "githubnext/agentics abc123 test.md" {
		t.Errorf("Expected source 'githubnext/agentics abc123 test.md', got '%s'", result[0].SourceSpec)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	tests := []struct {
		name           string
		sourceSpec     string
		expectedResult string
	}{
		{
			name:           "invalid source format",
			sourceSpec:     "invalid",
			expectedResult: "Invalid",
		},
		{
			name:           "package not installed",
			sourceSpec:     "org/repo abc123 test.md",
			expectedResult: "Not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test.md")

			// Create a test workflow file
			content := "---\non: push\nsource: " + tt.sourceSpec + "\n---\n\n# Test"
			err := os.WriteFile(workflowPath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test workflow: %v", err)
			}

			result := checkUpdateAvailable(workflowPath, tt.sourceSpec, false)

			if result != tt.expectedResult {
				t.Errorf("Expected '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}
