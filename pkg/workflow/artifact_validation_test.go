//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func protectedArtifactWorkflow() *WorkflowData {
	return &WorkflowData{
		Name: "Artifact policy",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{},
			ArtifactValidation: &ArtifactValidationConfig{
				Steps: []any{map[string]any{"name": "Validate policy", "run": "node policy.cjs"}},
			},
		},
	}
}

func TestArtifactValidationConfigExtraction(t *testing.T) {
	compiler := NewCompiler()
	config := compiler.extractSafeOutputsConfig(map[string]any{
		"safe-outputs": map[string]any{
			"create-pull-request": map[string]any{},
			"artifact-validation": map[string]any{
				"steps": []any{map[string]any{"run": "node policy.cjs"}},
			},
		},
	})
	require.NotNil(t, config.ArtifactValidation)
	require.Len(t, config.ArtifactValidation.Steps, 1)
	require.NoError(t, validateArtifactValidation(&WorkflowData{SafeOutputs: config}))
}

func TestArtifactValidationRejectsInvalidConfig(t *testing.T) {
	for _, value := range []any{true, nil, "enabled", map[string]any{}, map[string]any{"steps": []any{}}, map[string]any{"steps": []any{}, "optional": true}} {
		_, err := parseArtifactValidationConfig(value)
		require.ErrorContains(t, err, "CTR-033")
	}
}

func TestArtifactValidationRejectsBypassableSteps(t *testing.T) {
	tests := []struct {
		name string
		step map[string]any
		rule string
	}{
		{"conditional", map[string]any{"run": "node policy.cjs", "if": "always()"}, "CTR-033"},
		{"ignored failure", map[string]any{"run": "node policy.cjs", "continue-on-error": true}, "CTR-033"},
		{"conditional failure tolerance", map[string]any{"run": "node policy.cjs", "continue-on-error": "${{ true }}"}, "CTR-033"},
		{"missing execution", map[string]any{"name": "Not a validator"}, "CTR-033"},
		{"reserved id", map[string]any{"run": "node policy.cjs", "id": "approved_artifacts"}, "CTR-033"},
		{"secret", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ secrets.WRITE_TOKEN }}"}}, "CTR-034"},
		{"bracket secret", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ secrets['WRITE_TOKEN'] }}"}}, "CTR-034"},
		{"case-insensitive secret", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ SeCrEtS.WRITE_TOKEN }}"}}, "CTR-034"},
		{"custom job token", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ needs.token.outputs.token }}"}}, "CTR-034"},
		{"input token", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ inputs.token }}"}}, "CTR-034"},
		{"variable token", map[string]any{"run": "node policy.cjs", "env": map[string]any{"TOKEN": "${{ vars.TOKEN }}"}}, "CTR-034"},
		{"replace root", map[string]any{"run": "node policy.cjs", "env": map[string]any{"GH_AW_ARTIFACT_STAGING_ROOT": "/other"}}, "CTR-033"},
		{"replace checkout", map[string]any{"uses": "actions/checkout@v4"}, "CTR-034"},
		{"node preload", map[string]any{"run": "node policy.cjs", "env": map[string]any{"NODE_OPTIONS": "--require untrusted.cjs"}}, "CTR-033"},
		{"shell override", map[string]any{"run": "node policy.cjs", "shell": "bash"}, "CTR-034"},
		{"script interpolation", map[string]any{"run": "node policy.cjs ${{ github.event.issue.title }}"}, "CTR-034"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := protectedArtifactWorkflow()
			data.SafeOutputs.ArtifactValidation.Steps = []any{test.step}
			require.ErrorContains(t, validateArtifactValidation(data), test.rule)
		})
	}
	data := protectedArtifactWorkflow()
	data.SafeOutputs.ArtifactValidation.Steps = []any{map[string]any{"run": "node policy.cjs", "continue-on-error": false}}
	require.NoError(t, validateArtifactValidation(data))
}

func TestArtifactValidationRejectsCredentialedLegacyValidators(t *testing.T) {
	data := protectedArtifactWorkflow()
	data.SafeOutputs.ArtifactValidation = nil
	data.SafeOutputs.Steps = []any{map[string]any{"run": "node policy.cjs"}}
	require.ErrorContains(t, validateArtifactValidation(data), "CTR-034")

	data.SafeOutputs.CreatePullRequests = nil
	data.SafeOutputs.AddComments = &AddCommentsConfig{}
	require.NoError(t, validateArtifactValidation(data))
}

func TestArtifactValidationRejectsJobOverrides(t *testing.T) {
	for _, name := range []string{"artifact_validation", "safe_outputs", "safe-outputs"} {
		data := protectedArtifactWorkflow()
		data.Jobs = map[string]any{name: map[string]any{"pre-steps": []any{map[string]any{"run": "node other.cjs"}}}}
		require.Error(t, validateArtifactValidation(data))
	}
}

