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

// roundTripTypes holds the underlying types of a two-level conversion expression.
type roundTripTypes struct {
	outer    types.Type
	inner    types.Type
	innerArg types.Type
}

// unpackConversionPair validates that outer is a two-level type conversion and
// returns the inner call expression and the resolved underlying types.
// Returns (nil, nil, false) when the expression is not a well-formed pair.
func unpackConversionPair(pass *analysis.Pass, outer *ast.CallExpr) (*ast.CallExpr, *roundTripTypes, bool) {
	if len(outer.Args) != 1 || outer.Ellipsis.IsValid() {
		return nil, nil, false
	}
	outerFunInfo, ok := pass.TypesInfo.Types[outer.Fun]
	if !ok || !outerFunInfo.IsType() {
		return nil, nil, false
	}
	outerType := pass.TypesInfo.TypeOf(outer)
	if outerType == nil {
		return nil, nil, false
	}
	inner, ok := outer.Args[0].(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	if len(inner.Args) != 1 || inner.Ellipsis.IsValid() {
		return nil, nil, false
	}
	// The inner call must also be a type conversion, not a function call.
	innerFunInfo, ok := pass.TypesInfo.Types[inner.Fun]
	if !ok || !innerFunInfo.IsType() {
		return nil, nil, false
	}
	innerType := pass.TypesInfo.TypeOf(inner)
	if innerType == nil {
		return nil, nil, false
	}
	innerArgType := pass.TypesInfo.TypeOf(inner.Args[0])
	if innerArgType == nil {
		return nil, nil, false
	}
	return inner, &roundTripTypes{
		outer:    outerType.Underlying(),
		inner:    innerType.Underlying(),
		innerArg: innerArgType.Underlying(),
	}, true
}

// analyzeRoundTrip checks whether a conversion expression is a redundant
// string/[]byte round-trip (string([]byte(s))) or a wasteful two-copy clone
// ([]byte(string(b))) and reports a diagnostic if so.
func analyzeRoundTrip(pass *analysis.Pass, n ast.Node, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	outer, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}
	// Cheap arg-count guard: eliminates ordinary multi-arg function calls before
	// the more expensive file-skip and nolint-directive lookups below.
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

	inner, rtt, ok := unpackConversionPair(pass, outer)
	if !ok {
		return
	}

	// Check string([]byte(s)) where s is already a string.
	if isStringType(rtt.outer) && isByteSliceType(rtt.inner) && isStringType(rtt.innerArg) {
		argText := astutil.NodeText(pass.Fset, inner.Args[0])
		pass.ReportRangef(outer,
			"string([]byte(%s)) is a redundant round-trip; the inner []byte conversion copies the string unnecessarily",
			argText,
		)
		return
	}

	// Check []byte(string(b)) where b is already a []byte.
	// This is the defensive-copy idiom: the result is a non-aliasing copy, not
	// a no-op.  The diagnostic is therefore not "redundant" but "wasteful":
	// two memory copies are made when one would suffice.
	if isByteSliceType(rtt.outer) && isStringType(rtt.inner) && isByteSliceType(rtt.innerArg) {
		argText := astutil.NodeText(pass.Fset, inner.Args[0])
		pass.ReportRangef(outer,
			"[]byte(string(%s)) makes two copies to clone %s; use slices.Clone(%s) or bytes.Clone(%s) for a single-copy independent slice",
			argText, argText, argText, argText,
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
