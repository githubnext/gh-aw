// Package formatintitoa reports strconv.FormatInt(int64(x), 10) calls where x
// is already an int and recommends strconv.Itoa(x).
package formatintitoa

import (
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the formatintitoa analysis pass.
var Analyzer = analyzerutil.New("formatintitoa", "reports strconv.FormatInt(int64(x), 10) calls where x is an int; use strconv.Itoa(x) instead", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}
	return analyzerutil.Preorder(pass, []ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) ||
			nolint.HasDirectiveForLinter(pos, noLintIndex, "formatintitoa") {
			return
		}
		pkgPath, name, ok := astutil.PackageCall(pass, call)
		if !ok || pkgPath != "strconv" || name != "FormatInt" {
			return
		}
		if len(call.Args) != 2 {
			return
		}
		base := pass.TypesInfo.Types[call.Args[1]].Value //nolint:uncheckedsliceindex // len(call.Args) == 2 is checked above.
		if base == nil || base.Kind() != constant.Int {
			return
		}
		if value, exact := constant.Int64Val(base); !exact || value != 10 {
			return
		}
		conversion, ok := astutil.UnwrapParenExpr(call.Args[0]).(*ast.CallExpr) //nolint:uncheckedsliceindex // len(call.Args) == 2 is checked above.
		if !ok {
			return
		}
		if len(conversion.Args) != 1 {
			return
		}
		argument := conversion.Args[0] //nolint:uncheckedsliceindex // len(conversion.Args) == 1 is checked above.
		target := pass.TypesInfo.Types[conversion.Fun]
		if !target.IsType() || !types.Identical(target.Type, types.Universe.Lookup("int64").Type()) ||
			!types.Identical(pass.TypesInfo.TypeOf(argument), types.Universe.Lookup("int").Type()) {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        "use strconv.Itoa instead of strconv.FormatInt(int64(x), 10) for an int argument",
			SuggestedFixes: buildFix(pass, call, argument),
		})
	})
}

func buildFix(pass *analysis.Pass, call *ast.CallExpr, argument ast.Expr) []analysis.SuggestedFix {
	selector, ok := astutil.UnwrapParenExpr(call.Fun).(*ast.SelectorExpr)
	if !ok || !astutil.IsPkgSelector(pass, selector, "strconv") ||
		astutil.HasOverlappingComment(pass.Files, call.Pos(), call.End()) {
		return nil
	}
	pkgText := astutil.NodeText(pass.Fset, selector.X)
	argText := astutil.NodeText(pass.Fset, argument)
	if pkgText == "" || argText == "" {
		return nil
	}
	return []analysis.SuggestedFix{{
		Message: "Replace strconv.FormatInt with strconv.Itoa",
		TextEdits: []analysis.TextEdit{{
			Pos:     call.Pos(),
			End:     call.End(),
			NewText: []byte(pkgText + ".Itoa(" + argText + ")"),
		}},
	}}
}
