package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFileTracker_CreationAndTracking(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file-tracker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository in temp directory
	gitCmd := []string{"git", "init"}
	if err := runCommandInDir(gitCmd, tempDir); err != nil {
		t.Skipf("Skipping test - git not available or failed to init: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create file tracker
	tracker, err := NewFileTracker()
	if err != nil {
		t.Fatalf("Failed to create file tracker: %v", err)
	}

	// Create test files
	testFile1 := filepath.Join(tempDir, "test1.md")
	testFile2 := filepath.Join(tempDir, "test2.lock.yml")

	// Create first file and track it
	content1 := "# Test Workflow 1"
	if err := os.WriteFile(testFile1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file 1: %v", err)
	}
	tracker.TrackCreated(testFile1)

	// Create second file and track it
	content2 := "name: test-workflow"
	if err := os.WriteFile(testFile2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write test file 2: %v", err)
	}
	tracker.TrackCreated(testFile2)

	// Verify tracking
	allFiles := tracker.GetAllFiles()
	if len(allFiles) != 2 {
		t.Errorf("Expected 2 tracked files, got %d", len(allFiles))
	}

	// Test staging files
	if err := tracker.StageAllFiles(false); err != nil {
		t.Errorf("Failed to stage files: %v", err)
	}

	// Test rollback
	if err := tracker.RollbackCreatedFiles(false); err != nil {
		t.Errorf("Failed to rollback files: %v", err)
	}

	// Verify files were deleted
	if _, err := os.Stat(testFile1); !os.IsNotExist(err) {
		t.Errorf("File %s should have been deleted during rollback", testFile1)
	}
	if _, err := os.Stat(testFile2); !os.IsNotExist(err) {
		t.Errorf("File %s should have been deleted during rollback", testFile2)
	}
}

func TestFileTracker_ModifiedFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file-tracker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository in temp directory
	gitCmd := []string{"git", "init"}
	if err := runCommandInDir(gitCmd, tempDir); err != nil {
		t.Skipf("Skipping test - git not available or failed to init: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create file tracker
	tracker, err := NewFileTracker()
	if err != nil {
		t.Fatalf("Failed to create file tracker: %v", err)
	}

	// Create existing file
	testFile := filepath.Join(tempDir, "existing.md")
	originalContent := "# Original Content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Track modification BEFORE modifying the file
	tracker.TrackModified(testFile)

	// Now modify the file
	modifiedContent := "# Modified Content"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Verify tracking
	if len(tracker.CreatedFiles) != 0 {
		t.Errorf("Expected 0 created files, got %d", len(tracker.CreatedFiles))
	}
	if len(tracker.ModifiedFiles) != 1 {
		t.Errorf("Expected 1 modified file, got %d", len(tracker.ModifiedFiles))
	}

	// Test staging files
	if err := tracker.StageAllFiles(false); err != nil {
		t.Errorf("Failed to stage files: %v", err)
	}

	// Rollback should not delete modified files (only created ones)
	if err := tracker.RollbackCreatedFiles(false); err != nil {
		t.Errorf("Failed to rollback files: %v", err)
	}

	// Verify file still exists (not deleted since it was modified, not created)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Modified file %s should not have been deleted during rollback", testFile)
	}

	// Verify file content is still modified
	currentContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}
	if string(currentContent) != modifiedContent {
		t.Errorf("File content should still be modified, got %q", string(currentContent))
	}

	// Test rollback of modified files
	if err := tracker.RollbackModifiedFiles(false); err != nil {
		t.Errorf("Failed to rollback modified files: %v", err)
	}

	// Verify file content was restored to original
	restoredContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file after modified rollback: %v", err)
	}
	if string(restoredContent) != originalContent {
		t.Errorf("File content should be restored to original %q, got %q", originalContent, string(restoredContent))
	}
}

// Helper function to run commands in a specific directory
func runCommandInDir(cmd []string, dir string) error {
	if len(cmd) == 0 {
		return nil
	}
	command := cmd[0]
	args := cmd[1:]

	c := exec.Command(command, args...)
	c.Dir = dir
	return c.Run()
}

func TestFileTracker_RollbackAllFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "file-tracker-rollback-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository in temp directory
	gitCmd := []string{"git", "init"}
	if err := runCommandInDir(gitCmd, tempDir); err != nil {
		t.Skipf("Skipping test - git not available or failed to init: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create file tracker
	tracker, err := NewFileTracker()
	if err != nil {
		t.Fatalf("Failed to create file tracker: %v", err)
	}

	// Create an existing file
	existingFile := filepath.Join(tempDir, "existing.md")
	originalContent := "# Original Content"
	if err := os.WriteFile(existingFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to write existing file: %v", err)
	}

	// Track modification before modifying
	tracker.TrackModified(existingFile)
	modifiedContent := "# Modified Content"
	if err := os.WriteFile(existingFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify existing file: %v", err)
	}

	// Create a new file
	newFile := filepath.Join(tempDir, "new.md")
	tracker.TrackCreated(newFile)
	if err := os.WriteFile(newFile, []byte("# New Content"), 0644); err != nil {
		t.Fatalf("Failed to write new file: %v", err)
	}

	// Rollback all files
	if err := tracker.RollbackAllFiles(false); err != nil {
		t.Errorf("Failed to rollback all files: %v", err)
	}

	// Verify new file was deleted
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Errorf("New file %s should have been deleted", newFile)
	}

	// Verify existing file was restored to original content
	currentContent, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read existing file after rollback: %v", err)
	}
	if string(currentContent) != originalContent {
		t.Errorf("Existing file should be restored to original %q, got %q", originalContent, string(currentContent))
	}
}
