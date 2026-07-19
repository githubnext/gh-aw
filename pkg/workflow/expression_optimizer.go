package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var expressionOptimizerLog = logger.New("workflow:expression_optimizer")

// OptimizeExpression applies boolean algebra simplifications to a ConditionNode tree,
// returning an equivalent but potentially simpler and shorter expression.
//
// Rules applied (bottom-up, fixpoint iteration):
//
//	Constant folding:      !true → false, !false → true
//	Double negation:       !!A → A
//	Boolean identity:      A && true → A,  A || false → A
//	Boolean annihilation:  A && false → false, A || true → true
//	Idempotent law:        A && A → A,  A || A → A
//	Complement law:        A && !A → false, A || !A → true
//	De Morgan (AND):       !(A && B) → !A || !B
//	De Morgan (OR):        !(A || B) → !A && !B
//	Absorption (AND):      A && (A || B) → A
//	Absorption (OR):       A || (A && B) → A
//	Subsumption (disj):    disj(A, A&&B, …) → disj(A, …)  [A&&B subsumed by A]
//	DisjunctionNode:       deduplication, false-filtering, true short-circuit
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

// collectOrTerms recursively flattens a chain of OrNode / DisjunctionNode into
// a flat slice of leaf terms. This allows the DisjunctionNode optimiser (which
// already performs dedup, false-filtering and true short-circuit) to operate on
// the entire or-chain in a single pass.
func collectOrTerms(node ConditionNode) []ConditionNode {
	switch n := node.(type) {
	case *OrNode:
		return append(collectOrTerms(n.Left), collectOrTerms(n.Right)...)
	case *DisjunctionNode:
		terms := make([]ConditionNode, 0, len(n.Terms))
		for _, t := range n.Terms {
			terms = append(terms, collectOrTerms(t)...)
		}
		return terms
	}
	return []ConditionNode{node}
}

// collectAndTerms recursively flattens a chain of AndNode into a flat slice of
// leaf terms so that cross-chain deduplication can be performed in a single pass.
func collectAndTerms(node ConditionNode) []ConditionNode {
	if and, ok := node.(*AndNode); ok {
		return append(collectAndTerms(and.Left), collectAndTerms(and.Right)...)
	}
	return []ConditionNode{node}
}

// rebuildAndChain assembles a left-folded AndNode chain from a non-empty slice.
func rebuildAndChain(terms []ConditionNode) ConditionNode {
	if len(terms) == 1 {
		return terms[0]
	}
	result := ConditionNode(&AndNode{Left: terms[0], Right: terms[1]})
	for _, t := range terms[2:] {
		result = &AndNode{Left: result, Right: t}
	}
	return result
}

// termSubsumedBy returns true when cand is subsumed by sub, meaning sub |= cand
// (cand is "more specific"). In a disjunction this makes cand redundant:
// disj(sub, cand) = sub. Only applies when neither term contains a status func.
//
// Example: sub=A, cand=A&&B → every model satisfying A&&B also satisfies A,
// so A already covers A&&B in a disjunction.
func termSubsumedBy(cand, sub ConditionNode) bool {
	if nodesEqual(cand, sub) {
		return false // identical terms are handled by dedup, not subsumption
	}
	if containsStatusFunc(cand) || containsStatusFunc(sub) {
		return false
	}
	for _, ct := range collectAndTerms(cand) {
		if nodesEqual(ct, sub) {
			return true
		}
	}
	return false
}

// --- node-specific optimisers ------------------------------------------------

func optimizeAndNode(n *AndNode) ConditionNode {
	left := optimizeNode(n.Left)
	right := optimizeNode(n.Right)
	if isBoolLiteral(left, false) || isBoolLiteral(right, false) {
		expressionOptimizerLog.Printf("AND annihilation: %s && %s → false", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: false}
	}

	terms := collectAndTerms(&AndNode{Left: left, Right: right})
	for _, t := range terms {
		if isBoolLiteral(t, false) {
			expressionOptimizerLog.Printf("AND annihilation (flatten): false term → false")
			return &BooleanLiteralNode{Value: false}
		}
	}

	hasStatusFuncInTerms := slices.ContainsFunc(terms, containsStatusFunc)
	filtered := filterAndIdentityTerms(terms, hasStatusFuncInTerms)
	if len(filtered) == 0 {
		return &BooleanLiteralNode{Value: true}
	}
	deduped := dedupeRenderedTerms(filtered, "AND")
	if len(deduped) == 1 {
		return deduped[0]
	}
	if !hasStatusFuncInTerms && hasComplementTerms(deduped, "AND complement (flatten)") {
		return &BooleanLiteralNode{Value: false}
	}
	if !hasStatusFuncInTerms {
		if absorbed := optimizeAndAbsorption(deduped); absorbed != nil {
			return absorbed
		}
	}
	return rebuildAndChain(deduped)
}

