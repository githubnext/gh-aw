package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCompileWorkflowsWithCustomWorkflowDir tests the --workflow-dir flag functionality
func TestCompileWorkflowsWithCustomWorkflowDir(t *testing.T) {
	// Save current directory and defer restoration
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Create a temporary git repository with custom workflow directory
	tmpDir, err := os.MkdirTemp("", "workflow-dir-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repository properly
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Create custom workflow directory
	customDir := "my-workflows"
	if err := os.MkdirAll(customDir, 0755); err != nil {
		t.Fatalf("Failed to create custom workflow directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on: push
---

# Test Workflow

This is a test workflow in a custom directory.
`
	workflowFile := filepath.Join(customDir, "test.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Test 1: Compile with custom workflow directory should work
	err = CompileWorkflows([]string{}, false, "", false, false, customDir, false, false, false)
	if err != nil {
		t.Errorf("CompileWorkflows with custom workflow-dir should succeed, got error: %v", err)
	}

	// Verify the lock file was created
	lockFile := filepath.Join(customDir, "test.lock.yml")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("Expected lock file to be created in custom directory")
	}

	// Test 2: Using absolute path should fail
	err = CompileWorkflows([]string{}, false, "", false, false, "/absolute/path", false, false, false)
	if err == nil {
		t.Error("CompileWorkflows with absolute workflow-dir should fail")
	}
	if err != nil && err.Error() != "workflow-dir must be a relative path, got: /absolute/path" {
		t.Errorf("Expected specific error message for absolute path, got: %v", err)
	}

	// Test 3: Empty workflow-dir should default to .github/workflows
	// Create the default directory and a file
	defaultDir := ".github/workflows"
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatalf("Failed to create default workflow directory: %v", err)
	}
	defaultWorkflowFile := filepath.Join(defaultDir, "default.md")
	if err := os.WriteFile(defaultWorkflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create default workflow file: %v", err)
	}

	err = CompileWorkflows([]string{}, false, "", false, false, "", false, false, false)
	if err != nil {
		t.Errorf("CompileWorkflows with default workflow-dir should succeed, got error: %v", err)
	}

	// Verify the lock file was created in default location
	defaultLockFile := filepath.Join(defaultDir, "default.lock.yml")
	if _, err := os.Stat(defaultLockFile); os.IsNotExist(err) {
		t.Error("Expected lock file to be created in default directory")
	}
}

// TestCompileWorkflowsCustomDirValidation tests the validation of workflow directory paths
func TestCompileWorkflowsCustomDirValidation(t *testing.T) {
	tests := []struct {
		name        string
		workflowDir string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty string defaults to .github/workflows",
			workflowDir: "",
			expectError: false,
		},
		{
			name:        "relative path is valid",
			workflowDir: "custom/workflows",
			expectError: false,
		},
		{
			name:        "absolute path is invalid",
			workflowDir: "/absolute/path",
			expectError: true,
			errorMsg:    "workflow-dir must be a relative path, got: /absolute/path",
		},
		{
			name:        "path with .. is cleaned but valid",
			workflowDir: "workflows/../workflows",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for each test
			tmpDir, err := os.MkdirTemp("", "workflow-dir-validation-test")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}
			defer func() {
				_ = os.Chdir(originalWd)
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Initialize git repository properly
			cmd := exec.Command("git", "init")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to initialize git repository: %v", err)
			}

			// For non-error cases, create the expected directory
			if !tt.expectError {
				expectedDir := tt.workflowDir
				if expectedDir == "" {
					expectedDir = ".github/workflows"
				}
				if err := os.MkdirAll(expectedDir, 0755); err != nil {
					t.Fatalf("Failed to create workflow directory: %v", err)
				}
				// Create a dummy workflow file
				workflowFile := filepath.Join(expectedDir, "test.md")
				workflowContent := `---
on: push
---

# Test Workflow
`
				if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			}

			// Test the compilation
			err = CompileWorkflows([]string{}, false, "", false, false, tt.workflowDir, false, false, false)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for workflow-dir '%s', but got none", tt.workflowDir)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for workflow-dir '%s', but got: %v", tt.workflowDir, err)
				}
			}
		})
	}
}
