package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerAPMJobLog = logger.New("workflow:compiler_apm_job")

// buildAPMJob creates a dedicated job that installs and packs APM (Agent Package Manager)
// dependencies into a bundle artifact. This job runs after the activation job and uploads
// the packed bundle so the agent job can download and restore it.
//
// The APM job uses minimal permissions ({}) because all required tokens are passed
// explicitly via env/secrets rather than relying on the workflow's GITHUB_TOKEN scope.
func (c *Compiler) buildAPMJob(data *WorkflowData) (*Job, error) {
	compilerAPMJobLog.Printf("Building APM job: %d packages", len(data.APMDependencies.Packages))

	engine, err := c.getAgenticEngine(data.AI)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine for APM job: %w", err)
	}

	var steps []string

	// Mint a GitHub App token before the pack step if a github-app is configured for APM.
	// The APM job depends on activation, so it can reference needs.activation.outputs.target_repo_name
	// instead of the activation-job-local steps.resolve-host-repo.outputs.target_repo_name.
	if data.APMDependencies.GitHubApp != nil {
		compilerAPMJobLog.Print("Adding APM GitHub App token mint step for cross-org access")
		var apmFallbackRepoExpr string
		if hasWorkflowCallTrigger(data.On) && !data.InlinedImports {
			apmFallbackRepoExpr = "${{ needs.activation.outputs.target_repo_name }}"
		}
		steps = append(steps, buildAPMAppTokenMintStep(data.APMDependencies.GitHubApp, apmFallbackRepoExpr)...)
	}

	// Add the APM pack step.
	compilerAPMJobLog.Printf("Adding APM pack step: %d packages", len(data.APMDependencies.Packages))
	apmTarget := engine.GetAPMTarget()
	apmPackStep := GenerateAPMPackStep(data.APMDependencies, apmTarget, data)
	for _, line := range apmPackStep {
		steps = append(steps, line+"\n")
	}

	// Upload the packed APM bundle as a separate artifact for the agent job to download.
	// The path comes from the apm_pack step output `bundle-path`, which microsoft/apm-action
	// sets to the location of the packed .tar.gz archive.
	// The APM job depends on activation, so it uses artifactPrefixExprForDownstreamJob.
	compilerAPMJobLog.Print("Adding APM bundle artifact upload step")
	apmArtifactName := artifactPrefixExprForDownstreamJob(data) + constants.APMArtifactName
	steps = append(steps, "      - name: Upload APM bundle artifact\n")
	steps = append(steps, "        if: success()\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          name: %s\n", apmArtifactName))
	steps = append(steps, "          path: ${{ steps.apm_pack.outputs.bundle-path }}\n")
	steps = append(steps, "          retention-days: 1\n")

	// Invalidate the APM GitHub App token after use to enforce least-privilege token lifecycle.
	if data.APMDependencies.GitHubApp != nil {
		compilerAPMJobLog.Print("Adding APM GitHub App token invalidation step")
		steps = append(steps, buildAPMAppTokenInvalidationStep()...)
	}

	// Set job-level GH_AW_INFO_APM_VERSION so the apm_pack step can reference it
	// via ${{ env.GH_AW_INFO_APM_VERSION }} in its with: block.
	apmVersion := data.APMDependencies.Version
	if apmVersion == "" {
		apmVersion = string(constants.DefaultAPMVersion)
	}
	env := map[string]string{
		"GH_AW_INFO_APM_VERSION": apmVersion,
	}

	// Minimal permissions: the APM job does not need any GitHub token scopes because
	// all tokens (for apm-action, create-github-app-token, upload-artifact) are either
	// passed explicitly via secrets/env or handled by the runner's ACTIONS_RUNTIME_TOKEN.
	permissions := NewPermissionsEmpty().RenderToYAML()

	job := &Job{
		Name:        string(constants.APMJobName),
		RunsOn:      c.formatFrameworkJobRunsOn(data),
		Permissions: c.indentYAMLLines(permissions, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       []string{string(constants.ActivationJobName)},
	}

	return job, nil
}

// buildAPMJobWrapper builds the APM job and adds it to the job manager.
func (c *Compiler) buildAPMJobWrapper(data *WorkflowData) error {
	apmJob, err := c.buildAPMJob(data)
	if err != nil {
		return fmt.Errorf("failed to build %s job: %w", constants.APMJobName, err)
	}
	c.jobManager.AddJob(apmJob)
	compilerAPMJobLog.Printf("APM job added to job manager")
	return nil
}
