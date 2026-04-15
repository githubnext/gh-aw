//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeImportRelPath verifies that computeImportRelPath produces the correct
// repo-root-relative path for a wide variety of file name and repo name structures.
func TestComputeImportRelPath(t *testing.T) {
	tests := []struct {
		name       string
		fullPath   string
		importPath string
		expected   string
	}{
		// ── Normal absolute paths ─────────────────────────────────────────────
		{
			name:       "absolute path normal repo",
			fullPath:   "/home/user/myrepo/.github/workflows/my-workflow.md",
			importPath: "my-workflow.md",
			expected:   ".github/workflows/my-workflow.md",
		},
		{
			name:       "absolute path subdirectory file",
			fullPath:   "/home/user/myrepo/.github/workflows/shared/tools.md",
			importPath: "shared/tools.md",
			expected:   ".github/workflows/shared/tools.md",
		},
		{
			name:       "absolute path deeply nested subdirectory",
			fullPath:   "/home/user/myrepo/.github/workflows/shared/deep/nested/file.md",
			importPath: "deep/nested/file.md",
			expected:   ".github/workflows/shared/deep/nested/file.md",
		},
		// ── Repo named ".github" ─────────────────────────────────────────────
		{
			name:       "repo named .github — uses LastIndex",
			fullPath:   "/root/.github/.github/workflows/my-workflow.md",
			importPath: "my-workflow.md",
			expected:   ".github/workflows/my-workflow.md",
		},
		{
			name:       "repo named .github with subdirectory",
			fullPath:   "/root/.github/.github/workflows/shared/tools.md",
			importPath: "shared/tools.md",
			expected:   ".github/workflows/shared/tools.md",
		},
		// ── GitHub Pages repo (name ends with .github.io) ────────────────────
		{
			name:       "github.io repo does not duplicate suffix",
			fullPath:   "/home/user/user.github.io/.github/workflows/my-workflow.md",
			importPath: "my-workflow.md",
			expected:   ".github/workflows/my-workflow.md",
		},
		{
			name:       "github.io repo with subdirectory",
			fullPath:   "/home/user/user.github.io/.github/workflows/shared/tools.md",
			importPath: "shared/tools.md",
			expected:   ".github/workflows/shared/tools.md",
		},
		// ── Repo with "github" anywhere in name ──────────────────────────────
		{
			name:       "repo with github in name",
			fullPath:   "/home/user/my-github-project/.github/workflows/workflow.md",
			importPath: "workflow.md",
			expected:   ".github/workflows/workflow.md",
		},
		{
			name:       "org-scoped path with github in repo name",
			fullPath:   "/srv/github-copilot-extensions/.github/workflows/release.md",
			importPath: "release.md",
			expected:   ".github/workflows/release.md",
		},
		// ── Relative paths already starting with ".github/" ──────────────────
		{
			name:       "relative path with .github/ prefix",
			fullPath:   ".github/workflows/my-workflow.md",
			importPath: "my-workflow.md",
			expected:   ".github/workflows/my-workflow.md",
		},
		{
			name:       "relative path with .github/ prefix and subdirectory",
			fullPath:   ".github/workflows/shared/tools.md",
			importPath: "shared/tools.md",
			expected:   ".github/workflows/shared/tools.md",
		},
		// ── Special file names ────────────────────────────────────────────────
		{
			name:       "file name with hyphens",
			fullPath:   "/home/user/repo/.github/workflows/ld-flag-cleanup-worker.md",
			importPath: "ld-flag-cleanup-worker.md",
			expected:   ".github/workflows/ld-flag-cleanup-worker.md",
		},
		{
			name:       "file name with underscores and dots",
			fullPath:   "/home/user/repo/.github/workflows/my.special_file-name.md",
			importPath: "my.special_file-name.md",
			expected:   ".github/workflows/my.special_file-name.md",
		},
		{
			name:       "file in a shared subdirectory",
			fullPath:   "/home/user/repo/.github/workflows/shared/ld-cleanup-shared-tools.md",
			importPath: "shared/ld-cleanup-shared-tools.md",
			expected:   ".github/workflows/shared/ld-cleanup-shared-tools.md",
		},
		// ── Windows-style paths (backslashes) ─────────────────────────────────
		// On Linux/macOS filepath.ToSlash is a no-op for backslashes, so paths
		// containing Windows separators fall back to importPath. On Windows, the
		// conversion works as expected. The test cases below document this behaviour.
		{
			name:       "windows backslash path falls back on Linux",
			fullPath:   `C:\Users\user\myrepo\.github\workflows\my-workflow.md`,
			importPath: "my-workflow.md",
			// On Linux, ToSlash is a no-op for '\', so '/.github/' is not found → fallback.
			expected: "my-workflow.md",
		},
		// ── Fallback: path outside .github/ ───────────────────────────────────
		{
			name:       "path outside .github falls back to importPath",
			fullPath:   "/tmp/some-other-dir/file.md",
			importPath: "file.md",
			expected:   "file.md",
		},
		{
			name:       "empty fullPath falls back to importPath",
			fullPath:   "",
			importPath: "workflow.md",
			expected:   "workflow.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeImportRelPath(tt.fullPath, tt.importPath)
			assert.Equal(t, tt.expected, got, "computeImportRelPath(%q, %q)", tt.fullPath, tt.importPath)
		})
	}
}

