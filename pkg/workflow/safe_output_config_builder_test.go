package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildHandlerConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected map[string]any
	}{
		{
			name: "CreateIssuesConfig with all fields",
			input: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
				},
				TitlePrefix:    "[AI] ",
				Labels:         []string{"bug", "ai"},
				AllowedLabels:  []string{"bug", "feature"},
				Assignees:      []string{"user1", "user2"},
				TargetRepoSlug: "owner/repo",
				AllowedRepos:   []string{"org/repo1", "org/repo2"},
				Expires:        72,
			},
			expected: map[string]any{
				"max":            int64(5),
				"title_prefix":   "[AI] ",
				"labels":         []any{"bug", "ai"},
				"allowed_labels": []any{"bug", "feature"},
				"assignees":      []any{"user1", "user2"},
				"target-repo":    "owner/repo",
				"allowed_repos":  []any{"org/repo1", "org/repo2"},
				"expires":        int64(72),
			},
		},
		{
			name: "AddCommentsConfig with target and hide older",
			input: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 3,
				},
				Target:            "issue",
				HideOlderComments: true,
				TargetRepoSlug:    "owner/repo",
				AllowedRepos:      []string{"owner/repo2"},
			},
			expected: map[string]any{
				"max":                 int64(3),
				"target":              "issue",
				"hide_older_comments": true,
				"target-repo":         "owner/repo",
				"allowed_repos":       []any{"owner/repo2"},
			},
		},
		{
			name: "Config with zero values omitted",
			input: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 0, // Should be omitted
				},
				TitlePrefix: "",         // Should be omitted
				Labels:      []string{}, // Should be omitted
			},
			expected: map[string]any{},
		},
		{
			name: "UpdateIssuesConfig with boolean pointers",
			input: &UpdateIssuesConfig{
				UpdateEntityConfig: UpdateEntityConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 1,
					},
					SafeOutputTargetConfig: SafeOutputTargetConfig{
						Target: "issue",
					},
				},
				Status: testBoolPtr(true),
				Title:  testBoolPtr(true),
				Body:   testBoolPtr(true),
			},
			expected: map[string]any{
				"max":          int64(1),
				"target":       "issue",
				"allow_status": true,
				"allow_title":  true,
				"allow_body":   true,
			},
		},
		{
			name: "CloseEntityConfig with required fields",
			input: &CloseEntityConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					Target: "issue",
				},
				SafeOutputFilterConfig: SafeOutputFilterConfig{
					RequiredLabels:      []string{"stale"},
					RequiredTitlePrefix: "[OLD]",
				},
			},
			expected: map[string]any{
				"max":                   int64(10),
				"target":                "issue",
				"required_labels":       []any{"stale"},
				"required_title_prefix": "[OLD]",
			},
		},
		{
			name:     "Nil pointer returns empty config",
			input:    (*CreateIssuesConfig)(nil),
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHandlerConfig(tt.input)
			assert.Equal(t, tt.expected, result, "Handler config should match expected")
		})
	}
}

