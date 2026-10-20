// Package deferrerrorunchecked implements a Go analysis linter that flags
// deferred function calls that return an error but where the error result is not checked.
package deferrerrorunchecked

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the deferred-error-unchecked analysis pass.
var Analyzer = analyzerutil.New("deferrerrorunchecked", "reports deferred function calls that discard an error return value", run)

var builtinErrorType = types.Universe.Lookup("error").Type()

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.DeferStmt)(nil)}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeDeferredError(pass, n, generatedFiles, noLintIndex)
	})
}

// analyzeDeferredError checks for deferred calls that return error but don't handle it.
func analyzeDeferredError(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	deferStmt, ok := n.(*ast.DeferStmt)
	if !ok {
		return
	}

	callExpr := deferStmt.Call
	if callExpr == nil {
		return
	}

	position := pass.Fset.PositionFor(deferStmt.Pos(), false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}

	if nolint.HasDirectiveForLinter(position, noLintIndex, "deferrerrorunchecked") {
		return
	}

	if pass.TypesInfo == nil {
		return
	}

	sig := deferredSignature(pass, callExpr)
	if sig == nil || !functionReturnsError(sig) {
		return
	}

	pass.Reportf(
		deferStmt.Pos(),
		"deferred call discards error return value; handle the error explicitly",
	)
}

func deferredSignature(pass *analysis.Pass, callExpr *ast.CallExpr) *types.Signature {
	switch fun := astutil.UnwrapParenExpr(callExpr.Fun).(type) {
	case *ast.Ident:
		obj := pass.TypesInfo.Uses[fun]
		if obj == nil {
			return nil
		}
		if sig, ok := obj.Type().(*types.Signature); ok {
			return sig
		}

	case *ast.SelectorExpr:
		sel := pass.TypesInfo.Selections[fun]
		if sel == nil || sel.Kind() != types.MethodVal {
			return nil
		}
		// Direct zero-argument Close methods belong to unchecked_deferredclose,
		// including sites suppressed specifically for that analyzer.
		if fun.Sel.Name == "Close" && len(callExpr.Args) == 0 {
			return nil
		}
		if sig, ok := sel.Obj().Type().(*types.Signature); ok {
			return sig
		}

	default:
		if sig, ok := pass.TypesInfo.Types[callExpr].Type.(*types.Signature); ok {
			return sig
		}
	}
	return nil
}

// functionReturnsError checks if a function signature returns error as the last result.
func functionReturnsError(sig *types.Signature) bool {
	results := sig.Results()
	if results == nil || results.Len() == 0 {
		return false
	}

	lastResult := results.At(results.Len() - 1)
	return lastResult.Type() == builtinErrorType
}
