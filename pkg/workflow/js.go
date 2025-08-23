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

//go:embed js/create_comment.cjs
var createCommentScript string

//go:embed js/sanitize_output.cjs
var sanitizeOutputScript string

//go:embed js/add_labels.cjs
var addLabelsScript string

//go:embed js/setup_agent_output.cjs
var setupAgentOutputScript string

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
			yaml.WriteString(fmt.Sprintf("            %s\n", line))
		}
	}
}
