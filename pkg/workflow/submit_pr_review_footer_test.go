//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEffectiveFooterString(t *testing.T) {
	t.Run("returns local footer when set", func(t *testing.T) {
		local := "if-body"
		result := getEffectiveFooterString(&local, nil)
		require.NotNil(t, result, "Should return local footer")
		assert.Equal(t, "if-body", *result, "Should return local footer value")
	})

	t.Run("local footer takes precedence over global", func(t *testing.T) {
		local := "none"
		globalTrue := true
		result := getEffectiveFooterString(&local, &globalTrue)
		require.NotNil(t, result, "Should return local footer")
		assert.Equal(t, "none", *result, "Local should override global")
	})

	t.Run("converts global true to always", func(t *testing.T) {
		globalTrue := true
		result := getEffectiveFooterString(nil, &globalTrue)
		require.NotNil(t, result, "Should convert global bool")
		assert.Equal(t, "always", *result, "Global true should map to always")
	})

	t.Run("converts global false to none", func(t *testing.T) {
		globalFalse := false
		result := getEffectiveFooterString(nil, &globalFalse)
		require.NotNil(t, result, "Should convert global bool")
		assert.Equal(t, "none", *result, "Global false should map to none")
	})

	t.Run("returns nil when both are nil", func(t *testing.T) {
		result := getEffectiveFooterString(nil, nil)
		assert.Nil(t, result, "Should return nil when neither is set")
	})
}

func TestSubmitPRReviewFooterConfig(t *testing.T) {
	t.Run("parses footer string values", func(t *testing.T) {
		testCases := []struct {
			name     string
			value    string
			expected string
		}{
			{name: "always", value: "always", expected: "always"},
			{name: "none", value: "none", expected: "none"},
			{name: "if-body", value: "if-body", expected: "if-body"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				compiler := NewCompiler()
				outputMap := map[string]any{
					"submit-pull-request-review": map[string]any{
						"footer": tc.value,
					},
				}

				config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
				require.NotNil(t, config, "Config should be parsed")
				require.NotNil(t, config.Footer, "Footer should be set")
				assert.Equal(t, tc.expected, *config.Footer, "Footer value should match")
			})
		}
	})

	t.Run("parses footer boolean values", func(t *testing.T) {
		testCases := []struct {
			name     string
			value    bool
			expected string
		}{
			{name: "true maps to always", value: true, expected: "always"},
			{name: "false maps to none", value: false, expected: "none"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				compiler := NewCompiler()
				outputMap := map[string]any{
					"submit-pull-request-review": map[string]any{
						"footer": tc.value,
					},
				}

				config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
				require.NotNil(t, config, "Config should be parsed")
				require.NotNil(t, config.Footer, "Footer value should be set")
				assert.Equal(t, tc.expected, *config.Footer, "Footer value should be mapped from boolean")
			})
		}
	})

	t.Run("ignores invalid footer values", func(t *testing.T) {
		compiler := NewCompiler()
		outputMap := map[string]any{
			"submit-pull-request-review": map[string]any{
				"footer": "invalid-value",
			},
		}

		config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
		require.NotNil(t, config, "Config should be parsed")
		assert.Nil(t, config.Footer, "Invalid footer value should be ignored")
	})

	t.Run("footer not set when omitted", func(t *testing.T) {
		compiler := NewCompiler()
		outputMap := map[string]any{
			"submit-pull-request-review": map[string]any{
				"max": 1,
			},
		}

		config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
		require.NotNil(t, config, "Config should be parsed")
		assert.Nil(t, config.Footer, "Footer should be nil when not configured")
	})

	t.Run("parses target field", func(t *testing.T) {
		compiler := NewCompiler()
		outputMap := map[string]any{
			"submit-pull-request-review": map[string]any{
				"max":    1,
				"target": "42",
			},
		}

		config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
		require.NotNil(t, config, "Config should be parsed")
		assert.Equal(t, "42", config.Target, "Target should be parsed")
	})

	t.Run("target empty when omitted", func(t *testing.T) {
		compiler := NewCompiler()
		outputMap := map[string]any{
			"submit-pull-request-review": map[string]any{
				"max": 1,
			},
		}

		config := compiler.parseSubmitPullRequestReviewConfig(outputMap)
		require.NotNil(t, config, "Config should be parsed")
		assert.Empty(t, config.Target, "Target should be empty when not configured")
	})
}

