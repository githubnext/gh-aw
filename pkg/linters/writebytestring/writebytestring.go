// Package writebytestring implements a Go analysis linter that flags
// w.Write([]byte(s)) calls where s is a string, which can be replaced with
// io.WriteString(w, s) to avoid an unnecessary []byte allocation.
package writebytestring

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
		if len(conv.Args) != 1 {
			return
		}
		strArg := conv.Args[0]
		if !isStringType(pass, strArg) {
			return
		}

		// The receiver must implement io.Writer (has a Write([]byte) (int, error) method).
		if !implementsWriter(pass, sel.X) {
			return
		}

		sText := astutil.NodeText(pass.Fset, strArg)
		wText := astutil.NodeText(pass.Fset, sel.X)
		if sText == "" || wText == "" {
			return
		}

		// When the receiver is an addressable value whose Write method lives on
		// the pointer type (e.g. var buf bytes.Buffer), io.WriteString requires
		// the pointer form so that the interface conversion compiles.
		writerArg := wText
		if t := pass.TypesInfo.TypeOf(sel.X); t != nil && !hasWriteMethod(t) {
			writerArg = "&" + wText
		}

		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: fmt.Sprintf("%s.Write([]byte(%s)) can be replaced with io.WriteString(%s, %s) to avoid a []byte allocation", wText, sText, writerArg, sText),
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

// implementsWriter reports whether expr's type implements io.Writer
// (i.e., has a method Write([]byte) (int, error)).
func implementsWriter(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	return hasWriteMethod(t) || hasWriteMethod(types.NewPointer(t))
}

// hasWriteMethod reports whether t (or *t) has a Write([]byte)(int,error) method.
func hasWriteMethod(t types.Type) bool {
	ms := types.NewMethodSet(t)
	sel := ms.Lookup(nil, "Write")
	if sel == nil {
		return false
	}
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	params := sig.Params()
	results := sig.Results()
	if params.Len() != 1 || results.Len() != 2 {
		return false
	}
	// Parameter must be []byte
	sl, ok := params.At(0).Type().Underlying().(*types.Slice)
	if !ok {
		return false
	}
	elem, ok := sl.Elem().(*types.Basic)
	if !ok || elem.Kind() != types.Byte {
		return false
	}
	// Results must be (int, error)
	intBasic, ok := results.At(0).Type().Underlying().(*types.Basic)
	if !ok || intBasic.Kind() != types.Int {
		return false
	}
	return nolint.ImplementsError(results.At(1).Type())
}
