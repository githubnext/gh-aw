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

	appendPromptStep(yaml,
		"Append XPIA security instructions to prompt",
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, xpiaPromptText, indent)
		},
		"", // no condition
		"          ")
}
