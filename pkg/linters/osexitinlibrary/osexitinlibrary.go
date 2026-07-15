// Package osexitinlibrary implements a Go analysis linter that flags
// os.Exit calls in library (pkg/) packages.
package osexitinlibrary

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the os-exit-in-library analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "osexitinlibrary",
	Doc:      "reports os.Exit calls inside library packages where they bypass deferred cleanup and prevent testing",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/osexitinlibrary",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	// Skip packages under cmd/ entry-points — they are allowed to call os.Exit.
	if strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
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
		if strings.HasSuffix(pkgPath, ".test") || filecheck.ShouldSkipFilename(pass.Fset.Position(call.Pos()).Filename, generatedFiles) {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if astutil.IsPkgSelector(pass, sel, "os") && sel.Sel.Name == "Exit" {
			position := pass.Fset.PositionFor(call.Pos(), false)
			if nolint.HasDirectiveForLinter(position, noLintIndex, "osexitinlibrary") {
				return
			}
			pass.ReportRangef(call, "os.Exit called in library package %s; move process termination to a cmd/ entry-point", pkgPath)
		}
	})

	return nil, nil
}
