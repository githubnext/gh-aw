//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateCodeScanningAlertConfigTargetRepo verifies that create-code-scanning-alert
// correctly parses target-repo and allowed-repos fields.
func TestCreateCodeScanningAlertConfigTargetRepo(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		configMap      map[string]any
		expectedRepo   string
		expectedRepos  []string
		expectedToken  string
		expectedDriver string
	}{
		{
			name: "target-repo and allowed-repos configured",
			configMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"max":           10,
					"target-repo":   "githubnext/gh-aw-side-repo",
					"allowed-repos": []any{"githubnext/gh-aw-side-repo"},
					"github-token":  "${{ secrets.TEMP_USER_PAT }}",
				},
			},
			expectedRepo:   "githubnext/gh-aw-side-repo",
			expectedRepos:  []string{"githubnext/gh-aw-side-repo"},
			expectedToken:  "${{ secrets.TEMP_USER_PAT }}",
			expectedDriver: "",
		},
		{
			name: "driver and target-repo configured",
			configMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"driver":      "My Scanner",
					"target-repo": "owner/other-repo",
				},
			},
			expectedRepo:   "owner/other-repo",
			expectedRepos:  nil,
			expectedToken:  "",
			expectedDriver: "My Scanner",
		},
		{
			name: "no cross-repo config",
			configMap: map[string]any{
				"create-code-scanning-alert": map[string]any{
					"max": 5,
				},
			},
			expectedRepo:  "",
			expectedRepos: nil,
			expectedToken: "",
		},
		{
			name: "nil config value",
			configMap: map[string]any{
				"create-code-scanning-alert": nil,
			},
			expectedRepo:  "",
			expectedRepos: nil,
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := compiler.parseCodeScanningAlertsConfig(tt.configMap)

			require.NotNil(t, cfg, "config should not be nil")
			assert.Equal(t, tt.expectedRepo, cfg.TargetRepoSlug, "TargetRepoSlug mismatch")
			assert.Equal(t, tt.expectedRepos, cfg.AllowedRepos, "AllowedRepos mismatch")
			assert.Equal(t, tt.expectedToken, cfg.GitHubToken, "GitHubToken mismatch")
			if tt.expectedDriver != "" {
				assert.Equal(t, tt.expectedDriver, cfg.Driver, "Driver mismatch")
			}
		})
	}
}

// TestPushToPullRequestBranchConfigTargetRepo verifies that push-to-pull-request-branch
// correctly parses target-repo and allowed-repos fields.
func TestPushToPullRequestBranchConfigTargetRepo(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name          string
		configMap     map[string]any
		expectedRepo  string
		expectedRepos []string
		expectedToken string
	}{
		{
			name: "target-repo and allowed-repos configured",
			configMap: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target-repo":   "githubnext/gh-aw-side-repo",
					"allowed-repos": []any{"githubnext/gh-aw-side-repo"},
					"github-token":  "${{ secrets.TEMP_USER_PAT }}",
				},
			},
			expectedRepo:  "githubnext/gh-aw-side-repo",
			expectedRepos: []string{"githubnext/gh-aw-side-repo"},
			expectedToken: "${{ secrets.TEMP_USER_PAT }}",
		},
		{
			name: "multiple allowed repos",
			configMap: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target-repo":   "org/primary-repo",
					"allowed-repos": []any{"org/primary-repo", "org/secondary-repo"},
				},
			},
			expectedRepo:  "org/primary-repo",
			expectedRepos: []string{"org/primary-repo", "org/secondary-repo"},
			expectedToken: "",
		},
		{
			name: "no cross-repo config",
			configMap: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target": "triggering",
				},
			},
			expectedRepo:  "",
			expectedRepos: nil,
			expectedToken: "",
		},
		{
			name: "nil push-to-pull-request-branch config",
			configMap: map[string]any{
				"push-to-pull-request-branch": nil,
			},
			expectedRepo:  "",
			expectedRepos: nil,
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := compiler.parsePushToPullRequestBranchConfig(tt.configMap)

			require.NotNil(t, cfg, "config should not be nil")
			assert.Equal(t, tt.expectedRepo, cfg.TargetRepoSlug, "TargetRepoSlug mismatch")
			assert.Equal(t, tt.expectedRepos, cfg.AllowedRepos, "AllowedRepos mismatch")
			assert.Equal(t, tt.expectedToken, cfg.GitHubToken, "GitHubToken mismatch")
		})
	}
}

