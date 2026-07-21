// Package seenmapbool implements a Go analysis linter that flags "seen" maps
// declared as map[string]bool (using true as sentinel) that should use
// map[string]struct{} to avoid allocating a bool per entry.
package seenmapbool

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/github/gh-aw/pkg/linters/internal/astutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the seen-map-bool analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "seenmapbool",
	Doc:      "reports map[string]bool used as a set (values always true) where map[string]struct{} should be used instead",
	URL:      "https://github.com/github/gh-aw/tree/main/pkg/linters/seenmapbool",
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

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return
			}
			pos := pass.Fset.PositionFor(fn.Pos(), false)
			if filecheck.ShouldSkipFilename(pos.Filename, generatedFiles) {
				return
			}
			body = fn.Body
		case *ast.FuncLit:
			if fn.Body == nil {
				return
			}
			body = fn.Body
		}
		inspectBody(pass, body, noLintIndex)
	})

	return nil, nil
}

// inspectBody walks a function body and reports map[string]bool variables
// that are only ever assigned the literal true (i.e., used as a set).
func inspectBody(pass *analysis.Pass, body *ast.BlockStmt, noLintIndex nolint.DirectiveIndex) {
	candidates := collectSeenMapCandidates(pass, body)
	if len(candidates) == 0 {
		return
	}

	nonSetMaps := findNonSetMaps(pass, body, candidates)

	for obj, declNode := range candidates {
		if nonSetMaps[obj] {
			continue
		}
		if nolint.HasDirectiveForLinter(pass.Fset.PositionFor(declNode.Pos(), false), noLintIndex, "seenmapbool") {
			continue
		}
		pass.ReportRangef(
			declNode,
			"map[string]bool %q used as a set; use map[string]struct{} to avoid allocating a bool per entry",
			obj.Name(),
		)
	}
}

// collectSeenMapCandidates returns a map of local map[string]bool variables
// declared in body (via := or var), stopping at nested function literals.
func collectSeenMapCandidates(pass *analysis.Pass, body *ast.BlockStmt) map[types.Object]ast.Node {
	candidates := make(map[types.Object]ast.Node)
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if _, ok := n.(*ast.FuncLit); ok {
			return false // do not descend into nested closures
		}
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if stmt.Tok.String() != ":=" {
				return true
			}
			for i, lhs := range stmt.Lhs {
				if i >= len(stmt.Rhs) {
					break
				}
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name == "_" {
					continue
				}
				obj := pass.TypesInfo.ObjectOf(ident)
				if obj == nil {
					continue
				}
				if isMapStringBool(pass.TypesInfo.TypeOf(ident)) && isMapStringBoolExpr(stmt.Rhs[i]) {
					candidates[obj] = ident
				}
			}
		case *ast.DeclStmt:
			genDecl, ok := stmt.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range genDecl.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valSpec.Names {
					if name.Name == "_" {
						continue
					}
					obj := pass.TypesInfo.ObjectOf(name)
					if obj == nil {
						continue
					}
					if isMapStringBool(pass.TypesInfo.TypeOf(name)) {
						candidates[obj] = name
					}
				}
			}
		}
		return true
	})
	return candidates
}

// findNonSetMaps returns the subset of candidates that are assigned a value
// other than the literal true (and therefore cannot be treated as sets).
func findNonSetMaps(pass *analysis.Pass, body *ast.BlockStmt, candidates map[types.Object]ast.Node) map[types.Object]bool {
	nonSetMaps := make(map[types.Object]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			indexExpr, ok := lhs.(*ast.IndexExpr)
			if !ok {
				continue
			}
			ident, ok := indexExpr.X.(*ast.Ident)
			if !ok {
				continue
			}
			obj := pass.TypesInfo.ObjectOf(ident)
			if obj == nil {
				continue
			}
			if _, isCandidate := candidates[obj]; !isCandidate {
				continue
			}
			if i < len(assign.Rhs) && !isBoolTrue(assign.Rhs[i]) {
				nonSetMaps[obj] = true
			}
		}
		return true
	})
	return nonSetMaps
}

// isMapStringBool returns true if t is map[string]bool.
func isMapStringBool(t types.Type) bool {
	if t == nil {
		return false
	}
	m, ok := t.Underlying().(*types.Map)
	if !ok {
		return false
	}
	key, ok := m.Key().(*types.Basic)
	if !ok || key.Kind() != types.String {
		return false
	}
	val, ok := m.Elem().(*types.Basic)
	return ok && val.Kind() == types.Bool
}

// isMapStringBoolExpr reports whether expr is a make(map[string]bool, ...) call
// or a map[string]bool{...} composite literal.
func isMapStringBoolExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		ident, ok := e.Fun.(*ast.Ident)
		if !ok || ident.Name != "make" {
			return false
		}
		if len(e.Args) == 0 {
			return false
		}
		return isMapStringBoolTypeExpr(e.Args[0])
	case *ast.CompositeLit:
		return isMapStringBoolTypeExpr(e.Type)
	}
	return false
}

// isMapStringBoolTypeExpr reports whether the AST node represents map[string]bool.
func isMapStringBoolTypeExpr(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}
	valIdent, ok := mapType.Value.(*ast.Ident)
	return ok && valIdent.Name == "bool"
}

// isBoolTrue reports whether expr is the boolean literal true.
func isBoolTrue(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "true"
}
