// Package sprintfstring implements a Go analysis linter that flags
// fmt.Sprintf("%s", s) calls where s is a single string value; the call is a
// no-op and s can be used directly.
package sprintfstring

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

// Analyzer is the sprintfstring analysis pass.
var Analyzer = analyzerutil.New("sprintfstring", `reports fmt.Sprintf("%s", s) calls where s is a single string value; use s directly instead`, run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeSprintfString(pass, n, generatedFiles, noLintIndex)
	})
}

func analyzeSprintfString(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return
	}
	pos := pass.Fset.PositionFor(call.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "sprintfstring") {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Sprintf" || !astutil.IsPkgSelector(pass, sel, "fmt") {
		return
	}
	formatLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || formatLit.Kind != token.STRING || (formatLit.Value != `"%s"` && formatLit.Value != "`%s`") {
		return
	}
	arg := call.Args[1]
	if argType := pass.TypesInfo.TypeOf(arg); argType == nil || argType != types.Typ[types.String] {
		return
	}
	argText := astutil.NodeText(pass.Fset, arg)
	if argText == "" {
		argText = "s"
	}
	pass.Report(analysis.Diagnostic{
		Pos:     call.Pos(),
		End:     call.End(),
		Message: `use ` + argText + ` directly instead of fmt.Sprintf("%s", ` + argText + `)`,
	})
}
