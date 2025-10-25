package workflow

import (
	"strings"
)

// hasPlaywrightTool checks if the playwright tool is enabled in the tools configuration
func hasPlaywrightTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.Playwright != nil
}

// generatePlaywrightPromptStep generates a separate step for playwright output directory instructions
// Only generates the step if playwright tool is enabled in the workflow
func (c *Compiler) generatePlaywrightPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStep(yaml,
		"Append playwright output directory instructions to prompt",
		playwrightPromptText,
		hasPlaywrightTool(data.ParsedTools))
}
