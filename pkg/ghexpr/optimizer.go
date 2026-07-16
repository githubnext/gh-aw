package ghexpr

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var optimizerLog = logger.New("ghexpr:optimizer")

// OptimizeExpression applies boolean-algebra simplifications to a ConditionNode tree,
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
// have semantics beyond plain booleans.  The optimizer never eliminates a status
// function call from an expression; it only applies rules when both operands of
// && / || are free of status functions.
//
// Execution is bounded: at most maxOptimizationPasses bottom-up passes are
// performed so the optimizer always terminates.
func OptimizeExpression(node ConditionNode) ConditionNode {
	if node == nil {
		return nil
	}

	const maxOptimizationPasses = 10

	current := node
	for pass := range maxOptimizationPasses {
		next := optimizeNode(current)
		if next.Render() == current.Render() {
			optimizerLog.Printf("Expression stabilised after %d pass(es)", pass+1)
			break
		}
		current = next
	}
	return current
}

// optimizeNode performs a single bottom-up optimisation pass.
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
		return node
	}
}

// --- helper predicates -------------------------------------------------------

func isBoolLiteral(node ConditionNode, value bool) bool {
	lit, ok := node.(*BooleanLiteralNode)
	return ok && lit.Value == value
}

// isStatusFunc returns true when node is a call to one of the GitHub Actions
// status-check functions: always(), success(), failure(), cancelled().
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
func nodesEqual(a, b ConditionNode) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Render() == b.Render()
}

// isNegationOf returns true when b is the logical negation of a or vice versa.
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

// collectOrTerms recursively flattens a chain of OrNode / DisjunctionNode into a flat slice.
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

// collectAndTerms recursively flattens a chain of AndNode into a flat slice.
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

// termSubsumedBy returns true when cand is subsumed by sub in a disjunction.
func termSubsumedBy(cand, sub ConditionNode) bool {
	if nodesEqual(cand, sub) {
		return false
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
		optimizerLog.Printf("AND annihilation: %s && %s → false", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: false}
	}

	terms := collectAndTerms(&AndNode{Left: left, Right: right})

	for _, t := range terms {
		if isBoolLiteral(t, false) {
			return &BooleanLiteralNode{Value: false}
		}
	}

	hasStatusFuncInTerms := slices.ContainsFunc(terms, containsStatusFunc)
	filtered := make([]ConditionNode, 0, len(terms))
	for _, t := range terms {
		if isBoolLiteral(t, true) && !hasStatusFuncInTerms {
			optimizerLog.Printf("AND identity (flatten): removed true literal")
			continue
		}
		filtered = append(filtered, t)
	}
	if len(filtered) == 0 {
		return &BooleanLiteralNode{Value: true}
	}

	seen := make(map[string]struct{}, len(filtered))
	deduped := make([]ConditionNode, 0, len(filtered))
	for _, t := range filtered {
		key := t.Render()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			deduped = append(deduped, t)
		} else {
			optimizerLog.Printf("AND dedup: removing duplicate term %q", key)
		}
	}
	if len(deduped) == 1 {
		return deduped[0]
	}

	if !hasStatusFuncInTerms {
		for i := range deduped {
			for j := i + 1; j < len(deduped); j++ {
				if isNegationOf(deduped[i], deduped[j]) {
					optimizerLog.Printf("AND complement (flatten): %s && %s → false", deduped[i].Render(), deduped[j].Render())
					return &BooleanLiteralNode{Value: false}
				}
			}
		}
	}

	if !hasStatusFuncInTerms {
		absorbed := make([]bool, len(deduped))
		for i, ti := range deduped {
			orTerms := collectOrTerms(ti)
			if len(orTerms) < 2 {
				continue
			}
			for j, tj := range deduped {
				if i == j || absorbed[i] {
					continue
				}
				for _, ot := range orTerms {
					if nodesEqual(ot, tj) {
						optimizerLog.Printf("AND absorption: (%s) && (%s) → %s (absorbed)", tj.Render(), ti.Render(), tj.Render())
						absorbed[i] = true
						break
					}
				}
			}
		}
		if slices.Contains(absorbed, true) {
			surviving := make([]ConditionNode, 0, len(deduped))
			for i, t := range deduped {
				if !absorbed[i] {
					surviving = append(surviving, t)
				}
			}
			if len(surviving) == 0 {
				return &BooleanLiteralNode{Value: true}
			}
			return optimizeNode(rebuildAndChain(surviving))
		}
	}

	return rebuildAndChain(deduped)
}

