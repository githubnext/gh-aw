//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSafeOutputsMax(t *testing.T) {
	t.Run("nil config is valid", func(t *testing.T) {
		err := validateSafeOutputsMax(nil)
		assert.NoError(t, err, "nil config should be valid")
	})

	t.Run("config with no max fields is valid", func(t *testing.T) {
		config := &SafeOutputsConfig{}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "config with no max fields should be valid")
	})

	t.Run("max of 1 is valid", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "max: 1 should be valid")
	})

	t.Run("max of 5 is valid", func(t *testing.T) {
		config := &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("5")},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "max: 5 should be valid")
	})

	t.Run("max of -1 is valid (unlimited)", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("-1")},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "max: -1 should be valid (means unlimited per spec)")
	})

	t.Run("max of 0 is invalid", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("0")},
			},
		}
		err := validateSafeOutputsMax(config)
		require.Error(t, err, "max: 0 should be invalid")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error should explain valid values")
		assert.Contains(t, err.Error(), "add-comment", "error should mention the field name")
	})

	t.Run("max of -2 is invalid", func(t *testing.T) {
		config := &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("-2")},
			},
		}
		err := validateSafeOutputsMax(config)
		require.Error(t, err, "max: -2 should be invalid")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error should explain valid values")
	})

	t.Run("max as GitHub Actions expression is skipped", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("${{ inputs.max }}")},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "GitHub Actions expression should be skipped")
	})

	t.Run("nil max is valid", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: nil},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "nil max should be valid")
	})

	t.Run("dispatch_repository tool max of 0 is invalid", func(t *testing.T) {
		maxVal := "0"
		config := &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"my-tool": {Max: &maxVal},
				},
			},
		}
		err := validateSafeOutputsMax(config)
		require.Error(t, err, "dispatch_repository max: 0 should be invalid")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error should explain valid values")
		assert.Contains(t, err.Error(), "my-tool", "error should mention the tool name")
		assert.Contains(t, err.Error(), "dispatch_repository", "error should use underscore form")
	})

	t.Run("dispatch_repository tool max of -1 is valid (unlimited)", func(t *testing.T) {
		maxVal := "-1"
		config := &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"my-tool": {Max: &maxVal},
				},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "dispatch_repository max: -1 should be valid")
	})

	t.Run("dispatch_repository tool max of 1 is valid", func(t *testing.T) {
		maxVal := "1"
		config := &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"my-tool": {Max: &maxVal},
				},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "dispatch_repository max: 1 should be valid")
	})

	t.Run("dispatch_repository tool max as expression is skipped", func(t *testing.T) {
		maxVal := "${{ inputs.max }}"
		config := &SafeOutputsConfig{
			DispatchRepository: &DispatchRepositoryConfig{
				Tools: map[string]*DispatchRepositoryToolConfig{
					"my-tool": {Max: &maxVal},
				},
			},
		}
		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "GitHub Actions expression for dispatch_repository should be skipped")
	})

	t.Run("multiple configs with one invalid returns error", func(t *testing.T) {
		config := &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("3")},
			},
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("0")},
			},
		}
		err := validateSafeOutputsMax(config)
		require.Error(t, err, "config with one invalid max should return error")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error should explain valid values")
	})
}

func TestValidateSafeOutputsMaxIntegration(t *testing.T) {
	compiler := &Compiler{}

	t.Run("max of 0 is rejected during config extraction via compiler", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-comment": map[string]any{
					"max": 0,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		require.NotNil(t, config, "config should be extracted")

		err := validateSafeOutputsMax(config)
		require.Error(t, err, "max: 0 should fail validation")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error message should explain valid values")
	})

	t.Run("max of -2 is rejected during config extraction via compiler", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"max": -2,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		require.NotNil(t, config, "config should be extracted")

		err := validateSafeOutputsMax(config)
		require.Error(t, err, "max: -2 should fail validation")
		assert.Contains(t, err.Error(), "max must be a positive integer or -1", "error message should explain valid values")
	})

	t.Run("max of -1 passes validation (unlimited)", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-comment": map[string]any{
					"max": -1,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		require.NotNil(t, config, "config should be extracted")

		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "max: -1 should pass validation (unlimited per spec)")
	})

	t.Run("max of 1 passes validation", func(t *testing.T) {
		frontmatter := map[string]any{
			"safe-outputs": map[string]any{
				"add-comment": map[string]any{
					"max": 1,
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		require.NotNil(t, config, "config should be extracted")

		err := validateSafeOutputsMax(config)
		assert.NoError(t, err, "max: 1 should pass validation")
	})
}
