// Package regexpcompileinfunction implements a Go analysis linter that flags
// calls to the regexp compile functions (Compile, MustCompile and their POSIX
// variants) inside function bodies. These should be moved to package-level
// variables for performance.
package regexpcompileinfunction

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:regexpcompileinfunction")

// compileFuncNames are the regexp compile functions that should be hoisted to
// package-level variables when called with a constant pattern.
var compileFuncNames = []string{"MustCompile", "Compile", "MustCompilePOSIX", "CompilePOSIX"}

// Analyzer is the regexp-compile-in-function analysis pass.
var Analyzer = analyzerutil.New("regexpcompileinfunction", "reports regexp compile calls inside function bodies that should be moved to package-level variables", run)

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !astutil.IsRegexpCompileCall(pass, call, compileFuncNames...) {
			continue
		}
		if !astutil.HasConstantStringArg(pass, call, 0) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
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
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "regexpcompileinfunction") {
			continue
		}
		pkgLog.Printf("flagging in-function regexp compilation at %s", pos)
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "regexp compilation inside function should be moved to package-level variable",
		})
	}

	return nil, nil
}
