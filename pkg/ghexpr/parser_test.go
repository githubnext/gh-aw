//go:build !integration

package ghexpr_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/ghexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseExpression covers basic parsing behavior.
func TestParseExpression(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		rendered string
		wantErr  bool
	}{
		{"single literal", "github.event_name", "github.event_name", false},
		{"AND", "a && b", "(a) && (b)", false},
		{"OR", "a || b", "a || b", false},
		{"NOT", "!a", "!(a)", false},
		{"AND higher precedence than OR", "a || b && c", "a || (b) && (c)", false},
		{"NOT with function call (parsed as opaque literal)", "!cancelled()", "!(cancelled())", false},
		{"parentheses override precedence", "(a || b) && c", "(a || b) && (c)", false},
		{"empty input", "", "", true},
		{"empty expression", "   ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ghexpr.ParseExpression(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, node)
			assert.Equal(t, tt.rendered, node.Render())
		})
	}
}

// TestStripExpressionWrapper tests the wrapper-stripping helper.
func TestStripExpressionWrapper(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"${{ github.actor }}", "github.actor"},
		{"${{github.actor}}", "github.actor"},
		{"github.actor", "github.actor"},
		{"", ""},
		{"${{ }}", ""},
		{"  ${{ foo }}  ", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ghexpr.StripExpressionWrapper(tt.input))
		})
	}
}

// TestBreakLongExpression tests line-breaking behavior.
func TestBreakLongExpression(t *testing.T) {
	short := "github.event_name == 'push'"
	result := ghexpr.BreakLongExpression(short)
	assert.Len(t, result, 1, "short expression should not be broken")
	assert.Equal(t, short, result[0])
}

// TestVisitExpressionTree tests the tree-visitor utility.
func TestVisitExpressionTree(t *testing.T) {
	node := &ghexpr.AndNode{
		Left: &ghexpr.ExpressionNode{Expression: "a"},
		Right: &ghexpr.OrNode{
			Left:  &ghexpr.ExpressionNode{Expression: "b"},
			Right: &ghexpr.ExpressionNode{Expression: "c"},
		},
	}

	var visited []string
	err := ghexpr.VisitExpressionTree(node, func(e *ghexpr.ExpressionNode) error {
		visited = append(visited, e.Expression)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, visited)
}

// TestHasNewlineInStringLiteral tests the newline-in-string-literal detector.
func TestHasNewlineInStringLiteral(t *testing.T) {
	assert.True(t, ghexpr.HasNewlineInStringLiteral("contains(github.event.comment.body, 'text\nmore')"))
	assert.False(t, ghexpr.HasNewlineInStringLiteral("github.event_name == 'push'"))
}

// TestEscapeForYAMLDoubleQuoted tests YAML escaping.
func TestEscapeForYAMLDoubleQuoted(t *testing.T) {
	assert.Equal(t, `line1\nline2`, ghexpr.EscapeForYAMLDoubleQuoted("line1\nline2"))
	assert.Equal(t, `a\\b`, ghexpr.EscapeForYAMLDoubleQuoted(`a\b`))
	assert.Equal(t, `say \"hi\"`, ghexpr.EscapeForYAMLDoubleQuoted(`say "hi"`))
}
