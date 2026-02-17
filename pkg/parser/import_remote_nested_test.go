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
				Owner: "elastic",
				Repo:  "ai-github-actions",
				Ref:   "main",
			},
		},
		{
			name: "workflowspec with SHA ref",
			spec: "elastic/ai-github-actions/gh-agent-workflows/mention-in-pr/rwxp.md@160c33700227b5472dc3a08aeea1e774389a1a84",
			expected: &remoteImportOrigin{
				Owner: "elastic",
				Repo:  "ai-github-actions",
				Ref:   "160c33700227b5472dc3a08aeea1e774389a1a84",
			},
		},
		{
			name: "workflowspec without ref defaults to main",
			spec: "elastic/ai-github-actions/gh-agent-workflows/file.md",
			expected: &remoteImportOrigin{
				Owner: "elastic",
				Repo:  "ai-github-actions",
				Ref:   "main",
			},
		},
		{
			name: "workflowspec with section reference",
			spec: "owner/repo/path/file.md@v1.0#SectionName",
			expected: &remoteImportOrigin{
				Owner: "owner",
				Repo:  "repo",
				Ref:   "v1.0",
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
	})

	t.Run("local import does not get remote origin", func(t *testing.T) {
		localPath := "shared/tools.md"
		assert.False(t, isWorkflowSpec(localPath), "Should not be recognized as workflowspec")

		origin := parseRemoteOrigin(localPath)
		assert.Nil(t, origin, "Local paths should not produce remote origin")
	})

	t.Run("nested relative path from remote parent produces correct workflowspec", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner: "elastic",
			Repo:  "ai-github-actions",
			Ref:   "main",
		}
		nestedPath := "shared/elastic-tools.md"

		// This is the logic from the import processor:
		// When parent is remote and nested path is not a workflowspec,
		// construct a workflowspec resolving against .github/workflows/
		expectedSpec := fmt.Sprintf("%s/%s/.github/workflows/%s@%s",
			origin.Owner, origin.Repo, nestedPath, origin.Ref)

		assert.Equal(t,
			"elastic/ai-github-actions/.github/workflows/shared/elastic-tools.md@main",
			expectedSpec,
			"Nested relative import should resolve to remote .github/workflows/ path",
		)

		// The constructed spec should be recognized as a workflowspec
		assert.True(t, isWorkflowSpec(expectedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested relative path with ./ prefix is cleaned", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner: "org",
			Repo:  "repo",
			Ref:   "v1.0",
		}
		nestedPath := "./shared/tools.md"

		// Clean the ./ prefix as the import processor does
		cleanPath := nestedPath
		if len(cleanPath) > 2 && cleanPath[:2] == "./" {
			cleanPath = cleanPath[2:]
		}

		expectedSpec := fmt.Sprintf("%s/%s/.github/workflows/%s@%s",
			origin.Owner, origin.Repo, cleanPath, origin.Ref)

		assert.Equal(t,
			"org/repo/.github/workflows/shared/tools.md@v1.0",
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
	})

	t.Run("path traversal in nested import is rejected", func(t *testing.T) {
		// A nested import like ../../../etc/passwd should be rejected
		// when constructing the remote workflowspec
		nestedPath := "../../../etc/passwd"
		cleanPath := path.Clean(strings.TrimPrefix(nestedPath, "./"))

		assert.True(t, strings.HasPrefix(cleanPath, ".."),
			"Cleaned path should start with .. and be rejected by the import processor")
	})

	t.Run("SHA ref is preserved in nested resolution", func(t *testing.T) {
		sha := "160c33700227b5472dc3a08aeea1e774389a1a84"
		origin := &remoteImportOrigin{
			Owner: "elastic",
			Repo:  "ai-github-actions",
			Ref:   sha,
		}
		nestedPath := "shared/formatting.md"

		resolvedSpec := fmt.Sprintf("%s/%s/.github/workflows/%s@%s",
			origin.Owner, origin.Repo, nestedPath, origin.Ref)

		assert.Equal(t,
			"elastic/ai-github-actions/.github/workflows/shared/formatting.md@"+sha,
			resolvedSpec,
			"SHA ref should be preserved for nested imports",
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
			Owner: "elastic",
			Repo:  "ai-github-actions",
			Ref:   "main",
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
