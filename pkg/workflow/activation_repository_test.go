//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCallRepositoryResolutionIsJobLocal(t *testing.T) {
	compiler := NewCompiler()
	for _, inlined := range []bool{false, true} {
		data := &WorkflowData{On: "workflow_call", InlinedImports: inlined}
		steps := strings.Join(compiler.generateSetupStepForJob("safe_outputs", data, "./actions/setup", SetupActionDestination, false, "", "", ""), "")
		assert.Contains(t, steps, "id: resolve-host-repo")
		if inlined {
			assert.Contains(t, steps, "JOB_WORKFLOW_REPOSITORY: ${{ github.repository }}")
			assert.Contains(t, steps, "JOB_WORKFLOW_SHA: ${{ github.sha }}")
			assert.Contains(t, steps, "JOB_WORKFLOW_REF: ${{ github.workflow_ref }}")
		} else {
			assert.Contains(t, steps, "JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}")
			assert.Contains(t, steps, "JOB_WORKFLOW_SHA: ${{ job.workflow_sha }}")
			assert.Contains(t, steps, "JOB_WORKFLOW_REF: ${{ job.workflow_ref }}")
		}
		assert.Equal(t, "${{ steps.resolve-host-repo.outputs.target_repo }}", activationTargetRepoExpr)
		assert.Equal(t, "${{ steps.resolve-host-repo.outputs.target_repo_name }}", activationTargetRepoNameExpr)
		assert.NotContains(t, steps, "needs.activation.outputs.target_repo")
	}
}

func TestMainCheckoutUsesExportedHostRepository(t *testing.T) {
	for _, inlined := range []bool{false, true} {
		t.Run(map[bool]string{false: "host imports", true: "inlined imports"}[inlined], func(t *testing.T) {
			compiler := NewCompiler()
			data := &WorkflowData{
				Name:           "repository-checkout",
				On:             "on:\n  workflow_call:\n",
				InlinedImports: inlined,
				AI:             "copilot",
			}
			activation, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
			require.NoError(t, err)

			var steps strings.Builder
			manager, _, err := compiler.generateInitialAndCheckoutSteps(&steps, data)
			require.NoError(t, err)
			if inlined {
				assert.NotContains(t, activation.Outputs, "target_repo")
				assert.Empty(t, manager.GetCrossRepoTargetRepo())
				return
			}

			require.Equal(t, "${{ steps.resolve-host-repo.outputs.target_repo }}", activation.Outputs["target_repo"])
			assert.Contains(t, strings.Join(activation.Steps, ""), "JOB_WORKFLOW_REPOSITORY: ${{ job.workflow_repository }}")
			checkout := strings.Join(manager.GenerateGitHubFolderCheckoutStep(manager.GetCrossRepoTargetRepo(), "", "", compiler.getActionPin), "")
			assert.Contains(t, checkout, "repository: ${{ steps.resolve-host-repo.outputs.target_repo }}")
			assert.NotContains(t, checkout, "|| github.repository")
			assert.Contains(t, checkout, "persist-credentials: false")
		})
	}
}

func TestInlinedWorkflowCallRepositoryFallbacks(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()
	app := &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.APP_KEY }}"}
	data := &WorkflowData{
		Name:           "inlined-repository-fallbacks",
		On:             "on:\n  workflow_call:\n",
		InlinedImports: true,
		AI:             "copilot",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp:        app,
			AddComments:      &AddCommentsConfig{},
			CreateIssues:     &CreateIssuesConfig{},
			DispatchWorkflow: &DispatchWorkflowConfig{Workflows: []string{"worker"}},
			Mentions:         &MentionsConfig{GitHubApp: app},
		},
	}
	activation, err := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	require.NoError(t, err)
	require.NotContains(t, activation.Outputs, "target_repo")
	require.NotContains(t, activation.Outputs, "target_repo_name")

	safeOutputs, _, err := compiler.buildConsolidatedSafeOutputsJob(data, string(constants.AgentJobName), "test.md")
	require.NoError(t, err)
	conclusion, err := compiler.buildConclusionJob(data, string(constants.AgentJobName), nil)
	require.NoError(t, err)
	output, err := compiler.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName: "create_issue", StepName: "Create issue", StepID: "create_issue", MainJobName: string(constants.AgentJobName),
		Script: "return {}",
	})
	require.NoError(t, err)

	for name, steps := range map[string][]string{
		"safe outputs": safeOutputs.Steps,
		"conclusion":   conclusion.Steps,
		"output job":   output.Steps,
		"mentions":     compiler.addAppTokenMintingSteps(data),
	} {
		t.Run(name, func(t *testing.T) {
			generated := strings.Join(steps, "")
			assert.Contains(t, generated, "GH_AW_TARGET_REPOSITORY: "+activationTargetRepoExpr)
			assert.Contains(t, generated, "repositories: "+activationTargetRepoNameExpr)
		})
	}

	var configSteps []string
	compiler.addHandlerManagerConfigEnvVar(&configSteps, data)
	config := extractHandlerConfig(t, strings.Join(configSteps, ""))
	assert.Equal(t, activationTargetRepoExpr, config["dispatch_workflow"]["target-repo"])
}
