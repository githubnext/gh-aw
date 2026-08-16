// Package walkfuncerrshadow implements a Go analysis linter that flags
// filepath.Walk and filepath.WalkDir callbacks whose err parameter shadows an
// outer err variable assigned from the walk call itself.
package walkfuncerrshadow

import (
	"go/ast"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
	"golang.org/x/tools/go/analysis"
)

var pkgLog = logger.New("linters:walkfuncerrshadow")

const analyzerName = "walkfuncerrshadow"

// Analyzer is the walkfuncerrshadow analysis pass.
var Analyzer = analyzerutil.New(analyzerName, "reports filepath.Walk/WalkDir callbacks whose err parameter shadows an outer err variable assigned from the walk call", run)

func run(pass *analysis.Pass) (any, error) {
	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())

	generatedFiles, noLintIndex, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	return analyzerutil.Preorder(pass, []ast.Node{(*ast.AssignStmt)(nil)}, func(n ast.Node) {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return
		}
		checkAssign(pass, assign, generatedFiles, noLintIndex)
	})
}

func checkAssign(pass *analysis.Pass, assign *ast.AssignStmt, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return
	}

	outerErr, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || outerErr.Name != "err" {
		return
	}

	call, ok := astutil.UnwrapParenExpr(assign.Rhs[0]).(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := astutil.UnwrapParenExpr(call.Fun).(*ast.SelectorExpr)
	if !ok || !astutil.IsPkgSelector(pass, sel, "path/filepath") {
		return
	}
	if sel.Sel.Name != "Walk" && sel.Sel.Name != "WalkDir" {
		return
	}
	if len(call.Args) < 2 {
		return
	}

	callback, ok := astutil.UnwrapParenExpr(call.Args[1]).(*ast.FuncLit)
	if !ok {
		return
	}
	callbackErr := callbackErrParam(pass, callback)
	if callbackErr == nil {
		return
	}

	assignPos := pass.Fset.PositionFor(assign.Pos(), false)
	if filecheck.ShouldSkipFilename(assignPos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(assignPos, noLintIndex, analyzerName) {
		return
	}

	pkgLog.Printf("flagging filepath.%s callback err shadow at %s", sel.Sel.Name, assignPos)
	pass.ReportRangef(
		callbackErr,
		"callback parameter err shadows outer err assigned from filepath.%s; rename the callback parameter (for example walkErr)",
		sel.Sel.Name,
	)
}

func callbackErrParam(pass *analysis.Pass, callback *ast.FuncLit) *ast.Ident {
	if callback == nil || callback.Type == nil || callback.Type.Params == nil {
		return nil
	}
	params := callback.Type.Params.List
	// checkAssign narrows to filepath.Walk / WalkDir callback literals, both of
	// which require exactly three parameters.
	if len(params) != 3 {
		return nil
	}
	last := params[2]
	if len(last.Names) != 1 || last.Names[0].Name != "err" {
		return nil
	}
	if pass.TypesInfo == nil || !nolint.ImplementsError(pass.TypesInfo.TypeOf(last.Type)) {
		return nil
	}
	return last.Names[0]
}
