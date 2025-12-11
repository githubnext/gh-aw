package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeSourceHash(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Test with known content
	content := "---\nname: test\n---\nThis is a test workflow"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compute hash
	hash1, err := ComputeSourceHash(testFile)
	if err != nil {
		t.Fatalf("ComputeSourceHash failed: %v", err)
	}

	// Verify hash format
	if !strings.HasPrefix(hash1, "sha256:") {
		t.Errorf("Hash should start with 'sha256:', got: %s", hash1)
	}

	// Compute hash again - should be identical
	hash2, err := ComputeSourceHash(testFile)
	if err != nil {
		t.Fatalf("ComputeSourceHash failed on second call: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic, got different values: %s vs %s", hash1, hash2)
	}

	// Change content and verify hash changes
	newContent := "---\nname: test\n---\nThis is a modified test workflow"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write modified test file: %v", err)
	}

	hash3, err := ComputeSourceHash(testFile)
	if err != nil {
		t.Fatalf("ComputeSourceHash failed after modification: %v", err)
	}

	if hash1 == hash3 {
		t.Errorf("Hash should change when content changes, both are: %s", hash1)
	}
}

func TestComputeSourceHash_NonExistentFile(t *testing.T) {
	_, err := ComputeSourceHash("/nonexistent/file.md")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestExtractLockFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		lockFileContent string
		expectedHash    string
		expectedVersion string
		shouldBeNil     bool
		expectError     bool
	}{
		{
			name: "valid metadata block",
			lockFileContent: `#
# ASCII Logo
#
# Metadata:
#   source_hash: sha256:abc123def456
#   compiled_at: 2025-12-11T13:17:10Z
#   gh_aw_version: v0.0.367
#   dependencies_hash: sha256:dep789
#
# This file was automatically generated
name: test-workflow`,
			expectedHash:    "sha256:abc123def456",
			expectedVersion: "v0.0.367",
			shouldBeNil:     false,
			expectError:     false,
		},
		{
			name: "no metadata block",
			lockFileContent: `#
# ASCII Logo
#
# This file was automatically generated
name: test-workflow`,
			shouldBeNil: true,
			expectError: false,
		},
		{
			name: "partial metadata block",
			lockFileContent: `#
# Metadata:
#   source_hash: sha256:abc123def456
#
# This file was automatically generated
name: test-workflow`,
			expectedHash: "sha256:abc123def456",
			shouldBeNil:  false,
			expectError:  false,
		},
		{
			name: "metadata with extra whitespace",
			lockFileContent: `#
# Metadata:
#   source_hash:    sha256:abc123def456   
#   compiled_at:  2025-12-11T13:17:10Z  
#   gh_aw_version:   v0.0.367   
#
name: test-workflow`,
			expectedHash:    "sha256:abc123def456",
			expectedVersion: "v0.0.367",
			shouldBeNil:     false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockFile := filepath.Join(tmpDir, "test.lock.yml")
			if err := os.WriteFile(lockFile, []byte(tt.lockFileContent), 0644); err != nil {
				t.Fatalf("Failed to write test lock file: %v", err)
			}

			metadata, err := ExtractLockFileMetadata(lockFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.shouldBeNil {
				if metadata != nil {
					t.Errorf("Expected nil metadata, got: %+v", metadata)
				}
				return
			}

			if metadata == nil {
				t.Fatal("Expected non-nil metadata, got nil")
			}

			if tt.expectedHash != "" && metadata.SourceHash != tt.expectedHash {
				t.Errorf("Expected source_hash %s, got %s", tt.expectedHash, metadata.SourceHash)
			}

			if tt.expectedVersion != "" && metadata.GhAwVersion != tt.expectedVersion {
				t.Errorf("Expected gh_aw_version %s, got %s", tt.expectedVersion, metadata.GhAwVersion)
			}
		})
	}
}

