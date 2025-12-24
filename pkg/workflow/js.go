package workflow

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var jsLog = logger.New("workflow:js")

//go:embed js/create_agent_task.cjs
var createAgentTaskScript string

//go:embed js/assign_issue.cjs
var assignIssueScriptSource string

//go:embed js/add_copilot_reviewer.cjs
var addCopilotReviewerScriptSource string

//go:embed js/add_reaction_and_edit_comment.cjs
var addReactionAndEditCommentScriptSource string

//go:embed js/check_membership.cjs
var checkMembershipScriptSource string

// init registers scripts from js.go with the DefaultScriptRegistry
func init() {
	DefaultScriptRegistry.Register("check_membership", checkMembershipScriptSource)
	DefaultScriptRegistry.Register("safe_outputs_mcp_server", safeOutputsMCPServerScriptSource)
	DefaultScriptRegistry.Register("update_project", updateProjectScriptSource)
	DefaultScriptRegistry.Register("interpolate_prompt", interpolatePromptScript)
	DefaultScriptRegistry.Register("assign_issue", assignIssueScriptSource)
	DefaultScriptRegistry.Register("add_copilot_reviewer", addCopilotReviewerScriptSource)
	DefaultScriptRegistry.Register("add_reaction_and_edit_comment", addReactionAndEditCommentScriptSource)
	DefaultScriptRegistry.Register("redact_secrets", redactSecretsScript)
}

// getAddReactionAndEditCommentScript returns the bundled add_reaction_and_edit_comment script
func getAddReactionAndEditCommentScript() string {
	return DefaultScriptRegistry.GetWithMode("add_reaction_and_edit_comment", RuntimeModeGitHubScript)
}

// getAssignIssueScript returns the bundled assign_issue script
func getAssignIssueScript() string {
	return DefaultScriptRegistry.GetWithMode("assign_issue", RuntimeModeGitHubScript)
}

// getAddCopilotReviewerScript returns the bundled add_copilot_reviewer script
func getAddCopilotReviewerScript() string {
	return DefaultScriptRegistry.GetWithMode("add_copilot_reviewer", RuntimeModeGitHubScript)
}

// getCheckMembershipScript returns the bundled check_membership script
func getCheckMembershipScript() string {
	return DefaultScriptRegistry.GetWithMode("check_membership", RuntimeModeGitHubScript)
}

//go:embed js/check_stop_time.cjs
var checkStopTimeScript string

//go:embed js/check_skip_if_match.cjs
var checkSkipIfMatchScript string

//go:embed js/check_command_position.cjs
var checkCommandPositionScript string

//go:embed js/check_workflow_timestamp_api.cjs
var checkWorkflowTimestampAPIScript string

//go:embed js/compute_text.cjs
var computeTextScript string

//go:embed js/log_parser_bootstrap.cjs
var logParserBootstrapScript string

//go:embed js/validate_errors.cjs
var validateErrorsScript string

//go:embed js/missing_tool.cjs
var missingToolScript string

//go:embed js/safe_outputs_mcp_server.cjs
var safeOutputsMCPServerScriptSource string

// getSafeOutputsMCPServerScript returns the bundled safe_outputs_mcp_server script
func getSafeOutputsMCPServerScript() string {
	return DefaultScriptRegistry.GetWithMode("safe_outputs_mcp_server", RuntimeModeGitHubScript)
}

//go:embed js/safe_outputs_mcp_server_entry_point.cjs
var safeOutputsMCPServerEntryPointScript string

//go:embed js/safe_outputs_tools.json
var safeOutputsToolsJSON string

//go:embed js/interpolate_prompt.cjs
var interpolatePromptScript string

//go:embed js/runtime_import.cjs
var runtimeImportScript string

//go:embed js/checkout_pr_branch.cjs
var checkoutPRBranchScript string

//go:embed js/redact_secrets.cjs
var redactSecretsScript string

//go:embed js/sanitize_content.cjs
var sanitizeContentScript string

//go:embed js/resolve_mentions.cjs
var resolveMentionsScript string

//go:embed js/sanitize_label_content.cjs
var sanitizeLabelContentScript string

//go:embed js/sanitize_workflow_name.cjs
var sanitizeWorkflowNameScript string

