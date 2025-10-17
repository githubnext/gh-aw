package workflow

import (
	"strings"
)

// hasEditTool checks if the edit tool is enabled in the tools configuration
func hasEditTool(tools map[string]any) bool {
	if tools == nil {
		return false
	}
	_, exists := tools["edit"]
	return exists
}

// generateEditToolPromptStep generates a separate step for edit tool accessibility instructions
// Only generates the step if edit tool is enabled in the workflow
func (c *Compiler) generateEditToolPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStep(yaml,
		"Append edit tool accessibility instructions to prompt",
		editToolPromptText,
		hasEditTool(data.Tools))
}
