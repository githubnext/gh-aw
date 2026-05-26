// Package ossetenvlibrary implements a Go analysis linter that flags
// os.Setenv and os.Unsetenv calls in non-main, non-test packages.
package ossetenvlibrary

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the os-setenv-in-library analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "ossetenvlibrary",
	Doc:      "reports calls to os.Setenv or os.Unsetenv in non-main, non-test packages",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/ossetenvlibrary",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() == "main" {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		if filecheck.IsTestFile(pass.Fset.PositionFor(call.Pos(), false).Filename) {
			return
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		if ident.Name != "os" {
			return
		}
		switch sel.Sel.Name {
		case "Setenv":
			pass.ReportRangef(call, "os.Setenv mutates the process environment; pass configuration explicitly instead")
		case "Unsetenv":
			pass.ReportRangef(call, "os.Unsetenv mutates the process environment; pass configuration explicitly instead")
		}
	})

	return nil, nil
}