//go:embed js/load_agent_output.cjs
var loadAgentOutputScript string

//go:embed js/lock-issue.cjs
var lockIssueScript string

//go:embed js/unlock-issue.cjs
var unlockIssueScript string

//go:embed js/staged_preview.cjs
var stagedPreviewScript string

//go:embed js/assign_agent_helpers.cjs
var assignAgentHelpersScript string

//go:embed js/safe_output_helpers.cjs
var safeOutputHelpersScript string

//go:embed js/safe_output_validator.cjs
var safeOutputValidatorScript string

//go:embed js/safe_output_processor.cjs
var safeOutputProcessorScript string

//go:embed js/is_truthy.cjs
var isTruthyScript string

//go:embed js/log_parser_shared.cjs
var logParserSharedScript string

//go:embed js/update_activation_comment.cjs
var updateActivationCommentScript string

//go:embed js/update_project.cjs
var updateProjectScriptSource string

// getUpdateProjectScript returns the bundled update_project script
func getUpdateProjectScript() string {
	return DefaultScriptRegistry.GetWithMode("update_project", RuntimeModeGitHubScript)
}

//go:embed js/generate_footer.cjs
var generateFooterScript string

//go:embed js/get_tracker_id.cjs
var getTrackerIDScript string

//go:embed js/push_repo_memory.cjs
var pushRepoMemoryScript string

//go:embed js/messages.cjs
var messagesScript string

//go:embed js/messages_core.cjs
var messagesCoreScript string

//go:embed js/messages_footer.cjs
var messagesFooterScript string

//go:embed js/messages_staged.cjs
var messagesStagedScript string

//go:embed js/messages_run_status.cjs
var messagesRunStatusScript string

//go:embed js/messages_close_discussion.cjs
var messagesCloseDiscussionScript string

//go:embed js/close_older_discussions.cjs
var closeOlderDiscussionsScript string

//go:embed js/close_entity_helpers.cjs
var closeEntityHelpersScript string

//go:embed js/expiration_helpers.cjs
var expirationHelpersScript string

//go:embed js/get_repository_url.cjs
var getRepositoryUrlScript string

//go:embed js/check_permissions_utils.cjs
var checkPermissionsUtilsScript string

//go:embed js/normalize_branch_name.cjs
var normalizeBranchNameScript string

//go:embed js/estimate_tokens.cjs
var estimateTokensScript string

//go:embed js/generate_compact_schema.cjs
var generateCompactSchemaScript string

//go:embed js/write_large_content_to_file.cjs
var writeLargeContentToFileScript string

//go:embed js/get_current_branch.cjs
var getCurrentBranchScript string

//go:embed js/get_base_branch.cjs
var getBaseBranchScript string

//go:embed js/generate_git_patch.cjs
var generateGitPatchJSScript string

//go:embed js/temporary_id.cjs
var temporaryIdScript string

//go:embed js/update_runner.cjs
var updateRunnerScript string

//go:embed js/update_pr_description_helpers.cjs
var updatePRDescriptionHelpersScript string

//go:embed js/read_buffer.cjs
var readBufferScript string

//go:embed js/mcp_server_core.cjs
var mcpServerCoreScript string

//go:embed js/safe_inputs_mcp_server.cjs
var safeInputsMCPServerScript string

//go:embed js/safe_inputs_mcp_server_http.cjs
var safeInputsMCPServerHTTPScript string

//go:embed js/safe_inputs_config_loader.cjs
var safeInputsConfigLoaderScript string

//go:embed js/safe_inputs_bootstrap.cjs
var safeInputsBootstrapScript string

//go:embed js/safe_inputs_tool_factory.cjs
var safeInputsToolFactoryScript string

//go:embed js/safe_inputs_validation.cjs
var safeInputsValidationScript string

//go:embed js/mcp_handler_shell.cjs
var mcpHandlerShellScript string

//go:embed js/mcp_handler_python.cjs
var mcpHandlerPythonScript string

//go:embed js/safe_output_type_validator.cjs
var safeOutputTypeValidatorScript string

//go:embed js/repo_helpers.cjs
var repoHelpersScript string

//go:embed js/remove_duplicate_title.cjs
var removeDuplicateTitleScript string