// TestUpdateIssueConfigGitHubToken verifies that update-issue correctly parses the github-token field.
func TestUpdateIssueConfigGitHubToken(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name          string
		configMap     map[string]any
		expectedToken string
		expectedRepo  string
		expectedRepos []string
	}{
		{
			name: "github-token and allowed-repos configured",
			configMap: map[string]any{
				"update-issue": map[string]any{
					"target-repo":   "githubnext/gh-aw-side-repo",
					"allowed-repos": []any{"githubnext/gh-aw-side-repo"},
					"github-token":  "${{ secrets.TEMP_USER_PAT }}",
				},
			},
			expectedToken: "${{ secrets.TEMP_USER_PAT }}",
			expectedRepo:  "githubnext/gh-aw-side-repo",
			expectedRepos: []string{"githubnext/gh-aw-side-repo"},
		},
		{
			name: "no token or cross-repo",
			configMap: map[string]any{
				"update-issue": map[string]any{
					"body": true,
				},
			},
			expectedToken: "",
			expectedRepo:  "",
			expectedRepos: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := compiler.parseUpdateIssuesConfig(tt.configMap)

			require.NotNil(t, cfg, "config should not be nil")
			assert.Equal(t, tt.expectedToken, cfg.GitHubToken, "GitHubToken mismatch")
			assert.Equal(t, tt.expectedRepo, cfg.TargetRepoSlug, "TargetRepoSlug mismatch")
			assert.Equal(t, tt.expectedRepos, cfg.AllowedRepos, "AllowedRepos mismatch")
		})
	}
}

// TestAddCommentGitHubTokenInHandlerConfig verifies that github-token is included in
// the handler manager config JSON for add-comment.
func TestAddCommentGitHubTokenInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
				},
				TargetRepoSlug: "githubnext/gh-aw-side-repo",
				AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps, "steps should not be empty")
	stepsContent := strings.Join(steps, "")

	// Extract and parse the handler config JSON
	handlerConfig := extractHandlerConfig(t, stepsContent)

	addComment, ok := handlerConfig["add_comment"]
	require.True(t, ok, "add_comment config should be present")

	assert.Equal(t, "${{ secrets.TEMP_USER_PAT }}", addComment["github-token"], "github-token should be in handler config")
	assert.Equal(t, "githubnext/gh-aw-side-repo", addComment["target-repo"], "target-repo should be in handler config")

	allowedRepos, ok := addComment["allowed_repos"]
	require.True(t, ok, "allowed_repos should be present")
	assert.Contains(t, allowedRepos, "githubnext/gh-aw-side-repo", "allowed_repos should contain the repo")
}

// TestCreateIssueGitHubTokenInHandlerConfig verifies that github-token is included in
// the handler manager config JSON for create-issue.
func TestCreateIssueGitHubTokenInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
				},
				TargetRepoSlug: "githubnext/gh-aw-side-repo",
				AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps)
	handlerConfig := extractHandlerConfig(t, strings.Join(steps, ""))

	createIssue, ok := handlerConfig["create_issue"]
	require.True(t, ok, "create_issue config should be present")

	assert.Equal(t, "${{ secrets.TEMP_USER_PAT }}", createIssue["github-token"], "github-token should be in handler config")
	assert.Equal(t, "githubnext/gh-aw-side-repo", createIssue["target-repo"], "target-repo should be in handler config")
}

// TestCreateDiscussionGitHubTokenInHandlerConfig verifies that github-token is included in
// the handler manager config JSON for create-discussion.
func TestCreateDiscussionGitHubTokenInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			CreateDiscussions: &CreateDiscussionsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
				},
				TargetRepoSlug: "githubnext/gh-aw-side-repo",
				AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps)
	handlerConfig := extractHandlerConfig(t, strings.Join(steps, ""))

	createDiscussion, ok := handlerConfig["create_discussion"]
	require.True(t, ok, "create_discussion config should be present")

	assert.Equal(t, "${{ secrets.TEMP_USER_PAT }}", createDiscussion["github-token"], "github-token should be in handler config")
}

