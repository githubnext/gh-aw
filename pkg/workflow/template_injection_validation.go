// Package workflow provides template injection vulnerability detection.
//
// # Template Injection Detection
//
// This file validates that GitHub Actions expressions are not used directly in
// shell commands where they could enable template injection attacks. It detects
// unsafe patterns where user-controlled data flows into shell execution context.
//
// # Validation Functions
//
//   - validateNoTemplateInjection() - Validates compiled YAML for template injection risks
//
// # Validation Pattern: Security Detection
//
// Template injection validation uses pattern detection:
//   - Scans compiled YAML for run: steps with inline expressions
//   - Identifies unsafe patterns: ${{ ... }} directly in shell commands
//   - Suggests safe patterns: use env: variables instead
//   - Focuses on high-risk contexts: github.event.*, steps.*.outputs.*
//
// # Unsafe Patterns (Template Injection Risk)
//
// Direct expression use in run: commands:
//   - run: echo "${{ github.event.issue.title }}"
//   - run: bash script.sh ${{ steps.foo.outputs.bar }}
//   - run: command "${{ inputs.user_data }}"
//
// # Safe Patterns (No Template Injection)
//
// Expression use through environment variables:
//   - env: { VALUE: "${{ github.event.issue.title }}" }
//     run: echo "$VALUE"
//   - env: { OUTPUT: "${{ steps.foo.outputs.bar }}" }
//     run: bash script.sh "$OUTPUT"
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It detects template injection vulnerabilities
//   - It validates expression usage in shell contexts
//   - It enforces safe expression handling patterns
//   - It provides security-focused compile-time checks
//
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md and
// specs/template-injection-prevention.md
package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var templateInjectionValidationLog = logger.New("workflow:template_injection_validation")

// Pre-compiled regex patterns for template injection detection
var (
	// runBlockRegex matches YAML run: blocks and captures their content
	// This regex matches both single-line and multi-line run commands in YAML
	// Pattern explanation:
	//   ^\s+run:\s*\|\s*\n((?:[ \t]+.+\n?)+?)\s*(?:^[ \t]*-\s|\z) - matches multi-line block scalar (run: |)
	//     - Stops at next step (^[ \t]*-\s) or end of string (\z)
	//   | - OR
	//   ^\s+run:\s*(.+)$ - matches single-line run command
	// Group 1 = multi-line content, Group 2 = single-line content
	runBlockRegex = regexp.MustCompile(`(?m)^\s+run:\s*\|\s*\n((?:[ \t]+.+\n?)+?)\s*(?:^[ \t]*-\s|\z)|^\s+run:\s*(.+)$`)

	// inlineExpressionRegex matches GitHub Actions template expressions ${{ ... }}
	inlineExpressionRegex = regexp.MustCompile(`\$\{\{[^}]+\}\}`)

	// unsafeContextRegex matches high-risk context expressions that could contain user input
	// These patterns are particularly dangerous when used directly in shell commands
	unsafeContextRegex = regexp.MustCompile(`\$\{\{\s*(github\.event\.|steps\.[^}]+\.outputs\.|inputs\.)[^}]+\}\}`)
)

// validateNoTemplateInjection checks compiled YAML for template injection vulnerabilities
// It detects cases where GitHub Actions expressions are used directly in shell commands
// instead of being passed through environment variables
func validateNoTemplateInjection(yamlContent string) error {
	templateInjectionValidationLog.Print("Validating compiled YAML for template injection risks")

	// Find all run: blocks in the YAML
	runMatches := runBlockRegex.FindAllStringSubmatch(yamlContent, -1)
	templateInjectionValidationLog.Printf("Found %d run blocks to scan", len(runMatches))

	var violations []TemplateInjectionViolation

	for _, match := range runMatches {
		// Extract run content from the regex match groups
		// Group 1 = multi-line block, Group 2 = single-line command
		var runContent string
		if len(match) > 1 && match[1] != "" {
			runContent = match[1] // Multi-line run block
		} else if len(match) > 2 && match[2] != "" {
			runContent = match[2] // Single-line run command
		} else {
			continue
		}

		// Check if this run block contains inline expressions
		if !inlineExpressionRegex.MatchString(runContent) {
			continue
		}

		// Extract all inline expressions from this run block
		expressions := inlineExpressionRegex.FindAllString(runContent, -1)

		// Check each expression for unsafe contexts
		for _, expr := range expressions {
			if unsafeContextRegex.MatchString(expr) {
				// Found an unsafe pattern - extract a snippet for context
				snippet := extractRunSnippet(runContent, expr)
				violations = append(violations, TemplateInjectionViolation{
					Expression: expr,
					Snippet:    snippet,
					Context:    detectExpressionContext(expr),
				})

				templateInjectionValidationLog.Printf("Found template injection risk: %s in run block", expr)
			}
		}
	}

	// If we found violations, return a detailed error
	if len(violations) > 0 {
		templateInjectionValidationLog.Printf("Template injection validation failed: %d violations found", len(violations))
		return formatTemplateInjectionError(violations)
	}

	templateInjectionValidationLog.Print("Template injection validation passed")
	return nil
}