//go:embed js/safe_outputs_config.cjs
var safeOutputsConfigScript string

//go:embed js/safe_outputs_append.cjs
var safeOutputsAppendScript string

//go:embed js/safe_outputs_handlers.cjs
var safeOutputsHandlersScript string

//go:embed js/safe_outputs_tools_loader.cjs
var safeOutputsToolsLoaderScript string

//go:embed js/safe_outputs_bootstrap.cjs
var safeOutputsBootstrapScript string

//go:embed js/resolve_mentions_from_payload.cjs
var resolveMentionsFromPayloadScript string

//go:embed js/sanitize_incoming_text.cjs
var sanitizeIncomingTextScript string

//go:embed js/sanitize_content_core.cjs
var sanitizeContentCoreScript string

// GetJavaScriptSources returns a map of all embedded JavaScript sources
// The keys are the relative paths from the js directory
// NOTE: This includes scripts from both js.go and scripts.go
func GetJavaScriptSources() map[string]string {
	return map[string]string{
		"sanitize_content.cjs":              sanitizeContentScript,
		"resolve_mentions.cjs":              resolveMentionsScript,
		"sanitize_label_content.cjs":        sanitizeLabelContentScript,
		"sanitize_workflow_name.cjs":        sanitizeWorkflowNameScript,
		"load_agent_output.cjs":             loadAgentOutputScript,
		"lock-issue.cjs":                    lockIssueScript,
		"unlock-issue.cjs":                  unlockIssueScript,
		"staged_preview.cjs":                stagedPreviewScript,
		"assign_agent_helpers.cjs":          assignAgentHelpersScript,
		"safe_output_helpers.cjs":           safeOutputHelpersScript,
		"safe_output_validator.cjs":         safeOutputValidatorScript,
		"safe_output_processor.cjs":         safeOutputProcessorScript,
		"temporary_id.cjs":                  temporaryIdScript,
		"is_truthy.cjs":                     isTruthyScript,
		"log_parser_bootstrap.cjs":          logParserBootstrapScript,
		"log_parser_shared.cjs":             logParserSharedScript,
		"update_activation_comment.cjs":     updateActivationCommentScript,
		"generate_footer.cjs":               generateFooterScript,
		"get_tracker_id.cjs":                getTrackerIDScript,
		"messages.cjs":                      messagesScript,
		"messages_core.cjs":                 messagesCoreScript,
		"messages_footer.cjs":               messagesFooterScript,
		"messages_staged.cjs":               messagesStagedScript,
		"messages_run_status.cjs":           messagesRunStatusScript,
		"messages_close_discussion.cjs":     messagesCloseDiscussionScript,
		"close_older_discussions.cjs":       closeOlderDiscussionsScript,
		"close_entity_helpers.cjs":          closeEntityHelpersScript,
		"expiration_helpers.cjs":            expirationHelpersScript,
		"get_repository_url.cjs":            getRepositoryUrlScript,
		"check_permissions_utils.cjs":       checkPermissionsUtilsScript,
		"normalize_branch_name.cjs":         normalizeBranchNameScript,
		"estimate_tokens.cjs":               estimateTokensScript,
		"generate_compact_schema.cjs":       generateCompactSchemaScript,
		"write_large_content_to_file.cjs":   writeLargeContentToFileScript,
		"get_current_branch.cjs":            getCurrentBranchScript,
		"get_base_branch.cjs":               getBaseBranchScript,
		"generate_git_patch.cjs":            generateGitPatchJSScript,
		"update_runner.cjs":                 updateRunnerScript,
		"update_pr_description_helpers.cjs": updatePRDescriptionHelpersScript,
		"update_context_helpers.cjs":        updateContextHelpersScriptSource,
		"read_buffer.cjs":                   readBufferScript,
		"mcp_server_core.cjs":               mcpServerCoreScript,
		"mcp_http_transport.cjs":            mcpHTTPTransportScriptSource,
		"mcp_logger.cjs":                    mcpLoggerScriptSource,
		"safe_inputs_mcp_server.cjs":        safeInputsMCPServerScript,
		"safe_inputs_mcp_server_http.cjs":   safeInputsMCPServerHTTPScript,
		"safe_inputs_config_loader.cjs":     safeInputsConfigLoaderScript,
		"safe_inputs_bootstrap.cjs":         safeInputsBootstrapScript,
		"safe_inputs_tool_factory.cjs":      safeInputsToolFactoryScript,
		"safe_inputs_validation.cjs":        safeInputsValidationScript,
		"mcp_handler_shell.cjs":             mcpHandlerShellScript,
		"mcp_handler_python.cjs":            mcpHandlerPythonScript,
		"safe_output_type_validator.cjs":    safeOutputTypeValidatorScript,
		"repo_helpers.cjs":                  repoHelpersScript,
		"remove_duplicate_title.cjs":        removeDuplicateTitleScript,
		"safe_outputs_config.cjs":           safeOutputsConfigScript,
		"safe_outputs_append.cjs":           safeOutputsAppendScript,
		"safe_outputs_handlers.cjs":         safeOutputsHandlersScript,
		"safe_outputs_tools_loader.cjs":     safeOutputsToolsLoaderScript,
		"safe_outputs_tools.json":           safeOutputsToolsJSON,
		"safe_outputs_bootstrap.cjs":              safeOutputsBootstrapScript,
		"safe_outputs_mcp_server.cjs":             safeOutputsMCPServerScriptSource,
		"safe_outputs_mcp_server_entry_point.cjs": safeOutputsMCPServerEntryPointScript,
		"resolve_mentions_from_payload.cjs":       resolveMentionsFromPayloadScript,
		"sanitize_incoming_text.cjs":        sanitizeIncomingTextScript,
		"sanitize_content_core.cjs":         sanitizeContentCoreScript,
		"add_copilot_reviewer.cjs":          addCopilotReviewerScriptSource,
		"add_reaction_and_edit_comment.cjs": addReactionAndEditCommentScriptSource,
		"assign_issue.cjs":                  assignIssueScriptSource,
		"check_command_position.cjs":        checkCommandPositionScript,
		"check_membership.cjs":              checkMembershipScriptSource,
		"check_skip_if_match.cjs":           checkSkipIfMatchScript,
		"check_stop_time.cjs":               checkStopTimeScript,
		"check_workflow_timestamp_api.cjs":  checkWorkflowTimestampAPIScript,
		"checkout_pr_branch.cjs":            checkoutPRBranchScript,
		"compute_text.cjs":                  computeTextScript,
		"create_agent_task.cjs":             createAgentTaskScript,
		"interpolate_prompt.cjs":            interpolatePromptScript,
		"runtime_import.cjs":                runtimeImportScript,
		"missing_tool.cjs":                  missingToolScript,
		"push_repo_memory.cjs":              pushRepoMemoryScript,
		"redact_secrets.cjs":                redactSecretsScript,
		"update_project.cjs":                updateProjectScriptSource,
		"validate_errors.cjs":               validateErrorsScript,
		// Scripts from scripts.go
		"noop.cjs":                             noopScriptSource,
		"notify_comment_error.cjs":             notifyCommentErrorScriptSource,
		"collect_ndjson_output.cjs":            collectJSONLOutputScriptSource,
		"sanitize_output.cjs":                  sanitizeOutputScriptSource,
		"create_issue.cjs":                     createIssueScriptSource,
		"add_labels.cjs":                       addLabelsScriptSource,
		"add_reviewer.cjs":                     addReviewerScriptSource,
		"assign_milestone.cjs":                 assignMilestoneScriptSource,
		"assign_to_agent.cjs":                  assignToAgentScriptSource,
		"assign_to_user.cjs":                   assignToUserScriptSource,
		"assign_copilot_to_created_issues.cjs": assignCopilotToCreatedIssuesScriptSource,
		"link_sub_issue.cjs":                   linkSubIssueScriptSource,
		"hide_comment.cjs":                     hideCommentScriptSource,
		"create_discussion.cjs":                createDiscussionScriptSource,
		"close_discussion.cjs":                 closeDiscussionScriptSource,
		"close_expired_discussions.cjs":        closeExpiredDiscussionsScriptSource,
		"close_expired_issues.cjs":             closeExpiredIssuesScriptSource,
		"close_issue.cjs":                      closeIssueScriptSource,
		"close_pull_request.cjs":               closePullRequestScriptSource,
		"update_issue.cjs":                     updateIssueScriptSource,
		"update_pull_request.cjs":              updatePullRequestScriptSource,
		"update_discussion.cjs":                updateDiscussionScriptSource,
		"update_release.cjs":                   updateReleaseScriptSource,
		"create_code_scanning_alert.cjs":       createCodeScanningAlertScriptSource,
		"create_pr_review_comment.cjs":         createPRReviewCommentScriptSource,
		"add_comment.cjs":                      addCommentScriptSource,
		"upload_assets.cjs":                    uploadAssetsScriptSource,
		"parse_firewall_logs.cjs":              parseFirewallLogsScriptSource,
		"push_to_pull_request_branch.cjs":      pushToPullRequestBranchScriptSource,
		"create_pull_request.cjs":              createPullRequestScriptSource,
		"generate_safe_inputs_config.cjs":      generateSafeInputsConfigScriptSource,
		"parse_claude_log.cjs":                 parseClaudeLogScriptSource,
		"parse_codex_log.cjs":                  parseCodexLogScriptSource,
		"parse_copilot_log.cjs":                parseCopilotLogScriptSource,
		"substitute_placeholders.cjs":          substitutePlaceholdersScriptSource,
	}
}

