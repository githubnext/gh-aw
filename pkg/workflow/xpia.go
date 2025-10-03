package workflow

import (
	"strings"
)

// generateXPIAPromptStep generates a separate step for XPIA security warnings
func (c *Compiler) generateXPIAPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Skip if safety-prompt is disabled
	if !data.SafetyPrompt {
		return
	}

	yaml.WriteString("      - name: Append XPIA security instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, xpiaPromptText, "          ")
}
