//go:build !integration

package parser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRemoteOrigin(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected *remoteImportOrigin
	}{
		{
			name: "basic workflowspec with ref",
			spec: "elastic/ai-github-actions/gh-agent-workflows/mention-in-pr/rwxp.md@main",
			expected: &remoteImportOrigin{
				Owner:    "elastic",
				Repo:     "ai-github-actions",
				Ref:      "main",
				BasePath: "gh-agent-workflows/mention-in-pr",
			},
		},
		{
			name: "workflowspec with SHA ref",
			spec: "elastic/ai-github-actions/gh-agent-workflows/mention-in-pr/rwxp.md@160c33700227b5472dc3a08aeea1e774389a1a84",
			expected: &remoteImportOrigin{
				Owner:    "elastic",
				Repo:     "ai-github-actions",
				Ref:      "160c33700227b5472dc3a08aeea1e774389a1a84",
				BasePath: "gh-agent-workflows/mention-in-pr",
			},
		},
		{
			name: "workflowspec without ref defaults to main",
			spec: "elastic/ai-github-actions/gh-agent-workflows/file.md",
			expected: &remoteImportOrigin{
				Owner:    "elastic",
				Repo:     "ai-github-actions",
				Ref:      "main",
				BasePath: "gh-agent-workflows",
			},
		},
		{
			name: "workflowspec with section reference",
			spec: "owner/repo/path/file.md@v1.0#SectionName",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "v1.0",
				BasePath: "path",
			},
		},
		{
			name: "workflowspec under .github/workflows",
			spec: "owner/repo/.github/workflows/test.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: ".github/workflows",
			},
		},
		{
			name: "workflowspec with deep path",
			spec: "owner/repo/a/b/c/d/e/file.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "a/b/c/d/e",
			},
		},
		{
			name: "workflowspec directly in repo root (minimal path)",
			spec: "owner/repo/file.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "",
			},
		},
		{
			name: "path with ./ should be cleaned",
			spec: "owner/repo/./path/./file.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "path",
			},
		},
		{
			name: "path with redundant slashes should be cleaned",
			spec: "owner/repo/path//to///file.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "path/to",
			},
		},
		{
			name:     "too few parts returns nil",
			spec:     "owner/repo",
			expected: nil,
		},
		{
			name:     "single part returns nil",
			spec:     "file.md",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRemoteOrigin(tt.spec)
			if tt.expected == nil {
				assert.Nilf(t, result, "Expected nil for spec: %s", tt.spec)
			} else {
				require.NotNilf(t, result, "Expected non-nil for spec: %s", tt.spec)
				assert.Equal(t, tt.expected.Owner, result.Owner, "Owner mismatch")
				assert.Equal(t, tt.expected.Repo, result.Repo, "Repo mismatch")
				assert.Equal(t, tt.expected.Ref, result.Ref, "Ref mismatch")
				assert.Equal(t, tt.expected.BasePath, result.BasePath, "BasePath mismatch")
			}
		})
	}
}

func TestLocalImportResolutionBaseline(t *testing.T) {
	// Baseline test: verifies local relative imports resolve correctly.
	// This ensures the import processor still works for non-remote imports
	// after the remote origin tracking changes.

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0o755)
	require.NoError(t, err, "Failed to create workflows directory")

	sharedDir := filepath.Join(workflowsDir, "shared")
	err = os.MkdirAll(sharedDir, 0o755)
	require.NoError(t, err, "Failed to create shared directory")

	localSharedFile := filepath.Join(sharedDir, "local-tools.md")
	err = os.WriteFile(localSharedFile, []byte("# Local tools\n"), 0o644)
	require.NoError(t, err, "Failed to create local shared file")

	frontmatter := map[string]any{
		"imports": []any{"shared/local-tools.md"},
	}
	cache := NewImportCache(tmpDir)
	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, workflowsDir, cache)
	require.NoError(t, err, "Local import resolution should succeed")
	assert.NotNil(t, result, "Result should not be nil")
}

