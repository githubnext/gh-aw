// Package hardcodedfilepath implements a Go analysis linter that flags
// hard-coded file path string literals and compares them against a known set
// of file path constants. When a literal value matches an existing named
// constant, the linter reports it with a suggestion to use the constant.
// When no matching constant exists, the linter reports it as a candidate for
// extraction into a named constant.
//
// The linter also correlates path literals with logging and print calls,
// annotating those findings with an extra note, because paths in log output
// are especially important to keep consistent via named constants.
package hardcodedfilepath

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the hardcoded-file-path analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "hardcodedfilepath",
	Doc:      "reports hard-coded file path string literals that should be replaced with named constants",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/hardcodedfilepath",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// constRef holds a reference to a named path constant.
type constRef struct {
	pkgAlias  string // local import alias or package name; empty for same-package consts
	constName string // exported constant name
}

func (r constRef) String() string {
	if r.pkgAlias == "" {
		return r.constName
	}
	return r.pkgAlias + "." + r.constName
}

// pathPrefixes lists the prefixes that make a string literal qualify as a
// filesystem path. Each prefix ends with "/" so that bare tokens like "/tmp",
// ".github", or "${RUNNER_TEMP}" alone are not flagged.
var pathPrefixes = []string{
	"/tmp/",
	"${RUNNER_TEMP}/",     //nolint:hardcodedfilepath
	"${{ runner.temp }}/", //nolint:hardcodedfilepath
	".github/",            //nolint:hardcodedfilepath
	"/opt/",
	"/usr/",
	"/var/",
	"/home/",
	"/etc/",
	"/run/",
}

// minPathRuneLen is the minimum character length of an unquoted path value
// to flag. Paths shorter than this are considered too generic.
const minPathRuneLen = 8

// isPathLike reports whether val (an unquoted string value) looks like a
// filesystem path worth inspecting.
func isPathLike(val string) bool {
	if utf8.RuneCountInString(val) < minPathRuneLen {
		return false
	}
	for _, prefix := range pathPrefixes {
		if strings.HasPrefix(val, prefix) {
			return true
		}
	}
	return false
}

// hasFormatVerb reports whether val contains common fmt format verbs. Strings
// with verbs are format templates, not standalone paths, so they are excluded.
func hasFormatVerb(val string) bool {
	return strings.ContainsAny(val, "%") &&
		(strings.Contains(val, "%s") || strings.Contains(val, "%d") ||
			strings.Contains(val, "%v") || strings.Contains(val, "%q") ||
			strings.Contains(val, "%w") || strings.Contains(val, "%f"))
}

// unquoteStringLit returns the raw string value of a Go string literal token,
// stripping surrounding double-quotes or backticks. It does not process escape
// sequences — only the outer delimiters are removed.
func unquoteStringLit(lit string) string {
	if len(lit) >= 2 {
		if (lit[0] == '"' && lit[len(lit)-1] == '"') ||
			(lit[0] == '`' && lit[len(lit)-1] == '`') {
			return lit[1 : len(lit)-1]
		}
	}
	return lit
}

// logPrintMethods is the set of method names that produce human-readable
// output and commonly include file paths in their arguments.
var logPrintMethods = map[string]bool{
	"Print": true, "Println": true, "Printf": true,
	"Fprint": true, "Fprintln": true, "Fprintf": true,
	"Fatal": true, "Fatalf": true, "Fatalln": true,
	"Panic": true, "Panicf": true, "Panicln": true,
	"Error": true, "Errorf": true,
	"Warn": true, "Warnf": true,
	"Info": true, "Infof": true,
	"Debug": true, "Debugf": true,
	"Trace": true, "Tracef": true,
	"Log": true, "Logf": true,
}

// isLogOrPrintCall reports whether call is a package-qualified log or print
// function call (e.g. log.Printf, fmt.Println, logger.Infof).
func isLogOrPrintCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if !logPrintMethods[sel.Sel.Name] {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return true // unknown object — conservative accept
	}
	_, isPkg := obj.(*types.PkgName)
	return isPkg
}

