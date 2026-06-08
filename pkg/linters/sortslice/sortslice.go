// Package sortslice implements a Go analysis linter that flags sort.Slice
// and sort.SliceStable calls that should use the type-safe slices.SortFunc
// or slices.SortStableFunc from the standard library slices package.
package sortslice

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the sort-slice analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sortslice",
	Doc:      "reports sort.Slice and sort.SliceStable calls that should use the type-safe slices.SortFunc or slices.SortStableFunc",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sortslice",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "sortslice")

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

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

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok || pkgIdent.Name != "sort" {
			return
		}

		switch sel.Sel.Name {
		case "Slice":
			pass.ReportRangef(call, "sort.Slice is not type-safe; use slices.SortFunc instead")
		case "SliceStable":
			pass.ReportRangef(call, "sort.SliceStable is not type-safe; use slices.SortStableFunc instead")
		}
	})

	return nil, nil
}
