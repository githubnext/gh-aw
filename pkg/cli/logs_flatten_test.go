package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFlattenSingleFileArtifacts(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(string) error
		expectedFiles   []string
		expectedDirs    []string
		unexpectedFiles []string
		unexpectedDirs  []string
	}{
		{
			name: "single file artifact gets flattened",
			setup: func(dir string) error {
				artifactDir := filepath.Join(dir, "my-artifact")
				if err := os.MkdirAll(artifactDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(artifactDir, "output.json"), []byte("test"), 0644)
			},
			expectedFiles:   []string{"output.json"},
			unexpectedDirs:  []string{"my-artifact"},
			unexpectedFiles: []string{"my-artifact/output.json"},
		},
		{
			name: "multi-file artifact not flattened",
			setup: func(dir string) error {
				artifactDir := filepath.Join(dir, "multi-artifact")
				if err := os.MkdirAll(artifactDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(artifactDir, "file1.txt"), []byte("test1"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(artifactDir, "file2.txt"), []byte("test2"), 0644)
			},
			expectedDirs:    []string{"multi-artifact"},
			expectedFiles:   []string{"multi-artifact/file1.txt", "multi-artifact/file2.txt"},
			unexpectedFiles: []string{"file1.txt", "file2.txt"},
		},
		{
			name: "artifact with subdirectory not flattened",
			setup: func(dir string) error {
				artifactDir := filepath.Join(dir, "nested-artifact")
				if err := os.MkdirAll(filepath.Join(artifactDir, "subdir"), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(artifactDir, "file.txt"), []byte("test"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(artifactDir, "subdir", "nested.txt"), []byte("test"), 0644)
			},
			expectedDirs:    []string{"nested-artifact"},
			expectedFiles:   []string{"nested-artifact/file.txt", "nested-artifact/subdir/nested.txt"},
			unexpectedFiles: []string{"file.txt"},
		},
		{
			name: "multiple single-file artifacts all get flattened",
			setup: func(dir string) error {
				for i := 1; i <= 3; i++ {
					artifactDir := filepath.Join(dir, fmt.Sprintf("artifact-%d", i))
					if err := os.MkdirAll(artifactDir, 0755); err != nil {
						return err
					}
					if err := os.WriteFile(filepath.Join(artifactDir, fmt.Sprintf("file%d.txt", i)), []byte("test"), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles:  []string{"file1.txt", "file2.txt", "file3.txt"},
			unexpectedDirs: []string{"artifact-1", "artifact-2", "artifact-3"},
		},
		{
			name: "empty artifact directory not touched",
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, "empty-artifact"), 0755)
			},
			expectedDirs: []string{"empty-artifact"},
		},
		{
			name: "regular files in output dir not affected",
			setup: func(dir string) error {
				// Create a regular file in output dir
				if err := os.WriteFile(filepath.Join(dir, "standalone.txt"), []byte("test"), 0644); err != nil {
					return err
				}
				// Create a single-file artifact
				artifactDir := filepath.Join(dir, "single-artifact")
				if err := os.MkdirAll(artifactDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(artifactDir, "artifact.json"), []byte("test"), 0644)
			},
			expectedFiles:  []string{"standalone.txt", "artifact.json"},
			unexpectedDirs: []string{"single-artifact"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Setup test structure
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Run flatten function
			if err := flattenSingleFileArtifacts(tmpDir, true); err != nil {
				t.Fatalf("flattenSingleFileArtifacts failed: %v", err)
			}

			// Verify expected files exist
			for _, file := range tt.expectedFiles {
				path := filepath.Join(tmpDir, file)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected file %s does not exist", file)
				}
			}

			// Verify expected directories exist
			for _, dir := range tt.expectedDirs {
				path := filepath.Join(tmpDir, dir)
				info, err := os.Stat(path)
				if os.IsNotExist(err) {
					t.Errorf("Expected directory %s does not exist", dir)
				} else if err == nil && !info.IsDir() {
					t.Errorf("Expected %s to be a directory", dir)
				}
			}

			// Verify unexpected files don't exist
			for _, file := range tt.unexpectedFiles {
				path := filepath.Join(tmpDir, file)
				if _, err := os.Stat(path); err == nil {
					t.Errorf("Unexpected file %s exists", file)
				}
			}

			// Verify unexpected directories don't exist
			for _, dir := range tt.unexpectedDirs {
				path := filepath.Join(tmpDir, dir)
				if _, err := os.Stat(path); err == nil {
					t.Errorf("Unexpected directory %s exists", dir)
				}
			}
		})
	}
}

func TestFlattenSingleFileArtifactsInvalidDirectory(t *testing.T) {
	// Test with non-existent directory
	err := flattenSingleFileArtifacts("/nonexistent/directory", false)
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}
