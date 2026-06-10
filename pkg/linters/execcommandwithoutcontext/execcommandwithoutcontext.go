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

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
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
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := execCommandSelector(pass, call)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
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
				Message: fmt.Sprintf("use exec.CommandContext(%s, ...) instead of exec.Command to propagate context cancellation", ctxParamName),
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: fmt.Sprintf("Replace exec.Command with exec.CommandContext(%s, ...)", ctxParamName),
						TextEdits: []analysis.TextEdit{
							{
								Pos:     sel.Sel.Pos(),
								End:     sel.Sel.End(),
								NewText: []byte("CommandContext"),
							},
							{
								Pos:     call.Lparen + 1,
								End:     call.Lparen + 1,
								NewText: []byte(ctxParamName + ", "),
							},
						},
					},
				},
			})
			break
		}
	}

	return nil, nil
}

// execCommandSelector reports the selector expression for calls to
// exec.Command from os/exec.
func execCommandSelector(pass *analysis.Pass, call *ast.CallExpr) (*ast.SelectorExpr, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" {
		return nil, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return nil, false
	}
	if pkgName.Imported().Path() != "os/exec" {
		return nil, false
	}
	return sel, true
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
