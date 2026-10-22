package aw

import (
	"errors"
	"strings"

	"github.com/rhysd/actionlint"
)

// ParseActionsExpression accepts exactly one nonempty Actions expression,
// optionally enclosed in a complete ${{ }} wrapper. It returns the unwrapped
// expression and its syntax tree without performing semantic validation.
func ParseActionsExpression(expression string) (string, actionlint.ExprNode, error) {
	text := strings.TrimSpace(expression)
	if body, wrapped := strings.CutPrefix(text, "${{"); wrapped {
		var closed bool
		text, closed = strings.CutSuffix(body, "}}")
		if !closed {
			return "", nil, errors.New("expected one valid GitHub Actions expression")
		}
		text = strings.TrimSpace(text)
	}
	input := text + " }}"
	lexer := actionlint.NewExprLexer(input)
	node, err := actionlint.NewExprParser().Parse(lexer)
	// The parser stops at the first closing delimiter, so parsing successfully
	// does not by itself exclude interpolated text or additional expressions.
	if err != nil || lexer.Offset() != len(input) {
		return "", nil, errors.New("expected one valid GitHub Actions expression")
	}
	return text, node, nil
}
