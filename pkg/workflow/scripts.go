// Package workflow provides embedded JavaScript scripts for GitHub Actions workflows.
//
// # Script Registry Pattern
//
// This file uses the ScriptRegistry pattern to manage lazy bundling of JavaScript
// scripts. Instead of having separate sync.Once patterns for each script, all scripts
// are registered with the DefaultScriptRegistry and bundled on-demand.
//
// See script_registry.go for the ScriptRegistry implementation.
package workflow

import (
	_ "embed"
)

// Source scripts that may contain local requires
//
//go:embed js/collect_ndjson_output.cjs
var collectJSONLOutputScriptSource string

//go:embed js/compute_text.cjs
var computeTextScriptSource string

//go:embed js/sanitize_output.cjs
var sanitizeOutputScriptSource string

//go:embed js/create_issue.cjs
var createIssueScriptSource string

//go:embed js/add_labels.cjs
var addLabelsScriptSource string

//go:embed js/add_reviewer.cjs
var addReviewerScriptSource string

//go:embed js/assign_milestone.cjs
var assignMilestoneScriptSource string

//go:embed js/assign_to_agent.cjs
var assignToAgentScriptSource string

//go:embed js/link_sub_issue.cjs
var linkSubIssueScriptSource string

//go:embed js/create_discussion.cjs
var createDiscussionScriptSource string

//go:embed js/close_discussion.cjs
var closeDiscussionScriptSource string

//go:embed js/close_issue.cjs
var closeIssueScriptSource string

//go:embed js/close_pull_request.cjs
var closePullRequestScriptSource string

//go:embed js/update_issue.cjs
var updateIssueScriptSource string

//go:embed js/update_release.cjs
var updateReleaseScriptSource string

//go:embed js/create_code_scanning_alert.cjs
var createCodeScanningAlertScriptSource string

//go:embed js/create_pr_review_comment.cjs
var createPRReviewCommentScriptSource string

//go:embed js/add_comment.cjs
var addCommentScriptSource string

//go:embed js/upload_assets.cjs
var uploadAssetsScriptSource string

//go:embed js/parse_firewall_logs.cjs
var parseFirewallLogsScriptSource string

//go:embed js/push_to_pull_request_branch.cjs
var pushToPullRequestBranchScriptSource string

//go:embed js/create_pull_request.cjs
var createPullRequestScriptSource string

//go:embed js/notify_comment_error.cjs
var notifyCommentErrorScriptSource string

//go:embed js/noop.cjs
var noopScriptSource string

// Log parser source scripts
//
//go:embed js/parse_claude_log.cjs
var parseClaudeLogScriptSource string

//go:embed js/parse_codex_log.cjs
var parseCodexLogScriptSource string

//go:embed js/parse_copilot_log.cjs
var parseCopilotLogScriptSource string

// init registers all scripts with the DefaultScriptRegistry.
// Scripts are bundled lazily on first access via the getter functions.
func init() {
	// Safe output scripts
	DefaultScriptRegistry.Register("collect_jsonl_output", collectJSONLOutputScriptSource)
	DefaultScriptRegistry.Register("compute_text", computeTextScriptSource)
	DefaultScriptRegistry.Register("sanitize_output", sanitizeOutputScriptSource)
	DefaultScriptRegistry.Register("create_issue", createIssueScriptSource)
	DefaultScriptRegistry.Register("add_labels", addLabelsScriptSource)
	DefaultScriptRegistry.Register("add_reviewer", addReviewerScriptSource)
	DefaultScriptRegistry.Register("assign_milestone", assignMilestoneScriptSource)
	DefaultScriptRegistry.Register("assign_to_agent", assignToAgentScriptSource)
	DefaultScriptRegistry.Register("link_sub_issue", linkSubIssueScriptSource)
	DefaultScriptRegistry.Register("create_discussion", createDiscussionScriptSource)
	DefaultScriptRegistry.Register("close_discussion", closeDiscussionScriptSource)
	DefaultScriptRegistry.Register("close_issue", closeIssueScriptSource)
	DefaultScriptRegistry.Register("close_pull_request", closePullRequestScriptSource)
	DefaultScriptRegistry.Register("update_issue", updateIssueScriptSource)
	DefaultScriptRegistry.Register("update_release", updateReleaseScriptSource)
	DefaultScriptRegistry.Register("create_code_scanning_alert", createCodeScanningAlertScriptSource)
	DefaultScriptRegistry.Register("create_pr_review_comment", createPRReviewCommentScriptSource)
	DefaultScriptRegistry.Register("add_comment", addCommentScriptSource)
	DefaultScriptRegistry.Register("upload_assets", uploadAssetsScriptSource)
	DefaultScriptRegistry.Register("parse_firewall_logs", parseFirewallLogsScriptSource)
	DefaultScriptRegistry.Register("push_to_pull_request_branch", pushToPullRequestBranchScriptSource)
	DefaultScriptRegistry.Register("create_pull_request", createPullRequestScriptSource)
	DefaultScriptRegistry.Register("notify_comment_error", notifyCommentErrorScriptSource)
	DefaultScriptRegistry.Register("noop", noopScriptSource)

	// Log parser scripts
	DefaultScriptRegistry.Register("parse_claude_log", parseClaudeLogScriptSource)
	DefaultScriptRegistry.Register("parse_codex_log", parseCodexLogScriptSource)
	DefaultScriptRegistry.Register("parse_copilot_log", parseCopilotLogScriptSource)
}

