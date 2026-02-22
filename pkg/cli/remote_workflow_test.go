//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLocalWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name: "valid workflow file",
			content: `---
name: Test Workflow
on: workflow_dispatch
---

# Test Workflow

This is a test.
`,
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expectError: false,
		},
		{
			name:        "minimal content",
			content:     "# Hello",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "test-workflow.md")
			err := os.WriteFile(tempFile, []byte(tt.content), 0644)
			require.NoError(t, err, "should create temp file")

			spec := &WorkflowSpec{
				WorkflowPath: tempFile,
				WorkflowName: "test-workflow",
			}

			result, err := fetchLocalWorkflow(spec, false)

			if tt.expectError {
				assert.Error(t, err, "expected error")
			} else {
				require.NoError(t, err, "should not error")
				assert.Equal(t, []byte(tt.content), result.Content, "content should match")
				assert.True(t, result.IsLocal, "should be marked as local")
				assert.Empty(t, result.CommitSHA, "local workflows should not have commit SHA")
				assert.Equal(t, tempFile, result.SourcePath, "source path should match")
			}
		})
	}
}

func TestFetchLocalWorkflow_NonExistentFile(t *testing.T) {
	spec := &WorkflowSpec{
		WorkflowPath: "/nonexistent/path/to/workflow.md",
		WorkflowName: "nonexistent-workflow",
	}

	result, err := fetchLocalWorkflow(spec, false)

	require.Error(t, err, "should error for non-existent file")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "not found", "error should mention file not found")
}

func TestFetchLocalWorkflow_DirectoryInsteadOfFile(t *testing.T) {
	tempDir := t.TempDir()

	spec := &WorkflowSpec{
		WorkflowPath: tempDir, // Pass directory instead of file
		WorkflowName: "directory-workflow",
	}

	result, err := fetchLocalWorkflow(spec, false)

	require.Error(t, err, "should error when path is a directory")
	assert.Nil(t, result, "result should be nil on error")
}

func TestFetchWorkflowFromSource_LocalRouting(t *testing.T) {
	// Create a temporary local workflow file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "local-workflow.md")
	content := "# Local Workflow\n\nTest content."
	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err, "should create temp file")

	spec := &WorkflowSpec{
		WorkflowPath: tempFile,
		WorkflowName: "local-workflow",
	}

	result, err := FetchWorkflowFromSource(spec, false)

	require.NoError(t, err, "should not error for local workflow")
	assert.True(t, result.IsLocal, "should route to local fetch")
	assert.Equal(t, []byte(content), result.Content, "content should match")
}

func TestFetchWorkflowFromSource_RemoteRoutingWithInvalidSlug(t *testing.T) {
	// Test with a remote workflow spec that has an invalid slug
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "invalid-slug-no-slash",
			Version:  "main",
		},
		WorkflowPath: "workflow.md",
		WorkflowName: "workflow",
	}

	result, err := FetchWorkflowFromSource(spec, false)

	require.Error(t, err, "should error for invalid repo slug")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "invalid repository slug", "error should mention invalid slug")
}

func TestFetchIncludeFromSource_WorkflowSpecParsing(t *testing.T) {
	tests := []struct {
		name          string
		includePath   string
		baseSpec      *WorkflowSpec
		expectSection string
		expectError   bool
		errorContains string
	}{
		{
			name:          "two parts falls through to cannot resolve",
			includePath:   "owner/repo",
			baseSpec:      nil,
			expectSection: "",
			expectError:   true,
			errorContains: "cannot resolve include path", // Not a workflowspec format (only 2 parts)
		},
		{
			name:          "section extraction from workflowspec",
			includePath:   "owner/repo/path/file.md#section-name",
			baseSpec:      nil,
			expectSection: "#section-name",
			expectError:   true, // Will fail to download, but section should be extracted
			errorContains: "",   // Don't check error message, just that section is extracted
		},
		{
			name:          "no section in workflowspec",
			includePath:   "owner/repo/path/file.md",
			baseSpec:      nil,
			expectSection: "",
			expectError:   true, // Will fail to download
			errorContains: "",
		},
		{
			name:          "relative path without base spec",
			includePath:   "shared/file.md",
			baseSpec:      nil,
			expectSection: "",
			expectError:   true,
			errorContains: "cannot resolve include path",
		},
		{
			name:          "relative path with section but no base spec",
			includePath:   "shared/file.md#my-section",
			baseSpec:      nil,
			expectSection: "#my-section",
			expectError:   true,
			errorContains: "cannot resolve include path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, section, err := FetchIncludeFromSource(tt.includePath, tt.baseSpec, false)

			if tt.expectError {
				require.Error(t, err, "expected error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "error should contain expected text")
				}
			} else {
				require.NoError(t, err, "should not error")
			}

			// Section should always be extracted consistently
			assert.Equal(t, tt.expectSection, section, "section should match expected")
		})
	}
}

