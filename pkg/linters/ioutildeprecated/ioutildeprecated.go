// Package ioutildeprecated implements a Go analysis linter that flags calls to
// functions from the deprecated io/ioutil package and suggests their replacements
// in the io and os packages (deprecated since Go 1.16).
package ioutildeprecated

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// replacements maps deprecated ioutil function names to their modern equivalents.
var replacements = map[string]string{
	"ReadAll":   "io.ReadAll",
	"ReadFile":  "os.ReadFile",
	"WriteFile": "os.WriteFile",
	"TempFile":  "os.CreateTemp",
	"TempDir":   "os.MkdirTemp",
	"ReadDir":   "os.ReadDir",
	"NopCloser": "io.NopCloser",
	"Discard":   "io.Discard",
}

// Analyzer is the ioutil-deprecated analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "ioutildeprecated",
	Doc:      "reports uses of deprecated io/ioutil functions that should be replaced with io or os package equivalents",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/ioutildeprecated",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	root, err := astutil.Root(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "ioutildeprecated")

	if pass.TypesInfo == nil {
		return nil, nil
	}

	// Handle regular qualified imports: ioutil.ReadAll(...), ioutil.Discard, etc.
	for cur := range root.Preorder((*ast.SelectorExpr)(nil)) {
		sel, ok := cur.Node().(*ast.SelectorExpr)
		if !ok {
			continue
		}

		pos := pass.Fset.PositionFor(sel.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}

		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(pkgIdent)
		if obj == nil {
			continue
		}
		pkgName, ok := obj.(*types.PkgName)
		if !ok || pkgName.Imported().Path() != "io/ioutil" {
			continue
		}

		funcName := sel.Sel.Name
		if replacement, found := replacements[funcName]; found {
			pass.ReportRangef(sel, "ioutil.%s is deprecated; use %s instead", funcName, replacement)
		}
	}

	// Handle dot imports: import . "io/ioutil" followed by bare ReadAll(r) or Discard.
	// In this case the identifier is an *ast.Ident (not a SelectorExpr), and
	// TypesInfo.Uses resolves it to an object whose package path is "io/ioutil".
	for cur := range root.Preorder((*ast.Ident)(nil)) {
		ident, ok := cur.Node().(*ast.Ident)
		if !ok {
			continue
		}

		// Skip identifiers that are the Sel field of a SelectorExpr; those are
		// already handled by the qualified-import loop above.
		if _, ok := cur.Parent().Node().(*ast.SelectorExpr); ok {
			continue
		}

		pos := pass.Fset.PositionFor(ident.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			continue
		}

		obj := pass.TypesInfo.Uses[ident]
		if obj == nil {
			continue
		}
		pkg := obj.Pkg()
		if pkg == nil || pkg.Path() != "io/ioutil" {
			continue
		}

		name := obj.Name()
		if replacement, found := replacements[name]; found {
			pass.ReportRangef(ident, "ioutil.%s is deprecated; use %s instead", name, replacement)
		}
	}

	return nil, nil
}
