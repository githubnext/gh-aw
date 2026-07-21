// Package stringsindexcontains implements a Go analysis linter that flags
// strings.Index(s, substr) comparisons with -1 or 0 (e.g. != -1, >= 0, > -1,
// == -1, < 0, <= -1) and their yoda-order variants that should use the more
// readable strings.Contains(s, substr) or !strings.Contains(s, substr) instead.
package stringsindexcontains

import (
	"go/ast"
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
		analyzeIndexContains(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

// analyzeIndexContains checks whether a binary expression is a strings.Index
// comparison with -1 or 0 that should use strings.Contains.
func analyzeIndexContains(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	expr, ok := n.(*ast.BinaryExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(expr.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsindexcontains") {
		return
	}
	indexCall, negated, matched := matchIndexComparison(pass, expr)
	if !matched {
		return
	}
	if len(indexCall.Args) != 2 {
		return
	}
	sText := astutil.NodeText(pass.Fset, indexCall.Args[0])
	subText := astutil.NodeText(pass.Fset, indexCall.Args[1])
	pkgText := astutil.CallQualifierText(pass.Fset, indexCall)
	if sText == "" || subText == "" || pkgText == "" {
		return
	}
	var msg string
	if negated {
		msg = "use !strings.Contains(" + sText + ", " + subText + ") instead of strings.Index comparison"
	} else {
		msg = "use strings.Contains(" + sText + ", " + subText + ") instead of strings.Index comparison"
	}
	fix := astutil.BuildContainsFix(expr, pkgText, sText, subText, negated, "Replace strings.Index comparison with strings.Contains")
	pass.Report(analysis.Diagnostic{
		Pos:            expr.Pos(),
		End:            expr.End(),
		Message:        msg,
		SuggestedFixes: fix,
	})
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

	indexCall, ok := astutil.AsStringsMethodCall(pass, left, "Index")
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
	if _, ok := astutil.AsStringsMethodCall(pass, expr.X, "Index"); ok {
		return expr.X, expr.Y, false
	}
	return expr.Y, expr.X, true
}
