// Package sprintferrorsnew implements a Go analysis linter that flags
// errors.New(fmt.Sprintf(...)) calls that should use fmt.Errorf instead.
package sprintferrorsnew

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the sprintferrorsnew analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sprintferrorsnew",
	Doc:      "reports errors.New(fmt.Sprintf(...)) calls that should use fmt.Errorf instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sprintferrorsnew",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "sprintferrorsnew")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if filecheck.IsTestFile(pass.Fset.Position(call.Pos()).Filename) {
			return
		}

		// Match errors.New(...)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "New" {
			return
		}
		if !astutil.IsPkgSelector(pass, sel, "errors") {
			return
		}
		if len(call.Args) != 1 {
			return
		}

		// Check if the sole argument is a direct fmt.Sprintf(...) call.
		argCall, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			return
		}
		argSel, ok := argCall.Fun.(*ast.SelectorExpr)
		if !ok || argSel.Sel.Name != "Sprintf" {
			return
		}
		if !astutil.IsPkgSelector(pass, argSel, "fmt") {
			return
		}

		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirective(position, noLintLinesByFile) {
			return
		}
		pass.Reportf(call.Pos(), "use fmt.Errorf instead of errors.New(fmt.Sprintf(...))")
	})

	return nil, nil
}