// TestCreateCodeScanningAlertCrossRepoInHandlerConfig verifies that target-repo, allowed-repos,
// and github-token are included in the handler manager config for create-code-scanning-alert.
func TestCreateCodeScanningAlertCrossRepoInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
				},
				TargetRepoSlug: "githubnext/gh-aw-side-repo",
				AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
				Driver:         "test-scanner",
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps)
	handlerConfig := extractHandlerConfig(t, strings.Join(steps, ""))

	alert, ok := handlerConfig["create_code_scanning_alert"]
	require.True(t, ok, "create_code_scanning_alert config should be present")

	assert.Equal(t, "${{ secrets.TEMP_USER_PAT }}", alert["github-token"], "github-token should be in handler config")
	assert.Equal(t, "githubnext/gh-aw-side-repo", alert["target-repo"], "target-repo should be in handler config")
	assert.Equal(t, "test-scanner", alert["driver"], "driver should be in handler config")

	allowedRepos, ok := alert["allowed_repos"]
	require.True(t, ok, "allowed_repos should be present")
	assert.Contains(t, allowedRepos, "githubnext/gh-aw-side-repo", "allowed_repos should contain the repo")
}

// TestUpdateIssueGitHubTokenInHandlerConfig verifies that github-token is included in
// the handler manager config JSON for update-issue.
func TestUpdateIssueGitHubTokenInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	bodyVal := true
	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			UpdateIssues: &UpdateIssuesConfig{
				UpdateEntityConfig: UpdateEntityConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
					},
					SafeOutputTargetConfig: SafeOutputTargetConfig{
						TargetRepoSlug: "githubnext/gh-aw-side-repo",
						AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
					},
				},
				Body: &bodyVal,
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps)
	handlerConfig := extractHandlerConfig(t, strings.Join(steps, ""))

	updateIssue, ok := handlerConfig["update_issue"]
	require.True(t, ok, "update_issue config should be present")

	assert.Equal(t, "${{ secrets.TEMP_USER_PAT }}", updateIssue["github-token"], "github-token should be in handler config")
	assert.Equal(t, "githubnext/gh-aw-side-repo", updateIssue["target-repo"], "target-repo should be in handler config")

	allowedRepos, ok := updateIssue["allowed_repos"]
	require.True(t, ok, "allowed_repos should be present")
	assert.Contains(t, allowedRepos, "githubnext/gh-aw-side-repo", "allowed_repos should contain the repo")
}

// TestPushToPullRequestBranchCrossRepoInHandlerConfig verifies that target-repo and allowed-repos
// are included in the handler manager config JSON for push-to-pull-request-branch.
func TestPushToPullRequestBranchCrossRepoInHandlerConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test",
		SafeOutputs: &SafeOutputsConfig{
			PushToPullRequestBranch: &PushToPullRequestBranchConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
				},
				TargetRepoSlug: "githubnext/gh-aw-side-repo",
				AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
			},
		},
	}

	var steps []string
	compiler.addHandlerManagerConfigEnvVar(&steps, workflowData)

	require.NotEmpty(t, steps)
	handlerConfig := extractHandlerConfig(t, strings.Join(steps, ""))

	pushBranch, ok := handlerConfig["push_to_pull_request_branch"]
	require.True(t, ok, "push_to_pull_request_branch config should be present")

	assert.Equal(t, "githubnext/gh-aw-side-repo", pushBranch["target-repo"], "target-repo should be in handler config")

	allowedRepos, ok := pushBranch["allowed_repos"]
	require.True(t, ok, "allowed_repos should be present")
	assert.Contains(t, allowedRepos, "githubnext/gh-aw-side-repo", "allowed_repos should contain the repo")
}

