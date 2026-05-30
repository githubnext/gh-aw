//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// prop builds a PropertyAccessNode for concise test expressions.
func prop(path string) *PropertyAccessNode { return &PropertyAccessNode{PropertyPath: path} }

// str builds a StringLiteralNode.
func str(v string) *StringLiteralNode { return &StringLiteralNode{Value: v} }

// boolLit builds a BooleanLiteralNode.
func boolLit(v bool) *BooleanLiteralNode { return &BooleanLiteralNode{Value: v} }

// expr builds a raw ExpressionNode.
func expr(e string) *ExpressionNode { return &ExpressionNode{Expression: e} }

// cmp builds a ComparisonNode.
func cmp(left ConditionNode, op string, right ConditionNode) *ComparisonNode {
	return &ComparisonNode{Left: left, Operator: op, Right: right}
}

// fn builds a FunctionCallNode.
func fn(name string, args ...ConditionNode) *FunctionCallNode {
	return &FunctionCallNode{FunctionName: name, Arguments: args}
}

// and2 builds a binary AND.
func and2(left, right ConditionNode) *AndNode { return &AndNode{Left: left, Right: right} }

// or2 builds a binary OR.
func or2(left, right ConditionNode) *OrNode { return &OrNode{Left: left, Right: right} }

// not1 builds a NOT node.
func not1(child ConditionNode) *NotNode { return &NotNode{Child: child} }

// disj builds a DisjunctionNode.
func disj(terms ...ConditionNode) *DisjunctionNode {
	return &DisjunctionNode{Terms: terms, Multiline: false}
}

// assertRender asserts that optimising node produces a result that renders to
// the expected string.
func assertRender(t *testing.T, expected string, node ConditionNode, msgAndArgs ...any) {
	t.Helper()
	result := OptimizeExpression(node)
	require.NotNil(t, result, "OptimizeExpression returned nil")
	assert.Equal(t, expected, result.Render(), msgAndArgs...)
}

// ---------------------------------------------------------------------------
// Nil / leaf nodes
// ---------------------------------------------------------------------------

func TestOptimizeExpression_NilInput(t *testing.T) {
	result := OptimizeExpression(nil)
	assert.Nil(t, result, "nil input should return nil")
}

func TestOptimizeExpression_ExpressionNode(t *testing.T) {
	node := expr("github.event_name == 'issues'")
	assertRender(t, "github.event_name == 'issues'", node, "ExpressionNode should pass through unchanged")
}

func TestOptimizeExpression_PropertyAccessNode(t *testing.T) {
	node := prop("github.event_name")
	assertRender(t, "github.event_name", node, "PropertyAccessNode should pass through unchanged")
}

func TestOptimizeExpression_StringLiteralNode(t *testing.T) {
	node := str("pull_request")
	assertRender(t, "'pull_request'", node, "StringLiteralNode should pass through unchanged")
}

func TestOptimizeExpression_BooleanLiteralTrue(t *testing.T) {
	assertRender(t, "true", boolLit(true), "true literal passes through")
}

func TestOptimizeExpression_BooleanLiteralFalse(t *testing.T) {
	assertRender(t, "false", boolLit(false), "false literal passes through")
}

func TestOptimizeExpression_ComparisonNode(t *testing.T) {
	node := cmp(prop("github.event_name"), "==", str("issues"))
	assertRender(t, "github.event_name == 'issues'", node, "ComparisonNode passes through unchanged")
}

func TestOptimizeExpression_FunctionCallNode(t *testing.T) {
	node := fn("contains", prop("needs.agent.outputs.output_types"), str("push"))
	assertRender(t, "contains(needs.agent.outputs.output_types, 'push')", node, "FunctionCallNode passes through unchanged")
}

// ---------------------------------------------------------------------------
// NOT constant folding
// ---------------------------------------------------------------------------

func TestOptimizeNot_TrueLiteral(t *testing.T) {
	assertRender(t, "false", not1(boolLit(true)), "!true → false")
}

func TestOptimizeNot_FalseLiteral(t *testing.T) {
	assertRender(t, "true", not1(boolLit(false)), "!false → true")
}

// ---------------------------------------------------------------------------
// Double negation elimination
// ---------------------------------------------------------------------------

func TestOptimizeNot_DoubleNegation_Property(t *testing.T) {
	node := not1(not1(prop("github.event_name")))
	assertRender(t, "github.event_name", node, "!!X → X for PropertyAccessNode")
}

func TestOptimizeNot_DoubleNegation_Comparison(t *testing.T) {
	inner := cmp(prop("github.event_name"), "==", str("issues"))
	node := not1(not1(inner))
	assertRender(t, "github.event_name == 'issues'", node, "!!comparison → comparison")
}

func TestOptimizeNot_DoubleNegation_Expression(t *testing.T) {
	node := not1(not1(expr("needs.agent.result != 'skipped'")))
	assertRender(t, "needs.agent.result != 'skipped'", node, "!!ExpressionNode → ExpressionNode")
}

func TestOptimizeNot_TripleNegation(t *testing.T) {
	// !!!X → !X
	inner := prop("github.event_name")
	node := not1(not1(not1(inner)))
	// !!!X = !!(! X) → !X
	assertRender(t, "!(github.event_name)", node, "!!!X → !X")
}

func TestOptimizeNot_QuadrupleNegation(t *testing.T) {
	// !!!!X → X
	inner := prop("github.event_name")
	node := not1(not1(not1(not1(inner))))
	assertRender(t, "github.event_name", node, "!!!!X → X")
}

func TestOptimizeNot_DoubleNegation_FunctionCall(t *testing.T) {
	// !!contains(…) → contains(…) — safe because contains is not a status function
	contains := fn("contains", prop("github.event_name"), str("issues"))
	assertRender(t, "contains(github.event_name, 'issues')", not1(not1(contains)), "!!contains → contains")
}

func TestOptimizeNot_SingleNegation_FunctionCallPreservation(t *testing.T) {
	// !contains(…) should remain as-is (single negation is fine)
	assertRender(t, "!contains(github.event_name, 'issues')", not1(fn("contains", prop("github.event_name"), str("issues"))), "single NOT of function call preserved")
}

// ---------------------------------------------------------------------------
// AND – boolean identity (A && true → A)
// ---------------------------------------------------------------------------

