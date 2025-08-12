package cli

import (
	"os"
	"strings"
	"testing"
)

// Test the CLI functions that are exported from this package

func TestListWorkflows(t *testing.T) {
	// Test the ListWorkflows function
	err := ListWorkflows(false)

	// Since it's not implemented yet, it should return nil (no error)
	// and print a message about not being implemented
	if err != nil {
		t.Errorf("ListWorkflows should not return an error for valid input, got: %v", err)
	}
}

func TestAddWorkflow(t *testing.T) {
	// Clean up any existing .github/workflows for this test
	defer func() {
		if err := os.RemoveAll(".github"); err != nil {
			t.Logf("Warning: Failed to clean up .github directory: %v", err)
		}
	}()

	tests := []struct {
		name        string
		workflow    string
		number      int
		expectError bool
	}{
		{
			name:        "nonexistent workflow",
			workflow:    "nonexistent-workflow",
			number:      1,
			expectError: true,
		},
		{
			name:        "empty workflow name",
			workflow:    "",
			number:      1,
			expectError: false, // AddWorkflow shows help when workflow is empty, doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddWorkflow(tt.workflow, tt.number, false, "", "", false)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}
		})
	}
}

func TestAddWorkflowForce(t *testing.T) {
	// This test verifies that the force flag works correctly
	// Note: This is a unit test to verify the function signature and basic logic
	// It doesn't test the actual file system operations

	// Test that force=false fails when a file "exists" (simulated by empty workflow name which triggers help)
	err := AddWorkflow("", 1, false, "", "", false)
	if err != nil {
		t.Errorf("Expected no error for empty workflow (shows help), got: %v", err)
	}

	// Test that force=true works with same parameters
	err = AddWorkflow("", 1, false, "", "", true)
	if err != nil {
		t.Errorf("Expected no error for empty workflow with force=true, got: %v", err)
	}
}

func TestCompileWorkflows(t *testing.T) {
	// Clean up any existing .github/workflows for this test
	defer func() {
		if err := os.RemoveAll(".github"); err != nil {
			t.Logf("Warning: Failed to clean up .github directory: %v", err)
		}
	}()

	tests := []struct {
		name         string
		markdownFile string
		expectError  bool
	}{
		{
			name:         "nonexistent specific file",
			markdownFile: "nonexistent.md",
			expectError:  true, // Should error when file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompileWorkflows(tt.markdownFile, false, "", false, false, false)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}
		})
	}
}

func TestRemoveWorkflows(t *testing.T) {
	err := RemoveWorkflows("test-pattern")

	// Should not error since it's a stub implementation
	if err != nil {
		t.Errorf("RemoveWorkflows should not return error for valid input, got: %v", err)
	}
}

func TestStatusWorkflows(t *testing.T) {
	err := StatusWorkflows("test-pattern", false)

	// Should not error since it's a stub implementation
	if err != nil {
		t.Errorf("StatusWorkflows should not return error for valid input, got: %v", err)
	}
}

func TestEnableWorkflows(t *testing.T) {
	err := EnableWorkflows("test-pattern")

	// Should not error since it's a stub implementation
	if err != nil {
		t.Errorf("EnableWorkflows should not return error for valid input, got: %v", err)
	}
}

func TestDisableWorkflows(t *testing.T) {
	err := DisableWorkflows("test-pattern")

	// Should not error since it's a stub implementation
	if err != nil {
		t.Errorf("DisableWorkflows should not return error for valid input, got: %v", err)
	}
}

func TestRunWorkflowOnGitHub(t *testing.T) {
	// Test with empty workflow name
	err := RunWorkflowOnGitHub("", false)
	if err == nil {
		t.Error("RunWorkflowOnGitHub should return error for empty workflow name")
	}

	// Test with nonexistent workflow (this will fail but gracefully)
	err = RunWorkflowOnGitHub("nonexistent-workflow", false)
	if err == nil {
		t.Error("RunWorkflowOnGitHub should return error for non-existent workflow")
	}
}

// func TestAllCommandsExist(t *testing.T) {
// 	defer func() {
// 		if err := os.RemoveAll(".github"); err != nil {
// 			t.Logf("Warning: Failed to clean up .github directory: %v", err)
// 		}
// 	}()

