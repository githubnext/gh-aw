// Package bytescomparestring implements a Go analysis linter that flags
// string(a) == string(b) and string(a) != string(b) comparisons where both
// a and b are []byte values, which should use bytes.Equal(a, b) instead for
// clearer intent.
package bytescomparestring

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

const bytesPkg = "bytes"

// Analyzer is the bytes-compare-string analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "bytescomparestring",
	Doc:      "flags string(a) == string(b) and string(a) != string(b) as []byte comparisons written the long way; use bytes.Equal for clearer intent",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/bytescomparestring",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "bytescomparestring")

	// seenImportFiles tracks files that have already received a bytes import
	// TextEdit in this pass, preventing duplicate overlapping edits when a
	// single file contains multiple flagged comparisons.
	seenImportFiles := make(map[token.Pos]bool)

	nodeFilter := []ast.Node{
		(*ast.BinaryExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return
		}

		// Only flag == and != operators.
		if bin.Op != token.EQL && bin.Op != token.NEQ {
			return
		}

		pos := pass.Fset.PositionFor(bin.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// Both sides must be string(x) conversions where x is []byte.
		lhsArg, ok := extractByteSliceStringConv(pass, bin.X)
		if !ok {
			return
		}
		rhsArg, ok := extractByteSliceStringConv(pass, bin.Y)
		if !ok {
			return
		}

		lText := astutil.NodeText(pass.Fset, lhsArg)
		rText := astutil.NodeText(pass.Fset, rhsArg)
		if lText == "" || rText == "" {
			return
		}

		// Determine the local qualifier for "bytes" and whether the fix is safe.
		qualifier, skipFix := bytesQualifier(pass, bin.Pos())

		if bin.Op == token.EQL {
			var fixes []analysis.SuggestedFix
			if !skipFix {
				fixes = buildFix(pass, bin, fmt.Sprintf("%s.Equal(%s, %s)", qualifier, lText, rText), seenImportFiles)
			}
			pass.Report(analysis.Diagnostic{
				Pos:            bin.Pos(),
				End:            bin.End(),
				Message:        fmt.Sprintf("string(%s) == string(%s) is a []byte comparison written the long way; use bytes.Equal(%s, %s) for clearer intent", lText, rText, lText, rText),
				SuggestedFixes: fixes,
			})
		} else {
			var fixes []analysis.SuggestedFix
			if !skipFix {
				fixes = buildFix(pass, bin, fmt.Sprintf("!%s.Equal(%s, %s)", qualifier, lText, rText), seenImportFiles)
			}
			pass.Report(analysis.Diagnostic{
				Pos:            bin.Pos(),
				End:            bin.End(),
				Message:        fmt.Sprintf("string(%s) != string(%s) is a []byte comparison written the long way; use !bytes.Equal(%s, %s) for clearer intent", lText, rText, lText, rText),
				SuggestedFixes: fixes,
			})
		}
	})

	return nil, nil
}

// bytesQualifier returns the local binding name for the "bytes" package in the
// file containing pos, and whether the SuggestedFix should be skipped.
// Returns ("bytes", false) when the package is not yet imported (the import
// will be added by the fix). Returns (alias, false) when the package is already
// imported under an alias. Returns ("", true) when a safe qualifier cannot be
// determined: dot-import, blank-import, or the qualifier name is shadowed by a
// local variable or parameter at pos.
func bytesQualifier(pass *analysis.Pass, pos token.Pos) (qualifier string, skipFix bool) {
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= pos && pos <= f.End() {
			file = f
			break
		}
	}

	qualifier = bytesPkg
	if file != nil {
		if localName, imported := astutil.ImportedAs(file, pass.TypesInfo, bytesPkg); imported {
			if localName == "." || localName == "_" {
				return "", true
			}
			qualifier = localName
		}
		// Not imported yet: qualifier stays bytesPkg, import will be added.
	}

	if astutil.QualifierShadowed(pass.Pkg, pos, qualifier, bytesPkg) {
		return "", true
	}

	return qualifier, false
}