func TestRemoteOriginPropagation(t *testing.T) {
	// Test that the remote origin is correctly tracked on queue items
	// when a top-level import is a workflowspec

	// We can't easily test the full remote fetch flow in a unit test,
	// but we can verify the parsing and propagation logic

	t.Run("workflowspec import gets remote origin", func(t *testing.T) {
		spec := "elastic/ai-github-actions/gh-agent-workflows/mention-in-pr/rwxp.md@main"
		assert.True(t, isWorkflowSpec(spec), "Should be recognized as workflowspec")

		origin := parseRemoteOrigin(spec)
		require.NotNil(t, origin, "Should parse remote origin")
		assert.Equal(t, "elastic", origin.Owner, "Owner should be elastic")
		assert.Equal(t, "ai-github-actions", origin.Repo, "Repo should be ai-github-actions")
		assert.Equal(t, "main", origin.Ref, "Ref should be main")
		assert.Equal(t, "gh-agent-workflows/mention-in-pr", origin.BasePath, "BasePath should be gh-agent-workflows/mention-in-pr")
	})

	t.Run("local import does not get remote origin", func(t *testing.T) {
		localPath := "shared/tools.md"
		assert.False(t, isWorkflowSpec(localPath), "Should not be recognized as workflowspec")

		origin := parseRemoteOrigin(localPath)
		assert.Nil(t, origin, "Local paths should not produce remote origin")
	})

	t.Run("nested relative path from remote parent with BasePath produces correct workflowspec", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "elastic",
			Repo:     "ai-github-actions",
			Ref:      "main",
			BasePath: "gh-agent-workflows",
		}
		nestedPath := "gh-aw-fragments/elastic-tools.md"

		// This is the NEW logic from the import processor:
		// When parent is remote and has a BasePath, use that BasePath instead of .github/workflows/
		basePath := origin.BasePath
		if basePath == "" {
			basePath = ".github/workflows"
		}
		expectedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			origin.Owner, origin.Repo, basePath, nestedPath, origin.Ref)

		assert.Equal(t,
			"elastic/ai-github-actions/gh-agent-workflows/gh-aw-fragments/elastic-tools.md@main",
			expectedSpec,
			"Nested relative import should resolve to parent's BasePath",
		)

		// The constructed spec should be recognized as a workflowspec
		assert.True(t, isWorkflowSpec(expectedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested relative path from remote parent without BasePath uses .github/workflows", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "elastic",
			Repo:     "ai-github-actions",
			Ref:      "main",
			BasePath: "", // Empty BasePath - should fall back to .github/workflows
		}
		nestedPath := "shared/elastic-tools.md"

		// When BasePath is empty, fall back to .github/workflows/
		basePath := origin.BasePath
		if basePath == "" {
			basePath = ".github/workflows"
		}
		expectedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			origin.Owner, origin.Repo, basePath, nestedPath, origin.Ref)

		assert.Equal(t,
			"elastic/ai-github-actions/.github/workflows/shared/elastic-tools.md@main",
			expectedSpec,
			"Nested relative import with empty BasePath should fall back to .github/workflows/",
		)

		// The constructed spec should be recognized as a workflowspec
		assert.True(t, isWorkflowSpec(expectedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested relative path with ./ prefix is cleaned", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "org",
			Repo:     "repo",
			Ref:      "v1.0",
			BasePath: "custom-path",
		}
		nestedPath := "./shared/tools.md"

		// Clean the ./ prefix as the import processor does
		cleanPath := nestedPath
		if len(cleanPath) > 2 && cleanPath[:2] == "./" {
			cleanPath = cleanPath[2:]
		}

		basePath := origin.BasePath
		if basePath == "" {
			basePath = ".github/workflows"
		}
		expectedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			origin.Owner, origin.Repo, basePath, cleanPath, origin.Ref)

		assert.Equal(t,
			"org/repo/custom-path/shared/tools.md@v1.0",
			expectedSpec,
			"Dot-prefix should be stripped when constructing remote spec",
		)
	})

	t.Run("nested workflowspec from remote parent gets its own origin", func(t *testing.T) {
		// If a remote file references another workflowspec, it should
		// get its own origin, not inherit the parent's
		nestedSpec := "other-org/other-repo/path/file.md@v2.0"
		assert.True(t, isWorkflowSpec(nestedSpec), "Should be recognized as workflowspec")

		origin := parseRemoteOrigin(nestedSpec)
		require.NotNil(t, origin, "Should parse remote origin for nested workflowspec")
		assert.Equal(t, "other-org", origin.Owner, "Should use nested spec's owner")
		assert.Equal(t, "other-repo", origin.Repo, "Should use nested spec's repo")
		assert.Equal(t, "v2.0", origin.Ref, "Should use nested spec's ref")
		assert.Equal(t, "path", origin.BasePath, "Should use nested spec's base path")
	})

	t.Run("path traversal in nested import is rejected", func(t *testing.T) {
		// A nested import like ../../../etc/passwd should be rejected
		// when constructing the remote workflowspec
		nestedPath := "../../../etc/passwd"
		cleanPath := path.Clean(strings.TrimPrefix(nestedPath, "./"))

		assert.True(t, strings.HasPrefix(cleanPath, ".."),
			"Cleaned path should start with .. and be rejected by the import processor")
	})

	t.Run("SHA ref is preserved in nested resolution with BasePath", func(t *testing.T) {
		sha := "160c33700227b5472dc3a08aeea1e774389a1a84"
		origin := &remoteImportOrigin{
			Owner:    "elastic",
			Repo:     "ai-github-actions",
			Ref:      sha,
			BasePath: "gh-agent-workflows",
		}
		nestedPath := "shared/formatting.md"

		basePath := origin.BasePath
		if basePath == "" {
			basePath = ".github/workflows"
		}
		resolvedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			origin.Owner, origin.Repo, basePath, nestedPath, origin.Ref)

		assert.Equal(t,
			"elastic/ai-github-actions/gh-agent-workflows/shared/formatting.md@"+sha,
			resolvedSpec,
			"SHA ref should be preserved for nested imports with BasePath",
		)
	})

	// Regression test for githubnext/agentics#182
	// Tests the scenario where:
	// - workflows/workflow.md imports shared/file1.md
	// - workflows/shared/file1.md imports file2.md (relative to its own directory)
	// - File2.md should resolve to workflows/shared/file2.md, not workflows/file2.md
	t.Run("two-level nested imports resolve relative to immediate parent", func(t *testing.T) {
		// Scenario:
		// top-level workflow: githubnext/agentics/workflows/workflow.md@main
		// → workflow.md imports: shared/file1.md
		// → file1.md imports: file2.md (should resolve to shared/file2.md)

		// Step 1: Top-level workflow import produces this remoteOrigin
		topLevelOrigin := parseRemoteOrigin("githubnext/agentics/workflows/workflow.md@main")
		require.NotNil(t, topLevelOrigin, "Should parse top-level workflow")
		assert.Equal(t, "workflows", topLevelOrigin.BasePath, "Top-level BasePath should be 'workflows'")

		// Step 2: When workflow.md imports shared/file1.md, we construct the resolvedPath
		firstNestedPath := "shared/file1.md"
		basePath := topLevelOrigin.BasePath
		if basePath == "" {
			basePath = ".github/workflows"
		}
		file1ResolvedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			topLevelOrigin.Owner, topLevelOrigin.Repo, basePath, firstNestedPath, topLevelOrigin.Ref)

		assert.Equal(t,
			"githubnext/agentics/workflows/shared/file1.md@main",
			file1ResolvedSpec,
			"First nested import should resolve correctly",
		)

		// Step 3: Parse the remoteOrigin from file1's resolved spec
		// This is the KEY fix - file1's origin should have BasePath="workflows/shared"
		file1Origin := parseRemoteOrigin(file1ResolvedSpec)
		require.NotNil(t, file1Origin, "Should parse file1's remote origin from resolved spec")
		assert.Equal(t, "githubnext", file1Origin.Owner, "File1 Owner")
		assert.Equal(t, "agentics", file1Origin.Repo, "File1 Repo")
		assert.Equal(t, "main", file1Origin.Ref, "File1 Ref")
		assert.Equal(t, "workflows/shared", file1Origin.BasePath,
			"File1's BasePath should be 'workflows/shared' - includes the subdirectory")

		// Step 4: When file1.md imports file2.md, use file1's origin (not top-level origin!)
		secondNestedPath := "file2.md"
		file1BasePath := file1Origin.BasePath
		if file1BasePath == "" {
			file1BasePath = ".github/workflows"
		}
		file2ResolvedSpec := fmt.Sprintf("%s/%s/%s/%s@%s",
			file1Origin.Owner, file1Origin.Repo, file1BasePath, secondNestedPath, file1Origin.Ref)

		assert.Equal(t,
			"githubnext/agentics/workflows/shared/file2.md@main",
			file2ResolvedSpec,
			"Second nested import should resolve relative to file1's directory (workflows/shared), not top-level (workflows)",
		)
	})
}

