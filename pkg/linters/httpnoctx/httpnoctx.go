// Package httpnoctx implements a Go analysis linter that flags HTTP calls
// that do not accept a context.Context: (*http.Client).Get, .Head, .Post,
// .PostForm and the package-level http.Get/Head/Post/PostForm shortcuts.
// It also flags http.NewRequest inside functions that already receive
// context.Context, and http.DefaultClient.Do which uses a timeout-less client.
// The fix is to build the request with http.NewRequestWithContext and use a
// client with a timeout so cancellation and deadline are propagated.
package httpnoctx

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the http-no-ctx analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "httpnoctx",
	Doc:      "reports context-free net/http request paths: http.Client/http package helpers without context, http.NewRequest in context-aware functions, and http.DefaultClient.Do",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/httpnoctx",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// contextFreeMethods is the set of http.Client (and package-level) HTTP
// methods that accept no context.Context argument.
var contextFreeMethods = map[string]bool{
	"Get":      true,
	"Head":     true,
	"Post":     true,
	"PostForm": true,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "httpnoctx")
	ctxType := contextContextType(pass)

	for cursor := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cursor.Node().(*ast.CallExpr)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if contextFreeMethods[sel.Sel.Name] {
			if isHTTPClientReceiver(pass, sel.X) {
				pass.ReportRangef(call,
					"(*http.Client).%s does not accept a context; use http.NewRequestWithContext + client.Do to propagate cancellation",
					sel.Sel.Name,
				)
				continue
			}

			if isHTTPPackage(pass, sel.X) {
				pass.ReportRangef(call,
					"http.%s does not accept a context; use http.NewRequestWithContext + http.DefaultClient.Do to propagate cancellation",
					sel.Sel.Name,
				)
				continue
			}
		}

		if sel.Sel.Name == "NewRequest" && isHTTPPackage(pass, sel.X) && hasContextInEnclosingFunc(pass, cursor, ctxType) {
			pass.ReportRangef(call,
				"http.NewRequest does not propagate context; use http.NewRequestWithContext when context.Context is in scope",
			)
			continue
		}

		if sel.Sel.Name == "Do" && isHTTPDefaultClient(pass, sel.X) {
			pass.ReportRangef(call,
				"http.DefaultClient.Do uses a timeout-less client; use a dedicated *http.Client with Timeout set",
			)
		}
	}

	return nil, nil
}

// isHTTPClientReceiver reports whether expr has type *http.Client.
func isHTTPClientReceiver(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "Client" && obj.Pkg() != nil && obj.Pkg().Path() == "net/http"
}

// isHTTPPackage reports whether expr is an identifier for the "net/http" package.
func isHTTPPackage(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
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
	return pkgName.Imported().Path() == "net/http"
}

func hasContextInEnclosingFunc(pass *analysis.Pass, cursor inspector.Cursor, ctxType types.Type) bool {
	if ctxType == nil {
		return false
	}

	for enclosing := range cursor.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
		fnType := enclosingFuncType(enclosing.Node())
		if fnType == nil || fnType.Params == nil {
			continue
		}

		for _, field := range fnType.Params.List {
			t := pass.TypesInfo.TypeOf(field.Type)
			if t == nil || !types.Identical(t, ctxType) {
				continue
			}
			for _, name := range field.Names {
				if name.Name != "_" {
					return true
				}
			}
		}
	}

	return false
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

func contextContextType(pass *analysis.Pass) types.Type {
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() != "context" {
			continue
		}
		if obj := pkg.Scope().Lookup("Context"); obj != nil {
			return obj.Type()
		}
	}
	return nil
}

func isHTTPDefaultClient(pass *analysis.Pass, expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "DefaultClient" {
		return false
	}
	return isHTTPPackage(pass, sel.X)
}
