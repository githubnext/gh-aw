// Package goroutinemissingrecover implements a Go analysis linter that flags
// goroutines started via a function literal whose body does not install a
// top-level defer/recover guard.
//
// An unrecovered panic inside a goroutine terminates the entire process and
// is not caught by the caller's recover, so any goroutine that might panic
// should defer a recover to contain the failure locally.
//
// Only goroutines launched with a function literal (`go func() { ... }()`)
// are checked. Goroutines that call a named function (`go f()`) are out of
// scope because the named function can install its own recovery.
package goroutinemissingrecover

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:goroutinemissingrecover")

// Analyzer is the goroutine-missing-recover analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "goroutinemissingrecover",
	Doc:      "reports goroutines started via a function literal that do not install a top-level defer/recover guard",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/goroutinemissingrecover",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())

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

	nodeFilter := []ast.Node{(*ast.GoStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		goStmt, ok := n.(*ast.GoStmt)
		if !ok {
			return
		}

		// Only flag goroutines started with a function literal, not named functions.
		// Unwrap parentheses: go (func() { ... })() is equivalent to go func() { ... }()
		call, ok := unwrapParens(goStmt.Call.Fun).(*ast.FuncLit)
		if !ok {
			return
		}

		position := pass.Fset.PositionFor(goStmt.Pos(), false)
		if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
			return
		}

		if nolint.HasDirectiveForLinter(position, noLintIndex, "goroutinemissingrecover") {
			return
		}

		if hasTopLevelRecoverDefer(call.Body, pass.TypesInfo) {
			return
		}

		pkgLog.Printf("flagging goroutine without recover at %s", position)
		pass.ReportRangef(goStmt, "goroutine launched via a function literal without a top-level defer/recover; add defer func() { if r := recover(); r != nil { ... } }() to contain panics")
	})

	return nil, nil
}

// unwrapParens removes any surrounding *ast.ParenExpr nodes, returning the
// innermost non-parenthesised expression. This handles the rare but valid
// syntax `go (func() { ... })()`.
func unwrapParens(expr ast.Expr) ast.Expr {
	for {
		p, ok := expr.(*ast.ParenExpr)
		if !ok {
			return expr
		}
		expr = p.X
	}
}

// hasTopLevelRecoverDefer reports whether body contains a top-level defer
// statement whose call is a function literal that itself calls recover().
// Only the direct statements of body are examined; nested function bodies are
// not descended into.
func hasTopLevelRecoverDefer(body *ast.BlockStmt, typesInfo *types.Info) bool {
	if body == nil {
		return false
	}
	for _, stmt := range body.List {
		deferStmt, ok := stmt.(*ast.DeferStmt)
		if !ok {
			continue
		}
		// Unwrap parentheses: defer (func() { ... })() is valid Go.
		fn, ok := unwrapParens(deferStmt.Call.Fun).(*ast.FuncLit)
		if !ok {
			continue
		}
		if containsRecoverCall(fn.Body, typesInfo) {
			return true
		}
	}
	return false
}

// containsRecoverCall reports whether body contains a direct call to the
// built-in recover() function. Nested function literals are not descended
// into: recover() inside a nested closure only guards that closure's stack
// frame, not the enclosing defer, so it does not count as a panic guard.
func containsRecoverCall(body *ast.BlockStmt, typesInfo *types.Info) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		// Do not descend into nested function literals — their recover() only
		// protects the nested function's own stack frame, not the outer defer.
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		// Verify it is the built-in recover, not a user-defined function with
		// the same name (matching the pattern used by mapclearloop for delete).
		if obj, isBuiltin := typesInfo.Uses[ident].(*types.Builtin); isBuiltin && obj.Name() == "recover" {
			found = true
			return false
		}
		return true
	})
	return found
}