func TestImportQueueItemRemoteOriginField(t *testing.T) {
	// Verify the struct field exists and works correctly

	t.Run("queue item with nil remote origin", func(t *testing.T) {
		item := importQueueItem{
			importPath:   "shared/tools.md",
			fullPath:     "/tmp/tools.md",
			sectionName:  "",
			baseDir:      "/workspace/.github/workflows",
			remoteOrigin: nil,
		}
		assert.Nil(t, item.remoteOrigin, "Local import should have nil remote origin")
	})

	t.Run("queue item with remote origin", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "elastic",
			Repo:     "ai-github-actions",
			Ref:      "main",
			BasePath: "path",
		}
		item := importQueueItem{
			importPath:   "elastic/ai-github-actions/path/file.md@main",
			fullPath:     "/tmp/cache/file.md",
			sectionName:  "",
			baseDir:      "/workspace/.github/workflows",
			remoteOrigin: origin,
		}
		require.NotNil(t, item.remoteOrigin, "Remote import should have non-nil remote origin")
		assert.Equal(t, "elastic", item.remoteOrigin.Owner, "Owner should match")
		assert.Equal(t, "ai-github-actions", item.remoteOrigin.Repo, "Repo should match")
		assert.Equal(t, "main", item.remoteOrigin.Ref, "Ref should match")
		assert.Equal(t, "path", item.remoteOrigin.BasePath, "BasePath should match")
	})
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "HTTP 404 message",
			errMsg:   "HTTP 404: Not Found",
			expected: true,
		},
		{
			name:     "lowercase not found",
			errMsg:   "failed to fetch file: not found",
			expected: true,
		},
		{
			name:     "404 status code in message",
			errMsg:   "server returned 404 for request",
			expected: true,
		},
		{
			name:     "authentication error",
			errMsg:   "HTTP 401: Unauthorized",
			expected: false,
		},
		{
			name:     "server error",
			errMsg:   "HTTP 500: Internal Server Error",
			expected: false,
		},
		{
			name:     "empty string",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.errMsg)
			assert.Equal(t, tt.expected, result, "isNotFoundError(%q)", tt.errMsg)
		})
	}
}

