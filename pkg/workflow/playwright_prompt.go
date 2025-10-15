package workflow

import (
	"strings"
)

// generatePlaywrightPromptStep generates a separate step for playwright output directory instructions
// Only generates the step if playwright tool is enabled in the workflow
func (c *Compiler) generatePlaywrightPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if playwright tool is enabled
	if data.Tools == nil {
		return
	}

	_, hasPlaywright := data.Tools["playwright"]
	if !hasPlaywright {
		return
	}

	yaml.WriteString("      - name: Append playwright output directory instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, playwrightPromptText, "          ")
}
