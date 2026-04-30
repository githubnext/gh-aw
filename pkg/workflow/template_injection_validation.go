// This file provides template injection vulnerability detection.
//
// # Template Injection Detection
//
// This file validates that GitHub Actions expressions are not used directly in
// shell commands where they could enable template injection attacks. It detects
// unsafe patterns where user-controlled data flows into shell execution context.
//
// # Validation Functions
//
//   - validateNoTemplateInjectionFromParsed() - Validates parsed YAML for template injection risks
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
// For detailed documentation, see scratchpad/validation-architecture.md and
// scratchpad/template-injection-prevention.md

package workflow

import (
	"regexp"
	"strings"
)

var templateInjectionValidationLog = newValidationLogger("template_injection")

// Pre-compiled regex patterns for template injection detection
var (
	// inlineExpressionRegex matches GitHub Actions template expressions ${{ ... }}
	inlineExpressionRegex = regexp.MustCompile(`\$\{\{[^}]+\}\}`)

	// unsafeContextRegex matches high-risk context expressions that could contain user input
	// These patterns are particularly dangerous when used directly in shell commands
	unsafeContextRegex = regexp.MustCompile(`\$\{\{\s*(github\.event\.|steps\.[^}]+\.outputs\.|inputs\.)[^}]+\}\}`)
)

// hasUnsafeExpressionInRunContent performs a fast line-by-line text scan to determine
// whether any unsafe context expression (${{ github.event.* }},
// ${{ steps.*.outputs.* }}, or ${{ inputs.* }}) appears inside the content of a
// YAML run: block.
//
// This is used as an efficient pre-flight check in generateAndValidateYAML.
// Most compiler-generated workflows place unsafe expressions only in env: values
// (the compiler's normal output pattern), so the expensive full YAML parse for
// template-injection validation can be skipped in the common case.
//
// The scanner is intentionally lightweight rather than fully conservative: when it
// encounters `run:` with no inline content (rest == ""), it enters run-block scanning
// mode and only returns true if a subsequent indented line matches unsafeContextRegex.
func hasUnsafeExpressionInRunContent(yamlContent string) bool {
	// Fast-path: no unsafe expressions anywhere → definitely no violation.
	if !unsafeContextRegex.MatchString(yamlContent) {
		return false
	}

	// Unsafe expressions exist somewhere; scan for any that appear inside a run: block
	// without doing a full YAML parse.
	lines := strings.Split(yamlContent, "\n")
	inRunBlock := false
	runBlockIndent := 0

	for _, line := range lines {
		// Compute indentation first; skip blank and all-whitespace lines in one step.
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) == 0 {
			// Blank / all-whitespace lines are allowed inside block scalars.
			continue
		}
		indent := len(line) - len(trimmed)

		if inRunBlock {
			// A non-blank line at the same or lesser indentation ends the block.
			if indent <= runBlockIndent {
				inRunBlock = false
				// Fall through: check whether this line starts a new run: block.
			} else {
				// Inside run block content — check for unsafe expressions.
				if unsafeContextRegex.MatchString(line) {
					return true
				}
				continue
			}
		}

		// Outside a run block: look for a run: key.
		// Handle both "run: ..." (map key) and "- run: ..." (inline sequence item).
		keyPart := trimmed
		if strings.HasPrefix(keyPart, "-") {
			keyPart = strings.TrimSpace(keyPart[1:])
		}
		if !strings.HasPrefix(keyPart, "run:") {
			continue
		}
		rest := strings.TrimSpace(keyPart[4:]) // text after "run:"

		if rest == "" {
			// Empty run: value is unusual; treat conservatively as if block content follows.
			inRunBlock = true
			runBlockIndent = indent
		} else if rest[0] == '|' || rest[0] == '>' {
			// Literal or folded block scalar — content is on subsequent lines.
			inRunBlock = true
			runBlockIndent = indent
		} else {
			// Inline run value, e.g. run: echo "hello ${{ github.event.foo }}".
			if unsafeContextRegex.MatchString(rest) {
				return true
			}
		}
	}

	return false
}

// validateNoTemplateInjectionFromParsed checks a pre-parsed workflow map for template
// injection vulnerabilities. It is called when the caller already holds a parsed
// representation of the compiled YAML, avoiding a redundant parse.
func validateNoTemplateInjectionFromParsed(workflow map[string]any) error {
	// Extract all run blocks from the workflow
	runBlocks := extractRunBlocks(workflow)
	templateInjectionValidationLog.Printf("Found %d run blocks to scan", len(runBlocks))

	var violations []TemplateInjectionViolation

	for _, runContent := range runBlocks {
		// Check if this run block contains inline expressions
		if !inlineExpressionRegex.MatchString(runContent) {
			continue
		}

		// Remove heredoc content from the run block to avoid false positives
		// Heredocs (e.g., << 'EOF' ... EOF) safely contain template expressions
		// because they're written to files, not executed in shell
		contentWithoutHeredocs := removeHeredocContent(runContent)

		// Extract all inline expressions from this run block (excluding heredocs)
		expressions := inlineExpressionRegex.FindAllString(contentWithoutHeredocs, -1)

		// Check each expression for unsafe contexts
		for _, expr := range expressions {
			if unsafeContextRegex.MatchString(expr) {
				// Found an unsafe pattern - extract a snippet for context
				snippet := extractRunSnippet(contentWithoutHeredocs, expr)
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
