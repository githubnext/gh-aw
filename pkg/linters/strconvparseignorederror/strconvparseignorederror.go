// Package strconvparseignorederror implements a Go analysis linter that flags
// strconv parsing calls (Atoi, ParseInt, ParseFloat, ParseBool, ParseUint)
// where the error return is discarded with _.
package strconvparseignorederror

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the strconv-parse-ignored-error analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "strconvparseignorederror",
	Doc:      "reports strconv parsing calls where the error return is discarded with _",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/strconvparseignorederror",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// strconvParseFuncs is the set of strconv functions to check.
var strconvParseFuncs = map[string]bool{
	"Atoi":       true,
	"ParseInt":   true,
	"ParseFloat": true,
	"ParseBool":  true,
	"ParseUint":  true,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return
		}
		// We need exactly 2 LHS targets where the second is _
		if len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			return
		}
		// Check second LHS is blank identifier
		blank, ok := assign.Lhs[1].(*ast.Ident)
		if !ok || blank.Name != "_" {
			return
		}
		// Check RHS is a call to strconv.ParseXxx or strconv.Atoi
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if !strconvParseFuncs[sel.Sel.Name] {
			return
		}
		// Verify the receiver is the strconv package
		if ident, ok := sel.X.(*ast.Ident); ok {
			obj := pass.TypesInfo.Uses[ident]
			if pkgName, ok := obj.(*types.PkgName); ok {
				if pkgName.Imported().Path() == "strconv" {
					pass.ReportRangef(call, "error return from strconv.%s is discarded; parse failures produce zero values silently", sel.Sel.Name)
				}
			}
		}
	})

	return nil, nil
}
