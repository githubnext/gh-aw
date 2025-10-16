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

	appendPromptStep(yaml,
		"Append playwright output directory instructions to prompt",
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, playwrightPromptText, indent)
		},
		"", // no condition
		"          ")
}