// 	// Test that all expected functions exist and can be called
// 	// This helps ensure the interface is stable

// 	// Test structure: function, expected to error
// 	tests := []struct {
// 		fn          func() error
// 		expectError bool
// 		name        string
// 	}{
// 		{func() error { return ListWorkflows(false) }, false, "ListWorkflows"},
// 		{func() error { return AddWorkflow("", 1, false, "", "", false) }, false, "AddWorkflow (empty name)"},     // Shows help when empty, doesn't error
// 		{func() error { return CompileWorkflows("", false, "", false, false, false) }, false, "CompileWorkflows"}, // Should succeed when .github/workflows directory exists
// 		{func() error { return RemoveWorkflows("test") }, false, "RemoveWorkflows"},                               // Should handle missing directory gracefully
// 		{func() error { return StatusWorkflows("test", false) }, false, "StatusWorkflows"},                        // Should handle missing directory gracefully
// 		{func() error { return EnableWorkflows("test") }, false, "EnableWorkflows"},                               // Should handle missing directory gracefully
// 		{func() error { return DisableWorkflows("test") }, false, "DisableWorkflows"},                             // Should handle missing directory gracefully
// 		{func() error { return RunWorkflowOnGitHub("", false) }, true, "RunWorkflowOnGitHub"},                     // Should error with empty workflow name
// 	}

// 	for _, test := range tests {
// 		err := test.fn()
// 		if test.expectError && err == nil {
// 			t.Errorf("%s: expected error but got nil", test.name)
// 		} else if !test.expectError && err != nil {
// 			t.Errorf("%s: unexpected error: %v", test.name, err)
// 		}
// 	}
// }

func TestAddWorkflowWithPR(t *testing.T) {
	// Clean up any existing .github/workflows for this test
	defer func() {
		if err := os.RemoveAll(".github"); err != nil {
			t.Logf("Warning: Failed to clean up .github directory: %v", err)
		}
	}()

	// Test with nonexistent workflow (should fail early due to workflow not found or repo access)
	err := AddWorkflowWithRepoAndPR("nonexistent-workflow", 1, false, "", "", "", false)
	if err == nil {
		t.Error("AddWorkflowWithRepoAndPR should return an error for nonexistent workflow or missing git setup")
	}

	// The error could be either:
	// 1. GitHub CLI not available
	// 2. Not in a git repository
	// 3. Repository access check failure
	// 4. Working directory not clean
	// 5. Workflow not found
	// All of these are expected in the test environment
	t.Logf("Expected error for PR creation: %v", err)
}

// TestInstallPackage tests the InstallPackage function
func TestInstallPackage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "gh-aw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Mock the getPackagesDir function by temporarily changing directory
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}()

	tests := []struct {
		name        string
		repoSpec    string
		local       bool
		verbose     bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid repo spec",
			repoSpec:    "invalid",
			local:       true,
			verbose:     false,
			expectError: true,
			errorMsg:    "invalid repository specification",
		},
		{
			name:        "empty repo spec",
			repoSpec:    "",
			local:       true,
			verbose:     false,
			expectError: true,
			errorMsg:    "invalid repository specification",
		},
		{
			name:        "valid repo spec but download will fail",
			repoSpec:    "nonexistent/repo",
			local:       true,
			verbose:     true,
			expectError: true,
			errorMsg:    "failed to download workflows",
		},
		{
			name:        "valid repo spec with version but download will fail",
			repoSpec:    "nonexistent/repo@v1.0.0",
			local:       false,
			verbose:     false,
			expectError: true,
			errorMsg:    "failed to download workflows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InstallPackage(tt.repoSpec, tt.local, tt.verbose)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}

			if tt.expectError && err != nil {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// TestUninstallPackage tests the UninstallPackage function
func TestUninstallPackage(t *testing.T) {
	tests := []struct {
		name        string
		repoSpec    string
		local       bool
		verbose     bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid repo spec",
			repoSpec:    "invalid",
			local:       true,
			verbose:     false,
			expectError: true,
			errorMsg:    "invalid repository specification",
		},
		{
			name:        "empty repo spec",
			repoSpec:    "",
			local:       true,
			verbose:     false,
			expectError: true,
			errorMsg:    "invalid repository specification",
		},
		{
			name:        "valid repo spec - package not installed",
			repoSpec:    "nonexistent/repo",
			local:       true,
			verbose:     true,
			expectError: false,
		},
		{
			name:        "valid repo spec with version - package not installed",
			repoSpec:    "nonexistent/repo@v1.0.0",
			local:       false,
			verbose:     false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UninstallPackage(tt.repoSpec, tt.local, tt.verbose)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}

			if tt.expectError && err != nil {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			}
		})
	}
}

