// Package ossetenvlibrary implements a Go analysis linter that flags
// os.Setenv and os.Unsetenv calls in non-main, non-test packages.
package ossetenvlibrary

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the os-setenv-in-library analysis pass.
var Analyzer = analyzerutil.New("ossetenvlibrary", "reports calls to os.Setenv or os.Unsetenv in non-main, non-test packages", run)

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	generatedFiles, noLintIndex, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		if strings.HasSuffix(pkgPath, ".test") || filecheck.ShouldSkipFilename(pass.Fset.PositionFor(call.Pos(), false).Filename, generatedFiles) {
			return
		}

		fn, ok := astutil.CalledOSFunc(pass, call, "Setenv", "Unsetenv")
		if !ok {
			return
		}
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirectiveForLinter(position, noLintIndex, "ossetenvlibrary") {
			return
		}
		switch fn.Name() {
		case "Setenv":
			pass.ReportRangef(call, "os.Setenv mutates the process environment; pass configuration explicitly instead")
		case "Unsetenv":
			pass.ReportRangef(call, "os.Unsetenv mutates the process environment; pass configuration explicitly instead")
		}
	})
}
