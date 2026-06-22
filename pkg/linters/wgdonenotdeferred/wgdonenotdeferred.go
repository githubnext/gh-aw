// Package wgdonenotdeferred implements a Go analysis linter that flags
// sync.WaitGroup Done() calls that are not deferred, which can lead to
// deadlocks if the function panics or returns early before Done() is reached.
package wgdonenotdeferred

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the wgdonenotdeferred analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "wgdonenotdeferred",
	Doc:      "reports sync.WaitGroup Done() calls that are not deferred, which can cause deadlock if the function panics",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/wgdonenotdeferred",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "wgdonenotdeferred")

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return
			}
			pos := pass.Fset.PositionFor(fn.Pos(), false)
			if filecheck.IsTestFile(pos.Filename) {
				return
			}
			body = fn.Body
		case *ast.FuncLit:
			if fn.Body == nil {
				return
			}
			body = fn.Body
		}
		if body == nil {
			return
		}
		inspectBody(pass, noLintLinesByFile, body)
	})

	return nil, nil
}

func inspectBody(pass *analysis.Pass, noLintLinesByFile map[string]map[int]struct{}, body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		// Do not descend into nested function literals — Preorder visits them separately,
		// so descending here would cause duplicate diagnostics.
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		// Flag any statement-level wg.Done() call that is not wrapped in defer.
		// A deferred call appears as a DeferStmt, not an ExprStmt, so checking for
		// ExprStmt naturally excludes deferred calls.
		if exprStmt, ok := node.(*ast.ExprStmt); ok {
			if call, ok := exprStmt.X.(*ast.CallExpr); ok {
				if isWaitGroupDone(pass, call) {
					pos := pass.Fset.PositionFor(call.Pos(), false)
					if !nolint.HasDirective(pos, noLintLinesByFile) {
						pass.ReportRangef(call,
							"sync.WaitGroup Done() should be deferred to prevent deadlock if the function panics")
					}
				}
			}
		}
		return true
	})
}

func isWaitGroupDone(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Done" {
		return false
	}
	return isWaitGroupType(pass.TypesInfo.TypeOf(sel.X))
}

func isWaitGroupType(t types.Type) bool {
	if t == nil {
		return false
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "sync" && obj.Name() == "WaitGroup"
}
