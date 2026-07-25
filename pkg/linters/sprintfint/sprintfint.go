// Package sprintfint implements a Go analysis linter that flags
// fmt.Sprintf("%d", x) calls where x is a single int value and suggests
// using strconv.Itoa(x) instead.
package sprintfint

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

const (
	strconvPkg = "strconv"
	fmtPkg     = "fmt"
)

// Analyzer is the sprintfint analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sprintfint",
	Doc:      `reports fmt.Sprintf("%d", x) calls where x is a single int value; use strconv.Itoa(x) instead`,
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sprintfint",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
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
	seenImportFiles := make(map[token.Pos]bool)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		analyzeSprintfInt(pass, n, generatedFiles, noLintIndex, seenImportFiles)
	})
	return nil, nil
}

// analyzeSprintfInt checks whether a call expression is fmt.Sprintf("%d", x)
// with an int argument and reports a diagnostic to use strconv.Itoa instead.
func analyzeSprintfInt(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex, seenImportFiles map[token.Pos]bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}
	pos := pass.Fset.PositionFor(call.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "sprintfint") {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Sprintf" {
		return
	}
	if !astutil.IsPkgSelector(pass, sel, "fmt") {
		return
	}
	if len(call.Args) != 2 {
		return
	}
	formatLit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || formatLit.Kind != token.STRING || formatLit.Value != `"%d"` {
		return
	}
	arg := call.Args[1]
	argType := pass.TypesInfo.TypeOf(arg)
	if argType == nil {
		return
	}
	if argType != types.Typ[types.Int] {
		return
	}
	pass.Report(analysis.Diagnostic{
		Pos:            call.Pos(),
		End:            call.End(),
		Message:        `use strconv.Itoa(x) instead of fmt.Sprintf("%d", x)`,
		SuggestedFixes: buildItoaFix(pass, call, arg, seenImportFiles),
	})
}

// buildItoaFix returns a SuggestedFix rewriting fmt.Sprintf("%d", x) →
// strconv.Itoa(x). It also emits import TextEdits to add "strconv" and, when
// the flagged call is the file's only "fmt" reference, to remove the now-
// unused "fmt" import so the resulting code compiles without a goimports pass.
//
// When "strconv" is already imported under an alias the alias is used as the
// qualifier so the fix compiles. When the qualifier name is shadowed by a local
// variable at the call site, no SuggestedFix is emitted (the diagnostic is
// still reported).
func buildItoaFix(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr, seenImportFiles map[token.Pos]bool) []analysis.SuggestedFix {
	argText := astutil.NodeText(pass.Fset, arg)
	if argText == "" {
		return nil
	}

	// Find the file that contains this call so we can inspect its imports.
	file := astutil.FileForPos(pass.Files, call.Pos())

	// Determine the local qualifier for "strconv": use the alias when the
	// package is already imported under a different name, or the default name
	// when it needs to be added.
	qualifier := strconvPkg
	if file != nil {
		if localName, imported := astutil.ImportedAs(file, pass.TypesInfo, strconvPkg); imported {
			// Dot-import or blank-import: can't safely qualify; skip fix.
			if localName == "." || localName == "_" {
				return nil
			}
			qualifier = localName
		}
		// Not imported yet: qualifier stays as strconvPkg; the import will be added.
	}

	// Skip the fix if the qualifier is shadowed by a local at the call site.
	if astutil.QualifierShadowed(pass.Pkg, call.Pos(), qualifier, strconvPkg) {
		return nil
	}

	edits := []analysis.TextEdit{{
		Pos:     call.Pos(),
		End:     call.End(),
		NewText: []byte(qualifier + ".Itoa(" + argText + ")"),
	}}

	if file != nil {
		edits = append(edits, buildImportEdits(pass, file, seenImportFiles)...)
	}

	return []analysis.SuggestedFix{{
		Message:   "Replace fmt.Sprintf with " + qualifier + ".Itoa",
		TextEdits: edits,
	}}
}

// buildImportEdits returns TextEdits that add "strconv" to file and, when the
// flagged call is the file's only "fmt" reference, also remove the now-unused
// "fmt" import. seenImportFiles prevents duplicate overlapping edits in files
// with multiple violations.
func buildImportEdits(pass *analysis.Pass, file *ast.File, seenImportFiles map[token.Pos]bool) []analysis.TextEdit {
	if seenImportFiles[file.Pos()] {
		return nil
	}

	_, strconvImported := astutil.ImportedAs(file, pass.TypesInfo, strconvPkg)
	_, fmtImported := astutil.ImportedAs(file, pass.TypesInfo, fmtPkg)

	// If the flagged call is the only "fmt" reference in the file the "fmt"
	// import will become unused after the fix and must be removed.
	orphanFmt := fmtImported && astutil.CountPkgUsesInFile(pass, file, fmtPkg) == 1

	needStrconv := !strconvImported
	needRemoveFmt := orphanFmt

	if !needStrconv && !needRemoveFmt {
		return nil
	}
	seenImportFiles[file.Pos()] = true

	switch {
	case needStrconv && needRemoveFmt:
		return astutil.SwapImportEdits(pass.Fset, file, strconvPkg, fmtPkg)
	case needStrconv:
		if edit, ok := astutil.AddImportEdit(pass, file, strconvPkg); ok {
			return []analysis.TextEdit{edit}
		}
	case needRemoveFmt:
		if edit, ok := astutil.RemoveImportEdit(pass.Fset, file, fmtPkg); ok {
			return []analysis.TextEdit{edit}
		}
	}
	return nil
}