// TestJobsFieldExtractedFromMdImport verifies that jobs: in a shared .md workflow's
// frontmatter is captured into ImportsResult.MergedJobs and merged correctly.
func TestJobsFieldExtractedFromMdImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a shared .md workflow with a jobs: section
	sharedContent := `---
name: Shared APM Workflow
jobs:
  apm:
    runs-on: ubuntu-slim
    needs: [activation]
    permissions: {}
    steps:
      - name: Pack
        uses: microsoft/apm-action@v1.4.1
        with:
          pack: 'true'
---

# APM shared workflow
`
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "apm.md"), []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create a main .md workflow that imports the shared workflow
	mainContent := `---
name: Main Workflow
on: issue_comment
imports:
  - uses: shared/apm.md
    with:
      packages:
        - microsoft/apm-sample-package
---

# Main Workflow
`
	result, err := ExtractFrontmatterFromContent(mainContent)
	if err != nil {
		t.Fatalf("ExtractFrontmatterFromContent() error = %v", err)
	}

	importsResult, err := ProcessImportsFromFrontmatterWithSource(result.Frontmatter, tmpDir, nil, "", "")
	if err != nil {
		t.Fatalf("ProcessImportsFromFrontmatterWithSource() error = %v", err)
	}

	assert.NotEmpty(t, importsResult.MergedJobs, "MergedJobs should be populated from shared .md import")
	assert.Contains(t, importsResult.MergedJobs, "apm", "MergedJobs should contain the 'apm' job")
	assert.Contains(t, importsResult.MergedJobs, "ubuntu-slim", "MergedJobs should contain the job runner")
}

// TestEnvFieldExtractedFromMdImport verifies that env: in a shared .md workflow's
// frontmatter is captured into ImportsResult.MergedEnv and merged correctly.
func TestEnvFieldExtractedFromMdImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a shared .md workflow with an env: section
	sharedContent := `---
env:
  TARGET_REPOSITORY: owner/repo
  SHARED_VAR: shared-value
---

# Shared workflow with env vars
`
	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755), "Failed to create shared dir")
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "target.md"), []byte(sharedContent), 0644), "Failed to write shared file")

	// Create a main .md workflow that imports the shared workflow
	mainContent := `---
name: Main Workflow
on: issue_comment
imports:
  - shared/target.md
---

# Main Workflow
`
	result, err := ExtractFrontmatterFromContent(mainContent)
	require.NoError(t, err, "ExtractFrontmatterFromContent should succeed")

	importsResult, err := ProcessImportsFromFrontmatterWithSource(result.Frontmatter, tmpDir, nil, "", "")
	require.NoError(t, err, "ProcessImportsFromFrontmatterWithSource should succeed")

	assert.NotEmpty(t, importsResult.MergedEnv, "MergedEnv should be populated from shared .md import")
	assert.Contains(t, importsResult.MergedEnv, "TARGET_REPOSITORY", "MergedEnv should contain TARGET_REPOSITORY")
	assert.Contains(t, importsResult.MergedEnv, "owner/repo", "MergedEnv should contain the repository value")
	assert.Contains(t, importsResult.MergedEnv, "SHARED_VAR", "MergedEnv should contain SHARED_VAR")
	assert.Equal(t, "shared/target.md", importsResult.MergedEnvSources["TARGET_REPOSITORY"], "MergedEnvSources should track the import path for TARGET_REPOSITORY")
	assert.Equal(t, "shared/target.md", importsResult.MergedEnvSources["SHARED_VAR"], "MergedEnvSources should track the import path for SHARED_VAR")
}

