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
			funcType := enclosingFuncType(encl.Node())
			if funcType == nil {
				continue
			}
			ctxParamName, hasCtx := contextParamName(pass, funcType)
			if !hasCtx {
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

func enclosingFuncType(node ast.Node) *ast.FuncType {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		return fn.Type
	case *ast.FuncLit:
		return fn.Type
	default:
		return nil
	}
}

// contextParamName returns the name of the first context.Context parameter
// in fn, and true, or "", false if none exists.
func contextParamName(pass *analysis.Pass, fn *ast.FuncType) (string, bool) {
	if fn == nil || fn.Params == nil {
		return "", false
	}
	ctxType := contextContextType(pass)
	if ctxType == nil {
		return "", false
	}
	for _, field := range fn.Params.List {
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil || !types.Identical(t, ctxType) {
			continue
		}
		for _, name := range field.Names {
			if name.Name != "_" {
				return name.Name, true
			}
		}
	}
	return "", false
}

// contextContextType returns the types.Type for context.Context, or nil if
// the context package is not imported.
func contextContextType(pass *analysis.Pass) types.Type {
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "context" {
			obj := pkg.Scope().Lookup("Context")
			if obj != nil {
				return obj.Type()
			}
		}
	}
	return nil
}
