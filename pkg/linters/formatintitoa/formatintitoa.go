// Package formatintitoa implements a Go analysis linter that flags
// strconv.FormatInt(int64(x), 10) calls where x is already an int and
// suggests using strconv.Itoa(x) instead.
package formatintitoa

import (
	"go/ast"
	"go/token"
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
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		analyze(pass, call, generatedFiles, noLintIndex)
	})
}

func analyze(pass *analysis.Pass, call *ast.CallExpr, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	pos := pass.Fset.PositionFor(call.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "formatintitoa") {
		return
	}
	sel, ok := astutil.UnwrapParenExpr(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "FormatInt" || !astutil.IsPkgSelector(pass, sel, "strconv") {
		return
	}
	if len(call.Args) != 2 || !isBase10(pass, call.Args[1]) {
		return
	}
	conv, ok := astutil.UnwrapParenExpr(call.Args[0]).(*ast.CallExpr)
	if !ok || len(conv.Args) != 1 {
		return
	}
	if t := pass.TypesInfo.TypeOf(conv.Fun); t != types.Typ[types.Int64] {
		return
	}
	if !isTypeConversion(pass, conv) {
		return
	}
	inner := conv.Args[0]
	if t := pass.TypesInfo.TypeOf(inner); t != types.Typ[types.Int] {
		return
	}

	argText := astutil.NodeText(pass.Fset, inner)
	if argText == "" {
		argText = "x"
	}
	diag := analysis.Diagnostic{
		Pos:     call.Pos(),
		End:     call.End(),
		Message: "use strconv.Itoa(" + argText + ") instead of strconv.FormatInt(int64(" + argText + "), 10)",
	}
	if !astutil.HasOverlappingComment(pass.Files, call.Pos(), call.End()) {
		diag.SuggestedFixes = []analysis.SuggestedFix{{
			Message: "Replace strconv.FormatInt with strconv.Itoa",
			TextEdits: []analysis.TextEdit{
				{Pos: sel.Sel.Pos(), End: sel.Sel.End(), NewText: []byte("Itoa")},
				{Pos: sel.End(), End: call.End(), NewText: []byte("(" + argText + ")")},
			},
		}}
	}
	pass.Report(diag)
}

func isBase10(pass *analysis.Pass, e ast.Expr) bool {
	lit, ok := astutil.UnwrapParenExpr(e).(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "10"
}

func isTypeConversion(pass *analysis.Pass, call *ast.CallExpr) bool {
	tv, ok := pass.TypesInfo.Types[call.Fun]
	return ok && tv.IsType()
}