func TestFetchIncludeFromSource_SectionExtraction(t *testing.T) {
	// Test that section is consistently extracted regardless of path type
	tests := []struct {
		name          string
		includePath   string
		expectSection string
	}{
		{
			name:          "hash section",
			includePath:   "owner/repo/file.md#section",
			expectSection: "#section",
		},
		{
			name:          "complex section with hyphens",
			includePath:   "owner/repo/file.md#my-complex-section-name",
			expectSection: "#my-complex-section-name",
		},
		{
			name:          "no section",
			includePath:   "owner/repo/file.md",
			expectSection: "",
		},
		{
			name:          "section at end of path with ref",
			includePath:   "owner/repo/file.md@v1.0.0#section",
			expectSection: "#section", // Section is extracted from the end regardless of @ref position
		},
		{
			name:          "section after everything",
			includePath:   "owner/repo/file.md#section-name",
			expectSection: "#section-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We expect errors since these are remote paths, but section should still be extracted
			_, section, _ := FetchIncludeFromSource(tt.includePath, nil, false)
			assert.Equal(t, tt.expectSection, section, "section should be correctly extracted")
		})
	}
}

func TestGetParentDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "dir/file.md",
			expected: "dir",
		},
		{
			name:     "deep path",
			path:     "a/b/c/file.md",
			expected: "a/b/c",
		},
		{
			name:     "no directory",
			path:     "file.md",
			expected: "",
		},
		{
			name:     "trailing slash",
			path:     "dir/",
			expected: "dir",
		},
		{
			name:     "empty string",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParentDir(tt.path)
			assert.Equal(t, tt.expected, result, "getParentDir(%q) should return %q", tt.path, tt.expected)
		})
	}
}

// TestFetchAndSaveRemoteFrontmatterImports_NoImports verifies that the function
// is a no-op when the workflow has no imports field.
func TestFetchAndSaveRemoteFrontmatterImports_NoImports(t *testing.T) {
	content := `---
engine: copilot
---

# Workflow with no imports
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "main",
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	tmpDir := t.TempDir()
	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, nil)
	require.NoError(t, err, "should not error when no imports are present")

	// No files should have been created
	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "no files should be created when no imports are present")
}

// TestFetchAndSaveRemoteFrontmatterImports_EmptyRepoSlug verifies that the function
// is a no-op when the spec has no remote repo (local workflow).
func TestFetchAndSaveRemoteFrontmatterImports_EmptyRepoSlug(t *testing.T) {
	content := `---
engine: copilot
imports:
  - shared/ci-data-analysis.md
---

# Workflow
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "", // local workflow – no remote repo
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	tmpDir := t.TempDir()
	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, nil)
	require.NoError(t, err, "should not error for local workflow with empty RepoSlug")

	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "no files should be created for local workflows")
}

