// Package tolowerequalfold implements a Go analysis linter that flags
// case-insensitive string comparisons performed via strings.ToLower (or
// strings.ToUpper) combined with == that should instead use strings.EqualFold.
package tolowerequalfold

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the tolower-equalfold analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "tolowerequalfold",
	Doc:      "reports case-insensitive string comparisons using strings.ToLower/ToUpper that should use strings.EqualFold",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/tolowerequalfold",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}

		if filecheck.IsTestFile(pass.Fset.Position(expr.Pos()).Filename) {
			return
		}

		lowerLeft := isCaseConvCall(expr.X)
		lowerRight := isCaseConvCall(expr.Y)

		if lowerLeft || lowerRight {
			pass.ReportRangef(expr,
				"use strings.EqualFold for case-insensitive comparison instead of strings.ToLower/ToUpper with ==")
		}
	})

	return nil, nil
}

// isCaseConvCall reports whether node is a call to strings.ToLower or strings.ToUpper.
func isCaseConvCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "strings" &&
		(sel.Sel.Name == "ToLower" || sel.Sel.Name == "ToUpper")
}
