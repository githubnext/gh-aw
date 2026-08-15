package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAmbientFolders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		frontmatter map[string]any
		wantFolders []string
		wantErr     string
	}{
		{
			name:        "nil frontmatter returns nil without error",
			frontmatter: nil,
			wantFolders: nil,
		},
		{
			name:        "missing key returns nil without error",
			frontmatter: map[string]any{},
			wantFolders: nil,
		},
		{
			name:        "nil value returns nil without error",
			frontmatter: map[string]any{"ambient-folders": nil},
			wantFolders: nil,
		},
		{
			name:        "array of any strings returns folders",
			frontmatter: map[string]any{"ambient-folders": []any{"docs", "src/lib"}},
			wantFolders: []string{"docs", "src/lib"},
		},
		{
			name:        "empty array of any returns empty slice",
			frontmatter: map[string]any{"ambient-folders": []any{}},
			wantFolders: []string{},
		},
		{
			name:        "typed []string slice is converted",
			frontmatter: map[string]any{"ambient-folders": []string{"a", "b", "c"}},
			wantFolders: []string{"a", "b", "c"},
		},
		{
			name:        "empty typed []string slice returns empty slice",
			frontmatter: map[string]any{"ambient-folders": []string{}},
			wantFolders: []string{},
		},
		{
			name:        "unsupported type returns error",
			frontmatter: map[string]any{"ambient-folders": "docs"},
			wantErr:     "ambient-folders has an unsupported type",
		},
		{
			name:        "unsupported map type returns error",
			frontmatter: map[string]any{"ambient-folders": map[string]any{"a": "b"}},
			wantErr:     "ambient-folders has an unsupported type",
		},
		{
			name:        "non-string entry in array returns error",
			frontmatter: map[string]any{"ambient-folders": []any{"docs", 42}},
			wantErr:     "ambient-folders entry has an unsupported type",
		},
		{
			name:        "non-string first entry in array returns error",
			frontmatter: map[string]any{"ambient-folders": []any{true}},
			wantErr:     "ambient-folders entry has an unsupported type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			folders, err := extractAmbientFolders(tt.frontmatter)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, folders)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantFolders, folders)
		})
	}
}
