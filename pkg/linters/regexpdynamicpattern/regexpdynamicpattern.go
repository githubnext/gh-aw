// Package regexpdynamicpattern implements a Go analysis linter that flags
// calls to regexp compile functions whose pattern argument is not a
// compile-time constant string. Malformed dynamic patterns can panic in
// MustCompile variants, return errors in Compile variants, and, when
// influenced by untrusted input, allow an attacker to control pattern
// complexity or size.
package regexpdynamicpattern

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:regexpdynamicpattern")

// Analyzer is the regexp-dynamic-pattern analysis pass.
var Analyzer = analyzerutil.New("regexpdynamicpattern", "reports regexp compile calls whose pattern is not a compile-time constant string", run)

// compileFuncNames are the regexp compile functions whose pattern argument
// must be a compile-time constant.
var compileFuncNames = []string{"MustCompile", "Compile", "MustCompilePOSIX", "CompilePOSIX"}

const diagnosticMessage = "regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants, return errors in Compile variants, or let untrusted input control pattern complexity/size"

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
		if astutil.HasConstantStringArg(pass, call, 0) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "regexpdynamicpattern") {
			continue
		}
		pkgLog.Printf("flagging dynamic regexp pattern at %s", pos)
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: diagnosticMessage,
		})
	}

	return nil, nil
}
