package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for artifact unfold rule implementation
// Unfold rule: If an artifact download folder contains a single file, move the file to root and delete the folder

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

func TestFlattenSingleFileArtifactsWithAuditFiles(t *testing.T) {
	// Test that flattening works correctly for typical audit artifact files
	tmpDir := t.TempDir()

	// Create artifact structure as it would be downloaded by gh run download
	artifacts := map[string]string{
		"aw_info/aw_info.json":             `{"engine_id":"claude","workflow_name":"test"}`,
		"safe_output/safe_output.jsonl":    `{"action":"create_issue","title":"test"}`,
		"aw-patch/aw.patch":                "diff --git a/test.txt b/test.txt\n",
		"agent_outputs/output1.txt":        "log output 1",
		"agent_outputs/output2.txt":        "log output 2",
		"agent_outputs/nested/subfile.txt": "nested file",
	}

	for path, content := range artifacts {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Run flattening
	if err := flattenSingleFileArtifacts(tmpDir, true); err != nil {
		t.Fatalf("flattenSingleFileArtifacts failed: %v", err)
	}

	// Verify single-file artifacts are flattened and findable by audit command
	auditExpectedFiles := []string{
		"aw_info.json",
		"safe_output.jsonl",
		"aw.patch",
	}

	for _, file := range auditExpectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Audit expected file %s not found at top level after flattening", file)
		} else {
			// Verify file content is intact
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read flattened file %s: %v", file, err)
			} else if len(content) == 0 {
				t.Errorf("Flattened file %s is empty", file)
			}
		}
	}

	// Verify multi-file artifact directory is preserved
	agentOutputsDir := filepath.Join(tmpDir, "agent_outputs")
	if info, err := os.Stat(agentOutputsDir); os.IsNotExist(err) {
		t.Error("agent_outputs directory should be preserved")
	} else if !info.IsDir() {
		t.Error("agent_outputs should be a directory")
	}

	// Verify files within multi-file artifact are intact
	multiFileArtifactFiles := []string{
		"agent_outputs/output1.txt",
		"agent_outputs/output2.txt",
		"agent_outputs/nested/subfile.txt",
	}

	for _, file := range multiFileArtifactFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Multi-file artifact file %s should be preserved", file)
		}
	}

	// Verify original artifact directories are removed
	removedDirs := []string{"aw_info", "safe_output", "aw-patch"}
	for _, dir := range removedDirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("Single-file artifact directory %s should be removed after flattening", dir)
		}
	}
}

func TestAuditCanFindFlattenedArtifacts(t *testing.T) {
	// Simulate what the audit command does - check that it can find artifacts after flattening
	tmpDir := t.TempDir()

	// Create realistic artifact structure before flattening
	testArtifacts := map[string]string{
		"aw_info/aw_info.json":          `{"engine_id":"claude","workflow_name":"github-mcp-tools-report","run_id":123456}`,
		"safe_output/safe_output.jsonl": `{"action":"create_discussion","title":"GitHub MCP Tools Report"}`,
		"aw-patch/aw.patch":             "diff --git a/report.md b/report.md\nnew file mode 100644\n",
	}

	for path, content := range testArtifacts {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	// Flatten artifacts (this happens during download)
	if err := flattenSingleFileArtifacts(tmpDir, false); err != nil {
		t.Fatalf("Flattening failed: %v", err)
	}

	// Simulate what generateAuditReport does - check for artifacts using filepath.Join(run.LogsPath, filename)
	artifactsToCheck := []struct {
		filename    string
		description string
	}{
		{"aw_info.json", "engine configuration"},
		{"safe_output.jsonl", "safe outputs"},
		{"aw.patch", "code changes"},
	}

	foundArtifacts := []string{}
	for _, artifact := range artifactsToCheck {
		artifactPath := filepath.Join(tmpDir, artifact.filename)
		if _, err := os.Stat(artifactPath); err == nil {
			foundArtifacts = append(foundArtifacts, fmt.Sprintf("%s (%s)", artifact.filename, artifact.description))
		}
	}

	// Verify all expected artifacts were found
	if len(foundArtifacts) != len(artifactsToCheck) {
		t.Errorf("Expected to find %d artifacts, but found %d", len(artifactsToCheck), len(foundArtifacts))
		t.Logf("Found artifacts: %v", foundArtifacts)
	}

	// Verify we can read aw_info.json directly (simulates parseAwInfo)
	awInfoPath := filepath.Join(tmpDir, "aw_info.json")
	data, err := os.ReadFile(awInfoPath)
	if err != nil {
		t.Fatalf("Failed to read aw_info.json after flattening: %v", err)
	}

	// Verify content is valid
	if !strings.Contains(string(data), "engine_id") {
		t.Error("aw_info.json content is corrupted after flattening")
	}
}
