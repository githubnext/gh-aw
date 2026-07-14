package workflow

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var toolDescriptionEnhancerLog = logger.New("workflow:tool_description_enhancer")

type toolConstraintBuilder func(*SafeOutputsConfig) []string

var toolConstraintBuilders = map[string]toolConstraintBuilder{
	"create_issue": func(safeOutputs *SafeOutputsConfig) []string { return createIssueConstraints(safeOutputs.CreateIssues) },
	"set_issue_field": func(safeOutputs *SafeOutputsConfig) []string {
		return setIssueFieldConstraints(safeOutputs.SetIssueField)
	},
	"create_agent_session": func(safeOutputs *SafeOutputsConfig) []string {
		return createAgentSessionConstraints(safeOutputs.CreateAgentSessions)
	},
	"create_discussion": func(safeOutputs *SafeOutputsConfig) []string {
		return createDiscussionConstraints(safeOutputs.CreateDiscussions)
	},
	"close_discussion": func(safeOutputs *SafeOutputsConfig) []string {
		return closeDiscussionConstraints(safeOutputs.CloseDiscussions)
	},
	"update_discussion": func(safeOutputs *SafeOutputsConfig) []string {
		return updateDiscussionConstraints(safeOutputs.UpdateDiscussions)
	},
	"close_issue": func(safeOutputs *SafeOutputsConfig) []string { return closeIssueConstraints(safeOutputs.CloseIssues) },
	"close_pull_request": func(safeOutputs *SafeOutputsConfig) []string {
		return closePullRequestConstraints(safeOutputs.ClosePullRequests)
	},
	"mark_pull_request_as_ready_for_review": func(safeOutputs *SafeOutputsConfig) []string {
		return markPullRequestAsReadyForReviewConstraints(safeOutputs.MarkPullRequestAsReadyForReview)
	},
	"add_comment": func(safeOutputs *SafeOutputsConfig) []string { return addCommentConstraints(safeOutputs.AddComments) },
	"create_pull_request": func(safeOutputs *SafeOutputsConfig) []string {
		return createPullRequestConstraints(safeOutputs.CreatePullRequests)
	},
	"create_pull_request_review_comment": func(safeOutputs *SafeOutputsConfig) []string {
		return createPullRequestReviewCommentConstraints(safeOutputs.CreatePullRequestReviewComments)
	},
	"submit_pull_request_review": func(safeOutputs *SafeOutputsConfig) []string {
		return submitPullRequestReviewConstraints(safeOutputs.SubmitPullRequestReview)
	},
	"reply_to_pull_request_review_comment": func(safeOutputs *SafeOutputsConfig) []string {
		return replyToPullRequestReviewCommentConstraints(safeOutputs.ReplyToPullRequestReviewComment)
	},
	"dismiss_pull_request_review": func(safeOutputs *SafeOutputsConfig) []string {
		return dismissPullRequestReviewConstraints(safeOutputs.DismissPullRequestReview)
	},
	"resolve_pull_request_review_thread": func(safeOutputs *SafeOutputsConfig) []string {
		return resolvePullRequestReviewThreadConstraints(safeOutputs.ResolvePullRequestReviewThread)
	},
	"create_code_scanning_alert": func(safeOutputs *SafeOutputsConfig) []string {
		return createCodeScanningAlertConstraints(safeOutputs.CreateCodeScanningAlerts)
	},
	"create_check_run": func(safeOutputs *SafeOutputsConfig) []string {
		return createCheckRunConstraints(safeOutputs.CreateCheckRun)
	},
	"add_labels": func(safeOutputs *SafeOutputsConfig) []string { return addLabelsConstraints(safeOutputs.AddLabels) },
	"remove_labels": func(safeOutputs *SafeOutputsConfig) []string {
		return removeLabelsConstraints(safeOutputs.RemoveLabels)
	},
	"replace_label": func(safeOutputs *SafeOutputsConfig) []string {
		return replaceLabelConstraints(safeOutputs.ReplaceLabel)
	},
	"add_reviewer": func(safeOutputs *SafeOutputsConfig) []string { return addReviewerConstraints(safeOutputs.AddReviewer) },
	"update_issue": func(safeOutputs *SafeOutputsConfig) []string { return updateIssueConstraints(safeOutputs.UpdateIssues) },
	"update_pull_request": func(safeOutputs *SafeOutputsConfig) []string {
		return updatePullRequestConstraints(safeOutputs.UpdatePullRequests)
	},
	"push_to_pull_request_branch": func(safeOutputs *SafeOutputsConfig) []string {
		return pushToPullRequestBranchConstraints(safeOutputs.PushToPullRequestBranch)
	},
	"upload_asset": func(safeOutputs *SafeOutputsConfig) []string { return uploadAssetConstraints(safeOutputs.UploadAssets) },
	"update_release": func(safeOutputs *SafeOutputsConfig) []string {
		return updateReleaseConstraints(safeOutputs.UpdateRelease)
	},
	"missing_tool": func(safeOutputs *SafeOutputsConfig) []string { return missingToolConstraints(safeOutputs.MissingTool) },
	"link_sub_issue": func(safeOutputs *SafeOutputsConfig) []string {
		return linkSubIssueConstraints(safeOutputs.LinkSubIssue)
	},
	"assign_milestone": func(safeOutputs *SafeOutputsConfig) []string {
		return assignMilestoneConstraints(safeOutputs.AssignMilestone)
	},
	"assign_to_agent": func(safeOutputs *SafeOutputsConfig) []string {
		return assignToAgentConstraints(safeOutputs.AssignToAgent)
	},
	"update_project": func(safeOutputs *SafeOutputsConfig) []string {
		return updateProjectConstraints(safeOutputs.UpdateProjects)
	},
	"create_project_status_update": func(safeOutputs *SafeOutputsConfig) []string {
		return createProjectStatusUpdateConstraints(safeOutputs.CreateProjectStatusUpdates)
	},
}

