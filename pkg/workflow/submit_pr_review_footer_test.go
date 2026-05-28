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

type submitPRReviewConfigCase struct {
	name      string
	outputMap map[string]any
	assert    func(t *testing.T, config *SubmitPullRequestReviewConfig)
}

func TestSubmitPRReviewFooterConfig(t *testing.T) {
	t.Run("footer", func(t *testing.T) {
		testSubmitPRReviewConfigCases(t, testSubmitPRReviewFooterCases())
	})
	t.Run("targets", func(t *testing.T) {
		testSubmitPRReviewConfigCases(t, testSubmitPRReviewTargetCases())
	})
	t.Run("allowed events", func(t *testing.T) {
		testSubmitPRReviewConfigCases(t, testSubmitPRReviewAllowedEventCases())
	})
	t.Run("misc", func(t *testing.T) {
		testSubmitPRReviewConfigCases(t, testSubmitPRReviewMiscCases())
	})
}

func testSubmitPRReviewConfigCases(t *testing.T, tests []submitPRReviewConfigCase) {
	t.Helper()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			compiler := NewCompiler()
			config := compiler.parseSubmitPullRequestReviewConfig(tc.outputMap)
			tc.assert(t, config)
		})
	}
}

func testSubmitPRReviewFooterCases() []submitPRReviewConfigCase {
	return []submitPRReviewConfigCase{
		{name: "parses footer always", outputMap: submitPRReviewOutputMap("footer", "always"), assert: assertFooterValue("always")},
		{name: "parses footer none", outputMap: submitPRReviewOutputMap("footer", "none"), assert: assertFooterValue("none")},
		{name: "parses footer if-body", outputMap: submitPRReviewOutputMap("footer", "if-body"), assert: assertFooterValue("if-body")},
		{name: "parses footer bool true", outputMap: submitPRReviewOutputMap("footer", true), assert: assertFooterValue("always")},
		{name: "parses footer bool false", outputMap: submitPRReviewOutputMap("footer", false), assert: assertFooterValue("none")},
		{name: "ignores invalid footer", outputMap: submitPRReviewOutputMap("footer", "invalid-value"), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Nil(t, config.Footer, "Invalid footer value should be ignored")
		}},
		{name: "footer omitted", outputMap: submitPRReviewOutputMap("max", 1), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Nil(t, config.Footer, "Footer should be nil when not configured")
		}},
	}
}

func testSubmitPRReviewTargetCases() []submitPRReviewConfigCase {
	return []submitPRReviewConfigCase{
		{name: "parses target", outputMap: submitPRReviewOutputMap("target", "42"), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, "42", config.Target, "Target should be parsed")
		}},
		{name: "target omitted", outputMap: submitPRReviewOutputMap("max", 1), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Empty(t, config.Target, "Target should be empty when not configured")
		}},
		{name: "parses target repo", outputMap: submitPRReviewOutputMap("target-repo", "consumer-org/consumer-repo"), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, "consumer-org/consumer-repo", config.TargetRepoSlug, "TargetRepoSlug should be parsed")
		}},
		{name: "target repo omitted", outputMap: submitPRReviewOutputMap("max", 1), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Empty(t, config.TargetRepoSlug, "TargetRepoSlug should be empty when not configured")
		}},
		{name: "parses allowed repos", outputMap: submitPRReviewOutputMapMany(map[string]any{"max": 1, "target-repo": "consumer-org/consumer-repo", "allowed-repos": []any{"consumer-org/other-repo", "consumer-org/another-repo"}}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, []string{"consumer-org/other-repo", "consumer-org/another-repo"}, config.AllowedRepos, "AllowedRepos should be parsed")
		}},
		{name: "returns nil for wildcard target repo", outputMap: submitPRReviewOutputMapMany(map[string]any{"max": 1, "target-repo": "*"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			assert.Nil(t, config, "Config should be nil for wildcard target-repo")
		}},
	}
}

