// Package fprintlnsprintf implements a Go analysis linter that flags
// fmt.Fprintln(w, fmt.Sprintf(...)) calls that should be rewritten as fmt.Fprintf(w, ...).
package fprintlnsprintf

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the fprintlnsprintf analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fprintlnsprintf",
	Doc:      "reports fmt.Fprintln(w, fmt.Sprintf(...)) calls that should be rewritten as fmt.Fprintf(w, ...)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/fprintlnsprintf",
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

		// Check if this is exactly fmt.Fprintln(w, fmt.Sprintf(...)).
		if !isFmtFunc(call, "Fprintln") {
			return
		}
		if len(call.Args) != 2 {
			return
		}

		// Skip test files.
		pos := pass.Fset.Position(call.Pos())
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		// Check if the printed argument is fmt.Sprintf(...).
		printedArg, ok := call.Args[1].(*ast.CallExpr)
		if !ok {
			return
		}
		if !isFmtFunc(printedArg, "Sprintf") {
			return
		}

		pass.Reportf(call.Pos(), "use fmt.Fprintf instead of fmt.Fprintln(w, fmt.Sprintf(...))")
	})

	return nil, nil
}

// isFmtFunc returns true if call is a call to fmt.<name>.
func isFmtFunc(call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "fmt" && sel.Sel.Name == name
}
