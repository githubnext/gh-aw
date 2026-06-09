package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGitHubAppTokenMintStepOwnerDerivation(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tests := []struct {
		name               string
		app                *GitHubAppConfig
		ownerSourceRepo    string
		expectedOwner      string
		expectedContains   string
		unexpectedContains string
	}{
		{
			name: "explicit owner wins",
			app: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
				Owner:      "explicit-org",
			},
			ownerSourceRepo:    "${{ github.event.inputs.trigger_ref }}",
			expectedOwner:      "owner: explicit-org",
			unexpectedContains: "id: safe-outputs-app-token-owner",
		},
		{
			name: "literal repository derives owner without helper step",
			app: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			ownerSourceRepo:    "acme/project",
			expectedOwner:      "owner: acme",
			unexpectedContains: "id: safe-outputs-app-token-owner",
		},
		{
			name: "repository expression derives owner with helper step",
			app: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			ownerSourceRepo:  "${{ github.event.inputs.trigger_ref }}",
			expectedOwner:    "owner: ${{ steps.safe-outputs-app-token-owner.outputs.owner }}",
			expectedContains: "id: safe-outputs-app-token-owner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := compiler.buildGitHubAppTokenMintStepWithMeta(
				tt.app,
				nil,
				"",
				tt.ownerSourceRepo,
				"Generate GitHub App token",
				"safe-outputs-app-token",
			)
			stepsStr := strings.Join(steps, "")

			assert.Contains(t, stepsStr, tt.expectedOwner)
			if tt.expectedContains != "" {
				assert.Contains(t, stepsStr, tt.expectedContains)
			}
			if tt.unexpectedContains != "" {
				assert.NotContains(t, stepsStr, tt.unexpectedContains)
			}
		})
	}
}

func TestWorkflowBuildersDeriveGitHubAppOwnerFromCheckoutRepository(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))
	compiler.jobManager = NewJobManager()

	data := &WorkflowData{
		Name: "Test Workflow",
		On:   `"on": "workflow_dispatch"`,
		Permissions: `"permissions":
  contents: read
  issues: read
  pull-requests: read`,
		CheckoutConfigs: []*CheckoutConfig{
			{
				Repository: "${{ github.event.inputs.trigger_ref }}",
				Path:       "target",
				GitHubApp: &GitHubAppConfig{
					AppID:        "${{ vars.CHECKOUT_APP_ID }}",
					PrivateKey:   "${{ secrets.CHECKOUT_APP_PRIVATE_KEY }}",
					Repositories: []string{"*"},
				},
			},
		},
		ParsedTools: &Tools{
			GitHub: &GitHubToolConfig{
				Mode: "local",
				GitHubApp: &GitHubAppConfig{
					AppID:        "${{ vars.MCP_APP_ID }}",
					PrivateKey:   "${{ secrets.MCP_APP_PRIVATE_KEY }}",
					Repositories: []string{"*"},
				},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:        "${{ vars.SAFE_OUTPUTS_APP_ID }}",
				PrivateKey:   "${{ secrets.SAFE_OUTPUTS_APP_PRIVATE_KEY }}",
				Repositories: []string{"*"},
			},
			CreateIssues: &CreateIssuesConfig{TitlePrefix: "[automated] "},
		},
	}

	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)
	checkoutSteps := strings.Join(checkoutMgr.GenerateCheckoutAppTokenSteps(compiler, nil), "")
	assert.Contains(t, checkoutSteps, "id: checkout-app-token-0-owner")
	assert.Contains(t, checkoutSteps, "owner: ${{ steps.checkout-app-token-0-owner.outputs.owner }}")
	assert.Contains(t, checkoutSteps, "GH_AW_TARGET_REPOSITORY: ${{ github.event.inputs.trigger_ref }}")

	mcpSteps := strings.Join(compiler.generateGitHubMCPAppTokenMintingSteps(data), "")
	assert.Contains(t, mcpSteps, "id: github-mcp-app-token-owner")
	assert.Contains(t, mcpSteps, "owner: ${{ steps.github-mcp-app-token-owner.outputs.owner }}")
	assert.Contains(t, mcpSteps, "GH_AW_TARGET_REPOSITORY: ${{ github.event.inputs.trigger_ref }}")

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(data, string(constants.AgentJobName), "test.md")
	require.NoError(t, err)
	safeOutputsSteps := strings.Join(job.Steps, "")
	assert.Contains(t, safeOutputsSteps, "id: safe-outputs-app-token-owner")
	assert.Contains(t, safeOutputsSteps, "owner: ${{ steps.safe-outputs-app-token-owner.outputs.owner }}")
	assert.Contains(t, safeOutputsSteps, "GH_AW_TARGET_REPOSITORY: ${{ github.event.inputs.trigger_ref }}")
}
