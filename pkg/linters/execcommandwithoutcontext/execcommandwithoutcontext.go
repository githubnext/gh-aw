// Package execcommandwithoutcontext implements a Go analysis linter that flags
// calls to exec.Command inside functions that already receive a context.Context
// parameter, where exec.CommandContext should be used instead to propagate
// cancellation.
package execcommandwithoutcontext

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the exec-command-without-context analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "execcommandwithoutcontext",
	Doc:      "reports exec.Command calls inside context-receiving functions where exec.CommandContext should be used to propagate cancellation",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/execcommandwithoutcontext",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !isExecCommandCall(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}

		for encl := range cur.Enclosing((*ast.FuncDecl)(nil)) {
			fn, ok := encl.Node().(*ast.FuncDecl)
			if !ok {
				continue
			}
			ctxParamName, hasCtx := contextParamName(pass, fn)
			if !hasCtx {
				break
			}
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fmt.Sprintf("use exec.CommandContext(%s, ...) instead of exec.Command to propagate context cancellation", ctxParamName),
			})
			break
		}
	}

	return nil, nil
}

// isExecCommandCall reports whether call is a call to exec.Command from os/exec.
func isExecCommandCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" {
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
	return pkgName.Imported().Path() == "os/exec"
}

// contextParamName returns the name of the first context.Context parameter
// in fn, and true, or "", false if none exists.
func contextParamName(pass *analysis.Pass, fn *ast.FuncDecl) (string, bool) {
	if fn.Type.Params == nil {
		return "", false
	}
	ctxType := contextContextType(pass)
	if ctxType == nil {
		return "", false
	}
	for _, field := range fn.Type.Params.List {
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
