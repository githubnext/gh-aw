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

	"github.com/githubnext/gh-aw/pkg/logger"
)

var scriptsLog = logger.New("workflow:scripts")

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

//go:embed js/assign_to_user.cjs
var assignToUserScriptSource string

//go:embed js/assign_copilot_to_created_issues.cjs
var assignCopilotToCreatedIssuesScriptSource string

//go:embed js/link_sub_issue.cjs
var linkSubIssueScriptSource string

//go:embed js/hide_comment.cjs
var hideCommentScriptSource string

//go:embed js/create_discussion.cjs
var createDiscussionScriptSource string

//go:embed js/close_discussion.cjs
var closeDiscussionScriptSource string

//go:embed js/close_expired_discussions.cjs
var closeExpiredDiscussionsScriptSource string

//go:embed js/close_expired_issues.cjs
var closeExpiredIssuesScriptSource string

//go:embed js/close_issue.cjs
var closeIssueScriptSource string

//go:embed js/close_pull_request.cjs
var closePullRequestScriptSource string

//go:embed js/update_issue.cjs
var updateIssueScriptSource string

//go:embed js/update_pull_request.cjs
var updatePullRequestScriptSource string

//go:embed js/update_discussion.cjs
var updateDiscussionScriptSource string

//go:embed js/update_pr_description_helpers.cjs
var updatePRDescriptionHelpersScriptSource string

//go:embed js/update_context_helpers.cjs
var updateContextHelpersScriptSource string

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

//go:embed js/generate_safe_inputs_config.cjs
var generateSafeInputsConfigScriptSource string

// Log parser source scripts
//
//go:embed js/parse_claude_log.cjs
var parseClaudeLogScriptSource string

//go:embed js/parse_codex_log.cjs
var parseCodexLogScriptSource string

//go:embed js/parse_copilot_log.cjs
var parseCopilotLogScriptSource string

// MCP server and transport scripts
//
//go:embed js/mcp_logger.cjs
var mcpLoggerScriptSource string

//go:embed js/mcp_http_transport.cjs
var mcpHTTPTransportScriptSource string

//go:embed js/substitute_placeholders.cjs
var substitutePlaceholdersScriptSource string

// Helper modules for bundling
//
//go:embed js/resolve_mentions_from_payload.cjs
var resolveMentionsFromPayloadScriptSource string

//go:embed js/sanitize_incoming_text.cjs
var sanitizeIncomingTextScriptSource string

//go:embed js/sanitize_content_core.cjs
var sanitizeContentCoreScriptSource string

// init registers all scripts with the DefaultScriptRegistry.
// Scripts are bundled lazily on first access via the getter functions.
func init() {
	scriptsLog.Print("Registering JavaScript scripts with DefaultScriptRegistry")
	// Safe output scripts
	DefaultScriptRegistry.Register("collect_jsonl_output", collectJSONLOutputScriptSource)
	DefaultScriptRegistry.Register("compute_text", computeTextScriptSource)
	DefaultScriptRegistry.Register("sanitize_output", sanitizeOutputScriptSource)
	DefaultScriptRegistry.Register("create_issue", createIssueScriptSource)
	DefaultScriptRegistry.Register("add_labels", addLabelsScriptSource)
	DefaultScriptRegistry.Register("add_reviewer", addReviewerScriptSource)
	DefaultScriptRegistry.Register("assign_milestone", assignMilestoneScriptSource)
	DefaultScriptRegistry.Register("assign_to_agent", assignToAgentScriptSource)
	DefaultScriptRegistry.Register("assign_to_user", assignToUserScriptSource)
	DefaultScriptRegistry.Register("assign_copilot_to_created_issues", assignCopilotToCreatedIssuesScriptSource)
	DefaultScriptRegistry.Register("link_sub_issue", linkSubIssueScriptSource)
	DefaultScriptRegistry.Register("hide_comment", hideCommentScriptSource)
	DefaultScriptRegistry.Register("create_discussion", createDiscussionScriptSource)
	DefaultScriptRegistry.Register("close_discussion", closeDiscussionScriptSource)
	DefaultScriptRegistry.Register("close_expired_discussions", closeExpiredDiscussionsScriptSource)
	DefaultScriptRegistry.Register("close_expired_issues", closeExpiredIssuesScriptSource)
	DefaultScriptRegistry.Register("close_issue", closeIssueScriptSource)
	DefaultScriptRegistry.Register("close_pull_request", closePullRequestScriptSource)
	DefaultScriptRegistry.Register("update_issue", updateIssueScriptSource)
	DefaultScriptRegistry.Register("update_pull_request", updatePullRequestScriptSource)
	DefaultScriptRegistry.Register("update_discussion", updateDiscussionScriptSource)
	DefaultScriptRegistry.Register("update_pr_description_helpers", updatePRDescriptionHelpersScriptSource)
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
	DefaultScriptRegistry.Register("generate_safe_inputs_config", generateSafeInputsConfigScriptSource)

	// Log parser scripts
	DefaultScriptRegistry.Register("parse_claude_log", parseClaudeLogScriptSource)
	DefaultScriptRegistry.Register("parse_codex_log", parseCodexLogScriptSource)
	DefaultScriptRegistry.Register("parse_copilot_log", parseCopilotLogScriptSource)

	// MCP server and transport scripts
	DefaultScriptRegistry.Register("mcp_logger", mcpLoggerScriptSource)
	DefaultScriptRegistry.Register("mcp_http_transport", mcpHTTPTransportScriptSource)

	// Template substitution scripts
	DefaultScriptRegistry.Register("substitute_placeholders", substitutePlaceholdersScriptSource)

	// Helper modules (for inlining via bundler)
	DefaultScriptRegistry.Register("resolve_mentions_from_payload", resolveMentionsFromPayloadScriptSource)
	DefaultScriptRegistry.Register("sanitize_incoming_text", sanitizeIncomingTextScriptSource)
	DefaultScriptRegistry.Register("sanitize_content_core", sanitizeContentCoreScriptSource)

	scriptsLog.Print("Completed script registration")
}

