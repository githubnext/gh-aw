//go:build !integration

package ghexpr_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/ghexpr"
	"github.com/stretchr/testify/assert"
)

// TestHasExpressionMarker tests the permissive opening-marker check.
func TestHasExpressionMarker(t *testing.T) {
	assert.True(t, ghexpr.HasExpressionMarker("${{ github.actor }}"))
	assert.True(t, ghexpr.HasExpressionMarker("prefix ${{ x }}"))
	assert.False(t, ghexpr.HasExpressionMarker("no expression here"))
	assert.False(t, ghexpr.HasExpressionMarker(""))
}

// TestContainsExpression tests the complete-expression check.
func TestContainsExpression(t *testing.T) {
	assert.True(t, ghexpr.ContainsExpression("${{ github.actor }}"))
	assert.True(t, ghexpr.ContainsExpression("text ${{ a }} text"))
	assert.False(t, ghexpr.ContainsExpression("${{}}"), "empty inner content")
	assert.False(t, ghexpr.ContainsExpression("plain"))
	assert.False(t, ghexpr.ContainsExpression("${{ no closing"))
}

// TestIsExpression tests the whole-string expression check.
func TestIsExpression(t *testing.T) {
	assert.True(t, ghexpr.IsExpression("${{ github.actor }}"))
	assert.True(t, ghexpr.IsExpression("${{github.actor}}"))
	assert.False(t, ghexpr.IsExpression("prefix ${{ github.actor }}"))
	assert.False(t, ghexpr.IsExpression("${{ github.actor }} suffix"))
	assert.False(t, ghexpr.IsExpression("plain text"))
}

// TestExpressionPattern verifies the compiled regex matches correctly.
func TestExpressionPattern(t *testing.T) {
	m := ghexpr.ExpressionPattern.FindStringSubmatch("${{ github.actor }}")
	assert.Len(t, m, 2)
	assert.Equal(t, " github.actor ", m[1])

	assert.False(t, ghexpr.ExpressionPattern.MatchString("plain text"))
}

// TestStringLiteralPattern covers string-literal pattern matching.
func TestStringLiteralPattern(t *testing.T) {
	assert.True(t, ghexpr.StringLiteralPattern.MatchString("'hello'"))
	assert.True(t, ghexpr.StringLiteralPattern.MatchString(`"world"`))
	assert.False(t, ghexpr.StringLiteralPattern.MatchString("no quotes"))
}

// TestNumberLiteralPattern covers number-literal pattern matching.
func TestNumberLiteralPattern(t *testing.T) {
	assert.True(t, ghexpr.NumberLiteralPattern.MatchString("42"))
	assert.True(t, ghexpr.NumberLiteralPattern.MatchString("-3.14"))
	assert.True(t, ghexpr.NumberLiteralPattern.MatchString("0"))
	assert.False(t, ghexpr.NumberLiteralPattern.MatchString("abc"))
}

// TestOrPattern verifies OR-pattern captures both sides.
func TestOrPattern(t *testing.T) {
	m := ghexpr.OrPattern.FindStringSubmatch("left || right")
	assert.Len(t, m, 3)
	assert.Equal(t, "left", m[1])
	assert.Equal(t, "right", m[2])
}

// TestTemplateIfPattern verifies the template-if pattern.
func TestTemplateIfPattern(t *testing.T) {
	m := ghexpr.TemplateIfPattern.FindStringSubmatch("{{#if github.actor }}")
	assert.NotNil(t, m)
	assert.Contains(t, m[1], "github.actor")
}

// TestRangePattern verifies range pattern matching.
func TestRangePattern(t *testing.T) {
	assert.True(t, ghexpr.RangePattern.MatchString("1-10"))
	assert.True(t, ghexpr.RangePattern.MatchString("100-200"))
	assert.False(t, ghexpr.RangePattern.MatchString("abc"))
	assert.False(t, ghexpr.RangePattern.MatchString("1"))
}
