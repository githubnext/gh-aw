// Package contextcancelnotdeferred implements a Go analysis linter that flags
// context cancel functions called manually instead of deferred.
package contextcancelnotdeferred

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

var pkgLog = logger.New("linters:contextcancelnotdeferred")

// Analyzer is the context-cancel-not-deferred analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "contextcancelnotdeferred",
	Doc:      "reports context cancel functions that are called directly instead of deferred",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/contextcancelnotdeferred",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())

	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		inspectCancelFuncDecl(pass, n, noLintIndex, generatedFiles)
	})

	return nil, nil
}

func inspectCancelFuncDecl(pass *analysis.Pass, n ast.Node, noLintIndex nolint.DirectiveIndex, generatedFiles filecheck.GeneratedIndex) {
	fn, ok := n.(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return
	}

	pos := pass.Fset.PositionFor(fn.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}

	cancelVars := make(map[types.Object]*cancelVarState)

	ast.Inspect(fn.Body, func(node ast.Node) bool {
		return inspectCancelNode(pass, cancelVars, node, noLintIndex)
	})

	for _, state := range cancelVars {
		if state.hasDirectCancel && !state.hasDeferCancel && !nolint.HasDirectiveForLinter(pass.Fset.PositionFor(state.createPos, false), noLintIndex, "contextcancelnotdeferred") {
			pkgLog.Printf("flagging non-deferred cancel func at %s", pass.Fset.PositionFor(state.createPos, false))
			pass.Report(analysis.Diagnostic{
				Pos:     state.createPos,
				Message: "context cancel function should be deferred immediately after context.WithCancel/WithTimeout/WithDeadline",
			})
		}
	}
}

func inspectCancelNode(pass *analysis.Pass, cancelVars map[types.Object]*cancelVarState, node ast.Node, noLintIndex nolint.DirectiveIndex) bool {
	if node == nil {
		return false
	}

	if _, ok := node.(*ast.FuncLit); ok {
		return false
	}

	if assign, ok := node.(*ast.AssignStmt); ok {
		for i, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok || !isContextWithCancelCall(pass, call) {
				continue
			}
			if len(assign.Rhs) == 1 && i == 0 && len(assign.Lhs) >= 2 {
				ident, ok := assign.Lhs[1].(*ast.Ident)
				if !ok || ident.Name == "_" {
					continue
				}
				obj := pass.TypesInfo.ObjectOf(ident)
				if obj == nil {
					continue
				}
				if prev, exists := cancelVars[obj]; exists && prev.hasDirectCancel && !prev.hasDeferCancel && !nolint.HasDirectiveForLinter(pass.Fset.PositionFor(prev.createPos, false), noLintIndex, "contextcancelnotdeferred") {
					pass.Report(analysis.Diagnostic{
						Pos:     prev.createPos,
						Message: "context cancel function should be deferred immediately after context.WithCancel/WithTimeout/WithDeadline",
					})
				}
				cancelVars[obj] = &cancelVarState{createPos: call.Pos()}
			}
		}
	}

	if deferStmt, ok := node.(*ast.DeferStmt); ok {
		if obj := getCancelCallObj(pass, deferStmt.Call); obj != nil {
			if state, found := cancelVars[obj]; found {
				state.hasDeferCancel = true
			}
		}
	}

	if exprStmt, ok := node.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			if obj := getCancelCallObj(pass, call); obj != nil {
				if state, found := cancelVars[obj]; found {
					state.hasDirectCancel = true
				}
			}
		}
	}

	return true
}

type cancelVarState struct {
	createPos       token.Pos
	hasDeferCancel  bool
	hasDirectCancel bool
}

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

func getCancelCallObj(pass *analysis.Pass, call *ast.CallExpr) types.Object {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil
	}
	return pass.TypesInfo.ObjectOf(ident)
}
