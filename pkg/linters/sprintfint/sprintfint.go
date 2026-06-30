// Package sprintfint implements a Go analysis linter that flags
// fmt.Sprintf("%d", x) calls where x is a single int value and suggests
// using strconv.Itoa(x) instead.
package sprintfint

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the sprintfint analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sprintfint",
	Doc:      "reports fmt.Sprintf(\"%d\", x) calls where x is a single int value; use strconv.Itoa(x) instead (suggested fixes may require goimports to add/remove imports)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sprintfint",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "sprintfint")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// Match fmt.Sprintf(format, arg) with exactly two arguments.
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" {
			return
		}
		if !astutil.IsPkgSelector(pass, sel, "fmt") {
			return
		}
		if len(call.Args) != 2 {
			return
		}

		// The format argument must be the string literal "%d".
		formatLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || formatLit.Kind != token.STRING || formatLit.Value != `"%d"` {
			return
		}

		// The value argument must have the exact type int (not int64, uint, etc.).
		arg := call.Args[1]
		argType := pass.TypesInfo.TypeOf(arg)
		if argType == nil {
			return
		}
		if argType != types.Typ[types.Int] {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        `use strconv.Itoa(x) instead of fmt.Sprintf("%d", x)`,
			SuggestedFixes: buildItoaFix(pass, call, arg),
		})
	})

	return nil, nil
}

// buildItoaFix returns a SuggestedFix rewriting
// fmt.Sprintf("%d", x) → strconv.Itoa(x).
func buildItoaFix(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr) []analysis.SuggestedFix {
	argText := astutil.NodeText(pass.Fset, arg)
	if argText == "" {
		return nil
	}
	return []analysis.SuggestedFix{{
		Message: "Replace fmt.Sprintf with strconv.Itoa",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     call.Pos(),
				End:     call.End(),
				NewText: []byte("strconv.Itoa(" + argText + ")"),
			},
		},
	}}
}