func testSubmitPRReviewAllowedEventCases() []submitPRReviewConfigCase {
	return []submitPRReviewConfigCase{
		{name: "parses allowed events", outputMap: submitPRReviewOutputMap("allowed-events", []any{"COMMENT", "REQUEST_CHANGES"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, []string{"COMMENT", "REQUEST_CHANGES"}, config.AllowedEvents, "AllowedEvents should be parsed")
		}},
		{name: "normalizes allowed events", outputMap: submitPRReviewOutputMap("allowed-events", []any{"comment", "approve"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, []string{"COMMENT", "APPROVE"}, config.AllowedEvents, "AllowedEvents should be normalized to uppercase")
		}},
		{name: "ignores invalid values when mixed", outputMap: submitPRReviewOutputMap("allowed-events", []any{"COMMENT", "INVALID_EVENT", "APPROVE"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed when at least one valid event remains")
			assert.Equal(t, []string{"COMMENT", "APPROVE"}, config.AllowedEvents, "Invalid events should be ignored while valid ones remain")
		}},
		{name: "returns nil when all events invalid", outputMap: submitPRReviewOutputMap("allowed-events", []any{"INVALID_EVENT", "ANOTHER_BAD_VALUE"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			assert.Nil(t, config, "Config should be nil when all allowed-events values are invalid")
		}},
		{name: "returns nil when allowed events not list", outputMap: submitPRReviewOutputMap("allowed-events", "COMMENT"), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			assert.Nil(t, config, "Config should be nil when allowed-events is not a list")
		}},
		{name: "allowed events omitted", outputMap: submitPRReviewOutputMap("max", 1), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Empty(t, config.AllowedEvents, "AllowedEvents should be empty when not configured")
		}},
		{name: "parses all valid event types", outputMap: submitPRReviewOutputMap("allowed-events", []any{"APPROVE", "COMMENT", "REQUEST_CHANGES"}), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.Equal(t, []string{"APPROVE", "COMMENT", "REQUEST_CHANGES"}, config.AllowedEvents, "All three event types should be parsed")
		}},
	}
}

func testSubmitPRReviewMiscCases() []submitPRReviewConfigCase {
	return []submitPRReviewConfigCase{
		{name: "parses supersede older reviews", outputMap: submitPRReviewOutputMap("supersede-older-reviews", true), assert: func(t *testing.T, config *SubmitPullRequestReviewConfig) {
			require.NotNil(t, config, "Config should be parsed")
			assert.True(t, config.SupersedeOlderReviews, "SupersedeOlderReviews should be parsed")
		}},
	}
}

func submitPRReviewOutputMap(field string, value any) map[string]any {
	return submitPRReviewOutputMapMany(map[string]any{"max": 1, field: value})
}

func submitPRReviewOutputMapMany(values map[string]any) map[string]any {
	return map[string]any{"submit-pull-request-review": values}
}

