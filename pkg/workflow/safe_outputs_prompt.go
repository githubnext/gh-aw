package workflow

import (
	"strings"
)

// generateSafeOutputsPromptStep generates a separate step for safe outputs instructions
// when safe-outputs are configured, informing the agent about available output capabilities
func (c *Compiler) generateSafeOutputsPromptStep(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	appendPromptStepWithHeredoc(yaml,
		"Append safe outputs instructions to prompt",
		func(y *strings.Builder) {
			generateSafeOutputsPromptSection(y, safeOutputs)
		})
}
