// Package trimleftright implements a Go analysis linter that flags calls to
// strings.TrimLeft or strings.TrimRight with a multi-character string literal
// cutset, where strings.TrimPrefix or strings.TrimSuffix is almost certainly
// the intended function.
//
// strings.TrimLeft(s, "foo") does NOT remove the prefix "foo"; it removes any
// leading rune that appears anywhere in the cutset characters 'f', 'o'.
// This is a well-known Go gotcha.
package trimleftright

import (
	"go/ast"
	"go/token"
	"strconv"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the trimleftright analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "trimleftright",
	Doc:      "reports likely mistaken strings.TrimLeft/TrimRight calls using repeated alphanumeric literal cutsets",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/trimleftright",
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

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "trimleftright") {
			return
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		funcName := sel.Sel.Name
		if funcName != "TrimLeft" && funcName != "TrimRight" {
			return
		}
		if !astutil.IsPkgSelector(pass, sel, "strings") {
			return
		}
		if len(call.Args) != 2 {
			return
		}

		// Only flag suspicious cutsets where a multi-rune alphanumeric literal
		// contains repeated runes (e.g., "foo"). This avoids flagging common,
		// intentional character-set trimming like whitespace classes.
		cutset, isCutset := stringLitValue(call.Args[1])
		if !isCutset || !looksSuspiciousCutset(cutset) {
			return
		}

		var suggested string
		switch funcName {
		case "TrimLeft":
			suggested = "TrimPrefix"
		case "TrimRight":
			suggested = "TrimSuffix"
		}

		pass.Report(analysis.Diagnostic{
			Pos: call.Pos(),
			End: call.End(),
			Message: "strings." + funcName + " with a multi-character cutset treats each character independently; " +
				"use strings." + suggested + " if you intend to remove the exact string",
		})
	})

	return nil, nil
}

// stringLitValue returns the unquoted string value of a string-literal AST node.
func stringLitValue(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// looksSuspiciousCutset reports likely TrimPrefix/TrimSuffix confusion.
// We require a multi-rune alphanumeric cutset with at least one repeated rune
// so valid character-set trimming (whitespace, punctuation, unique rune sets)
// is not flagged.
func looksSuspiciousCutset(cutset string) bool {
	runes := []rune(cutset)
	if len(runes) <= 1 {
		return false
	}

	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	seen := make(map[rune]struct{})
	for _, r := range runes {
		if _, ok := seen[r]; ok {
			return true
		}
		seen[r] = struct{}{}
	}

	return false
}
