// Package fileclosenotdeferred implements a Go analysis linter that flags
// file operations where Close() is not immediately deferred.
package fileclosenotdeferred

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the file-close-not-deferred analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fileclosenotdeferred",
	Doc:      "reports file operations where Close() is not immediately deferred, which can lead to resource leaks",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/file-close-not-deferred",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		pos := pass.Fset.PositionFor(fn.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		// Track file variables: varName -> (open position, has defer)
		fileVars := make(map[string]*fileVarState)

		// Walk all statements in the function body, including nested blocks
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if node == nil {
				return false
			}

			// Look for assignments like: file, err := os.Open(...)
			if assign, ok := node.(*ast.AssignStmt); ok {
				for i, rhs := range assign.Rhs {
					if call, ok := rhs.(*ast.CallExpr); ok && isFileOpenCall(call) {
						if i < len(assign.Lhs) {
							if ident, ok := assign.Lhs[i].(*ast.Ident); ok && ident.Name != "_" {
								fileVars[ident.Name] = &fileVarState{
									openPos:   call.Pos(),
									hasDefer:  false,
									hasManuaClose: false,
								}
							}
						}
					}
				}
			}

			// Look for defer file.Close()
			if deferStmt, ok := node.(*ast.DeferStmt); ok {
				if varName := getCloseCallVar(deferStmt.Call); varName != "" {
					if state, found := fileVars[varName]; found {
						state.hasDefer = true
					}
				}
			}

			// Look for non-deferred file.Close() in expression statements
			if exprStmt, ok := node.(*ast.ExprStmt); ok {
				if call, ok := exprStmt.X.(*ast.CallExpr); ok {
					if varName := getCloseCallVar(call); varName != "" {
						if state, found := fileVars[varName]; found {
							state.hasManuaClose = true
						}
					}
				}
			}

			// Look for non-deferred file.Close() in assignments (e.g., closeErr := fd.Close())
			if assign, ok := node.(*ast.AssignStmt); ok {
				for _, rhs := range assign.Rhs {
					if call, ok := rhs.(*ast.CallExpr); ok {
						if varName := getCloseCallVar(call); varName != "" {
							if state, found := fileVars[varName]; found {
								state.hasManuaClose = true
							}
						}
					}
				}
			}

			return true
		})

		// Report files with manual close but no defer
		for _, state := range fileVars {
			if state.hasManuaClose && !state.hasDefer {
				pass.Report(analysis.Diagnostic{
					Pos:     state.openPos,
					Message: "file Close() should be deferred immediately after successful open to prevent resource leaks",
				})
			}
		}
	})

	return nil, nil
}

type fileVarState struct {
	openPos       token.Pos
	hasDefer      bool
	hasManuaClose bool
}

// isFileOpenCall returns true if the call is os.Open, os.Create, or os.OpenFile
func isFileOpenCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "os" {
		return false
	}
	return sel.Sel.Name == "Open" || sel.Sel.Name == "Create" || sel.Sel.Name == "OpenFile"
}

// getCloseCallVar returns the variable name if call is like file.Close()
func getCloseCallVar(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}
