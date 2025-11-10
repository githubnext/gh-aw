package workflow

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var jsLog = logger.New("workflow:js")

//go:embed js/create_pull_request.cjs
var createPullRequestScript string

//go:embed js/create_agent_task.cjs
var createAgentTaskScript string

//go:embed js/assign_issue.cjs
var assignIssueScript string

//go:embed js/push_to_pull_request_branch.cjs
var pushToBranchScript string

//go:embed js/add_reaction_and_edit_comment.cjs
var addReactionAndEditCommentScript string

//go:embed js/check_membership.cjs
var checkMembershipScript string

//go:embed js/check_stop_time.cjs
var checkStopTimeScript string

//go:embed js/check_command_position.cjs
var checkCommandPositionScript string

//go:embed js/check_workflow_timestamp.cjs
var checkWorkflowTimestampScript string

//go:embed js/parse_claude_log.cjs
var parseClaudeLogScript string

//go:embed js/parse_codex_log.cjs
var parseCodexLogScript string

//go:embed js/parse_copilot_log.cjs
var parseCopilotLogScript string

//go:embed js/validate_errors.cjs
var validateErrorsScript string

//go:embed js/missing_tool.cjs
var missingToolScript string

//go:embed js/safe_outputs_mcp_server.cjs
var safeOutputsMCPServerScript string

//go:embed js/render_template.cjs
var renderTemplateScript string

//go:embed js/checkout_pr_branch.cjs
var checkoutPRBranchScript string

//go:embed js/redact_secrets.cjs
var redactSecretsScript string

//go:embed js/notify_comment_error.cjs
var notifyCommentErrorScript string

//go:embed js/sanitize_content.cjs
var sanitizeContentScript string

//go:embed js/sanitize_label_content.cjs
var sanitizeLabelContentScript string

//go:embed js/sanitize_workflow_name.cjs
var sanitizeWorkflowNameScript string

//go:embed js/load_agent_output.cjs
var loadAgentOutputScript string

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

//go:embed js/create_discussion.cjs
var createDiscussionScriptSource string

//go:embed js/update_issue.cjs
var updateIssueScriptSource string

//go:embed js/create_code_scanning_alert.cjs
var createCodeScanningAlertScriptSource string

//go:embed js/create_pr_review_comment.cjs
var createPRReviewCommentScriptSource string

//go:embed js/add_comment.cjs
var addCommentScriptSource string

//go:embed js/upload_assets.cjs
var uploadAssetsScriptSource string

//go:embed js/update_project.cjs
var updateProjectScriptSource string

//go:embed js/parse_firewall_logs.cjs
var parseFirewallLogsScriptSource string

// Bundled scripts (lazily bundled on-demand and cached)
var (
	collectJSONLOutputScript     string
	collectJSONLOutputScriptOnce sync.Once

	computeTextScript     string
	computeTextScriptOnce sync.Once

	sanitizeOutputScript     string
	sanitizeOutputScriptOnce sync.Once

	createIssueScript     string
	createIssueScriptOnce sync.Once

	addLabelsScript     string
	addLabelsScriptOnce sync.Once

	createDiscussionScript     string
	createDiscussionScriptOnce sync.Once

	updateIssueScript     string
	updateIssueScriptOnce sync.Once

	createCodeScanningAlertScript     string
	createCodeScanningAlertScriptOnce sync.Once

	createPRReviewCommentScript     string
	createPRReviewCommentScriptOnce sync.Once

	addCommentScript     string
	addCommentScriptOnce sync.Once

	uploadAssetsScript     string
	uploadAssetsScriptOnce sync.Once

	updateProjectScript     string
	updateProjectScriptOnce sync.Once

	parseFirewallLogsScript     string
	parseFirewallLogsScriptOnce sync.Once
)

// getCollectJSONLOutputScript returns the bundled collect_ndjson_output script
// Bundling is performed on first access and cached for subsequent calls
func getCollectJSONLOutputScript() string {
	collectJSONLOutputScriptOnce.Do(func() {
		jsLog.Print("Bundling collect_ndjson_output script")
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(collectJSONLOutputScriptSource, sources, "")
		if err != nil {
			jsLog.Printf("Bundling failed for collect_ndjson_output, using source as-is: %v", err)
			// If bundling fails, use the source as-is
			collectJSONLOutputScript = collectJSONLOutputScriptSource
		} else {
			jsLog.Printf("Successfully bundled collect_ndjson_output script: %d bytes", len(bundled))
			collectJSONLOutputScript = bundled
		}
	})
	return collectJSONLOutputScript
}

// getComputeTextScript returns the bundled compute_text script
// Bundling is performed on first access and cached for subsequent calls
func getComputeTextScript() string {
	computeTextScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(computeTextScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			computeTextScript = computeTextScriptSource
		} else {
			computeTextScript = bundled
		}
	})
	return computeTextScript
}

// getSanitizeOutputScript returns the bundled sanitize_output script
// Bundling is performed on first access and cached for subsequent calls
func getSanitizeOutputScript() string {
	sanitizeOutputScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(sanitizeOutputScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			sanitizeOutputScript = sanitizeOutputScriptSource
		} else {
			sanitizeOutputScript = bundled
		}
	})
	return sanitizeOutputScript
}

