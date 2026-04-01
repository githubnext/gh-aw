package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var consolidatedSafeOutputsLog = logger.New("workflow:compiler_safe_outputs_consolidated")

// hasCustomTokenSafeOutputs checks if any safe outputs require the @actions/github
// package to be installed at runtime. This is needed when:
//   - Any safe output has a per-handler github-token (handler_auth.cjs uses getOctokit())
//   - Project-related safe outputs are configured (they always create their own Octokit instances)
func (c *Compiler) hasCustomTokenSafeOutputs(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}

	// Project-related safe outputs always need @actions/github — they create
	// their own Octokit clients internally regardless of whether a custom token
	// is configured. Preserve the original project-existence check.
	if safeOutputs.UpdateProjects != nil ||
		safeOutputs.CreateProjects != nil ||
		safeOutputs.CreateProjectStatusUpdates != nil {
		consolidatedSafeOutputsLog.Print("Custom token required: project-related safe outputs detected")
		return true
	}

	// Check BaseSafeOutputConfig.GitHubToken on all safe output types.
	// Note: the top-level safeOutputs.GitHubToken is intentionally NOT checked
	// here — that token is used as the step-level github-script token and does
	// not require @actions/github/getOctokit(). Only per-handler tokens trigger
	// the npm install.
	for _, base := range c.collectBaseSafeOutputConfigs(safeOutputs) {
		if base != nil && base.GitHubToken != "" {
			consolidatedSafeOutputsLog.Print("Custom token required: per-handler github-token configured")
			return true
		}
	}

	consolidatedSafeOutputsLog.Print("No custom token required for safe outputs")
	return false
}