// formatStringList formats a slice of strings with proper quoting for readability
// Example: ["bug", "feature request", "docs"] -> ["bug" "feature request" "docs"]
func formatStringList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(quoted, " ") + "]"
}

func appendAllowedIssueFieldsConstraint(constraints *[]string, allowedFields []string) {
	if len(allowedFields) == 0 {
		return
	}
	if slices.Contains(allowedFields, "*") {
		*constraints = append(*constraints, "Any issue field is allowed.")
		return
	}
	*constraints = append(*constraints, fmt.Sprintf("Only these issue fields are allowed: %s.", formatStringList(allowedFields)))
}

func appendMaxConstraint(constraints *[]string, max *string, format string) {
	if templatableIntValue(max) > 0 {
		*constraints = append(*constraints, fmt.Sprintf(format, templatableIntValue(max)))
	}
}

// enhanceToolDescription adds configuration-specific constraints to tool descriptions
// This provides agents with context about limits and restrictions configured in the workflow
func enhanceToolDescription(toolName, baseDescription string, safeOutputs *SafeOutputsConfig) string {
	toolDescriptionEnhancerLog.Printf("Enhancing tool description: tool=%s", toolName)

	if safeOutputs == nil {
		return baseDescription
	}

	constraints := buildToolDescriptionConstraints(toolName, safeOutputs)

	if len(constraints) == 0 {
		toolDescriptionEnhancerLog.Printf("No constraints found for tool: %s", toolName)
		return baseDescription
	}

	toolDescriptionEnhancerLog.Printf("Added %d constraints to tool description: tool=%s", len(constraints), toolName)
	// Add constraints as a new paragraph at the end of the description
	return baseDescription + " CONSTRAINTS: " + strings.Join(constraints, " ")
}

func buildToolDescriptionConstraints(toolName string, safeOutputs *SafeOutputsConfig) []string {
	builder, ok := toolConstraintBuilders[toolName]
	if !ok {
		return nil
	}
	return builder(safeOutputs)
}

