//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeImportedAPMPackages(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result, err := mergeImportedAPMPackages(nil)
		require.NoError(t, err, "Should not error on nil input")
		assert.Nil(t, result, "Should return nil for nil input")
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		result, err := mergeImportedAPMPackages([]string{})
		require.NoError(t, err, "Should not error on empty input")
		assert.Nil(t, result, "Should return nil for empty input")
	})

	t.Run("single array config", func(t *testing.T) {
		configs := []string{`["microsoft/apm-sample-package","github/awesome-copilot/skills/review"]`}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, []string{"microsoft/apm-sample-package", "github/awesome-copilot/skills/review"}, result.Packages)
		assert.Empty(t, result.GitHubToken, "GitHubToken should be empty when not set")
		assert.Nil(t, result.GitHubApp, "GitHubApp should be nil when not set")
		assert.False(t, result.Isolated, "Isolated should be false when not set")
	})

	t.Run("single object config with packages key", func(t *testing.T) {
		configs := []string{`{"packages":["microsoft/apm-sample-package"]}`}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, []string{"microsoft/apm-sample-package"}, result.Packages)
	})

	t.Run("object config with github-token secrets expression", func(t *testing.T) {
		configs := []string{`{"packages":["org/pkg"],"github-token":"${{ secrets.MY_TOKEN }}"}`}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, []string{"org/pkg"}, result.Packages)
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", result.GitHubToken, "GitHubToken should be set from secrets expression")
	})

	t.Run("object config with unsubstituted import-inputs expression is stripped", func(t *testing.T) {
		// When the importer omits github-token, the expression stays unsubstituted
		configs := []string{`{"packages":["org/pkg"],"github-token":"${{ github.aw.import-inputs.github-token }}"}`}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, []string{"org/pkg"}, result.Packages)
		assert.Empty(t, result.GitHubToken, "Unsubstituted github-token expression should be stripped")
	})

	t.Run("object config with isolated true", func(t *testing.T) {
		configs := []string{`{"packages":["org/pkg"],"isolated":true}`}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.True(t, result.Isolated, "Isolated should be true")
	})

	t.Run("multiple configs merge packages and deduplicate", func(t *testing.T) {
		configs := []string{
			`["microsoft/apm-sample-package","shared/skill-a"]`,
			`["shared/skill-a","shared/skill-b"]`,
		}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		// skill-a deduplicated; order is first-seen
		assert.Equal(t, []string{"microsoft/apm-sample-package", "shared/skill-a", "shared/skill-b"}, result.Packages)
	})

	t.Run("multiple configs use first-wins for github-token", func(t *testing.T) {
		configs := []string{
			`{"packages":["pkg-a"],"github-token":"${{ secrets.FIRST_TOKEN }}"}`,
			`{"packages":["pkg-b"],"github-token":"${{ secrets.SECOND_TOKEN }}"}`,
		}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, "${{ secrets.FIRST_TOKEN }}", result.GitHubToken, "First token should win")
		assert.Equal(t, []string{"pkg-a", "pkg-b"}, result.Packages)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		configs := []string{`not-valid-json`}
		result, err := mergeImportedAPMPackages(configs)
		require.Error(t, err, "Should return error for invalid JSON")
		assert.Nil(t, result, "Should return nil result on error")
	})

	t.Run("empty string config is skipped", func(t *testing.T) {
		configs := []string{"", `["org/pkg"]`, ""}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		require.NotNil(t, result, "Should return non-nil result")
		assert.Equal(t, []string{"org/pkg"}, result.Packages)
	})

	t.Run("all empty configs returns nil", func(t *testing.T) {
		configs := []string{"", ""}
		result, err := mergeImportedAPMPackages(configs)
		require.NoError(t, err, "Should not error")
		assert.Nil(t, result, "Should return nil when all configs are empty")
	})
}

func TestIsUnsubstitutedImportExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "unsubstituted import-inputs expression",
			input:    "${{ github.aw.import-inputs.github-token }}",
			expected: true,
		},
		{
			name:     "unsubstituted import-inputs with spaces",
			input:    "${{  github.aw.import-inputs.isolated  }}",
			expected: true,
		},
		{
			name:     "secrets expression is not stripped",
			input:    "${{ secrets.MY_TOKEN }}",
			expected: false,
		},
		{
			name:     "github context expression is not stripped",
			input:    "${{ github.repository }}",
			expected: false,
		},
		{
			name:     "regular string",
			input:    "v0.8.0",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUnsubstitutedImportExpression(tt.input)
			assert.Equal(t, tt.expected, result, "isUnsubstitutedImportExpression(%q)", tt.input)
		})
	}
}
