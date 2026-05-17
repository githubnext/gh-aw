// Package rawloginlib implements a Go analysis linter that flags
// standard log package calls in library (pkg/) packages.
package rawloginlib

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
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

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		if strings.HasSuffix(filepath.Base(pass.Fset.Position(call.Pos()).Filename), "_test.go") {
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
			pass.Reportf(call.Pos(), "log.%s called in library package %s; use pkg/logger instead", sel.Sel.Name, pkgPath)
		}
	})

	return nil, nil
}
