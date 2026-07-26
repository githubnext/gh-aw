// Package stringsindexhasprefix implements a Go analysis linter that flags
// strings.Index(s, sub) comparisons with 0 (== 0 and != 0) and their yoda-order
// variants that should use the more readable strings.HasPrefix(s, sub) or
// !strings.HasPrefix(s, sub) instead.
package stringsindexhasprefix

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

// Analyzer is the strings-index-hasprefix analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringsindexhasprefix",
	Doc:      "reports strings.Index(s, sub) comparisons with 0 (== 0 and != 0) and their yoda-order variants that should use strings.HasPrefix(s, sub) or !strings.HasPrefix(s, sub)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringsindexhasprefix",
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
		analyzeIndexHasPrefix(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

// analyzeIndexHasPrefix checks whether a binary expression is a strings.Index
// comparison with 0 that should use strings.HasPrefix.
func analyzeIndexHasPrefix(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	expr, ok := n.(*ast.BinaryExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(expr.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringsindexhasprefix") {
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

	var replacement, msg string
	if negated {
		replacement = "!" + pkgText + ".HasPrefix(" + sText + ", " + subText + ")"
		msg = fmt.Sprintf("use !strings.HasPrefix(%s, %s) instead of strings.Index comparison", sText, subText)
	} else {
		replacement = pkgText + ".HasPrefix(" + sText + ", " + subText + ")"
		msg = fmt.Sprintf("use strings.HasPrefix(%s, %s) instead of strings.Index comparison", sText, subText)
	}

	pass.Report(analysis.Diagnostic{
		Pos:     expr.Pos(),
		End:     expr.End(),
		Message: msg,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Replace strings.Index comparison with strings.HasPrefix",
			TextEdits: []analysis.TextEdit{{
				Pos:     expr.Pos(),
				End:     expr.End(),
				NewText: []byte(replacement),
			}},
		}},
	})
}

// matchIndexComparison reports whether expr is a strings.Index comparison with 0.
// It returns the strings.Index call, whether the result is negated (i.e., !HasPrefix),
// and whether the pattern matched.
func matchIndexComparison(pass *analysis.Pass, expr *ast.BinaryExpr) (call *ast.CallExpr, negated bool, matched bool) {
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
	if !ok || litVal != 0 {
		return nil, false, false
	}

	switch op {
	case token.EQL:
		return indexCall, false, true
	case token.NEQ:
		return indexCall, true, true
	default:
		return nil, false, false
	}
}

// normalizeOperands returns (left, right) such that if the strings.Index call
// is on the right side, the operands are swapped and flipped=true.
// Both operands are unwrapped of any redundant parentheses before the check.
func normalizeOperands(pass *analysis.Pass, expr *ast.BinaryExpr) (left, right ast.Expr, flipped bool) {
	x := astutil.UnwrapParenExpr(expr.X)
	y := astutil.UnwrapParenExpr(expr.Y)
	if _, ok := astutil.AsStringsMethodCall(pass, x, "Index"); ok {
		return x, y, false
	}
	return y, x, true
}
