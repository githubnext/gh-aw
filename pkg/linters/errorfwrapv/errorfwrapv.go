// Package errorfwrapv implements a Go analysis linter that flags calls to
// fmt.Errorf that format error arguments with %v instead of %w, which breaks
// error-chain inspection via errors.Is and errors.As.
package errorfwrapv

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

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// Analyzer is the errorfwrapv analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "errorfwrapv",
	Doc:      "reports fmt.Errorf calls that format error arguments with %v instead of %w",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/errorfwrapv",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "errorfwrapv")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		position := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(position.Filename) {
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

		verbs := parseFormatVerbs(lit.Value)
		for argIdx, verb := range verbs {
			if verb != 'v' {
				continue
			}
			callArgIdx := argIdx + 1
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
			if nolint.HasDirective(position, noLintLinesByFile) {
				return
			}
			pass.ReportRangef(call, "fmt.Errorf formats an error argument with %%v; use %%w to preserve the error chain")
			return
		}
	})

	return nil, nil
}

func parseFormatVerbs(s string) map[int]rune {
	verbs := make(map[int]rune)
	if len(s) >= 2 {
		s = s[1 : len(s)-1]
	}

	argIdx := 0
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
		if s[i] == '[' {
			for i < len(s) && s[i] != ']' {
				i++
			}
			i++
			if i >= len(s) {
				break
			}
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
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i < len(s) && s[i] == '.' {
			i++
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
		}
		if i >= len(s) {
			break
		}
		verbs[argIdx] = rune(s[i])
		argIdx++
	}

	return verbs
}
