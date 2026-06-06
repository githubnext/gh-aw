// Package panicinlibrarycode implements a Go analysis linter that flags
// panic() calls in library (pkg/) packages.
package panicinlibrarycode

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
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

	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "panicinlibrarycode")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
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
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirective(position, noLintLinesByFile) {
			return true
		}

		pass.ReportRangef(call, "avoid panic in library code; return an error instead")
		return true
	})

	return nil, nil
}

func shouldSkipPanic(pass *analysis.Pass, call *ast.CallExpr, stack []ast.Node) bool {
	return isInSyncOnceDoFuncLit(pass, stack) ||
		panicMessageStartsWithBUG(pass, call) ||
		isInInitFunction(stack) ||
		hasDocumentedPanicContract(stack)
}

func isInSyncOnceDoFuncLit(pass *analysis.Pass, stack []ast.Node) bool {
	for forwardIdx, node := range slices.Backward(stack) {
		funcLit, ok := node.(*ast.FuncLit)
		if !ok || forwardIdx == 0 {
			continue
		}

		call, ok := stack[forwardIdx-1].(*ast.CallExpr)
		if !ok || !containsExpr(call.Args, funcLit) {
			continue
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Do" {
			continue
		}

		if isSyncOnceType(pass.TypesInfo.TypeOf(sel.X)) {
			return true
		}
	}

	return false
}

func containsExpr(args []ast.Expr, target ast.Expr) bool {
	return slices.Contains(args, target)
}

func isSyncOnceType(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := t.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}

	return named.Obj().Pkg().Path() == "sync" && named.Obj().Name() == "Once"
}

func panicMessageStartsWithBUG(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	prefix, ok := stringPrefix(pass, call.Args[0])
	if !ok {
		return false
	}

	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(prefix)), "BUG:")
}

func stringPrefix(pass *analysis.Pass, expr ast.Expr) (string, bool) {
	if tv, ok := pass.TypesInfo.Types[expr]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return constant.StringVal(tv.Value), true
	}

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		return stringPrefix(pass, e.X)
	case *ast.CallExpr:
		if len(e.Args) == 0 {
			return "", false
		}
		// Only inspect the format argument of fmt.Sprintf to avoid false negatives
		// from arbitrary user functions that happen to receive a "BUG:" string.
		if !isFmtSprintf(pass, e) {
			return "", false
		}
		return stringPrefix(pass, e.Args[0])
	default:
		return "", false
	}
}

// isFmtSprintf reports whether call is an invocation of the fmt.Sprintf function.
func isFmtSprintf(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Sprintf" {
		return false
	}
	if obj := pass.TypesInfo.Uses[sel.Sel]; obj != nil {
		return obj.Pkg() != nil && obj.Pkg().Path() == "fmt"
	}
	return false
}

// isInInitFunction reports whether the panic is inside a top-level init()
// function. Only top-level (no receiver) init functions are recognized;
// methods named init are ordinary methods and are not exempt.
func isInInitFunction(stack []ast.Node) bool {
	decl := enclosingFuncDecl(stack)
	return decl != nil && decl.Recv == nil && decl.Name != nil && decl.Name.Name == "init"
}

func hasDocumentedPanicContract(stack []ast.Node) bool {
	decl := enclosingFuncDecl(stack)
	if decl == nil || decl.Doc == nil {
		return false
	}

	doc := strings.ToLower(decl.Doc.Text())
	return strings.Contains(doc, "panics on") ||
		strings.Contains(doc, "panics if") ||
		strings.Contains(doc, "panic on") ||
		strings.Contains(doc, "panic if")
}

func enclosingFuncDecl(stack []ast.Node) *ast.FuncDecl {
	for _, node := range slices.Backward(stack) {
		if decl, ok := node.(*ast.FuncDecl); ok {
			return decl
		}
	}
	return nil
}