// removeJavaScriptComments removes JavaScript comments (// and /* */) from code
// while preserving comments that appear within string literals
func removeJavaScriptComments(code string) string {
	if jsLog.Enabled() {
		jsLog.Printf("Removing JavaScript comments from %d bytes of code", len(code))
	}
	var result strings.Builder
	lines := strings.Split(code, "\n")

	inBlockComment := false

	for _, line := range lines {
		processedLine := removeJavaScriptCommentsFromLine(line, &inBlockComment)
		result.WriteString(processedLine)
		result.WriteString("\n")
	}

	// Remove the trailing newline we added
	resultStr := result.String()
	if len(resultStr) > 0 && resultStr[len(resultStr)-1] == '\n' {
		resultStr = resultStr[:len(resultStr)-1]
	}

	if jsLog.Enabled() {
		jsLog.Printf("Removed comments, result: %d bytes", len(resultStr))
	}
	return resultStr
}

// removeJavaScriptCommentsFromLine removes JavaScript comments from a single line
// while preserving comments that appear within string literals and regex literals
func removeJavaScriptCommentsFromLine(line string, inBlockComment *bool) string {
	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		if *inBlockComment {
			// Look for end of block comment
			if i < len(runes)-1 && runes[i] == '*' && runes[i+1] == '/' {
				*inBlockComment = false
				i += 2 // Skip '*/'
			} else {
				i++
			}
			continue
		}

		// Check for start of comments
		if i < len(runes)-1 {
			// Block comment start
			if runes[i] == '/' && runes[i+1] == '*' {
				*inBlockComment = true
				i += 2 // Skip '/*'
				continue
			}
			// Line comment start
			if runes[i] == '/' && runes[i+1] == '/' {
				// Check if we're inside a string literal or regex literal
				beforeSlash := string(runes[:i])
				if !isInsideStringLiteral(beforeSlash) && !isInsideRegexLiteral(beforeSlash) {
					// Rest of line is a comment, stop processing
					break
				}
			}
		}

		// Check for regex literals
		if runes[i] == '/' {
			beforeSlash := string(runes[:i])
			if !isInsideStringLiteral(beforeSlash) && !isInsideRegexLiteral(beforeSlash) && canStartRegexLiteral(beforeSlash) {
				// This is likely a regex literal
				result.WriteRune(runes[i]) // Write the opening /
				i++

				// Process inside regex literal
				for i < len(runes) {
					if runes[i] == '/' {
						// Check if it's escaped
						escapeCount := 0
						j := i - 1
						for j >= 0 && runes[j] == '\\' {
							escapeCount++
							j--
						}
						if escapeCount%2 == 0 {
							// Not escaped, end of regex
							result.WriteRune(runes[i]) // Write the closing /
							i++
							// Skip regex flags (g, i, m, etc.)
							for i < len(runes) && (runes[i] >= 'a' && runes[i] <= 'z' || runes[i] >= 'A' && runes[i] <= 'Z') {
								result.WriteRune(runes[i])
								i++
							}
							break
						}
					}
					result.WriteRune(runes[i])
					i++
				}
				continue
			}
		}

		// Check for string literals
		if runes[i] == '"' || runes[i] == '\'' || runes[i] == '`' {
			quote := runes[i]
			result.WriteRune(runes[i])
			i++

			// Process inside string literal
			for i < len(runes) {
				result.WriteRune(runes[i])
				if runes[i] == quote {
					// Check if it's escaped
					escapeCount := 0
					j := i - 1
					for j >= 0 && runes[j] == '\\' {
						escapeCount++
						j--
					}
					if escapeCount%2 == 0 {
						// Not escaped, end of string
						i++
						break
					}
				}
				i++
			}
			continue
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String()
}

// isInsideStringLiteral checks if we're currently inside a string literal
// by counting unescaped quotes before the current position
func isInsideStringLiteral(text string) bool {
	runes := []rune(text)
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false

	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\'':
			if !inDoubleQuote && !inBacktick {
				// Check if escaped
				escapeCount := 0
				j := i - 1
				for j >= 0 && runes[j] == '\\' {
					escapeCount++
					j--
				}
				if escapeCount%2 == 0 {
					inSingleQuote = !inSingleQuote
				}
			}
		case '"':
			if !inSingleQuote && !inBacktick {
				// Check if escaped
				escapeCount := 0
				j := i - 1
				for j >= 0 && runes[j] == '\\' {
					escapeCount++
					j--
				}
				if escapeCount%2 == 0 {
					inDoubleQuote = !inDoubleQuote
				}
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		}
	}

	return inSingleQuote || inDoubleQuote || inBacktick
}

