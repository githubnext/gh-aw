// Package timenowsub implements a Go analysis linter that flags
// time.Now().Sub(t) calls that can be simplified to time.Since(t).
package timenowsub

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the time-now-sub analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "timenowsub",
	Doc:      "reports time.Now().Sub(t) calls that should be simplified to time.Since(t)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/timenowsub",
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
		outer, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Match <expr>.Sub(<arg>) where <expr> is time.Now().
		sel, ok := outer.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sub" {
			return
		}
		if len(outer.Args) != 1 {
			return
		}

		// Verify the receiver is a call to time.Now().
		nowCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return
		}
		if !isTimeNow(pass, nowCall) {
			return
		}

		pos := pass.Fset.PositionFor(outer.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "timenowsub") {
			return
		}

		argText := astutil.NodeText(pass.Fset, outer.Args[0])
		if argText == "" {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:     outer.Pos(),
			End:     outer.End(),
			Message: fmt.Sprintf("time.Now().Sub(%s) can be simplified to time.Since(%s)", argText, argText),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: fmt.Sprintf("Replace time.Now().Sub(%s) with time.Since(%s)", argText, argText),
				TextEdits: []analysis.TextEdit{{
					Pos:     outer.Pos(),
					End:     outer.End(),
					NewText: []byte("time.Since(" + argText + ")"),
				}},
			}},
		})
	})

	return nil, nil
}

// isTimeNow reports whether call is a call to time.Now().
func isTimeNow(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Now" {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(sel.Sel)
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	return fn.FullName() == "time.Now"
}
