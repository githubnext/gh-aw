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
	// and at least one configured handler targets the current repository.
	if !c.trialMode && data.SafeOutputs.Staged && hasSafeOutputWithoutTargetRepo(data.SafeOutputs) {
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

// hasSafeOutputWithoutTargetRepo returns true if any configured safe output handler
// targets the current repository (i.e., has no target-repo specified).
// Handlers without a target-repo field always target the current repo.
func hasSafeOutputWithoutTargetRepo(so *SafeOutputsConfig) bool {
	// Handlers without a target-repo field always target the current repo.
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

	// Handlers with a target-repo field qualify only when no cross-repo target is specified.
	if so.CreateIssues != nil && so.CreateIssues.TargetRepoSlug == "" {
		return true
	}
	if so.CreateDiscussions != nil && so.CreateDiscussions.TargetRepoSlug == "" {
		return true
	}
	if so.CloseDiscussions != nil && so.CloseDiscussions.TargetRepoSlug == "" {
		return true
	}
	if so.CloseIssues != nil && so.CloseIssues.TargetRepoSlug == "" {
		return true
	}
	if so.ClosePullRequests != nil && so.ClosePullRequests.TargetRepoSlug == "" {
		return true
	}
	if so.MarkPullRequestAsReadyForReview != nil && so.MarkPullRequestAsReadyForReview.TargetRepoSlug == "" {
		return true
	}
	if so.AddComments != nil && so.AddComments.TargetRepoSlug == "" {
		return true
	}
	if so.CreatePullRequests != nil && so.CreatePullRequests.TargetRepoSlug == "" {
		return true
	}
	if so.CreatePullRequestReviewComments != nil && so.CreatePullRequestReviewComments.TargetRepoSlug == "" {
		return true
	}
	if so.SubmitPullRequestReview != nil && so.SubmitPullRequestReview.TargetRepoSlug == "" {
		return true
	}
	if so.ReplyToPullRequestReviewComment != nil && so.ReplyToPullRequestReviewComment.TargetRepoSlug == "" {
		return true
	}
	if so.ResolvePullRequestReviewThread != nil && so.ResolvePullRequestReviewThread.TargetRepoSlug == "" {
		return true
	}
	if so.CreateCodeScanningAlerts != nil && so.CreateCodeScanningAlerts.TargetRepoSlug == "" {
		return true
	}
	if so.AddLabels != nil && so.AddLabels.TargetRepoSlug == "" {
		return true
	}
	if so.RemoveLabels != nil && so.RemoveLabels.TargetRepoSlug == "" {
		return true
	}
	if so.AddReviewer != nil && so.AddReviewer.TargetRepoSlug == "" {
		return true
	}
	if so.AssignMilestone != nil && so.AssignMilestone.TargetRepoSlug == "" {
		return true
	}
	if so.AssignToAgent != nil && so.AssignToAgent.TargetRepoSlug == "" {
		return true
	}
	if so.AssignToUser != nil && so.AssignToUser.TargetRepoSlug == "" {
		return true
	}
	if so.UnassignFromUser != nil && so.UnassignFromUser.TargetRepoSlug == "" {
		return true
	}
	if so.UpdateIssues != nil && so.UpdateIssues.TargetRepoSlug == "" {
		return true
	}
	if so.UpdatePullRequests != nil && so.UpdatePullRequests.TargetRepoSlug == "" {
		return true
	}
	if so.UpdateDiscussions != nil && so.UpdateDiscussions.TargetRepoSlug == "" {
		return true
	}
	if so.UpdateRelease != nil && so.UpdateRelease.TargetRepoSlug == "" {
		return true
	}
	if so.PushToPullRequestBranch != nil && so.PushToPullRequestBranch.TargetRepoSlug == "" {
		return true
	}
	if so.HideComment != nil && so.HideComment.TargetRepoSlug == "" {
		return true
	}
	if so.SetIssueType != nil && so.SetIssueType.TargetRepoSlug == "" {
		return true
	}
	if so.DispatchWorkflow != nil && so.DispatchWorkflow.TargetRepoSlug == "" {
		return true
	}
	if so.CreateAgentSessions != nil && so.CreateAgentSessions.TargetRepoSlug == "" {
		return true
	}
	if so.LinkSubIssue != nil && so.LinkSubIssue.TargetRepoSlug == "" {
		return true
	}
	return false
}
