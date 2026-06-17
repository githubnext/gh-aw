// Package uncheckedtypeassertion implements a Go analysis linter that flags
// single-value type assertions x.(T) that may panic at runtime if the dynamic
// type does not match, and where the two-value safe form x.(T) is not used.
package uncheckedtypeassertion

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the unchecked-type-assertion analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "uncheckedtypeassertion",
	Doc:      "reports single-value type assertions that may panic if the dynamic type does not match",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/uncheckedtypeassertion",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "uncheckedtypeassertion")

	// Build a parent map for each file so we can detect the two-value form.
	fileParents := make(map[*ast.File]map[ast.Node]ast.Node)
	for _, f := range pass.Files {
		fileParents[f] = buildParentMap(f)
	}

	nodeFilter := []ast.Node{
		(*ast.TypeAssertExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		inspectTypeAssertExpr(pass, noLintLinesByFile, fileParents, n)
	})

	return nil, nil
}

func inspectTypeAssertExpr(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, fileParents map[*ast.File]map[ast.Node]ast.Node, n ast.Node) {
	typeAssert, ok := n.(*ast.TypeAssertExpr)
	if !ok {
		return
	}

	// Type-switch guards have nil Type; skip them.
	if typeAssert.Type == nil {
		return
	}

	pos := pass.Fset.PositionFor(typeAssert.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) {
		return
	}

	// Find the parent map for the file containing this node.
	var parents map[ast.Node]ast.Node
	for _, f := range pass.Files {
		if f.Pos() <= typeAssert.Pos() && typeAssert.Pos() <= f.End() {
			parents = fileParents[f]
			break
		}
	}

	// Skip the safe two-value form:  v, ok := x.(T)  or  v, ok = x.(T)
	if parents != nil {
		if isSafeTwoValueAssertion(typeAssert, parents) {
			return
		}
	}

	t := pass.TypesInfo.TypeOf(typeAssert.Type)
	if t == nil {
		return
	}
	if nolint.HasDirective(pos, noLintLinesByFile) {
		return
	}

	pass.ReportRangef(
		typeAssert,
		"type assertion x.(%s) is unchecked and may panic; use the two-value form v, ok := x.(%s) instead",
		t, t,
	)
}

func isSafeTwoValueAssertion(typeAssert *ast.TypeAssertExpr, parents map[ast.Node]ast.Node) bool {
	parent := parents[typeAssert]
	for parent != nil {
		paren, ok := parent.(*ast.ParenExpr)
		if !ok {
			break
		}
		parent = parents[paren]
	}

	switch p := parent.(type) {
	case *ast.AssignStmt:
		return len(p.Lhs) == 2 && len(p.Rhs) == 1
	case *ast.ValueSpec:
		return len(p.Names) == 2 && len(p.Values) == 1
	default:
		return false
	}
}

// buildParentMap constructs a map from each AST node to its direct parent node.
func buildParentMap(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node

	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})

	return parents
}
