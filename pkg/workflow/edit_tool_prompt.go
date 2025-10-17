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

	appendPromptStep(yaml,
		"Append edit tool accessibility instructions to prompt",
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, editToolPromptText, indent)
		},
		"", // no condition
		"          ")
}
