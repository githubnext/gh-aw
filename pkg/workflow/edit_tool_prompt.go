package workflow

import (
	"strings"
)

// generateEditToolPromptStep generates a separate step for edit tool accessibility instructions
// Only generates the step if edit tool is enabled in the workflow
func (c *Compiler) generateEditToolPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if edit tool is enabled
	if data.Tools == nil {
		return
	}

	_, hasEdit := data.Tools["edit"]
	if !hasEdit {
		return
	}

	yaml.WriteString("      - name: Append edit tool accessibility instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, editToolPromptText, "          ")
}
