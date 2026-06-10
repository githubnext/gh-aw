// Package regexpcompileinfunction implements a Go analysis linter that flags
// calls to regexp.MustCompile() and regexp.Compile() inside function bodies.
// These should be moved to package-level variables for performance.
package regexpcompileinfunction

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

// Analyzer is the regexp-compile-in-function analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "regexpcompileinfunction",
	Doc:      "reports regexp.MustCompile and regexp.Compile calls inside function bodies that should be moved to package-level variables",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/regexpcompileinfunction",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "regexpcompileinfunction")

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !isRegexpCompileCall(call) {
			continue
		}
		if !hasConstantStringPattern(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}

		// Check if we're inside a function (not package-level).
		inside := false
		for range cur.Enclosing((*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)) {
			inside = true
			break
		}
		if !inside {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "regexp compilation inside function should be moved to package-level variable",
		})
	}

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