func TestOptimizeAnd_IdentityTrueRight(t *testing.T) {
	node := and2(expr("github.event_name == 'issues'"), boolLit(true))
	assertRender(t, "github.event_name == 'issues'", node, "A && true → A")
}

func TestOptimizeAnd_IdentityTrueLeft(t *testing.T) {
	node := and2(boolLit(true), expr("github.event_name == 'issues'"))
	assertRender(t, "github.event_name == 'issues'", node, "true && A → A")
}

func TestOptimizeAnd_IdentityTrueBoth(t *testing.T) {
	node := and2(boolLit(true), boolLit(true))
	assertRender(t, "true", node, "true && true → true")
}

// ---------------------------------------------------------------------------
// AND – boolean annihilation (A && false → false)
// ---------------------------------------------------------------------------

func TestOptimizeAnd_AnnihilationFalseRight(t *testing.T) {
	node := and2(expr("github.event_name == 'issues'"), boolLit(false))
	assertRender(t, "false", node, "A && false → false")
}

func TestOptimizeAnd_AnnihilationFalseLeft(t *testing.T) {
	node := and2(boolLit(false), expr("github.event_name == 'issues'"))
	assertRender(t, "false", node, "false && A → false")
}

func TestOptimizeAnd_AnnihilationBothFalse(t *testing.T) {
	assertRender(t, "false", and2(boolLit(false), boolLit(false)), "false && false → false")
}

func TestOptimizeAnd_TrueAndFalse(t *testing.T) {
	assertRender(t, "false", and2(boolLit(true), boolLit(false)), "true && false → false (annihilation beats identity)")
}

// ---------------------------------------------------------------------------
// AND – idempotent law (A && A → A)
// ---------------------------------------------------------------------------

func TestOptimizeAnd_IdempotentComparison(t *testing.T) {
	c := cmp(prop("github.event_name"), "==", str("issues"))
	node := and2(c, c)
	assertRender(t, "github.event_name == 'issues'", node, "A && A → A for comparison")
}

func TestOptimizeAnd_IdempotentExpression(t *testing.T) {
	e := expr("needs.agent.result != 'skipped'")
	assertRender(t, "needs.agent.result != 'skipped'", and2(e, e), "A && A → A for ExpressionNode")
}

func TestOptimizeAnd_IdempotentProperty(t *testing.T) {
	p := prop("github.event_name")
	assertRender(t, "github.event_name", and2(p, p), "A && A → A for property")
}

// ---------------------------------------------------------------------------
// AND – complement law (A && !A → false)
// ---------------------------------------------------------------------------

func TestOptimizeAnd_ComplementNotOnRight(t *testing.T) {
	e := expr("needs.agent.result != 'skipped'")
	node := and2(e, not1(e))
	assertRender(t, "false", node, "A && !A → false")
}

func TestOptimizeAnd_ComplementNotOnLeft(t *testing.T) {
	e := expr("needs.agent.result != 'skipped'")
	node := and2(not1(e), e)
	assertRender(t, "false", node, "!A && A → false")
}

// ---------------------------------------------------------------------------
// OR – boolean identity (A || false → A)
// ---------------------------------------------------------------------------

func TestOptimizeOr_IdentityFalseRight(t *testing.T) {
	node := or2(expr("github.event_name == 'issues'"), boolLit(false))
	assertRender(t, "github.event_name == 'issues'", node, "A || false → A")
}

func TestOptimizeOr_IdentityFalseLeft(t *testing.T) {
	node := or2(boolLit(false), expr("github.event_name == 'issues'"))
	assertRender(t, "github.event_name == 'issues'", node, "false || A → A")
}

func TestOptimizeOr_IdentityFalseBoth(t *testing.T) {
	assertRender(t, "false", or2(boolLit(false), boolLit(false)), "false || false → false")
}

// ---------------------------------------------------------------------------
// OR – boolean annihilation (A || true → true)
// ---------------------------------------------------------------------------

func TestOptimizeOr_AnnihilationTrueRight(t *testing.T) {
	node := or2(expr("github.event_name == 'issues'"), boolLit(true))
	assertRender(t, "true", node, "A || true → true")
}

func TestOptimizeOr_AnnihilationTrueLeft(t *testing.T) {
	node := or2(boolLit(true), expr("github.event_name == 'issues'"))
	assertRender(t, "true", node, "true || A → true")
}

func TestOptimizeOr_AnnihilationBothTrue(t *testing.T) {
	assertRender(t, "true", or2(boolLit(true), boolLit(true)), "true || true → true")
}

func TestOptimizeOr_FalseOrTrue(t *testing.T) {
	assertRender(t, "true", or2(boolLit(false), boolLit(true)), "false || true → true")
}

// ---------------------------------------------------------------------------
// OR – idempotent law (A || A → A)
// ---------------------------------------------------------------------------

func TestOptimizeOr_IdempotentComparison(t *testing.T) {
	c := cmp(prop("github.event_name"), "==", str("issues"))
	assertRender(t, "github.event_name == 'issues'", or2(c, c), "A || A → A for comparison")
}

func TestOptimizeOr_IdempotentExpression(t *testing.T) {
	e := expr("needs.agent.result != 'skipped'")
	assertRender(t, "needs.agent.result != 'skipped'", or2(e, e), "A || A → A for expression")
}

// ---------------------------------------------------------------------------
// OR – complement law (A || !A → true)
// ---------------------------------------------------------------------------

func TestOptimizeOr_ComplementNotOnRight(t *testing.T) {
	e := expr("github.event_name == 'issues'")
	node := or2(e, not1(e))
	assertRender(t, "true", node, "A || !A → true")
}

func TestOptimizeOr_ComplementNotOnLeft(t *testing.T) {
	e := expr("github.event_name == 'issues'")
	node := or2(not1(e), e)
	assertRender(t, "true", node, "!A || A → true")
}

// ---------------------------------------------------------------------------
// DisjunctionNode optimisations
// ---------------------------------------------------------------------------

func TestOptimizeDisjunction_NoTerms(t *testing.T) {
	node := &DisjunctionNode{Terms: nil}
	result := OptimizeExpression(node)
	require.NotNil(t, result, "empty disjunction must not be nil")
	assert.IsType(t, &DisjunctionNode{}, result, "empty disjunction stays DisjunctionNode")
}