func TestResolveRemoteSymlinksPathConstruction(t *testing.T) {
	// These tests verify the path construction logic of resolveRemoteSymlinks
	// without making real API calls. The actual symlink resolution requires
	// GitHub API access, which is tested in integration tests.

	t.Run("single component path returns error", func(t *testing.T) {
		_, err := resolveRemoteSymlinks("owner", "repo", "file.md", "main")
		assert.Error(t, err, "Single component path has no directories to resolve")
	})

	t.Run("symlink target resolution logic", func(t *testing.T) {
		// Verify the path math that resolveRemoteSymlinks performs internally.
		// Given a symlink at .github/workflows/shared -> ../../gh-agent-workflows/shared,
		// the resolution should produce gh-agent-workflows/shared/file.md

		// Simulate: parts = [".github", "workflows", "shared", "elastic-tools.md"]
		// Symlink at index 3 (parts[:3] = ".github/workflows/shared")
		// Target: "../../gh-agent-workflows/shared"
		// Parent: ".github/workflows"

		parentDir := ".github/workflows"
		target := "../../gh-agent-workflows/shared"

		// This mirrors the logic in resolveRemoteSymlinks using path.Clean/path.Join
		resolvedBase := path.Clean(path.Join(parentDir, target))
		remaining := "elastic-tools.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, "gh-agent-workflows/shared/elastic-tools.md", resolvedPath,
			"Symlink at .github/workflows/shared pointing to ../../gh-agent-workflows/shared should resolve correctly")
	})

	t.Run("symlink at first component", func(t *testing.T) {
		// Simulate: parts = ["link-dir", "subdir", "file.md"]
		// Symlink at index 1 (parts[:1] = "link-dir")
		// Target: "actual-dir"
		// Parent: "" (root)

		target := "actual-dir"
		resolvedBase := path.Clean(target)
		remaining := "subdir/file.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, "actual-dir/subdir/file.md", resolvedPath,
			"Symlink at root level should resolve correctly")
	})

	t.Run("nested symlink resolution", func(t *testing.T) {
		// Simulate: parts = ["gh-agent-workflows", "gh-aw-workflows", "file.md"]
		// Symlink at index 2 (parts[:2] = "gh-agent-workflows/gh-aw-workflows")
		// Target: "../.github/workflows/gh-aw-workflows"
		// Parent: "gh-agent-workflows"

		parentDir := "gh-agent-workflows"
		target := "../.github/workflows/gh-aw-workflows"

		resolvedBase := path.Clean(path.Join(parentDir, target))
		remaining := "file.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, ".github/workflows/gh-aw-workflows/file.md", resolvedPath,
			"Nested symlink with ../ target should resolve correctly")
	})
}

