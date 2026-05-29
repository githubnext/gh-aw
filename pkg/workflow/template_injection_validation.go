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
	// allowedRunScriptExpressionRegex matches trusted compiler-owned expressions that are
	// intentionally rendered in generated run scripts and are not user-controlled.
	allowedRunScriptExpressionRegex = regexp.MustCompile(`^\$\{\{\s*(env\.[^}]+|vars\.[^}]+|runner\.[^}]+|github\.(repository|run_id|workspace)|steps\.parse-guard-vars\.outputs\.(approval_labels|blocked_users|trusted_users)|job\.services\[[^]]+\]\.ports\[[^]]+\])\s*\}\}$`)
)

// hasAnyExpressionInRunContent performs a fast line-by-line text scan to determine
// whether any GitHub Actions expression (${{ ... }}) appears inside a YAML run: block.
// Used by the compiler regression guardrail to detect expressions that should have
// been rewritten to env variables.
func hasAnyExpressionInRunContent(yamlContent string) bool {
	return hasExpressionInRunContent(yamlContent, InlineExpressionPattern)
}

func hasExpressionInRunContent(yamlContent string, expressionRegex *regexp.Regexp) bool {
	// Fast-path: no matching expressions anywhere → definitely no violation.
	if !expressionRegex.MatchString(yamlContent) {
		return false
	}

	// Matching expressions exist somewhere; scan for any that appear inside a run: block
	// without doing a full YAML parse.
	// Use SplitSeq to iterate over lines lazily, avoiding the up-front allocation of the
	// full []string slice that strings.Split would create for large YAML content.
	inRunBlock := false
	runBlockIndent := 0

	for line := range strings.SplitSeq(yamlContent, "\n") {
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
				// Inside run block content — check for matching expressions.
				if expressionRegex.MatchString(line) {
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
			if expressionRegex.MatchString(rest) {
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
		if !InlineExpressionPattern.MatchString(runContent) {
			continue
		}

		// Remove non-executable regions from the run block to avoid false positives:
		//   - heredocs are written to files/stdin, not executed directly
		//   - bash # comments are ignored by the shell
		contentWithoutHeredocs := stripShellLineComments(removeHeredocContent(runContent))

		// Extract all inline expressions from this run block (excluding heredocs)
		expressions := InlineExpressionPattern.FindAllString(contentWithoutHeredocs, -1)

		// Check each expression for unsafe contexts
		for _, expr := range expressions {
			if UnsafeContextPattern.MatchString(expr) {
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

// validateNoGitHubExpressionsInRunScriptsFromParsed checks a pre-parsed workflow map
// for any GitHub Actions expression usage in run: scripts.
//
// This is a compiler regression guardrail: run: scripts in compiled lock files should
// never contain ${{ ... }} directly because the compiler must rewrite expressions into
// env: variables. It runs after validateNoTemplateInjectionFromParsed as a broader
// catch-all for any remaining expression contexts.
func validateNoGitHubExpressionsInRunScriptsFromParsed(workflow map[string]any) error {
	runBlocks := extractRunBlocks(workflow)
	templateInjectionValidationLog.Printf("Found %d run blocks to scan for raw expressions", len(runBlocks))

	var violations []TemplateInjectionViolation

	for _, runContent := range runBlocks {
		// Align with template-injection validation by excluding non-executable regions:
		// heredoc bodies and bash # comments.
		contentWithoutHeredocs := stripShellLineComments(removeHeredocContent(runContent))
		expressions := InlineExpressionPattern.FindAllString(contentWithoutHeredocs, -1)
		for _, expr := range expressions {
			if allowedRunScriptExpressionRegex.MatchString(expr) {
				continue
			}
			snippet := extractRunSnippet(contentWithoutHeredocs, expr)
			violations = append(violations, TemplateInjectionViolation{
				Expression: expr,
				Snippet:    snippet,
				Context:    detectExpressionContext(expr),
			})
		}
	}

	if len(violations) > 0 {
		templateInjectionValidationLog.Printf("Run-script expression guardrail failed: %d violation(s) found", len(violations))
		return formatRunScriptExpressionGuardrailError(violations)
	}

	return nil
}
