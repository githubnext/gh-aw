// Package writebytestring implements a Go analysis linter that flags
// w.Write([]byte(s)) calls where s is a string, which can be replaced with
// io.WriteString(w, s) to avoid an unnecessary []byte allocation.
package writebytestring

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

const (
	ioPkg            = "io"
	importSpecIndent = "\t"
)

// writerIface is a synthetic *types.Interface matching io.Writer:
//
//	Write(p []byte) (n int, err error)
//
// Built once at package init so it can be reused across analysis passes.
var writerIface = func() *types.Interface {
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	errType := types.Universe.Lookup("error").Type()
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "p", byteSlice))
	results := types.NewTuple(
		types.NewVar(token.NoPos, nil, "n", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "err", errType),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	method := types.NewFunc(token.NoPos, nil, "Write", sig)
	iface := types.NewInterfaceType([]*types.Func{method}, nil)
	iface.Complete()
	return iface
}()

// Analyzer is the write-byte-string analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "writebytestring",
	Doc:      "reports w.Write([]byte(s)) calls where s is a string that can be replaced with io.WriteString(w, s)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/writebytestring",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "writebytestring")
	filesWithImportEdit := make(map[token.Pos]bool)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Match <expr>.Write(<arg>)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Write" {
			return
		}
		if len(call.Args) != 1 {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// The single argument must be a []byte(s) conversion where s is a string.
		conv, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			return
		}
		if !isByteSliceConversion(pass, conv) {
			return
		}
		strArg := conv.Args[0]
		if !isStringType(pass, strArg) {
			return
		}

		// The receiver must implement io.Writer.
		if !implementsWriter(pass, sel.X) {
			return
		}

		sText := astutil.NodeText(pass.Fset, strArg)
		wText := astutil.NodeText(pass.Fset, sel.X)
		if sText == "" || wText == "" {
			return
		}

		// io.WriteString requires an exact predeclared string argument. If the
		// argument is a named string type (e.g. type MyStr string), wrap it with
		// string(...) so the emitted fix compiles.
		sExpr := sText
		if st := pass.TypesInfo.TypeOf(strArg); st != nil && !isExactString(st) {
			sExpr = "string(" + sText + ")"
		}

		// When the receiver is an addressable value whose Write method lives on
		// the pointer type (e.g. var buf bytes.Buffer), io.WriteString requires
		// the pointer form so that the interface conversion compiles.
		writerArg := wText
		if t := pass.TypesInfo.TypeOf(sel.X); t != nil &&
			!types.Implements(t, writerIface) &&
			types.Implements(types.NewPointer(t), writerIface) {
			writerArg = "&" + wText
		}

		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        fmt.Sprintf("%s.Write([]byte(%s)) can be replaced with io.WriteString(%s, %s) to potentially avoid a []byte allocation if the writer implements io.StringWriter", wText, sText, writerArg, sExpr),
			SuggestedFixes: buildFix(pass, call, writerArg, sExpr, filesWithImportEdit),
		})
	})

	return nil, nil
}

// isByteSliceConversion reports whether conv is a []byte or []uint8 conversion expression.
func isByteSliceConversion(pass *analysis.Pass, conv *ast.CallExpr) bool {
	funTypeInfo, ok := pass.TypesInfo.Types[conv.Fun]
	if !ok || !funTypeInfo.IsType() {
		return false
	}
	return isByteSlice(pass, conv)
}

// isByteSlice reports whether expr has type []byte ([]uint8).
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

// isStringType reports whether expr has type string (or named string type).
func isStringType(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}

// isExactString reports whether t is the predeclared string type, not a named
// type whose underlying type is string. io.WriteString(w Writer, s string)
// requires a predeclared string; named string types need an explicit string(...)
// conversion to satisfy the parameter type.
func isExactString(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.String
}

// implementsWriter reports whether expr's type implements io.Writer.
// It uses types.Implements against a synthetic io.Writer interface so the check
// is idiomatic and avoids manually re-implementing the signature comparison.
// Only T and *T are tried; **T is never constructed so pointer types are not
// double-wrapped.
func implementsWriter(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	if types.Implements(t, writerIface) {
		return true
	}
	// Only add a pointer wrapper when t is not already a pointer, to avoid
	// constructing a semantically meaningless **T type.
	if _, alreadyPtr := t.Underlying().(*types.Pointer); alreadyPtr {
		return false
	}
	return types.Implements(types.NewPointer(t), writerIface)
}

