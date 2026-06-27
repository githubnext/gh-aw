// Package sprintferrdot implements a Go analysis linter that flags redundant
// .Error() calls on error values passed to fmt format functions.
//
// When an error is formatted with %s or %v, the fmt package calls .Error()
// automatically, so calling .Error() explicitly before passing the value
// to the format function is redundant.
package sprintferrdot

import (
	"go/ast"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

var errorIface = universeErrorInterface()

// universeErrorInterface returns the built-in error interface type, or nil.
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

// Analyzer is the sprintf-err-dot analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "sprintferrdot",
	Doc:      "reports redundant .Error() calls on error arguments passed to fmt format functions",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/sprintferrdot",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}
	noLintLinesByFile := nolint.BuildLineIndex(pass, "sprintferrdot")

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			return
		}

		formatArgIdx, variadicStart, ok := fmtFormatCallInfo(pass, call)
		if !ok {
			return
		}
		if formatArgIdx >= len(call.Args) || variadicStart > len(call.Args) {
			return
		}

		formatStr, ok := extractStringLit(call.Args[formatArgIdx])
		if !ok {
			return
		}

		verbs := parseSimpleFormatVerbs(formatStr)
		if verbs == nil {
			return
		}

		variadicArgs := call.Args[variadicStart:]
		for i, arg := range variadicArgs {
			if i >= len(verbs) {
				break
			}
			if verbs[i] != 's' && verbs[i] != 'v' {
				continue
			}
			if isErrorDotCall(pass, arg) {
				if nolint.HasDirective(pass.Fset.PositionFor(arg.Pos(), false), noLintLinesByFile) {
					continue
				}
				pass.Reportf(arg.Pos(),
					"redundant .Error() call: pass the error value directly with %%%c", verbs[i])
			}
		}
	})

	return nil, nil
}

// fmtFormatCallInfo returns the format-string argument index and the
// variadic-args start index for recognised fmt format functions.
func fmtFormatCallInfo(pass *analysis.Pass, call *ast.CallExpr) (formatArgIdx, variadicStart int, ok bool) {
	sel, isSel := call.Fun.(*ast.SelectorExpr)
	if !isSel {
		return 0, 0, false
	}
	if !astutil.IsPkgSelector(pass, sel, "fmt") {
		return 0, 0, false
	}
	switch sel.Sel.Name {
	case "Sprintf", "Errorf", "Printf":
		return 0, 1, true
	case "Fprintf", "Fscanf":
		return 1, 2, true
	default:
		return 0, 0, false
	}
}

// extractStringLit returns the unquoted value of a string literal expression.
func extractStringLit(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return s, true
}

// parseSimpleFormatVerbs returns the list of format verbs in sequential order.
// It returns nil when the format string uses complex features such as explicit
// argument indices (%[n]verb) or * widths/precisions, since those prevent
// reliable positional mapping.
func parseSimpleFormatVerbs(format string) []rune {
	var verbs []rune
	i := 0
	for i < len(format) {
		if format[i] != '%' {
			i++
			continue
		}
		i++
		if i >= len(format) {
			break
		}
		if format[i] == '%' {
			i++
			continue
		}
		// Skip flags: -, +, #, space, 0
		for i < len(format) && (format[i] == '-' || format[i] == '+' || format[i] == '#' || format[i] == ' ' || format[i] == '0') {
			i++
		}
		if i >= len(format) {
			break
		}
		// Explicit arg index or * width/precision: too complex to analyse.
		if format[i] == '[' || format[i] == '*' {
			return nil
		}
		// Skip width digits.
		for i < len(format) && format[i] >= '0' && format[i] <= '9' {
			i++
		}
		// Skip precision.
		if i < len(format) && format[i] == '.' {
			i++
			if i < len(format) && format[i] == '*' {
				return nil
			}
			for i < len(format) && format[i] >= '0' && format[i] <= '9' {
				i++
			}
		}
		if i >= len(format) {
			break
		}
		verbs = append(verbs, rune(format[i]))
		i++
	}
	return verbs
}

// isErrorDotCall reports whether expr is a zero-argument .Error() call on a
// value that implements the error interface.
func isErrorDotCall(pass *analysis.Pass, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Error" {
		return false
	}
	if errorIface == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(sel.X)
	if t == nil {
		return false
	}
	return types.Implements(t, errorIface) || types.Implements(types.NewPointer(t), errorIface)
}