func TestOptimizeDisjunction_SingleTerm(t *testing.T) {
	e := expr("github.event_name == 'issues'")
	node := disj(e)
	// Single-term DisjunctionNode renders as the term itself.
	assertRender(t, "github.event_name == 'issues'", node, "single-term disjunction → just the term")
}

func TestOptimizeDisjunction_RemovesFalseTerms(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.event_name == 'issue_comment'")
	node := disj(a, boolLit(false), b)
	assertRender(t, "github.event_name == 'issues' || github.event_name == 'issue_comment'", node, "false terms are removed")
}

func TestOptimizeDisjunction_AllFalseTerms(t *testing.T) {
	node := disj(boolLit(false), boolLit(false), boolLit(false))
	assertRender(t, "false", node, "all-false disjunction → false")
}

func TestOptimizeDisjunction_TrueTermShortCircuit(t *testing.T) {
	node := disj(expr("github.event_name == 'issues'"), boolLit(true), expr("github.event_name == 'push'"))
	assertRender(t, "true", node, "true term short-circuits entire disjunction")
}

func TestOptimizeDisjunction_DeduplicatesTerms(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.event_name == 'issue_comment'")
	// Duplicate 'a' twice
	node := disj(a, b, a, a, b)
	assertRender(t, "github.event_name == 'issues' || github.event_name == 'issue_comment'", node, "duplicate terms are removed")
}

func TestOptimizeDisjunction_FalseOnlyOneTerm(t *testing.T) {
	node := disj(boolLit(false))
	// Single false term → false
	assertRender(t, "false", node, "single false term → false")
}

func TestOptimizeDisjunction_MultilinePreserved(t *testing.T) {
	a := &ExpressionNode{Expression: "github.event_name == 'issues'", Description: "Issue events"}
	b := &ExpressionNode{Expression: "github.event_name == 'push'", Description: "Push events"}
	node := &DisjunctionNode{Terms: []ConditionNode{a, b}, Multiline: true}
	result := OptimizeExpression(node)
	// After optimisation the multiline flag must be preserved.
	if dn, ok := result.(*DisjunctionNode); ok {
		assert.True(t, dn.Multiline, "Multiline flag should be preserved after optimisation")
	}
}

func TestOptimizeDisjunction_DeduplicatesAndReducesToSingle(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	// All three are the same term
	node := disj(a, a, a)
	assertRender(t, "github.event_name == 'issues'", node, "three identical terms → one")
}

// ---------------------------------------------------------------------------
// Status function protection
// ---------------------------------------------------------------------------

func TestOptimizeAnd_StatusFunc_AlwaysNotEliminated(t *testing.T) {
	// always() && X must NOT simplify to X even though always() is "truthy"
	always := fn("always")
	condition := expr("steps.run.outcome == 'success'")
	node := and2(always, condition)
	result := OptimizeExpression(node)
	// The rendered output must still contain "always()"
	rendered := result.Render()
	assert.Contains(t, rendered, "always()", "always() must not be eliminated from AND")
	assert.Contains(t, rendered, "steps.run.outcome == 'success'", "other condition must remain")
}

func TestOptimizeAnd_StatusFunc_SuccessNotEliminated(t *testing.T) {
	success := fn("success")
	condition := expr("steps.run.outcome == 'success'")
	node := and2(success, condition)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "success()", "success() must not be eliminated from AND")
}

func TestOptimizeAnd_StatusFunc_FailureNotEliminated(t *testing.T) {
	failure := fn("failure")
	condition := expr("steps.notify.outcome == 'skipped'")
	node := and2(failure, condition)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "failure()", "failure() must not be eliminated from AND")
}

func TestOptimizeAnd_StatusFunc_CancelledNotEliminated(t *testing.T) {
	// !cancelled() && X — both !cancelled() and X must remain
	notCancelled := not1(fn("cancelled"))
	condition := expr("needs.agent.result != 'skipped'")
	node := and2(notCancelled, condition)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "cancelled()", "!cancelled() must not be eliminated")
	assert.Contains(t, rendered, "needs.agent.result != 'skipped'", "other condition must remain")
}

func TestOptimizeAnd_StatusFunc_AlwaysWithTrueRight(t *testing.T) {
	// always() && true  — identity rule must NOT fire because the left is a status function
	always := fn("always")
	node := and2(always, boolLit(true))
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "always()", "always() && true must preserve always()")
}

func TestOptimizeAnd_StatusFunc_TrueWithAlways(t *testing.T) {
	// true && always() — identity rule must NOT fire for right status function
	always := fn("always")
	node := and2(boolLit(true), always)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "always()", "true && always() must preserve always()")
}

func TestOptimizeOr_StatusFunc_NotEliminated(t *testing.T) {
	// always() || X — annihilation rule must not fire (always is not a boolean true)
	always := fn("always")
	condition := expr("github.event_name == 'issues'")
	node := or2(always, condition)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "always()", "always() must not be eliminated from OR")
}

func TestOptimizeAnd_StatusFunc_Idempotent_Skipped(t *testing.T) {
	// always() && always() — with AND chain dedup, this now simplifies to always()
	// since deduplication applies even to status function terms.
	always := fn("always")
	node := and2(always, always)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "always()", "always() && always() must still contain always()")
}

// ---------------------------------------------------------------------------
// Nested / composed expression optimisation
// ---------------------------------------------------------------------------

func TestOptimizeNested_AndInsideOr(t *testing.T) {
	// (A && true) || false → A
	a := expr("github.event_name == 'issues'")
	node := or2(and2(a, boolLit(true)), boolLit(false))
	assertRender(t, "github.event_name == 'issues'", node, "(A && true) || false → A")
}

func TestOptimizeNested_OrInsideAnd(t *testing.T) {
	// (A || false) && true → A
	a := expr("github.event_name == 'issues'")
	node := and2(or2(a, boolLit(false)), boolLit(true))
	assertRender(t, "github.event_name == 'issues'", node, "(A || false) && true → A")
}