func optimizeOrNode(n *OrNode) ConditionNode {
	left := optimizeNode(n.Left)
	right := optimizeNode(n.Right)

	_, leftIsOr := left.(*OrNode)
	_, leftIsDisj := left.(*DisjunctionNode)
	_, rightIsOr := right.(*OrNode)
	_, rightIsDisj := right.(*DisjunctionNode)
	if leftIsOr || leftIsDisj || rightIsOr || rightIsDisj {
		terms := append(collectOrTerms(left), collectOrTerms(right)...)
		optimizerLog.Printf("OR flatten: collected %d terms", len(terms))
		return optimizeDisjunctionNode(&DisjunctionNode{Terms: terms})
	}

	if isBoolLiteral(left, true) || isBoolLiteral(right, true) {
		optimizerLog.Printf("OR annihilation: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}

	if isBoolLiteral(right, false) {
		return left
	}
	if isBoolLiteral(left, false) {
		return right
	}

	if containsStatusFunc(left) || containsStatusFunc(right) {
		return &OrNode{Left: left, Right: right}
	}

	if nodesEqual(left, right) {
		optimizerLog.Printf("OR idempotent: %s || %s → %s", left.Render(), right.Render(), left.Render())
		return left
	}

	if isNegationOf(left, right) {
		optimizerLog.Printf("OR complement: %s || %s → true", left.Render(), right.Render())
		return &BooleanLiteralNode{Value: true}
	}

	for _, pair := range [][2]ConditionNode{{left, right}, {right, left}} {
		simple, complex := pair[0], pair[1]
		if termSubsumedBy(complex, simple) {
			optimizerLog.Printf("OR absorption: %s || (%s) → %s (absorbed)", simple.Render(), complex.Render(), simple.Render())
			return simple
		}
	}

	return &OrNode{Left: left, Right: right}
}

func optimizeNotNode(n *NotNode) ConditionNode {
	child := optimizeNode(n.Child)

	if lit, ok := child.(*BooleanLiteralNode); ok {
		optimizerLog.Printf("NOT constant folding: !%v → %v", lit.Value, !lit.Value)
		return &BooleanLiteralNode{Value: !lit.Value}
	}

	if notChild, ok := child.(*NotNode); ok {
		optimizerLog.Printf("NOT double negation: !!%s → %s", notChild.Child.Render(), notChild.Child.Render())
		return optimizeNode(notChild.Child)
	}

	if andChild, ok := child.(*AndNode); ok && !containsStatusFunc(andChild) {
		optimizerLog.Printf("NOT De Morgan (AND): !(%s && %s) → !%s || !%s",
			andChild.Left.Render(), andChild.Right.Render(),
			andChild.Left.Render(), andChild.Right.Render())
		return optimizeNode(&OrNode{
			Left:  &NotNode{Child: andChild.Left},
			Right: &NotNode{Child: andChild.Right},
		})
	}

	if orChild, ok := child.(*OrNode); ok && !containsStatusFunc(orChild) {
		optimizerLog.Printf("NOT De Morgan (OR): !(%s || %s) → !%s && !%s",
			orChild.Left.Render(), orChild.Right.Render(),
			orChild.Left.Render(), orChild.Right.Render())
		return optimizeNode(&AndNode{
			Left:  &NotNode{Child: orChild.Left},
			Right: &NotNode{Child: orChild.Right},
		})
	}

	if disjChild, ok := child.(*DisjunctionNode); ok {
		if len(disjChild.Terms) == 0 {
			return &NotNode{Child: child}
		}
		if !containsStatusFunc(disjChild) {
			optimizerLog.Printf("NOT De Morgan (Disjunction): !(disjunction[%d]) → AND chain of negations", len(disjChild.Terms))
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

	optimised := make([]ConditionNode, 0, len(n.Terms))
	for _, term := range n.Terms {
		optimised = append(optimised, optimizeNode(term))
	}

	for _, term := range optimised {
		if isBoolLiteral(term, true) {
			optimizerLog.Printf("Disjunction short-circuit on true")
			return &BooleanLiteralNode{Value: true}
		}
	}

	filtered := make([]ConditionNode, 0, len(optimised))
	for _, term := range optimised {
		if !isBoolLiteral(term, false) {
			filtered = append(filtered, term)
		}
	}
	if len(filtered) == 0 {
		optimizerLog.Printf("Disjunction all-false → false")
		return &BooleanLiteralNode{Value: false}
	}

	seen := make(map[string]struct{}, len(filtered))
	deduped := make([]ConditionNode, 0, len(filtered))
	for _, term := range filtered {
		key := term.Render()
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			deduped = append(deduped, term)
		} else {
			optimizerLog.Printf("Disjunction dedup: removing duplicate term %q", key)
		}
	}

	if len(deduped) == 1 {
		return deduped[0]
	}

	if !slices.ContainsFunc(deduped, containsStatusFunc) {
		for i := range deduped {
			for j := i + 1; j < len(deduped); j++ {
				if isNegationOf(deduped[i], deduped[j]) {
					optimizerLog.Printf("Disjunction complement: %s || %s → true", deduped[i].Render(), deduped[j].Render())
					return &BooleanLiteralNode{Value: true}
				}
			}
		}

		subsumed := make([]bool, len(deduped))
		for i, cand := range deduped {
			for j, sub := range deduped {
				if i == j {
					continue
				}
				if termSubsumedBy(cand, sub) {
					optimizerLog.Printf("Disjunction subsumption: %s subsumed by %s", cand.Render(), sub.Render())
					subsumed[i] = true
					break
				}
			}
		}
		if slices.Contains(subsumed, true) {
			surviving := make([]ConditionNode, 0, len(deduped))
			for i, t := range deduped {
				if !subsumed[i] {
					surviving = append(surviving, t)
				}
			}
			if len(surviving) == 0 {
				return &BooleanLiteralNode{Value: false}
			}
			if len(surviving) == 1 {
				return surviving[0]
			}
			return optimizeNode(&DisjunctionNode{Terms: surviving, Multiline: n.Multiline})
		}
	}

	return &DisjunctionNode{Terms: deduped, Multiline: n.Multiline}
}