func filterAndIdentityTerms(terms []ConditionNode, hasStatusFuncInTerms bool) []ConditionNode {
	filtered := make([]ConditionNode, 0, len(terms))
	for _, t := range terms {
		if isBoolLiteral(t, true) && !hasStatusFuncInTerms {
			expressionOptimizerLog.Printf("AND identity (flatten): removed true literal")
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

func dedupeRenderedTerms(terms []ConditionNode, label string) []ConditionNode {
	seen := make(map[string]struct{}, len(terms))
	deduped := make([]ConditionNode, 0, len(terms))
	for _, term := range terms {
		key := term.Render()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			deduped = append(deduped, term)
		} else {
			expressionOptimizerLog.Printf("%s dedup: removing duplicate term %q", label, key)
		}
	}
	return deduped
}

func hasComplementTerms(terms []ConditionNode, label string) bool {
	for i := range terms {
		for j := i + 1; j < len(terms); j++ {
			if isNegationOf(terms[i], terms[j]) {
				expressionOptimizerLog.Printf("%s: %s && %s → false", label, terms[i].Render(), terms[j].Render())
				return true
			}
		}
	}
	return false
}

func optimizeAndAbsorption(deduped []ConditionNode) ConditionNode {
	absorbed := make([]bool, len(deduped))
	for i, ti := range deduped {
		orTerms := collectOrTerms(ti)
		if len(orTerms) < 2 {
			continue
		}
		markAndAbsorbedTerm(absorbed, deduped, orTerms, i, ti)
	}
	if !slices.Contains(absorbed, true) {
		return nil
	}
	surviving := survivingTerms(deduped, absorbed)
	if len(surviving) == 0 {
		return &BooleanLiteralNode{Value: true}
	}
	return optimizeNode(rebuildAndChain(surviving))
}

func markAndAbsorbedTerm(absorbed []bool, deduped, orTerms []ConditionNode, i int, ti ConditionNode) {
	for j, tj := range deduped {
		if i == j || absorbed[i] {
			continue
		}
		for _, ot := range orTerms {
			if nodesEqual(ot, tj) {
				expressionOptimizerLog.Printf("AND absorption: (%s) && (%s) → %s (absorbed)", tj.Render(), ti.Render(), tj.Render())
				absorbed[i] = true
				break
			}
		}
	}
}

func optimizeOrNode(n *OrNode) ConditionNode {
	left := optimizeNode(n.Left)
	right := optimizeNode(n.Right)
	if flattened := optimizeFlattenedOr(left, right); flattened != nil {
		return flattened
	}
	if isBoolLiteral(left, true) || isBoolLiteral(right, true) {
		expressionOptimizerLog.Printf("OR annihilation: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}
	if identity := optimizeOrIdentity(left, right); identity != nil {
		return identity
	}
	if containsStatusFunc(left) || containsStatusFunc(right) {
		return &OrNode{Left: left, Right: right}
	}
	if nodesEqual(left, right) {
		expressionOptimizerLog.Printf("OR idempotent: %s || %s → %s", left.Render(), right.Render(), left.Render())
		return left
	}
	if isNegationOf(left, right) {
		expressionOptimizerLog.Printf("OR complement: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}
	if absorbed := optimizeOrAbsorption(left, right); absorbed != nil {
		return absorbed
	}
	return &OrNode{Left: left, Right: right}
}

func optimizeFlattenedOr(left, right ConditionNode) ConditionNode {
	_, leftIsOr := left.(*OrNode)
	_, leftIsDisj := left.(*DisjunctionNode)
	_, rightIsOr := right.(*OrNode)
	_, rightIsDisj := right.(*DisjunctionNode)
	if !leftIsOr && !leftIsDisj && !rightIsOr && !rightIsDisj {
		return nil
	}
	terms := append(collectOrTerms(left), collectOrTerms(right)...)
	expressionOptimizerLog.Printf("OR flatten: collected %d terms", len(terms))
	return optimizeDisjunctionNode(&DisjunctionNode{Terms: terms})
}

func optimizeOrIdentity(left, right ConditionNode) ConditionNode {
	if isBoolLiteral(right, false) {
		expressionOptimizerLog.Printf("OR identity (right false): %s || false → %s", left.Render(), left.Render())
		return left
	}
	if isBoolLiteral(left, false) {
		expressionOptimizerLog.Printf("OR identity (left false): false || %s → %s", right.Render(), right.Render())
		return right
	}
	return nil
}

func optimizeOrAbsorption(left, right ConditionNode) ConditionNode {
	for _, pair := range [][2]ConditionNode{{left, right}, {right, left}} {
		simple, complex := pair[0], pair[1]
		if termSubsumedBy(complex, simple) {
			expressionOptimizerLog.Printf("OR absorption: %s || (%s) → %s (absorbed)", simple.Render(), complex.Render(), simple.Render())
			return simple
		}
	}
	return nil
}

func optimizeNotNode(n *NotNode) ConditionNode {
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

	// De Morgan: !(A && B) → !A || !B
	// Only applied when neither operand contains a status function, since
	// rearranging status functions changes execution semantics.
	if andChild, ok := child.(*AndNode); ok && !containsStatusFunc(andChild) {
		expressionOptimizerLog.Printf("NOT De Morgan (AND): !(%s && %s) → !%s || !%s",
			andChild.Left.Render(), andChild.Right.Render(),
			andChild.Left.Render(), andChild.Right.Render())
		return optimizeNode(&OrNode{
			Left:  &NotNode{Child: andChild.Left},
			Right: &NotNode{Child: andChild.Right},
		})
	}

	// De Morgan: !(A || B) → !A && !B
	if orChild, ok := child.(*OrNode); ok && !containsStatusFunc(orChild) {
		expressionOptimizerLog.Printf("NOT De Morgan (OR): !(%s || %s) → !%s && !%s",
			orChild.Left.Render(), orChild.Right.Render(),
			orChild.Left.Render(), orChild.Right.Render())
		return optimizeNode(&AndNode{
			Left:  &NotNode{Child: orChild.Left},
			Right: &NotNode{Child: orChild.Right},
		})
	}

	// De Morgan: !(A || B || ...) → !A && !B && ... (DisjunctionNode form)
	// Move the empty-terms guard before the containsStatusFunc call to avoid
	// an unnecessary tree walk when the disjunction is empty.
	if disjChild, ok := child.(*DisjunctionNode); ok {
		if len(disjChild.Terms) == 0 {
			return &NotNode{Child: child}
		}
		if !containsStatusFunc(disjChild) {
			expressionOptimizerLog.Printf("NOT De Morgan (Disjunction): !(disjunction[%d]) → AND chain of negations", len(disjChild.Terms))
			negations := make([]ConditionNode, len(disjChild.Terms))
			for i, term := range disjChild.Terms {
				negations[i] = &NotNode{Child: term}
			}
			return optimizeNode(rebuildAndChain(negations))
		}
	}

	return &NotNode{Child: child}
}

func optimizeDisjunctionNode(n *DisjunctionNode) ConditionNode {
	if len(n.Terms) == 0 {
		return n
	}
	optimised := optimizeDisjunctionTerms(n.Terms)
	if hasTrueDisjunctionTerm(optimised) {
		return &BooleanLiteralNode{Value: true}
	}
	filtered := filterFalseDisjunctionTerms(optimised)
	if len(filtered) == 0 {
		expressionOptimizerLog.Printf("Disjunction all-false → false")
		return &BooleanLiteralNode{Value: false}
	}
	deduped := dedupeRenderedTerms(filtered, "Disjunction")
	if len(deduped) == 1 {
		return deduped[0]
	}
	if !slices.ContainsFunc(deduped, containsStatusFunc) {
		if hasDisjunctionComplement(deduped) {
			return &BooleanLiteralNode{Value: true}
		}
		if absorbed := optimizeDisjunctionSubsumption(deduped, n.Multiline); absorbed != nil {
			return absorbed
		}
	}
	return &DisjunctionNode{Terms: deduped, Multiline: n.Multiline}
}

func optimizeDisjunctionTerms(terms []ConditionNode) []ConditionNode {
	optimised := make([]ConditionNode, 0, len(terms))
	for _, term := range terms {
		optimised = append(optimised, optimizeNode(term))
	}
	return optimised
}

func hasTrueDisjunctionTerm(terms []ConditionNode) bool {
	for _, term := range terms {
		if isBoolLiteral(term, true) {
			expressionOptimizerLog.Printf("Disjunction short-circuit on true")
			return true
		}
	}
	return false
}

func filterFalseDisjunctionTerms(terms []ConditionNode) []ConditionNode {
	filtered := make([]ConditionNode, 0, len(terms))
	for _, term := range terms {
		if !isBoolLiteral(term, false) {
			filtered = append(filtered, term)
		}
	}
	return filtered
}

func hasDisjunctionComplement(terms []ConditionNode) bool {
	for i := range terms {
		for j := i + 1; j < len(terms); j++ {
			if isNegationOf(terms[i], terms[j]) {
				expressionOptimizerLog.Printf("Disjunction complement: %s || %s → true", terms[i].Render(), terms[j].Render())
				return true
			}
		}
	}
	return false
}

func optimizeDisjunctionSubsumption(deduped []ConditionNode, multiline bool) ConditionNode {
	subsumed := markDisjunctionSubsumedTerms(deduped)
	if !slices.Contains(subsumed, true) {
		return nil
	}
	surviving := survivingTerms(deduped, subsumed)
	if len(surviving) == 0 {
		return &BooleanLiteralNode{Value: false}
	}
	if len(surviving) == 1 {
		return surviving[0]
	}
	return optimizeNode(&DisjunctionNode{Terms: surviving, Multiline: multiline})
}

func markDisjunctionSubsumedTerms(deduped []ConditionNode) []bool {
	subsumed := make([]bool, len(deduped))
	for i, cand := range deduped {
		for j, sub := range deduped {
			if i == j {
				continue
			}
			if termSubsumedBy(cand, sub) {
				expressionOptimizerLog.Printf("Disjunction subsumption: %s subsumed by %s", cand.Render(), sub.Render())
				subsumed[i] = true
				break
			}
		}
	}
	return subsumed
}

func survivingTerms(terms []ConditionNode, removed []bool) []ConditionNode {
	surviving := make([]ConditionNode, 0, len(terms))
	for i, term := range terms {
		if !removed[i] {
			surviving = append(surviving, term)
		}
	}
	return surviving
}