func TestOptimizeNested_DoubleNegationInsideAnd(t *testing.T) {
	// !!A && B → A && B
	a := expr("github.event_name == 'issues'")
	b := expr("github.actor != 'bot'")
	node := and2(not1(not1(a)), b)
	assertRender(t, "(github.event_name == 'issues') && (github.actor != 'bot')", node, "!!A && B → A && B")
}

func TestOptimizeNested_ComplexConstantFolding(t *testing.T) {
	// !(!false) && !true → true && false → false
	notNotFalse := not1(not1(boolLit(false)))
	notTrue := not1(boolLit(true))
	node := and2(notNotFalse, notTrue)
	assertRender(t, "false", node, "!(!false) && !true → false")
}

func TestOptimizeNested_DeepDoubleNegation(t *testing.T) {
	// !!(!A) → !A
	a := expr("github.event_name == 'issues'")
	node := not1(not1(not1(a)))
	assertRender(t, "!(github.event_name == 'issues')", node, "!!!A → !A")
}

func TestOptimizeNested_AndWithFalseDeep(t *testing.T) {
	// A && (B && false) → A && false → false
	a := expr("github.event_name == 'issues'")
	b := expr("github.actor != 'bot'")
	node := and2(a, and2(b, boolLit(false)))
	assertRender(t, "false", node, "A && (B && false) → false")
}

func TestOptimizeNested_OrWithTrueDeep(t *testing.T) {
	// A || (B || true) → A || true → true
	a := expr("github.event_name == 'issues'")
	b := expr("github.actor != 'bot'")
	node := or2(a, or2(b, boolLit(true)))
	assertRender(t, "true", node, "A || (B || true) → true")
}

// ---------------------------------------------------------------------------
// Real-world patterns from lock files
// ---------------------------------------------------------------------------

func TestRealWorld_SafeOutputCondition(t *testing.T) {
	// (!cancelled()) && (needs.agent.result != 'skipped')
	// This must remain unchanged because it contains a status function.
	notCancelled := not1(fn("cancelled"))
	agentNotSkipped := cmp(prop("needs.agent.result"), "!=", str("skipped"))
	node := and2(notCancelled, agentNotSkipped)
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, "cancelled()", "cancelled() must be preserved")
	assert.Contains(t, rendered, "needs.agent.result != 'skipped'", "agent-not-skipped condition must be preserved")
}

func TestRealWorld_ReactionConditionDedup(t *testing.T) {
	// Build the same event-type condition twice to simulate a duplicate being
	// introduced by condition-building logic.
	issuesCond := cmp(prop("github.event_name"), "==", str("issues"))
	issuesCond2 := cmp(prop("github.event_name"), "==", str("issues"))
	commentCond := cmp(prop("github.event_name"), "==", str("issue_comment"))
	node := disj(issuesCond, issuesCond2, commentCond)
	result := OptimizeExpression(node)
	rendered := result.Render()
	// 'issues' should appear only once
	count := strings.Count(rendered, "github.event_name == 'issues'")
	assert.Equal(t, 1, count, "duplicate 'issues' condition should be deduped")
	assert.Contains(t, rendered, "github.event_name == 'issue_comment'")
}

func TestRealWorld_ForkCheckNotOptimized(t *testing.T) {
	// github.event.pull_request.head.repo.id == github.repository_id
	// Must remain unchanged (no literals, no duplicates).
	forkCheck := BuildNotFromFork()
	result := OptimizeExpression(forkCheck)
	assert.Equal(t, forkCheck.Render(), result.Render(), "fork check should be preserved unchanged")
}

func TestRealWorld_AlwaysWithDetectionGuard(t *testing.T) {
	// always() && steps.detection_guard.outputs.run_detection == 'true'
	// This is a common pattern in generated lock files; always() must be preserved.
	always := fn("always")
	guard := cmp(prop("steps.detection_guard.outputs.run_detection"), "==", str("true"))
	node := and2(always, guard)
	rendered := OptimizeExpression(node).Render()
	assert.Contains(t, rendered, "always()", "always() preserved in detection guard pattern")
	assert.Contains(t, rendered, "steps.detection_guard.outputs.run_detection == 'true'")
}

func TestRealWorld_ActivationCheck(t *testing.T) {
	// needs.activation.outputs.activated == 'true'  — simple leaf, no change.
	node := cmp(prop("needs.activation.outputs.activated"), "==", str("true"))
	assertRender(t, "needs.activation.outputs.activated == 'true'", node)
}

func TestRealWorld_ComplexSafeOutputCondition(t *testing.T) {
	// ((!cancelled()) && (needs.agent.result != 'skipped')) && contains(needs.agent.outputs.output_types, 'push')
	notCancelled := not1(fn("cancelled"))
	agentNotSkipped := cmp(prop("needs.agent.result"), "!=", str("skipped"))
	containsOutput := fn("contains", prop("needs.agent.outputs.output_types"), str("push"))
	node := and2(and2(notCancelled, agentNotSkipped), containsOutput)
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, "cancelled()", "cancelled() must be preserved in safe output pattern")
	assert.Contains(t, rendered, "needs.agent.result != 'skipped'")
	assert.Contains(t, rendered, "contains(needs.agent.outputs.output_types, 'push')")
}

func TestRealWorld_DuplicateActivationOutputInAnd(t *testing.T) {
	// BuildConditionTree with the same condition passed twice produces A && A → A
	activated := expr("needs.activation.outputs.activated == 'true'")
	node := and2(activated, activated)
	assertRender(t, "needs.activation.outputs.activated == 'true'", node, "A && A → A for activation check")
}

// ---------------------------------------------------------------------------
// Idempotency of OptimizeExpression (calling twice gives same result)
// ---------------------------------------------------------------------------

func TestOptimize_Idempotent_AndIdempotent(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	node := and2(a, a)
	once := OptimizeExpression(node)
	twice := OptimizeExpression(once)
	assert.Equal(t, once.Render(), twice.Render(), "second optimisation pass must be a no-op")
}

func TestOptimize_Idempotent_Complex(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.actor != 'bot'")
	node := and2(or2(a, boolLit(false)), and2(b, boolLit(true)))
	once := OptimizeExpression(node)
	twice := OptimizeExpression(once)
	assert.Equal(t, once.Render(), twice.Render(), "second optimisation must be no-op for complex expression")
}