// isInsideRegexLiteral checks if we're currently inside a regex literal
// by tracking unescaped forward slashes
func isInsideRegexLiteral(text string) bool {
	runes := []rune(text)
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inRegex := false

	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\'':
			if !inDoubleQuote && !inBacktick && !inRegex {
				// Check if escaped
				escapeCount := 0
				j := i - 1
				for j >= 0 && runes[j] == '\\' {
					escapeCount++
					j--
				}
				if escapeCount%2 == 0 {
					inSingleQuote = !inSingleQuote
				}
			}
		case '"':
			if !inSingleQuote && !inBacktick && !inRegex {
				// Check if escaped
				escapeCount := 0
				j := i - 1
				for j >= 0 && runes[j] == '\\' {
					escapeCount++
					j--
				}
				if escapeCount%2 == 0 {
					inDoubleQuote = !inDoubleQuote
				}
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote && !inRegex {
				inBacktick = !inBacktick
			}
		case '/':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				// Check if escaped
				escapeCount := 0
				j := i - 1
				for j >= 0 && runes[j] == '\\' {
					escapeCount++
					j--
				}
				if escapeCount%2 == 0 {
					if inRegex {
						// End of regex
						inRegex = false
					} else if canStartRegexLiteralAt(text, i) {
						// Start of regex
						inRegex = true
					}
				}
			}
		}
	}

	return inRegex
}

