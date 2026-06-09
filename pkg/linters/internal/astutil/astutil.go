// Package astutil provides shared AST/type helper functions used by linters.
package astutil

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
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
