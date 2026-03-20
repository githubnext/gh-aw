package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var expressionOptimizerLog = logger.New("workflow:expression_optimizer")

// OptimizeExpression applies boolean algebra simplifications to a ConditionNode tree,
// returning an equivalent but potentially simpler and shorter expression.
//
// Rules applied (in bottom-up order):
//   - Constant folding:      !true → false, !false → true
//   - Double negation:       !!A → A
//   - Boolean identity:      A && true → A,  A || false → A
//   - Boolean annihilation:  A && false → false, A || true → true
//   - Idempotent law:        A && A → A,  A || A → A
//   - Complement law:        A && !A → false, A || !A → true
//   - DisjunctionNode:       deduplication of terms, removal of false terms,
//     short-circuit on true terms
//
// SAFETY: GitHub Actions status functions (always, success, failure, cancelled)
// have semantics beyond plain booleans – they control step execution based on
// prior step/job status. The optimizer therefore never eliminates a status
// function call from an expression; it only applies rules when both operands of
// && / || are free of status functions.
//
// Execution is bounded: at most maxOptimizationPasses bottom-up passes are
// performed so the optimizer always terminates in O(n * maxOptimizationPasses)
// time relative to the number of nodes in the tree.
func OptimizeExpression(node ConditionNode) ConditionNode {
	if node == nil {
		return nil
	}

	const maxOptimizationPasses = 10

	current := node
	for pass := range maxOptimizationPasses {
		next := optimizeNode(current)
		// Stop early when the rendered form has stabilised (fixed point).
		if next.Render() == current.Render() {
			expressionOptimizerLog.Printf("Expression stabilised after %d pass(es)", pass+1)
			break
		}
		current = next
	}
	return current
}

// optimizeNode performs a single bottom-up optimisation pass over the tree.
// It recurses into children first so that simplifications at lower levels can
// unlock further simplifications at higher levels in the same pass.
func optimizeNode(node ConditionNode) ConditionNode {
	switch n := node.(type) {
	case *AndNode:
		return optimizeAndNode(n)
	case *OrNode:
		return optimizeOrNode(n)
	case *NotNode:
		return optimizeNotNode(n)
	case *DisjunctionNode:
		return optimizeDisjunctionNode(n)
	default:
		// Leaf nodes (ExpressionNode, ComparisonNode, FunctionCallNode,
		// PropertyAccessNode, StringLiteralNode, BooleanLiteralNode) are returned
		// unchanged.
		return node
	}
}

// --- helper predicates -------------------------------------------------------

// isBoolLiteral returns true when node is a BooleanLiteralNode with the given value.
func isBoolLiteral(node ConditionNode, value bool) bool {
	lit, ok := node.(*BooleanLiteralNode)
	return ok && lit.Value == value
}

// isStatusFunc returns true when node is a call to one of the GitHub Actions
// status-check functions: always(), success(), failure(), cancelled().
// These functions change the execution status of a step/job and must not be
// removed from an expression by boolean-algebra rules.
func isStatusFunc(node ConditionNode) bool {
	fn, ok := node.(*FunctionCallNode)
	if !ok {
		return false
	}
	switch fn.FunctionName {
	case "always", "success", "failure", "cancelled":
		return true
	}
	return false
}

// nodesEqual returns true when a and b render to identical strings.
// This is used as a conservative structural-equality test: if two nodes
// render identically they are semantically equivalent in the expression.
func nodesEqual(a, b ConditionNode) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Render() == b.Render()
}

// isNegationOf returns true when b is the logical negation of a or a is the
// logical negation of b (handles both A / !A and !A / A cases).
func isNegationOf(a, b ConditionNode) bool {
	if notB, ok := b.(*NotNode); ok && nodesEqual(a, notB.Child) {
		return true
	}
	if notA, ok := a.(*NotNode); ok && nodesEqual(notA.Child, b) {
		return true
	}
	return false
}

// containsStatusFunc returns true when any node in the tree is a status function.
// Used to gate complement / idempotent rules that must not fire on expressions
// containing status functions.
func containsStatusFunc(node ConditionNode) bool {
	if isStatusFunc(node) {
		return true
	}
	switch n := node.(type) {
	case *AndNode:
		return containsStatusFunc(n.Left) || containsStatusFunc(n.Right)
	case *OrNode:
		return containsStatusFunc(n.Left) || containsStatusFunc(n.Right)
	case *NotNode:
		return containsStatusFunc(n.Child)
	case *DisjunctionNode:
		return slices.ContainsFunc(n.Terms, containsStatusFunc)
	case *FunctionCallNode:
		return slices.ContainsFunc(n.Arguments, containsStatusFunc)
	}
	return false
}

// --- node-specific optimisers ------------------------------------------------

