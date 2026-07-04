// Package appendbytestring implements a Go analysis linter that flags
// append(b, []byte(s)...) calls where b is []byte and s is a string,
// which can be simplified to append(b, s...) without the redundant conversion.
package appendbytestring

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

// Analyzer is the append-byte-string analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "appendbytestring",
	Doc:      "reports append(b, []byte(s)...) calls where s is a string that can be simplified to append(b, s...)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/appendbytestring",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "appendbytestring")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Match append(b, x...) with exactly 2 arguments and an ellipsis.
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "append" {
			return
		}
		if len(call.Args) != 2 || !call.Ellipsis.IsValid() {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		// The first argument must be []byte.
		if !isByteSlice(pass, call.Args[0]) {
			return
		}

		// The second argument must be a []byte(s) conversion where s is a string.
		conv, ok := call.Args[1].(*ast.CallExpr)
		if !ok {
			return
		}
		if !isByteSliceConversion(conv) {
			return
		}
		if len(conv.Args) != 1 {
			return
		}
		strArg := conv.Args[0]
		if !isStringType(pass, strArg) {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        "append(b, []byte(s)...) can be simplified to append(b, s...); the []byte conversion is unnecessary",
			SuggestedFixes: buildFix(pass, call, conv, strArg),
		})
	})

	return nil, nil
}

// isByteSlice reports whether expr has type []byte.
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

// isByteSliceConversion reports whether conv is a []byte(...) type conversion expression.
func isByteSliceConversion(conv *ast.CallExpr) bool {
	// A type conversion looks like a call but has a type expression as the function.
	// []byte(...) has an ArrayType as the Fun.
	arr, ok := conv.Fun.(*ast.ArrayType)
	if !ok {
		return false
	}
	// Must be a slice (no length), element must be "byte".
	if arr.Len != nil {
		return false
	}
	ident, ok := arr.Elt.(*ast.Ident)
	return ok && ident.Name == "byte"
}

// isStringType reports whether expr has type string.
func isStringType(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}

// buildFix returns a SuggestedFix rewriting append(b, []byte(s)...) to append(b, s...).
func buildFix(pass *analysis.Pass, call *ast.CallExpr, conv *ast.CallExpr, strArg ast.Expr) []analysis.SuggestedFix {
	sText := astutil.NodeText(pass.Fset, strArg)
	if sText == "" {
		return nil
	}
	// Replace the entire second argument []byte(s) with just s.
	// The ellipsis token follows the closing paren of the outer append call,
	// so we only need to rewrite conv.Pos()..conv.End() to sText.
	return []analysis.SuggestedFix{{
		Message: "Replace []byte(s) with s in append",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     conv.Pos(),
				End:     token.Pos(int(call.Rparen)), // rewrite up to (but not including) the closing paren of append
				NewText: []byte(sText),
			},
		},
	}}
}