// collectKnownPathConsts builds a map from path string value to constRef by
// scanning:
//  1. All exported constants declared at package scope in pass.Pkg.
//  2. All exported constants in directly imported packages whose import path
//     contains "constants" (e.g. "github.com/example/pkg/constants").
//
// Only string constants whose value matches pathPrefixes are included.
func collectKnownPathConsts(pass *analysis.Pass) map[string]constRef {
	out := make(map[string]constRef)

	addConst := func(c *types.Const, alias, name string) {
		if !c.Exported() {
			return
		}
		basic, ok := c.Type().Underlying().(*types.Basic)
		if !ok || (basic.Kind() != types.String && basic.Kind() != types.UntypedString) {
			return
		}
		// c.Val().String() returns the value with surrounding double-quote characters.
		raw := c.Val().String()
		val := strings.TrimPrefix(strings.TrimSuffix(raw, `"`), `"`)
		if !isPathLike(val) {
			return
		}
		if _, exists := out[val]; !exists {
			out[val] = constRef{pkgAlias: alias, constName: name}
		}
	}

	// 1. Current package's own exported constants.
	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		c, ok := obj.(*types.Const)
		if !ok {
			continue
		}
		addConst(c, "", name)
	}

	// 2. Imported "constants" packages.
	for _, imp := range pass.Pkg.Imports() {
		if !strings.Contains(imp.Path(), "constants") {
			continue
		}
		alias := resolveImportAlias(pass, imp)
		for _, name := range imp.Scope().Names() {
			obj := imp.Scope().Lookup(name)
			c, ok := obj.(*types.Const)
			if !ok {
				continue
			}
			addConst(c, alias, name)
		}
	}

	return out
}

// resolveImportAlias returns the local name used for pkg in the files of pass.
// If an explicit alias is set it is returned; otherwise the package's own name
// is used.
func resolveImportAlias(pass *analysis.Pass, pkg *types.Package) string {
	for _, file := range pass.Files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path != pkg.Path() {
				continue
			}
			if imp.Name != nil && imp.Name.Name != "." && imp.Name.Name != "_" {
				return imp.Name.Name
			}
			return pkg.Name()
		}
	}
	return pkg.Name()
}

func run(pass *analysis.Pass) (any, error) {
	insp, err := astutil.Inspector(pass)
	if err != nil {
		return nil, err
	}

	noLintLines := nolint.BuildLineIndex(pass, "hardcodedfilepath")
	knownConsts := collectKnownPathConsts(pass)

	for cur := range insp.Root().Preorder((*ast.BasicLit)(nil)) {
		lit, ok := cur.Node().(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}

		pos := pass.Fset.PositionFor(lit.Pos(), false)
		if filecheck.IsTestFile(pos.Filename) {
			continue
		}
		if nolint.HasDirective(pos, noLintLines) {
			continue
		}

		raw := unquoteStringLit(lit.Value)
		if !isPathLike(raw) {
			continue
		}
		if hasFormatVerb(raw) {
			continue
		}

		// Skip literals that are the value of a const declaration — those are
		// the canonical definitions, not inline usages.
		if isConstDeclValue(cur) {
			continue
		}

		// Detect whether the literal is a direct argument of a log/print call.
		inLog := enclosingCallIsLogPrint(pass, cur)

		if ref, found := knownConsts[raw]; found {
			msg := fmt.Sprintf(
				"hard-coded file path %q: use constant %s instead of inline string literal",
				raw, ref,
			)
			if inLog {
				msg += " (path appears in log/print call — keeping consistent via constant is especially important)"
			}
			pass.ReportRangef(lit, "%s", msg)
		} else {
			msg := fmt.Sprintf(
				"hard-coded file path %q: consider extracting as a named constant",
				raw,
			)
			if inLog {
				msg += " (path appears in log/print call)"
			}
			pass.ReportRangef(lit, "%s", msg)
		}
	}

	return nil, nil
}

// isConstDeclValue reports whether the cursor's node is the value expression
// of a const declaration. It walks the enclosing GenDecl ancestors and checks
// for Tok == token.CONST.
func isConstDeclValue(cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.GenDecl)(nil)) {
		decl, ok := encl.Node().(*ast.GenDecl)
		if ok && decl.Tok == token.CONST {
			return true
		}
	}
	return false
}

// enclosingCallIsLogPrint reports whether the nearest enclosing *ast.CallExpr
// is a log/print call.
func enclosingCallIsLogPrint(pass *analysis.Pass, cur inspector.Cursor) bool {
	for encl := range cur.Enclosing((*ast.CallExpr)(nil)) {
		call, ok := encl.Node().(*ast.CallExpr)
		if !ok {
			continue
		}
		return isLogOrPrintCall(pass, call)
	}
	return false
}