func TestExtractLockFileMetadata_NonExistentFile(t *testing.T) {
	_, err := ExtractLockFileMetadata("/nonexistent/file.lock.yml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestCompareLockFileToSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "test.md")
	sourceContent := "---\nname: test\n---\nTest workflow"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compute hash for testing
	sourceHash, err := ComputeSourceHash(sourceFile)
	if err != nil {
		t.Fatalf("Failed to compute source hash: %v", err)
	}

	tests := []struct {
		name            string
		setupLockFile   bool
		lockFileContent string
		expectedStatus  StaleStatus
		expectError     bool
	}{
		{
			name:           "lock file doesn't exist",
			setupLockFile:  false,
			expectedStatus: StaleStatusNeverCompiled,
			expectError:    false,
		},
		{
			name:          "hash matches",
			setupLockFile: true,
			lockFileContent: `#
# Metadata:
#   source_hash: ` + sourceHash + `
#   compiled_at: 2025-12-11T13:17:10Z
#   gh_aw_version: v0.0.367
#
name: test-workflow`,
			expectedStatus: StaleStatusUpToDate,
			expectError:    false,
		},
		{
			name:          "hash mismatch",
			setupLockFile: true,
			lockFileContent: `#
# Metadata:
#   source_hash: sha256:differenthash123
#   compiled_at: 2025-12-11T13:17:10Z
#   gh_aw_version: v0.0.367
#
name: test-workflow`,
			expectedStatus: StaleStatusHashMismatch,
			expectError:    false,
		},
		{
			name:          "no metadata - fallback to timestamp",
			setupLockFile: true,
			lockFileContent: `#
# This file was automatically generated
name: test-workflow`,
			expectedStatus: StaleStatusUpToDate,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockFile := filepath.Join(tmpDir, "test-"+strings.ReplaceAll(tt.name, " ", "-")+".lock.yml")

			if tt.setupLockFile {
				if err := os.WriteFile(lockFile, []byte(tt.lockFileContent), 0644); err != nil {
					t.Fatalf("Failed to write lock file: %v", err)
				}
			}

			status, err := CompareLockFileToSource(lockFile, sourceFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}
		})
	}
}

func TestGenerateMetadataComment(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test source file
	sourceFile := filepath.Join(tmpDir, "test.md")
	sourceContent := "---\nname: test\n---\nTest workflow"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	tests := []struct {
		name         string
		version      string
		dependencies []string
		wantContains []string
	}{
		{
			name:         "basic metadata",
			version:      "v0.0.367",
			dependencies: []string{},
			wantContains: []string{
				"# Metadata:",
				"#   source_hash: sha256:",
				"#   compiled_at: ",
				"#   gh_aw_version: v0.0.367",
				"#   dependencies_hash: sha256:none",
			},
		},
		{
			name:    "with dependencies",
			version: "v0.0.368",
			dependencies: []string{
				"workflow1.md",
				"workflow2.md",
			},
			wantContains: []string{
				"# Metadata:",
				"#   source_hash: sha256:",
				"#   compiled_at: ",
				"#   gh_aw_version: v0.0.368",
				"#   dependencies_hash: sha256:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment, err := GenerateMetadataComment(sourceFile, tt.version, tt.dependencies)
			if err != nil {
				t.Fatalf("GenerateMetadataComment failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(comment, want) {
					t.Errorf("Expected metadata to contain %q, got:\n%s", want, comment)
				}
			}

			// Verify it's properly formatted as comments
			lines := strings.Split(comment, "\n")
			for i, line := range lines {
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "#") {
					t.Errorf("Line %d should start with '#', got: %s", i+1, line)
				}
			}
		})
	}
}

func TestGenerateMetadataComment_NonExistentFile(t *testing.T) {
	_, err := GenerateMetadataComment("/nonexistent/file.md", "v0.0.367", nil)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestComputeDependenciesHash(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []string
		wantPrefix   string
	}{
		{
			name:         "empty dependencies",
			dependencies: []string{},
			wantPrefix:   "sha256:none",
		},
		{
			name:         "single dependency",
			dependencies: []string{"workflow1.md"},
			wantPrefix:   "sha256:",
		},
		{
			name:         "multiple dependencies",
			dependencies: []string{"workflow1.md", "workflow2.md"},
			wantPrefix:   "sha256:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := computeDependenciesHash(tt.dependencies)

			if !strings.HasPrefix(hash, tt.wantPrefix) {
				t.Errorf("Expected hash to start with %s, got: %s", tt.wantPrefix, hash)
			}

			// Verify determinism
			hash2 := computeDependenciesHash(tt.dependencies)
			if hash != hash2 {
				t.Errorf("Hash should be deterministic, got different values: %s vs %s", hash, hash2)
			}
		})
	}

	// Verify different dependencies produce different hashes
	hash1 := computeDependenciesHash([]string{"a.md"})
	hash2 := computeDependenciesHash([]string{"b.md"})
	if hash1 == hash2 {
		t.Error("Different dependencies should produce different hashes")
	}
}
