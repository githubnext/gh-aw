package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var compilerSafeOutputsEnvLog = logger.New("workflow:compiler_safe_outputs_env")

func (c *Compiler) addAllSafeOutputConfigEnvVars(steps *[]string, data *WorkflowData) {
	compilerSafeOutputsEnvLog.Print("Adding safe output config environment variables")
	if data.SafeOutputs == nil {
		compilerSafeOutputsEnvLog.Print("No safe outputs configured, skipping env var addition")
		return
	}

	// Add the global staged env var once if staged mode is enabled, not in trial mode,
	// and at least one handler is configured. Staged mode is independent of target-repo.
	if !c.trialMode && data.SafeOutputs.Staged && hasSafeOutputConfigured(data.SafeOutputs) {
		*steps = append(*steps, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
		compilerSafeOutputsEnvLog.Print("Added staged flag")
	}

	// Check if copilot is in create-issue assignees - if so, output issues for assign_to_agent job
	if data.SafeOutputs.CreateIssues != nil {
		if hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees) {
			*steps = append(*steps, "          GH_AW_ASSIGN_COPILOT: \"true\"\n")
			compilerSafeOutputsEnvLog.Print("Copilot assignment requested - will output issues_to_assign_copilot")
		}
	}

	// Note: All handler configuration is read from the config.json file at runtime.
}

// hasSafeOutputConfigured returns true if any safe output handler is configured.
// Staged mode is independent of target-repo: it activates whenever staged is set
// and at least one handler is present.
func hasSafeOutputConfigured(so *SafeOutputsConfig) bool {
	if so.AutofixCodeScanningAlert != nil {
		return true
	}
	if so.UploadAssets != nil {
		return true
	}
	if so.UpdateProjects != nil {
		return true
	}
	if so.CreateProjects != nil {
		return true
	}
	if so.CreateProjectStatusUpdates != nil {
		return true
	}
	if so.CallWorkflow != nil {
		return true
	}
	if so.MissingTool != nil {
		return true
	}
	if so.MissingData != nil {
		return true
	}
	if so.NoOp != nil {
		return true
	}
	if so.CreateIssues != nil {
		return true
	}
	if so.CreateDiscussions != nil {
		return true
	}
	if so.CloseDiscussions != nil {
		return true
	}
	if so.CloseIssues != nil {
		return true
	}
	if so.ClosePullRequests != nil {
		return true
	}
	if so.MarkPullRequestAsReadyForReview != nil {
		return true
	}
	if so.AddComments != nil {
		return true
	}
	if so.CreatePullRequests != nil {
		return true
	}
	if so.CreatePullRequestReviewComments != nil {
		return true
	}
	if so.SubmitPullRequestReview != nil {
		return true
	}
	if so.ReplyToPullRequestReviewComment != nil {
		return true
	}
	if so.ResolvePullRequestReviewThread != nil {
		return true
	}
	if so.CreateCodeScanningAlerts != nil {
		return true
	}
	if so.AddLabels != nil {
		return true
	}
	if so.RemoveLabels != nil {
		return true
	}
	if so.AddReviewer != nil {
		return true
	}
	if so.AssignMilestone != nil {
		return true
	}
	if so.AssignToAgent != nil {
		return true
	}
	if so.AssignToUser != nil {
		return true
	}
	if so.UnassignFromUser != nil {
		return true
	}
	if so.UpdateIssues != nil {
		return true
	}
	if so.UpdatePullRequests != nil {
		return true
	}
	if so.UpdateDiscussions != nil {
		return true
	}
	if so.UpdateRelease != nil {
		return true
	}
	if so.PushToPullRequestBranch != nil {
		return true
	}
	if so.HideComment != nil {
		return true
	}
	if so.SetIssueType != nil {
		return true
	}
	if so.DispatchWorkflow != nil {
		return true
	}
	if so.CreateAgentSessions != nil {
		return true
	}
	if so.LinkSubIssue != nil {
		return true
	}
	return false
}
