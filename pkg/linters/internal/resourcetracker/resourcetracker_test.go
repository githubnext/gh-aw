//go:build !integration

package resourcetracker_test

import (
	"go/ast"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/github/gh-aw/pkg/linters/internal/resourcetracker"
)

// testAnalyzer tracks resources returned by acquire() and released via
// Release(), exercising the shared deferred-cleanup state machine.
var testAnalyzer = resourcetracker.NewAnalyzer(resourcetracker.Config[types.Object]{
	Name:    "resourcetrackertest",
	Doc:     "reports resources released without defer",
	Message: "resource should be released with defer",
	Acquisitions: func(pass *analysis.Pass, node ast.Node) []resourcetracker.Acquisition[types.Object] {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return nil
		}
		var acquisitions []resourcetracker.Acquisition[types.Object]
		for i, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok {
				continue
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok || fn.Name != "acquire" || i >= len(assign.Lhs) {
				continue
			}
			ident, ok := assign.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}
			obj := pass.TypesInfo.ObjectOf(ident)
			if obj == nil {
				continue
			}
			acquisitions = append(acquisitions, resourcetracker.Acquisition[types.Object]{Key: obj, Pos: call.Pos()})
		}
		return acquisitions
	},
	CleanupKey: func(pass *analysis.Pass, call *ast.CallExpr) (types.Object, bool) {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Release" {
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
	},
})

func TestAnalyzer(t *testing.T) {
	t.Parallel()
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, testAnalyzer, "resourcetracker")
}

func TestNewAnalyzerMetadata(t *testing.T) {
	t.Parallel()
	if testAnalyzer.Name != "resourcetrackertest" {
		t.Errorf("Name = %q, want %q", testAnalyzer.Name, "resourcetrackertest")
	}
	if testAnalyzer.Doc != "reports resources released without defer" {
		t.Errorf("Doc = %q, want %q", testAnalyzer.Doc, "reports resources released without defer")
	}
}