// Getter functions for bundled scripts.
// These use the ScriptRegistry for lazy bundling with caching.

// getCollectJSONLOutputScript returns the bundled collect_ndjson_output script
func getCollectJSONLOutputScript() string {
	return DefaultScriptRegistry.Get("collect_jsonl_output")
}

// getComputeTextScript returns the bundled compute_text script
func getComputeTextScript() string {
	return DefaultScriptRegistry.Get("compute_text")
}

// getSanitizeOutputScript returns the bundled sanitize_output script
func getSanitizeOutputScript() string {
	return DefaultScriptRegistry.Get("sanitize_output")
}

// getCreateIssueScript returns the bundled create_issue script
func getCreateIssueScript() string {
	return DefaultScriptRegistry.Get("create_issue")
}

// getAddLabelsScript returns the bundled add_labels script
func getAddLabelsScript() string {
	return DefaultScriptRegistry.Get("add_labels")
}

// getAddReviewerScript returns the bundled add_reviewer script
func getAddReviewerScript() string {
	return DefaultScriptRegistry.Get("add_reviewer")
}

// getAssignMilestoneScript returns the bundled assign_milestone script
func getAssignMilestoneScript() string {
	return DefaultScriptRegistry.Get("assign_milestone")
}

// getAssignToAgentScript returns the bundled assign_to_agent script
func getAssignToAgentScript() string {
	return DefaultScriptRegistry.Get("assign_to_agent")
}

// getLinkSubIssueScript returns the bundled link_sub_issue script
func getLinkSubIssueScript() string {
	return DefaultScriptRegistry.Get("link_sub_issue")
}

// getParseFirewallLogsScript returns the bundled parse_firewall_logs script
func getParseFirewallLogsScript() string {
	return DefaultScriptRegistry.Get("parse_firewall_logs")
}

// getCreateDiscussionScript returns the bundled create_discussion script
func getCreateDiscussionScript() string {
	return DefaultScriptRegistry.Get("create_discussion")
}

// getCloseDiscussionScript returns the bundled close_discussion script
func getCloseDiscussionScript() string {
	return DefaultScriptRegistry.Get("close_discussion")
}

// getCloseIssueScript returns the bundled close_issue script
func getCloseIssueScript() string {
	return DefaultScriptRegistry.Get("close_issue")
}

// getClosePullRequestScript returns the bundled close_pull_request script
func getClosePullRequestScript() string {
	return DefaultScriptRegistry.Get("close_pull_request")
}

// getUpdateIssueScript returns the bundled update_issue script
func getUpdateIssueScript() string {
	return DefaultScriptRegistry.Get("update_issue")
}

// getUpdateReleaseScript returns the bundled update_release script
func getUpdateReleaseScript() string {
	return DefaultScriptRegistry.Get("update_release")
}

// getCreateCodeScanningAlertScript returns the bundled create_code_scanning_alert script
func getCreateCodeScanningAlertScript() string {
	return DefaultScriptRegistry.Get("create_code_scanning_alert")
}

// getCreatePRReviewCommentScript returns the bundled create_pr_review_comment script
func getCreatePRReviewCommentScript() string {
	return DefaultScriptRegistry.Get("create_pr_review_comment")
}

// getAddCommentScript returns the bundled add_comment script
func getAddCommentScript() string {
	return DefaultScriptRegistry.Get("add_comment")
}

// getUploadAssetsScript returns the bundled upload_assets script
func getUploadAssetsScript() string {
	return DefaultScriptRegistry.Get("upload_assets")
}

// getPushToPullRequestBranchScript returns the bundled push_to_pull_request_branch script
func getPushToPullRequestBranchScript() string {
	return DefaultScriptRegistry.Get("push_to_pull_request_branch")
}

// getCreatePullRequestScript returns the bundled create_pull_request script
func getCreatePullRequestScript() string {
	return DefaultScriptRegistry.Get("create_pull_request")
}

// getNotifyCommentErrorScript returns the bundled notify_comment_error script
func getNotifyCommentErrorScript() string {
	return DefaultScriptRegistry.Get("notify_comment_error")
}

// getNoOpScript returns the bundled noop script
func getNoOpScript() string {
	return DefaultScriptRegistry.Get("noop")
}

// getInterpolatePromptScript returns the bundled interpolate_prompt script
func getInterpolatePromptScript() string {
	return DefaultScriptRegistry.Get("interpolate_prompt")
}

// getParseClaudeLogScript returns the bundled parse_claude_log script
func getParseClaudeLogScript() string {
	return DefaultScriptRegistry.Get("parse_claude_log")
}

// getParseCodexLogScript returns the bundled parse_codex_log script
func getParseCodexLogScript() string {
	return DefaultScriptRegistry.Get("parse_codex_log")
}

// getParseCopilotLogScript returns the bundled parse_copilot_log script
func getParseCopilotLogScript() string {
	return DefaultScriptRegistry.Get("parse_copilot_log")
}
