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

		if arg, ok := caseConvArg(expr.X); ok && sameOperand(pass, arg, expr.Y) {
			return
		}
		if arg, ok := caseConvArg(expr.Y); ok && sameOperand(pass, expr.X, arg) {
			return
		}

		if isCaseConvCall(expr.X) || isCaseConvCall(expr.Y) {
			pass.ReportRangef(expr,
				"use strings.EqualFold for case-insensitive comparison instead of strings.ToLower/ToUpper with ==")
		}
	})

	return nil, nil
}

// isCaseConvCall reports whether node is a call to strings.ToLower or strings.ToUpper.
func isCaseConvCall(n ast.Node) bool {
	_, ok := caseConvArg(n)
	return ok
}

// caseConvArg returns the argument when n is strings.ToLower/ToUpper(<arg>).
func caseConvArg(n ast.Node) (ast.Expr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	if len(call.Args) != 1 {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	if ident.Name != "strings" {
		return nil, false
	}
	if sel.Sel.Name != "ToLower" && sel.Sel.Name != "ToUpper" {
		return nil, false
	}
	return call.Args[0], true
}

func sameOperand(pass *analysis.Pass, left ast.Expr, right ast.Expr) bool {
	leftIdent, leftOK := left.(*ast.Ident)
	rightIdent, rightOK := right.(*ast.Ident)
	if !leftOK || !rightOK {
		return false
	}

	leftObj := pass.TypesInfo.ObjectOf(leftIdent)
	rightObj := pass.TypesInfo.ObjectOf(rightIdent)
	return leftObj != nil && rightObj != nil && leftObj == rightObj
}
