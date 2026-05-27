// Package ossetenvlibrary implements a Go analysis linter that flags
// os.Setenv and os.Unsetenv calls in non-main, non-test packages.
package ossetenvlibrary

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

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
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if strings.HasSuffix(pkgPath, ".test") || filecheck.IsTestFile(pass.Fset.PositionFor(call.Pos(), false).Filename) {
			return
		}

		fn, ok := calledOSFunc(pass, call)
		if !ok {
			return
		}
		switch fn.Name() {
		case "Setenv":
			pass.ReportRangef(call, "os.Setenv mutates the process environment; pass configuration explicitly instead")
		case "Unsetenv":
			pass.ReportRangef(call, "os.Unsetenv mutates the process environment; pass configuration explicitly instead")
		}
	})

	return nil, nil
}

func calledOSFunc(pass *analysis.Pass, call *ast.CallExpr) (*types.Func, bool) {
	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.Uses[fun.Sel]
	case *ast.Ident:
		obj = pass.TypesInfo.Uses[fun]
	default:
		return nil, false
	}

	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "os" {
		return nil, false
	}
	if fn.Name() != "Setenv" && fn.Name() != "Unsetenv" {
		return nil, false
	}
	return fn, true
}