func createIssueConstraints(config *CreateIssuesConfig) []string {
	if config == nil {
		return nil
	}

	toolDescriptionEnhancerLog.Printf("Found create_issue config: max=%v, titlePrefix=%s", config.Max, config.TitlePrefix)

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d issue(s) can be created.")
	if config.TitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if len(config.Labels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Labels %s will be automatically added.", formatStringList(config.Labels)))
	}
	if len(config.AllowedLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	appendAllowedIssueFieldsConstraint(&constraints, config.AllowedFields)
	if len(config.Assignees) > 0 {
		constraints = append(constraints, fmt.Sprintf("Assignees %s will be automatically assigned.", formatStringList(config.Assignees)))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Issues will be created in repository %q.", config.TargetRepoSlug))
	}
	if config.RequireTemporaryID {
		constraints = append(constraints, "temporary_id is required.")
	}
	if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
		constraints = append(constraints, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
	}
	return constraints
}

func setIssueFieldConstraints(config *SetIssueFieldConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d issue field update(s) can be made.")
	appendAllowedIssueFieldsConstraint(&constraints, config.AllowedFields)
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Issue fields will be updated in repository %q.", config.TargetRepoSlug))
	}
	return constraints
}

func createAgentSessionConstraints(config *CreateAgentSessionConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d agent task(s) can be created.")
	if config.Base != "" {
		constraints = append(constraints, fmt.Sprintf("Base branch for tasks: %q.", config.Base))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Tasks will be created in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		constraints = append(constraints, fmt.Sprintf("Sessions can target these repositories: %v.", config.AllowedRepos))
	}
	return constraints
}

func createDiscussionConstraints(config *CreateDiscussionsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d discussion(s) can be created.")
	if config.TitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if config.Category != "" {
		constraints = append(constraints, fmt.Sprintf("Discussions will be created in category %q.", config.Category))
	}
	if len(config.AllowedLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Discussions will be created in repository %q.", config.TargetRepoSlug))
	}
	return constraints
}

func closeDiscussionConstraints(config *CloseDiscussionsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d discussion(s) can be closed.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Discussions will be closed in repository %q.", config.TargetRepoSlug))
	}
	if config.RequiredTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Only discussions with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	if config.AllowBody != nil && !*config.AllowBody {
		constraints = append(constraints, "Closing comments are disabled: do not include a body field.")
	}
	return constraints
}

func updateDiscussionConstraints(config *UpdateDiscussionsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d discussion(s) can be updated.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.Title != nil && *config.Title {
		constraints = append(constraints, "Title updates are allowed.")
	}
	if config.Body != nil && *config.Body {
		constraints = append(constraints, "Body updates are allowed.")
	}
	if config.Labels != nil {
		if len(config.AllowedLabels) > 0 {
			constraints = append(constraints, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
		} else {
			constraints = append(constraints, "Label updates are allowed.")
		}
	}
	return constraints
}

func closeIssueConstraints(config *CloseIssuesConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d issue(s) can be closed.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.RequiredTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Only issues with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	if config.AllowBody != nil && !*config.AllowBody {
		constraints = append(constraints, "Closing comments are disabled: do not include a body field.")
	}
	return constraints
}

func closePullRequestConstraints(config *ClosePullRequestsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d pull request(s) can be closed.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Pull requests will be closed in repository %q.", config.TargetRepoSlug))
	}
	if len(config.RequiredLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only PRs with labels %v can be closed.", config.RequiredLabels))
	}
	if config.RequiredTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Only PRs with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	return constraints
}

func markPullRequestAsReadyForReviewConstraints(config *MarkPullRequestAsReadyForReviewConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d pull request(s) can be marked as ready for review.")
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Pull requests will be marked as ready in repository %q.", config.TargetRepoSlug))
	}
	return constraints
}

func addCommentConstraints(config *AddCommentsConfig) []string {
	var constraints []string
	if config != nil {
		appendMaxConstraint(&constraints, config.Max, "Maximum %d comment(s) can be added.")
		if config.Target != "" {
			constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
		}
		if config.TargetRepoSlug != "" {
			constraints = append(constraints, fmt.Sprintf("Comments will be added in repository %q.", config.TargetRepoSlug))
		}
		if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
			constraints = append(constraints, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
		}
	}
	return append(constraints, "Supports reply_to_id for discussion threading.")
}

