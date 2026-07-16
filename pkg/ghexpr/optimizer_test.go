//go:build !integration

package ghexpr_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/ghexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers for building nodes concisely in tests.
func prop(path string) *ghexpr.PropertyAccessNode {
	return &ghexpr.PropertyAccessNode{PropertyPath: path}
}
func str(v string) *ghexpr.StringLiteralNode { return &ghexpr.StringLiteralNode{Value: v} }
func blit(v bool) *ghexpr.BooleanLiteralNode { return &ghexpr.BooleanLiteralNode{Value: v} }
func expr(e string) *ghexpr.ExpressionNode   { return &ghexpr.ExpressionNode{Expression: e} }
func fn(name string, args ...ghexpr.ConditionNode) *ghexpr.FunctionCallNode {
	return &ghexpr.FunctionCallNode{FunctionName: name, Arguments: args}
}

func assertOptimized(t *testing.T, want string, node ghexpr.ConditionNode) {
	t.Helper()
	result := ghexpr.OptimizeExpression(node)
	require.NotNil(t, result)
	assert.Equal(t, want, result.Render())
}

// --- nil / leaf ---

func TestOptimizeExpression_NilInput(t *testing.T) {
	assert.Nil(t, ghexpr.OptimizeExpression(nil))
}

func TestOptimizeExpression_Leaf(t *testing.T) {
	assertOptimized(t, "github.event_name == 'issues'", expr("github.event_name == 'issues'"))
	assertOptimized(t, "github.event_name", prop("github.event_name"))
	assertOptimized(t, "'push'", str("push"))
	assertOptimized(t, "true", blit(true))
	assertOptimized(t, "false", blit(false))
}

// --- NOT constant folding ---

func TestOptimizeExpression_NOT_ConstantFolding(t *testing.T) {
	assertOptimized(t, "false", &ghexpr.NotNode{Child: blit(true)})
	assertOptimized(t, "true", &ghexpr.NotNode{Child: blit(false)})
}

// --- double negation ---

func TestOptimizeExpression_DoubleNegation(t *testing.T) {
	assertOptimized(t, "a", &ghexpr.NotNode{Child: &ghexpr.NotNode{Child: expr("a")}})
}

// --- AND identity and annihilation ---

func TestOptimizeExpression_AND_Identity(t *testing.T) {
	assertOptimized(t, "a", &ghexpr.AndNode{Left: expr("a"), Right: blit(true)})
	assertOptimized(t, "a", &ghexpr.AndNode{Left: blit(true), Right: expr("a")})
}

func TestOptimizeExpression_AND_Annihilation(t *testing.T) {
	assertOptimized(t, "false", &ghexpr.AndNode{Left: expr("a"), Right: blit(false)})
	assertOptimized(t, "false", &ghexpr.AndNode{Left: blit(false), Right: expr("a")})
}

// --- OR identity and annihilation ---

func TestOptimizeExpression_OR_Identity(t *testing.T) {
	assertOptimized(t, "a", &ghexpr.OrNode{Left: expr("a"), Right: blit(false)})
	assertOptimized(t, "a", &ghexpr.OrNode{Left: blit(false), Right: expr("a")})
}

func TestOptimizeExpression_OR_Annihilation(t *testing.T) {
	assertOptimized(t, "true", &ghexpr.OrNode{Left: expr("a"), Right: blit(true)})
}

// --- idempotent ---

func TestOptimizeExpression_Idempotent(t *testing.T) {
	assertOptimized(t, "a", &ghexpr.AndNode{Left: expr("a"), Right: expr("a")})
	assertOptimized(t, "a", &ghexpr.OrNode{Left: expr("a"), Right: expr("a")})
}

// --- De Morgan ---

func TestOptimizeExpression_DeMorgan_AND(t *testing.T) {
	node := &ghexpr.NotNode{Child: &ghexpr.AndNode{Left: expr("a"), Right: expr("b")}}
	assertOptimized(t, "!(a) || !(b)", node)
}

func TestOptimizeExpression_DeMorgan_OR(t *testing.T) {
	node := &ghexpr.NotNode{Child: &ghexpr.OrNode{Left: expr("a"), Right: expr("b")}}
	assertOptimized(t, "(!(a)) && (!(b))", node)
}

// --- Status functions are preserved ---

func TestOptimizeExpression_StatusFunctionPreserved(t *testing.T) {
	// always() && true should simplify but always() must survive
	node := &ghexpr.AndNode{Left: fn("always"), Right: blit(true)}
	result := ghexpr.OptimizeExpression(node)
	require.NotNil(t, result)
	rendered := result.Render()
	assert.Contains(t, rendered, "always()", "status function must be preserved")
}

// --- Disjunction short-circuit and dedup ---

func TestOptimizeExpression_Disjunction_TrueShortCircuit(t *testing.T) {
	node := &ghexpr.DisjunctionNode{Terms: []ghexpr.ConditionNode{expr("a"), blit(true), expr("b")}}
	assertOptimized(t, "true", node)
}

func TestOptimizeExpression_Disjunction_FalseFilter(t *testing.T) {
	node := &ghexpr.DisjunctionNode{Terms: []ghexpr.ConditionNode{blit(false), expr("a"), blit(false)}}
	assertOptimized(t, "a", node)
}

func TestOptimizeExpression_Disjunction_Dedup(t *testing.T) {
	node := &ghexpr.DisjunctionNode{Terms: []ghexpr.ConditionNode{expr("a"), expr("b"), expr("a")}}
	result := ghexpr.OptimizeExpression(node)
	require.NotNil(t, result)
	rendered := result.Render()
	assert.Equal(t, "a || b", rendered)
}
