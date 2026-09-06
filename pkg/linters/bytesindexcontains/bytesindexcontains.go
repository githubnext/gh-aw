// Package bytesindexcontains implements a Go analysis linter that flags
// bytes.Index(s, substr) comparisons with -1 or 0 (e.g. != -1, >= 0, > -1,
// == -1, < 0, <= -1) and their yoda-order variants that should use the more
// readable bytes.Contains(s, substr) or !bytes.Contains(s, substr) instead.
package bytesindexcontains

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the bytes-index-contains analysis pass.
var Analyzer = analyzerutil.New("bytesindexcontains", "reports bytes.Index(s, substr) comparisons with -1 or 0 (e.g. != -1, >= 0, > -1, == -1, < 0, <= -1) and their yoda-order variants that should use bytes.Contains(s, substr) or !bytes.Contains(s, substr)", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeIndexContains(pass, n, generatedFiles, noLintIndex)
	})
}

// analyzeIndexContains checks whether a binary expression is a bytes.Index
// comparison with -1 or 0 that should use bytes.Contains.
func analyzeIndexContains(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	expr, ok := n.(*ast.BinaryExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(expr.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "bytesindexcontains") {
		return
	}
	indexCall, negated, matched := matchIndexComparison(pass, expr)
	if !matched {
		return
	}
	if len(indexCall.Args) != 2 {
		return
	}
	if !astutil.IsByteSlice(pass, indexCall.Args[0]) || !astutil.IsByteSlice(pass, indexCall.Args[1]) {
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
		msg = "use !" + pkgText + ".Contains(" + sText + ", " + subText + ") instead of bytes.Index comparison"
	} else {
		msg = "use " + pkgText + ".Contains(" + sText + ", " + subText + ") instead of bytes.Index comparison"
	}
	fix := astutil.BuildContainsFix(pass.Files, expr, pkgText, sText, subText, negated, "Replace bytes.Index comparison with bytes.Contains")
	pass.Report(analysis.Diagnostic{
		Pos:            expr.Pos(),
		End:            expr.End(),
		Message:        msg,
		SuggestedFixes: fix,
	})
}

// matchIndexComparison reports whether expr is a bytes.Index comparison with -1 or 0.
// It returns the bytes.Index call, whether the result is negated (i.e., checks for absence),
// and whether the pattern matched.
//
// Matched patterns (contains → negated=false):
//   - bytes.Index(s, sub) != -1
//   - bytes.Index(s, sub) >= 0
//   - -1 != bytes.Index(s, sub)
//   - 0 <= bytes.Index(s, sub)
//
// Matched patterns (not-contains → negated=true):
//   - bytes.Index(s, sub) == -1
//   - bytes.Index(s, sub) < 0
//   - -1 == bytes.Index(s, sub)
//   - 0 > bytes.Index(s, sub)
func matchIndexComparison(pass *analysis.Pass, expr *ast.BinaryExpr) (call *ast.CallExpr, negated bool, matched bool) {
	left, right, flipped := normalizeComparisonOperands(pass, expr)
	indexCall, ok := asBytesIndexCall(pass, left)
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
	case token.NEQ:
		if litVal == -1 {
			return indexCall, false, true
		}
	case token.GEQ:
		if litVal == 0 {
			return indexCall, false, true
		}
	case token.GTR:
		if litVal == -1 {
			return indexCall, false, true
		}
	case token.EQL:
		if litVal == -1 {
			return indexCall, true, true
		}
	case token.LSS:
		if litVal == 0 {
			return indexCall, true, true
		}
	case token.LEQ:
		if litVal == -1 {
			return indexCall, true, true
		}
	}

	return nil, false, false
}

func normalizeComparisonOperands(pass *analysis.Pass, expr *ast.BinaryExpr) (left, right ast.Expr, flipped bool) {
	x := astutil.UnwrapParenExpr(expr.X)
	y := astutil.UnwrapParenExpr(expr.Y)
	if _, ok := asBytesIndexCall(pass, x); ok {
		return x, y, false
	}
	if _, ok := asBytesIndexCall(pass, y); ok {
		return y, x, true
	}
	return x, y, false
}

func asBytesIndexCall(pass *analysis.Pass, expr ast.Expr) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Index" {
		return nil, false
	}
	if !astutil.IsPkgSelector(pass, sel, "bytes") {
		return nil, false
	}
	return call, true
}