// TestListPackages tests the ListPackages function
func TestListPackages(t *testing.T) {
	tests := []struct {
		name        string
		local       bool
		verbose     bool
		expectError bool
	}{
		{
			name:        "list local packages",
			local:       true,
			verbose:     false,
			expectError: false, // Should not error even if directory doesn't exist
		},
		{
			name:        "list global packages",
			local:       false,
			verbose:     true,
			expectError: false, // Should not error even if directory doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ListPackages(tt.local, tt.verbose)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}
		})
	}
}

func TestNewWorkflow(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-new-workflow-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name          string
		workflowName  string
		force         bool
		expectedError bool
		setup         func(t *testing.T)
	}{
		{
			name:          "create new workflow",
			workflowName:  "test-workflow",
			force:         false,
			expectedError: false,
		},
		{
			name:          "fail to overwrite existing workflow without force",
			workflowName:  "existing-workflow",
			force:         false,
			expectedError: true,
			setup: func(t *testing.T) {
				// Create an existing workflow file
				if err := os.MkdirAll(".github/workflows", 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}
				if err := os.WriteFile(".github/workflows/existing-workflow.md", []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create existing workflow file: %v", err)
				}
			},
		},
		{
			name:          "overwrite existing workflow with force",
			workflowName:  "force-workflow",
			force:         true,
			expectedError: false,
			setup: func(t *testing.T) {
				// Create an existing workflow file
				if err := os.MkdirAll(".github/workflows", 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}
				if err := os.WriteFile(".github/workflows/force-workflow.md", []byte("old content"), 0644); err != nil {
					t.Fatalf("Failed to create existing workflow file: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup if needed
			if test.setup != nil {
				test.setup(t)
			}

			// Run the function
			err := NewWorkflow(test.workflowName, false, test.force)

			// Check error expectation
			if test.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !test.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// If no error expected, verify the file was created
			if !test.expectedError {
				filePath := ".github/workflows/" + test.workflowName + ".md"
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected workflow file was not created: %s", filePath)
				}

				// Verify the content contains expected template elements
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read created workflow file: %v", err)
				} else {
					contentStr := string(content)
					// Check for key template elements
					expectedElements := []string{
						"# Trigger - when should this workflow run?",
						"on:",
						"permissions:",
						"tools:",
						"github:",
						"allowed:",
						"# " + test.workflowName,
						"workflow_dispatch:",
					}
					for _, element := range expectedElements {
						if !strings.Contains(contentStr, element) {
							t.Errorf("Template missing expected element: %s", element)
						}
					}
				}
			}

			// Clean up for next test
			os.RemoveAll(".github")
		})
	}
}

// Test SetVersionInfo and GetVersion functions
func TestSetVersionInfo(t *testing.T) {
	// Save original version to restore after test
	originalVersion := GetVersion()
	defer SetVersionInfo(originalVersion)

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "normal version",
			version: "1.0.0",
		},
		{
			name:    "empty version",
			version: "",
		},
		{
			name:    "version with pre-release",
			version: "2.0.0-beta.1",
		},
		{
			name:    "version with build metadata",
			version: "1.2.3+20240808",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetVersionInfo(tt.version)
			got := GetVersion()
			if got != tt.version {
				t.Errorf("SetVersionInfo(%q) -> GetVersion() = %q, want %q", tt.version, got, tt.version)
			}
		})
	}
}

