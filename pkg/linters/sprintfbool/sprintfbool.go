// Package sprintfbool implements a Go analysis linter that flags
// fmt.Sprintf("%t", b) calls where b is a single bool value and suggests
// using strconv.FormatBool(b) instead.
package sprintfbool

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

// Analyzer is the sprintfbool analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sprintfbool",
	Doc:      `reports fmt.Sprintf("%t", b) calls where b is a single bool value; use strconv.FormatBool(b) instead`,
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sprintfbool",
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
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "sprintfbool") {
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

		// The format argument must be the string literal "%t".
		formatLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || formatLit.Kind != token.STRING || formatLit.Value != `"%t"` {
			return
		}

		// The value argument must have the exact type bool.
		arg := call.Args[1]
		argType := pass.TypesInfo.TypeOf(arg)
		if argType == nil {
			return
		}
		if argType != types.Typ[types.Bool] {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        `use strconv.FormatBool(b) instead of fmt.Sprintf("%t", b)`,
			SuggestedFixes: buildFormatBoolFix(pass, call, arg, seenImportFiles),
		})
	})

	return nil, nil
}

func buildFormatBoolFix(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr, seenImportFiles map[token.Pos]bool) []analysis.SuggestedFix {
	argText := astutil.NodeText(pass.Fset, arg)
	if argText == "" {
		return nil
	}

	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= call.Pos() && call.Pos() <= f.End() {
			file = f
			break
		}
	}

	qualifier := strconvPkg
	if file != nil {
		if localName, imported := astutil.ImportedAs(file, pass.TypesInfo, strconvPkg); imported {
			if localName == "." || localName == "_" {
				return nil
			}
			qualifier = localName
		}
	}

	if astutil.QualifierShadowed(pass.Pkg, call.Pos(), qualifier, strconvPkg) {
		return nil
	}

	edits := []analysis.TextEdit{{
		Pos:     call.Pos(),
		End:     call.End(),
		NewText: []byte(qualifier + ".FormatBool(" + argText + ")"),
	}}

	if file != nil {
		edits = append(edits, buildImportEdits(pass, file, seenImportFiles)...)
	}

	return []analysis.SuggestedFix{{
		Message:   "Replace fmt.Sprintf with " + qualifier + ".FormatBool",
		TextEdits: edits,
	}}
}

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

	if !fmtDecl.Lparen.IsValid() || len(fmtDecl.Specs) == 1 {
		return []analysis.TextEdit{{
			Pos:     fmtDecl.Pos(),
			End:     fmtDecl.End(),
			NewText: []byte(`import "` + strconvPkg + `"`),
		}}
	}

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

func addImportEdit(pass *analysis.Pass, file *ast.File, pkg string) (analysis.TextEdit, bool) {
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

	return analysis.TextEdit{
		Pos:     file.Name.End(),
		End:     file.Name.End(),
		NewText: []byte("\n\nimport \"" + pkg + "\""),
	}, true
}

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
			if !genDecl.Lparen.IsValid() || len(genDecl.Specs) == 1 {
				return analysis.TextEdit{
					Pos:     genDecl.Pos(),
					End:     genDecl.End(),
					NewText: nil,
				}, true
			}
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

func importSpecLineRange(fset *token.FileSet, spec *ast.ImportSpec) (token.Pos, token.Pos) {
	tokFile := fset.File(spec.Pos())
	if tokFile == nil {
		return spec.Pos() - 1, spec.End() + 1
	}
	line := tokFile.Line(spec.Pos())
	lineStart := tokFile.LineStart(line)
	if line < tokFile.LineCount() {
		return lineStart, tokFile.LineStart(line + 1)
	}
	return lineStart, spec.End() + 1
}
