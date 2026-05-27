// Package regexpcompileinfunction implements a Go analysis linter that flags
// calls to regexp.MustCompile() and regexp.Compile() inside function bodies.
// These should be moved to package-level variables for performance.
package regexpcompileinfunction

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
)

// Analyzer is the regexp-compile-in-function analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "regexpcompileinfunction",
	Doc:      "reports regexp.MustCompile and regexp.Compile calls inside function bodies that should be moved to package-level variables",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/regexpcompileinfunction",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("inspect analyzer result has unexpected type %T", pass.ResultOf[inspect.Analyzer])
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok || !isRegexpCompileCall(call) {
			return true
		}
		if !hasConstantStringPattern(pass, call) {
			return true
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return true
		}

		// Check if we're inside a function (not package-level)
		if isInsideFunction(stack) {
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: "regexp compilation inside function should be moved to package-level variable",
			})
		}

		return true
	})

	return nil, nil
}

// isRegexpCompileCall checks if the call is to regexp.MustCompile or regexp.Compile.
func isRegexpCompileCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "regexp" && (sel.Sel.Name == "MustCompile" || sel.Sel.Name == "Compile")
}

// isInsideFunction checks if the current node is inside a function body.
func isInsideFunction(stack []ast.Node) bool {
	for i := range slices.Backward(stack) {
		switch stack[i].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return true
		}
	}
	return false
}

// hasConstantStringPattern checks whether the regexp pattern is a compile-time constant string,
// such as a string literal or const identifier (but not variables/parameters).
func hasConstantStringPattern(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	patternArg := call.Args[0]
	if lit, ok := patternArg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return true
	}

	tv, ok := pass.TypesInfo.Types[patternArg]
	if !ok || tv.Value == nil || tv.Type == nil {
		return false
	}

	basic, ok := tv.Type.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}
