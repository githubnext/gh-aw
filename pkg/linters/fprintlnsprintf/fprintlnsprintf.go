// Package fprintlnsprintf implements a Go analysis linter that flags
// fmt.Fprintln(w, fmt.Sprintf(...)) calls that should be rewritten as fmt.Fprintf(w, ...).
package fprintlnsprintf

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the fprintlnsprintf analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "fprintlnsprintf",
	Doc:      "reports fmt.Fprintln(w, fmt.Sprintf(...)) calls that should be rewritten as fmt.Fprintf(w, ...)",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/fprintlnsprintf",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "fprintlnsprintf")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		// Check if this is exactly fmt.Fprintln(w, fmt.Sprintf(...)).
		if !isFmtFunc(call, "Fprintln") {
			return
		}
		if len(call.Args) != 2 {
			return
		}

		// Skip test files.
		pos := pass.Fset.Position(call.Pos())
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		// Check if the printed argument is fmt.Sprintf(...).
		printedArg, ok := call.Args[1].(*ast.CallExpr)
		if !ok {
			return
		}
		if !isFmtFunc(printedArg, "Sprintf") {
			return
		}
		if nolint.HasDirective(pos, noLintLinesByFile) {
			return
		}

		pass.Report(analysis.Diagnostic{
			Pos:            call.Pos(),
			End:            call.End(),
			Message:        "use fmt.Fprintf instead of fmt.Fprintln(w, fmt.Sprintf(...))",
			SuggestedFixes: buildFprintfFix(call, printedArg),
		})
	})

	return nil, nil
}

// buildFprintfFix returns a SuggestedFix rewriting
// fmt.Fprintln(w, fmt.Sprintf("format", args...)) to
// fmt.Fprintf(w, "format\n", args...).
// A fix is only emitted when the format argument is a plain double-quoted
// string literal; other forms (raw strings, variables) are left unfixed.
func buildFprintfFix(call *ast.CallExpr, sprintfCall *ast.CallExpr) []analysis.SuggestedFix {
	if len(sprintfCall.Args) == 0 {
		return nil
	}
	formatLit, ok := sprintfCall.Args[0].(*ast.BasicLit)
	if !ok || formatLit.Kind != token.STRING {
		return nil
	}
	raw := formatLit.Value
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil // not a plain double-quoted literal
	}

	// Decode the literal to check for a trailing newline so we don't produce
	// a double \n when the format string already ends with one.
	decoded, err := strconv.Unquote(raw)
	if err != nil {
		return nil
	}

	// Build the replacement format literal: append \n only when not already present.
	var newFormatLit []byte
	if strings.HasSuffix(decoded, "\n") {
		newFormatLit = []byte(raw) // already ends with \n; keep as-is
	} else {
		newFormatLit = []byte(raw[:len(raw)-1] + `\n"`)
	}

	outerSel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	return []analysis.SuggestedFix{{
		Message: `Replace fmt.Fprintln with fmt.Fprintf`,
		TextEdits: []analysis.TextEdit{
			// 1. "Fprintln" → "Fprintf"
			{Pos: outerSel.Sel.Pos(), End: outerSel.Sel.End(), NewText: []byte("Fprintf")},
			// 2. Delete "fmt.Sprintf(" — from the start of sprintfCall to after its "("
			{Pos: sprintfCall.Pos(), End: sprintfCall.Lparen + 1, NewText: nil},
			// 3. Replace the format literal (adding \n if not already present).
			{Pos: formatLit.Pos(), End: formatLit.End(), NewText: newFormatLit},
			// 4. Delete the closing ")" of fmt.Sprintf(...)
			{Pos: sprintfCall.Rparen, End: sprintfCall.Rparen + 1, NewText: nil},
		},
	}}
}

// isFmtFunc returns true if call is a call to fmt.<name>.
func isFmtFunc(call *ast.CallExpr, name string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "fmt" && sel.Sel.Name == name
}
