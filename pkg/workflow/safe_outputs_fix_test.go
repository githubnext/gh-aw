//go:build !integration

package workflow

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// hasSafeOutputType new switch cases
// ========================================

// TestHasSafeOutputTypeNewKeys verifies that the 11 operation types added to hasSafeOutputType
// are correctly detected. These were previously silently returning false, causing import
// conflict detection to pass through conflicts for those types.
func TestHasSafeOutputTypeNewKeys(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		config *SafeOutputsConfig
	}{
		{
			name:   "update-discussion",
			key:    "update-discussion",
			config: &SafeOutputsConfig{UpdateDiscussions: &UpdateDiscussionsConfig{}},
		},
		{
			name:   "mark-pull-request-as-ready-for-review",
			key:    "mark-pull-request-as-ready-for-review",
			config: &SafeOutputsConfig{MarkPullRequestAsReadyForReview: &MarkPullRequestAsReadyForReviewConfig{}},
		},
		{
			name:   "autofix-code-scanning-alert",
			key:    "autofix-code-scanning-alert",
			config: &SafeOutputsConfig{AutofixCodeScanningAlert: &AutofixCodeScanningAlertConfig{}},
		},
		{
			name:   "assign-to-user",
			key:    "assign-to-user",
			config: &SafeOutputsConfig{AssignToUser: &AssignToUserConfig{}},
		},
		{
			name:   "unassign-from-user",
			key:    "unassign-from-user",
			config: &SafeOutputsConfig{UnassignFromUser: &UnassignFromUserConfig{}},
		},
		{
			name:   "create-project",
			key:    "create-project",
			config: &SafeOutputsConfig{CreateProjects: &CreateProjectsConfig{}},
		},
		{
			name:   "create-project-status-update",
			key:    "create-project-status-update",
			config: &SafeOutputsConfig{CreateProjectStatusUpdates: &CreateProjectStatusUpdateConfig{}},
		},
		{
			name:   "link-sub-issue",
			key:    "link-sub-issue",
			config: &SafeOutputsConfig{LinkSubIssue: &LinkSubIssueConfig{}},
		},
		{
			name:   "hide-comment",
			key:    "hide-comment",
			config: &SafeOutputsConfig{HideComment: &HideCommentConfig{}},
		},
		{
			name:   "dispatch-workflow",
			key:    "dispatch-workflow",
			config: &SafeOutputsConfig{DispatchWorkflow: &DispatchWorkflowConfig{}},
		},
		{
			name:   "missing-data",
			key:    "missing-data",
			config: &SafeOutputsConfig{MissingData: &MissingDataConfig{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should return true when the field is set
			assert.True(t, hasSafeOutputType(tt.config, tt.key), "hasSafeOutputType should return true for key %q when field is set", tt.key)

			// Should return false when the field is nil (empty config)
			assert.False(t, hasSafeOutputType(&SafeOutputsConfig{}, tt.key), "hasSafeOutputType should return false for key %q when field is nil", tt.key)
		})
	}
}

// ========================================
// SafeOutputsConfig YAML tag fixes
// ========================================

