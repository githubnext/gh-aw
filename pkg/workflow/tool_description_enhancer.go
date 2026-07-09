package workflow

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var toolDescriptionEnhancerLog = logger.New("workflow:tool_description_enhancer")

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

// enhanceToolDescription adds configuration-specific constraints to tool descriptions
// This provides agents with context about limits and restrictions configured in the workflow
func enhanceToolDescription(toolName, baseDescription string, safeOutputs *SafeOutputsConfig) string {
	toolDescriptionEnhancerLog.Printf("Enhancing tool description: tool=%s", toolName)

	if safeOutputs == nil {
		return baseDescription
	}

	constraints := collectToolConstraints(toolName, safeOutputs)

	if len(constraints) == 0 {
		toolDescriptionEnhancerLog.Printf("No constraints found for tool: %s", toolName)
		return baseDescription
	}

	toolDescriptionEnhancerLog.Printf("Added %d constraints to tool description: tool=%s", len(constraints), toolName)
	return baseDescription + " CONSTRAINTS: " + strings.Join(constraints, " ")
}

// collectToolConstraints dispatches to the appropriate constraint collector for each tool.
func collectToolConstraints(toolName string, s *SafeOutputsConfig) []string {
	switch toolName {
	case "create_issue":
		return collectCreateIssueConstraints(s)
	case "set_issue_field":
		return collectSetIssueFieldConstraints(s)
	case "create_agent_session":
		return collectCreateAgentSessionConstraints(s)
	case "create_discussion":
		return collectCreateDiscussionConstraints(s)
	case "close_discussion":
		return collectCloseDiscussionConstraints(s)
	case "update_discussion":
		return collectUpdateDiscussionConstraints(s)
	case "close_issue":
		return collectCloseIssueConstraints(s)
	case "close_pull_request":
		return collectClosePullRequestConstraints(s)
	case "mark_pull_request_as_ready_for_review":
		return collectMarkPRReadyConstraints(s)
	case "add_comment":
		return collectAddCommentConstraints(s)
	case "create_pull_request":
		return collectCreatePullRequestConstraints(s)
	case "create_pull_request_review_comment":
		return collectCreatePRReviewCommentConstraints(s)
	case "submit_pull_request_review":
		return collectSubmitPRReviewConstraints(s)
	case "reply_to_pull_request_review_comment":
		return collectReplyToPRReviewCommentConstraints(s)
	case "dismiss_pull_request_review":
		return collectDismissPRReviewConstraints(s)
	case "resolve_pull_request_review_thread":
		return collectResolvePRReviewThreadConstraints(s)
	case "create_code_scanning_alert":
		return collectCreateCodeScanningAlertConstraints(s)
	case "create_check_run":
		return collectCreateCheckRunConstraints(s)
	case "add_labels":
		return collectAddLabelsConstraints(s)
	case "remove_labels":
		return collectRemoveLabelsConstraints(s)
	case "replace_label":
		return collectReplaceLabelConstraints(s)
	case "add_reviewer":
		return collectAddReviewerConstraints(s)
	case "update_issue":
		return collectUpdateIssueConstraints(s)
	case "update_pull_request":
		return collectUpdatePullRequestConstraints(s)
	case "push_to_pull_request_branch":
		return collectPushToPRBranchConstraints(s)
	case "upload_asset":
		return collectUploadAssetConstraints(s)
	case "update_release":
		return collectUpdateReleaseConstraints(s)
	case "missing_tool":
		return collectMissingToolConstraints(s)
	case "link_sub_issue":
		return collectLinkSubIssueConstraints(s)
	case "assign_milestone":
		return collectAssignMilestoneConstraints(s)
	case "assign_to_agent":
		return collectAssignToAgentConstraints(s)
	case "update_project":
		return collectUpdateProjectConstraints(s)
	case "create_project_status_update":
		return collectCreateProjectStatusUpdateConstraints(s)
	}
	return nil
}

func collectCreateIssueConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateIssues
	if config == nil {
		return nil
	}
	toolDescriptionEnhancerLog.Printf("Found create_issue config: max=%v, titlePrefix=%s", config.Max, config.TitlePrefix)
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d issue(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.TitlePrefix != "" {
		c = append(c, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if len(config.Labels) > 0 {
		c = append(c, fmt.Sprintf("Labels %s will be automatically added.", formatStringList(config.Labels)))
	}
	if len(config.AllowedLabels) > 0 {
		c = append(c, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	appendAllowedIssueFieldsConstraint(&c, config.AllowedFields)
	if len(config.Assignees) > 0 {
		c = append(c, fmt.Sprintf("Assignees %s will be automatically assigned.", formatStringList(config.Assignees)))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Issues will be created in repository %q.", config.TargetRepoSlug))
	}
	if config.RequireTemporaryID {
		c = append(c, "temporary_id is required.")
	}
	if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
		c = append(c, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
	}
	return c
}

func collectSetIssueFieldConstraints(s *SafeOutputsConfig) []string {
	config := s.SetIssueField
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d issue field update(s) can be made.", templatableIntValue(config.Max)))
	}
	appendAllowedIssueFieldsConstraint(&c, config.AllowedFields)
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Issue fields will be updated in repository %q.", config.TargetRepoSlug))
	}
	return c
}

func collectCreateAgentSessionConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateAgentSessions
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d agent task(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.Base != "" {
		c = append(c, fmt.Sprintf("Base branch for tasks: %q.", config.Base))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Tasks will be created in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		c = append(c, fmt.Sprintf("Sessions can target these repositories: %v.", config.AllowedRepos))
	}
	return c
}

func collectCreateDiscussionConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateDiscussions
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d discussion(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.TitlePrefix != "" {
		c = append(c, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if config.Category != "" {
		c = append(c, fmt.Sprintf("Discussions will be created in category %q.", config.Category))
	}
	if len(config.AllowedLabels) > 0 {
		c = append(c, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Discussions will be created in repository %q.", config.TargetRepoSlug))
	}
	return c
}

func collectCloseDiscussionConstraints(s *SafeOutputsConfig) []string {
	config := s.CloseDiscussions
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d discussion(s) can be closed.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Discussions will be closed in repository %q.", config.TargetRepoSlug))
	}
	if config.RequiredTitlePrefix != "" {
		c = append(c, fmt.Sprintf("Only discussions with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	if config.AllowBody != nil && !*config.AllowBody {
		c = append(c, "Closing comments are disabled: do not include a body field.")
	}
	return c
}

func collectUpdateDiscussionConstraints(s *SafeOutputsConfig) []string {
	config := s.UpdateDiscussions
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d discussion(s) can be updated.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.Title != nil && *config.Title {
		c = append(c, "Title updates are allowed.")
	}
	if config.Body != nil && *config.Body {
		c = append(c, "Body updates are allowed.")
	}
	if config.Labels != nil {
		if len(config.AllowedLabels) > 0 {
			c = append(c, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
		} else {
			c = append(c, "Label updates are allowed.")
		}
	}
	return c
}

func collectCloseIssueConstraints(s *SafeOutputsConfig) []string {
	config := s.CloseIssues
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d issue(s) can be closed.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.RequiredTitlePrefix != "" {
		c = append(c, fmt.Sprintf("Only issues with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	if config.AllowBody != nil && !*config.AllowBody {
		c = append(c, "Closing comments are disabled: do not include a body field.")
	}
	return c
}

func collectClosePullRequestConstraints(s *SafeOutputsConfig) []string {
	config := s.ClosePullRequests
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d pull request(s) can be closed.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Pull requests will be closed in repository %q.", config.TargetRepoSlug))
	}
	if len(config.RequiredLabels) > 0 {
		c = append(c, fmt.Sprintf("Only PRs with labels %v can be closed.", config.RequiredLabels))
	}
	if config.RequiredTitlePrefix != "" {
		c = append(c, fmt.Sprintf("Only PRs with title prefix %q can be closed.", config.RequiredTitlePrefix))
	}
	return c
}

func collectMarkPRReadyConstraints(s *SafeOutputsConfig) []string {
	config := s.MarkPullRequestAsReadyForReview
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d pull request(s) can be marked as ready for review.", templatableIntValue(config.Max)))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Pull requests will be marked as ready in repository %q.", config.TargetRepoSlug))
	}
	return c
}

func collectAddCommentConstraints(s *SafeOutputsConfig) []string {
	var c []string
	if config := s.AddComments; config != nil {
		if templatableIntValue(config.Max) > 0 {
			c = append(c, fmt.Sprintf("Maximum %d comment(s) can be added.", templatableIntValue(config.Max)))
		}
		if config.Target != "" {
			c = append(c, fmt.Sprintf("Target: %s.", config.Target))
		}
		if config.TargetRepoSlug != "" {
			c = append(c, fmt.Sprintf("Comments will be added in repository %q.", config.TargetRepoSlug))
		}
		if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
			c = append(c, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
		}
	}
	c = append(c, "Supports reply_to_id for discussion threading.")
	return c
}

func collectCreatePullRequestConstraints(s *SafeOutputsConfig) []string {
	config := s.CreatePullRequests
	if config == nil {
		return nil
	}
	toolDescriptionEnhancerLog.Printf("Found create_pull_request config: max=%v, titlePrefix=%s, draft=%v", config.Max, config.TitlePrefix, config.Draft)
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d pull request(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.BranchPrefix != "" {
		c = append(c, fmt.Sprintf("Branch name will be prefixed with %q.", config.BranchPrefix))
	}
	if config.TitlePrefix != "" {
		c = append(c, fmt.Sprintf("Title will be prefixed with %q.", config.TitlePrefix))
	}
	if len(config.Labels) > 0 {
		c = append(c, fmt.Sprintf("Labels %s will be automatically added.", formatStringList(config.Labels)))
	}
	if len(config.AllowedLabels) > 0 {
		c = append(c, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.AllowedLabels)))
	}
	if config.Draft != nil && *config.Draft == "true" {
		c = append(c, "PRs will be created as drafts.")
	}
	if len(config.Reviewers) > 0 {
		c = append(c, fmt.Sprintf("Reviewers %s will be assigned.", formatStringList(config.Reviewers)))
	}
	if len(config.Assignees) > 0 {
		c = append(c, fmt.Sprintf("Assignees %s will be assigned to the created pull request and any fallback issue.", formatStringList(config.Assignees)))
	}
	if config.RequireTemporaryID {
		c = append(c, "temporary_id is required.")
	}
	if config.NormalizeClosingKeywords != nil && *config.NormalizeClosingKeywords {
		c = append(c, "Backtick-wrapped issue-closing keyword references (e.g. `Closes #1`) in the body field will be automatically normalized to plain text.")
	}
	return c
}

func collectCreatePRReviewCommentConstraints(s *SafeOutputsConfig) []string {
	config := s.CreatePullRequestReviewComments
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d review comment(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.Side != "" {
		c = append(c, fmt.Sprintf("Comments will be on the %s side of the diff.", config.Side))
	}
	return c
}

func collectSubmitPRReviewConstraints(s *SafeOutputsConfig) []string {
	config := s.SubmitPullRequestReview
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d review(s) can be submitted.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Reviews will be submitted in repository %q.", config.TargetRepoSlug))
	}
	return c
}

func collectReplyToPRReviewCommentConstraints(s *SafeOutputsConfig) []string {
	config := s.ReplyToPullRequestReviewComment
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d reply/replies can be created.", templatableIntValue(config.Max)))
	}
	return c
}

func collectDismissPRReviewConstraints(s *SafeOutputsConfig) []string {
	config := s.DismissPullRequestReview
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d review dismissal(s) can be performed.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Review dismissals will be performed in repository %q.", config.TargetRepoSlug))
	}
	c = append(c, "justification must contain at least 20 characters.")
	return c
}

func collectResolvePRReviewThreadConstraints(s *SafeOutputsConfig) []string {
	config := s.ResolvePullRequestReviewThread
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d review thread(s) can be resolved.", templatableIntValue(config.Max)))
	}
	return c
}

func collectCreateCodeScanningAlertConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateCodeScanningAlerts
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d alert(s) can be created.", templatableIntValue(config.Max)))
	}
	return c
}

func collectCreateCheckRunConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateCheckRun
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d check run(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.Name != "" {
		c = append(c, fmt.Sprintf("Check run name: %q.", config.Name))
	}
	return c
}

func collectAddLabelsConstraints(s *SafeOutputsConfig) []string {
	config := s.AddLabels
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d label(s) can be added.", templatableIntValue(config.Max)))
	}
	if len(config.Allowed) > 0 {
		c = append(c, fmt.Sprintf("Only these labels are allowed: %s.", formatStringList(config.Allowed)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	return c
}

func collectRemoveLabelsConstraints(s *SafeOutputsConfig) []string {
	config := s.RemoveLabels
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d label(s) can be removed.", templatableIntValue(config.Max)))
	}
	if len(config.Allowed) > 0 {
		c = append(c, fmt.Sprintf("Only these labels can be removed: %v.", config.Allowed))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	return c
}

func collectReplaceLabelConstraints(s *SafeOutputsConfig) []string {
	config := s.ReplaceLabel
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d label replacement(s) allowed.", templatableIntValue(config.Max)))
	}
	if len(config.AllowedTransitions) > 0 {
		pairs := make([]string, len(config.AllowedTransitions))
		for i, t := range config.AllowedTransitions {
			pairs[i] = fmt.Sprintf("%q → %q", t.From, t.To)
		}
		c = append(c, fmt.Sprintf("Only these label transitions are allowed: %s.", formatStringList(pairs)))
	}
	if len(config.AllowedAdd) > 0 {
		c = append(c, fmt.Sprintf("Only these labels can be added: %s.", formatStringList(config.AllowedAdd)))
	}
	if len(config.AllowedRemove) > 0 {
		c = append(c, fmt.Sprintf("Only these labels can be removed: %s.", formatStringList(config.AllowedRemove)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	return c
}

func collectAddReviewerConstraints(s *SafeOutputsConfig) []string {
	config := s.AddReviewer
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d reviewer(s) can be added.", templatableIntValue(config.Max)))
	}
	return c
}

func collectUpdateIssueConstraints(s *SafeOutputsConfig) []string {
	config := s.UpdateIssues
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d issue(s) can be updated.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	titlePrefix := config.TitlePrefix
	if config.RequiredTitlePrefix != "" {
		titlePrefix = config.RequiredTitlePrefix
	}
	if titlePrefix != "" {
		c = append(c, fmt.Sprintf("The target issue title must start with %q.", titlePrefix))
	}
	if config.Title != nil && *config.Title {
		c = append(c, "Title updates are allowed.")
	}
	if config.Body != nil && *config.Body {
		c = append(c, "Body updates are allowed.")
	}
	if config.Status != nil && *config.Status {
		c = append(c, "Status updates (open/closed) are allowed.")
	}
	return c
}

func collectUpdatePullRequestConstraints(s *SafeOutputsConfig) []string {
	config := s.UpdatePullRequests
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d pull request(s) can be updated.", templatableIntValue(config.Max)))
	}
	if config.Target != "" {
		c = append(c, fmt.Sprintf("Target: %s.", config.Target))
	}
	if len(config.RequiredLabels) > 0 {
		c = append(c, fmt.Sprintf("Only PRs with labels %v can be updated.", config.RequiredLabels))
	}
	if config.RequiredTitlePrefix != "" {
		c = append(c, fmt.Sprintf("Only PRs with title prefix %q can be updated.", config.RequiredTitlePrefix))
	}
	return c
}

func collectPushToPRBranchConstraints(s *SafeOutputsConfig) []string {
	config := s.PushToPullRequestBranch
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d push(es) can be made.", templatableIntValue(config.Max)))
	}
	if config.TitlePrefix != "" {
		c = append(c, fmt.Sprintf("The target pull request title must start with %q.", config.TitlePrefix))
	}
	return c
}

func collectUploadAssetConstraints(s *SafeOutputsConfig) []string {
	config := s.UploadAssets
	if config == nil {
		return nil
	}
	toolDescriptionEnhancerLog.Printf("Found upload_asset config: max=%d, maxSizeKB=%d, allowedExts=%v", config.Max, config.MaxSizeKB, config.AllowedExts)
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d asset(s) can be uploaded.", templatableIntValue(config.Max)))
	}
	if config.MaxSizeKB > 0 {
		c = append(c, fmt.Sprintf("Maximum file size: %dKB.", config.MaxSizeKB))
	}
	if len(config.AllowedExts) > 0 {
		c = append(c, fmt.Sprintf("Allowed file extensions: %v.", config.AllowedExts))
	}
	return c
}

