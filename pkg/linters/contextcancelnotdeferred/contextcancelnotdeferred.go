// Package contextcancelnotdeferred implements a Go analysis linter that flags
// context cancel functions called manually instead of deferred.
package contextcancelnotdeferred

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/resourcetracker"
)

// Analyzer is the context-cancel-not-deferred analysis pass.
var Analyzer = resourcetracker.NewAnalyzer(resourcetracker.Config[types.Object]{
	Name:         "contextcancelnotdeferred",
	Doc:          "reports context cancel functions that are called directly instead of deferred",
	Message:      "context cancel function should be deferred immediately after context.WithCancel/WithTimeout/WithDeadline",
	Acquisitions: cancelFuncAcquisitions,
	CleanupKey:   cancelCallKey,
})

// cancelFuncAcquisitions reports cancel functions bound by assignments such as
// ctx, cancel := context.WithCancel(...).
func cancelFuncAcquisitions(pass *analysis.Pass, node ast.Node) []resourcetracker.Acquisition[types.Object] {
	assign, ok := node.(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 || len(assign.Lhs) < 2 {
		return nil
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || !isContextWithCancelCall(pass, call) {
		return nil
	}
	ident, ok := assign.Lhs[1].(*ast.Ident)
	if !ok || ident.Name == "_" {
		return nil
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil
	}
	return []resourcetracker.Acquisition[types.Object]{{Key: obj, Pos: call.Pos()}}
}

// isContextWithCancelCall returns true if the call is context.WithCancel,
// context.WithTimeout, or context.WithDeadline.
func isContextWithCancelCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if !astutil.IsPkgSelector(pass, sel, "context") {
		return false
	}
	switch sel.Sel.Name {
	case "WithCancel", "WithTimeout", "WithDeadline":
		return true
	default:
		return false
	}
}

// cancelCallKey returns the object for the function invoked by call, enabling
// matching of cancel() invocations against tracked cancel functions.
func cancelCallKey(pass *analysis.Pass, call *ast.CallExpr) (types.Object, bool) {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil, false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}
	return obj, true
}
