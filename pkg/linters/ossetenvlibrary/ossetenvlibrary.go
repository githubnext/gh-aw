// Package ossetenvlibrary implements a Go analysis linter that flags
// os.Setenv and os.Unsetenv calls in non-main, non-test packages.
package ossetenvlibrary

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the os-setenv-in-library analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "ossetenvlibrary",
	Doc:      "reports calls to os.Setenv or os.Unsetenv in non-main, non-test packages",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/ossetenvlibrary",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

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

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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

	return nil, nil
}
