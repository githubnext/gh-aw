// Package jsonmarshalignoredeerror implements a Go analysis linter that flags
// json.Marshal and json.Unmarshal calls where the error return is discarded with _.
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
	Doc:      "reports json.Marshal and json.Unmarshal calls where the error return is discarded with _",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/jsonmarshalignoredeerror",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "jsonmarshalignoredeerror")
	nodeFilter := []ast.Node{(*ast.AssignStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return
		}

		// Pattern: val, _ := json.Marshal(x)  — 2 lhs, 1 rhs, Lhs[1] is blank
		if len(assign.Lhs) == 2 && len(assign.Rhs) == 1 {
			blank, ok := assign.Lhs[1].(*ast.Ident)
			if ok && blank.Name == "_" {
				call, ok := assign.Rhs[0].(*ast.CallExpr)
				if ok {
					if isJSONFunc(pass, call, "Marshal") {
						position := pass.Fset.PositionFor(call.Pos(), false)
						if nolint.HasDirective(position, noLintLinesByFile) {
							return
						}
						pass.ReportRangef(call, "error return from json.Marshal is discarded; marshal failures produce nil bytes silently")
					}
				}
			}
		}

		// Pattern: _ = json.Unmarshal(data, &v)  — 1 lhs, 1 rhs, Lhs[0] is blank
		if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
			blank, ok := assign.Lhs[0].(*ast.Ident)
			if ok && blank.Name == "_" {
				call, ok := assign.Rhs[0].(*ast.CallExpr)
				if ok {
					if isJSONFunc(pass, call, "Unmarshal") {
						position := pass.Fset.PositionFor(call.Pos(), false)
						if nolint.HasDirective(position, noLintLinesByFile) {
							return
						}
						pass.ReportRangef(call, "error return from json.Unmarshal is discarded; unmarshal failures leave the target value in a partial state")
					}
				}
			}
		}
	})
	return nil, nil
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