func assertFooterValue(expected string) func(t *testing.T, config *SubmitPullRequestReviewConfig) {
	return func(t *testing.T, config *SubmitPullRequestReviewConfig) {
		require.NotNil(t, config, "Config should be parsed")
		require.NotNil(t, config.Footer, "Footer should be set")
		assert.Equal(t, expected, *config.Footer, "Footer value should match")
	}
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

type submitPRReviewHandlerCase struct {
	name     string
	workflow *WorkflowData
	assert   func(t *testing.T, handlerConfig map[string]any)
}

func TestSubmitPRReviewFooterInHandlerConfig(t *testing.T) {
	testSubmitPRReviewHandlerCases(t, testSubmitPRReviewHandlerConfigCases())
}

func testSubmitPRReviewHandlerConfigCases() []submitPRReviewHandlerCase {
	footerValue := "if-body"
	return []submitPRReviewHandlerCase{
		{name: "footer included in submit handler config", workflow: submitPRReviewWorkflowData(func(cfg *SubmitPullRequestReviewConfig) { cfg.Footer = &footerValue }, true), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			assert.Equal(t, "if-body", submitConfig["footer"], "Footer should be in submit handler config")
			reviewCommentConfig, ok := handlerConfig["create_pull_request_review_comment"].(map[string]any)
			require.True(t, ok, "create_pull_request_review_comment config should exist")
			_, hasFooter := reviewCommentConfig["footer"]
			assert.False(t, hasFooter, "Footer should not be in review comment handler config")
		}},
		{name: "footer omitted from handler config", workflow: submitPRReviewWorkflowData(nil, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			_, hasFooter := submitConfig["footer"]
			assert.False(t, hasFooter, "Footer should not be in handler config when not set")
		}},
		{name: "target included in submit handler config", workflow: submitPRReviewWorkflowData(func(cfg *SubmitPullRequestReviewConfig) { cfg.SafeOutputTargetConfig.Target = "123" }, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			assert.Equal(t, "123", submitConfig["target"], "Target should be in submit handler config")
		}},
		{name: "target repo included in submit handler config", workflow: submitPRReviewWorkflowData(func(cfg *SubmitPullRequestReviewConfig) {
			cfg.SafeOutputTargetConfig.TargetRepoSlug = "consumer-org/consumer-repo"
		}, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			assert.Equal(t, "consumer-org/consumer-repo", submitConfig["target-repo"], "Target-repo should be in submit handler config")
		}},
		{name: "allowed events included in submit handler config", workflow: submitPRReviewWorkflowData(func(cfg *SubmitPullRequestReviewConfig) { cfg.AllowedEvents = []string{"COMMENT", "REQUEST_CHANGES"} }, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			allowedEvents, ok := submitConfig["allowed_events"].([]any)
			require.True(t, ok, "allowed_events should be present in handler config")
			require.Len(t, allowedEvents, 2, "allowed_events should have 2 entries")
			assert.Equal(t, "COMMENT", allowedEvents[0], "First allowed event should be COMMENT")
			assert.Equal(t, "REQUEST_CHANGES", allowedEvents[1], "Second allowed event should be REQUEST_CHANGES")
		}},
		{name: "allowed events omitted from submit handler config", workflow: submitPRReviewWorkflowData(nil, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			_, hasAllowedEvents := submitConfig["allowed_events"]
			assert.False(t, hasAllowedEvents, "allowed_events should not be in handler config when not set")
		}},
		{name: "supersede older reviews included when true", workflow: submitPRReviewWorkflowData(func(cfg *SubmitPullRequestReviewConfig) { cfg.SupersedeOlderReviews = true }, false), assert: func(t *testing.T, handlerConfig map[string]any) {
			submitConfig := requireSubmitPRReviewHandlerConfig(t, handlerConfig)
			supersedeOlderReviews, hasSupersedeOlderReviews := submitConfig["supersede_older_reviews"].(bool)
			require.True(t, hasSupersedeOlderReviews, "supersede_older_reviews should be in handler config when true")
			assert.True(t, supersedeOlderReviews, "supersede_older_reviews should be true")
		}},
	}
}

func testSubmitPRReviewHandlerCases(t *testing.T, tests []submitPRReviewHandlerCase) {
	t.Helper()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			handlerConfig := extractSubmitPRReviewHandlerConfig(t, tc.workflow)
			tc.assert(t, handlerConfig)
		})
	}
}

func submitPRReviewWorkflowData(update func(*SubmitPullRequestReviewConfig), includeReviewComment bool) *WorkflowData {
	workflow := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			SubmitPullRequestReview: &SubmitPullRequestReviewConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
			},
		},
	}
	if includeReviewComment {
		workflow.SafeOutputs.CreatePullRequestReviewComments = &CreatePullRequestReviewCommentsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("10")},
			Side:                 "RIGHT",
		}
	}
	if update != nil {
		update(workflow.SafeOutputs.SubmitPullRequestReview)
	}
	return workflow
}

func extractSubmitPRReviewHandlerConfig(t *testing.T, workflowData *WorkflowData) map[string]any {
	t.Helper()
	compiler := NewCompiler()
	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)
	require.NotEmpty(t, steps, "Steps should not be empty")
	require.Contains(t, strings.Join(steps, ""), "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG")

	for _, step := range steps {
		if !strings.Contains(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			continue
		}
		parts := strings.Split(step, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ")
		if len(parts) != 2 {
			continue
		}
		jsonStr := strings.TrimSpace(parts[1])
		jsonStr = strings.Trim(jsonStr, "\"")
		jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")
		var handlerConfig map[string]any
		err := json.Unmarshal([]byte(jsonStr), &handlerConfig)
		require.NoError(t, err, "Should unmarshal handler config")
		return handlerConfig
	}
	t.Fatal("Expected GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG to be present")
	return nil
}

func requireSubmitPRReviewHandlerConfig(t *testing.T, handlerConfig map[string]any) map[string]any {
	t.Helper()
	submitConfig, ok := handlerConfig["submit_pull_request_review"].(map[string]any)
	require.True(t, ok, "submit_pull_request_review config should exist")
	return submitConfig
}
