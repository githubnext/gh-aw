package workflow

import (
	"strings"
)

// generateTempFolderPromptStep generates a separate step for temporary folder usage instructions
func (c *Compiler) generateTempFolderPromptStep(yaml *strings.Builder) {
	generateStaticPromptStep(yaml,
		"Append temporary folder instructions to prompt",
		tempFolderPromptText,
		true) // Always include temp folder instructions
}
