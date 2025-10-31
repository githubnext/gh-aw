package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GitHubContextVar represents a GitHub context expression and its sanitized environment variable name
type GitHubContextVar struct {
	Expression string // Original expression like "${{ github.actor }}"
	EnvVarName string // Environment variable name like "GH_ACTOR"
	ShellVar   string // Shell variable reference like "${GH_ACTOR}"
}

// extractGitHubContextExpressions finds all GitHub context expressions in the markdown content
// and returns a map of unique expressions to their environment variable names.
// It skips expressions that are inside template conditional headers ({{#if ... }})
// since those are processed by a separate JavaScript rendering step.
func extractGitHubContextExpressions(content string) map[string]*GitHubContextVar {
	// First, identify all expressions that are inside template conditional headers
	templateConditionalRegex := regexp.MustCompile(`\{\{#if\s+([^}]+)\}\}`)
	templateConditionalMatches := templateConditionalRegex.FindAllStringSubmatch(content, -1)

	// Build a set of expressions that are in template conditionals and should be skipped
	skipExpressions := make(map[string]bool)
	for _, match := range templateConditionalMatches {
		if len(match) >= 2 {
			// The conditional content may contain multiple GitHub expressions
			// Extract all expressions from the conditional
			conditionalContent := match[1]
			exprRegex := regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)
			exprMatches := exprRegex.FindAllString(conditionalContent, -1)
			for _, expr := range exprMatches {
				skipExpressions[expr] = true
			}
		}
	}

	// Match GitHub Actions expressions: ${{ ... }}
	expressionRegex := regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)
	matches := expressionRegex.FindAllStringSubmatch(content, -1)

	contextVars := make(map[string]*GitHubContextVar)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		fullExpr := match[0]           // e.g., "${{ github.actor }}"
		innerExpr := strings.TrimSpace(match[1]) // e.g., "github.actor"

		// Skip expressions that are in template conditional headers
		if skipExpressions[fullExpr] {
			continue
		}

		// Only process expressions that start with "github."
		// Skip other expressions like "needs.", "steps.", "env.", etc.
		if !strings.HasPrefix(innerExpr, "github.") {
			continue
		}

		// Skip if we've already seen this expression
		if _, exists := contextVars[fullExpr]; exists {
			continue
		}

		// Create environment variable name from the expression
		envVarName := githubExpressionToEnvVar(innerExpr)
		shellVar := fmt.Sprintf("${%s}", envVarName)

		contextVars[fullExpr] = &GitHubContextVar{
			Expression: fullExpr,
			EnvVarName: envVarName,
			ShellVar:   shellVar,
		}
	}

	return contextVars
}

// githubExpressionToEnvVar converts a GitHub context expression to an environment variable name
// Examples:
//   - "github.actor" -> "GH_ACTOR"
//   - "github.event.workflow_run.id" -> "GH_EVENT_WORKFLOW_RUN_ID"
//   - "github.event.workflow_run.html_url" -> "GH_EVENT_WORKFLOW_RUN_HTML_URL"
func githubExpressionToEnvVar(expr string) string {
	// Remove "github." prefix
	expr = strings.TrimPrefix(expr, "github.")

	// Replace dots with underscores
	expr = strings.ReplaceAll(expr, ".", "_")

	// Convert to uppercase
	expr = strings.ToUpper(expr)

	// Add GH_ prefix
	return "GH_" + expr
}

// replaceGitHubContextWithEnvVars replaces GitHub context expressions in the content
// with shell variable references, but skips expressions that are inside template
// conditional headers ({{#if ... }}) since those are processed by JavaScript.
func replaceGitHubContextWithEnvVars(content string, contextVars map[string]*GitHubContextVar) string {
	// First, identify template conditional regions and protect them
	templateConditionalRegex := regexp.MustCompile(`\{\{#if\s+([^}]+)\}\}`)

	// Build a list of ranges to skip (template conditional headers)
	type skipRange struct {
		start int
		end   int
	}
	var skipRanges []skipRange

	matches := templateConditionalRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		// match[0] is start of full match, match[1] is end of full match
		skipRanges = append(skipRanges, skipRange{start: match[0], end: match[1]})
	}

	// Helper function to check if a position is in a skip range
	inSkipRange := func(pos int) bool {
		for _, r := range skipRanges {
			if pos >= r.start && pos < r.end {
				return true
			}
		}
		return false
	}

	// Sort expressions by length (longest first) to avoid partial replacements
	expressions := make([]string, 0, len(contextVars))
	for expr := range contextVars {
		expressions = append(expressions, expr)
	}
	sort.Slice(expressions, func(i, j int) bool {
		return len(expressions[i]) > len(expressions[j])
	})

	// Replace each expression with its shell variable, but only if not in a skip range
	result := content
	for _, expr := range expressions {
		contextVar := contextVars[expr]

		// Find all occurrences of this expression
		exprRegex := regexp.MustCompile(regexp.QuoteMeta(expr))
		indices := exprRegex.FindAllStringIndex(result, -1)

		// Replace in reverse order to maintain correct indices
		for i := len(indices) - 1; i >= 0; i-- {
			matchStart := indices[i][0]
			matchEnd := indices[i][1]

			// Skip if this occurrence is in a template conditional header
			if inSkipRange(matchStart) {
				continue
			}

			// Replace this occurrence
			result = result[:matchStart] + contextVar.ShellVar + result[matchEnd:]
		}
	}

	return result
}

// generateEnvVarDefinitions generates YAML environment variable definitions
// for the GitHub context expressions
func generateEnvVarDefinitions(contextVars map[string]*GitHubContextVar) []string {
	if len(contextVars) == 0 {
		return nil
	}

	// Sort environment variables by name for consistent output
	envVars := make([]*GitHubContextVar, 0, len(contextVars))
	for _, v := range contextVars {
		envVars = append(envVars, v)
	}
	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i].EnvVarName < envVars[j].EnvVarName
	})

	// Generate YAML lines
	lines := make([]string, len(envVars))
	for i, v := range envVars {
		lines[i] = fmt.Sprintf("          %s: %s", v.EnvVarName, v.Expression)
	}

	return lines
}
