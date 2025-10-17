package workflow

import (
	"strings"
)

// generateEditToolPromptStep generates a separate step for edit tool accessibility instructions
// Only generates the step if edit tool is enabled in the workflow
func (c *Compiler) generateEditToolPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if edit tool is enabled
	var hasEdit bool
	if data.Tools != nil {
		_, hasEdit = data.Tools["edit"]
	}

	generateStaticPromptStep(yaml,
		"Append edit tool accessibility instructions to prompt",
		editToolPromptText,
		hasEdit)
}
