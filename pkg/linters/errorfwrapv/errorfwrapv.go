// Package errorfwrapv implements a Go analysis linter that flags calls to
// fmt.Errorf that either format error arguments with %v or otherwise pass
// error arguments without any %w, which breaks error-chain inspection via
// errors.Is and errors.As.
package errorfwrapv

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

var errorIface = universeErrorInterface()

// universeErrorInterface returns the built-in error interface type, or nil if
// it cannot be resolved from types.Universe.
func universeErrorInterface() *types.Interface {
	errorObj := types.Universe.Lookup("error")
	if errorObj == nil {
		return nil
	}

	iface, ok := errorObj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	return iface
}

type formatVerb struct {
	argIdx int
	verb   rune
}

// Analyzer is the errorfwrapv analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "errorfwrapv",
	Doc:      "reports fmt.Errorf calls that pass error arguments without %w wrapping",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/errorfwrapv",
	Requires: []*analysis.Analyzer{inspect.Analyzer, nolint.Analyzer, filecheck.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	const formatArgOffset = 1 // call.Args[0] is the format string.

	if errorIface == nil {
		return nil, errors.New("failed to resolve built-in error interface from types.Universe")
	}

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

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		position := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
			return
		}

		if !astutil.IsFmtErrorf(pass, call) {
			return
		}

		if len(call.Args) == 0 {
			return
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return
		}
		suppressed := nolint.HasDirectiveForLinter(position, noLintIndex, "errorfwrapv")

		verbs := parseFormatVerbs(lit.Value)
		errorArgVerbs := make(map[int][]rune)
		wrappedErrorArgs := make(map[int]bool)
		for _, fv := range verbs {
			callArgIdx := fv.argIdx + formatArgOffset
			if callArgIdx >= len(call.Args) {
				continue
			}
			tv, ok := pass.TypesInfo.Types[call.Args[callArgIdx]]
			if !ok || tv.Type == nil {
				continue
			}
			if !types.Implements(tv.Type, errorIface) {
				continue
			}
			errorArgVerbs[callArgIdx] = append(errorArgVerbs[callArgIdx], fv.verb)
			if fv.verb == 'w' {
				wrappedErrorArgs[callArgIdx] = true
			}
			if fv.verb != 'v' {
				continue
			}
			if suppressed {
				return
			}
			pass.ReportRangef(call, "fmt.Errorf formats an error argument with %%v; use %%w to preserve the error chain")
			// Keep diagnostics to one per call to avoid noisy duplicate reports.
			return
		}

		if len(call.Args) <= formatArgOffset {
			return
		}

		for i := formatArgOffset; i < len(call.Args); i++ {
			tv, ok := pass.TypesInfo.Types[call.Args[i]]
			if !ok || tv.Type == nil {
				continue
			}
			if !types.Implements(tv.Type, errorIface) {
				continue
			}
			if wrappedErrorArgs[i] {
				continue
			}
			verbsForArg, ok := errorArgVerbs[i]
			if ok {
				needsWrap := false
				for _, verb := range verbsForArg {
					if verb == 'T' || verb == 'p' {
						continue
					}
					needsWrap = true
					break
				}
				if !needsWrap {
					continue
				}
			}
			if suppressed {
				return
			}
			pass.ReportRangef(call, "fmt.Errorf passes an error argument without %%w; use %%w to preserve the error chain")
			// Keep diagnostics to one per call to avoid noisy duplicate reports.
			return
		}
	})

	return nil, nil
}

func parseFormatVerbs(s string) []formatVerb {
	var verbs []formatVerb
	if len(s) >= 2 {
		s = s[1 : len(s)-1]
	}

	nextArgIdx := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		i++
		if i >= len(s) {
			break
		}
		if s[i] == '%' {
			continue
		}

		valueArgIdx := 0
		hasExplicitValueArg := false
		if idx, nextPos, ok := parseFormatArgIndex(s, i); ok {
			valueArgIdx = idx
			nextArgIdx = idx + 1
			hasExplicitValueArg = true
			i = nextPos
		}
		for i < len(s) {
			switch s[i] {
			case '-', '+', '#', '0', ' ':
				i++
			default:
				goto width
			}
		}

	width:
		i = consumeFormatWidthOrPrecision(s, i, &nextArgIdx)
		if i < len(s) && s[i] == '.' {
			i++
			i = consumeFormatWidthOrPrecision(s, i, &nextArgIdx)
		}
		if idx, nextPos, ok := parseFormatArgIndex(s, i); ok {
			valueArgIdx = idx
			nextArgIdx = idx + 1
			hasExplicitValueArg = true
			i = nextPos
		}
		if i >= len(s) {
			break
		}
		if !hasExplicitValueArg {
			valueArgIdx = nextArgIdx
			nextArgIdx++
		}
		verbs = append(verbs, formatVerb{argIdx: valueArgIdx, verb: rune(s[i])})
	}

	return verbs
}

func consumeFormatWidthOrPrecision(s string, i int, nextArgIdx *int) int {
	if idx, nextPos, ok := parseFormatArgIndex(s, i); ok && nextPos < len(s) && s[nextPos] == '*' {
		*nextArgIdx = idx + 1
		return nextPos + 1
	}
	if i < len(s) && s[i] == '*' {
		*nextArgIdx = *nextArgIdx + 1
		return i + 1
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

func parseFormatArgIndex(s string, i int) (int, int, bool) {
	if i >= len(s) || s[i] != '[' {
		return 0, i, false
	}

	j := i + 1
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	if j == i+1 || j >= len(s) || s[j] != ']' {
		return 0, i, false
	}

	n, err := strconv.Atoi(s[i+1 : j])
	if err != nil || n <= 0 {
		return 0, i, false
	}
	return n - 1, j + 1, true
}
