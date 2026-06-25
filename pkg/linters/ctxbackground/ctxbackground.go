// Package ctxbackground implements a Go analysis linter that flags
// calls to context.Background() inside functions that already receive
// a context.Context parameter.
package ctxbackground

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
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
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "ctxbackground")

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !isContextBackgroundCall(pass, call) {
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
			var ftype *ast.FuncType
			switch fn := encl.Node().(type) {
			case *ast.FuncDecl:
				ftype = fn.Type
			case *ast.FuncLit:
				ftype = fn.Type
			default:
				continue
			}
			ctxParamName, ok := contextParamName(pass, ftype)
			if !ok {
				break
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
			break
		}
	}

	return nil, nil
}

func isContextBackgroundCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Background" {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || pass.TypesInfo == nil {
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
	return pkgName.Imported().Path() == "context"
}

// contextParamName returns the first non-blank context.Context parameter name.
func contextParamName(pass *analysis.Pass, ftype *ast.FuncType) (string, bool) {
	if ftype == nil || ftype.Params == nil {
		return "", false
	}
	ctxType := contextType(pass)
	if ctxType == nil {
		return "", false
	}
	for _, field := range ftype.Params.List {
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
