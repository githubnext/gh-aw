// Package bytesbufferstring implements a Go analysis linter that flags
// string(buf.Bytes()) calls where buf is a bytes.Buffer value receiver,
// suggesting buf.String() instead.
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
	Doc:      "reports string(buf.Bytes()) calls where buf is a bytes.Buffer value and suggests buf.String() instead",
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

		// The argument must be buf.Bytes() where buf is a bytes.Buffer value.
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
		// Only flag value receivers (bytes.Buffer), not pointer receivers (*bytes.Buffer).
		// The rewrite string(buf.Bytes()) → buf.String() is not semantics-preserving when
		// buf is a nil *bytes.Buffer: string(buf.Bytes()) panics, while buf.String() returns
		// "<nil>". Restricting to value receivers avoids this semantic difference entirely.
		if !isBytesBufferValue(receiverType) {
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

// isBytesBufferValue reports whether t is exactly bytes.Buffer (value receiver, not pointer).
// We intentionally exclude *bytes.Buffer: the rewrite string(buf.Bytes()) → buf.String() is not
// semantics-preserving when buf is nil — the former panics while the latter returns "<nil>".
func isBytesBufferValue(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "bytes" && obj.Name() == "Buffer"
}