// buildFix returns the SuggestedFix for rewriting bin to replacement, adding a
// "bytes" import TextEdit when the file containing bin does not yet import it.
// seenImportFiles tracks files that have already received an import edit in
// this pass so that multi-violation files do not get duplicate overlapping edits.
func buildFix(pass *analysis.Pass, bin *ast.BinaryExpr, replacement string, seenImportFiles map[token.Pos]bool) []analysis.SuggestedFix {
	edits := []analysis.TextEdit{{
		Pos:     bin.Pos(),
		End:     bin.End(),
		NewText: []byte(replacement),
	}}
	if importEdit, ok := addBytesImportEdit(pass, bin.Pos(), seenImportFiles); ok {
		edits = append(edits, importEdit)
	}
	return []analysis.SuggestedFix{{
		Message:   "Replace with " + replacement,
		TextEdits: edits,
	}}
}

// addBytesImportEdit returns a TextEdit that inserts an import for "bytes" into
// the file containing pos, unless "bytes" is already imported in that file or
// an import edit for this file has already been emitted in this pass
// (tracked via seenImportFiles to prevent duplicate overlapping edits).
// Returns (TextEdit{}, false) when no edit is needed.
func addBytesImportEdit(pass *analysis.Pass, pos token.Pos, seenImportFiles map[token.Pos]bool) (analysis.TextEdit, bool) {
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= pos && pos <= f.End() {
			file = f
			break
		}
	}
	if file == nil {
		return analysis.TextEdit{}, false
	}

	// Skip if an import edit for this file was already emitted in this pass.
	if seenImportFiles[file.Pos()] {
		return analysis.TextEdit{}, false
	}

	// Check if "bytes" is already imported in this file.
	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+bytesPkg+`"` {
			return analysis.TextEdit{}, false
		}
	}

	// Compute the edit to add and mark the file so subsequent violations in the
	// same pass do not emit a duplicate overlapping TextEdit.

	// Find an existing grouped import declaration to add into.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT || !genDecl.Lparen.IsValid() {
			continue
		}
		// Insert "bytes" before the closing paren of the import block.
		seenImportFiles[file.Pos()] = true
		return analysis.TextEdit{
			Pos:     genDecl.Rparen,
			End:     genDecl.Rparen,
			NewText: []byte("\t\"" + bytesPkg + "\"\n"),
		}, true
	}

	// If the file has exactly one import (non-grouped), convert it to a grouped
	// import block while adding "bytes" before it (alphabetical order).
	if len(file.Imports) == 1 {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.IMPORT || genDecl.Lparen.IsValid() || len(genDecl.Specs) != 1 {
				continue
			}

			specText := astutil.NodeText(pass.Fset, genDecl.Specs[0])
			if specText == "" {
				continue
			}

			seenImportFiles[file.Pos()] = true
			return analysis.TextEdit{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				NewText: []byte("import (\n\t\"" + bytesPkg + "\"\n\t" + specText + "\n)"),
			}, true
		}
	}

	// No grouped import block; insert a standalone import after the package name.
	seenImportFiles[file.Pos()] = true
	return analysis.TextEdit{
		Pos:     file.Name.End(),
		End:     file.Name.End(),
		NewText: []byte("\n\nimport \"" + bytesPkg + "\""),
	}, true
}

// extractByteSliceStringConv checks whether expr is a string(x) conversion
// where x has underlying type []byte. If so, it returns x and true.
func extractByteSliceStringConv(pass *analysis.Pass, expr ast.Expr) (ast.Expr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}

	// Must be a type conversion, not a function call.
	funInfo, ok := pass.TypesInfo.Types[call.Fun]
	if !ok || !funInfo.IsType() {
		return nil, false
	}

	// The outer conversion must produce a string.
	resultInfo, ok := pass.TypesInfo.Types[call]
	if !ok {
		return nil, false
	}
	basic, ok := resultInfo.Type.Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return nil, false
	}

	// The argument must be []byte (or []uint8).
	arg := call.Args[0]
	if !isByteSlice(pass, arg) {
		return nil, false
	}

	return arg, true
}

// isByteSlice reports whether expr has underlying type []byte ([]uint8).
func isByteSlice(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	sl, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	elem, ok := sl.Elem().(*types.Basic)
	return ok && elem.Kind() == types.Byte
}