func createPullRequestConstraints(config *CreatePullRequestsConfig) []string {
	if config == nil {
		return nil
	}

	toolDescriptionEnhancerLog.Printf("Found create_pull_request config: max=%v, titlePrefix=%s, draft=%v", config.Max, config.TitlePrefix, config.Draft)

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d pull request(s) can be created.")
	if config.BranchPrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Branch name will be prefixed with %q.", config.BranchPrefix))
	}
	if config.TitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if len(config.Labels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Labels %s will be automatically added.", formatStringList(config.Labels)))
	}
	if len(config.AllowedLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	if config.Draft != nil && *config.Draft == "true" {
		constraints = append(constraints, "PRs will be created as drafts.")
	}
	if len(config.Reviewers) > 0 {
		constraints = append(constraints, fmt.Sprintf("Reviewers %s will be assigned.", formatStringList(config.Reviewers)))
	}
	if len(config.Assignees) > 0 {
		constraints = append(constraints, fmt.Sprintf("Assignees %s will be assigned to the created pull request and any fallback issue.", formatStringList(config.Assignees)))
	}
	if config.RequireTemporaryID {
		constraints = append(constraints, "temporary_id is required.")
	}
	if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
		constraints = append(constraints, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
	}
	return constraints
}

func createPullRequestReviewCommentConstraints(config *CreatePullRequestReviewCommentsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d review comment(s) can be created.")
	if config.Side != "" {
		constraints = append(constraints, fmt.Sprintf("Comments will be on the %s side of the diff.", config.Side))
	}
	return constraints
}

func submitPullRequestReviewConstraints(config *SubmitPullRequestReviewConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d review(s) can be submitted.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Reviews will be submitted in repository %q.", config.TargetRepoSlug))
	}
	return constraints
}

func replyToPullRequestReviewCommentConstraints(config *ReplyToPullRequestReviewCommentConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d reply/replies can be created.")
	return constraints
}

func dismissPullRequestReviewConstraints(config *DismissPullRequestReviewConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d review dismissal(s) can be performed.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Review dismissals will be performed in repository %q.", config.TargetRepoSlug))
	}
	return append(constraints, "justification must contain at least 20 characters.")
}

func resolvePullRequestReviewThreadConstraints(config *ResolvePullRequestReviewThreadConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d review thread(s) can be resolved.")
	return constraints
}

func createCodeScanningAlertConstraints(config *CreateCodeScanningAlertsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d alert(s) can be created.")
	return constraints
}

func createCheckRunConstraints(config *CreateCheckRunConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d check run(s) can be created.")
	if config.Name != "" {
		constraints = append(constraints, fmt.Sprintf("Check run name: %q.", config.Name))
	}
	return constraints
}

func addLabelsConstraints(config *AddLabelsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d label(s) can be added.")
	if len(config.Allowed) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.Allowed)))
	}
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	return constraints
}

func removeLabelsConstraints(config *RemoveLabelsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d label(s) can be removed.")
	if len(config.Allowed) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels can be removed: %v.", config.Allowed))
	}
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	return constraints
}

func replaceLabelConstraints(config *ReplaceLabelConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d label replacement(s) allowed.")
	if len(config.AllowedTransitions) > 0 {
		pairs := make([]string, len(config.AllowedTransitions))
		for i, transition := range config.AllowedTransitions {
			pairs[i] = fmt.Sprintf("%q → %q", transition.From, transition.To)
		}
		constraints = append(constraints, fmt.Sprintf("Only these label transitions are allowed: %s.", formatStringList(pairs)))
	}
	if len(config.AllowedAdd) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels can be added: %s.", formatStringList(config.AllowedAdd)))
	}
	if len(config.AllowedRemove) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only these labels can be removed: %s.", formatStringList(config.AllowedRemove)))
	}
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	return constraints
}

func addReviewerConstraints(config *AddReviewerConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d reviewer(s) can be added.")
	return constraints
}

