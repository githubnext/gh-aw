// Package lenstringzero implements a Go analysis linter that flags len(s) == 0
// and len(s) != 0 comparisons on string values that should use == "" or != "" instead.
package lenstringzero

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

var Analyzer = &analysis.Analyzer{
	Name:     "lenstringzero",
	Doc:      "reports len(s) == 0 and len(s) != 0 comparisons on string values that should use == \"\" or != \"\" instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/lenstringzero",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}

		pos := pass.Fset.PositionFor(expr.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		var lenArg ast.Expr
		if isLenCall(expr.X) && isIntZero(expr.Y) {
			lenArg = lenCallArg(expr.X)
		} else if isIntZero(expr.X) && isLenCall(expr.Y) {
			lenArg = lenCallArg(expr.Y)
		}
		if lenArg == nil {
			return
		}

		t := pass.TypesInfo.TypeOf(lenArg)
		if t == nil {
			return
		}
		basic, ok := t.Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.String {
			return
		}

		op := expr.Op.String()
		var cmpVerb string
		if expr.Op == token.EQL {
			cmpVerb = "empty"
		} else {
			cmpVerb = "non-empty"
		}
		pass.ReportRangef(expr,
			`use s %s "" to check for %s string instead of len(s) %s 0`,
			op, cmpVerb, op)
	})

	return nil, nil
}

func isLenCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "len"
}

func lenCallArg(expr ast.Expr) ast.Expr {
	return expr.(*ast.CallExpr).Args[0]
}

func isIntZero(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}
