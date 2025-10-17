package workflow

import (
	"strings"
)

// generateXPIAPromptStep generates a separate step for XPIA security warnings
func (c *Compiler) generateXPIAPromptStep(yaml *strings.Builder, data *WorkflowData) {
	generateStaticPromptStep(yaml,
		"Append XPIA security instructions to prompt",
		xpiaPromptText,
		data.SafetyPrompt)
}
