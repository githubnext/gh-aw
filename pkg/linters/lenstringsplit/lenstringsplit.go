// Package lenstringsplit implements a Go analysis linter that flags
// len(strings.Split(s, sep)) expressions that allocate a []string just to
// count substrings. strings.Count(s, sep)+1 achieves the same result without
// the intermediate allocation.
package lenstringsplit

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the len-strings-split analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "lenstringsplit",
	Doc:      "reports len(strings.Split(s, sep)) expressions that allocate a []string just to count substrings; use strings.Count(s, sep)+1 instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/lenstringsplit",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		outer, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if !isBuiltinLen(outer) {
			return
		}

		if len(outer.Args) != 1 {
			return
		}
		inner, ok := outer.Args[0].(*ast.CallExpr)
		if !ok {
			return
		}
		if !isStringsSplit(pass, inner) {
			return
		}

		pos := pass.Fset.PositionFor(outer.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		pass.ReportRangef(outer,
			"len(strings.Split(...)) allocates a []string just to count substrings; use strings.Count(...)+1 instead",
		)
	})

	return nil, nil
}

// isBuiltinLen reports whether call is an invocation of the builtin len function.
func isBuiltinLen(call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	return ok && ident.Name == "len"
}

// isStringsSplit reports whether call is strings.Split from the standard
// library "strings" package.
func isStringsSplit(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Split" {
		return false
	}
	return astutil.IsPkgSelector(pass, sel, "strings")
}
