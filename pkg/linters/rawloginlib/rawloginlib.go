// Package rawloginlib implements a Go analysis linter that flags
// standard log package calls in library (pkg/) packages.
package rawloginlib

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

var Analyzer = &analysis.Analyzer{
	Name:     "rawloginlib",
	Doc:      "reports use of the standard log package in library packages where pkg/logger should be used instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/rawloginlib",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// rawLogFuncs is the set of standard log functions that should not be called in library code.
var rawLogFuncs = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
	"Fatal": true, "Fatalf": true, "Fatalln": true,
	"Panic": true, "Panicf": true, "Panicln": true,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "rawloginlib")

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		if filecheck.IsTestFile(pass.Fset.Position(call.Pos()).Filename) {
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
		if ident.Name == "log" && rawLogFuncs[sel.Sel.Name] {
			position := pass.Fset.PositionFor(call.Pos(), false)
			if nolint.HasDirective(position, noLintLinesByFile) {
				return
			}
			pass.ReportRangef(call, "log.%s called in library package %s; use pkg/logger instead", sel.Sel.Name, pkgPath)
		}
	})

	return nil, nil
}