func updateIssueConstraints(config *UpdateIssuesConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d issue(s) can be updated.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	titlePrefix := config.TitlePrefix
	if config.RequiredTitlePrefix != "" {
		titlePrefix = config.RequiredTitlePrefix
	}
	if titlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("The target issue title must start with %q.", titlePrefix))
	}
	if config.Title != nil && *config.Title {
		constraints = append(constraints, "Title updates are allowed.")
	}
	if config.Body != nil && *config.Body {
		constraints = append(constraints, "Body updates are allowed.")
	}
	if config.Status != nil && *config.Status {
		constraints = append(constraints, "Status updates (open/closed) are allowed.")
	}
	return constraints
}

func updatePullRequestConstraints(config *UpdatePullRequestsConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d pull request(s) can be updated.")
	if config.Target != "" {
		constraints = append(constraints, fmt.Sprintf("Target: %s.", config.Target))
	}
	if len(config.RequiredLabels) > 0 {
		constraints = append(constraints, fmt.Sprintf("Only PRs with labels %v can be updated.", config.RequiredLabels))
	}
	if config.RequiredTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("Only PRs with title prefix %q can be updated.", config.RequiredTitlePrefix))
	}
	return constraints
}

func pushToPullRequestBranchConstraints(config *PushToPullRequestBranchConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d push(es) can be made.")
	if config.TitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("The target pull request title must start with %q.", config.TitlePrefix))
	}
	return constraints
}

func uploadAssetConstraints(config *UploadAssetsConfig) []string {
	if config == nil {
		return nil
	}

	toolDescriptionEnhancerLog.Printf("Found upload_asset config: max=%v, maxSizeKB=%d, allowedExts=%v", config.Max, config.MaxSizeKB, config.AllowedExts)

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d asset(s) can be uploaded.")
	if config.MaxSizeKB > 0 {
		constraints = append(constraints, fmt.Sprintf("Maximum file size: %dKB.", config.MaxSizeKB))
	}
	if len(config.AllowedExts) > 0 {
		constraints = append(constraints, fmt.Sprintf("Allowed file extensions: %v.", config.AllowedExts))
	}
	return constraints
}

func updateReleaseConstraints(config *UpdateReleaseConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d release(s) can be updated.")
	return constraints
}

func missingToolConstraints(config *MissingToolConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d missing tool report(s) can be created.")
	return constraints
}

func linkSubIssueConstraints(config *LinkSubIssueConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d sub-issue link(s) can be created.")
	if config.ParentTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("The parent issue title must start with %q.", config.ParentTitlePrefix))
	}
	if config.SubTitlePrefix != "" {
		constraints = append(constraints, fmt.Sprintf("The sub-issue title must start with %q.", config.SubTitlePrefix))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Sub-issues will be linked in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		constraints = append(constraints, fmt.Sprintf("Sub-issue linking can target these repositories: %v.", config.AllowedRepos))
	}
	return constraints
}

func assignMilestoneConstraints(config *AssignMilestoneConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d milestone assignment(s) can be made.")
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Milestones will be assigned in repository %q.", config.TargetRepoSlug))
	}
	return constraints
}

func assignToAgentConstraints(config *AssignToAgentConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d issue(s) can be assigned to agent.")
	if config.BaseBranch != "" {
		constraints = append(constraints, fmt.Sprintf("Pull requests will target the %q branch.", config.BaseBranch))
	}
	if config.TargetRepoSlug != "" {
		constraints = append(constraints, fmt.Sprintf("Issues will be assigned to agent in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		constraints = append(constraints, fmt.Sprintf("Agent assignment can target these repositories: %v.", config.AllowedRepos))
	}
	return constraints
}

func updateProjectConstraints(config *UpdateProjectConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d project operation(s) can be performed.")
	if config.Project != "" {
		constraints = append(constraints, fmt.Sprintf("Default project URL: %q.", config.Project))
	}
	return constraints
}

func createProjectStatusUpdateConstraints(config *CreateProjectStatusUpdateConfig) []string {
	if config == nil {
		return nil
	}

	var constraints []string
	appendMaxConstraint(&constraints, config.Max, "Maximum %d status update(s) can be created.")
	if config.Project != "" {
		constraints = append(constraints, fmt.Sprintf("Default project URL: %q.", config.Project))
	}
	return constraints
}