// collectBaseSafeOutputConfigs returns pointers to the BaseSafeOutputConfig
// embedded in every configured safe output type. Nil entries are skipped by callers.
func (c *Compiler) collectBaseSafeOutputConfigs(so *SafeOutputsConfig) []*BaseSafeOutputConfig {
	consolidatedSafeOutputsLog.Print("Collecting base safe output configs for custom token check")
	var configs []*BaseSafeOutputConfig
	if so.CreateIssues != nil {
		configs = append(configs, &so.CreateIssues.BaseSafeOutputConfig)
	}
	if so.CreateDiscussions != nil {
		configs = append(configs, &so.CreateDiscussions.BaseSafeOutputConfig)
	}
	if so.UpdateDiscussions != nil {
		configs = append(configs, &so.UpdateDiscussions.BaseSafeOutputConfig)
	}
	if so.CloseDiscussions != nil {
		configs = append(configs, &so.CloseDiscussions.BaseSafeOutputConfig)
	}
	if so.CloseIssues != nil {
		configs = append(configs, &so.CloseIssues.BaseSafeOutputConfig)
	}
	if so.ClosePullRequests != nil {
		configs = append(configs, &so.ClosePullRequests.BaseSafeOutputConfig)
	}
	if so.MarkPullRequestAsReadyForReview != nil {
		configs = append(configs, &so.MarkPullRequestAsReadyForReview.BaseSafeOutputConfig)
	}
	if so.AddComments != nil {
		configs = append(configs, &so.AddComments.BaseSafeOutputConfig)
	}
	if so.CreatePullRequests != nil {
		configs = append(configs, &so.CreatePullRequests.BaseSafeOutputConfig)
	}
	if so.CreatePullRequestReviewComments != nil {
		configs = append(configs, &so.CreatePullRequestReviewComments.BaseSafeOutputConfig)
	}
	if so.SubmitPullRequestReview != nil {
		configs = append(configs, &so.SubmitPullRequestReview.BaseSafeOutputConfig)
	}
	if so.ReplyToPullRequestReviewComment != nil {
		configs = append(configs, &so.ReplyToPullRequestReviewComment.BaseSafeOutputConfig)
	}
	if so.ResolvePullRequestReviewThread != nil {
		configs = append(configs, &so.ResolvePullRequestReviewThread.BaseSafeOutputConfig)
	}
	if so.CreateCodeScanningAlerts != nil {
		configs = append(configs, &so.CreateCodeScanningAlerts.BaseSafeOutputConfig)
	}
	if so.AutofixCodeScanningAlert != nil {
		configs = append(configs, &so.AutofixCodeScanningAlert.BaseSafeOutputConfig)
	}
	if so.AddLabels != nil {
		configs = append(configs, &so.AddLabels.BaseSafeOutputConfig)
	}
	if so.RemoveLabels != nil {
		configs = append(configs, &so.RemoveLabels.BaseSafeOutputConfig)
	}
	if so.AddReviewer != nil {
		configs = append(configs, &so.AddReviewer.BaseSafeOutputConfig)
	}
	if so.AssignMilestone != nil {
		configs = append(configs, &so.AssignMilestone.BaseSafeOutputConfig)
	}
	if so.AssignToAgent != nil {
		configs = append(configs, &so.AssignToAgent.BaseSafeOutputConfig)
	}
	if so.AssignToUser != nil {
		configs = append(configs, &so.AssignToUser.BaseSafeOutputConfig)
	}
	if so.UnassignFromUser != nil {
		configs = append(configs, &so.UnassignFromUser.BaseSafeOutputConfig)
	}
	if so.UpdateIssues != nil {
		configs = append(configs, &so.UpdateIssues.BaseSafeOutputConfig)
	}
	if so.UpdatePullRequests != nil {
		configs = append(configs, &so.UpdatePullRequests.BaseSafeOutputConfig)
	}
	if so.PushToPullRequestBranch != nil {
		configs = append(configs, &so.PushToPullRequestBranch.BaseSafeOutputConfig)
	}
	if so.UploadAssets != nil {
		configs = append(configs, &so.UploadAssets.BaseSafeOutputConfig)
	}
	if so.UpdateRelease != nil {
		configs = append(configs, &so.UpdateRelease.BaseSafeOutputConfig)
	}
	if so.CreateAgentSessions != nil {
		configs = append(configs, &so.CreateAgentSessions.BaseSafeOutputConfig)
	}
	if so.UpdateProjects != nil {
		configs = append(configs, &so.UpdateProjects.BaseSafeOutputConfig)
	}
	if so.CreateProjects != nil {
		configs = append(configs, &so.CreateProjects.BaseSafeOutputConfig)
	}
	if so.CreateProjectStatusUpdates != nil {
		configs = append(configs, &so.CreateProjectStatusUpdates.BaseSafeOutputConfig)
	}
	if so.LinkSubIssue != nil {
		configs = append(configs, &so.LinkSubIssue.BaseSafeOutputConfig)
	}
	if so.HideComment != nil {
		configs = append(configs, &so.HideComment.BaseSafeOutputConfig)
	}
	if so.SetIssueType != nil {
		configs = append(configs, &so.SetIssueType.BaseSafeOutputConfig)
	}
	if so.DispatchWorkflow != nil {
		configs = append(configs, &so.DispatchWorkflow.BaseSafeOutputConfig)
	}
	if so.CallWorkflow != nil {
		configs = append(configs, &so.CallWorkflow.BaseSafeOutputConfig)
	}
	if so.MissingTool != nil {
		configs = append(configs, &so.MissingTool.BaseSafeOutputConfig)
	}
	if so.MissingData != nil {
		configs = append(configs, &so.MissingData.BaseSafeOutputConfig)
	}
	if so.NoOp != nil {
		configs = append(configs, &so.NoOp.BaseSafeOutputConfig)
	}
	consolidatedSafeOutputsLog.Printf("Collected %d base safe output configs", len(configs))
	return configs
}

// SafeOutputStepConfig holds configuration for building a single safe output step
// within the consolidated safe-outputs job
type SafeOutputStepConfig struct {
	StepName                   string            // Human-readable step name (e.g., "Create Issue")
	StepID                     string            // Step ID for referencing outputs (e.g., "create_issue")
	Script                     string            // JavaScript script to execute (for inline mode)
	ScriptName                 string            // Name of the script in the registry (for file mode)
	CustomEnvVars              []string          // Environment variables specific to this step
	Condition                  ConditionNode     // Step-level condition (if clause)
	Token                      string            // GitHub token for this step
	UseCopilotRequestsToken    bool              // Whether to use Copilot requests token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use Copilot coding agent token preference chain
	PreSteps                   []string          // Optional steps to run before the script step
	PostSteps                  []string          // Optional steps to run after the script step
	Outputs                    map[string]string // Outputs from this step
	ContinueOnError            bool              // Whether to continue the job even if this step fails (continue-on-error: true)
}

// Note: The implementation functions have been moved to focused module files:
// - buildConsolidatedSafeOutputsJob, buildJobLevelSafeOutputEnvVars, buildDetectionSuccessCondition
//   are in compiler_safe_outputs_job.go
// - buildConsolidatedSafeOutputStep, buildSharedPRCheckoutSteps, buildHandlerManagerStep
//   are in compiler_safe_outputs_steps.go
// - addHandlerManagerConfigEnvVar is in compiler_safe_outputs_config.go
// - addAllSafeOutputConfigEnvVars is in compiler_safe_outputs_env.go
