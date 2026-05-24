// Package ctxbackground implements a Go analysis linter that flags
// calls to context.Background() inside functions that already receive
// a context.Context parameter.
package ctxbackground

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
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
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok || !isContextBackgroundCall(call) {
			return true
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return true
		}

		for i := range slices.Backward(stack) {
			fn, ok := stack[i].(*ast.FuncDecl)
			if !ok {
				continue
			}

			ctxParamName, ok := contextParamName(pass, fn)
			if !ok {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: "use the context.Context parameter instead of context.Background()",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Replace context.Background() with context parameter",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     call.End(),
								NewText: []byte(ctxParamName),
							},
						},
					},
				},
			})
			return true
		}
		return true
	})

	return nil, nil
}

func isContextBackgroundCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "context" && sel.Sel.Name == "Background"
}

// contextParamName returns the first non-blank context.Context parameter name.
func contextParamName(pass *analysis.Pass, fn *ast.FuncDecl) (string, bool) {
	if fn.Type.Params == nil {
		return "", false
	}
	ctxType := contextType(pass)
	if ctxType == nil {
		return "", false
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
				return name.Name, true
			}
		}
	}
	return "", false
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
