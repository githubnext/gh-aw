// Package regexpdynamicpattern implements a Go analysis linter that flags
// calls to regexp compile functions whose pattern argument is not a
// compile-time constant string. Malformed dynamic patterns can panic in
// MustCompile variants, return errors in Compile variants, and, when
// influenced by untrusted input, allow an attacker to control pattern
// complexity or size.
package regexpdynamicpattern

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
	"github.com/github/gh-aw/pkg/logger"
)

var pkgLog = logger.New("linters:regexpdynamicpattern")

// Analyzer is the regexp-dynamic-pattern analysis pass.
var Analyzer = analyzerutil.New("regexpdynamicpattern", "reports regexp compile calls whose pattern is not a compile-time constant string", run)

const diagnosticMessage = "regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants, return errors in Compile variants, or let untrusted input control pattern complexity/size"

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	pkgLog.Printf("analyzing package %s", pass.Pkg.Path())
	noLintIndex, generatedFiles, err := analyzerutil.Indexes(pass)
	if err != nil {
		return nil, err
	}

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call, ok := cur.Node().(*ast.CallExpr)
		if !ok || !isRegexpCompileCall(pass, call) {
			continue
		}
		if hasConstantStringPattern(pass, call) {
			continue
		}

		pos := pass.Fset.PositionFor(call.Pos(), false)
		if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
			continue
		}
		if nolint.HasDirectiveForLinter(pos, noLintIndex, "regexpdynamicpattern") {
			continue
		}
		pkgLog.Printf("flagging dynamic regexp pattern at %s", pos)
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: diagnosticMessage,
		})
	}

	return nil, nil
}

// isRegexpCompileCall checks if the call is to a regexp compile function,
// resolving the package identity via the type checker to handle aliased imports
// and avoid false positives from local identifiers named "regexp".
func isRegexpCompileCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "MustCompile", "Compile", "MustCompilePOSIX", "CompilePOSIX":
		// Recognized regexp compile function; continue with package identity checks.
	default:
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || pass.TypesInfo == nil {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return false
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok || pkgName.Imported() == nil {
		return false
	}
	return pkgName.Imported().Path() == "regexp"
}

// hasConstantStringPattern checks whether the regexp pattern is a compile-time constant string,
// such as a string literal, const identifier, or an expression built entirely from constants
// (e.g. concatenation of string literals/consts). Non-constant expressions such as
// fmt.Sprintf results, concatenation involving variables, or function parameters return false.
func hasConstantStringPattern(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	patternArg := call.Args[0]
	if lit, ok := patternArg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		return true
	}

	tv, ok := pass.TypesInfo.Types[patternArg]
	if !ok || tv.Value == nil || tv.Type == nil {
		return false
	}

	basic, ok := tv.Type.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}
