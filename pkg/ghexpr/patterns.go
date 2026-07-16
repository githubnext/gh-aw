package ghexpr

import (
	"regexp"
	"strings"
)

// HasExpressionMarker reports whether s contains a GitHub Actions expression
// opening marker ("${{").  This is a permissive check used when partial
// expressions should be treated as dynamic values.
func HasExpressionMarker(s string) bool {
	return strings.Contains(s, "${{")
}

// ContainsExpression reports whether s contains a complete, non-empty GitHub
// Actions expression — i.e. a "${{" marker followed by at least one character
// before a closing "}}".
func ContainsExpression(s string) bool {
	_, afterOpen, found := strings.Cut(s, "${{")
	if !found {
		return false
	}
	closeIdx := strings.Index(afterOpen, "}}")
	return closeIdx > 0
}

// IsExpression reports whether the entire string s is a GitHub Actions
// expression (starts with "${{" and ends with "}}").
func IsExpression(s string) bool {
	return strings.HasPrefix(s, "${{") && strings.HasSuffix(s, "}}")
}

// Core Expression Patterns

var (
	// ExpressionPattern matches GitHub Actions expressions: ${{ ... }}
	// Uses non-greedy matching to handle multiple expressions in one string.
	ExpressionPattern = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

	// ExpressionPatternDotAll matches expressions with dotall mode enabled so
	// that "." also matches newlines.  Useful for multi-line expression bodies.
	ExpressionPatternDotAll = regexp.MustCompile(`(?s)\$\{\{(.*?)\}\}`)
)

// Template Patterns

var (
	// InlineExpressionPattern matches inline ${{ … }} expressions in template strings.
	InlineExpressionPattern = regexp.MustCompile(`\$\{\{[^}]+\}\}`)

	// TemplateIfPattern matches {{#if condition }} template conditionals.
	// Captures the condition expression (which may itself contain ${{ … }}).
	//
	// Expression group: (?:\$\{\{[^\}]*\}\}|[^\}\{]|\{[^\{])*
	//   - \$\{\{[^\}]*\}\}  — already-wrapped ${{ ... }} expression
	//   - [^\}\{]           — any character that is not } or {
	//   - \{[^\{]           — a { not immediately followed by another { (handles ${ env refs)
	TemplateIfPattern = regexp.MustCompile(`\{\{#if\s+((?:\$\{\{[^\}]*\}\}|[^\}\{]|\{[^\{])*)\s*\}\}`)

	// TemplateElseIfPattern matches elseif/else-if/else_if template conditionals in all
	// supported syntax variants:
	//   {{#elseif expr}}  {{#else-if expr}}  {{#else_if expr}}
	//   {{elseif expr}}   {{else-if expr}}   {{else_if expr}}
	TemplateElseIfPattern = regexp.MustCompile(`\{\{#?else[-_]?if\s+((?:\$\{\{[^\}]*\}\}|[^\}\{]|\{[^\{])*)\s*\}\}`)
)

// Literal and Operator Patterns

var (
	// StringLiteralPattern matches string literals in single quotes, double quotes,
	// or backticks.  Example: 'hello', "world", `template`.
	StringLiteralPattern = regexp.MustCompile(`^'[^']*'$|^"[^"]*"$|^` + "`[^`]*`$")

	// NumberLiteralPattern matches integer and decimal numeric literals (with
	// optional leading minus).  Example: 42, -3.14, 0.5.
	NumberLiteralPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

	// RangePattern matches numeric range strings such as "1-10" or "100-200".
	RangePattern = regexp.MustCompile(`^\d+-\d+$`)

	// OrPattern matches a logical OR expression, capturing both sides.
	// Example: "value1 || value2".
	OrPattern = regexp.MustCompile(`^(.+?)\s*\|\|\s*(.+)$`)

	// ComparisonExtractionPattern extracts the left-hand property access from a
	// comparison expression.
	// Example: "github.workflow == 'CI'" → captures "github.workflow".
	ComparisonExtractionPattern = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:==|!=|<|>|<=|>=)\s*`)
)