// canStartRegexLiteral checks if a regex literal can start at the current position
// based on what comes before
func canStartRegexLiteral(beforeText string) bool {
	return canStartRegexLiteralAt(beforeText, len([]rune(beforeText)))
}

// canStartRegexLiteralAt checks if a regex literal can start at the given position
func canStartRegexLiteralAt(text string, pos int) bool {
	if pos == 0 {
		return true // Beginning of line
	}

	runes := []rune(text)
	if pos > len(runes) {
		return false
	}

	// Skip backward over whitespace
	i := pos - 1
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}

	if i < 0 {
		return true // Only whitespace before
	}

	lastChar := runes[i]

	// Regex can start after these characters/operators
	switch lastChar {
	case '=', '(', '[', ',', ':', ';', '!', '&', '|', '?', '+', '-', '*', '/', '%', '{', '}', '~', '^':
		return true
	case ')':
		// Check if it's after keywords like "return", "throw"
		word := extractWordBefore(runes, i)
		return word == "return" || word == "throw" || word == "typeof" || word == "new" || word == "in" || word == "of"
	default:
		// Check if it's after certain keywords
		word := extractWordBefore(runes, i+1)
		return word == "return" || word == "throw" || word == "typeof" || word == "new" || word == "in" || word == "of" ||
			word == "if" || word == "while" || word == "for" || word == "case"
	}
}

