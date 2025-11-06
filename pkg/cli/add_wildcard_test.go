package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseWorkflowSpecWithWildcard tests parsing workflow specs with wildcards
func TestParseWorkflowSpecWithWildcard(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		expectWildcard bool
		expectError    bool
		expectedRepo   string
		expectedVer    string
	}{
		{
			name:           "wildcard_without_version",
			spec:           "githubnext/agentics/*",
			expectWildcard: true,
			expectError:    false,
			expectedRepo:   "githubnext/agentics",
			expectedVer:    "",
		},
		{
			name:           "wildcard_with_version",
			spec:           "githubnext/agentics/*@v1.0.0",
			expectWildcard: true,
			expectError:    false,
			expectedRepo:   "githubnext/agentics",
			expectedVer:    "v1.0.0",
		},
		{
			name:           "wildcard_with_branch",
			spec:           "owner/repo/*@main",
			expectWildcard: true,
			expectError:    false,
			expectedRepo:   "owner/repo",
			expectedVer:    "main",
		},
		{
			name:           "non_wildcard_spec",
			spec:           "githubnext/agentics/workflow-name",
			expectWildcard: false,
			expectError:    false,
			expectedRepo:   "githubnext/agentics",
			expectedVer:    "",
		},
		{
			name:           "invalid_spec_too_few_parts",
			spec:           "owner/*",
			expectWildcard: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWorkflowSpec(tt.spec)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseWorkflowSpec() expected error for spec '%s', got nil", tt.spec)
				}
				return
			}

			if err != nil {
				t.Errorf("parseWorkflowSpec() unexpected error: %v", err)
				return
			}

			if result.IsWildcard != tt.expectWildcard {
				t.Errorf("parseWorkflowSpec() IsWildcard = %v, expected %v", result.IsWildcard, tt.expectWildcard)
			}

			if tt.expectWildcard {
				if result.WorkflowPath != "*" {
					t.Errorf("parseWorkflowSpec() WorkflowPath = %v, expected '*'", result.WorkflowPath)
				}
				if result.WorkflowName != "*" {
					t.Errorf("parseWorkflowSpec() WorkflowName = %v, expected '*'", result.WorkflowName)
				}
			}

			if result.RepoSlug != tt.expectedRepo {
				t.Errorf("parseWorkflowSpec() RepoSlug = %v, expected %v", result.RepoSlug, tt.expectedRepo)
			}

			if result.Version != tt.expectedVer {
				t.Errorf("parseWorkflowSpec() Version = %v, expected %v", result.Version, tt.expectedVer)
			}
		})
	}
}

// TestDiscoverWorkflowsInPackage tests discovering workflows in an installed package
func TestDiscoverWorkflowsInPackage(t *testing.T) {
	// Create a temporary packages directory structure
	tempDir := t.TempDir()

	// Override packages directory for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create a mock package structure (use .aw/packages, not .gh-aw/packages)
	packagePath := filepath.Join(tempDir, ".aw", "packages", "test-owner", "test-repo")
	workflowsDir := filepath.Join(packagePath, "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}

	// Create some mock workflow files
	workflows := []string{
		"workflow1.md",
		"workflow2.md",
		"nested/workflow3.md",
	}

	for _, wf := range workflows {
		filePath := filepath.Join(packagePath, wf)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte("# Test Workflow"), 0644); err != nil {
			t.Fatalf("Failed to create test workflow %s: %v", wf, err)
		}
	}

	// Test discovery
	discovered, err := discoverWorkflowsInPackage("test-owner/test-repo", "", false)
	if err != nil {
		t.Fatalf("discoverWorkflowsInPackage() error = %v", err)
	}

	if len(discovered) != len(workflows) {
		t.Errorf("discoverWorkflowsInPackage() found %d workflows, expected %d", len(discovered), len(workflows))
	}

	// Verify discovered workflow paths
	discoveredPaths := make(map[string]bool)
	for _, spec := range discovered {
		discoveredPaths[spec.WorkflowPath] = true
	}

	for _, expectedPath := range workflows {
		if !discoveredPaths[expectedPath] {
			t.Errorf("Expected workflow %s not found in discovered workflows", expectedPath)
		}
	}

	// Verify all specs have correct repo info
	for _, spec := range discovered {
		if spec.RepoSlug != "test-owner/test-repo" {
			t.Errorf("Workflow spec has incorrect RepoSlug: %s, expected test-owner/test-repo", spec.RepoSlug)
		}
		if spec.IsWildcard {
			t.Errorf("Discovered workflow spec should not be marked as wildcard")
		}
	}
}

// TestDiscoverWorkflowsInPackage_NotFound tests behavior when package is not found
func TestDiscoverWorkflowsInPackage_NotFound(t *testing.T) {
	// Create a temporary packages directory
	tempDir := t.TempDir()

	// Override packages directory for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Try to discover workflows in a non-existent package
	_, err := discoverWorkflowsInPackage("nonexistent/repo", "", false)
	if err == nil {
		t.Error("discoverWorkflowsInPackage() expected error for non-existent package, got nil")
	}

	if !strings.Contains(err.Error(), "package not found") {
		t.Errorf("discoverWorkflowsInPackage() error should mention 'package not found', got: %v", err)
	}
}

// TestDiscoverWorkflowsInPackage_EmptyPackage tests behavior with empty package
func TestDiscoverWorkflowsInPackage_EmptyPackage(t *testing.T) {
	// Create a temporary packages directory
	tempDir := t.TempDir()

	// Override packages directory for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create an empty package directory (use .aw/packages, not .gh-aw/packages)
	packagePath := filepath.Join(tempDir, ".aw", "packages", "empty-owner", "empty-repo")
	if err := os.MkdirAll(packagePath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Test discovery
	discovered, err := discoverWorkflowsInPackage("empty-owner/empty-repo", "", false)
	if err != nil {
		t.Fatalf("discoverWorkflowsInPackage() error = %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("discoverWorkflowsInPackage() found %d workflows in empty package, expected 0", len(discovered))
	}
}