func collectUpdateReleaseConstraints(s *SafeOutputsConfig) []string {
	config := s.UpdateRelease
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d release(s) can be updated.", templatableIntValue(config.Max)))
	}
	return c
}

func collectMissingToolConstraints(s *SafeOutputsConfig) []string {
	config := s.MissingTool
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d missing tool report(s) can be created.", templatableIntValue(config.Max)))
	}
	return c
}

func collectLinkSubIssueConstraints(s *SafeOutputsConfig) []string {
	config := s.LinkSubIssue
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d sub-issue link(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.ParentTitlePrefix != "" {
		c = append(c, fmt.Sprintf("The parent issue title must start with %q.", config.ParentTitlePrefix))
	}
	if config.SubTitlePrefix != "" {
		c = append(c, fmt.Sprintf("The sub-issue title must start with %q.", config.SubTitlePrefix))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Sub-issues will be linked in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		c = append(c, fmt.Sprintf("Sub-issue linking can target these repositories: %v.", config.AllowedRepos))
	}
	return c
}

func collectAssignMilestoneConstraints(s *SafeOutputsConfig) []string {
	config := s.AssignMilestone
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d milestone assignment(s) can be made.", templatableIntValue(config.Max)))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Milestones will be assigned in repository %q.", config.TargetRepoSlug))
	}
	return c
}