// extractWordBefore extracts the word that ends at the given position
func extractWordBefore(runes []rune, endPos int) string {
	if endPos < 0 || endPos >= len(runes) {
		return ""
	}

	// Find the start of the word
	start := endPos
	for start >= 0 && (isLetter(runes[start]) || isDigit(runes[start]) || runes[start] == '_' || runes[start] == '$') {
		start--
	}
	start++ // Move to the first character of the word

	if start > endPos {
		return ""
	}

	return string(runes[start : endPos+1])
}

// isLetter checks if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit checks if a rune is a digit
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// FormatJavaScriptForYAML formats a JavaScript script with proper indentation for embedding in YAML
func FormatJavaScriptForYAML(script string) []string {
	if jsLog.Enabled() {
		jsLog.Printf("Formatting JavaScript for YAML: %d bytes", len(script))
	}
	var formattedLines []string

	// Remove JavaScript comments first
	cleanScript := removeJavaScriptComments(script)

	scriptLines := strings.Split(cleanScript, "\n")
	for _, line := range scriptLines {
		// Skip empty lines when inlining to YAML
		if strings.TrimSpace(line) != "" {
			formattedLines = append(formattedLines, fmt.Sprintf("            %s\n", line))
		}
	}
	if jsLog.Enabled() {
		jsLog.Printf("Formatted %d lines for YAML", len(formattedLines))
	}
	return formattedLines
}

// WriteJavaScriptToYAML writes a JavaScript script with proper indentation to a strings.Builder
func WriteJavaScriptToYAML(yaml *strings.Builder, script string) {
	// Remove JavaScript comments first
	cleanScript := removeJavaScriptComments(script)

	scriptLines := strings.Split(cleanScript, "\n")
	for _, line := range scriptLines {
		// Skip empty lines when inlining to YAML
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(yaml, "            %s\n", line)
		}
	}
}

// WriteJavaScriptToYAMLPreservingComments writes a JavaScript script with proper indentation to a strings.Builder
// while preserving JSDoc and inline comments, but removing TypeScript-specific comments.
// Used for security-sensitive scripts like redact_secrets.
func WriteJavaScriptToYAMLPreservingComments(yaml *strings.Builder, script string) {
	scriptLines := strings.Split(script, "\n")
	previousLineWasEmpty := false
	hasWrittenContent := false // Track if we've written any content yet

	for i, line := range scriptLines {
		trimmed := strings.TrimSpace(line)

		// Skip TypeScript-specific comments
		if strings.HasPrefix(trimmed, "// @ts-") || strings.HasPrefix(trimmed, "/// <reference") {
			continue
		}

		// Handle empty lines
		if trimmed == "" {
			// Don't add blank lines at the beginning of the script
			if !hasWrittenContent {
				continue
			}

			// Look ahead to see if the next non-empty line is a JSDoc comment or function
			shouldKeepBlankLine := false
			for j := i + 1; j < len(scriptLines); j++ {
				nextTrimmed := strings.TrimSpace(scriptLines[j])
				if nextTrimmed == "" {
					continue
				}
				// Keep blank line if followed by JSDoc or function/const/async
				if strings.HasPrefix(nextTrimmed, "/**") ||
					strings.HasPrefix(nextTrimmed, "function ") ||
					strings.HasPrefix(nextTrimmed, "async function") ||
					strings.HasPrefix(nextTrimmed, "await main(") {
					shouldKeepBlankLine = true
				}
				break
			}

			if shouldKeepBlankLine && !previousLineWasEmpty {
				fmt.Fprintf(yaml, "\n")
				previousLineWasEmpty = true
			}
			continue
		}

		fmt.Fprintf(yaml, "            %s\n", line)
		previousLineWasEmpty = false
		hasWrittenContent = true
	}
}