// TemplateInjectionViolation represents a detected template injection risk
type TemplateInjectionViolation struct {
	Expression string // The unsafe expression (e.g., "${{ github.event.issue.title }}")
	Snippet    string // Code snippet showing the violation context
	Context    string // Expression context (e.g., "github.event", "steps.*.outputs")
}

// extractRunSnippet extracts a relevant snippet from the run block containing the expression
func extractRunSnippet(runContent string, expression string) string {
	lines := strings.Split(runContent, "\n")

	for _, line := range lines {
		if strings.Contains(line, expression) {
			// Return the trimmed line containing the expression
			trimmed := strings.TrimSpace(line)
			// Limit snippet length to avoid overwhelming error messages
			if len(trimmed) > 100 {
				return trimmed[:97] + "..."
			}
			return trimmed
		}
	}

	// Fallback: return the expression itself
	return expression
}

// detectExpressionContext identifies what type of expression this is
func detectExpressionContext(expression string) string {
	if strings.Contains(expression, "github.event.") {
		return "github.event"
	}
	if strings.Contains(expression, "steps.") && strings.Contains(expression, ".outputs.") {
		return "steps.*.outputs"
	}
	if strings.Contains(expression, "inputs.") {
		return "workflow inputs"
	}
	return "unknown context"
}

// formatTemplateInjectionError formats a user-friendly error message for template injection violations
func formatTemplateInjectionError(violations []TemplateInjectionViolation) error {
	var builder strings.Builder

	builder.WriteString("template injection vulnerabilities detected in compiled workflow\n\n")
	builder.WriteString("The following expressions are used directly in shell commands, which enables template injection attacks:\n\n")

	// Group violations by context for clearer reporting
	contextGroups := make(map[string][]TemplateInjectionViolation)
	for _, v := range violations {
		contextGroups[v.Context] = append(contextGroups[v.Context], v)
	}

	// Report violations grouped by context
	for context, contextViolations := range contextGroups {
		builder.WriteString(fmt.Sprintf("  %s context (%d occurrence(s)):\n", context, len(contextViolations)))

		// Show up to 3 examples per context to keep error message manageable
		maxExamples := 3
		for i, v := range contextViolations {
			if i >= maxExamples {
				builder.WriteString(fmt.Sprintf("    ... and %d more\n", len(contextViolations)-maxExamples))
				break
			}
			builder.WriteString(fmt.Sprintf("    - %s\n", v.Expression))
			builder.WriteString(fmt.Sprintf("      in: %s\n", v.Snippet))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("Security Risk:\n")
	builder.WriteString("  When expressions are used directly in shell commands, an attacker can inject\n")
	builder.WriteString("  malicious code through user-controlled inputs (issue titles, PR descriptions,\n")
	builder.WriteString("  comments, etc.) to execute arbitrary commands, steal secrets, or modify the repository.\n\n")

	builder.WriteString("Safe Pattern - Use environment variables instead:\n")
	builder.WriteString("  env:\n")
	builder.WriteString("    MY_VALUE: ${{ github.event.issue.title }}\n")
	builder.WriteString("  run: |\n")
	builder.WriteString("    echo \"Title: $MY_VALUE\"\n\n")

	builder.WriteString("Unsafe Pattern - Do NOT use expressions directly:\n")
	builder.WriteString("  run: |\n")
	builder.WriteString("    echo \"Title: ${{ github.event.issue.title }}\"  # UNSAFE!\n\n")

	builder.WriteString("References:\n")
	builder.WriteString("  - https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions\n")
	builder.WriteString("  - https://docs.zizmor.sh/audits/#template-injection\n")
	builder.WriteString("  - specs/template-injection-prevention.md\n")

	return fmt.Errorf("%s", builder.String())
}