func TestArtifactValidationRejectsUnsupportedCheckoutProfiles(t *testing.T) {
	for _, checkout := range []*CheckoutConfig{
		{Repository: "owner/another-repo"},
		{Path: "nested"},
		{Ref: "untrusted-branch"},
	} {
		data := protectedArtifactWorkflow()
		data.CheckoutConfigs = []*CheckoutConfig{checkout}
		require.ErrorContains(t, validateArtifactValidation(data), "CTR-033")
	}
	data := protectedArtifactWorkflow()
	data.SafeOutputs.CreatePullRequests.TargetRepoSlug = "*"
	require.ErrorContains(t, validateArtifactValidation(data), "CTR-033")
}

func TestArtifactValidationRejectsReservedPublisherEnvironment(t *testing.T) {
	data := protectedArtifactWorkflow()
	data.SafeOutputs.Env = map[string]string{"GH_AW_ARTIFACT_APPROVAL_REQUIRED": "false"}
	require.ErrorContains(t, validateArtifactValidation(data), "CTR-033")
}

func TestArtifactValidationRejectsCustomPublicationRoutes(t *testing.T) {
	for name, configure := range map[string]func(*SafeOutputsConfig){
		"jobs":    func(config *SafeOutputsConfig) { config.Jobs = map[string]*SafeJobConfig{"custom": {}} },
		"scripts": func(config *SafeOutputsConfig) { config.Scripts = map[string]*SafeScriptConfig{"custom": {}} },
		"actions": func(config *SafeOutputsConfig) { config.Actions = map[string]*SafeOutputActionConfig{"custom": {}} },
	} {
		t.Run(name, func(t *testing.T) {
			data := protectedArtifactWorkflow()
			configure(data.SafeOutputs)
			require.ErrorContains(t, validateArtifactValidation(data), "CTR-034")
		})
	}
}

func TestArtifactValidationRejectsSecretsInPolicyConfiguration(t *testing.T) {
	data := protectedArtifactWorkflow()
	data.SafeOutputs.BodyFooter = "${{ secrets.WRITE_TOKEN }}"
	_, err := NewCompiler().buildArtifactValidationJob(data)
	require.ErrorContains(t, err, "CTR-034")
}

func TestArtifactValidationRejectsAgentCredentialLeakage(t *testing.T) {
	data := protectedArtifactWorkflow()
	data.Env = "env:\n  WRITE_TOKEN: ${{ secrets.PUBLISH_TOKEN }}\n"
	require.ErrorContains(t, validateArtifactValidation(data), "CTR-034")
}

func TestArtifactValidationRejectsExecutionEnvironmentOverrides(t *testing.T) {
	for _, key := range []string{"NODE_OPTIONS", "GIT_CONFIG_GLOBAL", "ACTIONS_RESULTS_URL", "GITHUB_EVENT_PATH", "GH_AW_TARGET_REPO_SLUG", "PATH"} {
		t.Run(key, func(t *testing.T) {
			data := protectedArtifactWorkflow()
			data.RawFrontmatter = map[string]any{"env": map[string]any{key: "untrusted-value"}}
			require.ErrorContains(t, validateArtifactValidation(data), "CTR-034")
		})
	}
}

