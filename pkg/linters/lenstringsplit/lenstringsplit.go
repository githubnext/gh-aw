// Package lenstringsplit implements a Go analysis linter that flags
// len(strings.Split(s, sep)) expressions with a provably non-empty separator
// that allocate a []string just to count substrings. strings.Count(s, sep)+1
// achieves the same result for non-empty separators without the intermediate
// allocation.
package lenstringsplit

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the len-strings-split analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "lenstringsplit",
	Doc:      "reports len(strings.Split(s, sep)) expressions with a provably non-empty separator that allocate a []string just to count substrings; use strings.Count(s, sep)+1 instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/lenstringsplit",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "lenstringsplit")

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		outer, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if !isBuiltinLen(pass, outer) {
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
		if !hasProvablyNonEmptySeparator(pass, inner) {
			return
		}

		pos := pass.Fset.PositionFor(outer.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:            outer.Pos(),
			End:            outer.End(),
			Message:        "len(strings.Split(...)) allocates a []string just to count substrings; use strings.Count(...)+1 instead",
			SuggestedFixes: buildCountFix(pass, outer, inner),
		})
	})

	return nil, nil
}

// isBuiltinLen reports whether call is an invocation of the builtin len function.
func isBuiltinLen(pass *analysis.Pass, call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "len" {
		return false
	}
	obj := pass.TypesInfo.Uses[ident]
	return obj == nil || obj == types.Universe.Lookup("len")
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

func hasProvablyNonEmptySeparator(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 2 {
		return false
	}
	tv, ok := pass.TypesInfo.Types[call.Args[1]]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return false
	}
	return constant.StringVal(tv.Value) != ""
}

func buildCountFix(pass *analysis.Pass, outer, inner *ast.CallExpr) []analysis.SuggestedFix {
	if len(inner.Args) != 2 {
		return nil
	}

	sText := astutil.NodeText(pass.Fset, inner.Args[0])
	sepText := astutil.NodeText(pass.Fset, inner.Args[1])
	pkgText := splitPkgText(pass, inner)
	if sText == "" || sepText == "" || pkgText == "" {
		return nil
	}

	return []analysis.SuggestedFix{{
		Message: "Replace with strings.Count(...)+1",
		TextEdits: []analysis.TextEdit{{
			Pos:     outer.Pos(),
			End:     outer.End(),
			NewText: fmt.Appendf(nil, "%s.Count(%s, %s)+1", pkgText, sText, sepText),
		}},
	}}
}

func splitPkgText(pass *analysis.Pass, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return astutil.NodeText(pass.Fset, sel.X)
}
