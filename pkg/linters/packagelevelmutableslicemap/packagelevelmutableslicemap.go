// Package packagelevelmutableslicemap implements a Go analysis linter that
// flags package-level (file/package-scope) var declarations of slices or maps
// that are mutated from inside a function body via append re-assignment,
// index assignment, or delete().
//
// Package-level mutable slices/maps are shared across every goroutine and
// every call into the package for the lifetime of the process. Mutating one
// from inside a function — rather than storing the state on a struct or
// returning fresh values — risks data races under concurrent access and can
// leak state between unrelated calls.
package packagelevelmutableslicemap

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/github/gh-aw/pkg/linters/internal/analyzerutil"
	"github.com/github/gh-aw/pkg/linters/internal/filecheck"
	"github.com/github/gh-aw/pkg/linters/internal/nolint"
)

// Analyzer is the package-level-mutable-slice-map analysis pass.
var Analyzer = analyzerutil.New("packagelevelmutableslicemap", "reports mutation of package-level slice/map variables from inside function bodies, which risks data races and cross-call state leaks", run)

func run(pass *analysis.Pass) (any, error) {
	noLintIndex, err := nolint.Index(pass)
	if err != nil {
		return nil, err
	}
	generatedFiles, err := filecheck.Index(pass)
	if err != nil {
		return nil, err
	}

	targets := collectPackageLevelSliceMapVars(pass)
	if len(targets) == 0 {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
		(*ast.ExprStmt)(nil),
	}
	return analyzerutil.Preorder(pass, nodeFilter, func(n ast.Node) {
		analyzeNode(pass, n, targets, generatedFiles, noLintIndex)
	})
}

// collectPackageLevelSliceMapVars scans the top-level declarations of every
// file in the package and returns the set of package-scope var objects whose
// underlying type is a slice or a map, keyed by their declared name.
func collectPackageLevelSliceMapVars(pass *analysis.Pass) map[types.Object]string {
	targets := make(map[types.Object]string)
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valueSpec.Names {
					if name.Name == "_" {
						continue
					}
					obj := pass.TypesInfo.Defs[name]
					if obj == nil {
						continue
					}
					t := obj.Type()
					if t == nil {
						continue
					}
					switch t.Underlying().(type) {
					case *types.Slice, *types.Map:
						targets[obj] = name.Name
					}
				}
			}
		}
	}
	return targets
}

func analyzeNode(pass *analysis.Pass, n ast.Node, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	switch stmt := n.(type) {
	case *ast.AssignStmt:
		analyzeAssignStmt(pass, stmt, targets, generatedFiles, noLintIndex)
	case *ast.ExprStmt:
		analyzeExprStmt(pass, stmt, targets, generatedFiles, noLintIndex)
	}
}

func analyzeAssignStmt(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	if name, ok := matchAppendReassign(pass, stmt, targets); ok {
		report(pass, stmt.Pos(), name, "append() re-assignment", generatedFiles, noLintIndex)
		return
	}
	if name, ok := matchIndexAssign(pass, stmt, targets); ok {
		report(pass, stmt.Pos(), name, "index assignment", generatedFiles, noLintIndex)
	}
}

func analyzeExprStmt(pass *analysis.Pass, stmt *ast.ExprStmt, targets map[types.Object]string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "delete" {
		return
	}
	builtin, ok := pass.TypesInfo.Uses[ident].(*types.Builtin)
	if !ok || builtin.Name() != "delete" {
		return
	}
	if len(call.Args) != 2 {
		return
	}
	name, ok := targetIdentName(pass, call.Args[0], targets)
	if !ok {
		return
	}
	report(pass, stmt.Pos(), name, "delete()", generatedFiles, noLintIndex)
}

// matchAppendReassign reports whether stmt is `s = append(s, ...)` for a
// tracked package-level target s, returning its declared name.
func matchAppendReassign(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string) (string, bool) {
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return "", false
	}
	lhsIdent, ok := stmt.Lhs[0].(*ast.Ident)
	if !ok {
		return "", false
	}
	lhsObj := pass.TypesInfo.Uses[lhsIdent]
	name, tracked := targets[lhsObj]
	if !tracked {
		return "", false
	}
	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}
	fnIdent, ok := call.Fun.(*ast.Ident)
	if !ok || fnIdent.Name != "append" {
		return "", false
	}
	builtin, ok := pass.TypesInfo.Uses[fnIdent].(*types.Builtin)
	if !ok || builtin.Name() != "append" {
		return "", false
	}
	firstArgIdent, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return "", false
	}
	if pass.TypesInfo.Uses[firstArgIdent] != lhsObj {
		return "", false
	}
	return name, true
}

// matchIndexAssign reports whether stmt assigns into m[k] for a tracked
// package-level map/slice target m.
func matchIndexAssign(pass *analysis.Pass, stmt *ast.AssignStmt, targets map[types.Object]string) (string, bool) {
	for _, lhs := range stmt.Lhs {
		idxExpr, ok := lhs.(*ast.IndexExpr)
		if !ok {
			continue
		}
		if name, ok := targetIdentName(pass, idxExpr.X, targets); ok {
			return name, true
		}
	}
	return "", false
}

// targetIdentName reports whether expr is an identifier referring to a
// tracked package-level target, returning its declared name.
func targetIdentName(pass *analysis.Pass, expr ast.Expr, targets map[types.Object]string) (string, bool) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return "", false
	}
	obj := pass.TypesInfo.Uses[ident]
	if obj == nil {
		return "", false
	}
	name, ok := targets[obj]
	return name, ok
}

func report(pass *analysis.Pass, pos token.Pos, varName, kind string, generatedFiles filecheck.GeneratedIndex, noLintIndex nolint.DirectiveIndex) {
	position := pass.Fset.PositionFor(pos, false)
	if filecheck.ShouldSkipFilename(position.Filename, generatedFiles) {
		return
	}
	if nolint.HasDirectiveForLinter(position, noLintIndex, "packagelevelmutableslicemap") {
		return
	}
	pass.Report(analysis.Diagnostic{
		Pos:     pos,
		Message: "package-level slice/map variable " + varName + " is mutated via " + kind + "; mutating shared package state risks data races and can leak state across calls",
	})
}
