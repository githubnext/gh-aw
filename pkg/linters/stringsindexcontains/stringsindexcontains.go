// Package stringsindexcontains implements a Go analysis linter that flags
// strings.Index(s, substr) comparisons with -1 or 0 (e.g. != -1, >= 0, > -1,
// == -1, < 0, <= -1) and their yoda-order variants that should use the more
// readable strings.Contains(s, substr) or !strings.Contains(s, substr) instead.
package stringsindexcontains

import (
	"go/ast"
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the strings-index-contains analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringsindexcontains",
	Doc:      "reports strings.Index(s, substr) comparisons with -1 or 0 (e.g. != -1, >= 0, > -1, == -1, < 0, <= -1) and their yoda-order variants that should use strings.Contains(s, substr) or !strings.Contains(s, substr)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringsindexcontains",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "stringsindexcontains")

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		expr, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(expr.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// Match patterns:
		//   strings.Index(s, sub) != -1  → strings.Contains(s, sub)
		//   strings.Index(s, sub) >= 0   → strings.Contains(s, sub)
		//   strings.Index(s, sub) == -1  → !strings.Contains(s, sub)
		//   strings.Index(s, sub) < 0    → !strings.Contains(s, sub)
		// (and yoda variants: -1 != strings.Index(...), etc.)

		indexCall, negated, matched := matchIndexComparison(pass, expr)
		if !matched {
			return
		}

		if len(indexCall.Args) != 2 {
			return
		}

		sText := astutil.NodeText(pass.Fset, indexCall.Args[0])
		subText := astutil.NodeText(pass.Fset, indexCall.Args[1])
		pkgText := indexPkgText(pass, indexCall)
		if sText == "" || subText == "" || pkgText == "" {
			return
		}

		var msg string
		if negated {
			msg = "use !strings.Contains(" + sText + ", " + subText + ") instead of strings.Index comparison"
		} else {
			msg = "use strings.Contains(" + sText + ", " + subText + ") instead of strings.Index comparison"
		}

		fix := buildContainsFix(pass, expr, pkgText, sText, subText, negated)
		pass.Report(analysis.Diagnostic{
			Pos:            expr.Pos(),
			End:            expr.End(),
			Message:        msg,
			SuggestedFixes: fix,
		})
	})

	return nil, nil
}

// matchIndexComparison reports whether expr is a strings.Index comparison with -1 or 0.
// It returns the strings.Index call, whether the result is negated (i.e., checks for absence),
// and whether the pattern matched.
//
// Matched patterns (contains → negated=false):
//   - strings.Index(s, sub) != -1
//   - strings.Index(s, sub) >= 0
//   - -1 != strings.Index(s, sub)
//   - 0 <= strings.Index(s, sub)
//
// Matched patterns (not-contains → negated=true):
//   - strings.Index(s, sub) == -1
//   - strings.Index(s, sub) < 0
//   - -1 == strings.Index(s, sub)
//   - 0 > strings.Index(s, sub)
func matchIndexComparison(pass *analysis.Pass, expr *ast.BinaryExpr) (call *ast.CallExpr, negated bool, matched bool) {
	// Normalize so that the strings.Index call is on the left side.
	left, right, flipped := normalizeOperands(pass, expr)

	indexCall, ok := asStringsIndexCall(pass, left)
	if !ok {
		return nil, false, false
	}

	op := expr.Op
	if flipped {
		op = flipOp(op)
	}

	litVal, ok := constIntValue(pass, right)
	if !ok {
		return nil, false, false
	}

	// Check supported operator/literal combinations.
	switch op {
	case token.NEQ:
		// strings.Index(...) != -1  →  contains
		if litVal == -1 {
			return indexCall, false, true
		}
	case token.GEQ:
		// strings.Index(...) >= 0  →  contains
		if litVal == 0 {
			return indexCall, false, true
		}
	case token.GTR:
		// strings.Index(...) > -1  →  contains (less common but valid)
		if litVal == -1 {
			return indexCall, false, true
		}
	case token.EQL:
		// strings.Index(...) == -1  →  !contains
		if litVal == -1 {
			return indexCall, true, true
		}
	case token.LSS:
		// strings.Index(...) < 0  →  !contains
		if litVal == 0 {
			return indexCall, true, true
		}
	case token.LEQ:
		// strings.Index(...) <= -1  →  !contains (less common but valid)
		if litVal == -1 {
			return indexCall, true, true
		}
	}

	return nil, false, false
}

// normalizeOperands returns (left, right) such that if the strings.Index call
// is on the right side, the operands are swapped and flipped=true.
func normalizeOperands(pass *analysis.Pass, expr *ast.BinaryExpr) (left, right ast.Expr, flipped bool) {
	if _, ok := asStringsIndexCall(pass, expr.X); ok {
		return expr.X, expr.Y, false
	}
	return expr.Y, expr.X, true
}

// asStringsIndexCall returns the *ast.CallExpr if expr is a call to strings.Index.
func asStringsIndexCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Index" {
		return nil, false
	}
	if !astutil.IsPkgSelector(pass, sel, "strings") {
		return nil, false
	}
	return call, true
}

// constIntValue returns the integer constant value of expr, if it is a constant integer.
func constIntValue(pass *analysis.Pass, expr ast.Expr) (int64, bool) {
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.Int {
		return 0, false
	}
	v, exact := constant.Int64Val(tv.Value)
	return v, exact
}

// flipOp returns the comparison operator with left and right operands swapped.
func flipOp(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GTR
	case token.GTR:
		return token.LSS
	case token.LEQ:
		return token.GEQ
	case token.GEQ:
		return token.LEQ
	default:
		return op
	}
}

// indexPkgText returns the package selector text (e.g., "strings") from a strings.Index call.
func indexPkgText(pass *analysis.Pass, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return astutil.NodeText(pass.Fset, sel.X)
}

// buildContainsFix builds the suggested fix rewriting the comparison to strings.Contains.
func buildContainsFix(pass *analysis.Pass, expr *ast.BinaryExpr, pkgText, sText, subText string, negated bool) []analysis.SuggestedFix {
	var replacement string
	if negated {
		replacement = "!" + pkgText + ".Contains(" + sText + ", " + subText + ")"
	} else {
		replacement = pkgText + ".Contains(" + sText + ", " + subText + ")"
	}

	return []analysis.SuggestedFix{{
		Message: "Replace strings.Index comparison with strings.Contains",
		TextEdits: []analysis.TextEdit{{
			Pos:     expr.Pos(),
			End:     expr.End(),
			NewText: []byte(replacement),
		}},
	}}
}
