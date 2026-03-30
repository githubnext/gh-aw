//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