func TestCreatePRReviewCommentNoFooter(t *testing.T) {
	t.Run("create-pull-request-review-comment does not have footer field", func(t *testing.T) {
		compiler := NewCompiler()
		outputMap := map[string]any{
			"create-pull-request-review-comment": map[string]any{
				"side": "RIGHT",
			},
		}

		config := compiler.parsePullRequestReviewCommentsConfig(outputMap)
		require.NotNil(t, config, "Config should be parsed")
		// CreatePullRequestReviewCommentsConfig no longer has a Footer field;
		// footer control belongs on submit-pull-request-review
	})
}

func TestSubmitPRReviewFooterInHandlerConfig(t *testing.T) {
	t.Run("footer included in submit_pull_request_review handler config", func(t *testing.T) {
		compiler := NewCompiler()
		footerValue := "if-body"
		workflowData := &WorkflowData{
			Name: "Test",
			SafeOutputs: &SafeOutputsConfig{
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Footer:               &footerValue,
				},
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("10")},
					Side:                 "RIGHT",
				},
			},
		}

		var steps []string
		compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
		require.NotEmpty(t, steps, "Steps should not be empty")

		stepsContent := strings.Join(steps, "")
		require.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG")

		for _, step := range steps {
			if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
				parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
				if len(parts) == 2 {
					jsonStr := strings.TrimSpace(parts[1])
					jsonStr = strings.Trim(jsonStr, "\"")
					jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
					var handlerConfig map[string]any
					err := json.Unmarshal([]byte(jsonStr), &handlerConfig)
					require.NoError(t, err, "Should unmarshal handler config")

					submitConfig, ok := handlerConfig["submit_pull_request_review"].(map[string]any)
					require.True(t, ok, "submit_pull_request_review config should exist")
					assert.Equal(t, "if-body", submitConfig["footer"], "Footer should be in submit handler config")

					// create_pull_request_review_comment should NOT have footer
					reviewCommentConfig, ok := handlerConfig["create_pull_request_review_comment"].(map[string]any)
					require.True(t, ok, "create_pull_request_review_comment config should exist")
					_, hasFooter := reviewCommentConfig["footer"]
					assert.False(t, hasFooter, "Footer should not be in review comment handler config")
				}
			}
		}
	})

	t.Run("footer not in handler config when not set", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			Name: "Test",
			SafeOutputs: &SafeOutputsConfig{
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				},
			},
		}

		var steps []string
		compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
		require.NotEmpty(t, steps, "Steps should not be empty")

		stepsContent := strings.Join(steps, "")
		require.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG")

		for _, step := range steps {
			if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
				parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
				if len(parts) == 2 {
					jsonStr := strings.TrimSpace(parts[1])
					jsonStr = strings.Trim(jsonStr, "\"")
					jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
					var handlerConfig map[string]any
					err := json.Unmarshal([]byte(jsonStr), &handlerConfig)
					require.NoError(t, err, "Should unmarshal handler config")

					submitConfig, ok := handlerConfig["submit_pull_request_review"].(map[string]any)
					require.True(t, ok, "submit_pull_request_review config should exist")
					_, hasFooter := submitConfig["footer"]
					assert.False(t, hasFooter, "Footer should not be in handler config when not set")
				}
			}
		}
	})

	t.Run("target included in submit_pull_request_review handler config when set", func(t *testing.T) {
		compiler := NewCompiler()
		targetValue := "123"
		workflowData := &WorkflowData{
			Name: "Test",
			SafeOutputs: &SafeOutputsConfig{
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Target:               targetValue,
				},
			},
		}

		var steps []string
		compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
		require.NotEmpty(t, steps, "Steps should not be empty")

		stepsContent := strings.Join(steps, "")
		require.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG")

		for _, step := range steps {
			if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
				parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
				if len(parts) == 2 {
					jsonStr := strings.TrimSpace(parts[1])
					jsonStr = strings.Trim(jsonStr, "\"")
					jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
					var handlerConfig map[string]any
					err := json.Unmarshal([]byte(jsonStr), &handlerConfig)
					require.NoError(t, err, "Should unmarshal handler config")

					submitConfig, ok := handlerConfig["submit_pull_request_review"].(map[string]any)
					require.True(t, ok, "submit_pull_request_review config should exist")
					assert.Equal(t, "123", submitConfig["target"], "Target should be in submit handler config")
				}
			}
		}
	})
}
