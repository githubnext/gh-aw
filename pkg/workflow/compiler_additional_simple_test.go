package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompiler_SetFileTracker_Basic(t *testing.T) {
	// Create compiler
	compiler := NewCompiler(false, "", "test-version")

	// Initial state should have nil tracker
	if compiler.fileTracker != nil {
		t.Errorf("Expected initial fileTracker to be nil")
	}

	// Create mock tracker
	mockTracker := &SimpleBasicMockFileTracker{}

	// Set tracker
	compiler.SetFileTracker(mockTracker)

	// Verify tracker was set
	if compiler.fileTracker != mockTracker {
		t.Errorf("Expected tracker to be set")
	}

	// Set to nil
	compiler.SetFileTracker(nil)

	// Verify tracker is nil
	if compiler.fileTracker != nil {
		t.Errorf("Expected tracker to be nil after setting to nil")
	}
}

func TestCompiler_WriteReactionAction_Basic(t *testing.T) {
	// Create compiler
	compiler := NewCompiler(false, "", "test-version")

	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create a test markdown file path (doesn't need to actually exist)
	markdownPath := filepath.Join(tmpDir, "test.md")

	// Set up file tracker to verify file creation
	mockTracker := &SimpleBasicMockFileTracker{}
	compiler.SetFileTracker(mockTracker)

	// Test that writeReactionAction succeeds
	err := compiler.writeReactionAction(markdownPath)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify that the action file was created
	expectedActionFile := filepath.Join(tmpDir, ".github", "actions", "reaction", "action.yml")
	if _, err := os.Stat(expectedActionFile); os.IsNotExist(err) {
		t.Errorf("Expected action file to be created at: %s", expectedActionFile)
	}

	// Verify that file tracker was called
	if len(mockTracker.tracked) != 1 {
		t.Errorf("Expected file tracker to track 1 file, got %d", len(mockTracker.tracked))
	}

	if len(mockTracker.tracked) > 0 && mockTracker.tracked[0] != expectedActionFile {
		t.Errorf("Expected tracker to track %s, got %s", expectedActionFile, mockTracker.tracked[0])
	}
}

// SimpleBasicMockFileTracker is a basic implementation for testing
type SimpleBasicMockFileTracker struct {
	tracked []string
}

func (s *SimpleBasicMockFileTracker) TrackCreated(filePath string) {
	if s.tracked == nil {
		s.tracked = make([]string, 0)
	}
	s.tracked = append(s.tracked, filePath)
}