func TestOptimize_Idempotent_Disjunction(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.event_name == 'push'")
	node := disj(a, b, a)
	once := OptimizeExpression(node)
	twice := OptimizeExpression(once)
	assert.Equal(t, once.Render(), twice.Render(), "second pass must be no-op after deduplication")
}

// ---------------------------------------------------------------------------
// RenderCondition helper
// ---------------------------------------------------------------------------

func TestRenderCondition_SimplifiesIdentity(t *testing.T) {
	// RenderCondition must apply the optimizer – A && true → A
	a := expr("github.event_name == 'issues'")
	result := RenderCondition(and2(a, boolLit(true)))
	assert.Equal(t, "github.event_name == 'issues'", result, "RenderCondition should simplify A && true → A")
}

func TestRenderCondition_PreservesStatusFunc(t *testing.T) {
	always := fn("always")
	condition := expr("steps.run.outcome == 'success'")
	result := RenderCondition(and2(always, condition))
	assert.Contains(t, result, "always()", "RenderCondition must preserve status functions")
}

// ---------------------------------------------------------------------------
// Smart parenthesisation (needsParensAsAndOperand)
// ---------------------------------------------------------------------------

func TestAndNode_Render_FunctionCallNoParens(t *testing.T) {
	// FunctionCallNode (non-NOT) as AND operand should not be wrapped in parens.
	node := and2(fn("contains", prop("x"), str("y")), cmp(prop("a"), "==", str("b")))
	assert.Equal(t, "contains(x, 'y') && a == 'b'", OptimizeExpression(node).Render(), "FunctionCallNode not wrapped in AND")
}

func TestAndNode_Render_ComparisonNoParens(t *testing.T) {
	// ComparisonNode as AND operand should not be wrapped in parens.
	node := and2(cmp(prop("a"), "!=", str("b")), cmp(prop("c"), "==", str("d")))
	assert.Equal(t, "a != 'b' && c == 'd'", OptimizeExpression(node).Render(), "ComparisonNode not wrapped in AND")
}

func TestAndNode_Render_OrNodeWrapped(t *testing.T) {
	// OrNode as AND operand MUST be wrapped to preserve precedence.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	node := and2(or2(a, b), c)
	rendered := OptimizeExpression(node).Render()
	assert.Equal(t, "(a == '1' || b == '2') && c == '3'", rendered, "OrNode operand must be wrapped in AND")
}

func TestOrNode_Render_NoWrapping(t *testing.T) {
	// OrNode never wraps typed children in parens.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	node := or2(a, b)
	assert.Equal(t, "a == '1' || b == '2'", OptimizeExpression(node).Render(), "OrNode should not add parens")
}

func TestOrNode_Render_AndChildNoParens(t *testing.T) {
	// && has higher precedence than ||, so AndNode child of OrNode does NOT need parens.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	node := or2(and2(a, b), c)
	assert.Equal(t, "a == '1' && b == '2' || c == '3'", OptimizeExpression(node).Render(), "AndNode in OR does not need parens")
}

// ---------------------------------------------------------------------------
// AND chain flattening and deduplication
// ---------------------------------------------------------------------------

func TestOptimizeAndChain_Dedup(t *testing.T) {
	// A && (A && B) should deduplicate A → A && B
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	node := and2(a, and2(a, b))
	assertRender(t, "a == '1' && b == '2'", node, "A && (A && B) → A && B via chain dedup")
}

func TestOptimizeAndChain_ThreeDedup(t *testing.T) {
	// A && B && A → A && B
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	node := and2(and2(a, b), a)
	assertRender(t, "a == '1' && b == '2'", node, "(A && B) && A → A && B via chain dedup")
}

// ---------------------------------------------------------------------------
// OR chain flattening
// ---------------------------------------------------------------------------

func TestOptimizeOrChain_Flatten(t *testing.T) {
	// OrNode{OrNode{A,B}, C} → DisjunctionNode{A,B,C} (dedup step, no dups here)
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	node := or2(or2(a, b), c)
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Equal(t, "a == '1' || b == '2' || c == '3'", rendered, "OR chain should be flattened")
}

func TestOptimizeOrChain_FlattenAndDedup(t *testing.T) {
	// OrNode{OrNode{A,B}, A} → DisjunctionNode{A,B} (A is deduped)
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	node := or2(or2(a, b), a)
	assertRender(t, "a == '1' || b == '2'", node, "OR chain with duplicate should dedup")
}

// ---------------------------------------------------------------------------
// Real-world rendered pattern improvements
// ---------------------------------------------------------------------------

func TestRealWorld_AlwaysWithConditionRenderedCleanly(t *testing.T) {
	// always() && X should render WITHOUT extra parens around always()
	always := fn("always")
	guard := cmp(prop("steps.guard.outputs.run"), "==", str("true"))
	result := RenderCondition(and2(always, guard))
	assert.Equal(t, "always() && steps.guard.outputs.run == 'true'", result, "always() && X should render without extra parens")
}

func TestRealWorld_CancelledWithConditionRenderedCleanly(t *testing.T) {
	// !cancelled() && X: NotNode is wrapped in parens as AND operand to avoid YAML ! tag issue.
	notCancelled := not1(fn("cancelled"))
	skipped := cmp(prop("needs.agent.result"), "!=", str("skipped"))
	result := RenderCondition(and2(notCancelled, skipped))
	assert.Equal(t, "(!cancelled()) && needs.agent.result != 'skipped'", result, "!cancelled() wrapped in AND for YAML safety")
}

func TestRealWorld_NestedAndFlatRendering(t *testing.T) {
	// (!cancelled()) && (A != B) && (C == D) should flatten to: (!cancelled()) && A != B && C == D
	notCancelled := not1(fn("cancelled"))
	agentNotSkipped := cmp(prop("needs.agent.result"), "!=", str("skipped"))
	detectionSuccess := cmp(prop("needs.agent.outputs.detection_success"), "==", str("true"))
	node := and2(and2(notCancelled, agentNotSkipped), detectionSuccess)
	result := RenderCondition(node)
	assert.Equal(t, "(!cancelled()) && needs.agent.result != 'skipped' && needs.agent.outputs.detection_success == 'true'", result)
}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Helper predicate tests (white-box)
// ---------------------------------------------------------------------------

