package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadWorkflowFileWithRelativePath tests that readWorkflowFile correctly handles relative paths on all platforms
func TestReadWorkflowFileWithRelativePath(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	testFile := filepath.Join(workflowsDir, "test-workflow.md")
	testContent := []byte("---\non: push\n---\n# Test Workflow")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test reading the workflow file with a relative path
	// This simulates what happens when getWorkflowsDir() returns a string and it's used with filepath.Join
	content, path, err := readWorkflowFile("test-workflow.md", workflowsDir)
	if err != nil {
		t.Errorf("readWorkflowFile failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
	}

	// The returned path should be the full path to the file
	expectedPath := testFile
	if path != expectedPath {
		t.Errorf("Path mismatch. Expected: %s, Got: %s", expectedPath, path)
	}
}

// TestReadWorkflowFileWithAbsolutePath tests that readWorkflowFile correctly handles absolute paths
func TestReadWorkflowFileWithAbsolutePath(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	testFile := filepath.Join(workflowsDir, "test-workflow.md")
	testContent := []byte("---\non: push\n---\n# Test Workflow")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test reading the workflow file with an absolute path
	// The workflowsDir parameter should be ignored when filePath is absolute
	content, path, err := readWorkflowFile(testFile, workflowsDir)
	if err != nil {
		t.Errorf("readWorkflowFile failed with absolute path: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
	}

	// The returned path should be the absolute path
	if path != testFile {
		t.Errorf("Path mismatch. Expected: %s, Got: %s", testFile, path)
	}
}

// TestReadWorkflowFilePathSeparators tests that the function works correctly regardless of path separator
func TestReadWorkflowFilePathSeparators(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file in a subdirectory
	subDir := filepath.Join(workflowsDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subfolder: %v", err)
	}

	testFile := filepath.Join(subDir, "test-workflow.md")
	testContent := []byte("---\non: push\n---\n# Test Workflow")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test reading with OS-appropriate path separators
	// filepath.Join will use the correct separator for the current OS
	relativePath := filepath.Join("subfolder", "test-workflow.md")
	content, path, err := readWorkflowFile(relativePath, workflowsDir)
	if err != nil {
		t.Errorf("readWorkflowFile failed with subdirectory path: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, content)
	}

	// The returned path should be the full path to the file
	if path != testFile {
		t.Errorf("Path mismatch. Expected: %s, Got: %s", testFile, path)
	}
}

// TestReadWorkflowFileNonExistent tests error handling for non-existent files
func TestReadWorkflowFileNonExistent(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Try to read a non-existent file
	_, _, err := readWorkflowFile("non-existent.md", workflowsDir)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
