// Package httpnoctx implements a Go analysis linter that flags HTTP calls
// that do not accept a context.Context: (*http.Client).Get, .Head, .Post,
// .PostForm and the package-level http.Get/Head/Post/PostForm shortcuts.
// The fix is to build the request with http.NewRequestWithContext and call
// client.Do so that cancellation and deadline are propagated.
package httpnoctx

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the http-no-ctx analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "httpnoctx",
	Doc:      "reports http.Client and package-level HTTP calls that do not accept a context.Context; use http.NewRequestWithContext + client.Do instead",
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

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if !contextFreeMethods[sel.Sel.Name] {
			return
		}

		if isHTTPClientReceiver(pass, sel.X) {
			pass.ReportRangef(call,
				"(*http.Client).%s does not accept a context; use http.NewRequestWithContext + client.Do to propagate cancellation",
				sel.Sel.Name,
			)
			return
		}

		if isHTTPPackage(pass, sel.X) {
			pass.ReportRangef(call,
				"http.%s does not accept a context; use http.NewRequestWithContext + http.DefaultClient.Do to propagate cancellation",
				sel.Sel.Name,
			)
		}
	})

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
