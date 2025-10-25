package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var templateLog = logger.New("workflow:template")

// wrapExpressionsInTemplateConditionals transforms template conditionals by wrapping
// expressions in ${{ }}. For example:
// {{#if github.event.issue.number}} becomes {{#if ${{ github.event.issue.number }} }}
func wrapExpressionsInTemplateConditionals(markdown string) string {
	// Pattern to match {{#if expression}} where expression is not already wrapped in ${{ }}
	// This regex captures the entire {{#if ...}} block
	re := regexp.MustCompile(`\{\{#if\s+([^}]+)\}\}`)

	templateLog.Print("Wrapping expressions in template conditionals")

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

// validateNoIncludesInTemplateRegions checks that import directives
// are not used inside template conditional blocks ({{#if...}}{{/if}})
func validateNoIncludesInTemplateRegions(markdown string) error {
	templateLog.Print("Validating that imports are not inside template regions")

	// Find all template regions by matching {{#if...}}...{{/if}} blocks
	// This regex matches template conditional blocks with their content
	templateRegionPattern := regexp.MustCompile(`(?s)\{\{#if\s+[^}]+\}\}(.*?)\{\{/if\}\}`)

	matches := templateRegionPattern.FindAllStringSubmatch(markdown, -1)
	templateLog.Printf("Found %d template regions to validate", len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// Check the content inside the template region (capture group 1)
		regionContent := match[1]

		// Check for import directives in this region
		lines := strings.Split(regionContent, "\n")
		for lineNum, line := range lines {
			// Trim leading/trailing whitespace before checking
			trimmedLine := strings.TrimSpace(line)
			directive := parser.ParseImportDirective(trimmedLine)
			if directive != nil {
				return fmt.Errorf("import directives cannot be used inside template regions ({{#if...}}{{/if}}): found '%s' at line %d within template block", directive.Original, lineNum+1)
			}
		}
	}

	return nil
}

// generateTemplateRenderingStep generates a step that processes conditional template blocks
func (c *Compiler) generateTemplateRenderingStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if the markdown content contains any template patterns
	hasTemplatePattern := strings.Contains(data.MarkdownContent, "{{#if ")

	// Also check if GitHub tool is enabled (which adds template patterns to the prompt)
	hasGitHubContext := hasGitHubTool(data.ParsedTools)

	if !hasTemplatePattern && !hasGitHubContext {
		return
	}

	templateLog.Printf("Generating template rendering step: hasPattern=%v, hasGitHubContext=%v", hasTemplatePattern, hasGitHubContext)

	yaml.WriteString("      - name: Render template conditionals\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	WriteJavaScriptToYAML(yaml, renderTemplateScript)
}
