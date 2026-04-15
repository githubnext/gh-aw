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
		wantErr       bool
	}{
		{
			name:          "valid slug returns owner and repo",
			slug:          "github/gh-aw",
			expectedOwner: "github",
			expectedRepo:  "gh-aw",
			wantErr:       false,
		},
		{
			name:    "missing separator returns error",
			slug:    "github",
			wantErr: true,
		},
		{
			name:    "empty owner returns error",
			slug:    "/gh-aw",
			wantErr: true,
		},
		{
			name:    "empty repo returns error",
			slug:    "github/",
			wantErr: true,
		},
		{
			name:    "too many separators returns error",
			slug:    "github/gh-aw/x",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := SplitRepoSlug(tt.slug)
			if tt.wantErr {
				assert.Error(t, err, "should return error for slug: %q", tt.slug)
				return
			}
			require.NoError(t, err, "unexpected error for slug: %q", tt.slug)
			assert.Equal(t, tt.expectedOwner, owner, "owner mismatch for slug: %q", tt.slug)
			assert.Equal(t, tt.expectedRepo, repo, "repo mismatch for slug: %q", tt.slug)
		})
	}
}
