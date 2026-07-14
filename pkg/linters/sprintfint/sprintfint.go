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
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "sprintfint")

	// seenImportFiles tracks files that have already received an import edit in
	// this pass, preventing duplicate overlapping edits when a single file
	// contains multiple flagged calls.
	seenImportFiles := make(map[token.Pos]bool)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// Match fmt.Sprintf(format, arg) with exactly two arguments.
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

		// The format argument must be the string literal "%d".
		formatLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || formatLit.Kind != token.STRING || formatLit.Value != `"%d"` {
			return
		}

		// The value argument must have the exact type int (not int64, uint, etc.).
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
	})

	return nil, nil
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
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= call.Pos() && call.Pos() <= f.End() {
			file = f
			break
		}
	}

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

	strconvImported := false
	fmtImported := false
	for _, imp := range file.Imports {
		switch imp.Path.Value {
		case `"` + strconvPkg + `"`:
			strconvImported = true
		case `"` + fmtPkg + `"`:
			fmtImported = true
		}
	}

	// If the flagged call is the only "fmt" reference in the file the "fmt"
	// import will become unused after the fix and must be removed.
	orphanFmt := fmtImported && countPkgUsesInFile(pass, file, fmtPkg) == 1

	needStrconv := !strconvImported
	needRemoveFmt := orphanFmt

	if !needStrconv && !needRemoveFmt {
		return nil
	}
	seenImportFiles[file.Pos()] = true

	switch {
	case needStrconv && needRemoveFmt:
		return addStrconvRemoveFmtEdits(pass.Fset, file)
	case needStrconv:
		if edit, ok := addImportEdit(pass, file, strconvPkg); ok {
			return []analysis.TextEdit{edit}
		}
	case needRemoveFmt:
		if edit, ok := removeImportEdit(pass.Fset, file, fmtPkg); ok {
			return []analysis.TextEdit{edit}
		}
	}
	return nil
}

// countPkgUsesInFile returns the number of times the package at pkgPath is
// referenced as a selector base within file (e.g. each "fmt.X" call counts
// as one use of the "fmt" package).
func countPkgUsesInFile(pass *analysis.Pass, file *ast.File, pkgPath string) int {
	fileStart, fileEnd := file.Pos(), file.End()
	count := 0
	for ident, obj := range pass.TypesInfo.Uses {
		pkgName, ok := obj.(*types.PkgName)
		if !ok || pkgName.Imported() == nil || pkgName.Imported().Path() != pkgPath {
			continue
		}
		if p := ident.Pos(); p >= fileStart && p <= fileEnd {
			count++
		}
	}
	return count
}

// addStrconvRemoveFmtEdits returns the TextEdits that simultaneously add
// "strconv" and remove "fmt" from file's import section. Three structural
// cases are handled:
//   - single ungrouped import "fmt"          → replaced with import "strconv"
//   - grouped import block with only "fmt"   → replaced with import "strconv"
//   - grouped block with "fmt" + others      → insert "strconv", delete "fmt" line
func addStrconvRemoveFmtEdits(fset *token.FileSet, file *ast.File) []analysis.TextEdit {
	var fmtSpec *ast.ImportSpec
	var fmtDecl *ast.GenDecl

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			imp, ok := spec.(*ast.ImportSpec)
			if ok && imp.Path.Value == `"`+fmtPkg+`"` {
				fmtSpec = imp
				fmtDecl = genDecl
				break
			}
		}
		if fmtDecl != nil {
			break
		}
	}
	if fmtDecl == nil {
		return nil
	}

	// Single ungrouped import or grouped block with only "fmt":
	// replace the entire declaration with import "strconv".
	if !fmtDecl.Lparen.IsValid() || len(fmtDecl.Specs) == 1 {
		return []analysis.TextEdit{{
			Pos:     fmtDecl.Pos(),
			End:     fmtDecl.End(),
			NewText: []byte(`import "` + strconvPkg + `"`),
		}}
	}

	// Grouped block with "fmt" alongside other packages: insert "strconv"
	// before the closing paren and delete the entire "fmt" spec line.
	lineStart, lineEnd := importSpecLineRange(fset, fmtSpec)
	return []analysis.TextEdit{
		{
			Pos:     fmtDecl.Rparen,
			End:     fmtDecl.Rparen,
			NewText: []byte("\t\"" + strconvPkg + "\"\n"),
		},
		{
			Pos:     lineStart,
			End:     lineEnd,
			NewText: nil,
		},
	}
}