func TestIsBoolLiteral(t *testing.T) {
	assert.True(t, isBoolLiteral(boolLit(true), true))
	assert.True(t, isBoolLiteral(boolLit(false), false))
	assert.False(t, isBoolLiteral(boolLit(true), false))
	assert.False(t, isBoolLiteral(expr("x"), true))
}

func TestIsStatusFunc(t *testing.T) {
	assert.True(t, isStatusFunc(fn("always")), "always is a status func")
	assert.True(t, isStatusFunc(fn("success")), "success is a status func")
	assert.True(t, isStatusFunc(fn("failure")), "failure is a status func")
	assert.True(t, isStatusFunc(fn("cancelled")), "cancelled is a status func")
	assert.False(t, isStatusFunc(fn("contains")), "contains is not a status func")
	assert.False(t, isStatusFunc(fn("startsWith")), "startsWith is not a status func")
	assert.False(t, isStatusFunc(expr("x")), "ExpressionNode is not a status func")
}

func TestNodesEqual(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.event_name == 'issues'")
	c := expr("github.event_name == 'push'")
	assert.True(t, nodesEqual(a, b), "identical renders are equal")
	assert.False(t, nodesEqual(a, c), "different renders are not equal")
	assert.True(t, nodesEqual(nil, nil), "nil == nil")
	assert.False(t, nodesEqual(a, nil), "non-nil != nil")
	assert.False(t, nodesEqual(nil, a), "nil != non-nil")
}

func TestContainsStatusFunc(t *testing.T) {
	assert.True(t, containsStatusFunc(fn("always")), "direct always()")
	assert.True(t, containsStatusFunc(not1(fn("cancelled"))), "!cancelled() contains status func")
	assert.True(t, containsStatusFunc(and2(fn("success"), expr("x"))), "success() in AND")
	assert.False(t, containsStatusFunc(expr("github.event_name == 'issues'")), "plain expr has no status func")
	assert.False(t, containsStatusFunc(fn("contains", prop("x"), str("y"))), "contains has no status func")
	assert.False(t, containsStatusFunc(and2(expr("a"), expr("b"))), "plain AND has no status func")
}

func TestIsNegationOf(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := not1(a)
	assert.True(t, isNegationOf(a, b), "A and !A are negations")
	assert.True(t, isNegationOf(b, a), "!A and A are negations (symmetric)")
	c := expr("github.actor != 'bot'")
	assert.False(t, isNegationOf(a, c), "different expressions are not negations")
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestOptimizeExpression_SingleTermAndNode_NoOp(t *testing.T) {
	// A structure that needs no optimisation should be returned as-is (structurally).
	a := expr("needs.agent.result != 'skipped'")
	b := expr("needs.agent.outputs.detection_success == 'true'")
	node := and2(a, b)
	result := OptimizeExpression(node)
	assert.Equal(t, node.Render(), result.Render(), "already-minimal AND is unchanged")
}

func TestOptimizeExpression_SingleTermOrNode_NoOp(t *testing.T) {
	a := expr("github.event_name == 'issues'")
	b := expr("github.event_name == 'push'")
	node := or2(a, b)
	result := OptimizeExpression(node)
	assert.Equal(t, node.Render(), result.Render(), "already-minimal OR is unchanged")
}

func TestOptimizeExpression_ChainedAnd_BoolLiterals(t *testing.T) {
	// true && true && A → A  (left-associative: (true && true) && A)
	a := expr("github.event_name == 'issues'")
	node := and2(and2(boolLit(true), boolLit(true)), a)
	assertRender(t, "github.event_name == 'issues'", node, "true && true && A → A")
}

func TestOptimizeExpression_ChainedOr_BoolLiterals(t *testing.T) {
	// false || false || A → A
	a := expr("github.event_name == 'issues'")
	node := or2(or2(boolLit(false), boolLit(false)), a)
	assertRender(t, "github.event_name == 'issues'", node, "false || false || A → A")
}

func TestOptimizeExpression_DisjunctionSingleFalseTerm(t *testing.T) {
	node := disj(boolLit(false))
	assertRender(t, "false", node, "single false in disjunction → false")
}

func TestOptimizeExpression_DisjunctionSingleTrueTerm(t *testing.T) {
	node := disj(boolLit(true))
	assertRender(t, "true", node, "single true in disjunction → true")
}

func TestOptimizeExpression_NotOfNot_FunctionCall_NonStatus(t *testing.T) {
	// !!contains(…) → contains(…)
	contains := fn("contains", prop("github.labels"), str("bug"))
	node := not1(not1(contains))
	assertRender(t, "contains(github.labels, 'bug')", node, "!!contains → contains")
}

func TestOptimizeExpression_NotOfAndNode(t *testing.T) {
	// !(A && B) → !A || !B by De Morgan's law
	a := expr("github.event_name == 'issues'")
	b := expr("github.actor != 'bot'")
	node := not1(and2(a, b))
	assertRender(t, "!(github.event_name == 'issues') || !(github.actor != 'bot')", node, "!(A && B) → !A || !B via De Morgan")
}

func TestOptimizeExpression_DisjunctionWithDuplicateStatusFunc(t *testing.T) {
	// DisjunctionNode with duplicate always() calls — dedup is skipped via containsStatusFunc?
	// Actually dedup IS safe for DisjunctionNode because we just remove the render-equal duplicate.
	a := fn("always")
	node := disj(a, a, expr("github.event_name == 'push'"))
	result := OptimizeExpression(node)
	rendered := result.Render()
	// The rendered output should not contain "always() || always()"
	assert.NotContains(t, rendered, "always() || always()", "duplicate always() in disjunction should be deduped")
}

func TestOptimizeExpression_NilSafeForAllBranches(t *testing.T) {
	// Ensure that calling optimizeNode on each concrete type does not panic.
	nodes := []ConditionNode{
		expr("x"),
		prop("y"),
		str("z"),
		boolLit(true),
		boolLit(false),
		cmp(prop("a"), "==", str("b")),
		fn("always"),
		fn("contains", prop("x"), str("y")),
		not1(expr("x")),
		and2(expr("x"), expr("y")),
		or2(expr("x"), expr("y")),
		disj(expr("x"), expr("y")),
	}
	for _, n := range nodes {
		assert.NotPanics(t, func() { OptimizeExpression(n) }, "should not panic for node %T", n)
	}
}

// ---------------------------------------------------------------------------
// De Morgan's law transformations
// ---------------------------------------------------------------------------

func TestDeMorgan_NotOfAnd_NoStatusFunc(t *testing.T) {
	// !(A && B) → !A || !B
	// NotNode children of OrNode are NOT wrapped (|| has lowest precedence, no child needs parens).
	a := cmp(prop("github.event_name"), "==", str("issues"))
	b := cmp(prop("github.actor"), "!=", str("bot"))
	node := not1(and2(a, b))
	assertRender(t, "!(github.event_name == 'issues') || !(github.actor != 'bot')", node, "De Morgan: !(A && B) → !A || !B")
}

func TestDeMorgan_NotOfOr_NoStatusFunc(t *testing.T) {
	// !(A || B) → !A && !B
	// NotNode children of AndNode ARE wrapped in parens (YAML ! tag safety).
	// So the result is "(!(A)) && (!(B))".
	a := cmp(prop("github.event_name"), "==", str("issues"))
	b := cmp(prop("github.event_name"), "==", str("push"))
	node := not1(or2(a, b))
	assertRender(t, "(!(github.event_name == 'issues')) && (!(github.event_name == 'push'))", node, "De Morgan: !(A || B) → !A && !B")
}

func TestDeMorgan_NotOfDisjunction_NoStatusFunc(t *testing.T) {
	// !(A || B || C) → !A && !B && !C  (DisjunctionNode form)
	a := cmp(prop("github.event_name"), "==", str("issues"))
	b := cmp(prop("github.event_name"), "==", str("push"))
	c := cmp(prop("github.event_name"), "==", str("pull_request"))
	node := not1(disj(a, b, c))
	assertRender(t,
		"(!(github.event_name == 'issues')) && (!(github.event_name == 'push')) && (!(github.event_name == 'pull_request'))",
		node, "De Morgan: !(Disjunction{A,B,C}) → !A && !B && !C")
}

func TestDeMorgan_NotOfAnd_WithStatusFunc_Preserved(t *testing.T) {
	// !(always() && X) — must NOT apply De Morgan because of status function
	always := fn("always")
	condition := cmp(prop("steps.run.outcome"), "==", str("success"))
	node := not1(and2(always, condition))
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, "always()", "De Morgan must not fire on status function AND")
	// The result should still be a NOT wrapping an AND, not split to OR
	assert.NotContains(t, rendered, "||", "De Morgan must not produce OR when status func present")
}

