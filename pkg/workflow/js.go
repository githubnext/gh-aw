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

//go:embed js/push_to_pull_request_branch.cjs
var pushToBranchScript string

//go:embed js/upload_assets.cjs
var uploadAssetsScript string

//go:embed js/add_reaction_and_edit_comment.cjs
var addReactionAndEditCommentScript string

//go:embed js/check_membership.cjs
var checkMembershipScript string

//go:embed js/check_stop_time.cjs
var checkStopTimeScript string

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

//go:embed js/normalize_branch.cjs
var normalizeBranchScript string

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

// GetLogParserScript returns the JavaScript content for a log parser by name
func GetLogParserScript(name string) string {
	switch name {
	case "parse_claude_log":
		return parseClaudeLogScript
	case "parse_codex_log":
		return parseCodexLogScript
	case "parse_copilot_log":
		return parseCopilotLogScript
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
