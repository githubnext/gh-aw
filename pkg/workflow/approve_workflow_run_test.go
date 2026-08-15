//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
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
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
			},
			wantErr: "requires an external github-token or github-app",
		},
		{
			name: "per-handler external token",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubToken: "${{ secrets.APPROVE_TOKEN }}"},
				},
			},
		},
		{
			name: "per-handler GitHub App",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{GitHubApp: &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"}},
				},
			},
		},
		{
			name: "safe-outputs external token",
			safeOutputs: &SafeOutputsConfig{
				GitHubToken:        "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
			},
		},
		{
			name: "safe-outputs GitHub App",
			safeOutputs: &SafeOutputsConfig{
				GitHubApp:          &GitHubAppConfig{AppID: "app-id", PrivateKey: "private-key"},
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
			},
		},
		{
			name: "staged preview",
			safeOutputs: &SafeOutputsConfig{
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Staged: templatableBoolPtr("true")},
				},
			},
		},
		{
			name: "globally staged preview",
			safeOutputs: &SafeOutputsConfig{
				Staged:             templatableBoolPtr("true"),
				ApproveWorkflowRun: &ApproveWorkflowRunConfig{},
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
