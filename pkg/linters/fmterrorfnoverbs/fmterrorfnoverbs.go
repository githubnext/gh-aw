// Package fmterrorfnoverbs implements a Go analysis linter that flags calls to
// fmt.Errorf where the format string contains no format verbs, in which case
// errors.New is the idiomatic and cheaper alternative.
package fmterrorfnoverbs

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the fmterrorfnoverbs analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fmterrorfnoverbs",
	Doc:      "reports fmt.Errorf calls whose format string contains no verbs, preferring errors.New",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/fmterrorfnoverbs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "fmterrorfnoverbs")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
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

		// Unquote the string value
		val := lit.Value
		if len(val) >= 2 {
			val = val[1 : len(val)-1]
		}

		if !hasRealFormatVerb(val) {
			position := pass.Fset.PositionFor(call.Pos(), false)
			if nolint.HasDirective(position, noLintLinesByFile) {
				return
			}
			pass.ReportRangef(call, "fmt.Errorf called with no format verbs; use errors.New(%s) instead", lit.Value)
		}
	})

	return nil, nil
}

// hasRealFormatVerb reports whether val (the raw content between the surrounding
// quotes of a Go string literal) contains at least one format verb that is not
// an escaped percent pair (%%). The sequence %% renders as a literal % at
// runtime and does not consume an argument, so it is not a real verb.
func hasRealFormatVerb(val string) bool {
	for i := 0; i < len(val); i++ {
		if val[i] != '%' {
			continue
		}
		i++
		if i >= len(val) {
			// Trailing lone % is a malformed verb directive; treat it as present
			// rather than suggesting errors.New for a broken format string.
			return true
		}
		if val[i] == '%' {
			// %% is an escaped percent, not a verb; skip the second % and continue.
			continue
		}
		return true
	}
	return false
}
