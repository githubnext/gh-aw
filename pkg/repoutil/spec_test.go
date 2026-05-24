//go:build !integration

package repoutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpec_PublicAPI_SplitRepoSlug validates the documented behavior of
// SplitRepoSlug as described in the repoutil README.md specification.
func TestSpec_PublicAPI_SplitRepoSlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		{
			name:          "valid slug returns owner and repo",
			slug:          "github/gh-aw",
			expectedOwner: "github",
			expectedRepo:  "gh-aw",
			expectError:   false,
		},
		{
			name:        "missing separator returns error",
			slug:        "github",
			expectError: true,
		},
		{
			name:        "empty owner returns error",
			slug:        "/gh-aw",
			expectError: true,
		},
		{
			name:        "empty repo returns error",
			slug:        "github/",
			expectError: true,
		},
		{
			name:        "too many separators returns error",
			slug:        "github/gh-aw/x",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.expectError {
				require.Error(t, err, "should return error for slug: %q", tt.slug)
				assert.Empty(t, owner, "owner should be empty when error occurs for slug: %q", tt.slug)
				assert.Empty(t, repo, "repo should be empty when error occurs for slug: %q", tt.slug)
				return
			}
			require.NoError(t, err, "unexpected error for slug: %q", tt.slug)
			assert.Equal(t, tt.expectedOwner, owner, "owner should be %q for slug %q", tt.expectedOwner, tt.slug)
			assert.Equal(t, tt.expectedRepo, repo, "repo should be %q for slug %q", tt.expectedRepo, tt.slug)
		})
	}
}
