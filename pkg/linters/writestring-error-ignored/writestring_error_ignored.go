// Package writestringerrorignored implements a Go analysis linter that flags
// strings.Builder.WriteString, WriteByte, WriteRune, and WriteInt calls whose
// error returns are ignored or not captured, missing potential write failures.
package writestringerrorignored

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the writestring-error-ignored analysis pass.
var Analyzer = analyzerutil.New("writestringerrorignored",
	"reports ignored error returns from strings.Builder.WriteString, WriteByte, WriteRune, or WriteInt calls",
	run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
		(*ast.ExprStmt)(nil),
	}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeWriteStringError(pass, n, generatedFiles, noLintIndex)
	})
}

// analyzeWriteStringError checks for WriteString/WriteByte/WriteRune/WriteInt calls
// where the error is explicitly discarded.
func analyzeWriteStringError(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	switch stmt := n.(type) {
	case *ast.AssignStmt:
		analyzeAssignStmt(pass, stmt, generatedFiles, noLintIndex)
	case *ast.ExprStmt:
		analyzeExprStmt(pass, stmt, generatedFiles, noLintIndex)
	}
}

// analyzeAssignStmt checks for patterns like:
// - _ = builder.WriteString(...)  (error assigned to blank identifier)
// - _, _ := builder.WriteString(...)  (both results ignored)
// But NOT: n, _ := builder.WriteString(...) (result is being used)
func analyzeAssignStmt(pass *analysis.Pass, assign *ast.AssignStmt, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	// Pattern 1: _ = builder.WriteString(...)
	if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
		for _, lhs := range assign.Lhs {
			blank, ok := lhs.(*ast.Ident)
			if !ok || blank.Name != "_" {
				return
			}
		}
		for _, rhs := range assign.Rhs {
			call, ok := astutil.UnwrapParenExpr(rhs).(*ast.CallExpr)
			if !ok {
				return
			}
			if isBuilderWriteMethodCall(pass, call) {
				reportIfNotSkipped(pass, call, generatedFiles, noLintIndex)
			}
		}
		return
	}

	// Pattern 2: _, _ := builder.WriteString(...)
	// Only flag if BOTH return values are assigned to blank identifiers
	if len(assign.Lhs) == 2 && len(assign.Rhs) == 1 {
		// Check if both LHS are blank identifiers
		firstIsBlank := false
		secondIsBlank := false

		if ident, ok := assign.Lhs[0].(*ast.Ident); ok && ident.Name == "_" {
			firstIsBlank = true
		}
		if ident, ok := assign.Lhs[1].(*ast.Ident); ok && ident.Name == "_" {
			secondIsBlank = true
		}

		// Only flag if both are blank
		if !firstIsBlank || !secondIsBlank {
			return
		}

		for _, rhs := range assign.Rhs {
			call, ok := astutil.UnwrapParenExpr(rhs).(*ast.CallExpr)
			if !ok {
				return
			}
			if isBuilderWriteMethodCall(pass, call) {
				reportIfNotSkipped(pass, call, generatedFiles, noLintIndex)
			}
		}
		return
	}
}

// analyzeExprStmt checks for patterns like:
// builder.WriteString(...)  (bare call with error ignored)
func analyzeExprStmt(pass *analysis.Pass, exprStmt *ast.ExprStmt, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	call, ok := astutil.UnwrapParenExpr(exprStmt.X).(*ast.CallExpr)
	if !ok {
		return
	}

	// Note: Deferred calls are DeferStmt nodes, not ExprStmt nodes, so they won't be flagged here.
	if isBuilderWriteMethodCall(pass, call) {
		reportIfNotSkipped(pass, call, generatedFiles, noLintIndex)
	}
}

// isBuilderWriteMethodCall returns true if call is of the form:
// receiver.WriteString(...) or receiver.WriteByte(...) or receiver.WriteRune(...)
// where receiver is a *strings.Builder and the method returns error.
func isBuilderWriteMethodCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := sel.Sel.Name
	// Check for the specific write methods we care about
	isWriteMethod := methodName == "WriteString" ||
		methodName == "WriteByte" ||
		methodName == "WriteRune"

	if !isWriteMethod {
		return false
	}

	// Check if the receiver is a *strings.Builder
	receiverType := pass.TypesInfo.TypeOf(sel.X)
	if receiverType == nil {
		return false
	}

	// Dereference pointer if needed
	if ptr, ok := receiverType.(*types.Pointer); ok {
		receiverType = ptr.Elem()
	}

	// Check if it's strings.Builder
	named, ok := receiverType.(*types.Named)
	if !ok {
		return false
	}

	if named.Obj().Name() != "Builder" {
		return false
	}

	pkg := named.Obj().Pkg()
	if pkg == nil || pkg.Path() != "strings" {
		return false
	}

	// Verify the method signature returns error
	// Get the method from the type
	methodObj := pass.TypesInfo.Selections[sel]
	if methodObj == nil {
		return false
	}

	funcType, ok := methodObj.Type().(*types.Signature)
	if !ok {
		return false
	}

	// WriteByte returns just error (1 return value)
	// WriteString and WriteRune return (int, error) (2 return values)
	// Both should have error as their final return type
	if funcType.Results().Len() < 1 {
		return false
	}

	// Last return value should be error
	lastReturnType := funcType.Results().At(funcType.Results().Len() - 1).Type()
	if !types.Identical(lastReturnType, types.Universe.Lookup("error").Type()) {
		return false
	}

	return true
}

// reportIfNotSkipped reports a diagnostic if the call is not skipped by nolint or generated file checks.
func reportIfNotSkipped(pass *analysis.Pass, call *ast.CallExpr, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	pos := pass.Fset.PositionFor(call.Pos(), false)

	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}

	if nolint.HasDirectiveForLinter(pos, noLintIndex, "writestringerrorignored") {
		return
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	methodName := sel.Sel.Name
	pass.ReportRangef(call, "error from %s is ignored", methodName)
}