// TestSafeOutputsConfigYAMLTags verifies that SafeOutputsConfig uses singular YAML tags
// matching the schema keys (not plural forms that would fail additionalProperties validation).
func TestSafeOutputsConfigYAMLTags(t *testing.T) {
	trueVal := true
	config := &SafeOutputsConfig{
		CreateIssues:                    &CreateIssuesConfig{TitlePrefix: "test"},
		CreateDiscussions:               &CreateDiscussionsConfig{},
		CloseDiscussions:                &CloseDiscussionsConfig{},
		AddComments:                     &AddCommentsConfig{},
		CreatePullRequests:              &CreatePullRequestsConfig{},
		CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{},
		UpdateIssues:                    &UpdateIssuesConfig{},
		Footer:                          &trueVal,
	}

	out, err := yaml.Marshal(config)
	require.NoError(t, err, "SafeOutputsConfig should marshal to YAML without error")

	yamlStr := string(out)

	// Verify singular form keys are present
	assert.Contains(t, yamlStr, "create-issue:", "YAML should use singular 'create-issue'")
	assert.Contains(t, yamlStr, "create-discussion:", "YAML should use singular 'create-discussion'")
	assert.Contains(t, yamlStr, "close-discussion:", "YAML should use singular 'close-discussion'")
	assert.Contains(t, yamlStr, "add-comment:", "YAML should use singular 'add-comment'")
	assert.Contains(t, yamlStr, "create-pull-request:", "YAML should use singular 'create-pull-request'")
	assert.Contains(t, yamlStr, "update-issue:", "YAML should use singular 'update-issue'")

	// Verify plural form keys are absent (these were the old incorrect tags)
	assert.NotContains(t, yamlStr, "create-issues:", "YAML must not use plural 'create-issues'")
	assert.NotContains(t, yamlStr, "create-discussions:", "YAML must not use plural 'create-discussions'")
	assert.NotContains(t, yamlStr, "close-discussions:", "YAML must not use plural 'close-discussions'")
	assert.NotContains(t, yamlStr, "add-comments:", "YAML must not use plural 'add-comments'")
	assert.NotContains(t, yamlStr, "create-pull-requests:", "YAML must not use plural 'create-pull-requests'")
	assert.NotContains(t, yamlStr, "create-pull-request-review-comments:", "YAML must not use plural 'create-pull-request-review-comments'")
	assert.NotContains(t, yamlStr, "update-issues:", "YAML must not use plural 'update-issues'")
}

// ========================================
// Meta field merges in MergeSafeOutputs
// ========================================

// TestMergeSafeOutputsMetaFieldsUnit verifies that the five previously-unmerged meta fields
// (Footer, AllowGitHubReferences, GroupReports, MaxBotMentions, Mentions) are correctly
// merged from imported workflow configs when absent in the top-level config.
func TestMergeSafeOutputsMetaFieldsUnit(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tests := []struct {
		name      string
		topConfig *SafeOutputsConfig
		imported  string
		verify    func(t *testing.T, result *SafeOutputsConfig)
	}{
		{
			name:      "Footer imported when nil in main",
			topConfig: nil,
			imported:  `{"footer":false}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.Footer, "Footer should be imported")
				assert.False(t, *result.Footer, "Footer should be false")
			},
		},
		{
			name: "Footer not overridden when set in main",
			topConfig: &SafeOutputsConfig{
				Footer: boolPtr(true),
			},
			imported: `{"footer":false}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.Footer, "Footer should be present")
				assert.True(t, *result.Footer, "Main Footer (true) should take precedence over imported (false)")
			},
		},
		{
			name:      "AllowGitHubReferences imported when empty in main",
			topConfig: nil,
			imported:  `{"allowed-github-references":["owner/repo1","owner/repo2"]}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				assert.Equal(t, []string{"owner/repo1", "owner/repo2"}, result.AllowGitHubReferences, "AllowGitHubReferences should be imported")
			},
		},
		{
			name: "AllowGitHubReferences not overridden when set in main",
			topConfig: &SafeOutputsConfig{
				AllowGitHubReferences: []string{"owner/main-repo"},
			},
			imported: `{"allowed-github-references":["owner/imported-repo"]}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				assert.Equal(t, []string{"owner/main-repo"}, result.AllowGitHubReferences, "Main AllowGitHubReferences should take precedence")
			},
		},
		{
			name:      "GroupReports imported when false in main",
			topConfig: nil,
			imported:  `{"group-reports":true}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				assert.True(t, result.GroupReports, "GroupReports should be imported as true")
			},
		},
		{
			name: "GroupReports not overridden when true in main",
			topConfig: &SafeOutputsConfig{
				GroupReports: true,
			},
			imported: `{"group-reports":false}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				assert.True(t, result.GroupReports, "Main GroupReports should remain true")
			},
		},
		{
			name:      "MaxBotMentions imported when nil in main",
			topConfig: nil,
			imported:  `{"max-bot-mentions":5}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.MaxBotMentions, "MaxBotMentions should be imported")
				assert.Equal(t, "5", *result.MaxBotMentions, "MaxBotMentions value should be '5'")
			},
		},
		{
			name: "MaxBotMentions not overridden when set in main",
			topConfig: &SafeOutputsConfig{
				MaxBotMentions: strPtr("10"),
			},
			imported: `{"max-bot-mentions":5}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.MaxBotMentions, "MaxBotMentions should be present")
				assert.Equal(t, "10", *result.MaxBotMentions, "Main MaxBotMentions should take precedence")
			},
		},
		{
			name:      "Mentions imported when nil in main",
			topConfig: nil,
			imported:  `{"mentions":{"allowed":["bot1","bot2"]}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.Mentions, "Mentions should be imported")
				assert.Equal(t, []string{"bot1", "bot2"}, result.Mentions.Allowed, "Mentions.Allowed should match")
			},
		},
		{
			name: "Mentions not overridden when set in main",
			topConfig: &SafeOutputsConfig{
				Mentions: &MentionsConfig{Allowed: []string{"main-bot"}},
			},
			imported: `{"mentions":{"allowed":["imported-bot"]}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.Mentions, "Mentions should be present")
				assert.Equal(t, []string{"main-bot"}, result.Mentions.Allowed, "Main Mentions should take precedence")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compiler.MergeSafeOutputs(tt.topConfig, []string{tt.imported}, nil)
			require.NoError(t, err, "MergeSafeOutputs should not error")
			require.NotNil(t, result, "result should not be nil")
			tt.verify(t, result)
		})
	}
}