// GetLogParserScript returns the JavaScript content for a log parser by name
func GetLogParserScript(name string) string {
	jsLog.Printf("Getting log parser script: %s", name)
	switch name {
	case "parse_claude_log":
		return getParseClaudeLogScript()
	case "parse_codex_log":
		return getParseCodexLogScript()
	case "parse_copilot_log":
		return getParseCopilotLogScript()
	case "parse_firewall_logs":
		return getParseFirewallLogsScript()
	case "validate_errors":
		return validateErrorsScript
	default:
		jsLog.Printf("Unknown log parser script requested: %s", name)
		return ""
	}
}

// GetLogParserBootstrap returns the JavaScript content for the log parser bootstrap helper
func GetLogParserBootstrap() string {
	return logParserBootstrapScript
}

// GetSafeOutputsMCPServerScript returns the JavaScript content for the GitHub Agentic Workflows Safe Outputs MCP server
func GetSafeOutputsMCPServerScript() string {
	return getSafeOutputsMCPServerScript()
}

// GetSafeOutputsToolsJSON returns the JSON content for the safe outputs tools definitions
func GetSafeOutputsToolsJSON() string {
	return safeOutputsToolsJSON
}

// GetReadBufferScript returns the embedded read_buffer.cjs script
func GetReadBufferScript() string {
	return readBufferScript
}

// GetMCPServerCoreScript returns the embedded mcp_server_core.cjs script
func GetMCPServerCoreScript() string {
	return mcpServerCoreScript
}

// GetMCPHTTPTransportScript returns the embedded mcp_http_transport.cjs script
func GetMCPHTTPTransportScript() string {
	return mcpHTTPTransportScriptSource
}

// GetMCPLoggerScript returns the embedded mcp_logger.cjs script
func GetMCPLoggerScript() string {
	return mcpLoggerScriptSource
}

// GetSafeInputsMCPServerScript returns the embedded safe_inputs_mcp_server.cjs script
func GetSafeInputsMCPServerScript() string {
	return safeInputsMCPServerScript
}

// GetSafeInputsMCPServerHTTPScript returns the embedded safe_inputs_mcp_server_http.cjs script
func GetSafeInputsMCPServerHTTPScript() string {
	return safeInputsMCPServerHTTPScript
}

// GetSafeInputsConfigLoaderScript returns the embedded safe_inputs_config_loader.cjs script
func GetSafeInputsConfigLoaderScript() string {
	return safeInputsConfigLoaderScript
}

// GetSafeInputsToolFactoryScript returns the embedded safe_inputs_tool_factory.cjs script
func GetSafeInputsToolFactoryScript() string {
	return safeInputsToolFactoryScript
}

// GetSafeInputsBootstrapScript returns the embedded safe_inputs_bootstrap.cjs script
func GetSafeInputsBootstrapScript() string {
	return safeInputsBootstrapScript
}

// GetSafeInputsValidationScript returns the embedded safe_inputs_validation.cjs script
func GetSafeInputsValidationScript() string {
	return safeInputsValidationScript
}

// GetMCPHandlerShellScript returns the embedded mcp_handler_shell.cjs script
func GetMCPHandlerShellScript() string {
	return mcpHandlerShellScript
}

// GetMCPHandlerPythonScript returns the embedded mcp_handler_python.cjs script
func GetMCPHandlerPythonScript() string {
	return mcpHandlerPythonScript
}

// GetSafeOutputsConfigScript returns the embedded safe_outputs_config.cjs script
func GetSafeOutputsConfigScript() string {
	return safeOutputsConfigScript
}

// GetSafeOutputsAppendScript returns the embedded safe_outputs_append.cjs script
func GetSafeOutputsAppendScript() string {
	return safeOutputsAppendScript
}

// GetSafeOutputsHandlersScript returns the embedded safe_outputs_handlers.cjs script
func GetSafeOutputsHandlersScript() string {
	return safeOutputsHandlersScript
}

// GetSafeOutputsToolsLoaderScript returns the embedded safe_outputs_tools_loader.cjs script
func GetSafeOutputsToolsLoaderScript() string {
	return safeOutputsToolsLoaderScript
}

// GetSafeOutputsBootstrapScript returns the embedded safe_outputs_bootstrap.cjs script
func GetSafeOutputsBootstrapScript() string {
	return safeOutputsBootstrapScript
}
