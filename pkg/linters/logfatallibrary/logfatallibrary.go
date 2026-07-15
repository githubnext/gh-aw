// Package logfatallibrary implements a Go analysis linter that flags
// log.Fatal, log.Fatalf, and log.Fatalln calls in library (pkg/) packages.
// These functions call os.Exit(1) internally, which bypasses deferred cleanup
// and makes the package untestable in isolation.
package logfatallibrary

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// fatalFuncs is the set of log functions that call os.Exit(1) internally.
var fatalFuncs = map[string]bool{
	"Fatal":   true,
	"Fatalf":  true,
	"Fatalln": true,
}

// Analyzer is the log-fatal-in-library analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "logfatallibrary",
	Doc:      "reports log.Fatal, log.Fatalf, and log.Fatalln calls inside library packages where they implicitly call os.Exit and bypass deferred cleanup",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/logfatallibrary",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	pkgPath := pass.Pkg.Path()
	// Skip packages under cmd/ entry-points — they are allowed to call log.Fatal.
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
		if !fatalFuncs[sel.Sel.Name] {
			return
		}
		if !astutil.IsPkgSelector(pass, sel, "log") {
			return
		}
		position := pass.Fset.PositionFor(call.Pos(), false)
		if nolint.HasDirectiveForLinter(position, noLintIndex, "logfatallibrary") {
			return
		}
		pass.ReportRangef(call, "log.%s called in library package %s; use error returns instead to avoid implicit os.Exit", sel.Sel.Name, pkgPath)
	})

	return nil, nil
}
