// Package sprintfbool implements a Go analysis linter that flags
// fmt.Sprintf("%t", b) calls where b is a single bool value and suggests
// using strconv.FormatBool(b) instead.
package sprintfbool

import (
	"go/ast"
	"go/token"
	"go/types"
	stdstrconv "strconv"

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

type replacement struct {
	argText   string
	qualifier string
	canFix    bool
}

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
	orphanFmtByFile := make(map[token.Pos]bool)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	type candidate struct {
		call *ast.CallExpr
		arg  ast.Expr
		file *ast.File
	}
	candidates := make([]candidate, 0)
	targetCallsByFile := make(map[token.Pos]int)
	fixableCallsByFile := make(map[token.Pos]int)
	filesByPos := make(map[token.Pos]*ast.File)

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

		file := fileForPos(pass.Files, call.Pos())
		if file != nil {
			targetCallsByFile[file.Pos()]++
			filesByPos[file.Pos()] = file
		}

		candidates = append(candidates, candidate{
			call: call,
			arg:  arg,
			file: file,
		})
	})

	replacements := make([]replacement, len(candidates))
	for i, c := range candidates {
		repl := replacementForCall(pass, c.call, c.arg, c.file)
		replacements[i] = repl
		if repl.canFix && c.file != nil {
			fixableCallsByFile[c.file.Pos()]++
		}
	}

	for filePos, targetCalls := range targetCallsByFile {
		file := filesByPos[filePos]
		if file == nil {
			continue
		}
		fmtImported := false
		for _, imp := range file.Imports {
			if importSpecPathEquals(imp, fmtPkg) {
				fmtImported = true
				break
			}
		}
		orphanFmtByFile[filePos] = fmtImported &&
			countPkgUsesInFile(pass, file, fmtPkg) == targetCalls &&
			fixableCallsByFile[filePos] == targetCalls
	}

	for i, c := range candidates {
		repl := replacements[i]
		argText := repl.argText
		if argText == "" {
			argText = astutil.NodeText(pass.Fset, c.arg)
		}
		if argText == "" {
			argText = "b"
		}

		var fixes []analysis.SuggestedFix
		if repl.canFix {
			fixes = buildFormatBoolFix(
				pass,
				c.call,
				repl.argText,
				repl.qualifier,
				c.file,
				seenImportFiles,
				orphanFmtByFile,
			)
		}

		pass.Report(analysis.Diagnostic{
			Pos:            c.call.Pos(),
			End:            c.call.End(),
			Message:        `use strconv.FormatBool(` + argText + `) instead of fmt.Sprintf("%t", ` + argText + `)`,
			SuggestedFixes: fixes,
		})
	}

	return nil, nil
}

func buildFormatBoolFix(
	pass *analysis.Pass,
	call *ast.CallExpr,
	argText string,
	qualifier string,
	file *ast.File,
	seenImportFiles map[token.Pos]bool,
	orphanFmtByFile map[token.Pos]bool,
) []analysis.SuggestedFix {
	edits := []analysis.TextEdit{{
		Pos:     call.Pos(),
		End:     call.End(),
		NewText: []byte(qualifier + ".FormatBool(" + argText + ")"),
	}}

	if file != nil {
		edits = append(edits, buildImportEdits(pass, file, seenImportFiles, orphanFmtByFile)...)
	}

	return []analysis.SuggestedFix{{
		Message:   "Replace fmt.Sprintf with " + qualifier + ".FormatBool",
		TextEdits: edits,
	}}
}

func buildImportEdits(
	pass *analysis.Pass,
	file *ast.File,
	seenImportFiles map[token.Pos]bool,
	orphanFmtByFile map[token.Pos]bool,
) []analysis.TextEdit {
	if seenImportFiles[file.Pos()] {
		return nil
	}

	strconvImported := false
	fmtImported := false
	for _, imp := range file.Imports {
		switch {
		case importSpecPathEquals(imp, strconvPkg):
			strconvImported = true
		case importSpecPathEquals(imp, fmtPkg):
			fmtImported = true
		}
	}

	orphanFmt := fmtImported && orphanFmtByFile[file.Pos()]
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
			if ok && importSpecPathEquals(imp, fmtPkg) {
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
			Pos:     lineStart,
			End:     lineEnd,
			NewText: nil,
		},
		{
			Pos:     fmtDecl.Rparen,
			End:     fmtDecl.Rparen,
			NewText: []byte("\t\"" + strconvPkg + "\"\n"),
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
			if !ok || !importSpecPathEquals(imp, pkg) {
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

func fileForPos(files []*ast.File, pos token.Pos) *ast.File {
	for _, file := range files {
		if file.Pos() <= pos && pos <= file.End() {
			return file
		}
	}
	return nil
}

func replacementForCall(pass *analysis.Pass, call *ast.CallExpr, arg ast.Expr, file *ast.File) replacement {
	argText := astutil.NodeText(pass.Fset, arg)
	if argText == "" {
		return replacement{}
	}

	qualifier := strconvPkg
	if file != nil {
		if localName, imported := astutil.ImportedAs(file, pass.TypesInfo, strconvPkg); imported {
			if localName == "." || localName == "_" {
				return replacement{argText: argText}
			}
			qualifier = localName
		}
	}

	if astutil.QualifierShadowed(pass.Pkg, call.Pos(), qualifier, strconvPkg) {
		return replacement{argText: argText}
	}
	if astutil.HasOverlappingComment(pass.Files, call.Pos(), call.End()) {
		return replacement{argText: argText}
	}

	return replacement{
		argText:   argText,
		qualifier: qualifier,
		canFix:    true,
	}
}

func importSpecPathEquals(spec *ast.ImportSpec, pkgPath string) bool {
	if spec == nil || spec.Path == nil {
		return false
	}
	unquoted, err := stdstrconv.Unquote(spec.Path.Value)
	if err != nil {
		return spec.Path.Value == `"`+pkgPath+`"` || spec.Path.Value == "`"+pkgPath+"`"
	}
	return unquoted == pkgPath
}
