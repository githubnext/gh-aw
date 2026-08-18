//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveWorkflowRunConfiguration(t *testing.T) {
	workflowFile := filepath.Join(t.TempDir(), "approve.md")
	err := os.WriteFile(workflowFile, []byte(`---
on: issues
engine: copilot
safe-outputs:
  approve-workflow-run:
    max: 2
    staged: true
    fork: true
    allowed-pull-requests:
      - "42"
    allowed-workflows:
      - pull-request-*.yaml
    protected-files:
      exclude:
        - AGENTS.md
---

Approve eligible workflow runs.
`), 0o600)
	require.NoError(t, err)

	data, err := NewCompiler(WithVersion("test")).ParseWorkflowFile(workflowFile)
	require.NoError(t, err)
	require.NotNil(t, data.SafeOutputs)
	require.NotNil(t, data.SafeOutputs.ApproveWorkflowRun)
	assert.Equal(t, strPtr("2"), data.SafeOutputs.ApproveWorkflowRun.Max)
	require.NotNil(t, data.SafeOutputs.ApproveWorkflowRun.Staged)
	assert.Equal(t, TemplatableBool("true"), *data.SafeOutputs.ApproveWorkflowRun.Staged)
	assert.True(t, data.SafeOutputs.ApproveWorkflowRun.Fork)
	assert.Equal(t, []string{"42"}, data.SafeOutputs.ApproveWorkflowRun.AllowedPullRequests)
	assert.Equal(t, []string{"pull-request-*.yaml"}, data.SafeOutputs.ApproveWorkflowRun.AllowedWorkflows)
	assert.Equal(t, []string{"AGENTS.md"}, data.SafeOutputs.ApproveWorkflowRun.ProtectedFilesExclude)

	enabledTools := computeEnabledToolNames(data)
	assert.Contains(t, enabledTools, "approve_workflow_run")
	assert.True(t, hasHandlerManagerTypes(data))
}

func TestApproveWorkflowRunDefaultConfiguration(t *testing.T) {
	config := NewCompiler().parseApproveWorkflowRunConfig(map[string]any{
		"approve-workflow-run": nil,
	})

	require.NotNil(t, config)
	assert.Equal(t, strPtr("1"), config.Max)
	assert.False(t, config.Fork)
}

func TestValidateSafeOutputsApproveWorkflowRunAuthentication(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		wantErr     string
	}{
		{
			name: "missing credentials",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{AllowedWorkflows: []string{"pull-request-*.yml"}},
			},
			wantErr: "requires an external github-token or github-app",
		},
		{
			name: "per-handler external token",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubToken: "${{ secrets.APPROVE_TOKEN }}"},
					AllowedWorkflows:     []string{"pull-request-*.yml"},
				},
			},
		},
		{
			name: "per-handler GitHub App",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubApp: &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"}},
					AllowedWorkflows:     []string{"pull-request-*.yml"},
				},
			},
		},
		{
			name: "safe-outputs external token",
			safeOutputs: &SafeOutputsConfig{
				GitHubToken:        "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{AllowedWorkflows: []string{"pull-request-*.yml"}},
			},
		},
		{
			name: "safe-outputs GitHub App",
			safeOutputs: &SafeOutputsConfig{
				GitHubApp:          &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"},
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{AllowedWorkflows: []string{"pull-request-*.yml"}},
			},
		},
		{
			name: "staged preview",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: templatableBoolPtr("true")},
					AllowedWorkflows:     []string{"pull-request-*.yml"},
				},
			},
		},
		{
			name: "globally staged preview",
			safeOutputs: &SafeOutputsConfig{
				Staged:             templatableBoolPtr("true"),
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{AllowedWorkflows: []string{"pull-request-*.yml"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsApproveWorkflowRun(tt.safeOutputs)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestApproveWorkflowRunHandlerAuthentication(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected string
	}{
		{
			name: "per-handler external token",
			config: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubToken: "${{ secrets.APPROVE_TOKEN }}"},
				},
			},
			expected: "${{ secrets.APPROVE_TOKEN }}",
		},
		{
			name: "per-handler GitHub App token",
			config: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubApp: &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"}},
				},
			},
			expected: "${{ steps.approve-workflow-run-app-token.outputs.token }}",
		},
		{
			name: "safe-outputs external token",
			config: &SafeOutputsConfig{
				GitHubToken:        "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
			},
			expected: "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
		},
		{
			name: "safe-outputs GitHub App token",
			config: &SafeOutputsConfig{
				GitHubApp:          &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"},
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
			},
			expected: "${{ steps.safe-outputs-app-token.outputs.token }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerConfig := handlerRegistry["approve_workflow_run"](tt.config)
			assert.Equal(t, tt.expected, handlerConfig["github-token"])
		})
	}
}

