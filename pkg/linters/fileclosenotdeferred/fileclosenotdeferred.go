// Package fileclosenotdeferred implements a Go analysis linter that flags
// file operations where Close() is not immediately deferred.
package fileclosenotdeferred

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/resourcetracker"
)

// Analyzer is the file-close-not-deferred analysis pass.
var Analyzer = resourcetracker.NewAnalyzer(resourcetracker.Config[types.Object]{
	Name:         "fileclosenotdeferred",
	Doc:          "reports file operations where Close() is not immediately deferred, which can lead to resource leaks",
	Message:      "file Close() should be deferred immediately after successful open to prevent resource leaks",
	Acquisitions: fileOpenAcquisitions,
	CleanupKey:   closeCallKey,
})

// fileOpenAcquisitions reports file variables bound by assignments such as
// file, err := os.Open(...).
func fileOpenAcquisitions(pass *analysis.Pass, node ast.Node) []resourcetracker.Acquisition[types.Object] {
	assign, ok := node.(*ast.AssignStmt)
	if !ok {
		return nil
	}

	var acquisitions []resourcetracker.Acquisition[types.Object]
	for i, rhs := range assign.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || !isFileOpenCall(pass, call) {
			continue
		}
		if i >= len(assign.Lhs) {
			continue
		}
		ident, ok := assign.Lhs[i].(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil {
			continue
		}
		acquisitions = append(acquisitions, resourcetracker.Acquisition[types.Object]{Key: obj, Pos: call.Pos()})
	}
	return acquisitions
}

// isFileOpenCall returns true if the call is os.Open, os.Create, or os.OpenFile
func isFileOpenCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if !astutil.IsPkgSelector(pass, sel, "os") {
		return false
	}
	return sel.Sel.Name == "Open" || sel.Sel.Name == "Create" || sel.Sel.Name == "OpenFile"
}

// closeCallKey returns the types.Object for the receiver if call is like file.Close(),
// enabling correct identification across variable shadowing.
func closeCallKey(pass *analysis.Pass, call *ast.CallExpr) (types.Object, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return nil, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}
	return obj, true
}
