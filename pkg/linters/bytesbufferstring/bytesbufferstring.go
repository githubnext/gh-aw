// Package bytesbufferstring implements a Go analysis linter that flags
// string(buf.Bytes()) calls where buf is *bytes.Buffer, suggesting buf.String()
// instead.
package bytesbufferstring

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the bytes-buffer-string analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "bytesbufferstring",
	Doc:      "reports string(buf.Bytes()) calls where buf is *bytes.Buffer and suggests buf.String() instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/bytesbufferstring",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "bytesbufferstring")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Match string(...) type conversion.
		typeInfo, ok := pass.TypesInfo.Types[call.Fun]
		if !ok || !typeInfo.IsType() {
			return
		}
		basic, ok := typeInfo.Type.(*types.Basic)
		if !ok || basic.Kind() != types.String {
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

		// The argument must be buf.Bytes() where buf is *bytes.Buffer.
		inner, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Bytes" {
			return
		}
		if len(inner.Args) != 0 {
			return
		}
		receiverType := pass.TypesInfo.TypeOf(sel.X)
		if receiverType == nil {
			return
		}
		if !isBytesBufferPtr(receiverType) {
			return
		}

		receiverText := astutil.NodeText(pass.Fset, sel.X)
		if receiverText == "" {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: fmt.Sprintf("string(%s.Bytes()) can be simplified to %s.String()", receiverText, receiverText),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: fmt.Sprintf("Replace string(%s.Bytes()) with %s.String()", receiverText, receiverText),
				TextEdits: []analysis.TextEdit{{
					Pos:     call.Pos(),
					End:     call.End(),
					NewText: []byte(receiverText + ".String()"),
				}},
			}},
		})
	})

	return nil, nil
}

// isBytesBufferPtr reports whether t is *bytes.Buffer or bytes.Buffer.
func isBytesBufferPtr(t types.Type) bool {
	// Unwrap pointer if present.
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "bytes" && obj.Name() == "Buffer"
}
