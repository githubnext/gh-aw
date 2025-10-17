package workflow

import (
	"strings"
)

// hasPlaywrightTool checks if the playwright tool is enabled in the tools configuration
func hasPlaywrightTool(tools map[string]any) bool {
	if tools == nil {
		return false
	}
	_, exists := tools["playwright"]
	return exists
}

// generatePlaywrightPromptStep generates a separate step for playwright output directory instructions
// Only generates the step if playwright tool is enabled in the workflow
func (c *Compiler) generatePlaywrightPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStep(yaml,
		"Append playwright output directory instructions to prompt",
		playwrightPromptText,
		hasPlaywrightTool(data.Tools))
}
