package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var templateLog = logger.New("workflow:template")

// wrapExpressionsInTemplateConditionals transforms template conditionals by wrapping
// expressions in ${{ }}. For example:
// {{#if github.event.issue.number}} becomes {{#if ${{ github.event.issue.number }} }}
// {#if github.event.issue.number} becomes {#if ${{ github.event.issue.number }} }
func wrapExpressionsInTemplateConditionals(markdown string) string {
	templateLog.Print("Wrapping expressions in template conditionals")

	// First, handle double-brace pattern: {{#if expression}}
	doubleBraceRe := regexp.MustCompile(`\{\{#if\s+([^}]+)\}\}`)
	result := doubleBraceRe.ReplaceAllStringFunc(markdown, func(match string) string {
		submatches := doubleBraceRe.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		expr := strings.TrimSpace(submatches[1])

		// Check if expression is already wrapped in ${{ ... }}
		if strings.HasPrefix(expr, "${{") {
			templateLog.Print("Double-brace expression already wrapped, skipping")
			return match
		}

		// Check if expression is an environment variable reference (starts with ${)
		if strings.HasPrefix(expr, "${") {
			templateLog.Print("Double-brace environment variable reference detected, skipping wrap")
			return match
		}

		// Check if expression is a placeholder reference (starts with __)
		if strings.HasPrefix(expr, "__") {
			templateLog.Print("Double-brace placeholder reference detected, skipping wrap")
			return match
		}

		templateLog.Printf("Wrapping double-brace expression: %s", expr)
		return "{{#if ${{ " + expr + " }} }}"
	})

	// Second, handle single-brace pattern: {#if expression}
	// This pattern should not match {{#if}} (which was already processed above)
	// Use negative lookbehind/lookahead to avoid matching double braces
	singleBraceRe := regexp.MustCompile(`(?:^|[^{])\{#if\s+([^}]+)\}(?:[^}]|$)`)
	result = singleBraceRe.ReplaceAllStringFunc(result, func(match string) string {
		// The match may include a character before { or after }, so we need to preserve those
		prefix := ""
		suffix := ""
		content := match

		// Check if there's a character before the opening brace
		if len(match) > 0 && match[0] != '{' {
			prefix = string(match[0])
			content = match[1:]
		}

		// Check if there's a character after the closing brace
		if len(content) > 0 && content[len(content)-1] != '}' {
			suffix = string(content[len(content)-1])
			content = content[:len(content)-1]
		}

		// Now extract the expression from {#if expression}
		innerRe := regexp.MustCompile(`\{#if\s+([^}]+)\}`)
		submatches := innerRe.FindStringSubmatch(content)
		if len(submatches) < 2 {
			return match
		}

		expr := strings.TrimSpace(submatches[1])

		// Check if expression is already wrapped in ${{ ... }}
		if strings.HasPrefix(expr, "${{") {
			templateLog.Print("Single-brace expression already wrapped, skipping")
			return match
		}

		// Check if expression is an environment variable reference (starts with ${)
		if strings.HasPrefix(expr, "${") {
			templateLog.Print("Single-brace environment variable reference detected, skipping wrap")
			return match
		}

		// Check if expression is a placeholder reference (starts with __)
		if strings.HasPrefix(expr, "__") {
			templateLog.Print("Single-brace placeholder reference detected, skipping wrap")
			return match
		}

		templateLog.Printf("Wrapping single-brace expression: %s", expr)
		return prefix + "{#if ${{ " + expr + " }} }" + suffix
	})

	return result
}

// generateInterpolationAndTemplateStep generates a step that interpolates GitHub expression variables
// and renders template conditionals in the prompt file.
// This combines both variable interpolation and template filtering into a single step.
//
// Parameters:
//   - yaml: The string builder to write the YAML to
//   - expressionMappings: Array of ExpressionMapping containing the mappings between placeholders and GitHub expressions
//   - data: WorkflowData containing markdown content and parsed tools
//
// The generated step:
//   - Uses actions/github-script action
//   - Sets GH_AW_PROMPT environment variable to the prompt file path
//   - Sets GH_AW_EXPR_* environment variables with the actual GitHub expressions (${{ ... }})
//   - Runs interpolate_prompt.cjs script to replace placeholders and render template conditionals
func (c *Compiler) generateInterpolationAndTemplateStep(yaml *strings.Builder, expressionMappings []*ExpressionMapping, data *WorkflowData) {
	// Check if we need interpolation
	hasExpressions := len(expressionMappings) > 0

	// Check if we need template rendering
	hasTemplatePattern := strings.Contains(data.MarkdownContent, "{{#if ")
	hasGitHubContext := hasGitHubTool(data.ParsedTools)
	hasTemplates := hasTemplatePattern || hasGitHubContext

	// Skip if neither interpolation nor template rendering is needed
	if !hasExpressions && !hasTemplates {
		templateLog.Print("No interpolation or template rendering needed, skipping step generation")
		return
	}

	templateLog.Printf("Generating interpolation and template step: expressions=%d, hasPattern=%v, hasGitHubContext=%v",
		len(expressionMappings), hasTemplatePattern, hasGitHubContext)

	yaml.WriteString("      - name: Interpolate variables and render templates\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")

	// Add environment variables for extracted expressions
	for _, mapping := range expressionMappings {
		// Write the environment variable with the original GitHub expression
		fmt.Fprintf(yaml, "          %s: ${{ %s }}\n", mapping.EnvVar, mapping.Content)
	}

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	WriteJavaScriptToYAML(yaml, getInterpolatePromptScript())
}
