package workflow

import (
	"strings"
)

// generateTemplateRenderingStep generates a step that processes conditional template blocks
func (c *Compiler) generateTemplateRenderingStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if the markdown content contains any template patterns
	hasTemplatePattern := strings.Contains(data.MarkdownContent, "{{#if ")
	if !hasTemplatePattern {
		return
	}

	yaml.WriteString("      - name: Render template conditionals\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	WriteJavaScriptToYAML(yaml, renderTemplateScript)
}
