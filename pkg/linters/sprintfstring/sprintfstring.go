// Package sprintfstring implements a Go analysis linter that flags
// fmt.Sprintf("%s", s) calls where s is a single string value and suggests
// using the string value directly instead.
package sprintfstring

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

const fmtPkg = "fmt"

// Analyzer is the sprintfstring analysis pass.
var Analyzer = analyzerutil.New("sprintfstring", "reports fmt.Sprintf(\"%s\", s) calls where s is a single string value; use the string value directly instead", run)

type candidate struct {
	call *ast.CallExpr
	file *ast.File
	arg  ast.Expr
}

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	candidates, targetCallsByFile, filesByPos := collectCandidates(pass, generatedFiles, noLintIndex)
	orphanFmtByFile := computeOrphanFmtStatus(pass, filesByPos, targetCallsByFile)
	for _, c := range candidates {
		reportCandidate(pass, c, orphanFmtByFile)
	}
	return nil, nil
}

func collectCandidates(pass *analysis.Pass, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) ([]candidate, map[token.Pos]int, map[token.Pos]*ast.File) {
	candidates := make([]candidate, 0)
	targetCallsByFile := make(map[token.Pos]int)
	filesByPos := make(map[token.Pos]*ast.File)

	analyzerutil.Preorder(pass, []ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			return
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "sprintfstring") {
			return
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" {
			return
		}
		if !astutil.IsPkgSelector(pass, sel, fmtPkg) {
			return
		}
		if len(call.Args) != 2 {
			return
		}
		formatLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || formatLit.Kind != token.STRING || formatLit.Value != `"%s"` {
			return
		}
		arg := call.Args[1]
		argType := pass.TypesInfo.TypeOf(arg)
		if argType == nil || !types.Identical(argType, types.Typ[types.String]) {
			return
		}

		file := astutil.FileForPos(pass.Files, call.Pos())
		if file != nil {
			targetCallsByFile[file.Pos()]++
			filesByPos[file.Pos()] = file
		}
		candidates = append(candidates, candidate{call: call, file: file, arg: arg})
	})
	return candidates, targetCallsByFile, filesByPos
}

func computeOrphanFmtStatus(pass *analysis.Pass, filesByPos map[token.Pos]*ast.File, targetCallsByFile map[token.Pos]int) map[token.Pos]bool {
	orphanFmtByFile := make(map[token.Pos]bool)
	for filePos, targetCalls := range targetCallsByFile {
		file := filesByPos[filePos]
		if file == nil {
			continue
		}
		_, imported := astutil.ImportedAs(file, pass.TypesInfo, fmtPkg)
		if !imported {
			continue
		}
		orphanFmtByFile[filePos] = astutil.CountPkgUsesInFile(pass, file, fmtPkg) == targetCalls
	}
	return orphanFmtByFile
}

func reportCandidate(pass *analysis.Pass, c candidate, orphanFmtByFile map[token.Pos]bool) {
	replacementText := astutil.NodeText(pass.Fset, c.arg)
	if replacementText == "" {
		return
	}

	diag := analysis.Diagnostic{
		Pos:     c.call.Pos(),
		End:     c.call.End(),
		Message: fmt.Sprintf("use %s directly instead of fmt.Sprintf(\"%%s\", %s)", replacementText, replacementText),
	}
	if !astutil.HasOverlappingComment(pass.Files, c.call.Pos(), c.call.End()) {
		diag.SuggestedFixes = []analysis.SuggestedFix{{
			Message: "Replace fmt.Sprintf call with direct string value",
			TextEdits: []analysis.TextEdit{{
				Pos:     c.call.Pos(),
				End:     c.call.End(),
				NewText: []byte(replacementText),
			}},
		}}
		if file := c.file; file != nil && orphanFmtByFile[file.Pos()] {
			diag.SuggestedFixes[0].TextEdits = append(diag.SuggestedFixes[0].TextEdits, buildImportRemovalEdit(pass, file)...)
		}
	}
	pass.Report(diag)
}

func buildImportRemovalEdit(pass *analysis.Pass, file *ast.File) []analysis.TextEdit {
	edit, ok := astutil.RemoveImportEdit(pass.Fset, file, fmtPkg)
	if !ok {
		return nil
	}
	return []analysis.TextEdit{edit}
}
