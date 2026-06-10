// Package fileclosenotdeferred implements a Go analysis linter that flags
// file operations where Close() is not immediately deferred.
package fileclosenotdeferred

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the file-close-not-deferred analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fileclosenotdeferred",
	Doc:      "reports file operations where Close() is not immediately deferred, which can lead to resource leaks",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/fileclosenotdeferred",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		inspectFileFuncDecl(pass, n)
	})

	return nil, nil
}

func inspectFileFuncDecl(pass *analysis.Pass, n ast.Node) {
	fn, ok := n.(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return
	}

	pos := pass.Fset.PositionFor(fn.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) {
		return
	}

	// Track file variables: types.Object -> *fileVarState (open position, hasDefer, hasManualClose)
	// Keyed by types.Object so variable shadowing across inner scopes is handled correctly.
	fileVars := make(map[types.Object]*fileVarState)

	// Walk all statements in the function body, including nested blocks,
	// but stop at function literals so closures are analysed independently.
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		return analyzeASTNodeForFileClosePatterns(pass, fileVars, node)
	})

	// Report files with manual close but no defer
	for _, state := range fileVars {
		if state.hasManualClose && !state.hasDefer {
			pass.Report(analysis.Diagnostic{
				Pos:     state.openPos,
				Message: "file Close() should be deferred immediately after successful open to prevent resource leaks",
			})
		}
	}
}

func analyzeASTNodeForFileClosePatterns(pass *analysis.Pass, fileVars map[types.Object]*fileVarState, node ast.Node) bool {
	if node == nil {
		return false
	}

	// Do not descend into function literals — closures are independent execution
	// contexts and should be analyzed separately to avoid false positives.
	if _, ok := node.(*ast.FuncLit); ok {
		return false
	}

	// Look for assignments like: file, err := os.Open(...)
	if assign, ok := node.(*ast.AssignStmt); ok {
		trackFileOpenAssignment(pass, fileVars, assign)
	}

	// Look for defer file.Close()
	if deferStmt, ok := node.(*ast.DeferStmt); ok {
		if obj := getCloseReceiverObject(pass, deferStmt.Call); obj != nil {
			if state, found := fileVars[obj]; found {
				state.hasDefer = true
			}
		}
	}

	// Look for non-deferred file.Close() in expression statements
	if exprStmt, ok := node.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			markManualClose(pass, fileVars, call)
		}
	}

	// Look for non-deferred file.Close() in assignments (e.g., closeErr := fd.Close())
	if assign, ok := node.(*ast.AssignStmt); ok {
		for _, rhs := range assign.Rhs {
			if call, ok := rhs.(*ast.CallExpr); ok {
				markManualClose(pass, fileVars, call)
			}
		}
	}

	return true
}

func trackFileOpenAssignment(pass *analysis.Pass, fileVars map[types.Object]*fileVarState, assign *ast.AssignStmt) {
	for i, rhs := range assign.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || !isFileOpenCall(call) {
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
		// If this object was already tracked from a prior open on the
		// same binding (plain = reassignment), report any unresolved
		// violation immediately before overwriting the state.
		if prev, exists := fileVars[obj]; exists && prev.hasManualClose && !prev.hasDefer {
			pass.Report(analysis.Diagnostic{
				Pos:     prev.openPos,
				Message: "file Close() should be deferred immediately after successful open to prevent resource leaks",
			})
		}
		fileVars[obj] = &fileVarState{openPos: call.Pos()}
	}
}

func markManualClose(pass *analysis.Pass, fileVars map[types.Object]*fileVarState, call *ast.CallExpr) {
	obj := getCloseReceiverObject(pass, call)
	if obj == nil {
		return
	}
	if state, found := fileVars[obj]; found {
		state.hasManualClose = true
	}
}

type fileVarState struct {
	openPos        token.Pos
	hasDefer       bool
	hasManualClose bool
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

// getCloseReceiverObject returns the types.Object for the receiver if call is like file.Close(),
// enabling correct identification across variable shadowing.
func getCloseReceiverObject(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return nil
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}
	return pass.TypesInfo.ObjectOf(ident)
}
