// Package appendoneelement implements a Go analysis linter that flags
// append(s, []T{x}...) calls where a single-element slice literal is
// spread, which can be simplified to append(s, x).
package appendoneelement

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the append-one-element analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "appendoneelement",
	Doc:      "reports append(s, []T{x}...) calls where a single-element slice literal is spread and can be simplified to append(s, x)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/appendoneelement",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Must be append(x, y...) with exactly 2 arguments and ellipsis.
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "append" {
			return
		}
		if pass.TypesInfo.ObjectOf(ident) != types.Universe.Lookup("append") {
			return
		}
		if len(call.Args) != 2 || !call.Ellipsis.IsValid() {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "appendoneelement") {
			return
		}

		// The second argument must be a composite literal with exactly one element.
		lit, ok := call.Args[1].(*ast.CompositeLit)
		if !ok {
			return
		}
		// Must be a slice type ([]T). *ast.ArrayType covers both slices and arrays;
		// only slices have Len == nil.
		arrayType, ok := lit.Type.(*ast.ArrayType)
		if !ok || arrayType.Len != nil {
			return
		}
		if len(lit.Elts) != 1 {
			return
		}

		elem := lit.Elts[0]
		if _, ok := elem.(*ast.KeyValueExpr); ok {
			return
		}
		if nestedLit, ok := elem.(*ast.CompositeLit); ok && nestedLit.Type == nil {
			return
		}
		elemText := astutil.NodeText(pass.Fset, elem)
		if elemText == "" {
			return
		}
		litText := astutil.NodeText(pass.Fset, lit)
		if litText == "" {
			return
		}

		sliceText := astutil.NodeText(pass.Fset, call.Args[0])
		if sliceText == "" {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: fmt.Sprintf("append(s, %s...) can be simplified to append(s, %s)", litText, elemText),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: fmt.Sprintf("Replace %s... with %s", litText, elemText),
				TextEdits: []analysis.TextEdit{
					{
						Pos:     call.Pos(),
						End:     call.End(),
						NewText: fmt.Appendf(nil, "append(%s, %s)", sliceText, elemText),
					},
				},
			}},
		})
	})

	return nil, nil
}
