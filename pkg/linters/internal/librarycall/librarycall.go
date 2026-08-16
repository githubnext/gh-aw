// Package librarycall provides a shared implementation for linters that ban a
// small set of package-level function calls inside library packages.
//
// A restriction is expressed as data (restricted package, restricted function
// names, and a diagnostic message) while this package owns the common policy:
// skipping main and cmd/ entry points, skipping test and generated files,
// walking call expressions, and honoring nolint directives.
package librarycall

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:librarycall")

// Restriction declares which package functions may not be called from library
// packages and how the violation is reported.
type Restriction struct {
	// Linter is the analyzer name, also used to match nolint directives.
	Linter string
	// PkgPath is the import path of the restricted package, such as "os".
	PkgPath string
	// Funcs lists the restricted function names in PkgPath.
	Funcs []string
	// Message builds the diagnostic for a restricted call. It receives the
	// matched function name and the path of the package being analyzed.
	Message func(funcName, pkgPath string) string
}

// Analyzer returns an analysis pass that enforces the restriction.
func (r Restriction) Analyzer(doc string) *analysis.Analyzer {
	return analyzerutil.New(r.Linter, doc, r.Run)
}

// Run reports restricted calls in library packages.
func (r Restriction) Run(pass *analysis.Pass) (any, error) {
	if !IsLibraryPackage(pass) {
		pkgLog.Printf("%s: skipping non-library package", r.Linter)
		return nil, nil
	}
	pkgPath := pass.Pkg.Path()
	pkgLog.Printf("%s: analyzing package %s", r.Linter, pkgPath)

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

	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}
		position := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
			return
		}
		fn, ok := astutil.CalledPkgFunc(pass, call, r.PkgPath, r.Funcs...)
		if !ok {
			return
		}
		if nolint.HasDirectiveForLinter(position, noLintIndex, r.Linter) {
			return
		}
		pkgLog.Printf("%s: flagging %s.%s call at %s", r.Linter, r.PkgPath, fn.Name(), position)
		pass.ReportRangef(call, "%s", r.Message(fn.Name(), pkgPath))
	})
}

// IsLibraryPackage reports whether pass analyzes a library package, that is a
// package that is neither a main package, a cmd/ entry point, nor a test
// binary.
func IsLibraryPackage(pass *analysis.Pass) bool {
	if pass == nil || pass.Pkg == nil {
		return false
	}
	pkgPath := pass.Pkg.Path()
	if pass.Pkg.Name() == "main" || strings.HasSuffix(pkgPath, "/main") || strings.Contains(pkgPath, "/cmd/") {
		return false
	}
	return !strings.HasSuffix(pkgPath, ".test")
}