// TestMergeProtectedFilesExcludeAsSet verifies that when both the top-level and imported
// configs define the same handler, their protected-files exclude lists are merged as a set.
func TestMergeProtectedFilesExcludeAsSet(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tests := []struct {
		name      string
		topConfig *SafeOutputsConfig
		imported  string
		verify    func(t *testing.T, result *SafeOutputsConfig)
	}{
		{
			name: "create-pull-request exclude merged from import when top has none",
			topConfig: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			imported: `{"create-pull-request":{"protected-files":{"exclude":["AGENTS.md"]}}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.CreatePullRequests, "CreatePullRequests should be present")
				assert.Equal(t, []string{"AGENTS.md"}, result.CreatePullRequests.ProtectedFilesExclude,
					"Imported exclude should be merged")
			},
		},
		{
			name: "create-pull-request exclude merged as set when both define it",
			topConfig: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					ProtectedFilesExclude: []string{"CLAUDE.md"},
				},
			},
			imported: `{"create-pull-request":{"protected-files":{"exclude":["AGENTS.md"]}}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.CreatePullRequests, "CreatePullRequests should be present")
				assert.ElementsMatch(t, []string{"CLAUDE.md", "AGENTS.md"}, result.CreatePullRequests.ProtectedFilesExclude,
					"Exclude lists should be merged as a set")
			},
		},
		{
			name: "push-to-pull-request-branch exclude merged from import when top has none",
			topConfig: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			imported: `{"push-to-pull-request-branch":{"protected-files":{"exclude":["AGENTS.md"]}}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.PushToPullRequestBranch, "PushToPullRequestBranch should be present")
				assert.Equal(t, []string{"AGENTS.md"}, result.PushToPullRequestBranch.ProtectedFilesExclude,
					"Imported exclude should be merged")
			},
		},
		{
			name: "deduplication: same item not added twice",
			topConfig: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					ProtectedFilesExclude: []string{"AGENTS.md"},
				},
			},
			imported: `{"create-pull-request":{"protected-files":{"exclude":["AGENTS.md","CLAUDE.md"]}}}`,
			verify: func(t *testing.T, result *SafeOutputsConfig) {
				require.NotNil(t, result.CreatePullRequests, "CreatePullRequests should be present")
				assert.ElementsMatch(t, []string{"AGENTS.md", "CLAUDE.md"}, result.CreatePullRequests.ProtectedFilesExclude,
					"Duplicate items should be deduplicated")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compiler.MergeSafeOutputs(tt.topConfig, []string{tt.imported}, nil)
			require.NoError(t, err, "MergeSafeOutputs should not error")
			require.NotNil(t, result, "result should not be nil")
			tt.verify(t, result)
		})
	}
}
