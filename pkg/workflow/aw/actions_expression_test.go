package aw

import (
	"testing"

	"github.com/rhysd/actionlint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseActionsExpression(t *testing.T) {
	for _, tt := range []struct {
		name, input, expression string
	}{
		{"bare", "always()", "always()"},
		{"bare whitespace", " \n always() \t", "always()"},
		{"wrapped", "${{always()}}", "always()"},
		{"wrapped whitespace", " \n${{ \n always() \t }} \t", "always()"},
		{"literal delimiters", "contains('${{ text }}', '}}')", "contains('${{ text }}', '}}')"},
		{"wrapped literal delimiters", "${{ contains('${{ text }}', '}}') }}", "contains('${{ text }}', '}}')"},
		{"boolean", "${{ false }}", "false"},
		{"guard", "always() && needs.detection.result == 'success'", "always() && needs.detection.result == 'success'"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			text, node, err := ParseActionsExpression(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expression, text)
			require.NotNil(t, node)
		})
	}
	for _, input := range []string{
		"", " \n\t", "${{ }}", "${{}}", "${{ always()", "always() }}",
		"always() &&", "${{ always() }} trailing", "prefix ${{ always() }}",
		"${{ false }}${{ true }}", "${{ always() }} || true",
		"always() }} && needs.detection.result == 'success'",
		"${{ always() }} && needs.detection.result == 'success' }}",
		"always() # comment",
	} {
		t.Run(input, func(t *testing.T) {
			text, node, err := ParseActionsExpression(input)
			require.Error(t, err)
			assert.Empty(t, text)
			assert.Nil(t, node)
		})
	}
}

func TestParseActionsExpressionReturnsAST(t *testing.T) {
	_, node, err := ParseActionsExpression("${{ always() && needs.detection.result == 'success' }}")
	require.NoError(t, err)
	logical, ok := node.(*actionlint.LogicalOpNode)
	require.True(t, ok)
	assert.Equal(t, "&&", logical.Kind.String())
	require.IsType(t, &actionlint.FuncCallNode{}, logical.Left)
	require.IsType(t, &actionlint.CompareOpNode{}, logical.Right)
}
