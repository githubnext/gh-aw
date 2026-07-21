// Package stringscountcontains implements a Go analysis linter that flags
// strings.Count(s, sub) comparisons with 0 or 1 (e.g. > 0, >= 1, == 0,
// != 0, < 1, <= 0) and their yoda-order variants that should use the more
// readable strings.Contains(s, sub) or !strings.Contains(s, sub) instead.
package stringscountcontains

import (
	"fmt"
	"go/ast"
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

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		analyzeCountContains(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

// analyzeCountContains checks whether a binary expression is a strings.Count
// comparison with 0 or 1 that should use strings.Contains.
func analyzeCountContains(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	expr, ok := n.(*ast.BinaryExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(expr.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringscountcontains") {
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
	pkgText := astutil.CallQualifierText(pass.Fset, countCall)
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
		SuggestedFixes: astutil.BuildContainsFix(expr, pkgText, sText, subText, negated, "Replace strings.Count comparison with strings.Contains"),
	})
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

	countCall, ok := astutil.AsStringsMethodCall(pass, left, "Count")
	if !ok {
		return nil, false, false
	}

	op := expr.Op
	if flipped {
		op = astutil.FlipComparisonOp(op)
	}

	litVal, ok := astutil.ConstIntValue(pass, right)
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
	if _, ok := astutil.AsStringsMethodCall(pass, expr.X, "Count"); ok {
		return expr.X, expr.Y, false
	}
	return expr.Y, expr.X, true
}
