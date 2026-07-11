// Package httprespbodyclose implements a Go analysis linter that flags
// HTTP response Body.Close() calls that are made directly instead of deferred.
package httprespbodyclose

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("linters:httprespbodyclose")

// Analyzer is the http-resp-body-close analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "httprespbodyclose",
	Doc:      "reports HTTP response Body.Close() calls that are not deferred, which risks resource leaks on early return",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/httprespbodyclose",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	log.Printf("analyzing package %s", pass.Pkg.Path())

	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "httprespbodyclose")

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		inspectFunc(pass, n, noLintLinesByFile)
	})

	return nil, nil
}

func inspectFunc(pass *analysis.Pass, n ast.Node, noLintLinesByFile map[string]map[int]struct{}) {
	var body *ast.BlockStmt
	switch fn := n.(type) {
	case *ast.FuncDecl:
		if fn.Body == nil {
			return
		}
		body = fn.Body
	case *ast.FuncLit:
		if fn.Body == nil {
			return
		}
		body = fn.Body
	}

	pos := pass.Fset.PositionFor(n.Pos(), false)
	if filecheck.IsTestFile(pos.Filename) {
		return
	}

	respVars := make(map[types.Object]*respVarState)

	ast.Inspect(body, func(node ast.Node) bool {
		return inspectNode(pass, respVars, node, noLintLinesByFile)
	})

	for _, state := range respVars {
		if state.hasManualClose && !state.hasDeferClose {
			if !nolint.HasDirective(pass.Fset.PositionFor(state.assignPos, false), noLintLinesByFile) {
				reportMissingDefer(pass, state)
			}
		}
	}
}

func inspectNode(pass *analysis.Pass, respVars map[types.Object]*respVarState, node ast.Node, noLintLinesByFile map[string]map[int]struct{}) bool {
	if node == nil {
		return false
	}
	if _, ok := node.(*ast.FuncLit); ok {
		return false
	}

	if assign, ok := node.(*ast.AssignStmt); ok {
		trackHTTPAssignment(pass, respVars, assign, noLintLinesByFile)
		// Also check RHS for manual Body.Close() in assignments like: err := resp.Body.Close()
		for _, rhs := range assign.Rhs {
			if call, ok := rhs.(*ast.CallExpr); ok {
				markBodyClose(pass, respVars, call)
			}
		}
	}

	if deferStmt, ok := node.(*ast.DeferStmt); ok {
		if obj := bodyCloseReceiver(pass, deferStmt.Call); obj != nil {
			if state, found := respVars[obj]; found {
				state.hasDeferClose = true
			}
		}
	}

	if exprStmt, ok := node.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			markBodyClose(pass, respVars, call)
		}
	}

	return true
}

func markBodyClose(pass *analysis.Pass, respVars map[types.Object]*respVarState, call *ast.CallExpr) {
	obj := bodyCloseReceiver(pass, call)
	if obj == nil {
		return
	}
	if state, found := respVars[obj]; found {
		state.hasManualClose = true
		if !state.manualClosePos.IsValid() {
			state.manualClosePos = call.Pos()
		}
	}
}

func trackHTTPAssignment(pass *analysis.Pass, respVars map[types.Object]*respVarState, assign *ast.AssignStmt, noLintLinesByFile map[string]map[int]struct{}) {
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil || !isHTTPResponsePtr(obj.Type()) {
			continue
		}

		var callPos token.Pos
		switch {
		case len(assign.Rhs) == 1:
			// Multi-return function: resp, err := client.Do(req)
			if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
				callPos = call.Pos()
			}
		case i < len(assign.Rhs):
			// Parallel assignment: resp, x := f1(), f2()
			if call, ok := assign.Rhs[i].(*ast.CallExpr); ok {
				callPos = call.Pos()
			}
		}
		if !callPos.IsValid() {
			continue
		}

		// Report any prior unresolved violation before overwriting state.
		if prev, exists := respVars[obj]; exists && prev.hasManualClose && !prev.hasDeferClose {
			if !nolint.HasDirective(pass.Fset.PositionFor(prev.assignPos, false), noLintLinesByFile) {
				reportMissingDefer(pass, prev)
			}
		}
		respVars[obj] = &respVarState{assignPos: callPos}
	}
}

type respVarState struct {
	assignPos      token.Pos
	manualClosePos token.Pos
	hasDeferClose  bool
	hasManualClose bool
}

func reportMissingDefer(pass *analysis.Pass, state *respVarState) {
	log.Printf("flagging non-deferred Body.Close() at %s", pass.Fset.PositionFor(state.assignPos, false))

	diag := analysis.Diagnostic{
		Pos:     state.assignPos,
		Message: "HTTP response Body.Close() should be deferred immediately after receiving the response to prevent resource leaks",
	}
	if state.manualClosePos.IsValid() {
		diag.Related = []analysis.RelatedInformation{
			{
				Pos:     state.manualClosePos,
				Message: "manual Body.Close() call",
			},
		}
	}
	pass.Report(diag)
}

// isHTTPResponsePtr reports whether t is *net/http.Response.
func isHTTPResponsePtr(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "net/http" && obj.Name() == "Response"
}

// bodyCloseReceiver returns the types.Object for the *http.Response receiver
// of a resp.Body.Close() call, or nil if the call is not of that form.
func bodyCloseReceiver(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return nil
	}
	bodySel, ok := sel.X.(*ast.SelectorExpr)
	if !ok || bodySel.Sel.Name != "Body" {
		return nil
	}
	ident, ok := bodySel.X.(*ast.Ident)
	if !ok {
		return nil
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil || !isHTTPResponsePtr(obj.Type()) {
		return nil
	}
	return obj
}