// TestHandlerManagerStepUsesPerOutputToken verifies that the handler manager step
// uses the per-output github-token when no global safe-outputs token is set.
func TestHandlerManagerStepUsesPerOutputToken(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		expectedTokens []string
	}{
		{
			name: "add-comment token is used for step",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
					},
					TargetRepoSlug: "githubnext/gh-aw-side-repo",
					AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
				},
			},
			expectedTokens: []string{"${{ secrets.TEMP_USER_PAT }}"},
		},
		{
			name: "create-issue token is used for step",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
					},
					TargetRepoSlug: "githubnext/gh-aw-side-repo",
					AllowedRepos:   []string{"githubnext/gh-aw-side-repo"},
				},
			},
			expectedTokens: []string{"${{ secrets.TEMP_USER_PAT }}"},
		},
		{
			name: "create-discussion token is used for step",
			safeOutputs: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
					},
					TargetRepoSlug: "githubnext/gh-aw-side-repo",
				},
			},
			expectedTokens: []string{"${{ secrets.TEMP_USER_PAT }}"},
		},
		{
			name: "update-issue token is used for step",
			safeOutputs: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					UpdateEntityConfig: UpdateEntityConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
						},
					},
				},
			},
			expectedTokens: []string{"${{ secrets.TEMP_USER_PAT }}"},
		},
		{
			name: "create-code-scanning-alert token is used for step",
			safeOutputs: &SafeOutputsConfig{
				CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.TEMP_USER_PAT }}",
					},
					TargetRepoSlug: "githubnext/gh-aw-side-repo",
				},
			},
			expectedTokens: []string{"${{ secrets.TEMP_USER_PAT }}"},
		},
		{
			name: "global safe-outputs token takes precedence over per-output token",
			safeOutputs: &SafeOutputsConfig{
				GitHubToken: "${{ secrets.GLOBAL_PAT }}",
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.PER_OUTPUT_PAT }}",
					},
				},
			},
			expectedTokens: []string{"${{ secrets.GLOBAL_PAT }}"},
		},
		{
			name: "no custom token falls back to default",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{},
				},
			},
			// Should fall back to GH_AW_GITHUB_TOKEN || GITHUB_TOKEN
			expectedTokens: []string{"secrets.GH_AW_GITHUB_TOKEN", "secrets.GITHUB_TOKEN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:        "Test",
				SafeOutputs: tt.safeOutputs,
			}

			steps := compiler.buildHandlerManagerStep(workflowData)
			stepsContent := strings.Join(steps, "")

			// Find the github-token line in the "with" section
			for _, expectedToken := range tt.expectedTokens {
				assert.Contains(t, stepsContent, expectedToken,
					"handler manager step should use token: %s", expectedToken)
			}
		})
	}
}

// TestParseAllowedReposFromConfig verifies the parseAllowedReposFromConfig helper function.
func TestParseAllowedReposFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{
			name: "single repo as array",
			input: map[string]any{
				"allowed-repos": []any{"owner/repo"},
			},
			expected: []string{"owner/repo"},
		},
		{
			name: "multiple repos",
			input: map[string]any{
				"allowed-repos": []any{"owner/repo1", "owner/repo2", "other-owner/repo3"},
			},
			expected: []string{"owner/repo1", "owner/repo2", "other-owner/repo3"},
		},
		{
			name:     "no allowed-repos key",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "empty allowed-repos array",
			input: map[string]any{
				"allowed-repos": []any{},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAllowedReposFromConfig(tt.input)
			if tt.expected == nil {
				assert.Empty(t, result, "parseAllowedReposFromConfig should return nil or empty for: %s", tt.name)
			} else {
				assert.Equal(t, tt.expected, result, "parseAllowedReposFromConfig mismatch")
			}
		})
	}
}

// extractHandlerConfig is a helper that parses the GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
// JSON from the rendered step strings.
func extractHandlerConfig(t *testing.T, stepsContent string) map[string]map[string]any {
	t.Helper()

	var configJSON string
	for line := range strings.SplitSeq(stepsContent, "\n") {
		if strings.Contains(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
			parts := strings.SplitN(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: ", 2)
			if len(parts) == 2 {
				configJSON = strings.TrimSpace(parts[1])
				configJSON = strings.Trim(configJSON, "\"")
				configJSON = strings.ReplaceAll(configJSON, "\\\"", "\"")
				break
			}
		}
	}

	require.NotEmpty(t, configJSON, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var not found in steps")

	var result map[string]map[string]any
	err := json.Unmarshal([]byte(configJSON), &result)
	require.NoError(t, err, "Handler config JSON should be valid: %s", configJSON)

	return result
}
