// This file provides validation for GitHub Actions expressions that serialize the
// entire secrets context, such as toJSON(secrets).
//
// # Secrets Serialization Validation
//
// Expressions like ${{ toJSON(secrets) }} pass the entire secrets context to a
// function that converts it to a string. This exposes ALL secrets to the agent
// rather than only the specific secret values it requires.
//
// The validation uses the same strict/non-strict pattern as other secret checks:
//   - In strict mode an error is returned.
//   - In non-strict mode a warning is printed and compilation continues.
//
// For strict mode orchestration, see strict_mode_validation.go.
// For environment secrets validation, see strict_mode_env_validation.go.
// For steps secrets validation, see strict_mode_steps_validation.go.

package workflow

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var expressionSecretsSerializationLog = newValidationLogger("expression_secrets_serialization")

// secretsSerializationPattern matches function calls that pass the entire secrets
// context as an argument, e.g. toJSON(secrets).
//
// GitHub Actions function names are case-insensitive, so (?i) is used. The
// word-boundary \b before the function name prevents matching substrings
// (e.g. "nottoJSON"). The pattern deliberately does NOT match
// toJSON(secrets.SPECIFIC_KEY) because a dot after "secrets" would be consumed
// by \s*\) only if there is no further content — the closing \) requires that
// nothing follows "secrets" except optional whitespace.
var secretsSerializationPattern = regexp.MustCompile(`(?i)\btoJSON\s*\(\s*secrets\s*\)`)

// findSecretsSerializationExpressions scans content for GitHub Actions expressions
// that serialize the entire secrets context (e.g. ${{ toJSON(secrets) }}).
// Returns the full ${{ … }} expression strings that matched.
func findSecretsSerializationExpressions(content string) []string {
	if !strings.Contains(content, "${{") {
		return nil
	}

	expressionSecretsSerializationLog.Printf("Scanning content (%d bytes) for secrets serialization expressions", len(content))

	matches := ExpressionPatternDotAll.FindAllStringSubmatch(content, -1)
	var found []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		exprContent := match[1]
		if secretsSerializationPattern.MatchString(exprContent) {
			full := "${{ " + strings.TrimSpace(exprContent) + " }}"
			found = append(found, full)
		}
	}

	if len(found) > 0 {
		expressionSecretsSerializationLog.Printf("Found %d secrets serialization expression(s)", len(found))
	}
	return found
}

// neutralizeSecretsSerializationExpressions replaces ${{ toJSON(secrets) }} (and
// case/whitespace variants) with ${{ false }} so that the expression allowlist does
// not re-error on a pattern that was already handled by
// validateSecretsSerializationExpressions in non-strict mode.
//
// "${{ false }}" is a valid literal expression and passes the allowlist check.
func neutralizeSecretsSerializationExpressions(content string) string {
	return ExpressionPatternDotAll.ReplaceAllStringFunc(content, func(match string) string {
		groups := ExpressionPatternDotAll.FindStringSubmatch(match)
		if len(groups) >= 2 && secretsSerializationPattern.MatchString(groups[1]) {
			return "${{ false }}"
		}
		return match
	})
}

// and frontmatter YAML for expressions that serialize all secrets, such as
// ${{ toJSON(secrets) }}.
//
// Serializing the entire secrets context exposes ALL secrets to the agent, which
// is a significant security risk. Callers should use specific secret references
// (e.g. secrets.MY_SECRET) instead.
//
// In strict mode this returns an error; in non-strict mode it emits a warning to
// stderr and increments the compiler warning count.
func (c *Compiler) validateSecretsSerializationExpressions(workflowData *WorkflowData) error {
	expressionSecretsSerializationLog.Printf("Validating secrets serialization expressions (strictMode=%t)", c.strictMode)

	var allFound []string
	for _, content := range []string{workflowData.MarkdownContent, workflowData.FrontmatterYAML} {
		allFound = append(allFound, findSecretsSerializationExpressions(content)...)
	}

	if len(allFound) == 0 {
		expressionSecretsSerializationLog.Printf("No secrets serialization expressions found")
		return nil
	}

	allFound = sliceutil.Deduplicate(allFound)
	sort.Strings(allFound)

	expressionSecretsSerializationLog.Printf("Detected %d secrets serialization expression(s): %v", len(allFound), allFound)

	effectiveStrict := c.effectiveStrictMode(workflowData.RawFrontmatter)

	msg := fmt.Sprintf(
		"secrets serialization expression(s) detected that would expose all secrets to the agent. "+
			"Found: %s. "+
			"Use specific secret references (e.g. secrets.MY_SECRET) instead of passing the entire secrets context.",
		strings.Join(allFound, ", "),
	)

	if effectiveStrict {
		return fmt.Errorf("strict mode: %s", msg)
	}

	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Warning: "+msg))
	c.IncrementWarningCount()
	return nil
}
