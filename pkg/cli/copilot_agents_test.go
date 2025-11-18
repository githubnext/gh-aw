package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestEnsureFileMatchesTemplate tests the common helper function
func TestEnsureFileMatchesTemplate(t *testing.T) {
	tests := []struct {
		name             string
		subdir           string
		fileName         string
		templateContent  string
		fileType         string
		existingContent  string
		skipInstructions bool
		expectedFile     bool
		expectedContent  string
	}{
		{
			name:             "creates new file",
			subdir:           ".github/test",
			fileName:         "test.md",
			templateContent:  "# Test Template",
			fileType:         "test file",
			existingContent:  "",
			skipInstructions: false,
			expectedFile:     true,
			expectedContent:  "# Test Template",
		},
		{
			name:             "does not modify existing correct file",
			subdir:           ".github/test",
			fileName:         "test.md",
			templateContent:  "# Test Template",
			fileType:         "test file",
			existingContent:  "# Test Template",
			skipInstructions: false,
			expectedFile:     true,
			expectedContent:  "# Test Template",
		},
		{
			name:             "updates modified file",
			subdir:           ".github/test",
			fileName:         "test.md",
			templateContent:  "# Test Template",
			fileType:         "test file",
			existingContent:  "# Old Content",
			skipInstructions: false,
			expectedFile:     true,
			expectedContent:  "# Test Template",
		},
		{
			name:             "skips when skipInstructions is true",
			subdir:           ".github/test",
			fileName:         "test.md",
			templateContent:  "# Test Template",
			fileType:         "test file",
			existingContent:  "",
			skipInstructions: true,
			expectedFile:     false,
			expectedContent:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := testutil.TempDir(t, "test-*")

			// Change to temp directory and initialize git repo
			oldWd, _ := os.Getwd()
			defer func() {
				_ = os.Chdir(oldWd)
			}()
			err := os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Initialize git repo
			if err := exec.Command("git", "init").Run(); err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}

			targetDir := filepath.Join(tempDir, tt.subdir)
			targetPath := filepath.Join(targetDir, tt.fileName)

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
				if err := os.WriteFile(targetPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial file: %v", err)
				}
			}

			// Call the helper function
			err = ensureFileMatchesTemplate(tt.subdir, tt.fileName, tt.templateContent, tt.fileType, false, tt.skipInstructions)
			if err != nil {
				t.Fatalf("ensureFileMatchesTemplate() returned error: %v", err)
			}

			// Check file existence
			_, statErr := os.Stat(targetPath)
			if tt.expectedFile && os.IsNotExist(statErr) {
				t.Fatalf("Expected file to exist but it doesn't: %s", targetPath)
			}
			if !tt.expectedFile && !os.IsNotExist(statErr) {
				t.Fatalf("Expected file to not exist but it does: %s", targetPath)
			}

			// Check content if file should exist
			if tt.expectedFile {
				content, err := os.ReadFile(targetPath)
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}

				contentStr := strings.TrimSpace(string(content))
				expectedStr := strings.TrimSpace(tt.expectedContent)

				if contentStr != expectedStr {
					t.Errorf("Expected content does not match.\nExpected: %q\nActual: %q", expectedStr, contentStr)
				}
			}
		})
	}
}

// TestEnsureFileMatchesTemplate_VerboseOutput tests verbose logging
func TestEnsureFileMatchesTemplate_VerboseOutput(t *testing.T) {
	tests := []struct {
		name            string
		existingContent string
		expectedLog     string
	}{
		{
			name:            "logs creation",
			existingContent: "",
			expectedLog:     "Created",
		},
		{
			name:            "logs update",
			existingContent: "# Old Content",
			expectedLog:     "Updated",
		},
		{
			name:            "logs up-to-date",
			existingContent: "# Test Template",
			expectedLog:     "up-to-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := testutil.TempDir(t, "test-*")

			// Change to temp directory and initialize git repo
			oldWd, _ := os.Getwd()
			defer func() {
				_ = os.Chdir(oldWd)
			}()
			err := os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Initialize git repo
			if err := exec.Command("git", "init").Run(); err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}

			subdir := ".github/test"
			fileName := "test.md"
			targetDir := filepath.Join(tempDir, subdir)
			targetPath := filepath.Join(targetDir, fileName)

			// Create initial content if specified
			if tt.existingContent != "" {
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
				if err := os.WriteFile(targetPath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create initial file: %v", err)
				}
			}

			// Call the helper function with verbose=true
			// Note: This test doesn't capture stdout, it just verifies no errors occur
			err = ensureFileMatchesTemplate(subdir, fileName, "# Test Template", "test file", true, false)
			if err != nil {
				t.Fatalf("ensureFileMatchesTemplate() returned error: %v", err)
			}

			// Verify file exists
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				t.Fatalf("Expected file to exist")
			}
		})
	}
}