func TestDeMorgan_NotOfOr_WithStatusFunc_Preserved(t *testing.T) {
	// !(always() || X) — must NOT apply De Morgan
	always := fn("always")
	condition := cmp(prop("steps.run.outcome"), "==", str("success"))
	node := not1(or2(always, condition))
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, "always()", "De Morgan must not fire on status function OR")
}

func TestDeMorgan_NotOfAnd_SelfComplement(t *testing.T) {
	// Multi-step: !(A && !A)
	// Pass 1: De Morgan: !(A && !A) → !A || !!A
	// Pass 2: double-negation: !!A → A  →  !A || A
	// Pass 3: OR complement: !A || A → true
	a := cmp(prop("github.event_name"), "==", str("issues"))
	node := not1(and2(a, not1(a)))
	assertRender(t, "true", node, "!(A && !A) → true via De Morgan + double-neg + complement")
}

func TestDeMorgan_NotOfOr_SelfComplement(t *testing.T) {
	// Multi-step: !(A || A)
	// Pass 1: De Morgan: !(A || A) → !A && !A
	// Pass 2: AND idempotent: !A && !A → !A
	a := cmp(prop("github.event_name"), "==", str("issues"))
	node := not1(or2(a, a))
	assertRender(t, "!(github.event_name == 'issues')", node, "!(A || A) → !A via De Morgan + idempotent")
}

func TestDeMorgan_NotOfAnd_DoubleNeg_Simplification(t *testing.T) {
	// Multi-step: !(!!A && B)
	// Pass 1 (bottom-up): !!A → A  →  !(A && B)
	// Pass 1 (De Morgan):  !(A && B) → !A || !B
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	node := not1(and2(not1(not1(a)), b))
	assertRender(t, "!(a == '1') || !(b == '2')", node, "!(!!A && B) → !A || !B via double-neg + De Morgan")
}

// ---------------------------------------------------------------------------
// DisjunctionNode complement law (A || !A → true)
// ---------------------------------------------------------------------------

func TestOptimizeDisjunction_ComplementPair(t *testing.T) {
	// disj(A, !A) → true
	a := cmp(prop("github.event_name"), "==", str("issues"))
	node := disj(a, not1(a))
	assertRender(t, "true", node, "disj(A, !A) → true via complement")
}

func TestOptimizeDisjunction_ComplementSymmetric(t *testing.T) {
	// disj(!A, A) → true (symmetric)
	a := cmp(prop("github.event_name"), "==", str("issues"))
	node := disj(not1(a), a)
	assertRender(t, "true", node, "disj(!A, A) → true via complement (symmetric)")
}

func TestOptimizeDisjunction_ComplementWithExtraTerms(t *testing.T) {
	// disj(A, B, !A) → true even when the complement pair is not adjacent
	a := cmp(prop("github.event_name"), "==", str("issues"))
	b := cmp(prop("github.actor"), "!=", str("bot"))
	node := disj(a, b, not1(a))
	assertRender(t, "true", node, "disj(A, B, !A) → true when complement pair is non-adjacent")
}

func TestOptimizeDisjunction_ComplementWithStatusFunc_Preserved(t *testing.T) {
	// disj(always(), !always()) — complement rule must NOT fire because of status function
	always := fn("always")
	notAlways := not1(always)
	node := disj(always, notAlways)
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.NotEqual(t, "true", rendered, "complement must not fire on status function in disjunction")
	assert.Contains(t, rendered, "always()", "always() must be preserved")
}

// ---------------------------------------------------------------------------
// Utility function tests (white-box)
// ---------------------------------------------------------------------------

