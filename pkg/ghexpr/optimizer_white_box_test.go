//go:build !integration

// White-box tests for unexported optimizer helpers. Must use package ghexpr
// (not ghexpr_test) to access unexported symbols.
package ghexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helpers identical to those in optimizer_test.go but re-declared here for
// access to unexported symbols (different test package).
func wbProp(path string) *PropertyAccessNode { return &PropertyAccessNode{PropertyPath: path} }
func wbStr(v string) *StringLiteralNode      { return &StringLiteralNode{Value: v} }
func wbBool(v bool) *BooleanLiteralNode      { return &BooleanLiteralNode{Value: v} }
func wbExpr(e string) *ExpressionNode        { return &ExpressionNode{Expression: e} }
func wbFn(name string, args ...ConditionNode) *FunctionCallNode {
	return &FunctionCallNode{FunctionName: name, Arguments: args}
}
func wbAnd(l, r ConditionNode) *AndNode { return &AndNode{Left: l, Right: r} }
func wbOr(l, r ConditionNode) *OrNode   { return &OrNode{Left: l, Right: r} }
func wbNot(c ConditionNode) *NotNode    { return &NotNode{Child: c} }
func wbCmp(l ConditionNode, op string, r ConditionNode) *ComparisonNode {
	return &ComparisonNode{Left: l, Operator: op, Right: r}
}
func wbDisj(terms ...ConditionNode) *DisjunctionNode {
	return &DisjunctionNode{Terms: terms, Multiline: false}
}

// ---------------------------------------------------------------------------
// isBoolLiteral
// ---------------------------------------------------------------------------

func TestIsBoolLiteral(t *testing.T) {
	assert.True(t, isBoolLiteral(wbBool(true), true))
	assert.True(t, isBoolLiteral(wbBool(false), false))
	assert.False(t, isBoolLiteral(wbBool(true), false))
	assert.False(t, isBoolLiteral(wbExpr("x"), true))
}

// ---------------------------------------------------------------------------
// isStatusFunc
// ---------------------------------------------------------------------------

func TestIsStatusFunc(t *testing.T) {
	assert.True(t, isStatusFunc(wbFn("always")), "always is a status func")
	assert.True(t, isStatusFunc(wbFn("success")), "success is a status func")
	assert.True(t, isStatusFunc(wbFn("failure")), "failure is a status func")
	assert.True(t, isStatusFunc(wbFn("cancelled")), "cancelled is a status func")
	assert.False(t, isStatusFunc(wbFn("contains")), "contains is not a status func")
	assert.False(t, isStatusFunc(wbFn("startsWith")), "startsWith is not a status func")
	assert.False(t, isStatusFunc(wbExpr("x")), "ExpressionNode is not a status func")
}

// ---------------------------------------------------------------------------
// nodesEqual
// ---------------------------------------------------------------------------

func TestNodesEqual(t *testing.T) {
	a := wbExpr("github.event_name == 'issues'")
	b := wbExpr("github.event_name == 'issues'")
	c := wbExpr("github.event_name == 'push'")
	assert.True(t, nodesEqual(a, b), "identical renders are equal")
	assert.False(t, nodesEqual(a, c), "different renders are not equal")
	assert.True(t, nodesEqual(nil, nil), "nil == nil")
	assert.False(t, nodesEqual(a, nil), "non-nil != nil")
	assert.False(t, nodesEqual(nil, a), "nil != non-nil")
}

// ---------------------------------------------------------------------------
// containsStatusFunc
// ---------------------------------------------------------------------------

func TestContainsStatusFunc(t *testing.T) {
	assert.True(t, containsStatusFunc(wbFn("always")), "direct always()")
	assert.True(t, containsStatusFunc(wbNot(wbFn("cancelled"))), "!cancelled() contains status func")
	assert.True(t, containsStatusFunc(wbAnd(wbFn("success"), wbExpr("x"))), "success() in AND")
	assert.False(t, containsStatusFunc(wbExpr("github.event_name == 'issues'")), "plain expr has no status func")
	assert.False(t, containsStatusFunc(wbFn("contains", wbProp("x"), wbStr("y"))), "contains has no status func")
	assert.False(t, containsStatusFunc(wbAnd(wbExpr("a"), wbExpr("b"))), "plain AND has no status func")
}

// ---------------------------------------------------------------------------
// isNegationOf
// ---------------------------------------------------------------------------

func TestIsNegationOf(t *testing.T) {
	a := wbExpr("github.event_name == 'issues'")
	b := wbNot(a)
	assert.True(t, isNegationOf(a, b), "A and !A are negations")
	assert.True(t, isNegationOf(b, a), "!A and A are negations (symmetric)")
	c := wbExpr("github.actor != 'bot'")
	assert.False(t, isNegationOf(a, c), "different expressions are not negations")
}

// ---------------------------------------------------------------------------
// collectOrTerms
// ---------------------------------------------------------------------------

func TestCollectOrTerms_Flat(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	terms := collectOrTerms(a)
	require.Len(t, terms, 1, "leaf should produce one term")
	assert.Equal(t, "a == '1'", terms[0].Render())
}

func TestCollectOrTerms_OrNode(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	c := wbCmp(wbProp("c"), "==", wbStr("3"))
	terms := collectOrTerms(wbOr(wbOr(a, b), c))
	require.Len(t, terms, 3, "should collect 3 terms from nested OR")
}

func TestCollectOrTerms_DisjunctionNode(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	terms := collectOrTerms(wbDisj(a, b))
	require.Len(t, terms, 2, "should collect 2 terms from DisjunctionNode")
}

func TestCollectOrTerms_MixedOrAndDisj(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	c := wbCmp(wbProp("c"), "==", wbStr("3"))
	terms := collectOrTerms(wbOr(wbDisj(a, b), c))
	require.Len(t, terms, 3, "should collect 3 terms from OrNode with DisjunctionNode child")
}

// ---------------------------------------------------------------------------
// collectAndTerms
// ---------------------------------------------------------------------------

func TestCollectAndTerms_Flat(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	terms := collectAndTerms(a)
	require.Len(t, terms, 1, "leaf should produce one term")
	assert.Equal(t, "a == '1'", terms[0].Render())
}

func TestCollectAndTerms_AndNode(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	c := wbCmp(wbProp("c"), "==", wbStr("3"))
	terms := collectAndTerms(wbAnd(a, wbAnd(b, c)))
	require.Len(t, terms, 3, "should collect 3 terms from nested AND")
}

// ---------------------------------------------------------------------------
// rebuildAndChain
// ---------------------------------------------------------------------------

func TestRebuildAndChain_Single(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	result := rebuildAndChain([]ConditionNode{a})
	assert.Equal(t, "a == '1'", result.Render(), "single term rebuilds without AND")
}

func TestRebuildAndChain_Two(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	result := rebuildAndChain([]ConditionNode{a, b})
	assert.Equal(t, "a == '1' && b == '2'", result.Render(), "two terms rebuild as binary AND")
}

func TestRebuildAndChain_Three(t *testing.T) {
	a := wbCmp(wbProp("a"), "==", wbStr("1"))
	b := wbCmp(wbProp("b"), "==", wbStr("2"))
	c := wbCmp(wbProp("c"), "==", wbStr("3"))
	result := rebuildAndChain([]ConditionNode{a, b, c})
	assert.Equal(t, "a == '1' && b == '2' && c == '3'", result.Render(), "three terms rebuild as left-folded AND chain")
}
