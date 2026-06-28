// Package osgetenvlibrary implements a Go analysis linter that flags
// os.Getenv and os.LookupEnv calls in non-main, non-test packages.
package osgetenvlibrary

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the os-getenv-in-library analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "osgetenvlibrary",
	Doc:      "reports calls to os.Getenv or os.LookupEnv in non-main, non-test packages",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/osgetenvlibrary",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "osgetenvlibrary")

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
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirective(position, noLintLinesByFile) {
			return
		}
		switch fn.Name() {
		case "Getenv":
			pass.ReportRangef(call, "os.Getenv couples the library to the process environment; pass configuration explicitly instead")
		case "LookupEnv":
			pass.ReportRangef(call, "os.LookupEnv couples the library to the process environment; pass configuration explicitly instead")
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
	if fn.Name() != "Getenv" && fn.Name() != "LookupEnv" {
		return nil, false
	}
	return fn, true
}
