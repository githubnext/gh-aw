// Package stringscountcontains implements a Go analysis linter that flags
// strings.Count(s, sub) comparisons with 0 or 1 (e.g. > 0, >= 1, == 0,
// != 0, < 1, <= 0) and their yoda-order variants that should use the more
// readable strings.Contains(s, sub) or !strings.Contains(s, sub) instead.
package stringscountcontains

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the strings-count-contains analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringscountcontains",
	Doc:      "reports strings.Count(s, sub) comparisons with 0 or 1 (e.g. > 0, >= 1, == 0, != 0, < 1, <= 0) and their yoda-order variants that should use strings.Contains(s, sub) or !strings.Contains(s, sub)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringscountcontains",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "stringscountcontains")

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

		countCall, negated, matched := matchCountComparison(pass, expr)
		if !matched {
			return
		}

		if len(countCall.Args) != 2 {
			return
		}

		sText := astutil.NodeText(pass.Fset, countCall.Args[0])
		subText := astutil.NodeText(pass.Fset, countCall.Args[1])
		pkgText := countPkgText(pass, countCall)
		if sText == "" || subText == "" || pkgText == "" {
			return
		}

		var msg string
		if negated {
			msg = fmt.Sprintf("use !strings.Contains(%s, %s) instead of strings.Count comparison", sText, subText)
		} else {
			msg = fmt.Sprintf("use strings.Contains(%s, %s) instead of strings.Count comparison", sText, subText)
		}

		pass.Report(analysis.Diagnostic{
			Pos:            expr.Pos(),
			End:            expr.End(),
			Message:        msg,
			SuggestedFixes: buildContainsFix(pass, expr, pkgText, sText, subText, negated),
		})
	})

	return nil, nil
}

// matchCountComparison reports whether expr is a strings.Count comparison with
// 0 or 1 that can be replaced with strings.Contains.
//
// Matched patterns (contains → negated=false):
//   - strings.Count(s, sub) > 0
//   - strings.Count(s, sub) >= 1
//   - strings.Count(s, sub) != 0
//   - 0 < strings.Count(s, sub)
//   - 1 <= strings.Count(s, sub)
//   - 0 != strings.Count(s, sub)
//
// Matched patterns (not-contains → negated=true):
//   - strings.Count(s, sub) == 0
//   - strings.Count(s, sub) < 1
//   - strings.Count(s, sub) <= 0
//   - 0 == strings.Count(s, sub)
//   - 1 > strings.Count(s, sub)
func matchCountComparison(pass *analysis.Pass, expr *ast.BinaryExpr) (call *ast.CallExpr, negated bool, matched bool) {
	// Normalize so the strings.Count call is on the left side.
	left, right, flipped := normalizeOperands(pass, expr)

	countCall, ok := asStringsCountCall(pass, left)
	if !ok {
		return nil, false, false
	}

	op := expr.Op
	if flipped {
		op = astutil.FlipComparisonOp(op)
	}

	litVal, ok := constIntValue(pass, right)
	if !ok {
		return nil, false, false
	}

	switch op {
	case token.GTR:
		// strings.Count(...) > 0  →  contains
		if litVal == 0 {
			return countCall, false, true
		}
	case token.GEQ:
		// strings.Count(...) >= 1  →  contains
		if litVal == 1 {
			return countCall, false, true
		}
	case token.NEQ:
		// strings.Count(...) != 0  →  contains
		if litVal == 0 {
			return countCall, false, true
		}
	case token.EQL:
		// strings.Count(...) == 0  →  !contains
		if litVal == 0 {
			return countCall, true, true
		}
	case token.LSS:
		// strings.Count(...) < 1  →  !contains
		if litVal == 1 {
			return countCall, true, true
		}
	case token.LEQ:
		// strings.Count(...) <= 0  →  !contains
		if litVal == 0 {
			return countCall, true, true
		}
	}

	return nil, false, false
}

// normalizeOperands returns (left, right) such that if the strings.Count call
// is on the right side, the operands are swapped and flipped=true.
func normalizeOperands(pass *analysis.Pass, expr *ast.BinaryExpr) (left, right ast.Expr, flipped bool) {
	if _, ok := asStringsCountCall(pass, expr.X); ok {
		return expr.X, expr.Y, false
	}
	return expr.Y, expr.X, true
}

// asStringsCountCall returns the *ast.CallExpr if expr is a call to strings.Count.
func asStringsCountCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Count" {
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

// countPkgText returns the package selector text (e.g., "strings") from a strings.Count call.
func countPkgText(pass *analysis.Pass, call *ast.CallExpr) string {
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
		Message: "Replace strings.Count comparison with strings.Contains",
		TextEdits: []analysis.TextEdit{{
			Pos:     expr.Pos(),
			End:     expr.End(),
			NewText: []byte(replacement),
		}},
	}}
}