func TestCollectOrTerms_Flat(t *testing.T) {
	// collectOrTerms on a leaf returns just the leaf.
	a := cmp(prop("a"), "==", str("1"))
	terms := collectOrTerms(a)
	require.Len(t, terms, 1, "leaf should produce one term")
	assert.Equal(t, "a == '1'", terms[0].Render())
}

func TestCollectOrTerms_OrNode(t *testing.T) {
	// collectOrTerms flattens a two-level OR.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	terms := collectOrTerms(or2(or2(a, b), c))
	require.Len(t, terms, 3, "should collect 3 terms from nested OR")
}

func TestCollectOrTerms_DisjunctionNode(t *testing.T) {
	// collectOrTerms flattens a DisjunctionNode.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	terms := collectOrTerms(disj(a, b))
	require.Len(t, terms, 2, "should collect 2 terms from DisjunctionNode")
}

func TestCollectOrTerms_MixedOrAndDisj(t *testing.T) {
	// collectOrTerms flattens OrNode{DisjunctionNode{A,B}, C}.
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	terms := collectOrTerms(or2(disj(a, b), c))
	require.Len(t, terms, 3, "should collect 3 terms from OrNode with DisjunctionNode child")
}

func TestCollectAndTerms_Flat(t *testing.T) {
	// collectAndTerms on a leaf returns just the leaf.
	a := cmp(prop("a"), "==", str("1"))
	terms := collectAndTerms(a)
	require.Len(t, terms, 1, "leaf should produce one term")
	assert.Equal(t, "a == '1'", terms[0].Render())
}

func TestCollectAndTerms_AndNode(t *testing.T) {
	// collectAndTerms flattens A && (B && C).
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	terms := collectAndTerms(and2(a, and2(b, c)))
	require.Len(t, terms, 3, "should collect 3 terms from nested AND")
}

func TestRebuildAndChain_Single(t *testing.T) {
	a := cmp(prop("a"), "==", str("1"))
	result := rebuildAndChain([]ConditionNode{a})
	assert.Equal(t, "a == '1'", result.Render(), "single term rebuilds without AND")
}

func TestRebuildAndChain_Two(t *testing.T) {
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	result := rebuildAndChain([]ConditionNode{a, b})
	assert.Equal(t, "a == '1' && b == '2'", result.Render(), "two terms rebuild as binary AND")
}

func TestRebuildAndChain_Three(t *testing.T) {
	a := cmp(prop("a"), "==", str("1"))
	b := cmp(prop("b"), "==", str("2"))
	c := cmp(prop("c"), "==", str("3"))
	result := rebuildAndChain([]ConditionNode{a, b, c})
	// Left-folded: (a && b) && c
	assert.Equal(t, "a == '1' && b == '2' && c == '3'", result.Render(), "three terms rebuild as left-folded AND chain")
}

// ---------------------------------------------------------------------------
// Z3-inspired rules: Absorption, Subsumption, Resolution, Factoring
// ---------------------------------------------------------------------------

// --- Absorption (AND) -------------------------------------------------------

// A && (A || B) → A
func TestAbsorption_And_LeftAbsorbsRight(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	// a && (a || b) → a
	node := and2(a, or2(a, b))
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "A && (A || B) should reduce to A")
}

// A && (B || A) → A  (OR term appears second)
func TestAbsorption_And_LeftAbsorbsRight_Reversed(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := and2(a, or2(b, a))
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "A && (B || A) should reduce to A")
}

// (A || B) && A → A  (OR on left)
func TestAbsorption_And_RightAbsorbsLeft(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := and2(or2(a, b), a)
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "(A || B) && A should reduce to A")
}

// --- Absorption (OR) -------------------------------------------------------

// A || (A && B) → A
func TestAbsorption_Or_LeftAbsorbsRight(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := or2(a, and2(a, b))
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "A || (A && B) should reduce to A")
}

// (A && B) || A → A  (AND on left)
func TestAbsorption_Or_RightAbsorbsLeft(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := or2(and2(a, b), a)
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "(A && B) || A should reduce to A")
}

// A || (B && A) → A  (conjunct appears second in AND)
func TestAbsorption_Or_ConjunctSecond(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := or2(a, and2(b, a))
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "A || (B && A) should reduce to A")
}

// Status function blocks absorption.
func TestAbsorption_StatusFunc_Preserved(t *testing.T) {
	a := fn("always")
	b := cmp(prop("x"), "==", str("1"))
	// always() || (always() && x == '1') — must NOT be absorbed; always() is guarded
	node := or2(a, and2(a, b))
	result := OptimizeExpression(node)
	// Should not simplify to just always() since status funcs are guarded.
	rendered := result.Render()
	assert.Contains(t, rendered, "always()", "status function must be preserved in absorption")
}

// --- Subsumption (Disjunction) ---------------------------------------------

// disj(A, A&&B) → A  (A&&B is subsumed by A)
func TestSubsumption_Disj_BasicSubsumed(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := disj(a, and2(a, b))
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "disj(A, A&&B) should reduce to A")
}

// disj(A&&B, A) → A  (order reversed)
func TestSubsumption_Disj_SubsumingTermSecond(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	node := disj(and2(a, b), a)
	result := OptimizeExpression(node)
	assert.Equal(t, a.Render(), result.Render(), "disj(A&&B, A) should reduce to A")
}

// disj(A, B, A&&C) → disj(A, B)  (only the subsumed term removed)
func TestSubsumption_Disj_WithExtraTerms(t *testing.T) {
	a := cmp(prop("x"), "==", str("1"))
	b := cmp(prop("y"), "==", str("2"))
	c := cmp(prop("z"), "==", str("3"))
	node := disj(a, b, and2(a, c))
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, a.Render(), "A should survive subsumption")
	assert.Contains(t, rendered, b.Render(), "B should survive (not subsumed)")
	assert.NotContains(t, rendered, c.Render(), "A&&C should be subsumed, c should not appear alone")
}

// Status function blocks subsumption.
func TestSubsumption_Disj_StatusFunc_Preserved(t *testing.T) {
	a := fn("success")
	b := cmp(prop("x"), "==", str("1"))
	node := disj(a, and2(a, b))
	result := OptimizeExpression(node)
	rendered := result.Render()
	assert.Contains(t, rendered, "success()", "status function must be preserved in subsumption")
}
