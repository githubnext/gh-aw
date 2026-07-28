// Package stringbytesroundtrip implements a Go analysis linter that flags two
// related but semantically distinct patterns:
//   - string([]byte(s)) when s is already a string: genuinely redundant — the
//     result is value-identical to s and both conversions can be removed.
//   - []byte(string(b)) when b is already a []byte: not redundant but wasteful
//     — this is the defensive-copy idiom that produces a non-aliasing clone via
//     two copies; prefer slices.Clone(b) or bytes.Clone(b) for a single copy.
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
	Doc:      "reports string([]byte(s)) as a redundant round-trip when s is already a string, and []byte(string(b)) as a wasteful two-copy clone when b is already a []byte (prefer slices.Clone or bytes.Clone)",
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
// string/[]byte round-trip (string([]byte(s))) or a wasteful two-copy clone
// ([]byte(string(b))) and reports a diagnostic if so.
func analyzeRoundTrip(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	outer, inner, ok := roundTripCalls(n)
	if !ok || shouldSkipRoundTrip(pass, outer, generatedFiles, noLintIndex) {
		return
	}
	outerUnderlying, innerUnderlying, innerArgUnderlying, ok := roundTripUnderlyingTypes(pass, outer, inner)
	if !ok {
		return
	}
	if reportRedundantRoundTrip(pass, outer, inner, outerUnderlying, innerUnderlying, innerArgUnderlying) {
		return
	}
	reportWastefulCloneRoundTrip(pass, outer, inner, outerUnderlying, innerUnderlying, innerArgUnderlying)
}

func roundTripCalls(n ast.Node) (*ast.CallExpr, *ast.CallExpr, bool) {
	outer, ok := n.(*ast.CallExpr)
	if !ok || len(outer.Args) != 1 || outer.Ellipsis.IsValid() {
		return nil, nil, false
	}
	inner, ok := outer.Args[0].(*ast.CallExpr)
	if !ok || len(inner.Args) != 1 || inner.Ellipsis.IsValid() {
		return nil, nil, false
	}
	return outer, inner, true
}

func shouldSkipRoundTrip(pass *analysis.Pass, outer *ast.CallExpr, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) bool {
	pos := pass.Fset.PositionFor(outer.Pos(), false)
	if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
		return true
	}
	return nolint.HasDirectiveForLinter(pos, noLintIndex, "stringbytesroundtrip")
}

func roundTripUnderlyingTypes(pass *analysis.Pass, outer, inner *ast.CallExpr) (types.Type, types.Type, types.Type, bool) {
	outerFunInfo, ok := pass.TypesInfo.Types[outer.Fun]
	if !ok || !outerFunInfo.IsType() {
		return nil, nil, nil, false
	}
	innerFunInfo, ok := pass.TypesInfo.Types[inner.Fun]
	if !ok || !innerFunInfo.IsType() {
		return nil, nil, nil, false
	}
	outerType := pass.TypesInfo.TypeOf(outer)
	innerType := pass.TypesInfo.TypeOf(inner)
	innerArgType := pass.TypesInfo.TypeOf(inner.Args[0])
	if outerType == nil || innerType == nil || innerArgType == nil {
		return nil, nil, nil, false
	}
	return outerType.Underlying(), innerType.Underlying(), innerArgType.Underlying(), true
}

func reportRedundantRoundTrip(pass *analysis.Pass, outer, inner *ast.CallExpr, outerUnderlying, innerUnderlying, innerArgUnderlying types.Type) bool {
	if !isStringType(outerUnderlying) || !isByteSliceType(innerUnderlying) || !isStringType(innerArgUnderlying) {
		return false
	}
	argText := astutil.NodeText(pass.Fset, inner.Args[0])
	pass.ReportRangef(outer,
		"string([]byte(%s)) is a redundant round-trip; the inner []byte conversion copies the string unnecessarily",
		argText,
	)
	return true
}

func reportWastefulCloneRoundTrip(pass *analysis.Pass, outer, inner *ast.CallExpr, outerUnderlying, innerUnderlying, innerArgUnderlying types.Type) {
	if !isByteSliceType(outerUnderlying) || !isStringType(innerUnderlying) || !isByteSliceType(innerArgUnderlying) {
		return
	}
	argText := astutil.NodeText(pass.Fset, inner.Args[0])
	pass.ReportRangef(outer,
		"[]byte(string(%s)) makes two copies to clone %s; use slices.Clone(%s) or bytes.Clone(%s) for a single-copy independent slice",
		argText, argText, argText, argText,
	)
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
