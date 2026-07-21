// Package strconvparseignorederror implements a Go analysis linter that flags
// strconv parsing calls (Atoi, ParseInt, ParseFloat, ParseBool, ParseUint)
// where the error return is discarded with _.
package strconvparseignorederror

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the strconv-parse-ignored-error analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "strconvparseignorederror",
	Doc:      "reports strconv parsing calls where the error return is discarded with _",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/strconvparseignorederror",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

// strconvParseFuncs is the set of strconv functions to check.
var strconvParseFuncs = map[string]bool{
	"Atoi":       true,
	"ParseInt":   true,
	"ParseFloat": true,
	"ParseBool":  true,
	"ParseUint":  true,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	nolintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	nodeFilter := []ast.Node{(*ast.AssignStmt)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		analyzeStrconvAssign(pass, n, generatedFiles, nolintIndex)
	})
	return nil, nil
}

// analyzeStrconvAssign checks whether an assignment discards the error return
// from a strconv parsing function and reports a diagnostic if so.
func analyzeStrconvAssign(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, nolintIndex nolint.DirectiveIndex) {
	assign, ok := n.(*ast.AssignStmt)
	if !ok {
		return
	}
	if len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return
	}
	blank, ok := assign.Lhs[1].(*ast.Ident)
	if !ok || blank.Name != "_" {
		return
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if !strconvParseFuncs[sel.Sel.Name] {
		return
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}
	obj := pass.TypesInfo.Uses[ident]
	pkgName, ok := obj.(*types.PkgName)
	if !ok || pkgName.Imported().Path() != "strconv" {
		return
	}
	position := pass.Fset.PositionFor(call.Pos(), false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(position, nolintIndex, "strconvparseignorederror") {
		return
	}
	pass.ReportRangef(call, "error return from strconv.%s is discarded; parse failures produce zero values silently", sel.Sel.Name)
}