func TestBuildSafeOutputConfigs(t *testing.T) {
	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		expectedKeys  []string
		validateField func(t *testing.T, config map[string]map[string]any)
	}{
		{
			name: "Single handler - CreateIssues",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 5,
					},
					TitlePrefix:   "[AI] ",
					AllowedLabels: []string{"bug"},
				},
			},
			expectedKeys: []string{"create_issue"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				issueConfig, exists := config["create_issue"]
				require.True(t, exists, "create_issue should exist")
				assert.Equal(t, int64(5), issueConfig["max"])
				assert.Equal(t, "[AI] ", issueConfig["title_prefix"])
				assert.Equal(t, []any{"bug"}, issueConfig["allowed_labels"])
			},
		},
		{
			name: "Multiple handlers",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 5,
					},
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 3,
					},
					Target: "issue",
				},
				AddLabels: &AddLabelsConfig{
					Allowed: []string{"bug", "feature"},
				},
			},
			expectedKeys: []string{"create_issue", "add_comment", "add_labels"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				assert.Len(t, config, 3, "Should have 3 handlers")

				issueConfig := config["create_issue"]
				assert.Equal(t, int64(5), issueConfig["max"])

				commentConfig := config["add_comment"]
				assert.Equal(t, int64(3), commentConfig["max"])
				assert.Equal(t, "issue", commentConfig["target"])

				labelConfig := config["add_labels"]
				assert.Equal(t, []any{"bug", "feature"}, labelConfig["allowed"])
			},
		},
		{
			name: "CreatePullRequests with base_branch customization",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 1,
					},
					TitlePrefix: "[PR] ",
				},
			},
			expectedKeys: []string{"create_pull_request"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				prConfig, exists := config["create_pull_request"]
				require.True(t, exists)
				assert.Equal(t, "${{ github.ref_name }}", prConfig["base_branch"], "Should add base_branch")
				assert.Equal(t, 1024, prConfig["max_patch_size"], "Should add default max_patch_size")
			},
		},
		{
			name: "UpdatePullRequests with default allow fields",
			safeOutputs: &SafeOutputsConfig{
				UpdatePullRequests: &UpdatePullRequestsConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 1,
						},
						SafeOutputTargetConfig: SafeOutputTargetConfig{
							Target: "pr",
						},
					},
				},
			},
			expectedKeys: []string{"update_pull_request"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				prConfig, exists := config["update_pull_request"]
				require.True(t, exists)
				assert.Equal(t, true, prConfig["allow_title"], "Should default to true")
				assert.Equal(t, true, prConfig["allow_body"], "Should default to true")
			},
		},
		{
			name: "Custom MaximumPatchSize",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: 1,
					},
				},
				MaximumPatchSize: 2048,
			},
			expectedKeys: []string{"create_pull_request"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				prConfig := config["create_pull_request"]
				assert.Equal(t, 2048, prConfig["max_patch_size"], "Should use custom max_patch_size")
			},
		},
		{
			name:         "Nil SafeOutputsConfig returns empty",
			safeOutputs:  nil,
			expectedKeys: []string{},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				assert.Empty(t, config)
			},
		},
		{
			name: "Empty configs are included (handler enabled by presence)",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					// All zero values
				},
			},
			expectedKeys: []string{"create_issue"},
			validateField: func(t *testing.T, config map[string]map[string]any) {
				assert.Contains(t, config, "create_issue", "Empty config should still be included")
				issueConfig := config["create_issue"]
				assert.Empty(t, issueConfig, "Config should be empty map")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSafeOutputConfigs(tt.safeOutputs)

			// Check expected keys
			for _, key := range tt.expectedKeys {
				assert.Contains(t, result, key, "Should contain key %s", key)
			}

			// Additional validation
			if tt.validateField != nil {
				tt.validateField(t, result)
			}
		})
	}
}

func TestBuildHandlerConfig_EdgeCases(t *testing.T) {
	t.Run("Non-struct input", func(t *testing.T) {
		result := buildHandlerConfig("not a struct")
		assert.Empty(t, result, "Should return empty for non-struct")
	})

	t.Run("Struct with no mapped fields", func(t *testing.T) {
		type UnknownConfig struct {
			UnmappedField string
		}
		result := buildHandlerConfig(&UnknownConfig{UnmappedField: "value"})
		assert.Empty(t, result, "Should skip unmapped fields")
	})

	t.Run("Struct with embedded BaseSafeOutputConfig", func(t *testing.T) {
		config := &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Max:         10,
				GitHubToken: "token-value",
			},
			TitlePrefix: "[Test]",
		}
		result := buildHandlerConfig(config)
		assert.Equal(t, int64(10), result["max"], "Should extract Max from embedded struct")
		assert.Equal(t, "token-value", result["github-token"], "Should extract GitHubToken from embedded struct")
		assert.Equal(t, "[Test]", result["title_prefix"], "Should extract direct fields")
	})
}