// buildFix returns a SuggestedFix rewriting w.Write([]byte(s)) to io.WriteString(w, s).
//
// When "io" is already imported under an alias the alias is used as the
// qualifier so the fix compiles. When the qualifier name is shadowed by a local
// variable at the call site, no SuggestedFix is emitted (the diagnostic is
// still reported).
func buildFix(pass *analysis.Pass, call *ast.CallExpr, writerArg, sText string, filesWithImportEdit map[token.Pos]bool) []analysis.SuggestedFix {
	// Find the file containing this call.
	var file *ast.File
	for _, f := range pass.Files {
		if f.Pos() <= call.Pos() && call.Pos() <= f.End() {
			file = f
			break
		}
	}

	// Determine the local qualifier for "io": use the alias when the package
	// is already imported under a different name, or the default name when it
	// needs to be added.
	qualifier := ioPkg
	if file != nil {
		if localName, imported := astutil.ImportedAs(file, pass.TypesInfo, ioPkg); imported {
			// Dot-import or blank-import: can't safely qualify; skip fix.
			if localName == "." || localName == "_" {
				return nil
			}
			qualifier = localName
		}
		// Not imported yet: qualifier stays as ioPkg; the import will be added.
	}

	// Skip the fix if the qualifier is shadowed by a local at the call site.
	if astutil.QualifierShadowed(pass.Pkg, call.Pos(), qualifier, ioPkg) {
		return nil
	}

	replacement := fmt.Sprintf("%s.WriteString(%s, %s)", qualifier, writerArg, sText)
	edits := []analysis.TextEdit{{
		Pos:     call.Pos(),
		End:     call.End(),
		NewText: []byte(replacement),
	}}
	if importEdit, ok := addIOImportEdit(pass, call.Pos(), filesWithImportEdit); ok {
		edits = append(edits, importEdit)
	}

	return []analysis.SuggestedFix{{
		Message:   "Replace with " + replacement,
		TextEdits: edits,
	}}
}

// addIOImportEdit returns a TextEdit that inserts an import for "io" into the
// file containing pos, unless "io" is already imported in that file or an
// import edit for this file has already been emitted in this pass.
func addIOImportEdit(pass *analysis.Pass, pos token.Pos, filesWithImportEdit map[token.Pos]bool) (analysis.TextEdit, bool) {
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

	if filesWithImportEdit[file.Pos()] {
		return analysis.TextEdit{}, false
	}
	markAndReturn := func(edit analysis.TextEdit) (analysis.TextEdit, bool) {
		filesWithImportEdit[file.Pos()] = true
		return edit, true
	}

	for _, imp := range file.Imports {
		if imp.Path.Value == `"`+ioPkg+`"` {
			return analysis.TextEdit{}, false
		}
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT || !genDecl.Lparen.IsValid() {
			continue
		}
		// Keep the edit minimal by appending at the end of the grouped import;
		// ordering/formatting can be normalized by formatters if desired.
		return markAndReturn(analysis.TextEdit{
			Pos:     genDecl.Rparen,
			End:     genDecl.Rparen,
			NewText: []byte(importSpecIndent + `"` + ioPkg + `"` + "\n"),
		})
	}

	var singleUngroupedImportDecl *ast.GenDecl
	importDeclCount := 0
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		importDeclCount++
		if !genDecl.Lparen.IsValid() && len(genDecl.Specs) == 1 {
			singleUngroupedImportDecl = genDecl
		}
	}
	if importDeclCount == 1 && singleUngroupedImportDecl != nil {
		specText := astutil.NodeText(pass.Fset, singleUngroupedImportDecl.Specs[0])
		if specText != "" {
			// Rebuild as a grouped import while preserving the existing import spec
			// text and adding "io".
			return markAndReturn(analysis.TextEdit{
				Pos:     singleUngroupedImportDecl.Pos(),
				End:     singleUngroupedImportDecl.End(),
				NewText: []byte("import (\n" + importSpecIndent + specText + "\n" + importSpecIndent + `"` + ioPkg + `"` + "\n)"),
			})
		}
		// If we fail to render the existing import spec, fall back to a
		// standalone import insertion below rather than emitting a broken edit.
	}

	return markAndReturn(analysis.TextEdit{
		Pos:     file.Name.End(),
		End:     file.Name.End(),
		NewText: []byte("\n\nimport \"" + ioPkg + "\""),
	})
}
