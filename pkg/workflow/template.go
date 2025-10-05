package workflow

import (
	"regexp"
	"strings"
)

// wrapExpressionsInTemplateConditionals transforms template conditionals by wrapping
// expressions in ${{ }}. For example:
// {{#if github.event.issue.number}} becomes {{#if ${{ github.event.issue.number }} }}
func wrapExpressionsInTemplateConditionals(markdown string) string {
	// Pattern to match {{#if expression}} where expression is not already wrapped in ${{ }}
	// This regex captures the entire {{#if ...}} block
	re := regexp.MustCompile(`\{\{#if\s+([^}]+)\}\}`)

	result := re.ReplaceAllStringFunc(markdown, func(match string) string {
		// Extract the expression part (everything between "{{#if " and "}}")
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		expr := strings.TrimSpace(submatches[1])

		// Check if expression is already wrapped in ${{ ... }}
		// Look for the pattern starting with "${{"
		if strings.HasPrefix(expr, "${{") {
			return match // Already wrapped, return as-is
		}

		// Always wrap expressions that don't start with ${{
		return "{{#if ${{ " + expr + " }} }}"
	})

	return result
}

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
