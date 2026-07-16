// This file re-exports GitHub Actions expression parsing functions from pkg/ghexpr
// so that existing code in pkg/workflow continues to compile without modification.
// New code should import pkg/ghexpr directly.
package workflow

import "github.com/github/gh-aw/pkg/ghexpr"

// ParseExpression re-exports [ghexpr.ParseExpression].
var ParseExpression = ghexpr.ParseExpression

// VisitExpressionTree re-exports [ghexpr.VisitExpressionTree].
var VisitExpressionTree = ghexpr.VisitExpressionTree

// BreakLongExpression re-exports [ghexpr.BreakLongExpression].
var BreakLongExpression = ghexpr.BreakLongExpression

// BreakAtParentheses re-exports [ghexpr.BreakAtParentheses].
var BreakAtParentheses = ghexpr.BreakAtParentheses

// stripExpressionWrapper removes the ${{ }} wrapper from an expression if present.
func stripExpressionWrapper(expression string) string {
	return ghexpr.StripExpressionWrapper(expression)
}

// hasNewlineInStringLiteral reports whether s contains a newline inside a
// single-quoted GitHub Actions expression string literal.
func hasNewlineInStringLiteral(s string) bool {
	return ghexpr.HasNewlineInStringLiteral(s)
}

// escapeForYAMLDoubleQuoted escapes s so it can be placed inside a YAML double-quoted scalar.
func escapeForYAMLDoubleQuoted(s string) string {
	return ghexpr.EscapeForYAMLDoubleQuoted(s)
}
