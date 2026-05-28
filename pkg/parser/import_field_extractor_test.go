//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"strings"
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

// TestExtractOTLPEndpointsFromObsMap verifies that all three endpoint forms
// (string, object, array) are correctly extracted from a raw observability map.
func TestExtractOTLPEndpointsFromObsMap(t *testing.T) {
	tests := []struct {
		name string
		obs  map[string]any
		want []observabilityImportEndpoint
	}{
		{
			name: "nil map returns nil",
			obs:  nil,
			want: nil,
		},
		{
			name: "empty map returns nil",
			obs:  map[string]any{},
			want: nil,
		},
		{
			name: "missing otlp key returns nil",
			obs:  map[string]any{"other": "value"},
			want: nil,
		},
		{
			name: "string form without headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "string form with string headers (backward-compat)",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
					"headers":  "Authorization=Bearer tok",
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: "Authorization=Bearer tok"},
			},
		},
		{
			name: "string form with map headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": "https://traces.example.com:4317",
					"headers":  map[string]any{"Authorization": "Bearer tok"},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: map[string]any{"Authorization": "Bearer tok"}},
			},
		},
		{
			name: "object form",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{
						"url":     "https://traces.example.com:4317",
						"headers": map[string]any{"X-API-Key": "key1"},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317", Headers: map[string]any{"X-API-Key": "key1"}},
			},
		},
		{
			name: "object form without headers",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": map[string]any{"url": "https://traces.example.com:4317"},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://traces.example.com:4317"},
			},
		},
		{
			name: "array form with multiple endpoints",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": "https://primary.example.com:4317"},
						map[string]any{"url": "https://secondary.example.com:4317", "headers": map[string]any{"X-Key": "v"}},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://primary.example.com:4317"},
				{URL: "https://secondary.example.com:4317", Headers: map[string]any{"X-Key": "v"}},
			},
		},
		{
			name: "array form skips entries with empty URL",
			obs: map[string]any{
				"otlp": map[string]any{
					"endpoint": []any{
						map[string]any{"url": ""},
						map[string]any{"url": "https://valid.example.com:4317"},
					},
				},
			},
			want: []observabilityImportEndpoint{
				{URL: "https://valid.example.com:4317"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOTLPEndpointsFromObsMap(tt.obs)
			assert.Equal(t, tt.want, got, "extractOTLPEndpointsFromObsMap")
		})
	}
}

// TestMergeObservabilityConfigs verifies that multiple observability JSON blobs are
// merged into a single config with all OTLP endpoints in array form, with deduplication.
func TestMergeObservabilityConfigs(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		got := mergeObservabilityConfigs(nil)
		assert.Empty(t, got, "nil configs should return empty string")

		got = mergeObservabilityConfigs([]string{})
		assert.Empty(t, got, "empty configs should return empty string")
	})

	t.Run("single import with string endpoint", func(t *testing.T) {
		configs := []string{`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce non-empty result")
		assert.Contains(t, got, `"https://traces.example.com:4317"`, "should include the endpoint URL")
	})

	t.Run("two imports with distinct string endpoints", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://primary.example.com:4317"}}`,
			`{"otlp":{"endpoint":"https://secondary.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "primary.example.com", "should include first endpoint")
		assert.Contains(t, got, "secondary.example.com", "should include second endpoint")
	})

	t.Run("two imports with same URL deduplicates", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`,
			`{"otlp":{"endpoint":"https://traces.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		// The URL should appear only once in the output
		count := strings.Count(got, "traces.example.com")
		assert.Equal(t, 1, count, "duplicate URL should appear only once")
	})

	t.Run("mix of string and object notation", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":"https://primary.example.com:4317"}}`,
			`{"otlp":{"endpoint":{"url":"https://secondary.example.com:4317","headers":{"X-Key":"val"}}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "primary.example.com", "should include first endpoint (string form)")
		assert.Contains(t, got, "secondary.example.com", "should include second endpoint (object form)")
		assert.Contains(t, got, "X-Key", "should preserve headers from object form")
	})

	t.Run("import with array notation contributes all endpoints", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":[{"url":"https://a.example.com:4317"},{"url":"https://b.example.com:4317"}]}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "a.example.com", "should include first array endpoint")
		assert.Contains(t, got, "b.example.com", "should include second array endpoint")
	})

	t.Run("three imports with mixed notation and overlap deduplicates correctly", func(t *testing.T) {
		configs := []string{
			// import 1: string form
			`{"otlp":{"endpoint":"https://a.example.com:4317"}}`,
			// import 2: array with one duplicate and one new
			`{"otlp":{"endpoint":[{"url":"https://a.example.com:4317"},{"url":"https://b.example.com:4317"}]}}`,
			// import 3: object form with new endpoint
			`{"otlp":{"endpoint":{"url":"https://c.example.com:4317"}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")

		countA := strings.Count(got, "a.example.com")
		assert.Equal(t, 1, countA, "a.example.com should appear exactly once (dedup)")
		assert.Contains(t, got, "b.example.com", "should include b.example.com")
		assert.Contains(t, got, "c.example.com", "should include c.example.com")
	})

	t.Run("headers preserved in original format (map not string)", func(t *testing.T) {
		configs := []string{
			`{"otlp":{"endpoint":{"url":"https://traces.example.com","headers":{"Authorization":"Bearer tok"}}}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "should produce merged result")
		assert.Contains(t, got, "Authorization", "should preserve header key")
		assert.Contains(t, got, "Bearer tok", "should preserve header value")
	})

	t.Run("invalid JSON blob is skipped", func(t *testing.T) {
		configs := []string{
			`not-valid-json`,
			`{"otlp":{"endpoint":"https://valid.example.com:4317"}}`,
		}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "valid config should still produce output")
		assert.Contains(t, got, "valid.example.com", "should include valid endpoint")
	})

	t.Run("config with no endpoints returns empty string", func(t *testing.T) {
		configs := []string{`{"otlp":{}}`}
		got := mergeObservabilityConfigs(configs)
		assert.Empty(t, got, "config without endpoints should return empty string")
	})

	t.Run("github-app config is preserved even when no endpoints are set", func(t *testing.T) {
		configs := []string{`{"otlp":{"github-app":{"app-id":"${{ vars.APP_ID }}","private-key":"${{ secrets.APP_PRIVATE_KEY }}"}}}`}
		got := mergeObservabilityConfigs(configs)
		require.NotEmpty(t, got, "github-app-only config should still produce merged observability")
		assert.Contains(t, got, `"github-app"`, "should include github-app block")
		assert.Contains(t, got, `"app-id":"${{ vars.APP_ID }}"`, "should preserve app-id")
		assert.Contains(t, got, `"private-key":"${{ secrets.APP_PRIVATE_KEY }}"`, "should preserve private-key")
	})
}
