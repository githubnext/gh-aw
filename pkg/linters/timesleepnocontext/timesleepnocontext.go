// Package timesleepnocontext implements a Go analysis linter that flags
// bare time.Sleep calls inside functions that already receive a
// context.Context parameter, where a context-aware select should be used
// to propagate cancellation.
package timesleepnocontext

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

// Analyzer is the time-sleep-no-context analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "timesleepnocontext",
	Doc:      "reports time.Sleep calls inside context-receiving functions where a context-aware select should be used to allow cancellation",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/timesleepnocontext",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "timesleepnocontext")

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok {
			continue
		}
		if !isTimeSleepCall(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}

		for encl := range cur.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
			funcNode := encl.Node()
			funcType := astutil.EnclosingFuncType(funcNode)
			if funcType == nil {
				continue
			}
			ctxParamName, hasCtx := astutil.ContextParamName(pass, funcType)
			if !hasCtx {
				if _, isFuncLit := funcNode.(*ast.FuncLit); isFuncLit && !astutil.IsGoOrDeferClosure(encl) {
					break
				}
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fmt.Sprintf("use select with %s.Done() instead of time.Sleep to allow context cancellation", ctxParamName),
			})
			break
		}
	}

	return nil, nil
}

// isTimeSleepCall reports whether call is a call to time.Sleep.
func isTimeSleepCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Sleep" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "time"
}