// Getter functions for bundled scripts.
// These use the ScriptRegistry for lazy bundling with caching.

// getCollectJSONLOutputScript returns the bundled collect_ndjson_output script
func getCollectJSONLOutputScript() string {
	return DefaultScriptRegistry.GetWithMode("collect_jsonl_output", RuntimeModeGitHubScript)
}

// getComputeTextScript returns the bundled compute_text script
func getComputeTextScript() string {
	return DefaultScriptRegistry.GetWithMode("compute_text", RuntimeModeGitHubScript)
}

// getSanitizeOutputScript returns the bundled sanitize_output script
func getSanitizeOutputScript() string {
	return DefaultScriptRegistry.GetWithMode("sanitize_output", RuntimeModeGitHubScript)
}

// getCreateIssueScript returns the bundled create_issue script
func getCreateIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("create_issue", RuntimeModeGitHubScript)
}

// getAddLabelsScript returns the bundled add_labels script
func getAddLabelsScript() string {
	return DefaultScriptRegistry.GetWithMode("add_labels", RuntimeModeGitHubScript)
}

// getAddReviewerScript returns the bundled add_reviewer script
func getAddReviewerScript() string {
	return DefaultScriptRegistry.GetWithMode("add_reviewer", RuntimeModeGitHubScript)
}

// getAssignMilestoneScript returns the bundled assign_milestone script
func getAssignMilestoneScript() string {
	return DefaultScriptRegistry.GetWithMode("assign_milestone", RuntimeModeGitHubScript)
}

// getAssignToAgentScript returns the bundled assign_to_agent script
func getAssignToAgentScript() string {
	return DefaultScriptRegistry.GetWithMode("assign_to_agent", RuntimeModeGitHubScript)
}

// getAssignToUserScript returns the bundled assign_to_user script
func getAssignToUserScript() string {
	return DefaultScriptRegistry.GetWithMode("assign_to_user", RuntimeModeGitHubScript)
}

// getAssignCopilotToCreatedIssuesScript returns the bundled assign_copilot_to_created_issues script
func getAssignCopilotToCreatedIssuesScript() string {
	return DefaultScriptRegistry.GetWithMode("assign_copilot_to_created_issues", RuntimeModeGitHubScript)
}

// getLinkSubIssueScript returns the bundled link_sub_issue script
func getLinkSubIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("link_sub_issue", RuntimeModeGitHubScript)
}

// getParseFirewallLogsScript returns the bundled parse_firewall_logs script
func getParseFirewallLogsScript() string {
	return DefaultScriptRegistry.GetWithMode("parse_firewall_logs", RuntimeModeGitHubScript)
}

// getCreateDiscussionScript returns the bundled create_discussion script
func getCreateDiscussionScript() string {
	return DefaultScriptRegistry.GetWithMode("create_discussion", RuntimeModeGitHubScript)
}

// getCloseDiscussionScript returns the bundled close_discussion script
func getCloseDiscussionScript() string {
	return DefaultScriptRegistry.GetWithMode("close_discussion", RuntimeModeGitHubScript)
}

// getCloseExpiredDiscussionsScript returns the bundled close_expired_discussions script
func getCloseExpiredDiscussionsScript() string {
	return DefaultScriptRegistry.GetWithMode("close_expired_discussions", RuntimeModeGitHubScript)
}

