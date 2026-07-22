// Package stringbytesroundtrip implements a Go analysis linter that flags
// redundant round-trip type conversions: string([]byte(s)) when s is already
// a string, and []byte(string(b)) when b is already a []byte.  Both
// conversions create an unnecessary intermediate copy and leave the caller with
// the same underlying type as the input.
package stringbytesroundtrip

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the string-bytes-roundtrip analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "stringbytesroundtrip",
	Doc:      "reports redundant string/[]byte round-trip conversions such as string([]byte(s)) or []byte(string(b)) that produce a wasteful intermediate copy",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/stringbytesroundtrip",
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

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		analyzeRoundTrip(pass, n, generatedFiles, noLintIndex)
	})
	return nil, nil
}

// analyzeRoundTrip checks whether a conversion expression is a redundant
// string/[]byte round-trip and reports a diagnostic if so.
func analyzeRoundTrip(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	outer, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}
	// Must be a type conversion (single argument, no ellipsis).
	if len(outer.Args) != 1 || outer.Ellipsis.IsValid() {
		return
	}

	pos := pass.Fset.PositionFor(outer.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(pos, noLintIndex, "stringbytesroundtrip") {
		return
	}

	// Must be a type conversion, not a function call.
	outerFunInfo, ok := pass.TypesInfo.Types[outer.Fun]
	if !ok || !outerFunInfo.IsType() {
		return
	}

	outerType := pass.TypesInfo.TypeOf(outer)
	if outerType == nil {
		return
	}

	inner, ok := outer.Args[0].(*ast.CallExpr)
	if !ok {
		return
	}
	if len(inner.Args) != 1 || inner.Ellipsis.IsValid() {
		return
	}

	// The inner call must also be a type conversion, not a function call.
	innerFunInfo, ok := pass.TypesInfo.Types[inner.Fun]
	if !ok || !innerFunInfo.IsType() {
		return
	}

	innerType := pass.TypesInfo.TypeOf(inner)
	if innerType == nil {
		return
	}
	innerArgType := pass.TypesInfo.TypeOf(inner.Args[0])
	if innerArgType == nil {
		return
	}

	outerUnderlying := outerType.Underlying()
	innerUnderlying := innerType.Underlying()
	innerArgUnderlying := innerArgType.Underlying()

	// Check string([]byte(s)) where s is already a string.
	if isStringType(outerUnderlying) && isByteSliceType(innerUnderlying) && isStringType(innerArgUnderlying) {
		argText := astutil.NodeText(pass.Fset, inner.Args[0])
		pass.ReportRangef(outer,
			"string([]byte(%s)) is a redundant round-trip; the inner []byte conversion copies the string unnecessarily",
			argText,
		)
		return
	}

	// Check []byte(string(b)) where b is already a []byte.
	if isByteSliceType(outerUnderlying) && isStringType(innerUnderlying) && isByteSliceType(innerArgUnderlying) {
		argText := astutil.NodeText(pass.Fset, inner.Args[0])
		pass.ReportRangef(outer,
			"[]byte(string(%s)) is a redundant round-trip; the inner string conversion copies the bytes unnecessarily",
			argText,
		)
	}
}

func isStringType(t types.Type) bool {
	basic, ok := t.(*types.Basic)
	return ok && basic.Kind() == types.String
}

func isByteSliceType(t types.Type) bool {
	sl, ok := t.(*types.Slice)
	if !ok {
		return false
	}
	elem, ok := sl.Elem().Underlying().(*types.Basic)
	return ok && elem.Kind() == types.Byte
}