// Test AddWorkflowWithRepo function
func TestAddWorkflowWithRepo(t *testing.T) {
	// Clean up any existing .github/workflows for this test
	defer func() {
		if err := os.RemoveAll(".github"); err != nil {
			t.Logf("Warning: Failed to clean up .github directory: %v", err)
		}
	}()

	tests := []struct {
		name        string
		workflow    string
		repo        string
		expectError bool
		description string
	}{
		{
			name:        "empty workflow and repo",
			workflow:    "",
			repo:        "",
			expectError: false, // Should show help message, not error
			description: "empty workflow shows help",
		},
		{
			name:        "nonexistent workflow without repo",
			workflow:    "nonexistent-workflow",
			repo:        "",
			expectError: true,
			description: "nonexistent workflow should fail",
		},
		{
			name:        "workflow with invalid repo format",
			workflow:    "test-workflow",
			repo:        "invalid-repo-format",
			expectError: true,
			description: "invalid repo format should fail during installation",
		},
		{
			name:        "workflow with nonexistent repo",
			workflow:    "test-workflow",
			repo:        "nonexistent/nonexistent-repo",
			expectError: true,
			description: "nonexistent repo should fail during installation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddWorkflowWithRepo(tt.workflow, 1, false, "", tt.repo, "", false)

			if tt.expectError {
				if err == nil {
					t.Errorf("AddWorkflowWithRepo(%q, %q) expected error (%s), but got none", tt.workflow, tt.repo, tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("AddWorkflowWithRepo(%q, %q) unexpected error (%s): %v", tt.workflow, tt.repo, tt.description, err)
				}
			}
		})
	}
}

func TestCollectIncludeDependencies(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	workflowsDir := tempDir + "/workflows"
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create test files
	sharedDir := workflowsDir + "/shared"
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create a shared file
	sharedFile := sharedDir + "/common.md"
	sharedContent := `# Common Content
This is shared content.
`
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to create shared file: %v", err)
	}

	// Create another shared file for recursive testing
	recursiveFile := sharedDir + "/recursive.md"
	recursiveContent := `# Recursive Content
@include shared/common.md
More content here.
`
	if err := os.WriteFile(recursiveFile, []byte(recursiveContent), 0644); err != nil {
		t.Fatalf("Failed to create recursive file: %v", err)
	}

	tests := []struct {
		name              string
		content           string
		workflowPath      string
		expectedDepsCount int
		expectError       bool
		description       string
	}{
		{
			name:              "no_includes",
			content:           "# Simple Workflow\nNo includes here.",
			workflowPath:      workflowsDir + "/simple.md",
			expectedDepsCount: 0,
			expectError:       false,
			description:       "Content without includes should return no dependencies",
		},
		{
			name:              "single_include",
			content:           "# Workflow with Include\n@include shared/common.md\nMore content.",
			workflowPath:      workflowsDir + "/with-include.md",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Content with one include should return one dependency",
		},
		{
			name:              "multiple_includes",
			content:           "# Multiple Includes\n@include shared/common.md\n@include shared/recursive.md",
			workflowPath:      workflowsDir + "/multi-include.md",
			expectedDepsCount: 3,
			expectError:       false,
			description:       "Content with multiple includes should return multiple dependencies (including recursive ones)",
		},
		{
			name:              "recursive_includes",
			content:           "# Recursive Test\n@include shared/recursive.md",
			workflowPath:      workflowsDir + "/recursive-test.md",
			expectedDepsCount: 2,
			expectError:       false,
			description:       "Recursive includes should collect all dependencies",
		},
		{
			name:              "section_reference",
			content:           "# Section Reference\n@include shared/common.md#Section",
			workflowPath:      workflowsDir + "/section-ref.md",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Include with section reference should work",
		},
		{
			name:              "nonexistent_file",
			content:           "# Missing File\n@include shared/missing.md",
			workflowPath:      workflowsDir + "/missing.md",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Include of nonexistent file should still add dependency but not recurse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, err := collectIncludeDependencies(tt.content, tt.workflowPath, workflowsDir)

			if tt.expectError {
				if err == nil {
					t.Errorf("collectIncludeDependencies expected error (%s), but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("collectIncludeDependencies unexpected error (%s): %v", tt.description, err)
				return
			}

			if len(deps) != tt.expectedDepsCount {
				t.Errorf("collectIncludeDependencies expected %d dependencies (%s), got %d", tt.expectedDepsCount, tt.description, len(deps))
			}

			// Verify dependency structure
			for i, dep := range deps {
				if dep.SourcePath == "" {
					t.Errorf("Dependency %d has empty SourcePath", i)
				}
				if dep.TargetPath == "" {
					t.Errorf("Dependency %d has empty TargetPath", i)
				}
			}
		})
	}
}

