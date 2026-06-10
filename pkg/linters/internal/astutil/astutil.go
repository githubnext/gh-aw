// Package astutil provides shared AST/type helper functions used by linters.
package astutil

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"

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

// Inspector extracts the *inspector.Inspector from pass.ResultOf.
// It returns an error if the result has an unexpected type.
func Inspector(pass *analysis.Pass) (*inspector.Inspector, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}
	return insp, nil
}

// NodeText formats node as Go source text using go/printer.
func NodeText(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