// addImportEdit returns a TextEdit that inserts an import for pkg into file,
// choosing the least-invasive insertion point: append to an existing grouped
// block, convert a single non-grouped import to a grouped block, or insert a
// standalone declaration after the package name.
func addImportEdit(pass *analysis.Pass, file *ast.File, pkg string) (analysis.TextEdit, bool) {
	// Append to an existing grouped import block.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT || !genDecl.Lparen.IsValid() {
			continue
		}
		return analysis.TextEdit{
			Pos:     genDecl.Rparen,
			End:     genDecl.Rparen,
			NewText: []byte("\t\"" + pkg + "\"\n"),
		}, true
	}

	// Convert a single non-grouped import into a grouped block.
	if len(file.Imports) == 1 {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.IMPORT || genDecl.Lparen.IsValid() {
				continue
			}
			specText := astutil.NodeText(pass.Fset, genDecl.Specs[0])
			if specText == "" {
				continue
			}
			return analysis.TextEdit{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				NewText: []byte("import (\n\t" + specText + "\n\t\"" + pkg + "\"\n)"),
			}, true
		}
	}

	// No existing import block; insert a standalone import after the package name.
	return analysis.TextEdit{
		Pos:     file.Name.End(),
		End:     file.Name.End(),
		NewText: []byte("\n\nimport \"" + pkg + "\""),
	}, true
}

// removeImportEdit returns a TextEdit that removes the import of pkg from
// file's import section. For an ungrouped or sole-spec grouped declaration the
// entire decl is removed; for a multi-spec grouped block only the spec line is
// deleted using line-boundary positions from fset to handle any indentation.
func removeImportEdit(fset *token.FileSet, file *ast.File, pkg string) (analysis.TextEdit, bool) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			imp, ok := spec.(*ast.ImportSpec)
			if !ok || imp.Path.Value != `"`+pkg+`"` {
				continue
			}
			// Ungrouped or single-spec grouped: remove the entire declaration.
			if !genDecl.Lparen.IsValid() || len(genDecl.Specs) == 1 {
				return analysis.TextEdit{
					Pos:     genDecl.Pos(),
					End:     genDecl.End(),
					NewText: nil,
				}, true
			}
			// Multi-spec grouped: remove just this spec's line.
			lineStart, lineEnd := importSpecLineRange(fset, imp)
			return analysis.TextEdit{
				Pos:     lineStart,
				End:     lineEnd,
				NewText: nil,
			}, true
		}
	}
	return analysis.TextEdit{}, false
}

// importSpecLineRange returns the [start, end) byte range that covers the
// entire source line of spec — including any leading whitespace and the
// trailing newline. It uses the token.File's line table so it works correctly
// regardless of indentation style.
func importSpecLineRange(fset *token.FileSet, spec *ast.ImportSpec) (token.Pos, token.Pos) {
	tokFile := fset.File(spec.Pos())
	if tokFile == nil {
		// Unreachable in practice; fall back to simple single-char arithmetic.
		return spec.Pos() - 1, spec.End() + 1
	}
	line := tokFile.Line(spec.Pos())
	lineStart := tokFile.LineStart(line)
	if line < tokFile.LineCount() {
		return lineStart, tokFile.LineStart(line + 1)
	}
	// Last line has no following newline — extend past the spec token end.
	return lineStart, spec.End() + 1
}
