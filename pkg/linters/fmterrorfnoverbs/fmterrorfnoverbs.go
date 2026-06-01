// Package fmterrorfnoverbs implements a Go analysis linter that flags calls to
// fmt.Errorf where the format string contains no format verbs, in which case
// errors.New is the idiomatic and cheaper alternative.
package fmterrorfnoverbs

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the fmterrorfnoverbs analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fmterrorfnoverbs",
	Doc:      "reports fmt.Errorf calls whose format string contains no verbs, preferring errors.New",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/fmterrorfnoverbs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
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

		if !isFmtErrorf(pass, call) {
			return
		}

		if len(call.Args) == 0 {
			return
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return
		}

		// Unquote the string value
		val := lit.Value
		if len(val) >= 2 {
			val = val[1 : len(val)-1]
		}

		if !strings.Contains(val, "%") {
			pass.ReportRangef(call, "fmt.Errorf called with no format verbs; use errors.New(%s) instead", lit.Value)
		}
	})

	return nil, nil
}

// isFmtErrorf returns true if the call expression is a call to fmt.Errorf.
func isFmtErrorf(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Errorf" {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(pkgIdent)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "fmt"
}
