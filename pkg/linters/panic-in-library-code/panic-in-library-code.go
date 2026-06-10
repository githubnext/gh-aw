// Package panicinlibrarycode implements a Go analysis linter that flags
// panic() calls in library (pkg/) packages.
package panicinlibrarycode

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
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

	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "panicinlibrarycode")

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok {
			continue
		}
		// Skip test files
		if strings.HasSuffix(pkgPath, ".test") || filecheck.IsTestFile(pass.Fset.Position(call.Pos()).Filename) {
			continue
		}

		// Check if this is a call to the builtin panic function
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "panic" {
			continue
		}

		// Verify it's the builtin panic, not a user-defined function
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if _, ok := obj.(*types.Builtin); !ok {
				continue // Not the builtin panic
			}
		}

		if shouldSkipPanic(pass, call, cur) {
			continue
		}
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirective(position, noLintLinesByFile) {
			continue
		}

		pass.ReportRangef(call, "avoid panic in library code; return an error instead")
	}

	return nil, nil
}

func shouldSkipPanic(pass *analysis.Pass, call *ast.CallExpr, cur inspector.Cursor) bool {
	return isInSyncOnceDoFuncLit(pass, cur) ||
		panicMessageStartsWithBUG(pass, call) ||
		isInInitFunction(cur) ||
		hasDocumentedPanicContract(cur)
}

func isInSyncOnceDoFuncLit(pass *analysis.Pass, cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.FuncLit)(nil)) {
		funcLit, ok := encl.Node().(*ast.FuncLit)
		if !ok {
			break
		}
		parent := encl.Parent()
		call, ok := parent.Node().(*ast.CallExpr)
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
func isInInitFunction(cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.FuncDecl)(nil)) {
		decl, ok := encl.Node().(*ast.FuncDecl)
		if !ok {
			break
		}
		if decl.Recv == nil && decl.Name != nil && decl.Name.Name == "init" {
			return true
		}
		break // only check the immediate enclosing FuncDecl
	}
	return false
}

func hasDocumentedPanicContract(cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.FuncDecl)(nil)) {
		decl, ok := encl.Node().(*ast.FuncDecl)
		if !ok {
			break
		}
		if decl.Doc != nil {
			doc := strings.ToLower(decl.Doc.Text())
			if strings.Contains(doc, "panics on") ||
				strings.Contains(doc, "panics if") ||
				strings.Contains(doc, "panic on") ||
				strings.Contains(doc, "panic if") {
				return true
			}
		}
		break // only check the immediate enclosing FuncDecl
	}
	return false
}
