package workflow

import (
	"strings"
)

// generateTempFolderPromptStep generates a separate step for temporary folder usage instructions
func (c *Compiler) generateTempFolderPromptStep(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Append temporary folder instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, tempFolderPromptText, "          ")
}
