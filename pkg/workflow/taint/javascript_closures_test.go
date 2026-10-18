package taint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptModuleLocalNamesExcludeNestedBlocks(t *testing.T) {
	declaration := func(name string) *JavaScriptNode {
		return &JavaScriptNode{Kind: "VariableStatement", DeclarationList: &JavaScriptNode{
			Kind: "VariableDeclarationList", Declarations: []*JavaScriptNode{{
				Kind: "VariableDeclaration", Name: &JavaScriptNode{Kind: "Identifier", Text: name},
			}},
		}}
	}
	source := &JavaScriptNode{Kind: "SourceFile", Statements: []*JavaScriptNode{
		declaration("moduleLocal"),
		{Kind: "Block", Statements: []*JavaScriptNode{declaration("process")}},
		{Kind: "IfStatement", ThenStatement: &JavaScriptNode{Kind: "Block", Statements: []*JavaScriptNode{declaration("global")}}},
	}}
	assert.Equal(t, map[string]struct{}{"moduleLocal": {}}, jsModuleLocalNames(source))
	member := func(object *JavaScriptNode, name string) *JavaScriptNode {
		return &JavaScriptNode{Kind: "PropertyAccessExpression", Expression: object, Name: &JavaScriptNode{Kind: "Identifier", Text: name}}
	}
	source.Statements = append([]*JavaScriptNode{{
		Kind: "ExpressionStatement", Expression: &JavaScriptNode{
			Kind: "CallExpression", Line: 1,
			Expression: member(&JavaScriptNode{Kind: "Identifier", Text: "console"}, "log"),
			Arguments:  []*JavaScriptNode{member(member(&JavaScriptNode{Kind: "Identifier", Text: "process"}, "env"), "TOKEN")},
		},
	}}, source.Statements...)
	a := &analyzer{rule: SecretLeakRule()}
	j := newJavaScript(a, JavaScriptInput{
		Name: "entry.cjs", Env: map[string][]Source{"TOKEN": {{Reference: "secrets.token"}}},
	}, nil)
	j.eval(source)
	require.Len(t, a.result.Findings, 1, "a later block declaration must not hide the runtime global before the block")
}
