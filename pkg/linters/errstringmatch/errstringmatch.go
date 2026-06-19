// Package errstringmatch implements a Go analysis linter that flags calls to
// strings.Contains/HasPrefix/HasSuffix/EqualFold/Index/LastIndex/Compare on
// err.Error() with a string literal — all perform brittle substring matching on
// error messages instead of using errors.Is or errors.As.
package errstringmatch

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the err-string-match analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "errstringmatch",
	Doc:      "reports strings.Contains/HasPrefix/HasSuffix/EqualFold/Index/LastIndex/Compare(err.Error(), \"...\") calls that perform brittle substring matching on error messages",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/errstringmatch",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// brittleErrStringFuncs is the set of strings package functions that perform
// brittle error-message matching when their first argument is err.Error().
var brittleErrStringFuncs = map[string]bool{
	"Contains":  true,
	"HasPrefix": true,
	"HasSuffix": true,
	"EqualFold": true,
	"Index":     true,
	"LastIndex": true,
	"Compare":   true,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "errstringmatch")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		outer, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		position := pass.Fset.PositionFor(outer.Pos(), false)
		if filecheck.IsTestFile(position.Filename) {
			return
		}

		// Match strings.<BrittleFunc>(X, Y)
		funcName, matched := brittleErrStringFuncName(outer)
		if !matched {
			return
		}
		if len(outer.Args) != 2 {
			return
		}

		// First arg must be a call to err.Error()
		if !isErrDotError(pass, outer.Args[0]) {
			return
		}

		// Second arg must be a string literal (or at least a string type)
		if !isStringLiteral(pass, outer.Args[1]) {
			return
		}
		if nolint.HasDirective(position, noLintLinesByFile) {
			return
		}

		pass.ReportRangef(outer, "avoid strings.%s(err.Error(), ...) — use errors.Is, errors.As, or a sentinel error instead", funcName)
	})

	return nil, nil
}

// brittleErrStringFuncName returns the matched strings function name and true
// when call is a strings.<BrittleFunc>(...) call expression.
func brittleErrStringFuncName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	if ident.Name != "strings" {
		return "", false
	}
	if brittleErrStringFuncs[sel.Sel.Name] {
		return sel.Sel.Name, true
	}
	return "", false
}

// isErrDotError returns true when expr is a method call of the form <expr>.Error()
// where the receiver implements the error interface.
func isErrDotError(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Error" {
		return false
	}
	if len(call.Args) != 0 {
		return false
	}
	// Check that the receiver implements the error interface.
	t := pass.TypesInfo.TypeOf(sel.X)
	if t == nil {
		return false
	}
	return nolint.ImplementsError(t)
}

// isStringLiteral returns true when expr is a string literal or untyped string constant.
func isStringLiteral(pass *analysis.Pass, expr ast.Expr) bool {
	if astutil.IsStringLiteral(expr) {
		return true
	}
	// Also accept typed/untyped string constants (e.g. a const identifier).
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}