func collectAssignToAgentConstraints(s *SafeOutputsConfig) []string {
	config := s.AssignToAgent
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d issue(s) can be assigned to agent.", templatableIntValue(config.Max)))
	}
	if config.BaseBranch != "" {
		c = append(c, fmt.Sprintf("Pull requests will target the %q branch.", config.BaseBranch))
	}
	if config.TargetRepoSlug != "" {
		c = append(c, fmt.Sprintf("Issues will be assigned to agent in repository %q.", config.TargetRepoSlug))
	}
	if len(config.AllowedRepos) > 0 {
		c = append(c, fmt.Sprintf("Agent assignment can target these repositories: %v.", config.AllowedRepos))
	}
	return c
}

func collectUpdateProjectConstraints(s *SafeOutputsConfig) []string {
	config := s.UpdateProjects
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d project operation(s) can be performed.", templatableIntValue(config.Max)))
	}
	if config.Project != "" {
		c = append(c, fmt.Sprintf("Default project URL: %q.", config.Project))
	}
	return c
}

func collectCreateProjectStatusUpdateConstraints(s *SafeOutputsConfig) []string {
	config := s.CreateProjectStatusUpdates
	if config == nil {
		return nil
	}
	var c []string
	if templatableIntValue(config.Max) > 0 {
		c = append(c, fmt.Sprintf("Maximum %d status update(s) can be created.", templatableIntValue(config.Max)))
	}
	if config.Project != "" {
		c = append(c, fmt.Sprintf("Default project URL: %q.", config.Project))
	}
	return c
}
