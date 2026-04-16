package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// extractRunBlocks walks the YAML tree and extracts all run: field values
func extractRunBlocks(data any) []string {
	var runBlocks []string

	switch v := data.(type) {
	case map[string]any:
		// Check if this map has a "run" key
		if runValue, ok := v["run"]; ok {
			if runStr, ok := runValue.(string); ok {
				runBlocks = append(runBlocks, runStr)
			}
		}
		// Recursively process all values in the map
		for _, value := range v {
			runBlocks = append(runBlocks, extractRunBlocks(value)...)
		}
	case []any:
		// Recursively process all items in the slice
		for _, item := range v {
			runBlocks = append(runBlocks, extractRunBlocks(item)...)
		}
	}

	return runBlocks
}

// heredocPattern holds pre-compiled regexp patterns for a single heredoc delimiter suffix.
type heredocPattern struct {
	quoted   *regexp.Regexp
	unquoted *regexp.Regexp
}

// heredocPatterns are compiled once at program start for performance.
// Each entry covers one of the common delimiter suffixes used by heredocs in shell scripts.
// Since Go regex doesn't support backreferences, we match common heredoc delimiter suffixes explicitly.
// Matches both exact delimiters (EOF) and prefixed delimiters (GH_AW_SAFE_OUTPUTS_CONFIG_EOF).
var heredocPatterns = func() []heredocPattern {
	suffixes := []string{"EOF", "EOL", "END", "HEREDOC", "JSON", "YAML", "SQL"}
	patterns := make([]heredocPattern, len(suffixes))
	for i, suffix := range suffixes {
		// Pattern for quoted delimiter ending with suffix: << 'PREFIX_SUFFIX' or << "PREFIX_SUFFIX"
		// \w* matches zero or more word characters (allowing both exact match and prefixes)
		// (?ms) enables multiline and dotall modes, .*? is non-greedy
		// \s*\w*%s\s*$ allows for leading/trailing whitespace on the closing delimiter
		patterns[i] = heredocPattern{
			quoted:   regexp.MustCompile(fmt.Sprintf(`(?ms)<<\s*['"]\w*%s['"].*?\n\s*\w*%s\s*$`, suffix, suffix)),
			unquoted: regexp.MustCompile(fmt.Sprintf(`(?ms)<<\s*\w*%s.*?\n\s*\w*%s\s*$`, suffix, suffix)),
		}
	}
	return patterns
}()

// removeHeredocContent removes heredoc sections from shell commands.
// Heredocs (e.g., cat > file << 'EOF' ... EOF) are safe for template expressions
// because the content is written to files, not executed in the shell.
func removeHeredocContent(content string) string {
	result := content
	for _, p := range heredocPatterns {
		result = p.quoted.ReplaceAllString(result, "# heredoc removed")
		result = p.unquoted.ReplaceAllString(result, "# heredoc removed")
	}
	return result
}

// TemplateInjectionViolation represents a detected template injection risk
type TemplateInjectionViolation struct {
	Expression string // The unsafe expression (e.g., "${{ github.event.issue.title }}")
	Snippet    string // Code snippet showing the violation context
	Context    string // Expression context (e.g., "github.event", "steps.*.outputs")
}

// extractRunSnippet extracts a relevant snippet from the run block containing the expression
func extractRunSnippet(runContent string, expression string) string {
	lines := strings.SplitSeq(runContent, "\n")

	for line := range lines {
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
	templateInjectionValidationLog.Printf("Detecting expression context for: %s", expression)
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
		fmt.Fprintf(&builder, "  %s context (%d occurrence(s)):\n", context, len(contextViolations))

		// Show up to 3 examples per context to keep error message manageable
		maxExamples := 3
		for i, v := range contextViolations {
			if i >= maxExamples {
				fmt.Fprintf(&builder, "    ... and %d more\n", len(contextViolations)-maxExamples)
				break
			}
			fmt.Fprintf(&builder, "    - %s\n", v.Expression)
			fmt.Fprintf(&builder, "      in: %s\n", v.Snippet)
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
	builder.WriteString("  - scratchpad/template-injection-prevention.md\n")

	return fmt.Errorf("%s", builder.String())
}
