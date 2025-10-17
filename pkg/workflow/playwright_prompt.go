package workflow

import (
	"strings"
)

// generatePlaywrightPromptStep generates a separate step for playwright output directory instructions
// Only generates the step if playwright tool is enabled in the workflow
func (c *Compiler) generatePlaywrightPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if playwright tool is enabled
	var hasPlaywright bool
	if data.Tools != nil {
		_, hasPlaywright = data.Tools["playwright"]
	}

	generateStaticPromptStep(yaml,
		"Append playwright output directory instructions to prompt",
		playwrightPromptText,
		hasPlaywright)
}
