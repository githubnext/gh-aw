//go:build !integration

package repoutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRepoSlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "valid slug",
			slug:          "github/gh-aw",
			expectedOwner: "github",
			expectedRepo:  "gh-aw",
			expectError:   false,
		},
		{
			name:          "another valid slug",
			slug:          "octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			expectError:   false,
		},
		{
			name:        "invalid slug - no separator",
			slug:        "githubnext",
			expectError: true,
		},
		{
			name:        "invalid slug - multiple separators",
			slug:        "github/gh-aw/extra",
			expectError: true,
		},
		{
			name:        "invalid slug - empty",
			slug:        "",
			expectError: true,
		},
		{
			name:        "invalid slug - only separator",
			slug:        "/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.expectError {
				require.Error(t, err, "SplitRepoSlug(%q) should return an error", tt.slug)
			} else {
				require.NoError(t, err, "SplitRepoSlug(%q) should not return an error", tt.slug)
				assert.Equal(t, tt.expectedOwner, owner, "SplitRepoSlug(%q) owner mismatch", tt.slug)
				assert.Equal(t, tt.expectedRepo, repo, "SplitRepoSlug(%q) repo mismatch", tt.slug)
			}
		})
	}
}

// Additional edge case tests

func TestSplitRepoSlug_Whitespace(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "leading whitespace",
			slug:          " owner/repo",
			expectedOwner: " owner",
			expectedRepo:  "repo",
			expectError:   false, // Will split but owner will have space
		},
		{
			name:          "trailing whitespace",
			slug:          "owner/repo ",
			expectedOwner: "owner",
			expectedRepo:  "repo ",
			expectError:   false, // Will split but repo will have space
		},
		{
			name:          "whitespace in middle",
			slug:          "owner /repo",
			expectedOwner: "owner ",
			expectedRepo:  "repo",
			expectError:   false, // Split will work but owner will have space
		},
		{
			name:          "tab character",
			slug:          "owner\t/repo",
			expectedOwner: "owner\t",
			expectedRepo:  "repo",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.expectError {
				require.Error(t, err, "SplitRepoSlug(%q) should return an error", tt.slug)
			} else {
				require.NoError(t, err, "SplitRepoSlug(%q) should not return an error", tt.slug)
				assert.Equal(t, tt.expectedOwner, owner, "SplitRepoSlug(%q) owner mismatch", tt.slug)
				assert.Equal(t, tt.expectedRepo, repo, "SplitRepoSlug(%q) repo mismatch", tt.slug)
			}
		})
	}
}

func TestSplitRepoSlug_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "hyphen in owner",
			slug:          "github-next/repo",
			expectedOwner: "github-next",
			expectedRepo:  "repo",
			expectError:   false,
		},
		{
			name:          "hyphen in repo",
			slug:          "owner/my-repo",
			expectedOwner: "owner",
			expectedRepo:  "my-repo",
			expectError:   false,
		},
		{
			name:          "underscore in names",
			slug:          "my_org/my_repo",
			expectedOwner: "my_org",
			expectedRepo:  "my_repo",
			expectError:   false,
		},
		{
			name:          "numbers in names",
			slug:          "org123/repo456",
			expectedOwner: "org123",
			expectedRepo:  "repo456",
			expectError:   false,
		},
		{
			name:          "dots in names",
			slug:          "org.name/repo.name",
			expectedOwner: "org.name",
			expectedRepo:  "repo.name",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.expectError {
				require.Error(t, err, "SplitRepoSlug(%q) should return an error", tt.slug)
			} else {
				require.NoError(t, err, "SplitRepoSlug(%q) should not return an error", tt.slug)
				assert.Equal(t, tt.expectedOwner, owner, "SplitRepoSlug(%q) owner mismatch", tt.slug)
				assert.Equal(t, tt.expectedRepo, repo, "SplitRepoSlug(%q) repo mismatch", tt.slug)
			}
		})
	}
}

func TestSplitRepoSlug_Idempotent(t *testing.T) {
	// Test that splitting and rejoining gives the same result
	slugs := []string{
		"owner/repo",
		"github-next/gh-aw",
		"my_org/my_repo",
		"org123/repo456",
	}

	for _, slug := range slugs {
		t.Run(slug, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(slug)
			require.NoError(t, err, "SplitRepoSlug(%q) should not return an error", slug)

			rejoined := owner + "/" + repo
			assert.Equal(t, slug, rejoined, "Split and rejoin should preserve the original slug %q", slug)
		})
	}
}

func BenchmarkSplitRepoSlug_Valid(b *testing.B) {
	slug := "github/gh-aw"
	for b.Loop() {
		_, _, _ = SplitRepoSlug(slug)
	}
}

func BenchmarkSplitRepoSlug_Invalid(b *testing.B) {
	slug := "invalid"
	for b.Loop() {
		_, _, _ = SplitRepoSlug(slug)
	}
}