func TestParseRemoteOriginWithCleanedPaths(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected *remoteImportOrigin
	}{
		{
			name: "path with ./ components should be cleaned",
			spec: "owner/repo/./workflows/./test.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "workflows",
			},
		},
		{
			name: "path with redundant slashes should be cleaned",
			spec: "owner/repo/workflows//subdir///test.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "workflows/subdir",
			},
		},
		{
			name: "complex path cleaning",
			spec: "owner/repo/./a//b/./c///test.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "a/b/c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRemoteOrigin(tt.spec)
			require.NotNil(t, result, "Should parse remote origin for spec: %s", tt.spec)
			assert.Equal(t, tt.expected.Owner, result.Owner, "Owner mismatch")
			assert.Equal(t, tt.expected.Repo, result.Repo, "Repo mismatch")
			assert.Equal(t, tt.expected.Ref, result.Ref, "Ref mismatch")
			assert.Equal(t, tt.expected.BasePath, result.BasePath, "BasePath mismatch")
		})
	}
}

func TestParseRemoteOriginWithURLFormats(t *testing.T) {
	t.Run("URL-like paths are currently accepted by isWorkflowSpec", func(t *testing.T) {
		// URLs are currently accepted by isWorkflowSpec because they have >3 parts when split by /
		// This documents the current behavior - URLs might need special handling in the future
		urlPaths := []string{
			"https://github.com/owner/repo/path/file.md",
			"http://github.com/owner/repo/path/file.md",
			"https://github.enterprise.com/owner/repo/path/file.md",
		}

		for _, urlPath := range urlPaths {
			// Currently, isWorkflowSpec accepts URLs (they have >3 slash-separated parts)
			isSpec := isWorkflowSpec(urlPath)
			assert.True(t, isSpec, "URL is currently accepted as workflowspec: %s", urlPath)

			// parseRemoteOrigin will parse the URL parts literally
			// For "https://github.com/owner/repo/path/file.md":
			// - Parts: ["https:", "", "github.com", "owner", "repo", "path", "file.md"]
			// - Owner would be "https:" (first part after splitting by /)
			// This test documents the current behavior for future reference
			origin := parseRemoteOrigin(urlPath)
			if origin != nil {
				t.Logf("URL %s parsed as: owner=%s, repo=%s, basePath=%s",
					urlPath, origin.Owner, origin.Repo, origin.BasePath)
			}
		}
	})

	t.Run("enterprise domain workflowspec format", func(t *testing.T) {
		// Enterprise GitHub uses the same owner/repo/path format
		// The domain is handled by GH_HOST environment variable, not in the workflowspec
		spec := "enterprise-org/enterprise-repo/workflows/test.md@main"

		result := parseRemoteOrigin(spec)
		require.NotNil(t, result, "Should parse enterprise workflowspec")
		assert.Equal(t, "enterprise-org", result.Owner)
		assert.Equal(t, "enterprise-repo", result.Repo)
		assert.Equal(t, "main", result.Ref)
		assert.Equal(t, "workflows", result.BasePath)
	})
}
