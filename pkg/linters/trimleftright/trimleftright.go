// Package trimleftright implements a Go analysis linter that flags calls to
// strings.TrimLeft or strings.TrimRight with a multi-character string literal
// cutset, where strings.TrimPrefix or strings.TrimSuffix is almost certainly
// the intended function.
//
// strings.TrimLeft(s, "foo") does NOT remove the prefix "foo"; it removes any
// leading byte that appears anywhere in the cutset characters 'f', 'o'.
// This is a well-known Go gotcha.
package trimleftright

import (
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the trimleftright analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "trimleftright",
	Doc:      "reports strings.TrimLeft/TrimRight calls with a multi-character literal cutset that likely intend strings.TrimPrefix/TrimSuffix",
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

		// The cutset (second argument) must be a string literal with more than
		// one rune to indicate the caller likely confused TrimLeft with TrimPrefix.
		cutset, isCutset := stringLitValue(call.Args[1])
		if !isCutset || len([]rune(cutset)) <= 1 {
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
			SuggestedFixes: buildFix(pass, call, sel, suggested),
		})
	})

	return nil, nil
}

// buildFix returns a SuggestedFix that renames TrimLeft→TrimPrefix or
// TrimRight→TrimSuffix, preserving the existing package qualifier.
func buildFix(pass *analysis.Pass, call *ast.CallExpr, sel *ast.SelectorExpr, suggested string) []analysis.SuggestedFix {
	qualifier := astutil.NodeText(pass.Fset, sel.X)
	if qualifier == "" {
		return nil
	}
	arg0 := astutil.NodeText(pass.Fset, call.Args[0])
	arg1 := astutil.NodeText(pass.Fset, call.Args[1])
	if arg0 == "" || arg1 == "" {
		return nil
	}
	newText := qualifier + "." + suggested + "(" + arg0 + ", " + arg1 + ")"
	return []analysis.SuggestedFix{{
		Message: "Replace with " + qualifier + "." + suggested,
		TextEdits: []analysis.TextEdit{{
			Pos:     call.Pos(),
			End:     call.End(),
			NewText: []byte(newText),
		}},
	}}
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
