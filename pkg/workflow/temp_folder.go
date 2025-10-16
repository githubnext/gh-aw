package workflow

import (
	"strings"
)

// generateTempFolderPromptStep generates a separate step for temporary folder usage instructions
func (c *Compiler) generateTempFolderPromptStep(yaml *strings.Builder) {
	appendPromptStep(yaml,
		"Append temporary folder instructions to prompt",
		func(y *strings.Builder, indent string) {
			WritePromptTextToYAML(y, tempFolderPromptText, indent)
		},
		"", // no condition
		"          ")
}
