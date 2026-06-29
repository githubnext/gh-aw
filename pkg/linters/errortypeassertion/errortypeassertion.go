// Package errortypeassertion implements a Go analysis linter that flags type
// assertions on values typed as the built-in error interface when asserting to
// concrete types, and recommends errors.As for wrapped error traversal.
package errortypeassertion

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the error-type-assertion analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "errortypeassertion",
	Doc:      "reports type assertions from error to concrete types; use errors.As for wrapped errors",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/errortypeassertion",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "errortypeassertion")

	builtinErrorObj := types.Universe.Lookup("error")
	if builtinErrorObj == nil {
		return nil, nil
	}
	builtinErrorType := builtinErrorObj.Type()

	insp.Preorder([]ast.Node{(*ast.TypeAssertExpr)(nil)}, func(n ast.Node) {
		typeAssert, ok := n.(*ast.TypeAssertExpr)
		if !ok || typeAssert.Type == nil {
			return
		}

		pos := pass.Fset.PositionFor(typeAssert.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) || nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		assertedFrom := pass.TypesInfo.TypeOf(typeAssert.X)
		if assertedFrom == nil || !types.Identical(assertedFrom, builtinErrorType) {
			return
		}

		assertedTo := pass.TypesInfo.TypeOf(typeAssert.Type)
		if assertedTo == nil {
			return
		}
		if _, isInterface := assertedTo.Underlying().(*types.Interface); isInterface {
			return
		}

		pass.ReportRangef(
			typeAssert,
			"type assertion on error to %s bypasses wrapped errors; use errors.As instead",
			assertedTo,
		)
	})

	return nil, nil
}
