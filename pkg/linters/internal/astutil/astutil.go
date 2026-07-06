// Package astutil provides shared AST/type helper functions used by linters.
package astutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// IsLocalObject reports whether obj is a local (non-package-scope) object.
func IsLocalObject(obj types.Object) bool {
	if obj == nil {
		return false
	}
	parent := obj.Parent()
	if parent == nil {
		return false
	}
	pkg := obj.Pkg()
	return pkg == nil || parent != pkg.Scope()
}

// RhsExprForIndex returns the RHS expression mapped to idx when available.
// When rhs has a single expression, only idx==0 is considered mapped.
func RhsExprForIndex(rhs []ast.Expr, idx int) (ast.Expr, bool) {
	switch {
	case len(rhs) == 0:
		return nil, false
	case len(rhs) == 1 && idx == 0:
		return rhs[0], true
	case idx < len(rhs):
		return rhs[idx], true
	default:
		return nil, false
	}
}

// IsStringLiteral reports whether expr is a string literal.
func IsStringLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

// EnclosingFuncType extracts a function type from a FuncDecl or FuncLit node.
func EnclosingFuncType(node ast.Node) *ast.FuncType {
	switch fn := node.(type) {
	case *ast.FuncDecl:
		return fn.Type
	case *ast.FuncLit:
		return fn.Type
	default:
		return nil
	}
}

// ContextContextType returns the types.Type for context.Context, or nil if
// the context package is not imported.
func ContextContextType(pass *analysis.Pass) types.Type {
	if pass == nil || pass.Pkg == nil {
		return nil
	}
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

// ContextParamName returns the name of the first context.Context parameter in
// fn, and true, or "", false if none exists.
func ContextParamName(pass *analysis.Pass, fn *ast.FuncType) (string, bool) {
	if pass == nil || pass.TypesInfo == nil || fn == nil || fn.Params == nil {
		return "", false
	}
	ctxType := ContextContextType(pass)
	if ctxType == nil {
		return "", false
	}
	for _, field := range fn.Params.List {
		t := pass.TypesInfo.TypeOf(field.Type)
		if t == nil || !types.Identical(t, ctxType) {
			continue
		}
		for _, name := range field.Names {
			if name.Name != "_" {
				return name.Name, true
			}
		}
	}
	return "", false
}

// IsFmtErrorf reports whether call is a call to fmt.Errorf (including aliases).
func IsFmtErrorf(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Errorf" {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(pkgIdent)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}
	return pkgName.Imported().Path() == "fmt"
}

// CalledOSFunc reports whether call resolves to a function in package os. If
// allowedNames are provided, the function name must match one of them.
func CalledOSFunc(pass *analysis.Pass, call *ast.CallExpr, allowedNames ...string) (*types.Func, bool) {
	if pass == nil || pass.TypesInfo == nil || call == nil {
		return nil, false
	}

	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.Uses[fun.Sel]
	case *ast.Ident:
		obj = pass.TypesInfo.Uses[fun]
	default:
		return nil, false
	}

	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "os" {
		return nil, false
	}
	if len(allowedNames) == 0 {
		return fn, true
	}
	if slices.Contains(allowedNames, fn.Name()) {
		return fn, true
	}
	return nil, false
}

// IsPkgSelector reports whether sel is a selector on an imported package with
// the given import path.
func IsPkgSelector(pass *analysis.Pass, sel *ast.SelectorExpr, pkgPath string) bool {
	if pass == nil || pass.TypesInfo == nil || sel == nil {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(pkgIdent)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok || pkgName.Imported() == nil {
		return false
	}
	return pkgName.Imported().Path() == pkgPath
}

// FlipComparisonOp returns the comparison operator with left and right
// operands swapped.
func FlipComparisonOp(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GTR
	case token.GTR:
		return token.LSS
	case token.LEQ:
		return token.GEQ
	case token.GEQ:
		return token.LEQ
	default:
		return op
	}
}

// IsGoOrDeferClosure reports whether the FuncLit at funcLitCur is the direct
// callee of a go or defer statement, handling parenthesized forms like
// defer (func(){})().
func IsGoOrDeferClosure(funcLitCur inspector.Cursor) bool {
	// Walk up from the FuncLit, unwrapping any ParenExpr wrappers, to find the
	// enclosing CallExpr. This handles parenthesized forms like defer (func(){})().
	cur := funcLitCur.Parent()
	for {
		if cur.Node() == nil {
			return false
		}
		if _, ok := cur.Node().(*ast.ParenExpr); ok {
			cur = cur.Parent()
			continue
		}
		break
	}

	call, ok := cur.Node().(*ast.CallExpr)
	if !ok {
		return false
	}
	// Unwrap ParenExpr from call.Fun and verify it resolves to our FuncLit.
	callee := call.Fun
	for {
		if paren, ok := callee.(*ast.ParenExpr); ok {
			callee = paren.X
		} else {
			break
		}
	}
	if callee != funcLitCur.Node() {
		return false
	}

	grandparent := cur.Parent().Node()
	if grandparent == nil {
		return false
	}

	switch grandparent.(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		return true
	default:
		return false
	}
}

// Inspector extracts the *inspector.Inspector from pass.ResultOf.
// It returns an error if the result has an unexpected type.
func Inspector(pass *analysis.Pass) (*inspector.Inspector, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}
	return insp, nil
}

// Root extracts the inspector root cursor from pass.ResultOf.
// It returns an error if the inspect result has an unexpected type.
func Root(pass *analysis.Pass) (inspector.Cursor, error) {
	insp, err := Inspector(pass)
	if err != nil {
		return inspector.Cursor{}, err
	}
	return insp.Root(), nil
}

// NodeText formats node as Go source text using go/printer.
func NodeText(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
