package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsJobsLog = logger.New("workflow:safe_outputs_jobs")

// ========================================
// Safe Output Job Configuration and Builder
// ========================================

// SafeOutputJobConfig holds configuration for building a safe output job
// This config struct extracts the common parameters across all safe output job builders
type SafeOutputJobConfig struct {
	// Job metadata
	JobName     string // e.g., "create_issue"
	StepName    string // e.g., "Create Output Issue"
	StepID      string // e.g., "create_issue"
	MainJobName string // Main workflow job name for dependencies

	// Custom environment variables specific to this safe output type
	CustomEnvVars []string

	// JavaScript script constant to include in the GitHub Script step
	Script string

	// Script name for looking up custom action path (optional)
	// If provided and action mode is custom, the compiler will use a custom action
	// instead of inline JavaScript. Example: "create_issue"
	ScriptName string

	// Job configuration
	Permissions                *Permissions      // Job permissions
	Outputs                    map[string]string // Job outputs
	Condition                  ConditionNode     // Job condition (if clause)
	Needs                      []string          // Job dependencies
	PreSteps                   []string          // Optional steps to run before the GitHub Script step
	PostSteps                  []string          // Optional steps to run after the GitHub Script step
	Token                      string            // GitHub token for this output type
	UseCopilotRequestsToken    bool              // Whether to use Copilot token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use agent token preference chain (config token > GH_AW_AGENT_TOKEN)
	TargetRepoSlug             string            // Target repository for cross-repo operations
}

// buildSafeOutputJob creates a safe output job with common scaffolding
// This extracts the repeated pattern found across safe output job builders:
// 1. Validate configuration
// 2. Build custom environment variables
// 3. Invoke buildGitHubScriptStep
// 4. Create Job with standard metadata
func (c *Compiler) buildSafeOutputJob(data *WorkflowData, config SafeOutputJobConfig) (*Job, error) {
	safeOutputsJobsLog.Printf("Building safe output job: %s (actionMode=%s)", config.JobName, c.actionMode)
	var steps []string
	if len(config.PreSteps) > 0 {
		safeOutputsJobsLog.Printf("Adding %d pre-steps to job", len(config.PreSteps))
		steps = append(steps, config.PreSteps...)
	}
	steps = append(steps, c.buildSafeOutputGitHubAppTokenSteps(data, config)...)
	steps = append(steps, c.buildSafeOutputScriptSteps(data, config)...)
	if len(config.PostSteps) > 0 {
		steps = append(steps, config.PostSteps...)
	}
	jobCondition := safeOutputJobCondition(config)
	needs := safeOutputJobNeeds(config)
	safeOutputsJobsLog.Printf("Job %s needs: %v", config.JobName, needs)
	return &Job{
		Name:           config.JobName,
		If:             RenderCondition(jobCondition),
		RunsOn:         c.formatFrameworkJobRunsOn(data),
		Environment:    c.indentYAMLLines(resolveSafeOutputsEnvironment(data), "    "),
		Permissions:    config.Permissions.RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required for all safe output jobs
		Steps:          steps,
		Outputs:        config.Outputs,
		Needs:          needs,
	}, nil
}

func (c *Compiler) buildSafeOutputGitHubAppTokenSteps(data *WorkflowData, config SafeOutputJobConfig) []string {
	if data.SafeOutputs == nil || data.SafeOutputs.GitHubApp == nil {
		return nil
	}
	safeOutputsJobsLog.Print("Adding GitHub App token minting step with auto-computed permissions")
	var appTokenFallbackRepo string
	if hasWorkflowCallTrigger(data.On) {
		appTokenFallbackRepo = "${{ needs.activation.outputs.target_repo_name }}"
	}
	return c.buildGitHubAppTokenMintStepForRepository(
		data.SafeOutputs.GitHubApp,
		config.Permissions,
		appTokenFallbackRepo,
		inferSingleCheckoutRepositoryForGitHubAppOwner(data),
	)
}

func (c *Compiler) buildSafeOutputScriptSteps(data *WorkflowData, config SafeOutputJobConfig) []string {
	stepConfig := safeOutputGitHubScriptStepConfig(config)
	if c.actionMode.UsesExternalActions() && config.ScriptName != "" {
		safeOutputsJobsLog.Printf("Using custom action mode (%s) for script: %s", c.actionMode, config.ScriptName)
		return c.buildCustomActionStep(data, stepConfig, config.ScriptName)
	}
	if config.ScriptName != "" {
		stepConfig.ScriptFile = config.ScriptName + ".cjs"
		safeOutputsJobsLog.Printf("Using inline mode with external script: %s", stepConfig.ScriptFile)
	} else {
		safeOutputsJobsLog.Printf("Using inline mode (actions/github-script)")
	}
	return c.buildGitHubScriptStep(data, stepConfig)
}

func safeOutputGitHubScriptStepConfig(config SafeOutputJobConfig) GitHubScriptStepConfig {
	return GitHubScriptStepConfig{
		StepName:                   config.StepName,
		StepID:                     config.StepID,
		MainJobName:                config.MainJobName,
		CustomEnvVars:              config.CustomEnvVars,
		Script:                     config.Script,
		CustomToken:                config.Token,
		UseCopilotRequestsToken:    config.UseCopilotRequestsToken,
		UseCopilotCodingAgentToken: config.UseCopilotCodingAgentToken,
	}
}

func safeOutputJobCondition(config SafeOutputJobConfig) ConditionNode {
	if config.Condition != nil {
		return config.Condition
	}
	safeOutputsJobsLog.Printf("No custom condition provided, using default for job: %s", config.JobName)
	return BuildSafeOutputType(config.JobName)
}

func safeOutputJobNeeds(config SafeOutputJobConfig) []string {
	if len(config.Needs) > 0 {
		return config.Needs
	}
	return []string{config.MainJobName}
}