// getCloseExpiredIssuesScript returns the bundled close_expired_issues script
func getCloseExpiredIssuesScript() string {
	return DefaultScriptRegistry.GetWithMode("close_expired_issues", RuntimeModeGitHubScript)
}

// getCloseIssueScript returns the bundled close_issue script
func getCloseIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("close_issue", RuntimeModeGitHubScript)
}

// getClosePullRequestScript returns the bundled close_pull_request script
func getClosePullRequestScript() string {
	return DefaultScriptRegistry.GetWithMode("close_pull_request", RuntimeModeGitHubScript)
}

// getUpdateIssueScript returns the bundled update_issue script
func getUpdateIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("update_issue", RuntimeModeGitHubScript)
}

// getUpdatePullRequestScript returns the bundled update_pull_request script
func getUpdatePullRequestScript() string {
	return DefaultScriptRegistry.GetWithMode("update_pull_request", RuntimeModeGitHubScript)
}

// getUpdateDiscussionScript returns the bundled update_discussion script
func getUpdateDiscussionScript() string {
	return DefaultScriptRegistry.GetWithMode("update_discussion", RuntimeModeGitHubScript)
}

// getUpdateReleaseScript returns the bundled update_release script
func getUpdateReleaseScript() string {
	return DefaultScriptRegistry.GetWithMode("update_release", RuntimeModeGitHubScript)
}

// getCreateCodeScanningAlertScript returns the bundled create_code_scanning_alert script
func getCreateCodeScanningAlertScript() string {
	return DefaultScriptRegistry.GetWithMode("create_code_scanning_alert", RuntimeModeGitHubScript)
}

// getCreatePRReviewCommentScript returns the bundled create_pr_review_comment script
func getCreatePRReviewCommentScript() string {
	return DefaultScriptRegistry.GetWithMode("create_pr_review_comment", RuntimeModeGitHubScript)
}

// getAddCommentScript returns the bundled add_comment script
func getAddCommentScript() string {
	return DefaultScriptRegistry.GetWithMode("add_comment", RuntimeModeGitHubScript)
}

// getUploadAssetsScript returns the bundled upload_assets script
func getUploadAssetsScript() string {
	return DefaultScriptRegistry.GetWithMode("upload_assets", RuntimeModeGitHubScript)
}

// getPushToPullRequestBranchScript returns the bundled push_to_pull_request_branch script
func getPushToPullRequestBranchScript() string {
	return DefaultScriptRegistry.GetWithMode("push_to_pull_request_branch", RuntimeModeGitHubScript)
}

// getCreatePullRequestScript returns the bundled create_pull_request script
func getCreatePullRequestScript() string {
	return DefaultScriptRegistry.GetWithMode("create_pull_request", RuntimeModeGitHubScript)
}

// getNotifyCommentErrorScript returns the bundled notify_comment_error script
func getNotifyCommentErrorScript() string {
	return DefaultScriptRegistry.GetWithMode("notify_comment_error", RuntimeModeGitHubScript)
}

// getNoOpScript returns the bundled noop script
func getNoOpScript() string {
	return DefaultScriptRegistry.GetWithMode("noop", RuntimeModeGitHubScript)
}

// getInterpolatePromptScript returns the bundled interpolate_prompt script
func getInterpolatePromptScript() string {
	return DefaultScriptRegistry.GetWithMode("interpolate_prompt", RuntimeModeGitHubScript)
}

// getParseClaudeLogScript returns the bundled parse_claude_log script
func getParseClaudeLogScript() string {
	return DefaultScriptRegistry.GetWithMode("parse_claude_log", RuntimeModeGitHubScript)
}

// getParseCodexLogScript returns the bundled parse_codex_log script
func getParseCodexLogScript() string {
	return DefaultScriptRegistry.GetWithMode("parse_codex_log", RuntimeModeGitHubScript)
}

// getParseCopilotLogScript returns the bundled parse_copilot_log script
func getParseCopilotLogScript() string {
	return DefaultScriptRegistry.GetWithMode("parse_copilot_log", RuntimeModeGitHubScript)
}

// getGenerateSafeInputsConfigScript returns the bundled generate_safe_inputs_config script
func getGenerateSafeInputsConfigScript() string {
	return DefaultScriptRegistry.GetWithMode("generate_safe_inputs_config", RuntimeModeGitHubScript)
}

// getSubstitutePlaceholdersScript returns the bundled substitute_placeholders script
func getSubstitutePlaceholdersScript() string {
	return DefaultScriptRegistry.GetWithMode("substitute_placeholders", RuntimeModeGitHubScript)
}

// getRedactSecretsScript returns the bundled redact_secrets script
func getRedactSecretsScript() string {
	return DefaultScriptRegistry.GetWithMode("redact_secrets", RuntimeModeGitHubScript)
}
