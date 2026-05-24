// Package panicinlibrarycode implements a Go analysis linter that flags
// panic() calls in library (pkg/) packages.
package panicinlibrarycode

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the panic-in-library-code analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "panicinlibrarycode",
	Doc:      "reports panic() calls in library code under pkg/ that should return errors instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/panic-in-library-code",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	// Skip packages under cmd/ entry-points — they are allowed to call panic.
	if strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call := n.(*ast.CallExpr)
		// Skip test files
		if strings.HasSuffix(pkgPath, ".test") || filecheck.IsTestFile(pass.Fset.Position(call.Pos()).Filename) {
			return true
		}

		// Check if this is a call to the builtin panic function
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name != "panic" {
			return true
		}

		// Verify it's the builtin panic, not a user-defined function
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if _, ok := obj.(*types.Builtin); !ok {
				return true // Not the builtin panic
			}
		}

		if shouldSkipPanic(pass, call, stack) {
			return true
		}

		pass.ReportRangef(call, "avoid panic in library code; return an error instead")
		return true
	})

	return nil, nil
}

// shouldSkipPanic returns true when a panic call is considered acceptable and
// should not be reported. The following cases are exempt:
//
//   - Panics inside init() functions: init() cannot return an error; panicking
//     on critical startup failure is the only viable option.
//   - Panics inside sync.Once.Do() func literals: these are conceptually
//     equivalent to init() — they run at most once for one-time setup.
//   - Panics whose message starts with "BUG:": these document internal invariant
//     violations that are unreachable in a correctly-functioning program.
//   - Panics in functions whose doc comment explicitly documents the panic
//     contract (contains the word "panics"): the author has consciously chosen
//     to panic and documented it.
func shouldSkipPanic(pass *analysis.Pass, call *ast.CallExpr, stack []ast.Node) bool {
	if isInInitFunction(stack) {
		return true
	}
	if isInSyncOnceDoFuncLit(pass, stack) {
		return true
	}
	if panicMessageStartsWithBUG(pass, call) {
		return true
	}
	if hasDocumentedPanicContract(stack) {
		return true
	}
	return false
}

// isInInitFunction reports whether the panic call is nested inside an init()
// function declaration.
func isInInitFunction(stack []ast.Node) bool {
	decl := enclosingFuncDecl(stack)
	return decl != nil && decl.Name != nil && decl.Name.Name == "init"
}

// isInSyncOnceDoFuncLit reports whether the panic call is nested inside a
// func literal that is passed directly as the argument to a sync.Once.Do call.
func isInSyncOnceDoFuncLit(pass *analysis.Pass, stack []ast.Node) bool {
	// Walk up the stack: look for a FuncLit whose parent is a CallExpr of the
	// form <syncOnce>.Do(<funcLit>).
	for i := len(stack) - 1; i >= 1; i-- {
		funcLit, ok := stack[i].(*ast.FuncLit)
		if !ok {
			continue
		}
		parent := stack[i-1]
		callExpr, ok := parent.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Do" {
			continue
		}
		// Verify the receiver is sync.Once (or *sync.Once).
		if t := pass.TypesInfo.TypeOf(sel.X); t != nil && isSyncOnceType(t) {
			// Verify that funcLit is actually one of the call arguments.
			if containsExpr(callExpr.Args, funcLit) {
				return true
			}
		}
	}
	return false
}

// isSyncOnceType returns true if t (or *t) is sync.Once.
func isSyncOnceType(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "sync" && obj.Name() == "Once"
}

// containsExpr reports whether target is present in args (by pointer identity).
func containsExpr(args []ast.Expr, target ast.Expr) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

// panicMessageStartsWithBUG returns true when the first (or only) string
// argument to the panic call has a constant prefix of "BUG:".
// Such panics document internal invariant violations that are unreachable in
// correct programs.
func panicMessageStartsWithBUG(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	prefix, ok := stringPrefix(pass, call.Args[0])
	if !ok {
		return false
	}
	return strings.HasPrefix(prefix, "BUG:")
}

// stringPrefix returns the constant string prefix of an expression, if
// determinable at compile time. It handles string literals, fmt.Sprintf (by
// inspecting the format string), and binary string concatenation.
func stringPrefix(pass *analysis.Pass, expr ast.Expr) (string, bool) {
	// Constant string value (covers string literals and const references).
	if tv, ok := pass.TypesInfo.Types[expr]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		// Only "+" concatenation; take the prefix from the left operand.
		if e.Op != token.ADD {
			return "", false
		}
		return stringPrefix(pass, e.X)
	case *ast.CallExpr:
		// fmt.Sprintf(format, ...) — inspect the format argument.
		if len(e.Args) == 0 {
			return "", false
		}
		return stringPrefix(pass, e.Args[0])
	}
	return "", false
}

// hasDocumentedPanicContract returns true when the enclosing function has a
// doc comment that explicitly documents a panic (i.e., contains the word
// "panics"). This signals that the author has consciously chosen the panic
// semantics and has documented the contract for callers.
func hasDocumentedPanicContract(stack []ast.Node) bool {
	decl := enclosingFuncDecl(stack)
	if decl == nil || decl.Doc == nil {
		return false
	}
	for _, comment := range decl.Doc.List {
		if strings.Contains(strings.ToLower(comment.Text), "panics") {
			return true
		}
	}
	return false
}

// enclosingFuncDecl returns the nearest *ast.FuncDecl ancestor in the stack,
// or nil if the panic is not inside a named function (e.g., inside a func
// literal at package level).
func enclosingFuncDecl(stack []ast.Node) *ast.FuncDecl {
	for i := len(stack) - 1; i >= 0; i-- {
		if fd, ok := stack[i].(*ast.FuncDecl); ok {
			return fd
		}
	}
	return nil
}
