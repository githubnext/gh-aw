package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed sh/checkout_pr.sh
var checkoutPRScript string

//go:embed sh/pr_context_prompt.md
var prContextPromptText string

// WriteShellScriptToYAML writes a shell script with proper indentation to a strings.Builder
func WriteShellScriptToYAML(yaml *strings.Builder, script string, indent string) {
	scriptLines := strings.Split(script, "\n")
	for _, line := range scriptLines {
		// Skip empty lines at the beginning or end
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(yaml, "%s%s\n", indent, line)
		}
	}
}

// WritePromptTextToYAML writes prompt text with proper indentation to a strings.Builder
func WritePromptTextToYAML(yaml *strings.Builder, text string, indent string) {
	textLines := strings.Split(text, "\n")
	for _, line := range textLines {
		fmt.Fprintf(yaml, "%s%s\n", indent, line)
	}
}