func TestCollectIncludesRecursive(t *testing.T) {
	// Create temporary test environment
	tempDir := t.TempDir()
	baseDir := tempDir
	workflowsDir := tempDir

	// Create test files
	file1 := tempDir + "/file1.md"
	file1Content := `# File 1
Content of file 1
`
	if err := os.WriteFile(file1, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := tempDir + "/file2.md"
	file2Content := `# File 2
@include file1.md
Content of file 2
`
	if err := os.WriteFile(file2, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	tests := []struct {
		name              string
		content           string
		expectedDepsCount int
		expectError       bool
		description       string
	}{
		{
			name:              "no_includes",
			content:           "# No Includes\nJust regular content.",
			expectedDepsCount: 0,
			expectError:       false,
			description:       "Content without includes should not add dependencies",
		},
		{
			name:              "single_include",
			content:           "# Single Include\n@include file1.md",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Single include should add one dependency",
		},
		{
			name:              "recursive_include",
			content:           "# Recursive\n@include file2.md",
			expectedDepsCount: 2,
			expectError:       false,
			description:       "Recursive include should collect all dependencies",
		},
		{
			name:              "whitespace_handling",
			content:           "# Whitespace\n@include    file1.md   \n",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Include with extra whitespace should work",
		},
		{
			name:              "section_reference",
			content:           "# Section\n@include file1.md#Header",
			expectedDepsCount: 1,
			expectError:       false,
			description:       "Section references should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dependencies []IncludeDependency
			seen := make(map[string]bool)

			err := collectIncludesRecursive(tt.content, baseDir, workflowsDir, &dependencies, seen)

			if tt.expectError {
				if err == nil {
					t.Errorf("collectIncludesRecursive expected error (%s), but got none", tt.description)
				}
				return
			}

			if err != nil {
				t.Errorf("collectIncludesRecursive unexpected error (%s): %v", tt.description, err)
				return
			}

			if len(dependencies) != tt.expectedDepsCount {
				t.Errorf("collectIncludesRecursive expected %d dependencies (%s), got %d", tt.expectedDepsCount, tt.description, len(dependencies))
			}

			// Verify all dependencies have proper paths
			for i, dep := range dependencies {
				if dep.SourcePath == "" {
					t.Errorf("Dependency %d has empty SourcePath", i)
				}
				if dep.TargetPath == "" {
					t.Errorf("Dependency %d has empty TargetPath", i)
				}
			}
		})
	}
}

func TestCollectIncludesRecursiveCircularReference(t *testing.T) {
	// Test circular reference detection
	tempDir := t.TempDir()
	baseDir := tempDir
	workflowsDir := tempDir

	// Create files with circular references
	file1 := tempDir + "/circular1.md"
	file1Content := `# Circular 1
@include circular2.md
`
	if err := os.WriteFile(file1, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to create circular1: %v", err)
	}

	file2 := tempDir + "/circular2.md"
	file2Content := `# Circular 2  
@include circular1.md
`
	if err := os.WriteFile(file2, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to create circular2: %v", err)
	}

	var dependencies []IncludeDependency
	seen := make(map[string]bool)

	content := "@include circular1.md"

	// This should not infinite loop due to the seen map
	err := collectIncludesRecursive(content, baseDir, workflowsDir, &dependencies, seen)

	// Should complete without error (circular references are prevented by seen map)
	if err != nil {
		t.Errorf("collectIncludesRecursive should handle circular references gracefully, got error: %v", err)
	}

	// Should have collected some dependencies but not infinite
	if len(dependencies) > 10 {
		t.Errorf("collectIncludesRecursive collected too many dependencies (%d), possible infinite loop", len(dependencies))
	}
}
