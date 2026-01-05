package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowGitHubReferencesConfig(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
	}{
		{
			name: "allow current repo only",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"allow-github-references": []any{"repo"},
					"create-issue":            map[string]any{},
				},
			},
			expected: []string{"repo"},
		},
		{
			name: "allow multiple repos",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"allow-github-references": []any{"repo", "org/repo2", "org/repo3"},
					"create-issue":            map[string]any{},
				},
			},
			expected: []string{"repo", "org/repo2", "org/repo3"},
		},
		{
			name: "no restrictions (empty array)",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"allow-github-references": []any{},
					"create-issue":            map[string]any{},
				},
			},
			expected: nil, // Empty array results in nil
		},
		{
			name: "no allow-github-references field",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(false, "", "1.0.0")
			config := c.extractSafeOutputsConfig(tt.frontmatter)
			require.NotNil(t, config, "extractSafeOutputsConfig() should not return nil")

			if tt.expected == nil {
				assert.Nil(t, config.AllowGitHubReferences, "AllowGitHubReferences should be nil")
			} else {
				require.NotNil(t, config.AllowGitHubReferences, "AllowGitHubReferences should not be nil")
				assert.Equal(t, tt.expected, config.AllowGitHubReferences, "AllowGitHubReferences should match expected")
			}
		})
	}
}