func optimizeAndNode(n *AndNode) ConditionNode {
	// Bottom-up: optimise children first.
	left := optimizeNode(n.Left)
	right := optimizeNode(n.Right)

	// Annihilation: A && false → false
	if isBoolLiteral(left, false) || isBoolLiteral(right, false) {
		expressionOptimizerLog.Printf("AND annihilation: %s && %s → false", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: false}
	}

	// Identity: A && true → A  (guard: do not remove status functions)
	if isBoolLiteral(right, true) && !isStatusFunc(left) {
		expressionOptimizerLog.Printf("AND identity (right true): %s && true → %s", left.Render(), left.Render())
		return left
	}
	if isBoolLiteral(left, true) && !isStatusFunc(right) {
		expressionOptimizerLog.Printf("AND identity (left true): true && %s → %s", right.Render(), right.Render())
		return right
	}

	// Skip idempotent / complement rules when status functions are present.
	if containsStatusFunc(left) || containsStatusFunc(right) {
		return &AndNode{Left: left, Right: right}
	}

	// Idempotent: A && A → A
	if nodesEqual(left, right) {
		expressionOptimizerLog.Printf("AND idempotent: %s && %s → %s", left.Render(), right.Render(), left.Render())
		return left
	}

	// Complement: A && !A → false
	if isNegationOf(left, right) {
		expressionOptimizerLog.Printf("AND complement: %s && %s → false", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: false}
	}

	return &AndNode{Left: left, Right: right}
}

func optimizeOrNode(n *OrNode) ConditionNode {
	// Bottom-up: optimise children first.
	left := optimizeNode(n.Left)
	right := optimizeNode(n.Right)

	// Annihilation: A || true → true
	if isBoolLiteral(left, true) || isBoolLiteral(right, true) {
		expressionOptimizerLog.Printf("OR annihilation: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}

	// Identity: A || false → A
	if isBoolLiteral(right, false) {
		expressionOptimizerLog.Printf("OR identity (right false): %s || false → %s", left.Render(), left.Render())
		return left
	}
	if isBoolLiteral(left, false) {
		expressionOptimizerLog.Printf("OR identity (left false): false || %s → %s", right.Render(), right.Render())
		return right
	}

	// Skip idempotent / complement rules when status functions are present.
	if containsStatusFunc(left) || containsStatusFunc(right) {
		return &OrNode{Left: left, Right: right}
	}

	// Idempotent: A || A → A
	if nodesEqual(left, right) {
		expressionOptimizerLog.Printf("OR idempotent: %s || %s → %s", left.Render(), right.Render(), left.Render())
		return left
	}

	// Complement: A || !A → true
	if isNegationOf(left, right) {
		expressionOptimizerLog.Printf("OR complement: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}

	return &OrNode{Left: left, Right: right}
}

func optimizeNotNode(n *NotNode) ConditionNode {
	// Bottom-up: optimise child first.
	child := optimizeNode(n.Child)

	// Constant folding: !true → false, !false → true
	if lit, ok := child.(*BooleanLiteralNode); ok {
		expressionOptimizerLog.Printf("NOT constant folding: !%v → %v", lit.Value, !lit.Value)
		return &BooleanLiteralNode{Value: !lit.Value}
	}

	// Double negation: !!A → A
	if notChild, ok := child.(*NotNode); ok {
		expressionOptimizerLog.Printf("NOT double negation: !!%s → %s", notChild.Child.Render(), notChild.Child.Render())
		// Recurse so that the result of eliminating the double negation is
		// itself a candidate for further simplification.
		return optimizeNode(notChild.Child)
	}

	return &NotNode{Child: child}
}

func optimizeDisjunctionNode(n *DisjunctionNode) ConditionNode {
	if len(n.Terms) == 0 {
		return n
	}

	// Bottom-up: optimise each term first.
	optimised := make([]ConditionNode, 0, len(n.Terms))
	for _, term := range n.Terms {
		optimised = append(optimised, optimizeNode(term))
	}

	// Short-circuit: if any term is true the whole disjunction is true.
	for _, term := range optimised {
		if isBoolLiteral(term, true) {
			expressionOptimizerLog.Printf("Disjunction short-circuit on true")
			return &BooleanLiteralNode{Value: true}
		}
	}

	// Filter out false terms (identity: A || false → A).
	filtered := make([]ConditionNode, 0, len(optimised))
	for _, term := range optimised {
		if !isBoolLiteral(term, false) {
			filtered = append(filtered, term)
		}
	}
	if len(filtered) == 0 {
		expressionOptimizerLog.Printf("Disjunction all-false → false")
		return &BooleanLiteralNode{Value: false}
	}

	// Deduplicate terms by rendered form.
	seen := make(map[string]struct{}, len(filtered))
	deduped := make([]ConditionNode, 0, len(filtered))
	for _, term := range filtered {
		key := term.Render()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			deduped = append(deduped, term)
		} else {
			expressionOptimizerLog.Printf("Disjunction dedup: removing duplicate term %q", key)
		}
	}

	if len(deduped) == 1 {
		return deduped[0]
	}

	return &DisjunctionNode{Terms: deduped, Multiline: n.Multiline}
}
