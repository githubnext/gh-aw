package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed js/create_pull_request.cjs
var createPullRequestScript string

//go:embed js/create_issue.js
var createIssueScript string

//go:embed js/create_discussion.js
var createDiscussionScript string

//go:embed js/add_comment.cjs
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

//go:embed js/upload_assets.cjs
var uploadAssetsScript string

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
// while preserving comments that appear within string literals
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
				// Check if we're inside a string literal
				if !isInsideStringLiteral(string(runes[:i])) {
					// Rest of line is a comment, stop processing
					break
				}
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