// TestEnvFieldConflictBetweenImports verifies that defining the same env var in two different
// imports produces a compilation error.
func TestEnvFieldConflictBetweenImports(t *testing.T) {
	tmpDir := t.TempDir()

	sharedDir := filepath.Join(tmpDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755), "Failed to create shared dir")

	// First import defines SHARED_KEY
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "first.md"), []byte(`---
env:
  SHARED_KEY: value-from-first
---

# First shared workflow
`), 0644))

	// Second import also defines SHARED_KEY (conflict)
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "second.md"), []byte(`---
env:
  SHARED_KEY: value-from-second
---

# Second shared workflow
`), 0644))

	mainContent := `---
name: Main Workflow
on: issue_comment
imports:
  - shared/first.md
  - shared/second.md
---

# Main Workflow
`
	result, err := ExtractFrontmatterFromContent(mainContent)
	require.NoError(t, err, "ExtractFrontmatterFromContent should succeed")

	_, err = ProcessImportsFromFrontmatterWithSource(result.Frontmatter, tmpDir, nil, "", "")
	require.Error(t, err, "Should error when two imports define the same env var")
	assert.Contains(t, err.Error(), "SHARED_KEY", "Error should mention the conflicting variable name")
}

// TestExtractAllImportFields_BuiltinCacheHit verifies that extractAllImportFields uses the
// process-level builtin frontmatter cache for builtin files without inputs.
func TestExtractAllImportFields_BuiltinCacheHit(t *testing.T) {
	builtinPath := BuiltinPathPrefix + "test/cache-hit.md"
	content := []byte(`---
tools:
  bash: ["echo"]
engine: claude
---

# Cache Hit Test
`)

	// Register the builtin virtual file
	RegisterBuiltinVirtualFile(builtinPath, content)

	// Warm the cache by parsing once
	cachedResult, err := ExtractFrontmatterFromBuiltinFile(builtinPath, content)
	require.NoError(t, err, "should parse builtin file without error")
	assert.NotNil(t, cachedResult, "cached result should not be nil")

	// Verify the cache is populated
	cached, ok := GetBuiltinFrontmatterCache(builtinPath)
	assert.True(t, ok, "builtin cache should have an entry for the path")
	assert.Equal(t, cachedResult, cached, "cached result should match")

	// Call extractAllImportFields with no inputs — should hit the cache
	acc := newImportAccumulator()
	item := importQueueItem{
		fullPath:    builtinPath,
		importPath:  "test/cache-hit.md",
		sectionName: "",
		inputs:      nil,
	}
	visited := map[string]bool{builtinPath: true}

	err = acc.extractAllImportFields(content, item, visited)
	require.NoError(t, err, "extractAllImportFields should succeed for builtin file without inputs")

	// Verify engine was extracted from the cached frontmatter
	assert.NotEmpty(t, acc.engines, "engines should be populated from cached builtin file")
	assert.Contains(t, acc.engines[0], "claude", "engine should be 'claude' from the builtin file")
}

// TestExtractAllImportFields_BuiltinWithInputsBypassesCache verifies that builtin files
// with inputs bypass the cache and use the substituted content.
func TestExtractAllImportFields_BuiltinWithInputsBypassesCache(t *testing.T) {
	builtinPath := BuiltinPathPrefix + "test/cache-bypass.md"
	content := []byte(`---
tools:
  bash: ["echo"]
engine: copilot
---

# Cache Bypass Test
`)

	// Register the builtin virtual file
	RegisterBuiltinVirtualFile(builtinPath, content)

	// Warm the cache
	_, err := ExtractFrontmatterFromBuiltinFile(builtinPath, content)
	require.NoError(t, err, "should parse builtin file without error")

	// Call extractAllImportFields WITH inputs — should bypass the cache
	acc := newImportAccumulator()
	item := importQueueItem{
		fullPath:    builtinPath,
		importPath:  "test/cache-bypass.md",
		sectionName: "",
		inputs:      map[string]any{"key": "value"},
	}
	visited := map[string]bool{builtinPath: true}

	err = acc.extractAllImportFields(content, item, visited)
	require.NoError(t, err, "extractAllImportFields should succeed for builtin file with inputs")

	// Verify engine was still extracted (from direct parse, not cache)
	assert.NotEmpty(t, acc.engines, "engines should be populated even when bypassing cache")
	assert.Contains(t, acc.engines[0], "copilot", "engine should be 'copilot' from the builtin file")
}

func TestValidateImportInputType_Number(t *testing.T) {
	t.Parallel()

	paramDef := map[string]any{"type": "number"}
	importPath := "shared/sample.md"

	t.Run("accepts numeric values", func(t *testing.T) {
		t.Parallel()

		testCases := []any{
			0,
			1,
			int64(2),
			uint(3),
			float64(4.5),
		}

		for _, testValue := range testCases {
			err := validateImportInputType("retries", testValue, "number", paramDef, importPath)
			require.NoError(t, err, "expected %T to be accepted as number", testValue)
		}
	})

	t.Run("rejects non-numeric values", func(t *testing.T) {
		t.Parallel()

		err := validateImportInputType("retries", "3", "number", paramDef, importPath)
		require.Error(t, err, "string value should be rejected for number type")
		assert.Contains(t, err.Error(), "must be a number", "error should explain expected type")
	})
}