func TestArtifactValidationCompilesProtectedWorkflow(t *testing.T) {
	directory := testutil.TempDir(t, "artifact-validation")
	path := filepath.Join(directory, "policy.md")
	content := `---
on: workflow_dispatch
permissions:
  contents: read
engine: claude
strict: false
env:
  INHERITED_METADATA: should-not-enter-validation
safe-outputs:
  create-pull-request:
    github-token: ${{ secrets.HANDLER_TOKEN }}
  github-token: ${{ secrets.PUBLISH_TOKEN }}
  artifact-validation:
    steps:
      - name: Validate repository policy
        run: node policy.cjs
---
Enforce the repository policy.
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	require.NoError(t, NewCompiler().CompileWorkflow(path))
	compiled, err := os.ReadFile(filepath.Join(directory, "policy.lock.yml"))
	require.NoError(t, err)
	validation := extractJobSection(string(compiled), artifactValidationJobName)
	require.NotEmpty(t, validation)
	assert.Contains(t, validation, "contents: read")
	assert.NotContains(t, validation, "contents: write")
	assert.Contains(t, validation, `INHERITED_METADATA: ""`)
	assert.Contains(t, validation, "persist-credentials: false")
	assert.Contains(t, validation, "prepareArtifactValidation")
	assert.Contains(t, validation, "finalizeArtifactValidation")
	assert.NotContains(t, validation, "secrets.HANDLER_TOKEN")
	assert.NotContains(t, validation, "secrets.PUBLISH_TOKEN")
	publisher := extractJobSection(string(compiled), "safe_outputs")
	assert.Contains(t, publisher, "needs.artifact_validation.result == 'success'")
	assert.Contains(t, publisher, "artifact-ids: ${{ needs.artifact_validation.outputs.artifact_id }}")
	assert.Contains(t, publisher, "initializeArtifactPublication")
	assert.NotContains(t, publisher, "Download patch artifact")
	assert.Contains(t, publisher, "secrets.HANDLER_TOKEN")
}

func TestArtifactValidationJobIsReadOnlyAndFailClosed(t *testing.T) {
	compiler := NewCompiler()
	data := protectedArtifactWorkflow()
	data.Env = "env:\n  GLOBAL_TOKEN: should-not-enter-validation\n  OTEL_EXPORTER_OTLP_HEADERS: should-not-enter-validation\n"
	data.SafeOutputs.Env = map[string]string{"PUBLISH_TOKEN": "${{ secrets.WRITE_TOKEN }}"}
	data.Environment = "production"
	data.SafeOutputs.GitHubToken = "${{ secrets.PUBLISH_TOKEN }}"
	data.SafeOutputs.CreatePullRequests.GitHubToken = "${{ secrets.HANDLER_TOKEN }}"
	job, err := compiler.buildArtifactValidationJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "runs-on: ubuntu-latest", job.RunsOn)
	assert.Equal(t, artifactValidationJobName, job.Name)
	assert.Equal(t, "permissions:\n      contents: read", job.Permissions)
	assert.Empty(t, job.Environment)
	assert.Equal(t, `""`, job.Env["GLOBAL_TOKEN"])
	assert.Equal(t, `""`, job.Env["OTEL_EXPORTER_OTLP_HEADERS"])
	assert.NotContains(t, job.Env, "PUBLISH_TOKEN")
	assert.NotContains(t, job.If, "needs.artifact_validation")
	assert.Contains(t, job.If, "needs.agent.result == 'success'")
	steps := strings.Join(job.Steps, "")
	assert.Contains(t, steps, "persist-credentials: false")
	assert.Contains(t, steps, "github.event.pull_request.base.sha")
	assert.Contains(t, steps, "prepareArtifactValidation")
	assert.Contains(t, steps, "finalizeArtifactValidation")
	assert.Contains(t, steps, "runArtifactValidator")
	assert.Contains(t, steps, "GH_AW_ARTIFACT_VALIDATOR_IMAGE")
	assert.Contains(t, steps, "GH_AW_ARTIFACT_PREPARE_DIGEST: ${{ steps.artifact_prepare.outputs.prepare_digest }}")
	assert.NotContains(t, steps, "Download isolated artifacts")
	assert.NotContains(t, steps, "uses: actions/download-artifact")
	assert.NotContains(t, steps, "continue-on-error: true")
	assert.NotContains(t, steps, "secrets.PUBLISH_TOKEN")
	assert.NotContains(t, steps, "secrets.HANDLER_TOKEN")
	assert.Less(t, strings.Index(steps, "prepareArtifactValidation"), strings.Index(steps, "node policy.cjs"))
	assert.Less(t, strings.Index(steps, "node policy.cjs"), strings.Index(steps, "finalizeArtifactValidation"))
	assert.Less(t, strings.Index(steps, "finalizeArtifactValidation"), strings.Index(steps, "Upload approved"))
	assert.Equal(t, "${{ steps.approved_artifacts.outputs.artifact-id }}", job.Outputs["artifact_id"])
}

func TestArtifactPublicationConsumesOnlyApprovedDownload(t *testing.T) {
	compiler := NewCompiler()
	data := protectedArtifactWorkflow()
	download, err := compiler.buildSafeOutputsDownloadSteps(data, "")
	require.NoError(t, err)
	steps := strings.Join(download, "")
	assert.Contains(t, steps, "artifact-ids: ${{ needs.artifact_validation.outputs.artifact_id }}")
	assert.Contains(t, steps, "id: setup-agent-output-env")
	assert.Contains(t, steps, "initializeArtifactPublication")
	assert.NotContains(t, steps, "Download agent output artifact")
	assert.NotContains(t, steps, "Download patch artifact")
	assert.NotContains(t, steps, "merge-multiple")
	assert.NotContains(t, steps, "continue-on-error")
	condition := RenderCondition(buildSafeOutputsJobCondition(data, false))
	assert.Contains(t, condition, "needs.artifact_validation.result == 'success'")
	assert.Contains(t, condition, "needs.artifact_validation.outputs.artifact_id != ''")
	assert.Contains(t, compiler.buildSafeOutputsJobNeeds(data, "agent", false), artifactValidationJobName)
	env := compiler.buildJobLevelSafeOutputEnvVars(data, "policy")
	assert.Equal(t, `"true"`, env["GH_AW_ARTIFACT_APPROVAL_REQUIRED"])
	assert.Equal(t, `"/tmp/gh-aw/approved-artifacts"`, env["GH_AW_ARTIFACT_APPROVAL_ROOT"])
}
