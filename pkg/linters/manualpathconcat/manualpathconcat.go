// Package manualpathconcat implements a Go analysis linter that flags string
// concatenation using a literal "/" separator to build filesystem paths
// (e.g. dir + "/" + file), which should use filepath.Join (or path.Join for
// slash-separated paths) instead.
//
// Manual "/" concatenation is error-prone: it can produce double slashes when
// an operand already ends with a separator, it skips the Clean-style
// normalization that filepath.Join performs, and it hard-codes the forward
// slash separator instead of the OS-specific one.
package manualpathconcat

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// linterName is the analyzer name, also used for nolint directive matching.
const linterName = "manualpathconcat"

// Analyzer is the manual-path-concat analysis pass.
var Analyzer = analyzerutil.New(linterName, `reports manual "/" separator string concatenation used to build paths (e.g. dir + "/" + file) that should use filepath.Join or path.Join`, run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	// reported tracks the sub-expressions of an already reported concatenation
	// chain so that `a + "/" + b + "/" + c` produces a single diagnostic.
	reported := make(map[ast.Expr]bool)

	nodeFilter := []ast.Node{(*ast.BinaryExpr)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeBinaryExpr(pass, n, generatedFiles, noLintIndex, reported)
	})
}

// analyzeBinaryExpr reports binary expressions of the shape X + "/" + Y.
func analyzeBinaryExpr(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex, reported map[ast.Expr]bool) {
	bin, ok := n.(*ast.BinaryExpr)
	if !ok || reported[bin] {
		return
	}
	left, ok := matchSlashSeparator(bin)
	if !ok {
		return
	}
	// A fully constant expression may appear in a const declaration, where a
	// filepath.Join call is not valid Go, so it is left alone.
	if tv, found := pass.TypesInfo.Types[bin]; found && tv.Value != nil {
		return
	}
	pos := pass.Fset.PositionFor(bin.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, linterName) {
		return
	}
	markChain(bin, reported)

	leftText := astutil.NodeText(pass.Fset, left)
	rightText := astutil.NodeText(pass.Fset, bin.Y)
	message := `manual "/" path concatenation; use filepath.Join (or path.Join) instead`
	if isShortOperandText(leftText) && isShortOperandText(rightText) && !containsSlashConcat(left) {
		message = fmt.Sprintf(`manual "/" path concatenation; use filepath.Join(%s, %s) (or path.Join) instead`, leftText, rightText)
	}
	pass.Report(analysis.Diagnostic{
		Pos:     bin.Pos(),
		End:     bin.End(),
		Message: message,
	})
}

// matchSlashSeparator reports whether bin has the shape X + "/" + Y, which Go
// parses as ((X + "/") + Y), and returns the X operand.
func matchSlashSeparator(bin *ast.BinaryExpr) (left ast.Expr, ok bool) {
	if bin.Op != token.ADD {
		return nil, false
	}
	inner, isBinary := bin.X.(*ast.BinaryExpr)
	if !isBinary || inner.Op != token.ADD {
		return nil, false
	}
	if !isSlashLiteral(inner.Y) {
		return nil, false
	}
	// A left operand that is itself the separator (e.g. `"/" + "/" + name`)
	// carries no path segment to join, so it is not a manual join.
	if isSlashLiteral(inner.X) {
		return nil, false
	}
	return inner.X, true
}

// isSlashLiteral reports whether expr is the string literal "/".
func isSlashLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	val, err := strconv.Unquote(lit.Value)
	return err == nil && val == "/"
}

// maxOperandTextLen bounds the operand source text embedded in a diagnostic
// message so that long or multi-line operands do not produce unreadable output.
const maxOperandTextLen = 48

// isShortOperandText reports whether text is a non-empty, single-line operand
// short enough to quote in a diagnostic message.
func isShortOperandText(text string) bool {
	return text != "" && len(text) <= maxOperandTextLen && !strings.ContainsAny(text, "\n\r")
}

// containsSlashConcat reports whether expr contains a string concatenation that
// includes a "/" literal operand (e.g. `a + "/" + b`).
func containsSlashConcat(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.ADD {
		return false
	}
	if isSlashLiteral(bin.X) || isSlashLiteral(bin.Y) {
		return true
	}
	return containsSlashConcat(bin.X) || containsSlashConcat(bin.Y)
}

// markChain records bin and every nested left-hand concatenation operand so
// that the sub-expressions of a reported chain are not reported again.
func markChain(bin *ast.BinaryExpr, reported map[ast.Expr]bool) {
	for expr := ast.Expr(bin); ; {
		inner, ok := expr.(*ast.BinaryExpr)
		if !ok {
			return
		}
		reported[inner] = true
		expr = inner.X
	}
}
