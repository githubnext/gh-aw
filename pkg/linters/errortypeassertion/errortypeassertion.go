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
		// types.Universe always contains "error"; this branch indicates a broken
		// Go toolchain setup and should never be reached in practice.
		panic("errortypeassertion: types.Universe does not contain built-in error type")
	}
	builtinErrorType := builtinErrorObj.Type()

	insp.Preorder([]ast.Node{(*ast.TypeAssertExpr)(nil), (*ast.TypeSwitchStmt)(nil)}, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeAssertExpr:
			checkTypeAssertExpr(pass, noLintLinesByFile, builtinErrorType, node)
		case *ast.TypeSwitchStmt:
			checkTypeSwitchStmt(pass, noLintLinesByFile, builtinErrorType, node)
		}
	})

	return nil, nil
}

// checkTypeAssertExpr flags direct type assertions of the form err.(ConcreteType).
func checkTypeAssertExpr(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, builtinErrorType types.Type, typeAssert *ast.TypeAssertExpr) {
	// Type-switch guards have nil Type; skip them (handled by checkTypeSwitchStmt).
	if typeAssert.Type == nil {
		return
	}

	pos := pass.Fset.PositionFor(typeAssert.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) || nolint.HasDirective(pos, noLintLinesByFile) {
		return
	}

	// types.Identical matches only the exact built-in error type. Named interface
	// types that embed error (e.g. "type MyErr interface { error }") are
	// intentionally excluded: they carry additional methods that may justify a
	// direct assertion.
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
}

// checkTypeSwitchStmt flags concrete-type case arms in type switches on error,
// e.g. "case *os.PathError:" inside "switch err.(type)".
func checkTypeSwitchStmt(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, builtinErrorType types.Type, stmt *ast.TypeSwitchStmt) {
	x := typeSwitchX(stmt)
	if x == nil {
		return
	}

	// types.Identical matches only the exact built-in error type (see
	// checkTypeAssertExpr for rationale).
	assertedFrom := pass.TypesInfo.TypeOf(x)
	if assertedFrom == nil || !types.Identical(assertedFrom, builtinErrorType) {
		return
	}

	for _, clause := range stmt.Body.List {
		cc, ok := clause.(*ast.CaseClause)
		if !ok {
			continue
		}
		for _, typeExpr := range cc.List {
			pos := pass.Fset.PositionFor(typeExpr.Pos(), false)
			if filecheck.IsTestFile(pos.Filename) || nolint.HasDirective(pos, noLintLinesByFile) {
				continue
			}

			assertedTo := pass.TypesInfo.TypeOf(typeExpr)
			if assertedTo == nil {
				continue
			}
			if _, isInterface := assertedTo.Underlying().(*types.Interface); isInterface {
				continue
			}

			pass.ReportRangef(
				typeExpr,
				"type assertion on error to %s bypasses wrapped errors; use errors.As instead",
				assertedTo,
			)
		}
	}
}

// typeSwitchX returns the expression being switched on in a TypeSwitchStmt.
func typeSwitchX(stmt *ast.TypeSwitchStmt) ast.Expr {
	switch a := stmt.Assign.(type) {
	case *ast.AssignStmt:
		if len(a.Rhs) == 1 {
			if ta, ok := a.Rhs[0].(*ast.TypeAssertExpr); ok {
				return ta.X
			}
		}
	case *ast.ExprStmt:
		if ta, ok := a.X.(*ast.TypeAssertExpr); ok {
			return ta.X
		}
	}
	return nil
}
