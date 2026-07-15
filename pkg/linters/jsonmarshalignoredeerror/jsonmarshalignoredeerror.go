// Package jsonmarshalignoredeerror implements a Go analysis linter that flags
// json.Marshal and json.Unmarshal calls where the error return is discarded.
package jsonmarshalignoredeerror

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the json-marshal-ignored-error analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "jsonmarshalignoredeerror",
	Doc:      "reports json.Marshal and json.Unmarshal calls where the error return is discarded",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/jsonmarshalignoredeerror",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	nodeFilter := []ast.Node{(*ast.AssignStmt)(nil), (*ast.ExprStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			checkDiscardedJSONAssign(pass, stmt, noLintIndex)
		case *ast.ExprStmt:
			checkDiscardedJSONExpr(pass, stmt, noLintIndex)
		}
	})
	return nil, nil
}

func checkDiscardedJSONAssign(pass *analysis.Pass, assign *ast.AssignStmt, noLintIndex nolint.DirectiveIndex) {
	// Pattern: val, _ := json.Marshal(x)  — 2 lhs, 1 rhs, Lhs[1] is blank
	if len(assign.Lhs) == 2 && len(assign.Rhs) == 1 {
		blank, ok := assign.Lhs[1].(*ast.Ident)
		if ok && blank.Name == "_" {
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if ok && isJSONFunc(pass, call, "Marshal") {
				reportDiscardedJSONCall(pass, call, noLintIndex, "error return from json.Marshal is discarded; marshal failures produce nil bytes silently")
			}
		}
	}

	// Pattern: _ = json.Unmarshal(data, &v)  — 1 lhs, 1 rhs, Lhs[0] is blank
	if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
		blank, ok := assign.Lhs[0].(*ast.Ident)
		if ok && blank.Name == "_" {
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if ok && isJSONFunc(pass, call, "Unmarshal") {
				reportDiscardedJSONCall(pass, call, noLintIndex, "error return from json.Unmarshal is discarded; unmarshal failures leave the target value in a partial state")
			}
		}
	}
}

func checkDiscardedJSONExpr(pass *analysis.Pass, stmt *ast.ExprStmt, noLintIndex nolint.DirectiveIndex) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return
	}
	if isJSONFunc(pass, call, "Marshal") {
		reportDiscardedJSONCall(pass, call, noLintIndex, "error return from json.Marshal is discarded; marshal failures produce nil bytes silently")
		return
	}
	if isJSONFunc(pass, call, "Unmarshal") {
		reportDiscardedJSONCall(pass, call, noLintIndex, "error return from json.Unmarshal is discarded; unmarshal failures leave the target value in a partial state")
	}
}

func reportDiscardedJSONCall(pass *analysis.Pass, call *ast.CallExpr, noLintIndex nolint.DirectiveIndex, message string) {
	position := pass.Fset.PositionFor(call.Pos(), false)
	if nolint.HasDirectiveForLinter(position, noLintIndex, "jsonmarshalignoredeerror") {
		return
	}
	pass.ReportRangef(call, "%s", message)
}

func isJSONFunc(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != name {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "encoding/json"
}
