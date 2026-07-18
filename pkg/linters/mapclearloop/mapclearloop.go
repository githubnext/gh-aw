// Package mapclearloop implements a Go analysis linter that flags
// range-over-map loops whose only body statement is delete(m, k),
// which should be replaced by the built-in clear(m) introduced in Go 1.21.
package mapclearloop

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the map-clear-loop analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "mapclearloop",
	Doc:      "reports range-over-map loops that delete every entry and can be replaced with clear(m)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/mapclearloop",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
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

	nodeFilter := []ast.Node{(*ast.RangeStmt)(nil)}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		checkMapClearLoopNode(pass, n, noLintIndex, generatedFiles)
	})

	return nil, nil
}

// checkMapClearLoopNode inspects a single RangeStmt and reports if it can be
// replaced by a clear(m) call.
func checkMapClearLoopNode(pass *analysis.Pass, n ast.Node, noLintIndex nolint.DirectiveIndex, generatedFiles filecheck.GeneratedIndex) {
	rangeStmt, ok := n.(*ast.RangeStmt)
	if !ok {
		return
	}

	pos := pass.Fset.PositionFor(rangeStmt.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "mapclearloop") {
		return
	}

	mText, ok := validateMapClearPattern(pass, rangeStmt)
	if !ok {
		return
	}

	diag := analysis.Diagnostic{
		Pos:     rangeStmt.Pos(),
		End:     rangeStmt.End(),
		Message: "range-delete loop over map can be replaced with clear(" + mText + ")",
	}
	if !hasOverlappingComment(pass.Files, rangeStmt.Pos(), rangeStmt.End()) {
		diag.SuggestedFixes = []analysis.SuggestedFix{{
			Message: "Replace range-delete loop with clear",
			TextEdits: []analysis.TextEdit{{
				Pos:     rangeStmt.Pos(),
				End:     rangeStmt.End(),
				NewText: []byte("clear(" + mText + ")"),
			}},
		}}
	}
	pass.Report(diag)
}

// validateMapClearPattern reports whether rangeStmt is a range-over-map loop
// whose only body statement is delete(m, k), where k is the range key variable
// and m is the range expression. Returns the text of the map expression and
// whether the pattern matched.
func validateMapClearPattern(pass *analysis.Pass, rangeStmt *ast.RangeStmt) (mText string, ok bool) {
	// The range expression must be a map type.
	mapType := pass.TypesInfo.TypeOf(rangeStmt.X)
	if mapType == nil {
		return "", false
	}
	if _, ok := mapType.Underlying().(*types.Map); !ok {
		return "", false
	}

	// The key variable must be present (not blank or absent).
	keyIdent, ok := rangeStmt.Key.(*ast.Ident)
	if !ok || keyIdent.Name == "_" {
		return "", false
	}
	keyObj := pass.TypesInfo.Defs[keyIdent]
	if keyObj == nil {
		keyObj = pass.TypesInfo.Uses[keyIdent]
	}
	if keyObj == nil {
		return "", false
	}

	// The value variable must be absent or blank.
	if rangeStmt.Value != nil {
		valueIdent, ok := rangeStmt.Value.(*ast.Ident)
		if !ok || valueIdent.Name != "_" {
			return "", false
		}
	}

	// The body must be exactly delete(m, k) with the range key.
	if !validateDeleteBody(pass, rangeStmt, keyObj) {
		return "", false
	}

	mText = astutil.NodeText(pass.Fset, rangeStmt.X)
	if mText == "" {
		return "", false
	}
	if !builtinVisibleAtPos(pass.Pkg, rangeStmt.Pos(), "clear") {
		return "", false
	}
	return mText, true
}

// validateDeleteBody reports whether rangeStmt's body is exactly delete(m, k)
// where m is the range expression and k is keyObj.
func validateDeleteBody(pass *analysis.Pass, rangeStmt *ast.RangeStmt, keyObj types.Object) bool {
	if len(rangeStmt.Body.List) != 1 {
		return false
	}
	exprStmt, ok := rangeStmt.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	callExpr, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	delIdent, ok := callExpr.Fun.(*ast.Ident)
	if !ok || delIdent.Name != "delete" {
		return false
	}
	delBuiltin, ok := pass.TypesInfo.Uses[delIdent].(*types.Builtin)
	if !ok || delBuiltin.Name() != "delete" {
		return false
	}
	if len(callExpr.Args) != 2 {
		return false
	}
	if !sameObject(pass, callExpr.Args[0], rangeStmt.X) {
		return false
	}
	delKeyIdent, ok := callExpr.Args[1].(*ast.Ident)
	if !ok {
		return false
	}
	delKeyObj := pass.TypesInfo.Uses[delKeyIdent]
	return delKeyObj != nil && delKeyObj == keyObj
}

// builtinVisibleAtPos reports whether name resolves to a builtin object at pos.
func builtinVisibleAtPos(pkg *types.Package, pos token.Pos, name string) bool {
	if pkg == nil {
		return false
	}
	scope := pkg.Scope().Innermost(pos)
	if scope == nil {
		return false
	}
	_, obj := scope.LookupParent(name, pos)
	if obj == nil {
		return false
	}
	builtin, ok := obj.(*types.Builtin)
	return ok && builtin.Name() == name
}

// hasOverlappingComment reports whether any comment group overlaps [start, end).
func hasOverlappingComment(files []*ast.File, start, end token.Pos) bool {
	for _, file := range files {
		if end <= file.Pos() || start >= file.End() {
			continue
		}
		for _, group := range file.Comments {
			if group.Pos() < end && start < group.End() {
				return true
			}
		}
	}
	return false
}

// sameObject reports whether expr refers to the same declared object as ref.
// ref is expected to be an *ast.Ident or *ast.SelectorExpr.
func sameObject(pass *analysis.Pass, expr, ref ast.Expr) bool {
	switch r := ref.(type) {
	case *ast.Ident:
		e, ok := expr.(*ast.Ident)
		if !ok {
			return false
		}
		return pass.TypesInfo.Uses[e] == pass.TypesInfo.Uses[r]
	case *ast.SelectorExpr:
		e, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		return pass.TypesInfo.Uses[e.Sel] == pass.TypesInfo.Uses[r.Sel] &&
			sameObject(pass, e.X, r.X)
	default:
		return false
	}
}
