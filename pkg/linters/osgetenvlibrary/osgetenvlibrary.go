// Package osgetenvlibrary implements a Go analysis linter that flags
// os.Getenv and os.LookupEnv calls in non-main, non-test packages.
package osgetenvlibrary

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the os-getenv-in-library analysis pass.
var Analyzer = analyzerutil.New("osgetenvlibrary", "reports calls to os.Getenv or os.LookupEnv in non-main, non-test packages", run)

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return nil, nil
	}

	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
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

		fn, ok := astutil.CalledOSFunc(pass, call, "Getenv", "LookupEnv")
		if !ok {
			return
		}
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirectiveForLinter(position, noLintIndex, "osgetenvlibrary") {
			return
		}
		switch fn.Name() {
		case "Getenv":
			pass.ReportRangef(call, "os.Getenv couples the library to the process environment; pass configuration explicitly instead")
		case "LookupEnv":
			pass.ReportRangef(call, "os.LookupEnv couples the library to the process environment; pass configuration explicitly instead")
		}
	})
}