func TestApproveWorkflowRunAllowedPullRequests(t *testing.T) {
	config := NewCompiler().parseApproveWorkflowRunConfig(map[string]any{
		"approve-workflow-run": map[string]any{
			"allowed-pull-requests": "${{ inputs.allowed-pull-requests }}",
		},
	})

	require.NotNil(t, config)
	assert.Equal(t, []string{"${{ inputs.allowed-pull-requests }}"}, config.AllowedPullRequests)

	handlerConfig := handlerRegistry["approve_workflow_run"](&SafeOutputsConfig{ApproveWorkflowRun: config})
	expression, ok := handlerConfig["allowed_pull_requests"].(templatableJSONExpression)
	require.True(t, ok)
	assert.Equal(t, "${{ toJSON(inputs.allowed-pull-requests) }}", expression.expr)
}

func TestApproveWorkflowRunAllowedPullRequestsExpressionEmitsJSONArray(t *testing.T) {
	inputExpression := "${{ needs.approval_allowlist.outputs.eligible_pull_request_numbers }}"
	wrappedExpression := "${{ toJSON(needs.approval_allowlist.outputs.eligible_pull_request_numbers) }}"
	handlerConfig := handlerRegistry["approve_workflow_run"](&SafeOutputsConfig{
		ApproveWorkflowRun: &ApproveWorkflowRunConfig{AllowedPullRequests: []string{inputExpression}},
	})

	configJSON, err := marshalSafeOutputsConfig(map[string]any{"approve_workflow_run": handlerConfig})
	require.NoError(t, err)
	assert.Contains(t, string(configJSON), `"allowed_pull_requests":`+wrappedExpression)

	t.Run("multi-element array", func(t *testing.T) {
		runtimeConfig := strings.ReplaceAll(string(configJSON), wrappedExpression, `["123","456"]`)
		var config map[string]map[string]any
		require.NoError(t, json.Unmarshal([]byte(runtimeConfig), &config))
		assert.Equal(t, []any{"123", "456"}, config["approve_workflow_run"]["allowed_pull_requests"])
	})

	t.Run("numeric array", func(t *testing.T) {
		runtimeConfig := strings.ReplaceAll(string(configJSON), wrappedExpression, `[123,456]`)
		var config map[string]map[string]any
		require.NoError(t, json.Unmarshal([]byte(runtimeConfig), &config))
		assert.Equal(t, []any{float64(123), float64(456)}, config["approve_workflow_run"]["allowed_pull_requests"])
	})

	t.Run("empty string result stays valid JSON", func(t *testing.T) {
		// toJSON("") evaluates to `""` at runtime, which must remain valid JSON even
		// though it is spliced in unquoted (see wrapExpressionWithToJSON).
		runtimeConfig := strings.ReplaceAll(string(configJSON), wrappedExpression, `""`)
		var config map[string]map[string]any
		require.NoError(t, json.Unmarshal([]byte(runtimeConfig), &config))
		assert.Equal(t, "", config["approve_workflow_run"]["allowed_pull_requests"])
	})
}

func TestApproveWorkflowRunAllowedPullRequestsMixedLiteralsAndExpression(t *testing.T) {
	// A slice with more than one element (even if one element looks like an expression)
	// is not treated as a templatable JSON expression: it falls back to a plain JSON
	// array of strings, matching AddStringSlice behaviour.
	config := &ApproveWorkflowRunConfig{AllowedPullRequests: []string{"123", "456"}}
	handlerConfig := handlerRegistry["approve_workflow_run"](&SafeOutputsConfig{ApproveWorkflowRun: config})
	assert.Equal(t, []string{"123", "456"}, handlerConfig["allowed_pull_requests"])

	configJSON, err := marshalSafeOutputsConfig(map[string]any{"approve_workflow_run": handlerConfig})
	require.NoError(t, err)
	assert.Contains(t, string(configJSON), `"allowed_pull_requests":["123","456"]`)
}

func TestValidateSafeOutputsApproveWorkflowRunAllowedWorkflows(t *testing.T) {
	tests := []struct {
		name      string
		workflows []string
		wantErr   string
	}{
		{name: "required", wantErr: "requires a non-empty allowed-workflows list"},
		{name: "path rejected", workflows: []string{".github/workflows/ci.yml"}, wantErr: "must match a workflow filename"},
		{name: "invalid wildcard", workflows: []string{"[ci.yml"}, wantErr: "invalid wildcard pattern"},
		{name: "valid wildcard", workflows: []string{"pull-request-*.yaml"}},
		{name: "uppercase yaml extension", workflows: []string{"pull-request-*.YAML"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeOutputsApproveWorkflowRun(&SafeOutputsConfig{
				Staged: templatableBoolPtr("true"),
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					AllowedWorkflows: tt.workflows,
				},
			})
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
