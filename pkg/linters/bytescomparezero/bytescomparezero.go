// Package bytescomparezero implements a Go analysis linter that flags
// bytes.Compare(a, b) == 0 and bytes.Compare(a, b) != 0 comparisons (including
// yoda-order variants) that should use bytes.Equal(a, b) or !bytes.Equal(a, b).
package bytescomparezero

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

// Analyzer is the bytes-compare-zero analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "bytescomparezero",
	Doc:      "reports bytes.Compare(a, b) == 0 / != 0 (and yoda variants) that should use bytes.Equal(a, b) or !bytes.Equal(a, b)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/bytescomparezero",
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
		analyzeCompareZero(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

func analyzeCompareZero(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	expr, ok := n.(*ast.BinaryExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(expr.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "bytescomparezero") {
		return
	}

	compareCall, negated, matched := matchCompareZero(pass, expr)
	if !matched {
		return
	}
	if len(compareCall.Args) != 2 {
		return
	}

	aText := astutil.NodeText(pass.Fset, compareCall.Args[0])
	bText := astutil.NodeText(pass.Fset, compareCall.Args[1])
	pkgText := astutil.CallQualifierText(pass.Fset, compareCall)
	if aText == "" || bText == "" || pkgText == "" {
		return
	}

	var replacement, msg string
	if negated {
		replacement = "!" + pkgText + ".Equal(" + aText + ", " + bText + ")"
		msg = fmt.Sprintf("use !bytes.Equal(%s, %s) instead of bytes.Compare comparison with 0", aText, bText)
	} else {
		replacement = pkgText + ".Equal(" + aText + ", " + bText + ")"
		msg = fmt.Sprintf("use bytes.Equal(%s, %s) instead of bytes.Compare comparison with 0", aText, bText)
	}

	pass.Report(analysis.Diagnostic{
		Pos:     expr.Pos(),
		End:     expr.End(),
		Message: msg,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Replace bytes.Compare equality check with bytes.Equal",
			TextEdits: []analysis.TextEdit{{
				Pos:     expr.Pos(),
				End:     expr.End(),
				NewText: []byte(replacement),
			}},
		}},
	})
}

func matchCompareZero(pass *analysis.Pass, expr *ast.BinaryExpr) (call *ast.CallExpr, negated bool, matched bool) {
	left, right, flipped := normalizeOperands(pass, expr)
	compareCall, ok := asBytesCompareCall(pass, left)
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
		return compareCall, false, true
	case token.NEQ:
		return compareCall, true, true
	default:
		return nil, false, false
	}
}

func normalizeOperands(pass *analysis.Pass, expr *ast.BinaryExpr) (left, right ast.Expr, flipped bool) {
	x := astutil.UnwrapParenExpr(expr.X)
	y := astutil.UnwrapParenExpr(expr.Y)
	if _, ok := asBytesCompareCall(pass, x); ok {
		return x, y, false
	}
	return y, x, true
}

func asBytesCompareCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Compare" {
		return nil, false
	}
	if !astutil.IsPkgSelector(pass, sel, "bytes") {
		return nil, false
	}
	return call, true
}
