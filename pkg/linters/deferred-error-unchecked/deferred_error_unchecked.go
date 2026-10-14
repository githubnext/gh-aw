// Package deferrerrorunchecked implements a Go analysis linter that flags
// deferred function calls that return an error but where the error result is not checked.
package deferrerrorunchecked

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
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

	// Get the type of the function being called
	// For a call expression, we need to check the function's type
	var sig *types.Signature

	// Check the Fun part of the call
	switch fun := callExpr.Fun.(type) {
	case *ast.Ident:
		// Direct function call like: defer someFunc()
		obj := pass.TypesInfo.Uses[fun]
		if obj == nil {
			return
		}
		var ok bool
		sig, ok = obj.Type().(*types.Signature)
		if !ok {
			return
		}

	case *ast.SelectorExpr:
		// Method call like: defer obj.Close()
		sel := pass.TypesInfo.Selections[fun]
		if sel == nil {
			return
		}
		if sel.Kind() != types.MethodVal {
			return
		}
		var ok bool
		sig, ok = sel.Obj().Type().(*types.Signature)
		if !ok {
			return
		}

	default:
		// Other call types (function literals, etc.)
		// Try to get type from the CallExpr itself
		callType := pass.TypesInfo.Types[callExpr].Type
		if callType == nil {
			return
		}
		var ok bool
		sig, ok = callType.(*types.Signature)
		if !ok {
			return
		}
	}

	// Check if it returns error as the last result
	if !functionReturnsError(sig) {
		return
	}

	// Report: deferred call ignores error return
	pass.Reportf(
		deferStmt.Pos(),
		"deferred call discards error return value; handle the error explicitly",
	)
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
