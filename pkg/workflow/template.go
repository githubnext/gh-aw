package workflow

import (
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// wrapExpressionsInTemplateConditionals transforms template conditionals by wrapping
// GitHub Actions expressions in ${{ }}. For example:
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

		// Check if expression is already wrapped in ${{ }}
		if strings.HasPrefix(expr, "${{") && strings.HasSuffix(expr, "}}") {
			return match // Already wrapped, return as-is
		}

		// Check if this looks like a GitHub Actions expression that should be wrapped
		if shouldWrapExpression(expr) {
			return "{{#if ${{ " + expr + " }} }}"
		}

		// Not a GitHub expression, return as-is
		return match
	})

	return result
}

// shouldWrapExpression determines if an expression should be wrapped in ${{ }}
func shouldWrapExpression(expr string) bool {
	expr = strings.TrimSpace(expr)

	// Check if it starts with allowed prefixes
	allowedPrefixes := []string{
		"github.",
		"needs.",
		"steps.",
		"env.",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(expr, prefix) {
			return true
		}
	}

	// Check if it's an exact match for allowed expressions
	for _, allowed := range constants.AllowedExpressions {
		if expr == allowed {
			return true
		}
	}

	return false
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
