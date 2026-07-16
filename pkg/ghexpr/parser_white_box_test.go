//go:build !integration

// White-box tests for unexported parser types and methods. Must use package
// ghexpr (not ghexpr_test) to access unexported symbols.
package ghexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExpressionParserCurrentWithEmptyTokens tests the current() method when
// the token slice is empty.
func TestExpressionParserCurrentWithEmptyTokens(t *testing.T) {
	p := &ExpressionParser{
		tokens: []token{},
		pos:    0,
	}

	result := p.current()
	assert.Equal(t, tokenEOF, result.kind, "current() with empty tokens should return EOF token")
	assert.Equal(t, -1, result.pos, "current() with empty tokens should return pos -1")
}

// TestExpressionParserCurrentBeyondLength tests the current() method when pos
// is past the end of the token slice.
func TestExpressionParserCurrentBeyondLength(t *testing.T) {
	p := &ExpressionParser{
		tokens: []token{
			{tokenLiteral, "test", 0},
		},
		pos: 5, // Beyond array length
	}

	result := p.current()
	assert.Equal(t, tokenEOF, result.kind, "current() with pos beyond length should return EOF token")
}
