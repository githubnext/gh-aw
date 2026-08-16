// Package goroutinemissingrecover implements a Go analysis linter that flags
// goroutines started via a function literal whose body does not install a
// top-level defer/recover guard.
//
// An unrecovered panic inside a goroutine terminates the entire process and
// is not caught by the caller's recover, so any goroutine that might panic
// should defer a recover to contain the failure locally.
//
// The guard may be an inline literal (`defer func() { recover() }()`) or a
// deferred named function or method declared in the same package whose body
// calls recover() directly, since both forms stop a panic per the Go spec.
//
// Only goroutines launched with a function literal (`go func() { ... }()`)
// are checked. Goroutines that call a named function (`go f()`) are out of
// scope because the named function can install its own recovery.
package goroutinemissingrecover

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:goroutinemissingrecover")

// Analyzer is the goroutine-missing-recover analysis pass.
var Analyzer = analyzerutil.New("goroutinemissingrecover", "reports goroutines started via a function literal that do not install a top-level defer/recover guard", run)

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())

	generatedFiles, noLintIndex, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	funcBodies := indexFuncBodies(pass)

	nodeFilter := []ast.Node{(*ast.GoStmt)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
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

		if hasTopLevelRecoverDefer(call.Body, pass.TypesInfo, funcBodies) {
			return
		}

		pkgLog.Printf("flagging goroutine without recover at %s", position)
		pass.ReportRangef(goStmt, "goroutine launched via a function literal without a top-level defer/recover; add defer func() { if r := recover(); r != nil { ... } }() to contain panics")
	})
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
// statement whose deferred function calls recover() directly. Per the Go
// spec, recover() stops a panic when it is called directly by the deferred
// function, so both an inline function literal and a statically resolvable
// named function (or method) declared in the same package qualify. Only the
// direct statements of body are examined; nested function bodies are not
// descended into.
func hasTopLevelRecoverDefer(body *ast.BlockStmt, typesInfo *types.Info, funcBodies map[*types.Func]*ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	for _, stmt := range body.List {
		deferStmt, ok := stmt.(*ast.DeferStmt)
		if !ok {
			continue
		}
		// Unwrap parentheses: defer (func() { ... })() is valid Go.
		fun := unwrapParens(deferStmt.Call.Fun)
		if fn, ok := fun.(*ast.FuncLit); ok {
			if containsRecoverCall(fn.Body, typesInfo) {
				return true
			}
			continue
		}
		// The deferred call target is not a literal: try to resolve it to a
		// function declared in this package and inspect its body.
		deferredBody, ok := resolveFuncBody(fun, typesInfo, funcBodies)
		if !ok {
			continue
		}
		if containsRecoverCall(deferredBody, typesInfo) {
			return true
		}
	}
	return false
}

// indexFuncBodies maps every function and method declared in the analysed
// package to its body, so that a deferred named function can be inspected for
// a direct recover() call.
func indexFuncBodies(pass *analysis.Pass) map[*types.Func]*ast.BlockStmt {
	bodies := make(map[*types.Func]*ast.BlockStmt)
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Body == nil || funcDecl.Name == nil {
				continue
			}
			fn, ok := pass.TypesInfo.Defs[funcDecl.Name].(*types.Func)
			if !ok {
				continue
			}
			bodies[fn.Origin()] = funcDecl.Body
		}
	}
	return bodies
}

// resolveFuncBody resolves a deferred call target expression to the body of a
// function declared in the analysed package. It returns false when the target
// is not a statically known function (for example a variable holding a func
// value) or when its body is unavailable (for example a function from another
// package or one without a Go body).
func resolveFuncBody(fun ast.Expr, typesInfo *types.Info, funcBodies map[*types.Func]*ast.BlockStmt) (*ast.BlockStmt, bool) {
	var ident *ast.Ident
	switch target := fun.(type) {
	case *ast.Ident:
		ident = target
	case *ast.SelectorExpr:
		if sel, ok := typesInfo.Selections[target]; ok {
			fn, ok := sel.Obj().(*types.Func)
			if !ok {
				return nil, false
			}
			body, ok := funcBodies[fn.Origin()]
			return body, ok
		}
		// Package-qualified function (pkg.Fn) — no selection recorded and the
		// body lives in another package, so it cannot be inspected.
		return nil, false
	case *ast.IndexExpr:
		// Explicit instantiation of a generic function: f[T].
		return resolveFuncBody(unwrapParens(target.X), typesInfo, funcBodies)
	case *ast.IndexListExpr:
		return resolveFuncBody(unwrapParens(target.X), typesInfo, funcBodies)
	default:
		return nil, false
	}
	fn, ok := typesInfo.Uses[ident].(*types.Func)
	if !ok {
		return nil, false
	}
	body, ok := funcBodies[fn.Origin()]
	return body, ok
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