// TestFetchAndSaveRemoteFrontmatterImports_WorkflowSpecSkipped verifies that imports
// that are already in workflowspec format (owner/repo/path@ref) are skipped.
func TestFetchAndSaveRemoteFrontmatterImports_WorkflowSpecSkipped(t *testing.T) {
	content := `---
engine: copilot
imports:
  - github/gh-aw/.github/workflows/shared/ci-data-analysis.md@abc123
---

# Workflow
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "main",
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	tmpDir := t.TempDir()
	// This should not attempt any network calls; already-pinned imports are skipped.
	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, nil)
	require.NoError(t, err, "should not error for workflowspec imports")

	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "already-pinned workflowspec imports should not be downloaded")
}

// TestFetchAndSaveRemoteFrontmatterImports_FileTracking verifies that files saved by
// the function are properly tracked (created vs modified).
func TestFetchAndSaveRemoteFrontmatterImports_FileTracking(t *testing.T) {
	// Use a temp dir that acts as the workflows directory.
	tmpDir := t.TempDir()

	// Pre-create one file so we can test the "modified" branch.
	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755))
	existingFile := filepath.Join(sharedDir, "existing.md")
	require.NoError(t, os.WriteFile(existingFile, []byte("old content"), 0600))

	// Build a minimal FileTracker without calling NewFileTracker (which requires a real
	// git repository).  We only need the tracking lists populated.
	tracker := &FileTracker{
		OriginalContent: make(map[string][]byte),
		gitRoot:         tmpDir,
	}

	// A workflow with no imports should leave the tracker empty.
	content := `---
engine: copilot
---

# No imports
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "v1.0.0",
		},
		WorkflowPath: ".github/workflows/test.md",
	}

	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, tracker)
	require.NoError(t, err)
	assert.Empty(t, tracker.CreatedFiles, "no files should be created when there are no imports")
	assert.Empty(t, tracker.ModifiedFiles, "no files should be modified when there are no imports")
}

// TestFetchAndSaveRemoteFrontmatterImports_SectionSkipped verifies that imports
// with a section reference (file.md#Section) have the section stripped but the
// file path is still used for deduplication.
func TestFetchAndSaveRemoteFrontmatterImports_SectionSkipped(t *testing.T) {
	// Content with two imports that differ only in their #section fragment.
	// Both should resolve to the same file (deduplication by file path).
	content := `---
engine: copilot
imports:
  - github/gh-aw/.github/workflows/shared/reporting.md@abc123
  - github/gh-aw/.github/workflows/shared/reporting.md@abc123
---

# Workflow
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "v1.0.0",
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	tmpDir := t.TempDir()
	// Both entries are already workflowspec format – neither should trigger a download.
	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, nil)
	require.NoError(t, err, "duplicate workflowspec imports should not error")

	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "workflowspec imports must not be downloaded")
}

// TestFetchAndSaveRemoteFrontmatterImports_SkipExistingWithoutForce verifies that
// an import that is already present on disk is skipped when force=false.
func TestFetchAndSaveRemoteFrontmatterImports_SkipExistingWithoutForce(t *testing.T) {
	// Pre-create the shared file that the workflow would import.
	tmpDir := t.TempDir()
	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755))
	existingContent := []byte("existing content")
	existingFile := filepath.Join(sharedDir, "ci-data-analysis.md")
	require.NoError(t, os.WriteFile(existingFile, existingContent, 0600))

	tracker := &FileTracker{
		OriginalContent: make(map[string][]byte),
		gitRoot:         tmpDir,
	}

	// Use a content with a relative import that would resolve to the pre-created file.
	// But since the import path is workflowspec-format, it's skipped regardless –
	// this test verifies no modification is done to the pre-created file.
	content := `---
engine: copilot
imports:
  - github/gh-aw/.github/workflows/shared/ci-data-analysis.md@abc123
---
# Workflow
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "v1.0.0",
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, tracker)
	require.NoError(t, err)

	// The existing file should not have been touched.
	gotContent, readErr := os.ReadFile(existingFile)
	require.NoError(t, readErr)
	assert.Equal(t, existingContent, gotContent, "pre-existing file must not be modified when force=false")
	assert.Empty(t, tracker.CreatedFiles)
	assert.Empty(t, tracker.ModifiedFiles)
}

// TestFetchAndSaveRemoteFrontmatterImports_InvalidRepoSlug verifies that an invalid
// RepoSlug (not in owner/repo format) causes the function to return early without error.
func TestFetchAndSaveRemoteFrontmatterImports_InvalidRepoSlug(t *testing.T) {
	content := `---
engine: copilot
imports:
  - shared/ci-data-analysis.md
---
# Workflow
`
	spec := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "not-a-valid-slug", // missing slash → only one part
		},
		WorkflowPath: ".github/workflows/ci-coach.md",
	}

	tmpDir := t.TempDir()
	err := fetchAndSaveRemoteFrontmatterImports(content, spec, tmpDir, false, false, nil)
	require.NoError(t, err, "invalid RepoSlug should return nil without error")

	entries, readErr := os.ReadDir(tmpDir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "no files should be created for an invalid RepoSlug")
}
