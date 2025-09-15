package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed js/create_pull_request.cjs
var createPullRequestScript string

//go:embed js/create_issue.cjs
var createIssueScript string

//go:embed js/create_discussion.cjs
var createDiscussionScript string

//go:embed js/create_comment.cjs
var createCommentScript string

//go:embed js/create_pr_review_comment.cjs
var createPRReviewCommentScript string

//go:embed js/create_code_scanning_alert.cjs
var createCodeScanningAlertScript string

//go:embed js/compute_text.cjs
var computeTextScript string

//go:embed js/collect_ndjson_output.cjs
var collectJSONLOutputScript string

//go:embed js/add_labels.cjs
var addLabelsScript string

//go:embed js/update_issue.cjs
var updateIssueScript string

//go:embed js/push_to_pr_branch.cjs
var pushToBranchScript string

//go:embed js/push_to_orphaned_branch.cjs
var pushToOrphanedBranchScript string

//go:embed js/setup_agent_output.cjs
var setupAgentOutputScript string

//go:embed js/add_reaction.cjs
var addReactionScript string

//go:embed js/add_reaction_and_edit_comment.cjs
var addReactionAndEditCommentScript string

//go:embed js/check_permissions.cjs
var checkPermissionsScript string

//go:embed js/parse_claude_log.cjs
var parseClaudeLogScript string

//go:embed js/parse_codex_log.cjs
var parseCodexLogScript string

//go:embed js/validate_errors.cjs
var validateErrorsScript string

//go:embed js/missing_tool.cjs
var missingToolScript string

//go:embed js/safe_outputs_mcp_server.cjs
var safeOutputsMCPServerScript string

// FormatJavaScriptForYAML formats a JavaScript script with proper indentation for embedding in YAML
func FormatJavaScriptForYAML(script string) []string {
	var formattedLines []string
	scriptLines := strings.Split(script, "\n")
	for _, line := range scriptLines {
		// Skip empty lines when inlining to YAML
		if strings.TrimSpace(line) != "" {
			formattedLines = append(formattedLines, fmt.Sprintf("            %s\n", line))
		}
	}
	return formattedLines
}

// WriteJavaScriptToYAML writes a JavaScript script with proper indentation to a strings.Builder
func WriteJavaScriptToYAML(yaml *strings.Builder, script string) {
	scriptLines := strings.Split(script, "\n")
	for _, line := range scriptLines {
		// Skip empty lines when inlining to YAML
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(yaml, "            %s\n", line)
		}
	}
}

// GetLogParserScript returns the JavaScript content for a log parser by name
func GetLogParserScript(name string) string {
	switch name {
	case "parse_claude_log":
		return parseClaudeLogScript
	case "parse_codex_log":
		return parseCodexLogScript
	case "validate_errors":
		return validateErrorsScript
	default:
		return ""
	}
}

// GetSafeOutputsMCPServerScript returns the JavaScript content for the safe-outputs MCP server
func GetSafeOutputsMCPServerScript() string {
	return safeOutputsMCPServerScript
}