// getCreateIssueScript returns the bundled create_issue script
// Bundling is performed on first access and cached for subsequent calls
func getCreateIssueScript() string {
	createIssueScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createIssueScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			createIssueScript = createIssueScriptSource
		} else {
			createIssueScript = bundled
		}
	})
	return createIssueScript
}

// getAddLabelsScript returns the bundled add_labels script
// Bundling is performed on first access and cached for subsequent calls
func getAddLabelsScript() string {
	addLabelsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(addLabelsScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			addLabelsScript = addLabelsScriptSource
		} else {
			addLabelsScript = bundled
		}
	})
	return addLabelsScript
}

// getParseFirewallLogsScript returns the bundled parse_firewall_logs script
// Bundling is performed on first access and cached for subsequent calls
func getParseFirewallLogsScript() string {
	parseFirewallLogsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(parseFirewallLogsScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			parseFirewallLogsScript = parseFirewallLogsScriptSource
		} else {
			parseFirewallLogsScript = bundled
		}
	})
	return parseFirewallLogsScript
}

// GetCreateProjectScript returns the bundled create_project script

// getCreateDiscussionScript returns the bundled create_discussion script
// Bundling is performed on first access and cached for subsequent calls
func getCreateDiscussionScript() string {
	createDiscussionScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createDiscussionScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			createDiscussionScript = createDiscussionScriptSource
		} else {
			createDiscussionScript = bundled
		}
	})
	return createDiscussionScript
}

// getUpdateIssueScript returns the bundled update_issue script
// Bundling is performed on first access and cached for subsequent calls
func getUpdateIssueScript() string {
	updateIssueScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(updateIssueScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			updateIssueScript = updateIssueScriptSource
		} else {
			updateIssueScript = bundled
		}
	})
	return updateIssueScript
}

// getCreateCodeScanningAlertScript returns the bundled create_code_scanning_alert script
// Bundling is performed on first access and cached for subsequent calls
func getCreateCodeScanningAlertScript() string {
	createCodeScanningAlertScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createCodeScanningAlertScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			createCodeScanningAlertScript = createCodeScanningAlertScriptSource
		} else {
			createCodeScanningAlertScript = bundled
		}
	})
	return createCodeScanningAlertScript
}

// getCreatePRReviewCommentScript returns the bundled create_pr_review_comment script
// Bundling is performed on first access and cached for subsequent calls
func getCreatePRReviewCommentScript() string {
	createPRReviewCommentScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(createPRReviewCommentScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			createPRReviewCommentScript = createPRReviewCommentScriptSource
		} else {
			createPRReviewCommentScript = bundled
		}
	})
	return createPRReviewCommentScript
}

// getAddCommentScript returns the bundled add_comment script
// Bundling is performed on first access and cached for subsequent calls
func getAddCommentScript() string {
	addCommentScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(addCommentScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			addCommentScript = addCommentScriptSource
		} else {
			addCommentScript = bundled
		}
	})
	return addCommentScript
}

// getUploadAssetsScript returns the bundled upload_assets script
// Bundling is performed on first access and cached for subsequent calls
func getUploadAssetsScript() string {
	uploadAssetsScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(uploadAssetsScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			uploadAssetsScript = uploadAssetsScriptSource
		} else {
			uploadAssetsScript = bundled
		}
	})
	return uploadAssetsScript
}

// getUpdateProjectScript returns the bundled update_project script
// Bundling is performed on first access and cached for subsequent calls
func getUpdateProjectScript() string {
	updateProjectScriptOnce.Do(func() {
		sources := GetJavaScriptSources()
		bundled, err := BundleJavaScriptFromSources(updateProjectScriptSource, sources, "")
		if err != nil {
			// If bundling fails, use the source as-is
			updateProjectScript = updateProjectScriptSource
		} else {
			updateProjectScript = bundled
		}
	})
	return updateProjectScript
}

// GetJavaScriptSources returns a map of all embedded JavaScript sources
// The keys are the relative paths from the js directory
func GetJavaScriptSources() map[string]string {
	return map[string]string{
		"sanitize_content.cjs":       sanitizeContentScript,
		"sanitize_label_content.cjs": sanitizeLabelContentScript,
		"sanitize_workflow_name.cjs": sanitizeWorkflowNameScript,
		"load_agent_output.cjs":      loadAgentOutputScript,
	}
}

// removeJavaScriptComments removes JavaScript comments (// and /* */) from code
// while preserving comments that appear within string literals
func removeJavaScriptComments(code string) string {
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
	switch name {
	case "parse_claude_log":
		return parseClaudeLogScript
	case "parse_codex_log":
		return parseCodexLogScript
	case "parse_copilot_log":
		return parseCopilotLogScript
	case "parse_firewall_logs":
		return getParseFirewallLogsScript()
	case "validate_errors":
		return validateErrorsScript
	default:
		return ""
	}
}

// GetSafeOutputsMCPServerScript returns the JavaScript content for the GitHub Agentic Workflows Safe Outputs MCP server
func GetSafeOutputsMCPServerScript() string {
	return safeOutputsMCPServerScript
}

