// Package ctxbackground implements a Go analysis linter that flags
// calls to context.Background() inside functions that already receive
// a context.Context parameter.
package ctxbackground

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the ctx-background analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "ctxbackground",
	Doc:      "reports calls to context.Background() inside functions that already receive a context.Context parameter",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/ctxbackground",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Check if any parameter is context.Context (and not blank).
		if !hasContextParam(pass, fn) {
			return
		}

		// Walk the function body for context.Background() calls.
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if ident.Name == "context" && sel.Sel.Name == "Background" {
				pass.Reportf(call.Pos(), "use the context.Context parameter instead of context.Background()")
			}
			return true
		})
	})

	return nil, nil
}

// hasContextParam returns true if fn has at least one non-blank parameter
// whose type is context.Context.
func hasContextParam(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	ctxType := contextType(pass)
	if ctxType == nil {
		return false
	}
	for _, field := range fn.Type.Params.List {
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil {
			continue
		}
		if !types.Identical(t, ctxType) {
			continue
		}
		// At least one name must not be blank.
		for _, name := range field.Names {
			if name.Name != "_" {
				return true
			}
		}
	}
	return false
}

// contextType returns the types.Type for context.Context, or nil if the
// package is not imported.
func contextType(pass *analysis.Pass) types.Type {
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
