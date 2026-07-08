// Package bytescomparestring implements a Go analysis linter that flags
// string(a) == string(b) and string(a) != string(b) comparisons where both
// a and b are []byte values, which should use bytes.Equal(a, b) instead to
// avoid unnecessary allocations.
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

// Analyzer is the bytes-compare-string analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "bytescomparestring",
	Doc:      "reports string(a) == string(b) and string(a) != string(b) comparisons where a and b are []byte values that should use bytes.Equal instead",
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

		op := bin.Op.String()
		if bin.Op == token.EQL {
			pass.Report(analysis.Diagnostic{
				Pos:     bin.Pos(),
				End:     bin.End(),
				Message: fmt.Sprintf("string(%s) == string(%s) allocates; use bytes.Equal(%s, %s) instead", lText, rText, lText, rText),
				SuggestedFixes: []analysis.SuggestedFix{{
					Message: fmt.Sprintf("Replace with bytes.Equal(%s, %s)", lText, rText),
					TextEdits: []analysis.TextEdit{{
						Pos:     bin.Pos(),
						End:     bin.End(),
						NewText: []byte(fmt.Sprintf("bytes.Equal(%s, %s)", lText, rText)),
					}},
				}},
			})
		} else {
			pass.Report(analysis.Diagnostic{
				Pos:     bin.Pos(),
				End:     bin.End(),
				Message: fmt.Sprintf("string(%s) %s string(%s) allocates; use !bytes.Equal(%s, %s) instead", lText, op, rText, lText, rText),
				SuggestedFixes: []analysis.SuggestedFix{{
					Message: fmt.Sprintf("Replace with !bytes.Equal(%s, %s)", lText, rText),
					TextEdits: []analysis.TextEdit{{
						Pos:     bin.Pos(),
						End:     bin.End(),
						NewText: []byte(fmt.Sprintf("!bytes.Equal(%s, %s)", lText, rText)),
					}},
				}},
			})
		}
	})

	return nil, nil
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
