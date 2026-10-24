// Package readerstringconv implements a Go analysis linter that flags
// strings.NewReader(string(b)) and bytes.NewReader([]byte(s)) calls that
// convert data only to build a reader over the other representation.
package readerstringconv

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the readerstringconv analysis pass.
var Analyzer = analyzerutil.New("readerstringconv", "reports strings.NewReader(string(b)) and bytes.NewReader([]byte(s)) calls that copy data only to pick a reader type", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyze(pass, n, generatedFiles, noLintIndex)
	})
}

func analyze(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return
	}
	sel, ok := astutil.UnwrapParenExpr(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "NewReader" {
		return
	}

	var pkg, want, msg string
	switch {
	case astutil.IsPkgSelector(pass, sel, "strings"):
		pkg, want = "strings", "[]byte"
		msg = "use bytes.NewReader(b) instead of strings.NewReader(string(b))"
	case astutil.IsPkgSelector(pass, sel, "bytes"):
		pkg, want = "bytes", "string"
		msg = "use strings.NewReader(s) instead of bytes.NewReader([]byte(s))"
	default:
		return
	}

	conv, ok := astutil.UnwrapParenExpr(call.Args[0]).(*ast.CallExpr)
	if !ok || len(conv.Args) != 1 {
		return
	}
	tv, ok := pass.TypesInfo.Types[conv.Fun]
	if !ok || !tv.IsType() {
		return
	}
	target := types.TypeString(tv.Type.Underlying(), nil)
	if pkg == "strings" && target != "string" || pkg == "bytes" && target != "[]byte" {
		return
	}
	argType := pass.TypesInfo.TypeOf(conv.Args[0])
	if argType == nil || types.TypeString(argType.Underlying(), nil) != want {
		return
	}

	pos := pass.Fset.PositionFor(call.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "readerstringconv") {
		return
	}
	pass.Reportf(call.Pos(), "%s", msg)
}
