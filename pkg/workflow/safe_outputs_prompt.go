package workflow

import (
	"strings"
)

// generateSafeOutputsPromptStep generates a separate step for safe outputs instructions
// This tells agents to use the safeoutputs MCP server instead of gh CLI
func (c *Compiler) generateSafeOutputsPromptStep(yaml *strings.Builder, hasSafeOutputs bool) {
	generateStaticPromptStep(yaml,
		"Append safe outputs instructions to prompt",
		safeOutputsPromptText,
		hasSafeOutputs)
}
